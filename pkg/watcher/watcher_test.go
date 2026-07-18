package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func TestApplyKeepsSearchIndexInSync(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "Index.md")
	if err := os.WriteFile(notePath, []byte("# Index\n\nOriginal phrase."), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	vw := &VaultWatcher{vault: v, search: si}

	if err := os.WriteFile(notePath, []byte("# Index\n\nUpdated unique phrase."), 0o644); err != nil {
		t.Fatal(err)
	}
	vw.apply(notePath, fsnotify.Write)

	results, err := si.Search("unique", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].Slug != "index" {
		t.Fatalf("Search(unique) = %#v, want index hit", results)
	}

	vw.apply(notePath, fsnotify.Remove)
	results, err = si.Search("unique", 10)
	if err != nil {
		t.Fatalf("Search() after remove error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search(unique) after remove = %#v, want no hits", results)
	}
}

// TestApplySkipsIgnoredPath verifies that writing to a file under an ignored
// directory does not add it to the vault or the search index. apply must observe
// the vault's ErrIgnored and no-op rather than indexing the note.
func TestApplySkipsIgnoredPath(t *testing.T) {
	root := t.TempDir()
	// Published note so the vault and search index are non-empty.
	indexPath := filepath.Join(root, "Index.md")
	if err := os.WriteFile(indexPath, []byte("# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Ignored authoring scaffolding.
	ignoredDir := filepath.Join(root, "ttmp/_guidelines")
	if err := os.MkdirAll(ignoredDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ignoredPath := filepath.Join(ignoredDir, "Style.md")
	if err := os.WriteFile(ignoredPath, []byte("# Style\n\nIgnored unique phrase."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vault-ignore"), []byte("ttmp/_guidelines/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	if _, ok := v.GetNote("ttmp/_guidelines/style"); ok {
		t.Fatalf("ignored note must not be loaded by New")
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	vw := &VaultWatcher{vault: v, search: si}

	// Simulate an fsnotify Write for the ignored file.
	vw.apply(ignoredPath, fsnotify.Write)

	// The ignored note must not have entered the vault or the search index.
	if _, ok := v.GetNote("ttmp/_guidelines/style"); ok {
		t.Errorf("ignored note must not be added to the vault by apply")
	}
	results, err := si.Search("Ignored", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search(Ignored) = %#v, want no hits (path is ignored)", results)
	}
}
