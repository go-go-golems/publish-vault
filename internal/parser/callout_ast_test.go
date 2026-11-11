package parser

import (
	"strings"
	"testing"
)

func TestStructuralCalloutComposition(t *testing.T) {
	cases := []struct {
		name, source     string
		contains, absent []string
	}{
		{"formatted title", "> [!note] A **bold** title\n> Text with [[Target]] and $x$.\n", []string{"<strong>bold</strong>", "wiki-link", `class="math`}, nil},
		{"empty title", "> [!warning]\n> Body\n", []string{"Warning</div>", "<p>Body</p>"}, nil},
		{"nested", "> [!note] Outer\n>\n> > [!tip] Inner\n> > Text\n", []string{`callout-note`, `callout-tip`, "<p>Text</p>"}, nil},
		{"sibling callouts", "> [!note] First\n> one\n\n> [!warning] Second\n> two\n", []string{`callout-note`, `callout-warning`, "<p>one</p>", "<p>two</p>"}, []string{"<blockquote>"}},
		{"ordinary quote", "> ordinary\n> quote\n", []string{"<blockquote>", "ordinary\nquote"}, []string{`class="callout`}},
		{"fenced code", "```\n> [!note] Fake\n```\n", []string{"[!note] Fake"}, []string{`class="callout`}},
		{"hard break", "> [!note] Title\n> First  \n> Second\n", []string{"First<br />"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, err := Parse([]byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.contains {
				if !strings.Contains(n.HTML, want) {
					t.Errorf("missing %q in %s", want, n.HTML)
				}
			}
			for _, bad := range tc.absent {
				if strings.Contains(n.HTML, bad) {
					t.Errorf("unexpected %q in %s", bad, n.HTML)
				}
			}
		})
	}
}
