// Package parser handles Markdown parsing with frontmatter and wiki-link support.
// Design: goldmark pipeline with custom AST transformers for [[wiki links]] and ![[embeds]].
package parser

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// WikiLink represents a parsed [[wiki link]] or ![[embed]].
type WikiLink struct {
	Target  string // The note slug/path being linked to
	Alias   string // Display text, if any (e.g. [[Target|Alias]])
	IsEmbed bool   // True for ![[embeds]]
	Heading string // Optional heading anchor (#heading)
}

// ParsedNote holds the result of parsing a Markdown file.
type ParsedNote struct {
	Frontmatter map[string]interface{}
	HTML        string
	WikiLinks   []WikiLink
	Tags        []string
	Title       string // From frontmatter or first H1
	Excerpt     string // First non-empty paragraph, plain text
}

// wikiLinkRegex matches [[Target]], [[Target|Alias]], [[Target#Heading]], ![[embed]]
var wikiLinkRegex = regexp.MustCompile(`(!?)\[\[([^\[\]]+)\]\]`)

// imageExtensions are the embed targets rendered as <img> instead of note
// embeds. Obsidian resolves ![[pic.png]] to an attachment anywhere in the
// vault; the vault layer fills in the src via ReplaceWikiEmbedImages.
var imageExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".svg": true, ".webp": true, ".avif": true, ".bmp": true, ".ico": true,
}

// isImageTarget reports whether a wiki-embed target names an image file.
func isImageTarget(target string) bool {
	return imageExtensions[strings.ToLower(path.Ext(strings.TrimSpace(target)))]
}

// Parse takes raw Markdown bytes and returns a ParsedNote.
func Parse(src []byte) (*ParsedNote, error) {
	// --- Pre-process: extract wiki links before goldmark sees them ---
	wikiLinks := extractWikiLinks(src)

	// --- Replace [[wiki links]] with placeholder HTML so goldmark doesn't mangle them ---
	processed := replaceWikiLinks(src)

	// --- Build goldmark with frontmatter ---
	md := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // allow raw HTML for our placeholders
		),
	)

	ctx := parser.NewContext()
	var buf bytes.Buffer
	if err := md.Convert(processed, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	frontmatter := normalizeFrontmatter(meta.Get(ctx))
	htmlOut := buf.String()

	// --- Render callouts (admonitions) ---
	htmlOut = renderCallouts(htmlOut)

	// --- Extract title ---
	title := extractTitle(frontmatter, src)

	// --- Extract tags ---
	tags := extractTags(frontmatter)

	// --- Extract excerpt ---
	excerpt := extractExcerpt(src)

	return &ParsedNote{
		Frontmatter: frontmatter,
		HTML:        htmlOut,
		WikiLinks:   wikiLinks,
		Tags:        tags,
		Title:       title,
		Excerpt:     excerpt,
	}, nil
}

// extractWikiLinks finds all [[wiki links]] and ![[embeds]] in the source.
func extractWikiLinks(src []byte) []WikiLink {
	matches := wikiLinkRegex.FindAllSubmatch(src, -1)
	seen := map[string]bool{}
	var links []WikiLink
	for _, m := range matches {
		isEmbed := string(m[1]) == "!"
		inner := string(m[2])
		target, alias, heading := parseWikiLinkInner(inner)
		// Image embeds are asset references, not note links; keeping them out
		// of WikiLinks keeps backlinks and the wiki-link index clean.
		if isEmbed && isImageTarget(target) {
			continue
		}
		key := target + "|" + alias
		if seen[key] {
			continue
		}
		seen[key] = true
		links = append(links, WikiLink{
			Target:  target,
			Alias:   alias,
			IsEmbed: isEmbed,
			Heading: heading,
		})
	}
	return links
}

// parseWikiLinkInner parses "Target#Heading|Alias" into its parts.
func parseWikiLinkInner(inner string) (string, string, string) {
	alias := ""
	heading := ""
	// Split alias
	if idx := strings.Index(inner, "|"); idx >= 0 {
		alias = strings.TrimSpace(inner[idx+1:])
		inner = inner[:idx]
	}
	// Split heading
	if idx := strings.Index(inner, "#"); idx >= 0 {
		heading = strings.TrimSpace(inner[idx+1:])
		inner = inner[:idx]
	}
	target := strings.TrimSpace(inner)
	return target, alias, heading
}

// replaceWikiLinks substitutes [[wiki links]] with HTML anchor placeholders.
// The frontend renderer will later resolve slugs to actual paths.
func replaceWikiLinks(src []byte) []byte {
	frontmatter, body := splitFrontmatter(src)
	replacedBody := wikiLinkRegex.ReplaceAllFunc(body, wikiLinkHTML)
	if len(frontmatter) == 0 {
		return replacedBody
	}
	out := make([]byte, 0, len(frontmatter)+len(replacedBody))
	out = append(out, frontmatter...)
	out = append(out, replacedBody...)
	return out
}

// splitFrontmatter separates an initial YAML frontmatter block from the Markdown
// body. Wiki-link placeholders must not be injected into frontmatter: doing so
// turns valid YAML such as `"[[Note]]"` into invalid raw HTML and makes
// goldmark-meta treat the entire preamble as visible document content.
func splitFrontmatter(src []byte) ([]byte, []byte) {
	if !bytes.HasPrefix(src, []byte("---\n")) && !bytes.HasPrefix(src, []byte("---\r\n")) {
		return nil, src
	}
	lines := bytes.SplitAfter(src, []byte("\n"))
	if len(lines) == 0 {
		return nil, src
	}
	offset := len(lines[0])
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(string(lines[i]))
		offset += len(lines[i])
		if trimmed == "---" {
			return src[:offset], src[offset:]
		}
	}
	return nil, src
}

func wikiLinkHTML(match []byte) []byte {
	isEmbed := match[0] == '!'
	inner := string(match)
	if isEmbed {
		inner = inner[3 : len(inner)-2] // strip ![[  ]]
	} else {
		inner = inner[2 : len(inner)-2] // strip [[  ]]
	}
	target, alias, heading := parseWikiLinkInner(inner)
	slug := slugify(target)
	display := alias
	if display == "" {
		display = target
	}
	if isEmbed {
		if isImageTarget(target) {
			// Image embed: rendered as <img>; the vault layer resolves
			// data-asset to a /vault-assets URL via ReplaceWikiEmbedImages.
			return []byte(`<img class="wiki-embed-image" data-asset="` + stdhtml.EscapeString(target) + `" alt="` + stdhtml.EscapeString(display) + `" loading="lazy">`)
		}
		return []byte(`<div class="wiki-embed" data-target="` + slug + `" data-heading="` + heading + `" data-raw="` + target + `"></div>`)
	}
	href := "/note/" + slug
	if heading != "" {
		href += "#" + slugify(heading)
	}
	return []byte(`<a href="` + href + `" class="wiki-link" data-target="` + slug + `" data-raw="` + target + `" data-alias="` + alias + `">` + display + `</a>`)
}

// slugify converts a note title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9\-_/]`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// wikiEmbedImageRe matches the exact placeholder emitted by wikiLinkHTML for
// image embeds (attribute order is fixed because we generate the tag).
var wikiEmbedImageRe = regexp.MustCompile(`<img class="wiki-embed-image" data-asset="([^"]*)" alt="([^"]*)" loading="lazy">`)

// ReplaceWikiEmbedImages resolves ![[image.png]] placeholders in pre-rendered
// HTML. The resolver maps the raw embed target (as written in the note) to a
// servable URL, or "" when the asset does not exist in the vault; unresolved
// embeds render as a visible broken-embed marker instead of an empty image.
func ReplaceWikiEmbedImages(html string, resolver func(target string) string) string {
	return wikiEmbedImageRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := wikiEmbedImageRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		target := stdhtml.UnescapeString(sub[1])
		src := resolver(target)
		if src == "" {
			return `<span class="wiki-embed wiki-embed-broken">⚠ Image not found: ` + stdhtml.EscapeString(target) + `</span>`
		}
		return `<img class="wiki-embed-image" src="` + stdhtml.EscapeString(src) + `" alt="` + sub[2] + `" loading="lazy">`
	})
}

// ReplaceWikiLinksString resolves wiki-link targets in pre-rendered HTML.
// The resolver function maps a short slugified target to the full vault slug.
// It replaces data-target and href attributes in wiki-link anchors and embeds.
var (
	dataTargetRe = regexp.MustCompile(`data-target="([^"]+)"`)
	hrefNoteRe   = regexp.MustCompile(`href="/note/([^"#]+)(#[^"]*)?"`)
	imgSrcRe     = regexp.MustCompile(`(?i)(<img\b[^>]*?\bsrc\s*=\s*)(["'])([^"']*)(["'])`)
)

func ReplaceWikiLinksString(html string, resolver func(string) string) string {
	html = dataTargetRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := dataTargetRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		resolved := resolver(sub[1])
		if resolved == "" {
			return match
		}
		return `data-target="` + resolved + `"`
	})
	html = hrefNoteRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := hrefNoteRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		resolved := resolver(sub[1])
		if resolved == "" {
			return `href="#unresolved-` + sub[1] + `"`
		}
		fragment := ""
		if len(sub) >= 3 {
			fragment = sub[2]
		}
		return `href="/note/` + resolved + fragment + `"`
	})
	return html
}

// ReplaceWikiLinkDisplay replaces the display text of wiki links when the
// resolved target differs from the raw text. The titleResolver maps a
// resolved slug to its note title (or "" if unknown).
// RewriteImageSources rewrites image src attributes in rendered HTML.
// The resolver receives the unescaped src value and returns the desired public
// URL. Attribute quoting and unrelated attributes are preserved.
func RewriteImageSources(htmlIn string, resolver func(string) string) string {
	return imgSrcRe.ReplaceAllStringFunc(htmlIn, func(match string) string {
		sub := imgSrcRe.FindStringSubmatch(match)
		if len(sub) < 5 {
			return match
		}
		prefix, quote, src, closingQuote := sub[1], sub[2], sub[3], sub[4]
		if quote != closingQuote {
			return match
		}
		resolved := resolver(stdhtml.UnescapeString(src))
		if resolved == "" {
			resolved = src
		}
		return prefix + quote + stdhtml.EscapeString(resolved) + quote
	})
}

func ReplaceWikiLinkDisplay(html string, titleResolver func(string) string) string {
	// Match <a ... data-raw="X" ...>Y</a> — replace Y with the resolved title
	// We do this by finding each wiki-link anchor and replacing its content.
	wikiLinkRe := regexp.MustCompile(`<a([^>]*?)class="wiki-link"([^>]*?)data-raw="([^"]*?)"([^>]*?)>([^<]*?)</a>`)
	html = wikiLinkRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := wikiLinkRe.FindStringSubmatch(match)
		if len(sub) < 6 {
			return match
		}
		rawDisplay := sub[5]
		if strings.Contains(match, `data-alias="`) {
			aliasSub := regexp.MustCompile(`data-alias="([^"]*)"`).FindStringSubmatch(match)
			if len(aliasSub) >= 2 && aliasSub[1] != "" {
				return match
			}
		}
		// Extract data-target to get the resolved slug
		targetSub := dataTargetRe.FindStringSubmatch(match)
		if len(targetSub) < 2 {
			return match
		}
		title := titleResolver(targetSub[1])
		if title == "" || title == rawDisplay {
			return match
		}
		// Rebuild the anchor with the new display text
		prefix := `<a` + sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
		return prefix + title + "</a>"
	})
	return html
}

// extractTitle returns the title from frontmatter "title" key, or the first H1.
// renderCallouts transforms Obsidian-style callout blockquotes into styled divs.
// Goldmark renders `> [!type] Title` as `<blockquote><p>[!type] Title<br/>...</p></blockquote>`.
// This function detects that pattern and replaces it with a styled callout div.
var calloutRe = regexp.MustCompile(`<blockquote>\s*<p>\[!(\w+)\]([+-])?([\s\S]*?)</blockquote>`)

func renderCallouts(html string) string {
	return calloutRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := calloutRe.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		calloutType := strings.ToLower(sub[1])
		foldChar := sub[2] // "+" = default open, "-" = default closed, "" = default open
		content := sub[3]
		// Remove trailing </p> if present
		content = strings.TrimSuffix(content, "</p>")
		content = strings.TrimSuffix(content, "</p>\n")
		content = strings.TrimSpace(content)

		// Split content into optional title and body
		// The title is text before the first <br />, the body is after
		var title, body string
		if idx := strings.Index(content, "<br />"); idx >= 0 {
			title = strings.TrimSpace(content[:idx])
			body = strings.TrimSpace(content[idx+6:])
		} else if idx := strings.Index(content, "<br/>"); idx >= 0 {
			title = strings.TrimSpace(content[:idx])
			body = strings.TrimSpace(content[idx+5:])
		} else {
			body = strings.TrimSpace(content)
		}

		// Map callout types to icons/labels
		label := titleASCII(calloutType)
		switch calloutType {
		case "summary":
			label = "Summary"
		case "note":
			label = "Note"
		case "tip":
			label = "Tip"
		case "warning":
			label = "Warning"
		case "important":
			label = "Important"
		case "caution":
			label = "Caution"
		case "info":
			label = "Info"
		case "question":
			label = "Question"
		case "quote":
			label = "Quote"
		case "example":
			label = "Example"
		case "abstract":
			label = "Abstract"
		}

		var b strings.Builder
		collapsible := foldChar == "-"
		b.WriteString(`<div class="callout callout-` + calloutType)
		if collapsible {
			b.WriteString(` callout-collapsible`)
		}
		b.WriteString(`">`)
		b.WriteString(`<div class="callout-title">`)
		b.WriteString(`<span class="callout-icon">` + calloutIcon(calloutType) + `</span> `)
		if collapsible {
			b.WriteString(`<span class="callout-toggle">\u25BC</span> `)
		}
		if title != "" {
			b.WriteString(title)
		} else {
			b.WriteString(label)
		}
		b.WriteString(`</div>`)
		if body != "" {
			b.WriteString(`<div class="callout-body"`)
			if collapsible {
				b.WriteString(` style="display:none"`)
			}
			b.WriteString(">\n")
			b.WriteString(body)
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>`)
		return b.String()
	})
}

func titleASCII(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func calloutIcon(typ string) string {
	switch typ {
	case "summary", "abstract":
		return "≡"
	case "note":
		return "✎"
	case "tip":
		return "💡"
	case "warning":
		return "⚠"
	case "important":
		return "❗"
	case "caution":
		return "🔥"
	case "info":
		return "ℹ"
	case "question":
		return "❓"
	case "quote":
		return "❝"
	case "example":
		return "📋"
	default:
		return "■"
	}
}

func normalizeFrontmatter(fm map[string]interface{}) map[string]interface{} {
	if fm == nil {
		return map[string]interface{}{}
	}
	normalized, ok := normalizeYAMLValue(fm).(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return normalized
}

func normalizeYAMLValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, child := range v {
			out[key] = normalizeYAMLValue(child)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, child := range v {
			out[fmt.Sprint(key)] = normalizeYAMLValue(child)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, child := range v {
			out[i] = normalizeYAMLValue(child)
		}
		return out
	default:
		return value
	}
}

func extractTitle(fm map[string]interface{}, src []byte) string {
	if t, ok := fm["title"]; ok {
		if ts, ok := t.(string); ok && ts != "" {
			return ts
		}
	}
	// Find first H1
	h1Re := regexp.MustCompile(`(?m)^#\s+(.+)$`)
	if m := h1Re.FindSubmatch(src); m != nil {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}

// extractTags collects tags from frontmatter "tags" key (string or []interface{}).
func extractTags(fm map[string]interface{}) []string {
	raw, ok := fm["tags"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		var tags []string
		for _, t := range v {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
		return tags
	case string:
		// comma-separated or single
		parts := strings.Split(v, ",")
		var tags []string
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				tags = append(tags, t)
			}
		}
		return tags
	}
	return nil
}

// PlainText removes frontmatter and common Markdown syntax for search/indexing.
func PlainText(src []byte) string {
	content := stripFrontmatter(src)
	return stripMarkdown(content)
}

// extractExcerpt returns the first non-empty paragraph as plain text.
func extractExcerpt(src []byte) string {
	plain := PlainText(src)
	// Take first 200 chars
	plain = strings.TrimSpace(plain)
	if len(plain) > 200 {
		plain = plain[:200] + "…"
	}
	return plain
}

// stripFrontmatter removes YAML frontmatter delimited by ---.
func stripFrontmatter(src []byte) []byte {
	s := strings.TrimSpace(string(src))
	if !strings.HasPrefix(s, "---") {
		return src
	}
	end := strings.Index(s[3:], "---")
	if end < 0 {
		return src
	}
	return []byte(s[end+6:])
}

// stripMarkdown removes common Markdown syntax for plain-text excerpt.
func stripMarkdown(src []byte) string {
	s := string(src)
	// Remove wiki links
	s = wikiLinkRegex.ReplaceAllStringFunc(s, func(m string) string {
		_, alias, _ := parseWikiLinkInner(m[2 : len(m)-2])
		if alias != "" {
			return alias
		}
		return m[2 : len(m)-2]
	})
	// Remove headings
	s = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(s, "")
	// Remove bold/italic
	s = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`_{1,3}([^_]+)_{1,3}`).ReplaceAllString(s, "$1")
	// Unwrap inline code but keep the code text searchable.
	s = regexp.MustCompile("`([^`]+)`").ReplaceAllString(s, "$1")
	// Remove links
	s = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(s, "$1")
	// Remove images
	s = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`).ReplaceAllString(s, "")
	// Collapse whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Slugify is exported for use by other packages.
func Slugify(s string) string {
	return slugify(s)
}

// Ensure ast import is used (goldmark requires it for custom transformers).
var _ = ast.KindDocument
