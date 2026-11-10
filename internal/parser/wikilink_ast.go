package parser

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	goldparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindWikiLink = ast.NewNodeKind("VaultWikiLink")

type wikiLinkNode struct {
	ast.BaseInline
	raw  []byte
	link WikiLink
}

func (*wikiLinkNode) Kind() ast.NodeKind              { return kindWikiLink }
func (n *wikiLinkNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type wikiLinkExtension struct{ math []MathSpan }

func (e wikiLinkExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(goldparser.WithInlineParsers(util.Prioritized(wikiInlineParser{}, 50)))
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(wikiNodeRenderer(e), 100)))
}

type wikiInlineParser struct{}

func (wikiInlineParser) Trigger() []byte { return []byte{'!', '['} }
func (wikiInlineParser) Parse(_ ast.Node, r text.Reader, _ goldparser.Context) ast.Node {
	line, _ := r.PeekLine()
	loc := wikiLinkRegex.FindIndex(line)
	if loc == nil || loc[0] != 0 {
		return nil
	}
	raw := bytes.Clone(line[:loc[1]])
	inner := raw
	embed := raw[0] == '!'
	if embed {
		inner = inner[1:]
	}
	target, alias, heading := parseWikiLinkInner(string(inner[2 : len(inner)-2]))
	r.Advance(loc[1])
	return &wikiLinkNode{raw: raw, link: WikiLink{Target: target, Alias: alias, Heading: heading, IsEmbed: embed}}
}

type wikiNodeRenderer struct{ math []MathSpan }

func (r wikiNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindWikiLink, r.render)
}
func (r wikiNodeRenderer) render(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, err := w.Write(wikiLinkHTML(node.(*wikiLinkNode), r.math))
		return ast.WalkSkipChildren, err
	}
	return ast.WalkContinue, nil
}

// wikiReferences derives backlinks from exactly the nodes that render as links.
// YAML frontmatter remains a separate source of references, as it is metadata
// rather than rendered Markdown.
func wikiReferences(doc ast.Node, frontmatter []byte, spans []MathSpan) []WikiLink {
	var links []WikiLink
	seen := make(map[string]bool)
	add := func(link WikiLink) {
		if link.Target == "" || (link.IsEmbed && isImageTarget(link.Target)) {
			return
		}
		key := link.Target + "|" + link.Alias
		if !seen[key] {
			seen[key] = true
			links = append(links, link)
		}
	}
	for _, raw := range wikiLinkRegex.FindAll(frontmatter, -1) {
		embed := raw[0] == '!'
		if embed {
			raw = raw[1:]
		}
		target, alias, heading := parseWikiLinkInner(string(raw[2 : len(raw)-2]))
		add(WikiLink{Target: target, Alias: alias, Heading: heading, IsEmbed: embed})
	}
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		n, ok := node.(*wikiLinkNode)
		if !ok {
			return ast.WalkContinue, nil
		}
		link := n.link
		link.Target = RestoreMathText(link.Target, spans)
		link.Alias = RestoreMathText(link.Alias, spans)
		link.Heading = RestoreMathText(link.Heading, spans)
		add(link)
		return ast.WalkContinue, nil
	})
	return links
}
