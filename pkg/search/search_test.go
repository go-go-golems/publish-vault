package search

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func TestExtractTagQuery(t *testing.T) {
	tests := []struct {
		input string
		tag   string
		isTag bool
	}{
		{"#philosophy", "philosophy", true},
		{"# Philosophy", "philosophy", true},
		{"tag:stoicism", "stoicism", true},
		{"TAG:epistemology", "epistemology", true},
		{"tag: Epistemology ", "epistemology", true},
		{"#", "", false},
		{"tag:", "", false},
		{"philosophy", "", false},
		{"", "", false},
		{"#phi", "phi", true},
	}

	for _, tt := range tests {
		tag, ok := extractTagQuery(tt.input)
		if ok != tt.isTag {
			t.Errorf("extractTagQuery(%q): got ok=%v, want %v", tt.input, ok, tt.isTag)
		}
		if tag != tt.tag {
			t.Errorf("extractTagQuery(%q): got tag=%q, want %q", tt.input, tag, tt.tag)
		}
	}
}

func writeTestNote(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexProgressForMemoryAndPersistentIndexes(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "one.md", "# One\n\nalpha body")
	writeTestNote(t, root, "two.md", "# Two\n\nbeta body")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	for _, tt := range []struct {
		name  string
		build func(Options) (*Index, error)
	}{
		{name: "memory", build: func(options Options) (*Index, error) { return NewWithOptions(v, options) }},
		{name: "persistent", build: func(options Options) (*Index, error) {
			return NewPersistentWithOptions(v, filepath.Join(t.TempDir(), "index"), options)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var observed []IndexProgress
			index, err := tt.build(Options{ObserveIndexed: func(progress IndexProgress) { observed = append(observed, progress) }})
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			t.Cleanup(func() {
				if err := index.Close(); err != nil {
					t.Errorf("Close: %v", err)
				}
			})
			if len(observed) != 3 || observed[0].ProcessedDocuments != 0 || observed[0].TotalDocuments != 2 {
				t.Fatalf("initial/final progress = %#v", observed)
			}
			last := observed[len(observed)-1]
			if last.ProcessedDocuments != 2 || last.TotalDocuments != 2 || last.IndexedBytes == 0 {
				t.Fatalf("final progress = %#v", last)
			}
		})
	}
}

func TestBatchedIndexProgressAndValidation(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 5; i++ {
		writeTestNote(t, root, filepath.Join("notes", string(rune('a'+i-1))+".md"), "# Note\n\nshared body")
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	var observed []IndexProgress
	idx, err := NewPersistentWithOptions(v, filepath.Join(t.TempDir(), "index"), Options{
		BatchDocuments: 2,
		BatchBytes:     1 << 20,
		ObserveIndexed: func(progress IndexProgress) { observed = append(observed, progress) },
	})
	if err != nil {
		t.Fatalf("NewPersistentWithOptions: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	wantProcessed := []uint64{0, 2, 4, 5}
	if len(observed) != len(wantProcessed) {
		t.Fatalf("progress = %#v, want %d observations", observed, len(wantProcessed))
	}
	for i, want := range wantProcessed {
		if observed[i].ProcessedDocuments != want || observed[i].TotalDocuments != 5 {
			t.Errorf("progress[%d] = %#v, want processed=%d total=5", i, observed[i], want)
		}
	}
	if observed[len(observed)-1].IndexedBytes == 0 {
		t.Fatal("final indexed bytes = 0")
	}

	for _, options := range []Options{
		{BatchDocuments: 2},
		{BatchBytes: 1024},
	} {
		indexPath := filepath.Join(t.TempDir(), "invalid")
		if err := os.MkdirAll(indexPath, 0o755); err != nil {
			t.Fatal(err)
		}
		sentinel := filepath.Join(indexPath, "sentinel")
		if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewPersistentWithOptions(v, indexPath, options); err == nil {
			t.Fatalf("NewPersistentWithOptions(%#v) succeeded, want validation error", options)
		}
		if _, err := os.Stat(sentinel); err != nil {
			t.Fatalf("invalid options removed existing target before validation: %v", err)
		}
	}
}

func TestBatchedIndexFlushesOversizedDocumentsAlone(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "one.md", "# One\n\nfirst oversized body")
	writeTestNote(t, root, "two.md", "# Two\n\nsecond oversized body")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	var processed []uint64
	idx, err := NewPersistentWithOptions(v, filepath.Join(t.TempDir(), "index"), Options{
		BatchDocuments: 100,
		BatchBytes:     1,
		ObserveIndexed: func(progress IndexProgress) { processed = append(processed, progress.ProcessedDocuments) },
	})
	if err != nil {
		t.Fatalf("NewPersistentWithOptions: %v", err)
	}
	defer func() { _ = idx.Close() }()
	want := []uint64{0, 1, 2}
	if !equalUint64s(processed, want) {
		t.Fatalf("processed = %v, want %v", processed, want)
	}
}

func TestBatchedAndSingleDocumentIndexesHaveEquivalentSearchResults(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "one.md", "---\ntags: [philosophy]\n---\n# Alpha Design\n\nPersistent memory indexing details.")
	writeTestNote(t, root, "two.md", "---\ntags: [writing]\n---\n# Beta Report\n\nSearch indexing and memory measurements.")
	writeTestNote(t, root, "three.md", "# Gamma\n\nUnrelated body text.")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	baseline, err := NewPersistent(v, filepath.Join(t.TempDir(), "baseline"))
	if err != nil {
		t.Fatalf("NewPersistent baseline: %v", err)
	}
	defer func() { _ = baseline.Close() }()
	batched, err := NewPersistentWithOptions(v, filepath.Join(t.TempDir(), "batched"), Options{BatchDocuments: 2, BatchBytes: 1024})
	if err != nil {
		t.Fatalf("NewPersistentWithOptions batched: %v", err)
	}
	defer func() { _ = batched.Close() }()

	for _, query := range []string{"memory", "search indexing", "#philosophy", "tag:writing", "alp", "unrelated"} {
		want, err := baseline.Search(query, 20)
		if err != nil {
			t.Fatalf("baseline.Search(%q): %v", query, err)
		}
		got, err := batched.Search(query, 20)
		if err != nil {
			t.Fatalf("batched.Search(%q): %v", query, err)
		}
		assertEquivalentResults(t, query, got, want)
	}
}

func assertEquivalentResults(t *testing.T, query string, got, want []SearchResult) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Search(%q) result count = %d, want %d; got=%#v want=%#v", query, len(got), len(want), got, want)
	}
	gotBySlug := make(map[string]SearchResult, len(got))
	for _, result := range got {
		gotBySlug[result.Slug] = result
	}
	for _, expected := range want {
		actual, ok := gotBySlug[expected.Slug]
		if !ok {
			t.Errorf("Search(%q) missing slug %q; got=%#v", query, expected.Slug, got)
			continue
		}
		if actual.Title != expected.Title || actual.Excerpt != expected.Excerpt || !equalStrings(actual.Tags, expected.Tags) || math.Abs(actual.Score-expected.Score) > 1e-12 {
			t.Errorf("Search(%q) slug %q = %#v, want %#v", query, expected.Slug, actual, expected)
		}
	}
}

func equalUint64s(left, right []uint64) bool {
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

func TestSearchByTag(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "note-1.md", "---\ntags: [philosophy, stoicism]\n---\n# Note One\n\nAbout stoicism.")
	writeTestNote(t, root, "note-2.md", "---\ntags: [philosophy, epistemology]\n---\n# Note Two\n\nAbout epistemology.")
	writeTestNote(t, root, "note-3.md", "---\ntags: [writing]\n---\n# Note Three\n\nAbout writing.")
	writeTestNote(t, root, "note-4.md", "# Note Four\n\nNo tags.")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	idx, err := New(v)
	if err != nil {
		t.Fatalf("Failed to create search index: %v", err)
	}

	tests := []struct {
		query     string
		wantSlugs map[string]bool
	}{
		{"#philosophy", map[string]bool{"note-1": true, "note-2": true}},
		{"#stoicism", map[string]bool{"note-1": true}},
		{"#writing", map[string]bool{"note-3": true}},
		{"tag:epistemology", map[string]bool{"note-2": true}},
		{"#nonexistent", map[string]bool{}},
	}

	for _, tt := range tests {
		results, err := idx.Search(tt.query, 20)
		if err != nil {
			t.Errorf("Search(%q): error: %v", tt.query, err)
			continue
		}

		gotSlugs := make(map[string]bool)
		for _, r := range results {
			gotSlugs[r.Slug] = true
		}

		if len(gotSlugs) != len(tt.wantSlugs) {
			t.Errorf("Search(%q): got %d results %v, want %d results", tt.query, len(gotSlugs), gotSlugs, len(tt.wantSlugs))
			continue
		}

		for slug := range tt.wantSlugs {
			if !gotSlugs[slug] {
				t.Errorf("Search(%q): missing expected slug %q in results %v", tt.query, slug, gotSlugs)
			}
		}
	}
}

func TestSearchByTagPrefix(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "note-1.md", "---\ntags: [philosophy]\n---\n# Note One\n\nAbout philosophy.")
	writeTestNote(t, root, "note-2.md", "---\ntags: [photography]\n---\n# Note Two\n\nAbout photography.")
	writeTestNote(t, root, "note-3.md", "---\ntags: [writing]\n---\n# Note Three\n\nAbout writing.")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	idx, err := New(v)
	if err != nil {
		t.Fatalf("Failed to create search index: %v", err)
	}

	// Short prefix "phi" should match "philosophy" via prefix query
	results, err := idx.Search("#phi", 20)
	if err != nil {
		t.Fatalf("Search(#phi): error: %v", err)
	}

	gotSlugs := make(map[string]bool)
	for _, r := range results {
		gotSlugs[r.Slug] = true
	}

	if !gotSlugs["note-1"] {
		t.Errorf("Search(#phi): expected note-1 in results, got %v", results)
	}
	if gotSlugs["note-2"] {
		t.Errorf("Search(#phi): did not expect note-2 (photography) in results")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "note.md", "# Note\n\nBody")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	idx, err := New(v)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("Close() first error = %v", err)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("Close() second error = %v", err)
	}
}

func TestClosedIndexOperationsReturnErrClosed(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "note.md", "# Note\n\nBody")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	idx, err := New(v)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := idx.Index(vault.SearchDocument{Slug: "note", Title: "Note", Body: "Body"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Index() error = %v, want ErrClosed", err)
	}
	if err := idx.Delete("note"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Delete() error = %v, want ErrClosed", err)
	}
	if _, err := idx.Search("Body", 10); !errors.Is(err, ErrClosed) {
		t.Fatalf("Search() error = %v, want ErrClosed", err)
	}
}

func TestNewPersistentRebuildsFreshWithoutStaleDeletedDocuments(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index")

	root1 := t.TempDir()
	writeTestNote(t, root1, "gone.md", "# Gone\n\nvanishingterm")
	v1, err := vault.New(root1)
	if err != nil {
		t.Fatalf("vault.New(root1) error = %v", err)
	}
	idx1, err := NewPersistent(v1, indexPath)
	if err != nil {
		t.Fatalf("NewPersistent(v1) error = %v", err)
	}
	results, err := idx1.Search("vanishingterm", 10)
	if err != nil {
		t.Fatalf("Search(v1) error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected initial persistent index to find vanishingterm")
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("idx1.Close() error = %v", err)
	}

	root2 := t.TempDir()
	writeTestNote(t, root2, "kept.md", "# Kept\n\nordinary content")
	v2, err := vault.New(root2)
	if err != nil {
		t.Fatalf("vault.New(root2) error = %v", err)
	}
	idx2, err := NewPersistent(v2, indexPath)
	if err != nil {
		t.Fatalf("NewPersistent(v2) error = %v", err)
	}
	defer func() { _ = idx2.Close() }()
	results, err = idx2.Search("vanishingterm", 10)
	if err != nil {
		t.Fatalf("Search(v2) error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("stale deleted document remained searchable: %#v", results)
	}
}

func TestRegularSearchUnchanged(t *testing.T) {
	root := t.TempDir()
	writeTestNote(t, root, "note-1.md", "---\ntags: [philosophy]\n---\n# Philosophy Basics\n\nIntroduction to philosophy.")
	writeTestNote(t, root, "note-2.md", "---\ntags: [writing]\n---\n# Writing Tips\n\nHow to write well.")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	idx, err := New(v)
	if err != nil {
		t.Fatalf("Failed to create search index: %v", err)
	}

	// Regular search (no # prefix) should still work
	results, err := idx.Search("philosophy", 20)
	if err != nil {
		t.Fatalf("Search(philosophy): error: %v", err)
	}

	if len(results) == 0 {
		t.Error("Search(philosophy): expected results, got none")
	}
}
