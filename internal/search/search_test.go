package search

import (
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
