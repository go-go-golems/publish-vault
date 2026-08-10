package parser

import (
	stdhtml "html"
	"regexp"
	"strings"
	"testing"
)

// TestScanMathSpans is the executable specification for the delimiter scanner.
// Each case names a way TeX and Markdown can be confused for one another.
func TestScanMathSpans(t *testing.T) {
	type want struct {
		tex     string
		display bool
	}
	tests := []struct {
		name  string
		input string
		want  []want
	}{
		{"inline", `$x$`, []want{{"x", false}}},
		{"display", `$$x$$`, []want{{"x", true}}},
		{"latex inline", `\(x\)`, []want{{"x", false}}},
		{"latex display", `\[x\]`, []want{{"x", true}}},
		{"two inline spans", `$a$ and $b$`, []want{{"a", false}, {"b", false}}},
		{
			"subscripts survive",
			`$a_1 + b_2 = c_3$`,
			[]want{{`a_1 + b_2 = c_3`, false}},
		},
		{
			"align environment",
			"$$\n\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n$$",
			[]want{{"\n\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n", true}},
		},

		// --- the currency rules ---
		{"currency both sides", `It costs $10 and $5.`, nil},
		{"currency leading", `$5 and $10`, nil},
		{"escaped dollar", `\$100 is a lot`, nil},
		{"dollar then space", `$ x $`, nil},
		{"escaped dollar inside math", `$\text{cost} = \$5$`, []want{{`\text{cost} = \$5`, false}}},

		// --- code must stay literal ---
		{"code span", "`$x$`", nil},
		{"double backtick code span", "``$x$``", nil},
		{"fenced block", "```\n$x$\n```", nil},
		{"tilde fenced block", "~~~\n$x$\n~~~", nil},
		{"fenced block with language", "```go\nfmt.Println(\"$x$\")\n```", nil},
		{"math after fenced block", "```\n$a$\n```\n\n$b$", []want{{"b", false}}},
		{"html comment", "<!-- $x$ -->", nil},

		// --- runaway protection ---
		{"unterminated inline", `$x and nothing else`, nil},
		{"blank line kills candidate", "$x\n\ny$", nil},
		{"unterminated display", `$$x and nothing else`, nil},

		// --- interaction with other syntax ---
		{"wiki link inside math stays literal", `$[[Note]]$`, []want{{`[[Note]]`, false}}},
		{"angle brackets", `$x < y$`, []want{{`x < y`, false}}},
		{"no math at all", `just some prose`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans := ScanMath([]byte(tt.input))
			if len(spans) != len(tt.want) {
				t.Fatalf("ScanMath(%q) returned %d spans, want %d: %+v",
					tt.input, len(spans), len(tt.want), spans)
			}
			for i, w := range tt.want {
				if spans[i].TeX != w.tex {
					t.Errorf("span %d TeX = %q, want %q", i, spans[i].TeX, w.tex)
				}
				if spans[i].Display != w.display {
					t.Errorf("span %d Display = %v, want %v", i, spans[i].Display, w.display)
				}
				if got := tt.input[spans[i].Start:spans[i].End]; !strings.Contains(got, w.tex) {
					t.Errorf("span %d offsets [%d:%d] = %q do not cover the TeX",
						i, spans[i].Start, spans[i].End, got)
				}
			}
		})
	}
}

// TestReplaceMathIdentityWhenNoMath guards the fast path: ReplaceMath runs for
// every note on every vault reload and most notes have no math at all.
func TestReplaceMathIdentityWhenNoMath(t *testing.T) {
	src := []byte("# Title\n\nJust prose with a price of $10 and $5.\n")
	got, spans := ReplaceMath(src)
	if spans != nil {
		t.Fatalf("ReplaceMath found %d spans in math-free input: %+v", len(spans), spans)
	}
	if &got[0] != &src[0] {
		t.Fatalf("ReplaceMath allocated a new slice for math-free input")
	}
}

var mathElemRe = regexp.MustCompile(`<(span|div) class="math math-(inline|display)">([\s\S]*?)</(?:span|div)>`)

// texRoundTrip extracts the TeX carried by each placeholder in rendered HTML,
// un-escaping it the way the browser does when reading textContent.
func texRoundTrip(t *testing.T, html string) []string {
	t.Helper()
	var out []string
	for _, m := range mathElemRe.FindAllStringSubmatch(html, -1) {
		out = append(out, stdhtml.UnescapeString(m[3]))
	}
	return out
}

// TestParseMathRoundTrip is the escaping guard: whatever TeX goes in must come
// back out of the placeholder byte-for-byte after the full Parse pipeline.
// A single accidental double-escape anywhere in that chain shows up here.
func TestParseMathRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  string
		tex  []string
	}{
		{"subscripts", `$a_1 + b_2$`, []string{`a_1 + b_2`}},
		{"ampersand and backslashes", "$$\\begin{align} a &= b \\\\ c &= d \\end{align}$$",
			[]string{`\begin{align} a &= b \\ c &= d \end{align}`}},
		{"angle brackets", `$x < y > z$`, []string{`x < y > z`}},
		{"quotes", `$\text{"quoted"}$`, []string{`\text{"quoted"}`}},
		{"braces", `$\{1, 2\}$`, []string{`\{1, 2\}`}},
		{"multiline display", "$$\nx^2\n+ y^2\n$$", []string{"\nx^2\n+ y^2\n"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			got := texRoundTrip(t, note.HTML)
			if len(got) != len(tt.tex) {
				t.Fatalf("got %d placeholders, want %d\nHTML: %s", len(got), len(tt.tex), note.HTML)
			}
			for i := range got {
				if got[i] != tt.tex[i] {
					t.Errorf("placeholder %d TeX = %q, want %q\nHTML: %s", i, got[i], tt.tex[i], note.HTML)
				}
			}
		})
	}
}

// TestParseMathSurvivesMarkdownMangling covers the four concrete ways goldmark
// destroys unprotected TeX. Each assertion here fails loudly if the math
// pre-pass is removed or reordered after replaceWikiLinks.
func TestParseMathSurvivesMarkdownMangling(t *testing.T) {
	t.Run("underscores do not become emphasis", func(t *testing.T) {
		note := mustParse(t, `The value $a_1 + b_2 = c_3$ holds.`)
		if strings.Contains(note.HTML, "<em>") {
			t.Errorf("underscores inside math became emphasis:\n%s", note.HTML)
		}
	})

	t.Run("hard wraps do not enter display math", func(t *testing.T) {
		note := mustParse(t, "$$\n\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n$$")
		inner := mathElemRe.FindStringSubmatch(note.HTML)
		if inner == nil {
			t.Fatalf("no math placeholder in output:\n%s", note.HTML)
		}
		if strings.Contains(inner[3], "<br") {
			t.Errorf("WithHardWraps() inserted <br/> inside display math:\n%s", inner[3])
		}
	})

	t.Run("display math is not nested inside a paragraph", func(t *testing.T) {
		note := mustParse(t, "Before.\n\n$$x^2$$\n\nAfter.")
		if regexp.MustCompile(`<p>[^<]*<div class="math`).MatchString(note.HTML) {
			t.Errorf("display math div nested inside a <p>:\n%s", note.HTML)
		}
	})

	t.Run("asterisks do not become emphasis", func(t *testing.T) {
		note := mustParse(t, `Convolution $f * g * h$ is associative.`)
		if strings.Contains(note.HTML, "<em>") {
			t.Errorf("asterisks inside math became emphasis:\n%s", note.HTML)
		}
	})
}

// TestParseMathLeavesOtherSyntaxAlone checks that the new pre-pass does not
// disturb the constructs that already share the pipeline.
func TestParseMathLeavesOtherSyntaxAlone(t *testing.T) {
	t.Run("frontmatter with a dollar value stays YAML", func(t *testing.T) {
		note := mustParse(t, "---\ntitle: Prices\nformula: \"$x^2$\"\n---\n\nBody.\n")
		if note.Title != "Prices" {
			t.Errorf("Title = %q, want %q (frontmatter was rewritten)", note.Title, "Prices")
		}
		if got := note.Frontmatter["formula"]; got != "$x^2$" {
			t.Errorf("frontmatter formula = %v, want %q", got, "$x^2$")
		}
		if strings.Contains(note.HTML, "math-inline") {
			t.Errorf("frontmatter was scanned for math:\n%s", note.HTML)
		}
	})

	t.Run("wiki links outside math still resolve", func(t *testing.T) {
		note := mustParse(t, "$x^2$ and [[Some Note]].")
		if !strings.Contains(note.HTML, `class="wiki-link"`) {
			t.Errorf("wiki link was not rendered:\n%s", note.HTML)
		}
		if len(note.WikiLinks) != 1 || note.WikiLinks[0].Target != "Some Note" {
			t.Errorf("WikiLinks = %+v, want one entry for %q", note.WikiLinks, "Some Note")
		}
	})

	t.Run("math inside a callout survives", func(t *testing.T) {
		note := mustParse(t, "> [!note] Identity\n> Euler: $e^{i\\pi} + 1 = 0$\n")
		if !strings.Contains(note.HTML, "callout-note") {
			t.Errorf("callout was not rendered:\n%s", note.HTML)
		}
		got := texRoundTrip(t, note.HTML)
		if len(got) != 1 || got[0] != `e^{i\pi} + 1 = 0` {
			t.Errorf("math inside callout = %q, want one entry `e^{i\\pi} + 1 = 0`\n%s", got, note.HTML)
		}
	})

	t.Run("currency prose renders as text", func(t *testing.T) {
		note := mustParse(t, "The book costs $30 and $25 used.")
		if strings.Contains(note.HTML, "math-inline") {
			t.Errorf("currency was read as math:\n%s", note.HTML)
		}
		if !strings.Contains(note.HTML, "$30") || !strings.Contains(note.HTML, "$25") {
			t.Errorf("dollar amounts were lost:\n%s", note.HTML)
		}
	})
}

// TestPlainTextStripsMathDelimiters covers the search-index path: a reader
// searching for "sigma" should find a note whose only mention of it is inside
// a formula.
func TestPlainTextStripsMathDelimiters(t *testing.T) {
	got := PlainText([]byte("The deviation is $\\sigma^2$ overall."))
	if strings.Contains(got, "$") {
		t.Errorf("PlainText() = %q, want math delimiters removed", got)
	}
	if !strings.Contains(got, "sigma") {
		t.Errorf("PlainText() = %q, want the TeX body preserved for indexing", got)
	}
}

// TestScanMathTerminates is a coarse guard against a scanner branch that fails
// to advance. Every input here is pathological in a different way.
func TestScanMathTerminates(t *testing.T) {
	inputs := []string{
		"$", "$$", "$$$", "\\", "`", "```", "~~~", "<!--",
		"$$$$$$", "\\[", "\\(", "$\\", "``$``", "\\\\$",
		"$a$$b$", "$$a$$$b$$", strings.Repeat("$", 64),
	}
	for _, in := range inputs {
		done := make(chan struct{})
		go func() { defer close(done); ScanMath([]byte(in)) }()
		<-done // the test binary's own timeout catches a hang
	}
}

func mustParse(t *testing.T, src string) *ParsedNote {
	t.Helper()
	note, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	return note
}
