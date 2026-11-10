// Package parser handles Markdown parsing with frontmatter and wiki-link support.
// Design: Goldmark extensions represent callouts and wiki links as AST nodes.
package parser

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"
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

// imageExtensions are embed targets rendered as images rather than note embeds.
// The vault resolver maps ![[pic.png]] to an attachment URL.
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
	note, document, err := ParseDocument(src)
	if err != nil {
		return nil, err
	}
	note.HTML, err = document.Render(nil)
	return note, err
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
//
// The match is case-insensitive because the vault's walk accepts "Note.MD".
// This is the single definition of "strip a note's extension" — pathToSlug and
// buildWikiLinkIndex call it too, so a link's target and a note's slug cannot
// disagree about whether ".MD" is an extension.
func StripNoteExtension(target string) string {
	if len(target) <= len(".md") {
		return target
	}
	if !strings.EqualFold(target[len(target)-len(".md"):], ".md") {
		return target
	}
	return target[:len(target)-len(".md")]
}

// sourceParts is the result of splitting a Markdown source into an optional
// leading YAML frontmatter block and the document body. It is the single
// boundary used by metadata-reference extraction and plain-text indexing, so
// both agree with goldmark-meta about where frontmatter ends.
//
// frontmatter is nil when no complete frontmatter block is present; body is
// then the original source and bodyOffset is 0. When a block is present,
// frontmatter and body are aliasing slices of src (frontmatter = src[:bodyOffset],
// body = src[bodyOffset:]), so the two reconstruct the source byte-for-byte
// without a copy. The delimiter rule is isFrontmatterDelimiter, which mirrors
// goldmark-meta's isSeparator: a non-empty line of dashes, optionally
// surrounded by whitespace. The opening line must be the very first line of the
// source; an unterminated block is left as ordinary source rather than being
// stripped, matching goldmark-meta (which only closes at a separator).
type sourceParts struct {
	frontmatter []byte
	body        []byte
	bodyOffset  int
}

func (p sourceParts) hasFrontmatter() bool {
	return p.frontmatter != nil
}

// splitSource separates an initial YAML frontmatter block from the Markdown
// body. The boundary mirrors goldmark-meta exactly (see isFrontmatterDelimiter)
// so that any delimiter the metadata parser accepts is also the boundary the
// metadata extraction protects: a four-dash preamble that goldmark-meta parses
// as metadata is not treated as Markdown body.
//
// No frontmatter is returned when the opening line is not a delimiter or when
// no closing delimiter is ever reached; the source is then returned untouched
// as the body, so an unterminated preamble is not silently deleted.
func splitSource(src []byte) sourceParts {
	lines := bytes.SplitAfter(src, []byte("\n"))
	// No complete first line (empty input, or a single line with no newline):
	// there can be no opening delimiter with a newline after it. This matches
	// the previous stripFrontmatter, which required splitLine to return ok.
	if len(lines) == 0 || !bytes.HasSuffix(lines[0], []byte("\n")) {
		return sourceParts{body: src}
	}
	if !isFrontmatterDelimiter(string(lines[0])) {
		return sourceParts{body: src}
	}
	offset := len(lines[0])
	for i := 1; i < len(lines); i++ {
		offset += len(lines[i])
		if isFrontmatterDelimiter(string(lines[i])) {
			return sourceParts{
				frontmatter: src[:offset],
				body:        src[offset:],
				bodyOffset:  offset,
			}
		}
	}
	return sourceParts{body: src}
}

var whitespaceRe = regexp.MustCompile(`\s+`)

// HeadingIndex maps normalized heading text and IDs to Goldmark-assigned IDs.
type HeadingIndex struct {
	byText map[string]string
	ids    map[string]bool
}

// Lookup returns the first heading with the given normalized text or exact ID.
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

// stripFrontmatter removes a leading YAML frontmatter block delimited by a
// goldmark-meta separator line. The boundary is the single splitSource every
// parser consumer uses (see sourceParts), so what is stripped for plain-text
// search/excerpts is exactly what the pre-passes protect and what goldmark-meta
// parses as metadata.
//
// Delimiters are matched as whole lines, mirroring goldmark-meta (the
// extension that actually parses the frontmatter): the block opens only when
// the very first line consists of dashes, and closes at the next such line.
// Matching a bare "---" substring instead would cut valid frontmatter short —
// a scalar such as `title: "before---after"` would end the block mid-document
// and leak the remaining YAML into the body.
func stripFrontmatter(src []byte) []byte {
	return splitSource(src).body
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
