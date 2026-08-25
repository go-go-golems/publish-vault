// profile-search-index is a private diagnostic harness for PV-MEM-002.
// It deliberately forces GC at fixed search progress checkpoints before writing
// heap profiles. The resulting run is perturbed and must not be used as a
// throughput or production-memory baseline. Raw profiles may contain vault
// content and must remain in a private, untracked directory.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/go-go-golems/measure/pkg/collector"
	measurepkg "github.com/go-go-golems/measure/pkg/measure"
	"github.com/go-go-golems/measure/pkg/measurement"
	"github.com/go-go-golems/measure/pkg/sink"
	"github.com/go-go-golems/measure/pkg/trace"
	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type checkpoint struct {
	SchemaVersion      int                `json:"schema_version"`
	Percent            int                `json:"percent"`
	ProcessedDocuments uint64             `json:"processed_documents"`
	TotalDocuments     uint64             `json:"total_documents"`
	IndexedBytes       uint64             `json:"indexed_bytes"`
	ElapsedNanos       int64              `json:"elapsed_nanos"`
	ForcedGC           bool               `json:"forced_gc"`
	ProfileFile        string             `json:"profile_file"`
	Memory             measurement.Memory `json:"memory"`
}

func main() {
	vaultDir := flag.String("vault", "", "vault directory")
	indexDir := flag.String("index", "", "new persistent index directory")
	privateDir := flag.String("private-dir", "", "private directory for raw heap profiles")
	outputDir := flag.String("output-dir", "", "content-free checkpoint and trace directory")
	flag.Parse()
	if *vaultDir == "" || *indexDir == "" || *privateDir == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "--vault, --index, --private-dir, and --output-dir are required")
		os.Exit(2)
	}
	if err := run(*vaultDir, *indexDir, *privateDir, *outputDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(vaultDir, indexDir, privateDir, outputDir string) error {
	for _, dir := range []struct {
		path string
		mode os.FileMode
	}{{privateDir, 0o700}, {outputDir, 0o755}} {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			return fmt.Errorf("create %s: %w", dir.path, err)
		}
	}
	if err := os.Chmod(privateDir, 0o700); err != nil {
		return fmt.Errorf("restrict private profile directory: %w", err)
	}

	loaded, err := vault.New(vaultDir)
	if err != nil {
		return fmt.Errorf("load vault: %w", err)
	}
	total := uint64(loaded.Count())
	if total == 0 {
		return fmt.Errorf("loaded vault has no published notes")
	}

	traceFile, err := os.OpenFile(filepath.Join(outputDir, "diagnostic.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create diagnostic trace: %w", err)
	}
	defer func() { _ = traceFile.Close() }()
	jsonl, err := sink.NewJSONL(traceFile)
	if err != nil {
		return fmt.Errorf("create JSONL sink: %w", err)
	}
	recorder, err := measurepkg.NewRecorder(measurepkg.Options{
		RunName: "search_index_profile", Interval: time.Second,
		Target: collector.SelfTarget(), Reader: collector.NewSystemReader(), Sink: jsonl,
	})
	if err != nil {
		return fmt.Errorf("create recorder: %w", err)
	}
	ctx := context.Background()
	if err := recorder.Start(ctx); err != nil {
		return fmt.Errorf("start recorder: %w", err)
	}
	phase, err := recorder.BeginPhase(ctx, "search_index", measurepkg.PhaseOptions{Total: total, Unit: "documents"})
	if err != nil {
		return fmt.Errorf("begin search phase: %w", err)
	}

	started := time.Now()
	thresholds := thresholdDocuments(total)
	captured := make(map[int]bool, len(thresholds))
	capture := func(progress search.IndexProgress) error {
		if err := phase.SetProgress(progress.ProcessedDocuments); err != nil {
			return err
		}
		for _, percent := range []int{0, 25, 50, 75, 100} {
			if captured[percent] || progress.ProcessedDocuments < thresholds[percent] {
				continue
			}
			if err := captureCheckpoint(ctx, recorder, privateDir, outputDir, percent, progress, started); err != nil {
				return err
			}
			captured[percent] = true
		}
		return nil
	}

	var callbackErr error
	idx, buildErr := search.NewPersistentWithOptions(loaded, indexDir, search.Options{
		ObserveIndexed: func(progress search.IndexProgress) {
			if callbackErr == nil {
				callbackErr = capture(progress)
			}
		},
	})
	if buildErr == nil && callbackErr != nil {
		buildErr = callbackErr
	}
	result := trace.Succeeded()
	if buildErr != nil {
		result = trace.Failed("index_build_failed")
	}
	if _, err := phase.End(ctx, result); err != nil && buildErr == nil {
		buildErr = err
	}
	receipt, finishErr := recorder.Finish(ctx, result)
	if finishErr != nil && buildErr == nil {
		buildErr = finishErr
	}
	if idx != nil {
		if err := idx.Close(); err != nil && buildErr == nil {
			buildErr = err
		}
	}
	if err := traceFile.Close(); err != nil && buildErr == nil {
		buildErr = err
	}
	if err := writeJSON(filepath.Join(outputDir, "diagnostic.receipt.json"), receipt, 0o600); err != nil && buildErr == nil {
		buildErr = err
	}
	if buildErr != nil {
		return fmt.Errorf("build diagnostic index: %w", buildErr)
	}
	for _, percent := range []int{0, 25, 50, 75, 100} {
		if !captured[percent] {
			return fmt.Errorf("checkpoint %d%% was not captured", percent)
		}
	}
	return nil
}

func thresholdDocuments(total uint64) map[int]uint64 {
	return map[int]uint64{
		0:   0,
		25:  (total*25 + 99) / 100,
		50:  (total*50 + 99) / 100,
		75:  (total*75 + 99) / 100,
		100: total,
	}
}

func captureCheckpoint(ctx context.Context, recorder *measurepkg.Recorder, privateDir, outputDir string, percent int, progress search.IndexProgress, started time.Time) error {
	if _, err := recorder.Checkpoint(ctx); err != nil {
		return fmt.Errorf("pre-GC checkpoint %d%%: %w", percent, err)
	}
	runtime.GC()
	profileName := fmt.Sprintf("heap-%03d.pprof", percent)
	profilePath := filepath.Join(privateDir, profileName)
	profileFile, err := os.OpenFile(profilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create profile %d%%: %w", percent, err)
	}
	if err := pprof.WriteHeapProfile(profileFile); err != nil {
		_ = profileFile.Close()
		return fmt.Errorf("write profile %d%%: %w", percent, err)
	}
	if err := profileFile.Close(); err != nil {
		return fmt.Errorf("close profile %d%%: %w", percent, err)
	}
	memory, err := collector.NewSystemReader().Read(ctx, collector.SelfTarget())
	if err != nil {
		return fmt.Errorf("read checkpoint memory %d%%: %w", percent, err)
	}
	if _, err := recorder.Checkpoint(ctx); err != nil {
		return fmt.Errorf("post-GC checkpoint %d%%: %w", percent, err)
	}
	value := checkpoint{
		SchemaVersion: 1, Percent: percent,
		ProcessedDocuments: progress.ProcessedDocuments, TotalDocuments: progress.TotalDocuments,
		IndexedBytes: progress.IndexedBytes, ElapsedNanos: time.Since(started).Nanoseconds(),
		ForcedGC: true, ProfileFile: profileName, Memory: memory,
	}
	return writeJSON(filepath.Join(outputDir, fmt.Sprintf("checkpoint-%03d.json", percent)), value, 0o600)
}

func writeJSON(path string, value any, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
