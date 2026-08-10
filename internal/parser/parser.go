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
	// --- Lift LaTeX math out before anything else reads or rewrites the source ---
	// Math runs first because replaceWikiLinks injects raw HTML (quotes,
	// attributes) into the body; scanning that output for `$` could swallow
	// markup. The math itself comes back in RestoreMath, after every HTML
	// post-pass.
	processed, mathSpans := replaceMathInBody(src)

	// --- Extract wiki links from the masked source, not the raw one ---
	// `[[Foo]]` inside a formula is literal TeX, not a link. Scanning the raw
	// source would still record it, and buildBacklinks would then give Foo a
	// backlink pointing at a note that does not link to it.
	wikiLinks := extractWikiLinks(processed)

	// --- Replace [[wiki links]] with placeholder HTML so goldmark doesn't mangle them ---
	processed = replaceWikiLinks(processed)

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

	// --- Point [[#Heading]] links at the ids goldmark actually emitted ---
	// Runs before RestoreMath on purpose: a heading containing math carries the
	// same placeholders on both sides of the comparison while the math is still
	// lifted out, so `## $\sigma$ notes` and `[[#$\sigma$ notes]]` still match.
	htmlOut = resolveSelfHeadingLinks(htmlOut)

	// --- Put math back, last, so no other pass ever sees its markup ---
	htmlOut = RestoreMath(htmlOut, mathSpans)

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
		// of WikiLinks keeps backlinks and the wiki-link index clean. So is a
		// [[#Heading]] link, which names an anchor in this very note: it used to
		// enter the list with an empty Target, where it could only ever fail to
		// resolve, and showed up as a blank entry in the agent Markdown view.
		if isEmbed && isImageTarget(target) {
			continue
		}
		if target == "" {
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
	target := StripNoteExtension(strings.TrimSpace(inner))
	return target, alias, heading
}

// StripNoteExtension removes a trailing ".md" from a wiki-link target.
//
// Obsidian treats [[Note]] and [[Note.md]] as the same link, and emits the
// second form under the "absolute path in vault" setting. publish-vault derives
// every slug from an extension-less path (pathToSlug, buildWikiLinkIndex), while
// slugify maps "." to "-" rather than dropping it — so an unstripped target
// slugifies to "…-md" and resolves either to nothing or, when the vault happens
// to hold a note named "… md", to the wrong note with no visible breakage.
// Stripping here, at the one place every consumer of a target goes through,
// keeps the two forms interchangeable.
//
// Only ".md" is stripped: the vault loads no other extension, so ".markdown" is
// part of a note's name rather than a suffix. A bare ".md" target is left as-is
// rather than reduced to the empty string.
func StripNoteExtension(target string) string {
	if len(target) <= len(".md") {
		return target
	}
	if !strings.EqualFold(target[len(target)-len(".md"):], ".md") {
		return target
	}
	return target[:len(target)-len(".md")]
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
	if target == "" {
		// [[#Heading]] — a link to a heading in *this* note, not to another
		// note. It carries no target, so it must not go through the /note/<slug>
		// path at all: an empty slug produced href="/note/#heading", which sends
		// the reader to the vault root, and an empty display string, which makes
		// the link invisible. The real anchor id is only known once goldmark has
		// rendered the headings, so emit a placeholder for
		// resolveSelfHeadingLinks to finish.
		if heading == "" {
			return match // degenerate [[#]] / [[|x]]: leave the source text alone
		}
		if display == "" {
			display = heading
		}
		return []byte(`<a href="#" class="wiki-link wiki-link-self" data-heading="` + stdhtml.EscapeString(heading) + `" data-alias="` + stdhtml.EscapeString(alias) + `">` + stdhtml.EscapeString(display) + `</a>`)
	}
	href := "/note/" + slug
	if heading != "" {
		// A provisional fragment. slugify is not how the target note's heading
		// ids are made, so the vault replaces this with the id the target
		// actually rendered (ResolveWikiLinkHeadings); data-heading carries the
		// heading text that pass needs, since the fragment alone has already
		// lost it.
		href += "#" + slugify(heading)
	}
	return []byte(`<a href="` + href + `" class="wiki-link" data-target="` + slug + `" data-raw="` + target + `" data-heading="` + stdhtml.EscapeString(heading) + `" data-alias="` + alias + `">` + display + `</a>`)
}

var (
	// selfHeadingLinkRe matches the placeholder emitted by wikiLinkHTML for a
	// same-note heading link. Attribute order is fixed because we generate it.
	selfHeadingLinkRe = regexp.MustCompile(`<a href="#" class="wiki-link wiki-link-self" data-heading="([^"]*)" data-alias="([^"]*)">`)
	// renderedHeadingRe captures the id and inner markup of every rendered
	// heading that carries an id.
	renderedHeadingRe = regexp.MustCompile(`(?s)<h[1-6][^>]*\bid="([^"]*)"[^>]*>(.*?)</h[1-6]>`)
	htmlTagRe         = regexp.MustCompile(`<[^>]*>`)
	whitespaceRe      = regexp.MustCompile(`\s+`)
)

// HeadingIndex maps a rendered note's headings to the ids goldmark gave them.
//
// It exists because a wiki-link fragment cannot be *computed*: goldmark's auto
// heading IDs and slugify disagree (goldmark drops "." and "—" outright where
// slugify turns them into "-", so "9.2 Kernel K0" becomes "92-kernel-k0" rather
// than "9-2-kernel-k0"), and goldmark additionally suffixes duplicate headings
// with "-1", "-2". Reading the ids back out of the rendered HTML is exact by
// construction and cannot drift when goldmark changes its algorithm.
type HeadingIndex struct {
	byText map[string]string
	ids    map[string]bool
}

// BuildHeadingIndex reads every id-bearing heading out of rendered note HTML.
func BuildHeadingIndex(htmlIn string) *HeadingIndex {
	idx := &HeadingIndex{byText: map[string]string{}, ids: map[string]bool{}}
	for _, m := range renderedHeadingRe.FindAllStringSubmatch(htmlIn, -1) {
		id, inner := m[1], m[2]
		idx.ids[id] = true
		key := normalizeHeadingKey(stdhtml.UnescapeString(htmlTagRe.ReplaceAllString(inner, "")))
		if key == "" {
			continue
		}
		if _, exists := idx.byText[key]; !exists {
			idx.byText[key] = id
		}
	}
	return idx
}

// Lookup resolves a heading as written in a wiki link to the id it must point
// at. Matching is on heading text, case-insensitively and with runs of
// whitespace collapsed, which is how Obsidian matches; the first heading with
// that text wins, again matching Obsidian. A heading that matches no text is
// tried against the ids themselves, so a link written in the already-slugified
// form still lands. A nil index resolves nothing, so callers can cache misses.
func (h *HeadingIndex) Lookup(heading string) (string, bool) {
	if h == nil {
		return "", false
	}
	if id, ok := h.byText[normalizeHeadingKey(heading)]; ok {
		return id, true
	}
	if h.ids[heading] {
		return heading, true
	}
	return "", false
}

// resolveSelfHeadingLinks points [[#Heading]] links at the heading goldmark
// actually rendered in this very note. Anything it cannot match becomes a
// visibly unresolved link rather than a silent jump to the top of the page.
func resolveSelfHeadingLinks(htmlIn string) string {
	if !strings.Contains(htmlIn, `class="wiki-link wiki-link-self"`) {
		return htmlIn
	}
	idx := BuildHeadingIndex(htmlIn)

	return selfHeadingLinkRe.ReplaceAllStringFunc(htmlIn, func(match string) string {
		sub := selfHeadingLinkRe.FindStringSubmatch(match)
		heading := stdhtml.UnescapeString(sub[1])
		id, ok := idx.Lookup(heading)
		if !ok {
			return `<a href="#unresolved-` + stdhtml.EscapeString(slugify(heading)) +
				`" class="wiki-link wiki-link-self broken" data-heading="` + sub[1] + `" data-alias="` + sub[2] + `">`
		}
		return `<a href="#` + stdhtml.EscapeString(id) +
			`" class="wiki-link wiki-link-self" data-heading="` + sub[1] + `" data-alias="` + sub[2] + `">`
	})
}

func normalizeHeadingKey(s string) string {
	return whitespaceRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), " ")
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

// crossNoteHeadingLinkRe matches a resolved wiki-link anchor that carries a
// heading. The attribute order is fixed because wikiLinkHTML generates it and
// ReplaceWikiLinksString only rewrites values in place.
var crossNoteHeadingLinkRe = regexp.MustCompile(
	`<a href="/note/([^"#]*)(#[^"]*)?" class="wiki-link" data-target="([^"]*)" data-raw="([^"]*)" data-heading="([^"]+)" data-alias="([^"]*)">`)

// ResolveWikiLinkHeadings replaces the provisional slugified fragment of a
// [[Note#Heading]] link with the id the *target* note actually rendered.
//
// The fragment wikiLinkHTML wrote is a guess made with slugify, and the target's
// heading ids come from goldmark, which does not agree with slugify on any
// heading containing punctuation (see HeadingIndex). The link therefore opened
// the right note at the top of the page whenever the two differed.
//
// resolver receives the resolved target slug and the heading as written, and
// returns the real id. When it declines — the target is not a published note,
// the heading does not exist there, or the link is a block reference
// ([[Note#^blockid]]) — the fragment is dropped rather than left pointing at an
// id that is known not to exist. The link still opens the note.
func ResolveWikiLinkHeadings(html string, resolver func(slug, heading string) (string, bool)) string {
	// rebuildHTML runs this over every note on every reload, and most notes hold
	// no heading link at all. A substring scan is far cheaper than the regexp,
	// and every anchor this pass can act on carries the attribute.
	if !strings.Contains(html, ` data-heading="`) {
		return html
	}
	return crossNoteHeadingLinkRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := crossNoteHeadingLinkRe.FindStringSubmatch(match)
		if len(sub) < 7 {
			return match
		}
		slug, target, raw, heading, alias := sub[1], sub[3], sub[4], sub[5], sub[6]
		fragment := ""
		if id, ok := resolver(slug, stdhtml.UnescapeString(heading)); ok {
			fragment = "#" + stdhtml.EscapeString(id)
		}
		return `<a href="/note/` + slug + fragment + `" class="wiki-link" data-target="` + target +
			`" data-raw="` + raw + `" data-heading="` + heading + `" data-alias="` + alias + `">`
	})
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

// StripFrontmatter removes a leading YAML frontmatter block delimited by ---.
// The source is returned unchanged when no frontmatter is present.
func StripFrontmatter(src []byte) []byte {
	return stripFrontmatter(src)
}

// stripFrontmatter removes YAML frontmatter delimited by ---.
//
// Delimiters are matched as whole lines, mirroring goldmark-meta (the extension
// that actually parses the frontmatter): the block opens only when the very
// first line consists of dashes, and closes at the next such line. Matching a
// bare "---" substring instead would cut valid frontmatter short — a scalar
// such as `title: "before---after"` would end the block mid-document and leak
// the remaining YAML into the body.
func stripFrontmatter(src []byte) []byte {
	s := string(src)
	line, rest, ok := splitLine(s)
	if !ok || !isFrontmatterDelimiter(line) {
		return src
	}
	for rest != "" {
		line, next, ok := splitLine(rest)
		// The closing delimiter counts even as the last line of a file with no
		// trailing newline ("---\ntitle: x\n---"), so test the line before
		// giving up on an incomplete one — otherwise the whole source, its
		// frontmatter included, would be returned as note body.
		if isFrontmatterDelimiter(line) {
			return []byte(next)
		}
		if !ok {
			break
		}
		rest = next
	}
	return src
}

// splitLine returns the first line of s (without its line break), the
// remainder, and whether s contained a complete line at all.
func splitLine(s string) (string, string, bool) {
	i := strings.IndexByte(s, '\n')
	if i < 0 {
		return s, "", false
	}
	return strings.TrimSuffix(s[:i], "\r"), s[i+1:], true
}

// isFrontmatterDelimiter reports whether a line delimits a frontmatter block:
// a non-empty run of dashes, optionally surrounded by whitespace. This matches
// goldmark-meta's isSeparator, so this package and the goldmark pipeline agree
// on where the frontmatter block ends.
func isFrontmatterDelimiter(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return strings.Trim(trimmed, "-") == ""
}

// stripMarkdown removes common Markdown syntax for plain-text excerpt.
func stripMarkdown(src []byte) string {
	// Drop math delimiters first, keeping the TeX body, so a note about the
	// normal distribution is findable by searching "sigma". Running before the
	// emphasis/link regexes matters: they would otherwise chew through the
	// underscores and backslashes inside a formula.
	s := string(StripMathDelimiters(src))
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
