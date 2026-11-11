package parser

import (
	"bytes"
	"html"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	goldparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	kindInlineMath  = ast.NewNodeKind("VaultInlineMath")
	kindDisplayMath = ast.NewNodeKind("VaultDisplayMath")
)

type inlineMath struct {
	ast.BaseInline
	tex     string
	display bool
}

func (*inlineMath) Kind() ast.NodeKind              { return kindInlineMath }
func (n *inlineMath) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type displayMath struct {
	ast.BaseBlock
	tex    string
	closer string
	closed bool
}

func (*displayMath) Kind() ast.NodeKind              { return kindDisplayMath }
func (n *displayMath) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }
func (*displayMath) IsRaw() bool                     { return true }

type mathExtension struct{}

func (mathExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		goldparser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 150)),
		goldparser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 40)),
		goldparser.WithASTTransformers(util.Prioritized(mathBlockTransformer{}, 90)),
	)
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(mathRenderer{}, 100)))
}

// closedCodeSpans reports whether backtick runs in the scanned prefix have a
// matching run already present. An unmatched run may close on a later line,
// changing which math closer scanMathAt is allowed to see.
func closedCodeSpans(prefix []byte) bool {
	for i := 0; i < len(prefix); {
		if prefix[i] == '\\' && i+1 < len(prefix) {
			i += 2
			continue
		}
		if prefix[i] != '`' {
			i++
			continue
		}
		runEnd := i
		for runEnd < len(prefix) && prefix[runEnd] == '`' {
			runEnd++
		}
		end := skipCodeSpan(prefix, i)
		if end == runEnd {
			return false
		}
		i = end
	}
	return true
}

// mathInlineParser operates only on Goldmark's current inline container. Code
// spans and blocks are handled by Goldmark before this parser sees their text.
type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$', '\\'} }
func (mathInlineParser) Parse(_ ast.Node, r text.Reader, _ goldparser.Context) ast.Node {
	first, _ := r.PeekLine()
	if len(first) == 0 || (first[0] == '\\' && !bytes.HasPrefix(first, []byte(`\(`)) && !bytes.HasPrefix(first, []byte(`\[`))) {
		return nil
	}
	if first[0] == '$' && !bytes.HasPrefix(first, []byte("$$")) && !validInlineOpener(first, 0) {
		return nil
	}
	// The common same-line case must not copy the rest of the paragraph for
	// each formula. An unmatched preceding backtick requires the complete
	// container: its code span may close on a later line.
	if span, ok := scanMathAt(first, 0); ok && closedCodeSpans(first[:span.End]) {
		r.Advance(span.End)
		return &inlineMath{tex: span.TeX, display: span.Display}
	}

	line, pos := r.Position()
	var content bytes.Buffer
	var span MathSpan
	var ok bool
	for {
		part, _ := r.PeekLine()
		if part == nil {
			break
		}
		content.Write(part)
		span, ok = scanMathAt(content.Bytes(), 0)
		if ok && closedCodeSpans(content.Bytes()[:span.End]) {
			break
		}
		r.AdvanceLine()
	}
	r.SetPosition(line, pos)
	// At the end of the container, unmatched backticks are literal, so the
	// scanner's result is final even if closedCodeSpans returned false.
	span, ok = scanMathAt(content.Bytes(), 0)
	if !ok {
		return nil
	}
	r.Advance(span.End)
	return &inlineMath{tex: span.TeX, display: span.Display}
}

// A display block starts at a Markdown block boundary. The closer must exist
// in the remaining source so an unmatched opener remains ordinary Markdown.
type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$', '\\'} }
func (mathBlockParser) Open(parent ast.Node, r text.Reader, pc goldparser.Context) (ast.Node, goldparser.State) {
	line, seg := r.PeekLine()
	off := pc.BlockOffset()
	if off < 0 || off+2 > len(line) {
		return nil, goldparser.NoChildren
	}
	var closer string
	switch string(line[off : off+2]) {
	case "$$":
		closer = "$$"
	case `\[`:
		closer = `\]`
	default:
		return nil, goldparser.NoChildren
	}
	// A same-line formula with trailing prose is handled by the inline parser.
	if i := bytes.Index(line[off+2:], []byte(closer)); i >= 0 && len(bytes.TrimSpace(line[off+2+i+len(closer):])) > 0 {
		return nil, goldparser.NoChildren
	}
	start := seg.Start - seg.Padding + off + 2
	end, _, ok := scanUntil(r.Source(), start, closer)
	if !ok || !mathCloserInContainer(r.Source(), start, end, parent) {
		return nil, goldparser.NoChildren
	}
	n := &displayMath{closer: closer}
	literal := seg.WithStart(seg.Start - seg.Padding + off)
	n.Lines().Append(literal)
	mathBlockLine(n, line[off+2:])
	r.AdvanceToEOL()
	return n, goldparser.NoChildren
}

// A source-wide lookahead must not use a closer in a later container. Check
// quote markers and list indentation in ancestor order on continuation lines.
func mathCloserInContainer(source []byte, start, end int, parent ast.Node) bool {
	var containers []ast.Node
	for n := parent; n != nil; n = n.Parent() {
		switch n.(type) {
		case *ast.Blockquote, *ast.ListItem:
			containers = append(containers, n)
		}
	}
	for i := start; i < end; i++ {
		if source[i] != '\n' || i+1 >= end {
			continue
		}
		j := i + 1
		for k := len(containers) - 1; k >= 0; k-- {
			switch container := containers[k].(type) {
			case *ast.Blockquote:
				spaces := 0
				for j < end && source[j] == ' ' && spaces < 3 {
					j++
					spaces++
				}
				if j >= end || source[j] != '>' {
					return false
				}
				j++
				if j < end && (source[j] == ' ' || source[j] == '\t') {
					j++
				}
			case *ast.ListItem:
				// Goldmark permits blank lines inside a list item without its
				// indentation; nonblank continuation lines need the offset.
				lineEnd := j
				for lineEnd < end && source[lineEnd] != '\n' {
					lineEnd++
				}
				if len(bytes.TrimSpace(source[j:lineEnd])) == 0 {
					continue
				}
				consumed, _ := util.IndentPosition(source[j:lineEnd], j-(i+1), container.Offset)
				if consumed < 0 {
					return false
				}
				j += consumed
			}
		}
	}
	return true
}

func (mathBlockParser) Continue(node ast.Node, r text.Reader, _ goldparser.Context) goldparser.State {
	n := node.(*displayMath)
	if n.closed {
		return goldparser.Close
	}
	line, segment := r.PeekLine()
	n.Lines().Append(segment)
	end := mathBlockLine(n, line)
	if n.closed && len(bytes.TrimSpace(line[end:])) > 0 {
		r.Advance(end)
		return goldparser.Close
	}
	r.AdvanceToEOL()
	return goldparser.Continue | goldparser.NoChildren
}
func mathBlockLine(n *displayMath, line []byte) int {
	if end, tex, ok := scanUntil(line, 0, n.closer); ok {
		n.tex += tex
		n.closed = true
		return end
	}
	n.tex += string(line)
	return len(line)
}
func (mathBlockParser) Close(node ast.Node, _ text.Reader, _ goldparser.Context) {
	n := node.(*displayMath)
	if n.closed || n.Parent() == nil {
		return
	}
	// A closer found by source lookahead may belong to a later container.
	// Restore unmatched source lines as a paragraph instead of fabricating
	// a display node and silently consuming the opening delimiter.
	paragraph := ast.NewParagraph()
	paragraph.Lines().AppendAll(n.Lines().Sliced(0, n.Lines().Len()))
	parent := n.Parent()
	parent.ReplaceChild(parent, n, paragraph)
}
func (mathBlockParser) CanInterruptParagraph() bool { return true }
func (mathBlockParser) CanAcceptIndentedLine() bool { return false }

// Promote display math out of paragraphs, retaining the surrounding inline
// nodes as independent paragraphs so block markup is never nested in <p>.
type mathBlockTransformer struct{}

func (mathBlockTransformer) Transform(doc *ast.Document, _ text.Reader, _ goldparser.Context) {
	var paragraphs []*ast.Paragraph
	_ = ast.Walk(doc, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter {
			if p, ok := n.(*ast.Paragraph); ok {
				for c := p.FirstChild(); c != nil; c = c.NextSibling() {
					if m, ok := c.(*inlineMath); ok && m.display {
						paragraphs = append(paragraphs, p)
						break
					}
				}
			}
		}
		return ast.WalkContinue, nil
	})
	for _, p := range paragraphs {
		parent := p.Parent()
		var fragment *ast.Paragraph
		for p.FirstChild() != nil {
			child := p.FirstChild()
			p.RemoveChild(p, child)
			if m, ok := child.(*inlineMath); ok && m.display {
				fragment = nil
				parent.InsertBefore(parent, p, &displayMath{tex: m.tex, closed: true})
			} else {
				if fragment == nil {
					fragment = ast.NewParagraph()
					parent.InsertBefore(parent, p, fragment)
				}
				fragment.AppendChild(fragment, child)
			}
		}
		parent.RemoveChild(parent, p)
	}
}

// renderMathElement is shared by math nodes and math in wiki-link labels so
// escaping and browser enhancement hooks have one definition.
func renderMathElement(tex string, display, block bool) string {
	class, tag := "math-inline", "span"
	if display {
		class = "math-display"
	}
	if block {
		tag = "div"
	}
	return "<" + tag + ` class="math ` + class + `">` + html.EscapeString(tex) + "</" + tag + ">"
}

type mathRenderer struct{}

func (mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindInlineMath, func(w util.BufWriter, _ []byte, n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter {
			math := n.(*inlineMath)
			_, err := w.WriteString(renderMathElement(math.tex, math.display, false))
			return ast.WalkSkipChildren, err
		}
		return ast.WalkContinue, nil
	})
	reg.Register(kindDisplayMath, func(w util.BufWriter, _ []byte, n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter {
			_, err := w.WriteString(renderMathElement(n.(*displayMath).tex, true, true))
			return ast.WalkSkipChildren, err
		}
		return ast.WalkContinue, nil
	})
}
