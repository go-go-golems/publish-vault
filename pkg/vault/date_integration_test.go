package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func writeVaultNote(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNoteDatesPopulatedFromFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Dated.md",
		"---\ncreated: 2024-01-15\nupdated: 2024-02-20T09:00:00Z\ntitle: Dated\n---\n# Dated\n")
	writeVaultNote(t, root, "Plain.md", "# Plain\n")
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	dated, ok := v.GetNote("dated")
	if !ok {
		t.Fatal("dated note not found")
	}
	if dated.Dates.Created == nil || dated.Dates.Created.APIValue() != "2024-01-15" {
		t.Fatalf("created date = %+v, want 2024-01-15", dated.Dates.Created)
	}
	if dated.Dates.Updated == nil || dated.Dates.Updated.APIValue() != "2024-02-20T09:00:00Z" {
		t.Fatalf("updated date = %+v, want 2024-02-20T09:00:00Z", dated.Dates.Updated)
	}
	if kind, d := dated.Dates.Display(); kind != NoteDateUpdated || d == nil {
		t.Fatalf("display = %q/%v, want updated", kind, d)
	}

	plain, ok := v.GetNote("plain")
	if !ok {
		t.Fatal("plain note not found")
	}
	if plain.Dates.Created != nil || plain.Dates.Updated != nil {
		t.Fatalf("plain note should have no dates, got %+v", plain.Dates)
	}
	if kind, d := plain.Dates.Display(); kind != "" || d != nil {
		t.Fatalf("plain display = %q/%v, want absent", kind, d)
	}
}

func TestSearchDocumentCarriesDates(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Dated.md",
		"---\ncreated: 2024-01-15\nupdated: 2024-02-20T09:00:00Z\ntitle: Dated\n---\n# Dated\n")
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, ok := v.GetNote("dated")
	if !ok {
		t.Fatal("dated note not found")
	}
	doc, err := v.SearchDocument(note)
	if err != nil {
		t.Fatalf("SearchDocument() error = %v", err)
	}
	if doc.CreatedAt == nil || !doc.CreatedAt.Equal(note.Dates.Created.Value) {
		t.Fatalf("doc.CreatedAt = %v, want %v", doc.CreatedAt, note.Dates.Created.Value)
	}
	if doc.UpdatedAt == nil || !doc.UpdatedAt.Equal(note.Dates.Updated.Value) {
		t.Fatalf("doc.UpdatedAt = %v, want %v", doc.UpdatedAt, note.Dates.Updated.Value)
	}
	if doc.DisplayAt == nil || !doc.DisplayAt.Equal(note.Dates.Updated.Value) {
		t.Fatalf("doc.DisplayAt = %v, want updated %v", doc.DisplayAt, note.Dates.Updated.Value)
	}
	if doc.DateKind != "updated" {
		t.Fatalf("doc.DateKind = %q, want updated", doc.DateKind)
	}
}

func TestInvalidDateCountsAggregated(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Bad.md", "---\ncreated: January someday\ntitle: Bad\n---\n# Bad\n")
	writeVaultNote(t, root, "WrongType.md", "---\ncreated: 2024\ntitle: WrongType\n---\n# WrongType\n")
	writeVaultNote(t, root, "Good.md", "---\ncreated: 2024-01-15\ntitle: Good\n---\n# Good\n")
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	counts := v.InvalidDateCounts()
	if counts["created:invalid_format"] != 1 {
		t.Fatalf("invalid_format count = %d, want 1 (counts=%v)", counts["created:invalid_format"], counts)
	}
	if counts["created:wrong_type"] != 1 {
		t.Fatalf("wrong_type count = %d, want 1 (counts=%v)", counts["created:wrong_type"], counts)
	}
}
