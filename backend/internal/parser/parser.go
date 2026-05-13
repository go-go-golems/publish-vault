// Package parser handles Markdown parsing with frontmatter and wiki-link support.
// Design: goldmark pipeline with custom AST transformers for [[wiki links]] and ![[embeds]].
package parser

import (
	"bytes"
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

	frontmatter := meta.Get(ctx)
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
func parseWikiLinkInner(inner string) (target, alias, heading string) {
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
	target = strings.TrimSpace(inner)
	return
}

// replaceWikiLinks substitutes [[wiki links]] with HTML anchor placeholders.
// The frontend renderer will later resolve slugs to actual paths.
func replaceWikiLinks(src []byte) []byte {
	return wikiLinkRegex.ReplaceAllFunc(src, func(match []byte) []byte {
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
			return []byte(`<div class="wiki-embed" data-target="` + slug + `" data-heading="` + heading + `" data-raw="` + target + `"></div>`)
		}
		href := "/note/" + slug
		if heading != "" {
			href += "#" + slugify(heading)
		}
		return []byte(`<a href="` + href + `" class="wiki-link" data-target="` + slug + `" data-raw="` + display + `">` + display + `</a>`)
	})
}

// slugify converts a note title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9\-_/]`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// ReplaceWikiLinksString resolves wiki-link targets in pre-rendered HTML.
// The resolver function maps a short slugified target to the full vault slug.
// It replaces data-target and href attributes in wiki-link anchors and embeds.
var (
	dataTargetRe = regexp.MustCompile(`data-target="([^"]+)"`)
	hrefNoteRe   = regexp.MustCompile(`href="/note/([^"]+)"`)
	dataRawRe    = regexp.MustCompile(`data-raw="([^"]*)"`)
)

func ReplaceWikiLinksString(html string, resolver func(string) string) string {
	html = dataTargetRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := dataTargetRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		resolved := resolver(sub[1])
		return `data-target="` + resolved + `"`
	})
	html = hrefNoteRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := hrefNoteRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		resolved := resolver(sub[1])
		return `href="/note/` + resolved + `"`
	})
	return html
}

// ReplaceWikiLinkDisplay replaces the display text of wiki links when the
// resolved target differs from the raw text. The titleResolver maps a
// resolved slug to its note title (or "" if unknown).
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
		prefix := sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
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
		label := strings.Title(calloutType)
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

// extractExcerpt returns the first non-empty paragraph as plain text.
func extractExcerpt(src []byte) string {
	// Strip frontmatter
	content := stripFrontmatter(src)
	// Strip wiki links, markdown syntax
	plain := stripMarkdown(content)
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
	// Remove inline code
	s = regexp.MustCompile("`[^`]+`").ReplaceAllString(s, "")
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
