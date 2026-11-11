package parser

import (
	"bytes"
	"html"
	"strings"

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
	raw    []byte
	link   WikiLink
	offset int
}

func (*wikiLinkNode) Kind() ast.NodeKind              { return kindWikiLink }
func (n *wikiLinkNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type wikiLinkExtension struct {
	resolver *Resolver
	headings *HeadingIndex
}

func (e wikiLinkExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(goldparser.WithInlineParsers(util.Prioritized(wikiInlineParser{}, 50)))
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(wikiNodeRenderer(e), 100)))
}

// scanWikiLinkAt recognizes a wiki link while treating complete math spans as
// atomic. Brackets inside TeX are not wiki delimiters; other nested brackets
// remain invalid, as in the original wiki-link grammar.
func scanWikiLinkAt(input []byte, start int) (int, bool) {
	i := start
	if i < len(input) && input[i] == '!' {
		i++
	}
	if !bytes.HasPrefix(input[i:], []byte("[[")) {
		return 0, false
	}
	i += 2
	content := i
	for i < len(input) {
		if input[i] == '$' || input[i] == '\\' {
			if span, ok := scanMathAt(input, i); ok {
				i = span.End
				continue
			}
		}
		if bytes.HasPrefix(input[i:], []byte("]]")) {
			return i + 2, i > content
		}
		if input[i] == '[' || input[i] == ']' {
			return 0, false
		}
		i++
	}
	return 0, false
}

// wikiLinkIndexes supplies the same grammar to non-AST consumers such as
// frontmatter reference collection and table pipe protection.
func wikiLinkIndexes(input []byte) [][2]int {
	var indexes [][2]int
	for i := 0; i < len(input); i++ {
		if input[i] != '[' && input[i] != '!' {
			continue
		}
		if end, ok := scanWikiLinkAt(input, i); ok {
			indexes = append(indexes, [2]int{i, end})
			i = end - 1
		}
	}
	return indexes
}

type wikiInlineParser struct{}

func (wikiInlineParser) Trigger() []byte { return []byte{'!', '['} }
func (wikiInlineParser) Parse(_ ast.Node, r text.Reader, _ goldparser.Context) ast.Node {
	first, _ := r.PeekLine()
	if !bytes.HasPrefix(first, []byte("[[")) && !bytes.HasPrefix(first, []byte("![[")) {
		return nil
	}
	_, position := r.Position()
	// PeekLine is a view of the remaining line. Most links end on that line,
	// so scan it directly instead of copying the suffix for every link.
	if end, ok := scanWikiLinkAt(first, 0); ok {
		return wikiNodeFromRaw(r, first[:end], position.Start)
	}
	lineNumber, position := r.Position()
	var input bytes.Buffer
	for {
		line, _ := r.PeekLine()
		if line == nil {
			break
		}
		input.Write(line)
		if bytes.Contains(input.Bytes(), []byte("]]")) {
			if _, ok := scanWikiLinkAt(input.Bytes(), 0); ok {
				break
			}
		}
		r.AdvanceLine()
	}
	r.SetPosition(lineNumber, position)
	end, ok := scanWikiLinkAt(input.Bytes(), 0)
	if !ok {
		return nil
	}
	return wikiNodeFromRaw(r, input.Bytes()[:end], position.Start)
}

func wikiNodeFromRaw(r text.Reader, source []byte, offset int) ast.Node {
	raw := bytes.Clone(source)
	inner := raw
	embed := raw[0] == '!'
	if embed {
		inner = inner[1:]
	}
	target, alias, heading := parseWikiLinkInner(string(inner[2 : len(inner)-2]))
	r.Advance(len(raw))
	return &wikiLinkNode{raw: raw, offset: offset, link: WikiLink{Target: target, Alias: alias, Heading: heading, IsEmbed: embed}}
}

type wikiNodeRenderer struct {
	resolver *Resolver
	headings *HeadingIndex
}

func (r wikiNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindWikiLink, r.render)
}
func (r wikiNodeRenderer) render(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*wikiLinkNode)
	link := n.link
	target, alias, heading := plainMath(link.Target), plainMath(link.Alias), plainMath(link.Heading)
	label := link.Alias
	if label == "" {
		label = link.Target
	}
	if label == "" {
		label = link.Heading
	}
	esc := html.EscapeString
	if link.IsEmbed && isImageTarget(target) {
		if r.resolver == nil || r.resolver.WikiImage == nil {
			_, err := w.WriteString(`<img class="wiki-embed-image" data-asset="` + esc(target) + `" alt="` + esc(plainMath(label)) + `" loading="lazy">`)
			return ast.WalkSkipChildren, err
		}
		src, ok := r.resolver.WikiImage(target)
		if !ok {
			_, err := w.WriteString(`<span class="wiki-embed wiki-embed-broken">⚠ Image not found: ` + esc(target) + `</span>`)
			return ast.WalkSkipChildren, err
		}
		_, err := w.WriteString(`<img class="wiki-embed-image" src="` + esc(src) + `" alt="` + esc(plainMath(label)) + `" loading="lazy">`)
		return ast.WalkSkipChildren, err
	}
	if target == "" {
		if heading == "" {
			_, err := w.WriteString(esc(string(n.raw)))
			return ast.WalkSkipChildren, err
		}
		id, ok := r.headings.Lookup(heading)
		class := "wiki-link wiki-link-self"
		if !ok {
			id = "unresolved-" + slugify(heading)
			class += " broken"
		}
		_, err := w.WriteString(`<a href="#` + esc(id) + `" class="` + class + `" data-heading="` + esc(heading) + `" data-alias="` + esc(alias) + `">` + renderMathLabel(label) + `</a>`)
		return ast.WalkSkipChildren, err
	}
	slug := slugify(target)
	title := ""
	found := true
	if r.resolver != nil && r.resolver.Note != nil {
		slug, title, found = r.resolver.Note(target)
	}
	if !found {
		slug = slugify(target)
	}
	if link.IsEmbed {
		if r.resolver != nil && !found {
			_, err := w.WriteString(`<span class="wiki-embed wiki-embed-broken">⚠ Note not published: ` + esc(slug) + `</span>`)
			return ast.WalkSkipChildren, err
		}
		_, err := w.WriteString(`<div class="wiki-embed" data-target="` + esc(slug) + `" data-heading="` + esc(heading) + `" data-raw="` + esc(target) + `"></div>`)
		return ast.WalkSkipChildren, err
	}
	href := "/note/" + slug
	if !found {
		href = "#unresolved-" + slug
	} else if heading != "" {
		if r.resolver != nil && r.resolver.Heading != nil {
			if id, ok := r.resolver.Heading(slug, heading); ok {
				href += "#" + id
			}
		} else {
			href += "#" + slugify(heading)
		}
	}
	if alias == "" && title != "" {
		label = title
	}
	_, err := w.WriteString(`<a href="` + esc(href) + `" class="wiki-link" data-target="` + esc(slug) + `" data-raw="` + esc(target) + `" data-heading="` + esc(heading) + `" data-alias="` + esc(alias) + `">` + renderMathLabel(label) + `</a>`)
	return ast.WalkSkipChildren, err
}

// Wiki fields are parsed once. Render and backlink analysis share the nodes;
// YAML frontmatter references remain an independent metadata convention.
func wikiReferences(doc ast.Node, frontmatter []byte) []WikiLink {
	var links []WikiLink
	seen := make(map[string]bool)
	add := func(link WikiLink) {
		link.Target = plainMath(link.Target)
		link.Alias = plainMath(link.Alias)
		link.Heading = plainMath(link.Heading)
		if link.Target == "" || (link.IsEmbed && isImageTarget(link.Target)) {
			return
		}
		key := link.Target + "|" + link.Alias
		if !seen[key] {
			seen[key] = true
			links = append(links, link)
		}
	}
	for _, loc := range wikiLinkIndexes(frontmatter) {
		raw := frontmatter[loc[0]:loc[1]]
		embed := raw[0] == '!'
		if embed {
			raw = raw[1:]
		}
		target, alias, heading := parseWikiLinkInner(string(raw[2 : len(raw)-2]))
		add(WikiLink{Target: target, Alias: alias, Heading: heading, IsEmbed: embed})
	}
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if n, ok := node.(*wikiLinkNode); ok {
				add(n.link)
			}
		}
		return ast.WalkContinue, nil
	})
	return links
}

func plainMath(s string) string {
	spans := ScanMath([]byte(s))
	if len(spans) == 0 {
		return s
	}
	var b strings.Builder
	prev := 0
	for _, span := range spans {
		b.WriteString(s[prev:span.Start])
		b.WriteString(span.TeX)
		prev = span.End
	}
	b.WriteString(s[prev:])
	return b.String()
}
func renderMathLabel(s string) string {
	spans := ScanMath([]byte(s))
	var b strings.Builder
	prev := 0
	for _, span := range spans {
		b.WriteString(html.EscapeString(s[prev:span.Start]))
		b.WriteString(renderMathElement(span.TeX, false, false))
		prev = span.End
	}
	b.WriteString(html.EscapeString(s[prev:]))
	return b.String()
}
