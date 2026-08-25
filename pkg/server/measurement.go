package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/measure/pkg/collector"
	measurepkg "github.com/go-go-golems/measure/pkg/measure"
	measureprom "github.com/go-go-golems/measure/pkg/prometheus"
	measuresink "github.com/go-go-golems/measure/pkg/sink"
	measuretrace "github.com/go-go-golems/measure/pkg/trace"
	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

const defaultMeasurementInterval = time.Second

var runtimeMeasurementPhases = []string{
	"resolve_root", "vault_walk_parse", "vault_normalize", "wiki_link_index",
	"backlinks", "render_html", "search_index", "index_publish", "snapshot_swap",
}

type runtimeMeasurement struct {
	interval time.Duration
	traceDir string
	exporter *measureprom.Exporter
	handler  http.Handler
	sequence atomic.Uint64
}

func newRuntimeMeasurement(interval time.Duration, traceDir, environment string) (*runtimeMeasurement, error) {
	if interval == 0 {
		interval = defaultMeasurementInterval
	}
	if interval < 100*time.Millisecond {
		return nil, fmt.Errorf("measure interval must be at least 100ms")
	}
	if traceDir != "" {
		absolute, err := filepath.Abs(traceDir)
		if err != nil {
			return nil, fmt.Errorf("resolve measure trace dir: %w", err)
		}
		if err := os.MkdirAll(absolute, 0o700); err != nil {
			return nil, fmt.Errorf("create measure trace dir: %w", err)
		}
		traceDir = absolute
	}
	registry, exporter, err := measureprom.NewRegistry(measureprom.Options{
		Labels:        measureprom.Labels{Application: "publish-vault", Environment: environment},
		AllowedPhases: runtimeMeasurementPhases,
	})
	if err != nil {
		return nil, err
	}
	handler, err := measureprom.Handler(registry)
	if err != nil {
		return nil, err
	}
	return &runtimeMeasurement{interval: interval, traceDir: traceDir, exporter: exporter, handler: handler}, nil
}

func (m *runtimeMeasurement) startRun(kind string) *measurementRun {
	return m.startRunWithExporter(kind, true)
}

func (m *runtimeMeasurement) startTraceOnlyRun(kind string) *measurementRun {
	return m.startRunWithExporter(kind, false)
}

func (m *runtimeMeasurement) startRunWithExporter(kind string, includeExporter bool) *measurementRun {
	if m == nil || (!includeExporter && m.traceDir == "") {
		return nil
	}
	number := m.sequence.Add(1)
	runID := fmt.Sprintf("publish-vault-%s-%d", kind, number)
	var sinks []measuresink.Named
	if includeExporter {
		sinks = append(sinks, measuresink.Named{Name: "prometheus", Sink: m.exporter, Required: true})
	}
	var traceFile *os.File
	var tracePath, receiptPath string
	if m.traceDir != "" {
		stem := fmt.Sprintf("%s-%06d-%s", time.Now().UTC().Format("20060102T150405.000000000Z"), number, kind)
		tracePath = filepath.Join(m.traceDir, stem+".jsonl")
		receiptPath = filepath.Join(m.traceDir, stem+".receipt.json")
		file, err := os.OpenFile(tracePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			log.Printf("measure: start %s trace: %v", kind, err)
			return nil
		}
		jsonl, err := measuresink.NewJSONL(file)
		if err != nil {
			_ = file.Close()
			_ = os.Remove(tracePath)
			log.Printf("measure: create %s JSONL sink: %v", kind, err)
			return nil
		}
		traceFile = file
		sinks = append(sinks, measuresink.Named{Name: "jsonl", Sink: jsonl, Required: true})
	}
	recorder, err := measurepkg.NewRecorder(measurepkg.Options{
		RunName: kind, Interval: m.interval, Target: collector.SelfTarget(),
		Reader: collector.NewSystemReader(), Sink: &measuresink.Fanout{Sinks: sinks},
		NewRunID: func() (string, error) { return runID, nil },
	})
	if err != nil {
		closeAndRemove(traceFile, tracePath)
		log.Printf("measure: create %s recorder: %v", kind, err)
		return nil
	}
	ctx := context.Background()
	if err := recorder.Start(ctx); err != nil {
		closeAndRemove(traceFile, tracePath)
		log.Printf("measure: start %s recorder: %v", kind, err)
		return nil
	}
	sampleCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- recorder.Run(sampleCtx) }()
	return &measurementRun{kind: kind, recorder: recorder, cancel: cancel, done: done, traceFile: traceFile, tracePath: tracePath, receiptPath: receiptPath}
}

type measurementRun struct {
	kind                                 string
	recorder                             *measurepkg.Recorder
	cancel                               context.CancelFunc
	done                                 chan error
	traceFile                            *os.File
	tracePath, receiptPath               string
	phase                                *measurepkg.Phase
	phaseName                            string
	lastNotes, lastBytes, lastTotalBytes uint64
	finishOnce                           sync.Once
}

func (r *measurementRun) switchPhase(name string, total uint64, unit string) {
	if r == nil || r.phaseName == name {
		return
	}
	r.endPhase(measuretrace.Succeeded())
	phase, err := r.recorder.BeginPhase(context.Background(), name, measurepkg.PhaseOptions{Total: total, Unit: unit})
	if err != nil {
		log.Printf("measure: begin phase %s: %v", name, err)
		return
	}
	r.phase, r.phaseName = phase, name
	r.lastNotes, r.lastBytes, r.lastTotalBytes = 0, 0, 0
}

func (r *measurementRun) setProgress(processed uint64) {
	if r == nil || r.phase == nil {
		return
	}
	if err := r.phase.SetProgress(processed); err != nil {
		log.Printf("measure: phase %s progress: %v", r.phaseName, err)
	}
}

func (r *measurementRun) annotateProgress() {
	if r == nil || r.phase == nil {
		return
	}
	attributes := []measuretrace.Attribute{
		{Key: "processed_notes", Value: strconv.FormatUint(r.lastNotes, 10)},
		{Key: "processed_bytes", Value: strconv.FormatUint(r.lastBytes, 10)},
		{Key: "total_bytes", Value: strconv.FormatUint(r.lastTotalBytes, 10)},
	}
	if err := r.recorder.Annotate(context.Background(), attributes...); err != nil {
		log.Printf("measure: annotate phase %s: %v", r.phaseName, err)
	}
}

func (r *measurementRun) endPhase(result measuretrace.Result) {
	if r == nil || r.phase == nil {
		return
	}
	r.annotateProgress()
	if _, err := r.phase.End(context.Background(), result); err != nil {
		log.Printf("measure: end phase %s: %v", r.phaseName, err)
	}
	r.phase, r.phaseName = nil, ""
}

func (r *measurementRun) observeVault(progress vault.LoadProgress) {
	if r == nil {
		return
	}
	r.switchPhase(string(progress.Stage), progress.TotalNotes, "notes")
	r.lastNotes = progress.ProcessedNotes
	r.lastBytes, r.lastTotalBytes = progress.ProcessedBytes, progress.TotalBytes
	r.setProgress(progress.ProcessedNotes)
}

func (r *measurementRun) observeSearch(progress search.IndexProgress) {
	if r == nil {
		return
	}
	r.switchPhase("search_index", progress.TotalDocuments, "documents")
	r.lastNotes = progress.ProcessedDocuments
	r.lastBytes = progress.IndexedBytes
	r.setProgress(progress.ProcessedDocuments)
}

func (r *measurementRun) finish(operationErr error) {
	if r == nil {
		return
	}
	r.finishOnce.Do(func() {
		result := measuretrace.Succeeded()
		if operationErr != nil {
			result = measuretrace.Failed("operation_failed")
		}
		r.endPhase(result)
		r.cancel()
		if err := <-r.done; err != nil {
			log.Printf("measure: sample %s: %v", r.kind, err)
		}
		receipt, err := r.recorder.Finish(context.Background(), result)
		if err != nil {
			log.Printf("measure: finish %s: %v", r.kind, err)
		}
		if r.traceFile != nil {
			if closeErr := r.traceFile.Close(); closeErr != nil {
				log.Printf("measure: close trace %s: %v", filepath.Base(r.tracePath), closeErr)
			}
			if err == nil {
				if writeErr := writeReceiptAtomic(r.receiptPath, receipt); writeErr != nil {
					log.Printf("measure: write receipt %s: %v", filepath.Base(r.receiptPath), writeErr)
				}
			}
		}
	})
}

func writeReceiptAtomic(path string, receipt measuretrace.Receipt) error {
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := measuretrace.EncodeReceipt(file, receipt); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return os.Rename(temporary, path)
}

func closeAndRemove(file *os.File, path string) {
	if file != nil {
		_ = file.Close()
	}
	if path != "" {
		_ = os.Remove(path)
	}
}
