package widgethost

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestReaderPageGolden renders a reader page using every vault.widgets
// helper plus the v3 shell spec against the fixed test vault and compares
// with a checked-in golden. Regenerate with:
//
//	UPDATE_GOLDEN=1 GOWORK=off go test ./internal/widgethost -run TestReaderPageGolden
const readerGoldenScript = `
const widget = require("widget.dsl");
const vault = require("vault.data");
const vw = require("vault.widgets");

const note = vault.note("index");

const shell = widget.app.shell((s) =>
	s.navigation((nav) =>
		nav.placement("sidebar").active("reader").section("pages", "Pages", (items) =>
			items.item("reader", "Reader", widget.act.navigate("/w/reader")))));

const page = widget.page(note.title, (p) =>
	p.id("reader")
		.shell(shell)
		.section(note.title, (s) =>
			s.view(vw.breadcrumb(note))
				.view(vw.frontmatter(note))
				.view(vw.noteHtml(note)))
		.section("Linked mentions", (s) =>
			s.view(vw.backlinks(note, { onSelect: widget.act.navigate("/note/${slug}") }))));
`

// redactVolatile replaces values that depend on the test vault's creation
// time (file mtimes) so the golden stays deterministic across runs.
func redactVolatile(node any) {
	switch n := node.(type) {
	case map[string]any:
		if _, ok := n["modTime"]; ok {
			n["modTime"] = "REDACTED"
		}
		for _, child := range n {
			redactVolatile(child)
		}
	case []any:
		for _, child := range n {
			redactVolatile(child)
		}
	}
}

func TestReaderPageGolden(t *testing.T) {
	h := newTestHost(t, map[string]string{"reader.js": readerGoldenScript})
	raw, err := h.RenderPage("reader", url.Values{})
	if err != nil {
		t.Fatalf("RenderPage(reader) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	redactVolatile(got)

	goldenPath := filepath.Join("testdata", "reader-page.golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		pretty, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(goldenPath, append(pretty, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("golden updated: %s", goldenPath)
		return
	}

	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with UPDATE_GOLDEN=1 to create): %v", err)
	}
	var want map[string]any
	if err := json.Unmarshal(goldenData, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Fatalf("reader page IR diverges from golden.\ngot:\n%s", gotJSON)
	}
}
