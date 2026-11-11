package parser

import (
	"strings"
	"testing"
)

func TestSoftBreaksAndExplicitBreaks(t *testing.T) {
	cases := []struct {
		name, markdown string
		breaks         int
	}{
		{"paragraph", "first line\nsecond line\n", 0},
		{"list item", "- first line\n  second line\n", 0},
		{"two spaces", "first line  \nsecond line\n", 1},
		{"backslash", "first line\\\nsecond line\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note, err := Parse([]byte(tc.markdown))
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(note.HTML, "<br"); got != tc.breaks {
				t.Fatalf("got %d breaks, want %d: %s", got, tc.breaks, note.HTML)
			}
		})
	}
}

func TestCalloutContainsStructuredBody(t *testing.T) {
	note, err := Parse([]byte("> [!note] A title\n> First line\n> second line\n>\n> - one\n> - two\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="callout callout-note"`, `class="callout-title"`, "A title", `class="callout-body"`, "<p>", "<ul>"} {
		if !strings.Contains(note.HTML, want) {
			t.Fatalf("missing %q in %s", want, note.HTML)
		}
	}
	if strings.Contains(note.HTML, "<br") {
		t.Fatalf("soft break became hard break: %s", note.HTML)
	}
}
