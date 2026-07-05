package search

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"retro-obsidian-publish/internal/vault"
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
