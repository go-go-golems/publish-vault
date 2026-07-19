package widgethost

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/go-go-golems/publish-vault/pkg/api"
	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type staticProvider struct {
	v  *vault.Vault
	si *search.Index
}

func (p staticProvider) Snapshot() (*vault.Vault, *search.Index) { return p.v, p.si }

func newTestHost(t *testing.T, pages map[string]string) *Host {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Index.md"), []byte("---\ntitle: Index\ntags: [home]\n---\n# Index\n\nHello.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Second.md"), []byte("# Second\n\nMore text.\n"), 0o600); err != nil {
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

	pagesDir := t.TempDir()
	for name, source := range pages {
		if err := os.WriteFile(filepath.Join(pagesDir, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return New(staticProvider{v: v, si: si}, api.PublicConfig{VaultName: "TestVault", PageTitle: "Test Vault"}, pagesDir)
}

const vaultTablePage = `
const widget = require("widget.dsl");
const vault = require("vault.data");
const rows = vault.notes().map((n) => ({ slug: n.slug, title: n.title }));
const schema = widget.data
	.fields("notes", (f) => f.key("slug").primary("title"))
	.build();
const table = widget.data
	.collection("notes", rows, (c) =>
		c.schema(schema).table((t) => t.rowSelect(widget.act.navigate("/note/${row.slug}"))),
	)
	.toNode();
const page = widget.page("Notes", (p) =>
	p.id("notes").section("All notes", (s) => s.view(table)));

const actions = {
	ping: (payload) => ({ ok: true, toast: "pong " + (payload && payload.name) }),
	silent: () => {},
};
`

func renderToMap(t *testing.T, h *Host, id string) map[string]any {
	t.Helper()
	raw, err := h.RenderPage(id, nil)
	if err != nil {
		t.Fatalf("RenderPage(%q) error = %v", id, err)
	}
	var page map[string]any
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatalf("unmarshal page: %v", err)
	}
	return page
}

func findComponents(node any, typ string) []map[string]any {
	var out []map[string]any
	switch n := node.(type) {
	case map[string]any:
		if n["kind"] == "component" && n["type"] == typ {
			out = append(out, n)
		}
		for _, child := range n {
			out = append(out, findComponents(child, typ)...)
		}
	case []any:
		for _, child := range n {
			out = append(out, findComponents(child, typ)...)
		}
	}
	return out
}

func TestRenderPageWithVaultData(t *testing.T) {
	h := newTestHost(t, map[string]string{"notes.js": vaultTablePage})

	page := renderToMap(t, h, "notes")
	if page["title"] != "Notes" || page["id"] != "notes" {
		t.Fatalf("unexpected page envelope: title=%v id=%v", page["title"], page["id"])
	}

	tables := findComponents(page["root"], "DataTable")
	if len(tables) != 1 {
		t.Fatalf("expected 1 DataTable, found %d", len(tables))
	}
	props := tables[0]["props"].(map[string]any)
	rows := props["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("expected 2 vault-backed rows, got %d", len(rows))
	}
	rowSelect := props["onRowSelect"].(map[string]any)
	if rowSelect["kind"] != "navigate" || rowSelect["to"] != "/note/${row.slug}" {
		t.Fatalf("unexpected onRowSelect: %v", rowSelect)
	}
}

// TestParityWithRagEvaluationGolden pins grammar identity with
// rag-evaluation-system: the fixture pair is copied verbatim from
// pkg/widgetdsl/testdata/v3/{examples,golden}/01-simple-table.* at v0.1.7.
// If this test fails after a widgetdsl upgrade, re-copy the fixtures and
// review the grammar change.
func TestParityWithRagEvaluationGolden(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "parity-simple-table.js"))
	if err != nil {
		t.Fatal(err)
	}
	h := newTestHost(t, map[string]string{"parity.js": string(source)})

	got := renderToMap(t, h, "parity")

	goldenData, err := os.ReadFile(filepath.Join("testdata", "parity-simple-table.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(goldenData, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("IR diverges from rag-evaluation golden.\ngot:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

func TestListPages(t *testing.T) {
	h := newTestHost(t, map[string]string{"notes.js": vaultTablePage, "zz.js": `const page = widget => widget;`})
	pages, err := h.ListPages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0].ID != "notes" || pages[1].ID != "zz" {
		t.Fatalf("unexpected pages: %+v", pages)
	}
	if pages[0].Title != "Notes" {
		t.Fatalf("expected static title extraction, got %q", pages[0].Title)
	}
}

func TestHandleAction(t *testing.T) {
	h := newTestHost(t, map[string]string{"notes.js": vaultTablePage})

	result, err := h.HandleAction("notes.ping", json.RawMessage(`{"name":"world"}`), nil)
	if err != nil {
		t.Fatalf("HandleAction error = %v", err)
	}
	if result["ok"] != true || result["toast"] != "pong world" {
		t.Fatalf("unexpected result: %v", result)
	}

	// Bare name resolution scans pages.
	result, err = h.HandleAction("ping", json.RawMessage(`{"name":"bare"}`), nil)
	if err != nil {
		t.Fatalf("bare HandleAction error = %v", err)
	}
	if result["toast"] != "pong bare" {
		t.Fatalf("unexpected bare result: %v", result)
	}

	// Undefined handler result defaults to {ok:true}.
	result, err = h.HandleAction("notes.silent", nil, nil)
	if err != nil {
		t.Fatalf("silent HandleAction error = %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected silent result: %v", result)
	}

	if _, err := h.HandleAction("notes.missing", nil, nil); err == nil {
		t.Fatal("expected error for missing action")
	}
}

func TestHTTPHandlers(t *testing.T) {
	h := newTestHost(t, map[string]string{"notes.js": vaultTablePage})
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/widget/pages")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/api/widget/pages/notes")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()
	var page map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page["title"] != "Notes" {
		t.Fatalf("unexpected page title: %v", page["title"])
	}

	resp3, err := http.Get(srv.URL + "/api/widget/pages/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("missing page status = %d", resp3.StatusCode)
	}

	resp4, err := http.Post(srv.URL+"/api/widget/actions/notes.ping", "application/json",
		strings.NewReader(`{"payload":{"name":"http"},"context":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp4.Body.Close() }()
	var result map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true || result["toast"] != "pong http" {
		t.Fatalf("unexpected action result: %v", result)
	}
}
