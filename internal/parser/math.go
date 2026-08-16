// Math scanning and placeholder emission for LaTeX regions in Markdown source.
//
// Why this exists: goldmark destroys TeX. `a_1 + b_2` pairs the underscores
// into <em>, `\{` loses its backslash to Markdown's escape rules, `&` (the
// alignment character in align/matrix environments) becomes &amp;, and the
// html.WithHardWraps() option the renderer runs with interleaves <br/> between
// the lines of a multi-line environment. By the time a client-side typesetter
// sees the HTML there is nothing left to typeset.
//
// So math is lifted out of the source *before* goldmark runs and replaced with
// an inert HTML placeholder carrying the TeX verbatim — the same pre-pass
// idiom replaceWikiLinks uses for [[wiki links]]. The browser reads the TeX
// back out of the placeholder's text content and hands it to MathJax.
//
// Design: ScanMath is a single left-to-right state machine rather than a
// regexp. Three of its rules cannot be expressed in RE2 at all: counting
// preceding backslashes (so `\$100` stays a literal dollar sign), skipping
// fenced code blocks (block structure), and the lookaround that keeps prose
// about prices ("costs $30 and $25") from being read as math.
package parser

import (
	"bytes"
	stdhtml "html"
	"regexp"
	"strconv"
	"strings"
)

// MathSpan describes one math region found in Markdown source.
type MathSpan struct {
	Start   int    // byte offset of the opening delimiter
	End     int    // byte offset just past the closing delimiter
	TeX     string // content between the delimiters, verbatim
	Display bool   // true for $$…$$ and \[…\]; false for $…$ and \(…\)
}

// ScanMath finds every math region in body.
//
// body must not include YAML frontmatter — a frontmatter value such as
// `formula: "$x^2$"` is YAML, not prose, and rewriting it produces invalid
// YAML. Callers go through replaceMathInBody, which splits it off first.
//
// Recognised delimiters: $$…$$ and \[…\] (display), $…$ and \(…\) (inline).
// Regions inside code spans, fenced code blocks, and HTML comments are
// skipped.
//
// Known limitation: indented (4-space) code blocks are NOT skipped. Detecting
// them correctly requires tracking list context, because a 4-space indent
// inside a list is a continuation line, not code — and treating such a line as
// code would silently drop math from nested list items, which is both more
// common in a notes vault and a worse failure than typesetting a stray $ in an
// indented code block. Use fences for code containing dollar signs.
func ScanMath(body []byte) []MathSpan {
	var spans []MathSpan
	i := 0
	n := len(body)

	for i < n {
		atLineStart := i == 0 || body[i-1] == '\n'

		if atLineStart {
			if fenceChar, fenceLen, ok := fenceOpensAt(body, i); ok {
				i = skipFencedBlock(body, i, fenceChar, fenceLen)
				continue
			}
		}

		switch c := body[i]; {
		case c == '<' && bytes.HasPrefix(body[i:], []byte("<!--")):
			if end := bytes.Index(body[i+4:], []byte("-->")); end >= 0 {
				i += 4 + end + 3
			} else {
				i = n
			}

		case c == '\\':
			// \[ and \( open math. Every other backslash escapes the byte that
			// follows it, and consuming both bytes here is exactly what makes
			// `\$` a literal dollar sign: the $ branch below never sees it.
			if i+1 < n && body[i+1] == '[' {
				if end, tex, ok := scanUntil(body, i+2, "\\]"); ok {
					spans = append(spans, MathSpan{Start: i, End: end, TeX: tex, Display: true})
					i = end
					continue
				}
			}
			if i+1 < n && body[i+1] == '(' {
				if end, tex, ok := scanUntil(body, i+2, "\\)"); ok {
					spans = append(spans, MathSpan{Start: i, End: end, TeX: tex, Display: false})
					i = end
					continue
				}
			}
			i += 2

		case c == '`':
			i = skipCodeSpan(body, i)

		case c == '$':
			if bytes.HasPrefix(body[i:], []byte("$$")) {
				if end, tex, ok := scanUntil(body, i+2, "$$"); ok {
					spans = append(spans, MathSpan{Start: i, End: end, TeX: tex, Display: true})
					i = end
					continue
				}
				i += 2
				continue
			}
			if validInlineOpener(body, i) {
				if end, tex, ok := scanInlineClose(body, i+1); ok {
					spans = append(spans, MathSpan{Start: i, End: end, TeX: tex, Display: false})
					i = end
					continue
				}
			}
			i++

		default:
			i++
		}
	}

	return spans
}

// Sentinels standing in for math while goldmark runs. They are Unicode
// Private Use Area code points wrapping a decimal index, chosen because
// goldmark's text renderer passes arbitrary non-ASCII through untouched, they
// carry no Markdown meaning, and no real note contains them.
//
// Emitting the final <span>/<div> in the pre-pass instead does NOT work, and
// the reason is subtle enough to be worth recording: goldmark treats an inline
// <span> as raw inline HTML but still parses the text *between* the tags as
// Markdown, so `$\{1,2\}$` came back as `{1,2}` — the backslashes eaten by
// Markdown's escape rules — and `$f *g* h$` would come back with an <em>.
// Only a token with no Markdown-active characters survives intact.
const (
	mathSentinelOpen  = "\uE000"
	mathSentinelClose = "\uE001"
)

var (
	mathSentinelRe = regexp.MustCompile(mathSentinelOpen + `([0-9]+)` + mathSentinelClose)
	// A display sentinel alone in its paragraph: the <p> goldmark wrapped it in
	// has to go, because a <div> may not live inside a <p>.
	mathParagraphRe = regexp.MustCompile(`(?s)<p>\s*` + mathSentinelOpen + `([0-9]+)` + mathSentinelClose + `\s*</p>`)
)

// ReplaceMath rewrites body, substituting each math region with an opaque
// sentinel, and returns the spans in the order the sentinels index them.
// Call RestoreMath on the rendered HTML afterwards.
//
// Returns body unchanged when there is no math. Most notes have none and this
// runs for every note on every vault reload, so the fast path matters.
func ReplaceMath(body []byte) ([]byte, []MathSpan) {
	spans := ScanMath(body)
	if len(spans) == 0 {
		return body, nil
	}
	i := 0
	out := replaceMathFunc(body, func(s MathSpan) string {
		token := mathSentinelOpen + strconv.Itoa(i) + mathSentinelClose
		i++
		if s.Display {
			// Blank lines put the sentinel in a paragraph of its own so
			// mathParagraphRe can unwrap it into a block-level <div>.
			return "\n\n" + token + "\n\n"
		}
		return token
	})
	return out, spans
}

// RestoreMath swaps the sentinels in rendered HTML for the real math
// placeholders.
//
// The TeX is carried as the placeholder's text content rather than in a data-*
// attribute so that it survives newlines unmangled, needs only &/</> escaping,
// and stays legible when JavaScript is unavailable or MathJax fails to load —
// the same graceful degradation enhanceMermaid gives diagrams.
//
// This runs after every other HTML post-pass so the emitted markup is the last
// thing written and nothing downstream rewrites it.
func RestoreMath(html string, spans []MathSpan) string {
	if len(spans) == 0 {
		return html
	}
	lookup := func(match string, sub []string) (MathSpan, bool) {
		if len(sub) < 2 {
			return MathSpan{}, false
		}
		idx, err := strconv.Atoi(sub[1])
		if err != nil || idx < 0 || idx >= len(spans) {
			return MathSpan{}, false
		}
		return spans[idx], true
	}

	html = mathParagraphRe.ReplaceAllStringFunc(html, func(match string) string {
		s, ok := lookup(match, mathParagraphRe.FindStringSubmatch(match))
		if !ok || !s.Display {
			return match // inline math alone in a paragraph keeps its <p>
		}
		return mathElement(s)
	})

	return mathSentinelRe.ReplaceAllStringFunc(html, func(match string) string {
		s, ok := lookup(match, mathSentinelRe.FindStringSubmatch(match))
		if !ok {
			return ""
		}
		return mathElement(s)
	})
}

// RestoreMathText replaces math sentinels with the bare TeX they stand for, no
// markup.
//
// It exists for values that end up somewhere markup cannot go: an HTML
// attribute, or a JSON field. RestoreMath rewrites every sentinel it finds
// anywhere in the document, and a `<span class="math math-inline">…</span>`
// landing inside an attribute value terminates that attribute at its first
// quote, producing garbage. Substituting the TeX keeps the value well-formed —
// and comparable with the text content of a rendered math element, which is the
// same TeX once tags are stripped, so a heading carrying math still matches the
// link that names it.
func RestoreMathText(s string, spans []MathSpan) string {
	if len(spans) == 0 || !strings.Contains(s, mathSentinelOpen) {
		return s
	}
	return mathSentinelRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := mathSentinelRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx, err := strconv.Atoi(sub[1])
		if err != nil || idx < 0 || idx >= len(spans) {
			return ""
		}
		return spans[idx].TeX
	})
}

func mathElement(s MathSpan) string {
	if s.Display {
		return `<div class="math math-display">` + stdhtml.EscapeString(s.TeX) + `</div>`
	}
	return `<span class="math math-inline">` + stdhtml.EscapeString(s.TeX) + `</span>`
}

// StripMathDelimiters removes math delimiters while keeping the TeX body, for
// the plain-text search index. A note about the normal distribution should be
// findable by searching "sigma"; the command names are the only searchable
// tokens a formula has. Translating TeX to prose ("x squared") is explicitly
// out of scope.
func StripMathDelimiters(body []byte) []byte {
	return replaceMathFunc(body, func(s MathSpan) string { return " " + s.TeX + " " })
}

// replaceMathFunc rewrites every math span through fn, leaving everything else
// byte-identical.
func replaceMathFunc(body []byte, fn func(MathSpan) string) []byte {
	spans := ScanMath(body)
	if len(spans) == 0 {
		return body
	}
	var out bytes.Buffer
	out.Grow(len(body) + 40*len(spans))
	prev := 0
	for _, s := range spans {
		out.Write(body[prev:s.Start])
		out.WriteString(fn(s))
		prev = s.End
	}
	out.Write(body[prev:])
	return out.Bytes()
}

// replaceMathInBody applies ReplaceMath to the Markdown body only, leaving any
// YAML frontmatter block untouched. Mirrors replaceWikiLinks.
// replaceMathInBody applies ReplaceMath to the Markdown body only, leaving any
// YAML frontmatter block untouched. The boundary is splitSource, shared with
// the wiki pre-passes and StripFrontmatter, so any delimiter goldmark-meta
// accepts as metadata is also the boundary math is protected behind — a
// four-dash preamble is not scanned for `$...$`.
func replaceMathInBody(src []byte) ([]byte, []MathSpan) {
	parts := splitSource(src)
	replaced, spans := ReplaceMath(parts.body)
	if !parts.hasFrontmatter() {
		return replaced, spans
	}
	out := make([]byte, 0, len(parts.frontmatter)+len(replaced))
	out = append(out, parts.frontmatter...)
	out = append(out, replaced...)
	return out, spans
}

// validInlineOpener reports whether the `$` at i can open inline math.
//
// The rules come from Pandoc, which has the most battle-tested version of
// them: an opener is not followed by whitespace (so "costs $ 30" is prose) and
// not followed by another `$` (that is a display opener, handled first).
func validInlineOpener(body []byte, i int) bool {
	if i+1 >= len(body) {
		return false
	}
	next := body[i+1]
	return !isMathSpace(next) && next != '$'
}

// scanInlineClose looks for the `$` that closes inline math opened just before
// `from`, returning the offset past it and the TeX between.
//
// A closer is not preceded by whitespace and not followed by an ASCII digit.
// That second rule is what makes "The book costs $30 and $25 used." render as
// prose: the `$` before 25 is preceded by a space, so it is not a closer, and
// with no other `$` on the line the candidate opener before 30 is abandoned.
//
// The scan also gives up at a blank line, which bounds the damage of an
// unmatched `$` to one paragraph instead of swallowing the rest of the note.
func scanInlineClose(body []byte, from int) (int, string, bool) {
	n := len(body)
	for j := from; j < n; j++ {
		switch body[j] {
		case '\\':
			j++ // the loop's j++ skips the escaped byte too
		case '`':
			// Code spans are skipped here as well as in the outer scan.
			// Without this, "costs $30 and $25; a closing `$` may not…" finds
			// its closer inside the code span — that backticked $ satisfies
			// both the preceding-character and following-character rules — and
			// swallows a sentence and a half of prose into a formula.
			j = skipCodeSpan(body, j) - 1
		case '\n':
			if blankLineAt(body, j) {
				return 0, "", false
			}
		case '$':
			prevOK := j > from && !isMathSpace(body[j-1])
			nextOK := j+1 >= n || !isASCIIDigit(body[j+1])
			if prevOK && nextOK {
				return j + 1, string(body[from:j]), true
			}
		}
	}
	return 0, "", false
}

// scanUntil finds closer starting at from, honouring backslash escapes so that
// `\$` cannot close $$…$$. The closer is tested before the escape rule, which
// is what lets `\]` and `\)` — closers that are themselves backslash
// sequences — be found at all.
func scanUntil(body []byte, from int, closer string) (int, string, bool) {
	end := []byte(closer)
	n := len(body)
	for j := from; j < n; j++ {
		if bytes.HasPrefix(body[j:], end) {
			return j + len(end), string(body[from:j]), true
		}
		switch body[j] {
		case '\\':
			j++
		case '`':
			j = skipCodeSpan(body, j) - 1
		}
	}
	return 0, "", false
}

// blankLineAt reports whether the newline at i is followed by a line
// containing nothing but spaces and tabs (or by end of input).
func blankLineAt(body []byte, i int) bool {
	j := i + 1
	for j < len(body) && (body[j] == ' ' || body[j] == '\t' || body[j] == '\r') {
		j++
	}
	return j >= len(body) || body[j] == '\n'
}

// skipCodeSpan returns the offset past the code span opened by the backtick run
// at i. Per CommonMark the closing run must be exactly as long as the opening
// one. An unterminated run is treated as literal text: only the run itself is
// consumed, so `$x$` after a stray backtick is still scanned for math.
func skipCodeSpan(body []byte, i int) int {
	n := len(body)
	runLen := 0
	for i+runLen < n && body[i+runLen] == '`' {
		runLen++
	}
	j := i + runLen
	for j < n {
		if body[j] != '`' {
			if body[j] == '\n' && blankLineAt(body, j) {
				return i + runLen // code spans do not cross blank lines
			}
			j++
			continue
		}
		closeLen := 0
		for j+closeLen < n && body[j+closeLen] == '`' {
			closeLen++
		}
		if closeLen == runLen {
			return j + closeLen
		}
		j += closeLen
	}
	return i + runLen
}

// fenceOpensAt reports whether a fenced code block opens at the line starting
// at i, returning the fence character and run length. Up to three leading
// spaces of indent are allowed, per CommonMark.
func fenceOpensAt(body []byte, i int) (byte, int, bool) {
	n := len(body)
	j := i
	for indent := 0; j < n && body[j] == ' ' && indent < 3; indent++ {
		j++
	}
	if j >= n || (body[j] != '`' && body[j] != '~') {
		return 0, 0, false
	}
	ch := body[j]
	runLen := 0
	for j+runLen < n && body[j+runLen] == ch {
		runLen++
	}
	if runLen < 3 {
		return 0, 0, false
	}
	return ch, runLen, true
}

// skipFencedBlock returns the offset just past the line that closes the fence
// opened at i, or len(body) for an unterminated fence (which, as in
// CommonMark, runs to end of document).
func skipFencedBlock(body []byte, i int, fenceChar byte, fenceLen int) int {
	n := len(body)
	j := lineEnd(body, i)
	for j < n {
		lineStop := lineEnd(body, j)
		if fenceClosesAt(body, j, lineStop, fenceChar, fenceLen) {
			return lineStop
		}
		j = lineStop
	}
	return n
}

// fenceClosesAt reports whether the line [lineStart, lineStop) closes a fence
// opened with fenceLen fenceChars.
//
// A closing fence must be at least as long as the opening one AND carry nothing
// but whitespace after its run. CommonMark allows an info string only on the
// *opening* fence, so a line such as ```` ```example ```` inside a block opened
// with ``` is code, not a terminator. Accepting it would end the skipped region
// early and let the scanner treat `$...$` in the remaining code as math,
// rewriting a code sample into rendered HTML.
func fenceClosesAt(body []byte, lineStart, lineStop int, fenceChar byte, fenceLen int) bool {
	ch, runLen, ok := fenceOpensAt(body, lineStart)
	if !ok || ch != fenceChar || runLen < fenceLen {
		return false
	}
	rest := body[lineStart:lineStop]
	if idx := bytes.LastIndexByte(rest, fenceChar); idx >= 0 {
		rest = rest[idx+1:]
	}
	return len(bytes.TrimSpace(rest)) == 0
}

// lineEnd returns the offset just past the newline terminating the line that
// starts at i, or len(body) for the final unterminated line.
func lineEnd(body []byte, i int) int {
	if idx := bytes.IndexByte(body[i:], '\n'); idx >= 0 {
		return i + idx + 1
	}
	return len(body)
}

func isMathSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }
