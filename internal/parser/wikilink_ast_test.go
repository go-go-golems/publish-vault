package parser

import (
	"strings"
	"testing"
)

func TestWikiLinkDisplayIsEscaped(t *testing.T) {
	note, err := Parse([]byte(`[[Target|<img src=x onerror=alert(1)>]]`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(note.HTML, `<img src=x`) || !strings.Contains(note.HTML, `&lt;img`) {
		t.Fatalf("link display is not escaped: %s", note.HTML)
	}
}

func TestWikiASTReferencesMatchRenderedLinks(t *testing.T) {
	src := "---\nrelated: '[[Metadata]]'\n---\n\n[[Rendered]] and `[[Inline Code]]`.\n\n    [[Indented Code]]\n\n```\n[[Fenced Code]]\n```\n\n> [!note] [[Title Link]]\n> [[Body Link]]\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, ref := range note.WikiLinks {
		got[ref.Target] = true
	}
	for _, name := range []string{"Metadata", "Rendered", "Title Link", "Body Link"} {
		if !got[name] {
			t.Errorf("missing %q in %#v", name, note.WikiLinks)
		}
	}
	for _, name := range []string{"Inline Code", "Indented Code", "Fenced Code"} {
		if got[name] {
			t.Errorf("code reference %q indexed", name)
		}
		if strings.Contains(note.HTML, `data-raw="`+name+`"`) {
			t.Errorf("code reference %q rendered as link", name)
		}
	}
}
