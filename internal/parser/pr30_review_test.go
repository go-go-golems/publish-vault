package parser

import (
	"fmt"
	"strings"
	"testing"
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
