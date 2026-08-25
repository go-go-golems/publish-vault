package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	measuretrace "github.com/go-go-golems/measure/pkg/trace"
)

func TestRuntimeMeasurementTraceReceiptProgressAndMetrics(t *testing.T) {
	root, traceDir := t.TempDir(), t.TempDir()
	for i := 0; i < 3; i++ {
		writeVaultNote(t, root, filepath.Join("folder", string(rune('a'+i))+".md"), "# Note\n\nbody payload")
	}
	measurement, err := newRuntimeMeasurement(100*time.Millisecond, traceDir, "test")
	if err != nil {
		t.Fatalf("newRuntimeMeasurement: %v", err)
	}
	state, err := NewRuntimeStateWithOptions(root, RuntimeOptions{
		SearchIndexPath: filepath.Join(t.TempDir(), "search"), measurement: measurement,
	})
	if err != nil {
		t.Fatalf("NewRuntimeStateWithOptions: %v", err)
	}
	if vault, _ := state.Snapshot(); vault.Count() != 3 {
		t.Fatalf("notes = %d, want 3", vault.Count())
	}
	receiptPath := singleGlob(t, filepath.Join(traceDir, "*.receipt.json"))
	receiptFile, err := os.Open(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := measuretrace.DecodeReceipt(receiptFile)
	_ = receiptFile.Close()
	if err != nil {
		t.Fatalf("DecodeReceipt: %v", err)
	}
	if receipt.Result.Status != measuretrace.StatusSucceeded {
		t.Fatalf("receipt status = %s", receipt.Result.Status)
	}
	var names []string
	for _, phase := range receipt.Phases {
		names = append(names, phase.Name)
		if phase.Name == "search_index" && (phase.Progress.Processed != 3 || phase.Progress.Total != 3) {
			t.Fatalf("search progress = %#v", phase.Progress)
		}
	}
	wantNames := []string{"resolve_root", "vault_walk_parse", "vault_normalize", "wiki_link_index", "backlinks", "render_html", "search_index", "index_publish", "snapshot_swap"}
	if strings.Join(names, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("phases = %v, want %v", names, wantNames)
	}
	tracePath := singleGlob(t, filepath.Join(traceDir, "*.jsonl"))
	traceFile, err := os.Open(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	events, err := measuretrace.ReadAll(traceFile)
	_ = traceFile.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	digits := regexp.MustCompile(`^[0-9]+$`)
	annotations := 0
	for _, event := range events {
		if event.EventType != measuretrace.EventAnnotation {
			continue
		}
		annotations++
		for _, attribute := range event.Attributes {
			if !digits.MatchString(attribute.Value) {
				t.Fatalf("annotation is not content-free: %#v", attribute)
			}
		}
	}
	if annotations < len(wantNames) {
		t.Fatalf("annotations = %d, want at least %d", annotations, len(wantNames))
	}
	request, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	response := &responseRecorder{header: http.Header{}}
	measurement.handler.ServeHTTP(response, request)
	body := response.body.String()
	if response.status != http.StatusOK || !strings.Contains(body, `measure_runs_total{application="publish-vault",environment="test",result="succeeded"} 1`) || !strings.Contains(body, `measure_phase_progress{application="publish-vault",environment="test",phase="search_index"} 3`) {
		t.Fatalf("unexpected metrics status=%d body=%s", response.status, body)
	}
}

func TestRuntimeMeasurementRecordsFailedLoadAndReloadRelease(t *testing.T) {
	t.Run("failed load", func(t *testing.T) {
		traceDir := t.TempDir()
		measurement, err := newRuntimeMeasurement(100*time.Millisecond, traceDir, "test")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NewRuntimeStateWithOptions(filepath.Join(t.TempDir(), "missing"), RuntimeOptions{measurement: measurement}); err == nil {
			t.Fatal("missing vault accepted")
		}
		receipt := readReceipt(t, singleGlob(t, filepath.Join(traceDir, "*.receipt.json")))
		if receipt.Result.Status != measuretrace.StatusFailed || len(receipt.Phases) != 1 || receipt.Phases[0].Name != "resolve_root" {
			t.Fatalf("failed receipt = %#v", receipt)
		}
	})

	t.Run("reload and delayed release", func(t *testing.T) {
		root, traceDir := t.TempDir(), t.TempDir()
		writeVaultNote(t, root, "one.md", "# One")
		measurement, err := newRuntimeMeasurement(100*time.Millisecond, traceDir, "test")
		if err != nil {
			t.Fatal(err)
		}
		state, err := NewRuntimeStateWithOptions(root, RuntimeOptions{measurement: measurement})
		if err != nil {
			t.Fatal(err)
		}
		previousDelay := oldSnapshotCloseDelay
		oldSnapshotCloseDelay = 0
		t.Cleanup(func() { oldSnapshotCloseDelay = previousDelay })
		writeVaultNote(t, root, "two.md", "# Two")
		if err := state.Reload(); err != nil {
			t.Fatalf("Reload: %v", err)
		}
		var receipts []string
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			receipts, _ = filepath.Glob(filepath.Join(traceDir, "*.receipt.json"))
			if len(receipts) == 3 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if len(receipts) != 3 {
			t.Fatalf("receipt files = %v, want initial/reload/release", receipts)
		}
		seen := map[string]measuretrace.Receipt{}
		for _, path := range receipts {
			receipt := readReceipt(t, path)
			seen[receipt.RunName] = receipt
		}
		if seen["reload"].Result.Status != measuretrace.StatusSucceeded || seen["snapshot_release"].Result.Status != measuretrace.StatusSucceeded {
			t.Fatalf("reload/release receipts = %#v", seen)
		}
		if phases := seen["snapshot_release"].Phases; len(phases) != 1 || phases[0].Name != "old_snapshot_release" || phases[0].Progress.Processed != 1 {
			t.Fatalf("release phases = %#v", phases)
		}
	})
}

func readReceipt(t *testing.T, path string) measuretrace.Receipt {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	receipt, err := measuretrace.DecodeReceipt(file)
	if err != nil {
		t.Fatalf("DecodeReceipt: %v", err)
	}
	return receipt
}

func TestPrivateMetricsShutdownHonorsExpiredContext(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	shutdown, address, err := startPrivateMetrics("127.0.0.1:0", handler)
	if err != nil {
		t.Fatal(err)
	}
	requestDone := make(chan struct{})
	go func() {
		response, _ := http.Get("http://" + address + "/metrics")
		if response != nil {
			_ = response.Body.Close()
		}
		close(requestDone)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("metrics handler did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	before := time.Now()
	err = shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error = %v", err)
	}
	if elapsed := time.Since(before); elapsed > time.Second {
		t.Fatalf("shutdown ignored deadline: %s", elapsed)
	}
	close(release)
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("blocked request did not exit after release")
	}
}

func TestPrivateMetricsListenerIsSeparateAndServesExporter(t *testing.T) {
	measurement, err := newRuntimeMeasurement(time.Second, "", "test")
	if err != nil {
		t.Fatal(err)
	}
	shutdown, address, err := startPrivateMetrics("127.0.0.1:0", measurement.handler)
	if err != nil {
		t.Fatalf("startPrivateMetrics: %v", err)
	}
	response, err := http.Get("http://" + address + "/metrics")
	if err != nil {
		t.Fatalf("GET metrics: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "measure_run_active") {
		t.Fatalf("metrics status=%d body=%s", response.StatusCode, body)
	}
	missing, err := http.Get("http://" + address + "/api/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("private listener exposed non-metrics route: %d", missing.StatusCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func singleGlob(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("glob %q = %v, want exactly one", pattern, matches)
	}
	return matches[0]
}

type responseRecorder struct {
	header http.Header
	body   strings.Builder
	status int
}

func (r *responseRecorder) Header() http.Header { return r.header }
func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}
func (r *responseRecorder) WriteHeader(status int) { r.status = status }
