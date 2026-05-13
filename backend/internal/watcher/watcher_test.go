package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"

	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
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
