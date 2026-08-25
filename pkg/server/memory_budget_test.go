package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	measurebudget "github.com/go-go-golems/measure/pkg/budget"
	measurereport "github.com/go-go-golems/measure/pkg/report"
	measuretrace "github.com/go-go-golems/measure/pkg/trace"
)

type fixtureBudgetConfig struct {
	SchemaVersion int `json:"schema_version"`
	Fixture       struct {
		Documents    int `json:"documents"`
		PayloadBytes int `json:"payload_bytes_per_document"`
	} `json:"fixture"`
	Budgets []struct {
		Phase     string `json:"phase"`
		Metric    string `json:"metric"`
		Condition string `json:"condition"`
	} `json:"budgets"`
}

func TestGeneratedFixtureMemoryBudget(t *testing.T) {
	configData, err := os.ReadFile(filepath.Join("testdata", "generated-fixture-memory-budget.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config fixtureBudgetConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	if config.SchemaVersion != 1 || config.Fixture.Documents <= 0 || config.Fixture.PayloadBytes <= 0 {
		t.Fatalf("invalid fixture config: %#v", config)
	}

	root, traceDir := t.TempDir(), t.TempDir()
	payload := strings.Repeat("bounded-memory-payload ", config.Fixture.PayloadBytes/23+1)[:config.Fixture.PayloadBytes]
	for i := 0; i < config.Fixture.Documents; i++ {
		content := fmt.Sprintf("# Generated %03d\n\nfixture-token-%03d %s", i, i, payload)
		writeVaultNote(t, root, fmt.Sprintf("batch/%03d.md", i), content)
	}
	measurement, err := newRuntimeMeasurement(100*time.Millisecond, traceDir, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	state, err := NewRuntimeStateWithOptions(root, RuntimeOptions{
		SearchIndexPath: filepath.Join(t.TempDir(), "persistent-search"),
		measurement:     measurement,
	})
	if err != nil {
		t.Fatalf("load generated fixture: %v", err)
	}
	vault, index := state.Snapshot()
	t.Cleanup(func() {
		if err := index.Close(); err != nil {
			t.Errorf("close generated fixture search index: %v", err)
		}
	})
	if vault.Count() != config.Fixture.Documents {
		t.Fatalf("notes = %d, want %d", vault.Count(), config.Fixture.Documents)
	}
	hits, err := index.Search("fixture-token-159", 5)
	if err != nil || len(hits) == 0 {
		t.Fatalf("fixture search correctness: hits=%d err=%v", len(hits), err)
	}

	traceFile, err := os.Open(singleGlob(t, filepath.Join(traceDir, "*.jsonl")))
	if err != nil {
		t.Fatal(err)
	}
	events, err := measuretrace.ReadAll(traceFile)
	_ = traceFile.Close()
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	summary, err := measurereport.Summarize(events)
	if err != nil {
		t.Fatalf("summarize trace: %v", err)
	}
	checks := make([]measurebudget.Check, 0, len(config.Budgets))
	for _, configured := range config.Budgets {
		condition, err := measurebudget.ParseCondition(configured.Condition)
		if err != nil {
			t.Fatalf("parse condition %q: %v", configured.Condition, err)
		}
		checks = append(checks, measurebudget.Check{Phase: configured.Phase, Metric: configured.Metric, Condition: condition})
	}
	evaluations, err := measurebudget.Evaluate(summary, checks)
	if err != nil {
		t.Fatalf("evaluate budgets: %v", err)
	}
	for _, evaluation := range evaluations {
		t.Logf("budget phase=%q metric=%s observed=%d threshold=%s%d passed=%t", evaluation.Check.Phase, evaluation.Check.Metric, evaluation.Observed, evaluation.Check.Condition.Operator, evaluation.Check.Condition.Bytes, evaluation.Passed)
		if !evaluation.Passed {
			t.Errorf("budget failed: phase=%q metric=%s observed=%d condition=%s%d", evaluation.Check.Phase, evaluation.Check.Metric, evaluation.Observed, evaluation.Check.Condition.Operator, evaluation.Check.Condition.Bytes)
		}
	}
}
