package parser

import (
	"strings"
	"sync"
	"testing"
)

func TestDocumentRendersAgainstResolverWithoutMutatingTree(t *testing.T) {
	_, doc, err := ParseDocument([]byte("# Local Heading\n\n[[Target#Exact Heading]] [[Target|Alias]] [[Missing]] [[#Local Heading]] ![[pic.png]] ![[Gone]]\n"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := &Resolver{
		Note: func(target string) (string, string, bool) {
			if target == "Target" {
				return "folder/target", "Resolved Title", true
			}
			return "", "", false
		},
		Heading: func(slug, heading string) (string, bool) {
			if slug == "folder/target" && heading == "Exact Heading" {
				return "exact-heading", true
			}
			return "", false
		},
		WikiImage: func(target string) (string, bool) {
			if target == "pic.png" {
				return "/vault-assets/pic.png", true
			}
			return "", false
		},
	}
	got, err := doc.Render(resolver)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/note/folder/target#exact-heading"`, `>Resolved Title</a>`, `>Alias</a>`, `href="#unresolved-missing"`, `href="#local-heading"`, `src="/vault-assets/pic.png"`, `Note not published: gone`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
	again, err := doc.Render(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if got != again {
		t.Fatalf("non-idempotent render: %s / %s", got, again)
	}
	unresolved, err := doc.Render(&Resolver{Note: func(string) (string, string, bool) { return "", "", false }})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unresolved, `href="#unresolved-target"`) {
		t.Fatalf("resolver state leaked: %s", unresolved)
	}
}

func TestDocumentConcurrentRendering(t *testing.T) {
	_, doc, err := ParseDocument([]byte("# Heading\n\n[[Target]] and $x$.\n"))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := doc.Render(&Resolver{Note: func(string) (string, string, bool) { return "target", "Target", true }})
			if err != nil || !strings.Contains(out, `href="/note/target"`) {
				t.Errorf("render: %v %s", err, out)
			}
		}()
	}
	wg.Wait()
}

func TestDocumentOwnsSourceAndResolvesHeadings(t *testing.T) {
	source := []byte("# A *formatted* heading\n\n[[#A formatted heading]]\n")
	_, doc, err := ParseDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	for i := range source {
		source[i] = 'x'
	}
	got, err := doc.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `href="#a-formatted-heading"`) || !strings.Contains(got, `<em>formatted</em>`) {
		t.Fatalf("document lost its owned source or heading: %s", got)
	}
}

func TestDocumentResolvesMarkdownAndRawImages(t *testing.T) {
	_, doc, err := ParseDocument([]byte("![alt](images/a.png)\n\n<img src='images/b.png' alt='b'>\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := doc.Render(&Resolver{Image: func(s string) string { return "/vault-assets/" + s }})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`src="/vault-assets/images/a.png"`, `src="/vault-assets/images/b.png"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
}

func TestRawHTMLTokenizationPreservesUnrelatedBytes(t *testing.T) {
	src := `<section data-value="a&amp;b"><img src="local.png"><span title="x&amp;y">A &amp; B</span></section>`
	_, doc, err := ParseDocument([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got, err := doc.Render(&Resolver{Image: func(string) string { return "/vault-assets/local.png" }})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<section data-value="a&amp;b">`, `src="/vault-assets/local.png"`, `<span title="x&amp;y">A &amp; B</span>`} {
		if !strings.Contains(got, want) {
			t.Errorf("raw token changed or image not resolved, missing %q: %s", want, got)
		}
	}
}

func TestWikiLinksDoNotRewriteUnrelatedRawHTML(t *testing.T) {
	_, doc, err := ParseDocument([]byte(`<div data-target="short">unrelated</div>` + "\n\n[[Short]]"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := doc.Render(&Resolver{Note: func(string) (string, string, bool) { return "resolved/full", "Title", true }})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `<div data-target="short">unrelated</div>`) || !strings.Contains(got, `href="/note/resolved/full"`) {
		t.Fatalf("resolver touched non-link markup or missed link: %s", got)
	}
}

func TestMathBlockInCalloutAndEscapedCurrency(t *testing.T) {
	src := "> [!note] Example\n>\n> $$\n> x_1 + y_2\n> $$\n\nPrice \\$5; code `$x$`; text $y$.\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="callout callout-note"`, `class="math math-display"`, "x_1 + y_2", `$5`, `<code>$x$</code>`, `class="math math-inline"`} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q: %s", want, note.HTML)
		}
	}
}

func TestUnevenTableKeepsAllCells(t *testing.T) {
	src := "| Header |\n| --- |\n| row | [Link](https://example.com) $$x$$ |\n| other | `a\\|b` | extra |\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<table>", `href="https://example.com"`, `class="math math-display"`, "<code>a|b</code>", ">extra</td>"} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q: %s", want, note.HTML)
		}
	}
}

func TestTablePreservesWikiAndMathPipes(t *testing.T) {
	src := "| Name | Formula |\n| --- | --- |\n| [[Target|Alias]] | $a|b$ |\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-alias="Alias"`, `>Alias</a>`, `class="math math-inline">a|b</span>`} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q: %s", want, note.HTML)
		}
	}
	if strings.Count(note.HTML, "<td>") != 2 {
		t.Fatalf("pipe split a cell: %s", note.HTML)
	}
}

func TestTableResolverKeepsCellBoundaries(t *testing.T) {
	_, doc, err := ParseDocument([]byte("| Report | Date |\n| --- | --- |\n| [[Target]] | 2026-05-23 |\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := doc.Render(&Resolver{Note: func(string) (string, string, bool) { return "target", "Resolved Title", true }})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `>Resolved Title</a></td>`) || strings.Contains(out, `2026-05-23</a>`) {
		t.Fatalf("table anchor crossed a cell boundary: %s", out)
	}
}

func TestDisplayMathInsideTablesAndLinksKeepsDisplaySemantics(t *testing.T) {
	src := "| Formula |\n| --- |\n| $$x$$ |\n\n[before $$y$$ after](https://example.com)\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(note.HTML, `class="math math-display"`); got != 2 {
		t.Fatalf("got %d display nodes, want 2: %s", got, note.HTML)
	}
	if strings.Contains(note.HTML, `class="math math-inline"`) {
		t.Fatalf("display math became inline: %s", note.HTML)
	}
}

func TestUnclosedMathInQuoteDoesNotConsumeOutsideContent(t *testing.T) {
	src := "> $$unclosed\n>\n> ## Preserved heading\n\nOutside paragraph.\n\n$$closed$$\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note.HTML, "$$unclosed") || !strings.Contains(note.HTML, "Outside paragraph.") || !strings.Contains(note.HTML, `id="preserved-heading"`) || strings.Count(note.HTML, `class="math math-display"`) != 1 {
		t.Fatalf("unclosed quote math consumed unrelated content: %s", note.HTML)
	}
}

func TestDisplayMathInNestedList(t *testing.T) {
	src := "- Item\n\n  $$\n  x_1 + y_2\n  $$\n\n- Next\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="math math-display"`, "x_1 + y_2", "Next"} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q: %s", want, note.HTML)
		}
	}
}

func TestDisplayMathInListWithinQuote(t *testing.T) {
	src := "> - Item\n>\n>   $$\n>   x_1 + y_2\n>   $$\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note.HTML, `class="math math-display"`) || !strings.Contains(note.HTML, "x_1 + y_2") {
		t.Fatalf("nested display math failed: %s", note.HTML)
	}
}

func TestUnclosedMathInListDoesNotConsumeHeading(t *testing.T) {
	src := "- $$unclosed\n\n  ## Preserved heading\n\nOutside paragraph.\n\n$$closed$$\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note.HTML, "$$unclosed") || !strings.Contains(note.HTML, `id="preserved-heading"`) || !strings.Contains(note.HTML, "Outside paragraph.") {
		t.Fatalf("unclosed list math consumed later blocks: %s", note.HTML)
	}
}

func TestDisplayMathFollowedByText(t *testing.T) {
	for _, src := range []string{"before $$x$$ after", "$$\nx\n$$ after"} {
		note, err := Parse([]byte(src))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(note.HTML, `class="math math-display"`) || !strings.Contains(note.HTML, "after") {
			t.Errorf("lost display math or trailing text for %q: %s", src, note.HTML)
		}
	}
}

func TestStructuralMathContexts(t *testing.T) {
	src := "$$\n\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n$$\n\n> [!note] Math\n> $x_1$ and `[[code]]`\n\n    $code$\n\n[[Target|$x_2$]]\n"
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="math math-display"`, `class="math math-inline"`, `x_1`, `x_2`, `$code$`} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q: %s", want, note.HTML)
		}
	}
	if strings.Contains(note.HTML, `<p><div class="math`) {
		t.Errorf("block math nested in paragraph: %s", note.HTML)
	}
}
