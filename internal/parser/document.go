package parser

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"

	nethtml "golang.org/x/net/html"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	goldparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Resolver supplies vault-specific decisions without coupling parsing to a vault.
// All callbacks are optional. A nil resolver renders standalone note output.
// Image may return an empty string to leave a destination unchanged.
type Resolver struct {
	Note      func(target string) (slug, title string, found bool)
	Heading   func(slug, heading string) (id string, found bool)
	WikiImage func(target string) (url string, found bool)
	Image     func(destination string) string
}

// Document owns the source bytes used by Goldmark nodes. It is immutable after
// parsing; rendering uses a separate renderer and never mutates its tree.
type Document struct {
	source   []byte
	root     ast.Node
	headings *HeadingIndex
}

func newMarkdown(resolver *Resolver, headings *HeadingIndex) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(meta.Meta, extension.GFM, extension.Footnote,
			expandedTableExtension{}, calloutExtension{}, mathExtension{}, wikiLinkExtension{resolver: resolver, headings: headings}),
		goldmark.WithParserOptions(goldparser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(htmlrenderer.WithXHTML(), htmlrenderer.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(imageRenderer{resolver: resolver}, 100))),
	)
}

// parseSource makes a source-owned document and returns its metadata context.
func parseSource(src []byte, headings *HeadingIndex) (*Document, goldparser.Context) {
	source := bytes.Clone(src)
	md := newMarkdown(nil, nil)
	ctx := goldparser.NewContext()
	root := md.Parser().Parse(text.NewReader(source), goldparser.WithContext(ctx))
	if headings == nil {
		headings = headingIndexFromAST(root, source)
	}
	return &Document{source: source, root: root, headings: headings}, ctx
}

// ParseDocument parses and analyzes a single immutable source document.
func ParseDocument(src []byte) (*ParsedNote, *Document, error) {
	doc, ctx := parseSource(src, nil)
	fm := normalizeFrontmatter(meta.Get(ctx))
	note := &ParsedNote{Frontmatter: fm, WikiLinks: wikiReferences(doc.root, splitSource(doc.source).frontmatter),
		Tags: extractTags(fm), Title: extractTitle(fm, doc.source), Excerpt: extractExcerpt(doc.source)}
	return note, doc, nil
}

// ParseForRendering reparses a previously analyzed source, reusing its heading
// index. The index must come from this exact source and parsing configuration.
func ParseForRendering(src []byte, headings *HeadingIndex) *Document {
	doc, _ := parseSource(src, headings)
	return doc
}

// Render produces HTML for this document using the supplied resolution policy.
func (d *Document) Render(r *Resolver) (string, error) {
	md := newMarkdown(r, d.headings)
	var out bytes.Buffer
	if err := md.Renderer().Render(&out, d.source, d.root); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Headings returns the immutable heading index assigned during parsing.
// The index does not retain the source document or syntax tree.
func (d *Document) Headings() *HeadingIndex { return d.headings }

func headingIndexFromAST(root ast.Node, source []byte) *HeadingIndex {
	index := &HeadingIndex{byText: map[string]string{}, ids: map[string]bool{}}
	_ = ast.Walk(root, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		value, ok := heading.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}
		var id string
		switch x := value.(type) {
		case []byte:
			id = string(x)
		case string:
			id = x
		default:
			id = fmt.Sprint(x)
		}
		index.ids[id] = true
		key := normalizeHeadingKey(nodeText(heading, source))
		if key != "" {
			if _, exists := index.byText[key]; !exists {
				index.byText[key] = id
			}
		}
		return ast.WalkContinue, nil
	})
	return index
}

// nodeText collects rendered textual content without serializing markup.
func nodeText(root ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(root, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		switch x := n.(type) {
		case *ast.Text:
			b.Write(util.UnescapePunctuations(x.Segment.Value(source)))
			if x.SoftLineBreak() || x.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(x.Value)
		case *inlineMath:
			b.WriteString(x.tex)
			return ast.WalkSkipChildren, nil
		case *wikiLinkNode:
			label := x.link.Alias
			if label == "" {
				label = x.link.Target
			}
			if label == "" {
				label = x.link.Heading
			}
			b.WriteString(plainMath(label))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return html.UnescapeString(b.String())
}

type imageRenderer struct{ resolver *Resolver }

func (r imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, func(w util.BufWriter, source []byte, node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		n := node.(*ast.RawHTML)
		var raw bytes.Buffer
		for i := 0; i < n.Segments.Len(); i++ {
			segment := n.Segments.At(i)
			raw.Write(segment.Value(source))
		}
		_, err := w.Write(rewriteRawImages(raw.Bytes(), r.resolver))
		return ast.WalkSkipChildren, err
	})
	reg.Register(ast.KindHTMLBlock, func(w util.BufWriter, source []byte, node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		n := node.(*ast.HTMLBlock)
		var raw bytes.Buffer
		for i := 0; i < n.Lines().Len(); i++ {
			segment := n.Lines().At(i)
			raw.Write(segment.Value(source))
		}
		if n.HasClosure() {
			raw.Write(n.ClosureLine.Value(source))
		}
		_, err := w.Write(rewriteRawImages(raw.Bytes(), r.resolver))
		return ast.WalkSkipChildren, err
	})
	reg.Register(ast.KindImage, func(w util.BufWriter, source []byte, node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		n := node.(*ast.Image)
		// Match Goldmark's destination handling before handing the URL to the
		// resolver: unescape Markdown punctuation/entities and encode the URL.
		dest := string(util.URLEscape(n.Destination, true))
		if r.resolver != nil && r.resolver.Image != nil {
			if resolved := r.resolver.Image(dest); resolved != "" {
				dest = resolved
			}
		}
		_, _ = w.WriteString(`<img src="` + html.EscapeString(dest) + `" alt="` + html.EscapeString(nodeText(n, source)) + `"`)
		if n.Title != nil {
			_, _ = w.WriteString(` title="` + html.EscapeString(string(n.Title)) + `"`)
		}
		if n.Attributes() != nil {
			htmlrenderer.RenderAttributes(w, n, htmlrenderer.ImageAttributeFilter)
		}
		_, err := w.WriteString(` />`)
		return ast.WalkSkipChildren, err
	})
}

// Raw HTML is trusted as before. Parse its tokens and change only img src
// attributes, rather than searching and rewriting serialized note HTML.
func rewriteRawImages(raw []byte, resolver *Resolver) []byte {
	if resolver == nil || resolver.Image == nil {
		return raw
	}
	tokenizer := nethtml.NewTokenizer(bytes.NewReader(raw))
	var out bytes.Buffer
	for {
		kind := tokenizer.Next()
		if kind == nethtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return out.Bytes()
			}
			return raw
		}
		// Token may mutate the tokenizer buffer while unescaping attributes.
		// Preserve the original bytes for all tags we do not change.
		original := bytes.Clone(tokenizer.Raw())
		if kind == nethtml.StartTagToken || kind == nethtml.SelfClosingTagToken {
			token := tokenizer.Token()
			if token.Data == "img" {
				changed := false
				for i := range token.Attr {
					if token.Attr[i].Key == "src" {
						if resolved := resolver.Image(token.Attr[i].Val); resolved != "" && resolved != token.Attr[i].Val {
							token.Attr[i].Val = resolved
							changed = true
						}
					}
				}
				if changed {
					out.WriteString(token.String())
					continue
				}
			}
		}
		out.Write(original)
	}
}
