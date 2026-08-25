// compare-real-search builds current and candidate persistent indexes over the
// same vault, compares a bounded query corpus, and emits only content-free
// hashes and counts. It never writes result slugs, titles, excerpts, or tags.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type queryResult struct {
	QuerySHA256  string  `json:"query_sha256"`
	ResultCount  int     `json:"result_count"`
	MaxScoreDiff float64 `json:"max_score_difference"`
	Passed       bool    `json:"passed"`
}

type report struct {
	SchemaVersion  int           `json:"schema_version"`
	Queries        int           `json:"queries"`
	BatchDocuments uint64        `json:"batch_documents"`
	BatchBytes     uint64        `json:"batch_bytes"`
	Results        []queryResult `json:"results"`
	Passed         bool          `json:"passed"`
}

const comparisonLimit = 10_000

var queries = []string{
	"memory", "search", "index", "architecture", "implementation", "performance",
	"golang", "project", "article", "design", "system", "documentation",
	"memory architecture", "search index", "implementation guide", "#project",
	"#article", "tag:project", "tag:article", "pub",
}

func main() {
	vaultDir := flag.String("vault", "", "vault directory")
	workDir := flag.String("work-dir", "", "temporary index parent")
	output := flag.String("output", "", "content-free report path")
	flag.Parse()
	if *vaultDir == "" || *workDir == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "--vault, --work-dir, and --output are required")
		os.Exit(2)
	}
	if err := run(*vaultDir, *workDir, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(vaultDir, workDir, output string) error {
	if err := os.RemoveAll(workDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workDir) }()
	loaded, err := vault.New(vaultDir)
	if err != nil {
		return fmt.Errorf("load vault: %w", err)
	}
	baseline, err := search.NewPersistent(loaded, filepath.Join(workDir, "current"))
	if err != nil {
		return fmt.Errorf("build current index: %w", err)
	}
	defer func() { _ = baseline.Close() }()
	candidate, err := search.NewPersistentWithOptions(loaded, filepath.Join(workDir, "candidate"), search.Options{BatchDocuments: 16, BatchBytes: 1 << 20})
	if err != nil {
		return fmt.Errorf("build candidate index: %w", err)
	}
	defer func() { _ = candidate.Close() }()

	value := report{SchemaVersion: 1, Queries: len(queries), BatchDocuments: 16, BatchBytes: 1 << 20, Passed: true}
	for _, query := range queries {
		want, err := baseline.Search(query, comparisonLimit)
		if err != nil {
			return fmt.Errorf("current search %q: %w", query, err)
		}
		got, err := candidate.Search(query, comparisonLimit)
		if err != nil {
			return fmt.Errorf("candidate search %q: %w", query, err)
		}
		maxDiff, equivalent := compareResults(got, want)
		result := queryResult{QuerySHA256: hash(query), ResultCount: len(want), MaxScoreDiff: maxDiff, Passed: equivalent}
		value.Results = append(value.Results, result)
		value.Passed = value.Passed && equivalent
	}
	if err := writeJSON(output, value); err != nil {
		return err
	}
	if !value.Passed {
		return fmt.Errorf("one or more search queries differed; see content-free report")
	}
	return nil
}

func compareResults(got, want []search.SearchResult) (float64, bool) {
	if len(got) != len(want) {
		return 0, false
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Slug < got[j].Slug })
	sort.Slice(want, func(i, j int) bool { return want[i].Slug < want[j].Slug })
	var maxDiff float64
	for i := range want {
		diff := math.Abs(got[i].Score - want[i].Score)
		if diff > maxDiff {
			maxDiff = diff
		}
		if got[i].Slug != want[i].Slug || got[i].Title != want[i].Title || got[i].Excerpt != want[i].Excerpt || !equalStrings(got[i].Tags, want[i].Tags) || diff > 1e-12 {
			return maxDiff, false
		}
	}
	return maxDiff, true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
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
