package vaultwidgets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"retro-obsidian-publish/internal/api"
	"retro-obsidian-publish/internal/search"
	"retro-obsidian-publish/internal/vault"
	"retro-obsidian-publish/internal/vaultdata"
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
	write("Index.md", "---\ntitle: Index\ntags: [home, start]\n---\n# Index\n\nSee [[Deep Note]].\n")
	if err := os.MkdirAll(filepath.Join(root, "Sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	write("Sub/Deep Note.md", "# Deep Note\n\nBody linking back to [[Index]].\n")

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
	provider := staticProvider{v: v, si: si}
	cfg := api.PublicConfig{VaultName: "TestVault", PageTitle: "Test Vault"}
	vaultdata.Register(reg, provider, cfg)
	Register(reg, provider, cfg)
	reg.Enable(vm)
	return vm
}

func runJSON(t *testing.T, vm *goja.Runtime, script string) map[string]any {
	t.Helper()
	value, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("RunString(%q) error = %v", script, err)
	}
	data, err := json.Marshal(value.Export())
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return out
}

func props(t *testing.T, n map[string]any) map[string]any {
	t.Helper()
	p, ok := n["props"].(map[string]any)
	if !ok {
		t.Fatalf("node has no props: %v", n)
	}
	return p
}

func assertNode(t *testing.T, n map[string]any, wantType string) {
	t.Helper()
	if n["kind"] != "component" || n["type"] != wantType {
		t.Fatalf("expected %s component node, got kind=%v type=%v", wantType, n["kind"], n["type"])
	}
}

func TestNoteHtmlDefaultsAndOverrides(t *testing.T) {
	vm := newTestVM(t)
	n := runJSON(t, vm, `
		const vault = require("vault.data");
		const vw = require("vault.widgets");
		vw.noteHtml(vault.note("index"));
	`)
	assertNode(t, n, "NoteHtml")
	p := props(t, n)
	if p["slug"] != "index" || p["html"] == "" {
		t.Fatalf("unexpected props: %v", p)
	}
	for _, flag := range []string{"embeds", "anchors", "highlight", "mermaid"} {
		if p[flag] != true {
			t.Fatalf("flag %s should default true: %v", flag, p)
		}
	}

	n = runJSON(t, vm, `
		const vault2 = require("vault.data");
		const vw2 = require("vault.widgets");
		vw2.noteHtml(vault2.note("index"), { embeds: false, mermaid: false });
	`)
	p = props(t, n)
	if p["embeds"] != false || p["mermaid"] != false || p["anchors"] != true {
		t.Fatalf("overrides not applied: %v", p)
	}
}

func TestBreadcrumbSegments(t *testing.T) {
	vm := newTestVM(t)
	n := runJSON(t, vm, `
		const vault = require("vault.data");
		const vw = require("vault.widgets");
		vw.breadcrumb(vault.note("sub/deep-note"));
	`)
	assertNode(t, n, "BreadcrumbBar")
	segments := props(t, n)["segments"].([]any)
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %v", segments)
	}
	first := segments[0].(map[string]any)
	if first["label"] != "Sub" {
		t.Fatalf("unexpected first segment: %v", first)
	}
}

func TestBacklinksResolvedServerSide(t *testing.T) {
	vm := newTestVM(t)
	n := runJSON(t, vm, `
		const vault = require("vault.data");
		const vw = require("vault.widgets");
		vw.backlinks(vault.note("index"), { onSelect: { kind: "navigate", to: "/note/${slug}" } });
	`)
	assertNode(t, n, "BacklinksPanel")
	p := props(t, n)
	entries := p["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 backlink entry, got %v", entries)
	}
	entry := entries[0].(map[string]any)
	if entry["slug"] != "sub/deep-note" || entry["title"] != "Deep Note" {
		t.Fatalf("backlink not resolved: %v", entry)
	}
	action := p["onSelect"].(map[string]any)
	if action["kind"] != "navigate" {
		t.Fatalf("action not passed through: %v", action)
	}
}

func TestFrontmatterTagListNoteCard(t *testing.T) {
	vm := newTestVM(t)

	n := runJSON(t, vm, `
		const vault = require("vault.data");
		const vw = require("vault.widgets");
		vw.frontmatter(vault.note("index"));
	`)
	assertNode(t, n, "FrontmatterPanel")
	if tags := props(t, n)["tags"].([]any); len(tags) != 2 {
		t.Fatalf("expected 2 tags: %v", tags)
	}

	n = runJSON(t, vm, `
		const vw3 = require("vault.widgets");
		vw3.tagList(["home", "start"], { onSelect: { kind: "navigate", to: "/search?q=%23${tag}" } });
	`)
	assertNode(t, n, "TagCloud")
	if tags := props(t, n)["tags"].([]any); len(tags) != 2 {
		t.Fatalf("tagList lost tags: %v", tags)
	}

	n = runJSON(t, vm, `
		const vault4 = require("vault.data");
		const vw4 = require("vault.widgets");
		vw4.noteCard(vault4.notes()[0]);
	`)
	assertNode(t, n, "NoteCard")
	if props(t, n)["title"] == "" {
		t.Fatalf("noteCard missing title")
	}
}

// The action value produced by the real widget.dsl act namespace must pass
// through helpers unchanged (it is consumed later by the frontend dispatcher).
func TestRealActNavigatePassesThrough(t *testing.T) {
	vm := newTestVM(t)
	// vault.widgets is registered alongside widget.dsl in the host; here we
	// simulate the exported action map shape act.navigate produces.
	n := runJSON(t, vm, `
		const vault = require("vault.data");
		const vw = require("vault.widgets");
		vw.backlinks(vault.note("index"), { onSelect: { kind: "navigate", event: "navigate", to: "/note/${slug}", payload: { slug: { kind: "path", path: "slug" } } } });
	`)
	action := props(t, n)["onSelect"].(map[string]any)
	if action["to"] != "/note/${slug}" || action["payload"] == nil {
		t.Fatalf("nested action structure lost: %v", action)
	}
}
