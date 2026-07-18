package vaultdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/publish-vault/internal/api"
	"github.com/go-go-golems/publish-vault/internal/search"
	"github.com/go-go-golems/publish-vault/internal/vault"
)

type staticProvider struct {
	v  *vault.Vault
	si *search.Index
}

func (p staticProvider) Snapshot() (*vault.Vault, *search.Index) { return p.v, p.si }

func newTestVM(t *testing.T) *goja.Runtime {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("Index.md", "---\ntitle: Index\ntags: [home]\n---\n# Index\n\nWelcome to [[Second Note]].\n")
	write("Second Note.md", "# Second Note\n\nSearchable unique phrase.\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}

	vm := goja.New()
	reg := require.NewRegistry()
	Register(reg, staticProvider{v: v, si: si}, api.PublicConfig{VaultName: "TestVault", PageTitle: "Test Vault"})
	reg.Enable(vm)
	return vm
}

// runJSON executes a script and returns its JSON-marshaled result.
func runJSON(t *testing.T, vm *goja.Runtime, script string) []byte {
	t.Helper()
	value, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("RunString(%q) error = %v", script, err)
	}
	data, err := json.Marshal(value.Export())
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	return data
}

func TestConfig(t *testing.T) {
	vm := newTestVM(t)
	var cfg struct {
		VaultName string `json:"vaultName"`
		PageTitle string `json:"pageTitle"`
		Notes     int    `json:"notes"`
	}
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").config()`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.VaultName != "TestVault" || cfg.PageTitle != "Test Vault" || cfg.Notes != 2 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestNotesShape(t *testing.T) {
	vm := newTestVM(t)
	var notes []map[string]any
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").notes()`), &notes); err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	for _, key := range []string{"slug", "title", "tags", "excerpt", "modTime", "path"} {
		if _, ok := notes[0][key]; !ok {
			t.Fatalf("note list item missing %q: %v", key, notes[0])
		}
	}
}

func TestNoteAndMissingNote(t *testing.T) {
	vm := newTestVM(t)
	var note map[string]any
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").note("index")`), &note); err != nil {
		t.Fatal(err)
	}
	if note["title"] != "Index" {
		t.Fatalf("unexpected note title: %v", note["title"])
	}
	if _, ok := note["html"]; !ok {
		t.Fatalf("full note missing html: %v", note)
	}

	missing := runJSON(t, vm, `require("vault.data").note("nope") === null`)
	if string(missing) != "true" {
		t.Fatalf("missing note should be null, got %s", missing)
	}
}

func TestSearch(t *testing.T) {
	vm := newTestVM(t)
	var results []map[string]any
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").search("unique")`), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for 'unique'")
	}

	var empty []any
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").search("zzzznotfound")`), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty results, got %v", empty)
	}
}

func TestTreeAndTags(t *testing.T) {
	vm := newTestVM(t)
	var tree map[string]any
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").tree()`), &tree); err != nil {
		t.Fatal(err)
	}
	if tree["isFolder"] != true {
		t.Fatalf("tree root should be a folder: %v", tree)
	}

	var tags []struct {
		Tag   string `json:"tag"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(runJSON(t, vm, `require("vault.data").tags()`), &tags); err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Tag != "home" || tags[0].Count != 1 {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}
