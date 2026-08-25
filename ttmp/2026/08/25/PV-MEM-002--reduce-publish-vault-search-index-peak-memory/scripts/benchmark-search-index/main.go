// benchmark-search-index measures only persistent search-index construction for
// PV-MEM-002. It is an experiment harness, not a production command.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/measure/pkg/collector"
	measurepkg "github.com/go-go-golems/measure/pkg/measure"
	"github.com/go-go-golems/measure/pkg/report"
	"github.com/go-go-golems/measure/pkg/sink"
	"github.com/go-go-golems/measure/pkg/trace"
	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type metadata struct {
	SchemaVersion  int               `json:"schema_version"`
	BatchDocuments uint64            `json:"batch_documents"`
	BatchBytes     uint64            `json:"batch_bytes"`
	Documents      uint64            `json:"documents"`
	IndexedBytes   uint64            `json:"indexed_bytes"`
	IndexBytes     uint64            `json:"index_bytes"`
	DurationNanos  int64             `json:"duration_nanos"`
	Peaks          trace.MemoryPeaks `json:"peaks"`
	Sources        trace.Sources     `json:"sources"`
	Summary        report.Summary    `json:"summary"`
}

func main() {
	vaultDir := flag.String("vault", "", "vault directory")
	indexDir := flag.String("index", "", "new persistent index directory")
	outputDir := flag.String("output-dir", "", "content-free artifact directory")
	batchDocuments := flag.Uint64("batch-documents", 0, "maximum documents per Bleve batch; zero disables batching")
	batchBytes := flag.Uint64("batch-bytes", 0, "estimated field bytes per Bleve batch; zero disables batching")
	interval := flag.Duration("interval", 100*time.Millisecond, "measure sampling interval")
	flag.Parse()
	if *vaultDir == "" || *indexDir == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "--vault, --index, and --output-dir are required")
		os.Exit(2)
	}
	if err := run(*vaultDir, *indexDir, *outputDir, *batchDocuments, *batchBytes, *interval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(vaultDir, indexDir, outputDir string, batchDocuments, batchBytes uint64, interval time.Duration) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.RemoveAll(indexDir); err != nil {
		return fmt.Errorf("remove prior index: %w", err)
	}
	loaded, err := vault.New(vaultDir)
	if err != nil {
		return fmt.Errorf("load vault: %w", err)
	}

	tracePath := filepath.Join(outputDir, "trace.jsonl")
	traceFile, err := os.OpenFile(tracePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create trace: %w", err)
	}
	jsonl, err := sink.NewJSONL(traceFile)
	if err != nil {
		_ = traceFile.Close()
		return fmt.Errorf("create JSONL sink: %w", err)
	}
	recorder, err := measurepkg.NewRecorder(measurepkg.Options{
		RunName: "search_index_benchmark", Interval: interval,
		Target: collector.SelfTarget(), Reader: collector.NewSystemReader(), Sink: jsonl,
	})
	if err != nil {
		_ = traceFile.Close()
		return fmt.Errorf("create recorder: %w", err)
	}
	ctx := context.Background()
	if err := recorder.Start(ctx); err != nil {
		_ = traceFile.Close()
		return fmt.Errorf("start recorder: %w", err)
	}
	sampleCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- recorder.Run(sampleCtx) }()

	// #nosec G115 -- Count is a nonnegative map length.
	total := uint64(loaded.Count())
	phase, err := recorder.BeginPhase(ctx, "search_index", measurepkg.PhaseOptions{Total: total, Unit: "documents"})
	if err != nil {
		cancel()
		<-done
		_ = traceFile.Close()
		return fmt.Errorf("begin phase: %w", err)
	}
	var finalProgress search.IndexProgress
	var callbackErr error
	started := time.Now()
	idx, buildErr := search.NewPersistentWithOptions(loaded, indexDir, search.Options{
		BatchDocuments: batchDocuments,
		BatchBytes:     batchBytes,
		ObserveIndexed: func(progress search.IndexProgress) {
			finalProgress = progress
			if err := phase.SetProgress(progress.ProcessedDocuments); err != nil && callbackErr == nil {
				callbackErr = err
			}
		},
	})
	if buildErr == nil && callbackErr != nil {
		buildErr = callbackErr
	}
	duration := time.Since(started)
	result := trace.Succeeded()
	if buildErr != nil {
		result = trace.Failed("index_build_failed")
	}
	if _, err := phase.End(ctx, result); err != nil && buildErr == nil {
		buildErr = err
	}
	cancel()
	if err := <-done; err != nil && buildErr == nil {
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
	if buildErr != nil {
		return fmt.Errorf("build persistent index: %w", buildErr)
	}

	events, err := readEvents(tracePath)
	if err != nil {
		return err
	}
	summary, err := report.Summarize(events)
	if err != nil {
		return fmt.Errorf("summarize trace: %w", err)
	}
	indexBytes, err := directoryBytes(indexDir)
	if err != nil {
		return err
	}
	value := metadata{
		SchemaVersion: 1, BatchDocuments: batchDocuments, BatchBytes: batchBytes,
		Documents: finalProgress.ProcessedDocuments, IndexedBytes: finalProgress.IndexedBytes,
		IndexBytes: indexBytes, DurationNanos: duration.Nanoseconds(), Peaks: receipt.Peaks,
		Sources: receipt.Sources, Summary: summary,
	}
	if err := writeJSON(filepath.Join(outputDir, "metadata.json"), value); err != nil {
		return err
	}
	return writeJSON(filepath.Join(outputDir, "receipt.json"), receipt)
}

func readEvents(path string) ([]trace.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return trace.ReadAll(file)
}

func directoryBytes(root string) (uint64, error) {
	var total uint64
	err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Size() > 0 {
			// #nosec G115 -- positive int64 file size is exactly representable.
			total += uint64(info.Size())
		}
		return nil
	})
	return total, err
}

func writeJSON(path string, value any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
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
