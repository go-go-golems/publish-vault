package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/yuin/goldmark/text"
)

func TestWikiLinkMathWithBrackets(t *testing.T) {
	cases := []struct{ name, src, target, alias string }{
		{"alias", `[[Target|$E[X]$]]`, "Target", "$E[X]$"},
		{"heading", `[[Target#Bound $E[X]$|details]]`, "Target", "details"},
		{"display", `[[Target|$$E[X]$$]]`, "Target", "$$E[X]$$"},
		{"nested wiki delimiters in math", `[[Target|$E[[X]]$]]`, "Target", "$E[[X]]$"},
		{"multiline formula", "[[Target|$E[\nX]$]]", "Target", "$E[\nX]$"},
		{"brackets and alias pipe in table", "| Link | Other |\n| --- | --- |\n| [[Target|$E[X]$]] | ok |\n", "Target", "$E[X]$"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, err := Parse([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if len(n.WikiLinks) != 1 || n.WikiLinks[0].Target != tc.target {
				t.Fatalf("references = %#v; HTML = %s", n.WikiLinks, n.HTML)
			}
			if !strings.Contains(n.HTML, `class="wiki-link"`) {
				t.Fatalf("missing link: %s", n.HTML)
			}
			if tc.name != "heading" && !strings.Contains(n.HTML, `class="math math-inline"`) {
				t.Fatalf("missing formula: %s", n.HTML)
			}
			if tc.name != "heading" && n.WikiLinks[0].Alias != plainMath(tc.alias) {
				t.Fatalf("alias = %q, want %q", n.WikiLinks[0].Alias, plainMath(tc.alias))
			}
			if strings.Contains(n.HTML, `[[Target`) {
				t.Fatalf("literal wiki source leaked: %s", n.HTML)
			}
			if strings.Contains(n.Excerpt, "[[Target") {
				t.Fatalf("literal wiki source leaked into excerpt: %s", n.Excerpt)
			}
			if strings.Contains(tc.src, "| Link |") && strings.Count(n.HTML, "<td>") != 2 {
				t.Fatalf("wiki pipe split table: %s", n.HTML)
			}
		})
	}
}

func TestWikiMathBracketsInFrontmatter(t *testing.T) {
	n, err := Parse([]byte("---\nrelated: '[[Target|$E[X]$]]'\n---\nBody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(n.WikiLinks) != 1 || n.WikiLinks[0].Target != "Target" || n.WikiLinks[0].Alias != "E[X]" {
		t.Fatalf("frontmatter references = %#v", n.WikiLinks)
	}
}

func TestWikiSeparatorsInsideMath(t *testing.T) {
	target, alias, heading := parseWikiLinkInner(`Note $P[A|B]$#Heading $x#y$|label`)
	if target != `Note $P[A|B]$` || heading != `Heading $x#y$` || alias != "label" {
		t.Fatalf("got target=%q heading=%q alias=%q", target, heading, alias)
	}
}

func TestInlineMathMultilineCodeSpanLookahead(t *testing.T) {
	src := "$x `code $\n continued` y$ tail"
	n, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(n.HTML, "x `code $\ncontinued` y") {
		t.Fatalf("cross-line code span changed the closing delimiter: %s", n.HTML)
	}
}

func TestHeadingAutolinksResolve(t *testing.T) {
	for _, heading := range []string{"See https://example.com", "Email me@example.com", "See <https://example.com>"} {
		src := "# " + heading + "\n\n[[#" + strings.ReplaceAll(strings.ReplaceAll(heading, "<", ""), ">", "") + "]]"
		_, doc, err := ParseDocument([]byte(src))
		if err != nil {
			t.Fatal(err)
		}
		html, err := doc.Render(nil)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(html, `class="wiki-link wiki-link-self broken"`) || !strings.Contains(html, `class="wiki-link wiki-link-self"`) {
			t.Fatalf("heading link did not resolve: %s", html)
		}
	}
}

func TestTableSegmentsEmptyCells(t *testing.T) {
	for _, tc := range []struct {
		line  string
		count int
	}{
		{"", 0}, {"| A |", 1}, {"A |", 1}, {"| A ||", 2}, {"A ||", 2}, {"| | A |", 2}, {"| | |", 2}, {"| A | | |", 3},
	} {
		src := []byte(tc.line)
		cells, _, _ := tableSegments(text.NewSegment(0, len(src)), src)
		if len(cells) != tc.count {
			t.Errorf("%q: got %d cells, want %d", tc.line, len(cells), tc.count)
		}
	}
}

func TestTableEmptyCellsProtectWikiLinks(t *testing.T) {
	for _, header := range []string{"| A ||", "A ||", "| | A |", "| | |"} {
		src := header + "\n| --- | --- |\n| [[Target|Alias]] | value |\n"
		n, err := Parse([]byte(src))
		if err != nil {
			t.Fatal(err)
		}
		if len(n.WikiLinks) != 1 || n.WikiLinks[0].Target != "Target" || !strings.Contains(n.HTML, `class="wiki-link"`) || strings.Count(n.HTML, "<td>") != 2 || !strings.Contains(n.HTML, "value") {
			t.Fatalf("header %q: references=%#v HTML=%s", header, n.WikiLinks, n.HTML)
		}
	}
}

func TestMarkdownImageDestinationNormalization(t *testing.T) {
	cases := []struct{ source, destination, escaped string }{
		{`![x](images/a&amp;b.png)`, `images/a&b.png`, `images/a&amp;b.png`},
		{`![x](images/a\(b\).png)`, `images/a(b).png`, `images/a(b).png`},
		{`![x](images/a&#32;b.png)`, `images/a%20b.png`, `images/a%20b.png`},
		{`![x](images/a%20b.png)`, `images/a%20b.png`, `images/a%20b.png`},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			_, doc, err := ParseDocument([]byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			var got string
			html, err := doc.Render(&Resolver{Image: func(s string) string { got = s; return s }})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.destination || !strings.Contains(html, `src="`+tc.escaped+`"`) {
				t.Fatalf("resolver received %q, rendered %s; want %q and %q", got, html, tc.destination, tc.escaped)
			}
		})
	}
}

func BenchmarkWikiLinkDenseParagraph(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("links_%d", n), func(b *testing.B) {
			src := []byte(strings.Repeat("[[Target]] ", n))
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			for i := 0; i < b.N; i++ {
				if _, err := Parse(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkInlineMathDenseParagraph(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("formulas_%d", n), func(b *testing.B) {
			src := []byte(strings.Repeat("$x$ ", n))
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			for i := 0; i < b.N; i++ {
				if _, err := Parse(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
