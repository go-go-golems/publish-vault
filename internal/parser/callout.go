package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	goldparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A callout header is recognized before inline parsing, when physical source
// lines are still available. Inline parsing subsequently handles its title.
var calloutHeader = regexp.MustCompile(`^\[!([A-Za-z0-9_]+)\]([+-])?[ \t]*`)

var (
	kindCallout      = ast.NewNodeKind("VaultCallout")
	kindCalloutTitle = ast.NewNodeKind("VaultCalloutTitle")
)

type callout struct {
	ast.BaseBlock
	typ, fold string
}

func (n *callout) Kind() ast.NodeKind            { return kindCallout }
func (n *callout) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type calloutTitle struct {
	ast.BaseBlock
	typ, fold string
}

func (n *calloutTitle) Kind() ast.NodeKind            { return kindCalloutTitle }
func (n *calloutTitle) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type calloutParagraphTransformer struct{}

func (calloutParagraphTransformer) Transform(p *ast.Paragraph, reader text.Reader, _ goldparser.Context) {
	quote, ok := p.Parent().(*ast.Blockquote)
	if !ok || quote.FirstChild() != p || p.Lines().Len() == 0 {
		return
	}
	line := p.Lines().At(0)
	// Segments can carry virtual padding. A marker cannot begin in padding.
	if line.Padding != 0 {
		return
	}
	matches := calloutHeader.FindSubmatchIndex(line.Value(reader.Source()))
	if matches == nil {
		return
	}
	source := reader.Source()
	typ := strings.ToLower(string(source[line.Start+matches[2] : line.Start+matches[3]]))
	fold := ""
	if matches[4] >= 0 {
		fold = string(source[line.Start+matches[4] : line.Start+matches[5]])
	}
	title := &calloutTitle{typ: typ, fold: fold}
	segment := line.WithStart(line.Start + matches[1])
	for segment.Stop > segment.Start && (source[segment.Stop-1] == '\n' || source[segment.Stop-1] == '\r') {
		segment.Stop--
	}
	if segment.Start < segment.Stop {
		title.Lines().Append(segment)
	}
	quote.InsertBefore(quote, p, title)
	if p.Lines().Len() == 1 {
		quote.RemoveChild(quote, p)
		return
	}
	p.Lines().SetSliced(1, p.Lines().Len())
}

type calloutASTTransformer struct{}

func (calloutASTTransformer) Transform(doc *ast.Document, _ text.Reader, _ goldparser.Context) {
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			return ast.WalkContinue, nil
		}
		q, ok := node.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}
		title, ok := q.FirstChild().(*calloutTitle)
		if !ok {
			return ast.WalkContinue, nil
		}
		c := &callout{typ: title.typ, fold: title.fold}
		for q.FirstChild() != nil {
			child := q.FirstChild()
			q.RemoveChild(q, child)
			c.AppendChild(c, child)
		}
		parent := q.Parent()
		parent.ReplaceChild(parent, q, c)
		return ast.WalkContinue, nil
	})
}

type calloutRenderer struct{}

func (calloutRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindCallout, renderCalloutNode)
	reg.Register(kindCalloutTitle, renderCalloutTitle)
}
func renderCalloutNode(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	c := node.(*callout)
	if entering {
		class := ""
		if c.fold == "-" {
			class = " callout-collapsible"
		}
		_, _ = fmt.Fprintf(w, `<div class="callout callout-%s%s">`, c.typ, class)
	} else {
		_, _ = w.WriteString(`</div></div>`)
	}
	return ast.WalkContinue, nil
}
func renderCalloutTitle(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	t := node.(*calloutTitle)
	if entering {
		_, _ = fmt.Fprintf(w, `<div class="callout-title"><span class="callout-icon">%s</span> `, calloutIcon(t.typ))
		if t.fold == "-" {
			_, _ = w.WriteString(`<span class="callout-toggle">▶</span> `)
		}
		if !t.HasChildren() {
			_, _ = w.WriteString(titleASCII(t.typ))
		}
	} else {
		_, _ = w.WriteString(`</div><div class="callout-body"`)
		if t.fold == "-" {
			_, _ = w.WriteString(` style="display:none"`)
		}
		_, _ = w.WriteString(`>`)
	}
	return ast.WalkContinue, nil
}
