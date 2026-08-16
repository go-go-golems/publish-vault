package parser

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseFrontmatterWikiLinksAndEmbeds(t *testing.T) {
	src := []byte(`---
title: Custom Title
tags:
  - philosophy
  - notes
---
# Ignored Heading

First paragraph links to [[Target Note|target alias]], [[Other#Deep Heading]], and embeds ![[Embedded Note]].
`)

	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Title != "Custom Title" {
		t.Fatalf("Title = %q, want Custom Title", parsed.Title)
	}
	if got, want := parsed.Tags, []string{"philosophy", "notes"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	}
	if len(parsed.WikiLinks) != 3 {
		t.Fatalf("WikiLinks len = %d, want 3 (%#v)", len(parsed.WikiLinks), parsed.WikiLinks)
	}
	if parsed.WikiLinks[0].Target != "Target Note" || parsed.WikiLinks[0].Alias != "target alias" {
		t.Fatalf("first wiki link = %#v", parsed.WikiLinks[0])
	}
	if parsed.WikiLinks[1].Target != "Other" || parsed.WikiLinks[1].Heading != "Deep Heading" {
		t.Fatalf("heading wiki link = %#v", parsed.WikiLinks[1])
	}
	if !parsed.WikiLinks[2].IsEmbed || parsed.WikiLinks[2].Target != "Embedded Note" {
		t.Fatalf("embed wiki link = %#v", parsed.WikiLinks[2])
	}
	if parsed.HTML == "" || parsed.Excerpt == "" {
		t.Fatalf("HTML and Excerpt should be populated: html=%q excerpt=%q", parsed.HTML, parsed.Excerpt)
	}
}

func TestParseFallsBackToFirstHeadingAndStringTags(t *testing.T) {
	parsed, err := Parse([]byte(`---
tags: alpha, beta
---
# Heading Title

Body text.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Title != "Heading Title" {
		t.Fatalf("Title = %q, want Heading Title", parsed.Title)
	}
	if got, want := parsed.Tags, []string{"alpha", "beta"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	}
}

func TestPlainTextPreservesInlineCode(t *testing.T) {
	got := PlainText([]byte("# Tooling\n\nRun `retroctl publish` after editing."))
	if !strings.Contains(got, "retroctl publish") {
		t.Fatalf("PlainText() = %q, want inline code contents preserved", got)
	}
	if strings.Contains(got, "`") {
		t.Fatalf("PlainText() = %q, want backticks stripped", got)
	}
}

func TestSlugifyPreservesSlashPaths(t *testing.T) {
	if got, want := Slugify("Folder/My Note!"), "folder/my-note"; got != want {
		t.Fatalf("Slugify() = %q, want %q", got, want)
	}
}

func TestCalloutRendering(t *testing.T) {
	src := []byte(`# Test

> [!summary]\n> This is a summary callout.\n> With multiple lines.
`)
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `class="callout callout-summary"`) {
		t.Fatalf("HTML should contain callout-summary div, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `callout-title`) {
		t.Fatalf("HTML should contain callout-title, got: %s", parsed.HTML)
	}
	if contains(parsed.HTML, `<blockquote`) {
		t.Fatalf("HTML should not contain raw blockquote, got: %s", parsed.HTML)
	}
}

func TestCalloutWithTitle(t *testing.T) {
	src := []byte("# Test\n\n> [!warning] Custom Warning Title\n> Body text here.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `callout-warning`) {
		t.Fatalf("HTML should contain callout-warning, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, "Custom Warning Title") {
		t.Fatalf("HTML should contain custom title, got: %s", parsed.HTML)
	}
}

func TestFrontmatterWikiLinksAreNotRenderedAsPreamble(t *testing.T) {
	src := []byte(`---
title: Report
related_reports:
  - "[[PROJECT REPORT - Keycloak OS1 Login Theme - A Technical Deep Dive]]"
---

# Report

See [[PROJECT REPORT - Keycloak OS1 Login Theme - A Technical Deep Dive]].
`)
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Title != "Report" {
		t.Fatalf("Title = %q, want Report", parsed.Title)
	}
	if contains(parsed.HTML, "related_reports:") || contains(parsed.HTML, "<!-- yaml:") {
		t.Fatalf("frontmatter should not render as document preamble, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `class="wiki-link"`) {
		t.Fatalf("body wiki link should still render, got: %s", parsed.HTML)
	}
}

func TestNestedFrontmatterIsJSONEncodable(t *testing.T) {
	parsed, err := Parse([]byte(`---
title: Nested Metadata
RelatedFiles:
  - Path: pkg/media/gst/recording.go
    Note: Direct recording builder with x264enc
ExternalSources:
  - URL: https://example.com
    Meta:
      Kind: reference
---

# Body
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := json.Marshal(parsed.Frontmatter); err != nil {
		t.Fatalf("frontmatter should be JSON encodable: %v (%#v)", err, parsed.Frontmatter)
	}
	related, ok := parsed.Frontmatter["RelatedFiles"].([]interface{})
	if !ok || len(related) != 1 {
		t.Fatalf("RelatedFiles = %#v, want one item slice", parsed.Frontmatter["RelatedFiles"])
	}
	if _, ok := related[0].(map[string]interface{}); !ok {
		t.Fatalf("RelatedFiles[0] = %T, want map[string]interface{}", related[0])
	}
}

func TestWikiLinkDataRaw(t *testing.T) {
	parsed, err := Parse([]byte("# Test\n\nSee [[Tribal/App-Auth]] and [[Fundamentals/Access|Custom Alias]].\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `data-raw="Tribal/App-Auth"`) {
		t.Fatalf("HTML should contain data-raw for first link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `data-raw="Fundamentals/Access"`) {
		t.Fatalf("HTML should contain raw target for aliased link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `data-alias="Custom Alias"`) {
		t.Fatalf("HTML should contain data-alias for aliased link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, ">Tribal/App-Auth</a>") {
		t.Fatalf("HTML should display raw target for first link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, ">Custom Alias</a>") {
		t.Fatalf("HTML should display alias for second link, got: %s", parsed.HTML)
	}
}

func TestReplaceWikiLinksStringPreservesHeadingFragment(t *testing.T) {
	html := `<p><a href="/note/tribal/app-auth#authorization-flow" class="wiki-link" data-target="tribal/app-auth" data-raw="Tribal/App Auth" data-alias="">Tribal/App Auth</a></p>`
	got := ReplaceWikiLinksString(html, func(target string) string {
		if target == "tribal/app-auth" {
			return "research/kb/tribal/app-auth"
		}
		return target
	})
	if !contains(got, `href="/note/research/kb/tribal/app-auth#authorization-flow"`) {
		t.Fatalf("heading fragment should be preserved, got: %s", got)
	}
	if !contains(got, `data-target="research/kb/tribal/app-auth"`) {
		t.Fatalf("data-target should resolve without fragment, got: %s", got)
	}
}

func TestReplaceWikiLinksStringUnresolvedTargetsAreNotCrawlableNoteLinks(t *testing.T) {
	html := `<p><a href="/note/gettier-problem" class="wiki-link" data-target="gettier-problem" data-raw="Gettier Problem" data-alias="">Gettier Problem</a></p>`
	got := ReplaceWikiLinksString(html, func(target string) string { return "" })
	if contains(got, `href="/note/gettier-problem"`) {
		t.Fatalf("unresolved wiki link stayed crawlable: %s", got)
	}
	if !contains(got, `href="#unresolved-gettier-problem"`) {
		t.Fatalf("unresolved wiki link did not become same-page anchor: %s", got)
	}
}

func TestReplaceWikiLinkDisplayPreservesAlias(t *testing.T) {
	html := `<p><a href="/note/research/kb/target" class="wiki-link" data-target="research/kb/target" data-raw="Target" data-alias="Custom Alias">Custom Alias</a></p>`
	got := ReplaceWikiLinkDisplay(html, func(_ string) string { return "Resolved Title" })
	if !contains(got, `>Custom Alias</a>`) {
		t.Fatalf("explicit alias should be preserved, got: %s", got)
	}
}

func TestRewriteImageSources(t *testing.T) {
	html := `<p><img src="images/planet.png" alt="Planet" /> <img src='Sketch Folder/m5 dial.png' /></p>`
	got := RewriteImageSources(html, func(src string) string {
		return "/assets/" + strings.ReplaceAll(src, " ", "%20")
	})
	if !contains(got, `src="/assets/images/planet.png"`) {
		t.Fatalf("double-quoted image src was not rewritten, got: %s", got)
	}
	if !contains(got, `src='/assets/Sketch%20Folder/m5%20dial.png'`) {
		t.Fatalf("single-quoted image src was not rewritten, got: %s", got)
	}
}

func TestRewriteImageSourcesPreservesMismatchedQuote(t *testing.T) {
	html := `<img src="broken.png' alt="Broken" />`
	got := RewriteImageSources(html, func(string) string { return "/assets/changed.png" })
	if got != html {
		t.Fatalf("mismatched quote should be preserved, got: %s", got)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestWikiLinkDataRawRealFormat(t *testing.T) {
	src := []byte("# Test\n\nSee [[Fundamentals/access-control-models]] here.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `data-raw="Fundamentals/access-control-models"`) {
		t.Fatalf("HTML should contain data-raw, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, ">Fundamentals/access-control-models</a>") {
		t.Fatalf("HTML should display raw target, got: %s", parsed.HTML)
	}
}

func TestCollapsibleCallout(t *testing.T) {
	src := []byte("# Test\n\n> [!warning]- Collapsed Warning\n> Hidden body.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `callout-collapsible`) {
		t.Fatalf("HTML should contain callout-collapsible class, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `callout-toggle`) {
		t.Fatalf("HTML should contain callout-toggle, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `style="display:none"`) {
		t.Fatalf("Collapsed body should be hidden, got: %s", parsed.HTML)
	}
}

func TestOpenCallout(t *testing.T) {
	src := []byte("# Test\n\n> [!tip]+ Always Open\n> Visible body.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if contains(parsed.HTML, `callout-collapsible`) {
		t.Fatalf("[!type]+ should NOT be collapsible, got: %s", parsed.HTML)
	}
}

func TestReplaceWikiLinkDisplayPreservesAnchorTag(t *testing.T) {
	html := `<td><a href="/note/target" class="wiki-link" data-target="target" data-raw="Target" data-alias="">Target</a></td>`
	got := ReplaceWikiLinkDisplay(html, func(slug string) string {
		if slug == "target" {
			return "Resolved Title"
		}
		return ""
	})
	if !strings.HasPrefix(got, "<td><a ") {
		t.Fatalf("Missing <a> opening tag, got: %s", got)
	}
	if !strings.Contains(got, ">Resolved Title</a>") {
		t.Fatalf("Display text not replaced, got: %s", got)
	}
	// Adjacent td content must not bleed into anchor
	if strings.Contains(got, "</td></a>") {
		t.Fatalf("td boundary crossed into anchor: %s", got)
	}
}

func TestReplaceWikiLinkDisplayInTable(t *testing.T) {
	html := `<table><tbody><tr><td><a href="/note/t" class="wiki-link" data-target="t" data-raw="T" data-alias="">T</a></td><td>2026-05-23</td><td>Canonical description</td></tr></tbody></table>`
	got := ReplaceWikiLinkDisplay(html, func(slug string) string {
		if slug == "t" {
			return "DMETA Design System Factory"
		}
		return ""
	})
	// Adjacent cell content must NOT be inside the anchor
	if strings.Contains(got, "2026-05-23</a>") {
		t.Fatalf("Date bled into anchor text: %s", got)
	}
	if strings.Contains(got, "Canonical description</a>") {
		t.Fatalf("Description bled into anchor text: %s", got)
	}
	if !strings.Contains(got, ">DMETA Design System Factory</a>") {
		t.Fatalf("Display not replaced correctly: %s", got)
	}
}

func TestReplaceWikiLinkDisplayPreservesExplicitAlias(t *testing.T) {
	html := `<a href="/note/t" class="wiki-link" data-target="t" data-raw="T" data-alias="My Alias">My Alias</a>`
	got := ReplaceWikiLinkDisplay(html, func(slug string) string {
		return "Resolved Title"
	})
	if !strings.Contains(got, ">My Alias</a>") {
		t.Fatalf("Explicit alias was overwritten: %s", got)
	}
}

func TestWikiLinkInTableRendersCleanly(t *testing.T) {
	src := []byte("| Report | Date |\n| --- | --- |\n| [[ARTICLE - DMETA Factory]] | 2026-05-23 |\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	// The initial parse should produce clean HTML
	if !contains(parsed.HTML, `<a href="/note/article-dmeta-factory" class="wiki-link"`) {
		t.Fatalf("Wiki link not rendered correctly, got: %s", parsed.HTML)
	}
	// Simulate ReplaceWikiLinkDisplay as rebuildHTML does
	got := ReplaceWikiLinkDisplay(parsed.HTML, func(slug string) string {
		if slug == "article-dmeta-factory" {
			return "DMETA Factory: Full Title"
		}
		return ""
	})
	if !strings.Contains(got, "<a href") {
		t.Fatalf("<a> tag missing after display replacement: %s", got)
	}
	if strings.Contains(got, "2026-05-23</a>") {
		t.Fatalf("Table date bled into anchor after display replacement: %s", got)
	}
}

func TestImageEmbedRendersImgPlaceholder(t *testing.T) {
	parsed, err := Parse([]byte("# Doc\n\n![[screen shot.png]]\n\n![[diagram.svg|System diagram]]\n\n![[Regular Note]]\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !strings.Contains(parsed.HTML, `<img class="wiki-embed-image" data-asset="screen shot.png" alt="screen shot.png" loading="lazy">`) {
		t.Fatalf("image embed placeholder missing, got: %s", parsed.HTML)
	}
	if !strings.Contains(parsed.HTML, `data-asset="diagram.svg" alt="System diagram"`) {
		t.Fatalf("alias should become alt text, got: %s", parsed.HTML)
	}
	if !strings.Contains(parsed.HTML, `class="wiki-embed" data-target="regular-note"`) {
		t.Fatalf("note embed should stay a note embed, got: %s", parsed.HTML)
	}
	// Image embeds are asset references, not wiki links.
	for _, l := range parsed.WikiLinks {
		if strings.HasSuffix(strings.ToLower(l.Target), ".png") || strings.HasSuffix(strings.ToLower(l.Target), ".svg") {
			t.Fatalf("image embed leaked into WikiLinks: %#v", l)
		}
	}
}

func TestReplaceWikiEmbedImages(t *testing.T) {
	html := `<img class="wiki-embed-image" data-asset="pic.png" alt="pic.png" loading="lazy"><img class="wiki-embed-image" data-asset="gone.png" alt="gone.png" loading="lazy">`
	out := ReplaceWikiEmbedImages(html, func(target string) string {
		if target == "pic.png" {
			return "/vault-assets/Attachments/pic.png"
		}
		return ""
	})
	if !strings.Contains(out, `<img class="wiki-embed-image" src="/vault-assets/Attachments/pic.png" alt="pic.png" loading="lazy">`) {
		t.Fatalf("resolved embed missing src, got: %s", out)
	}
	if !strings.Contains(out, `Image not found: gone.png`) {
		t.Fatalf("unresolved embed should render broken marker, got: %s", out)
	}
}

// TestStripFrontmatterOnlyMatchesDelimiterLines pins that the frontmatter block
// ends at a "---" line, not at the first "---" substring. A scalar containing
// dashes must not cut the block short and leak the remaining YAML into the body.
func TestStripFrontmatterOnlyMatchesDelimiterLines(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "dashes inside a quoted scalar",
			src:  "---\ntitle: \"before---after\"\ntags: [a]\n---\n# Body\n\nText.\n",
			want: "# Body\n\nText.\n",
		},
		{
			name: "dashes inside a block scalar",
			src:  "---\nsummary: |\n  a --- b\npublish: true\n---\nBody line\n",
			want: "Body line\n",
		},
		{
			name: "no frontmatter leaves a thematic break alone",
			src:  "# Title\n\n---\n\nAfter the rule.\n",
			want: "# Title\n\n---\n\nAfter the rule.\n",
		},
		{
			name: "unterminated frontmatter is left untouched",
			src:  "---\ntitle: x\nno closing delimiter\n",
			want: "---\ntitle: x\nno closing delimiter\n",
		},
		{
			name: "plain frontmatter",
			src:  "---\ntitle: x\n---\nBody\n",
			want: "Body\n",
		},
		{
			name: "closing delimiter at EOF without a trailing newline",
			src:  "---\ntitle: x\ninternal: secret\n---",
			want: "",
		},
		{
			name: "closing delimiter at EOF with a carriage return",
			src:  "---\r\ntitle: x\r\n---\r",
			want: "",
		},
		{
			name: "body at EOF without a trailing newline",
			src:  "---\ntitle: x\n---\nBody",
			want: "Body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(StripFrontmatter([]byte(tt.src))); got != tt.want {
				t.Errorf("StripFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestStripFrontmatterAgreesWithGoldmark pins that what StripFrontmatter treats
// as frontmatter is exactly what the goldmark pipeline parsed as frontmatter:
// the body it returns must not contain any parsed frontmatter key.
func TestStripFrontmatterAgreesWithGoldmark(t *testing.T) {
	src := []byte("---\ntitle: \"before---after\"\nsecret: hunter2\n---\n# Body\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Title != "before---after" {
		t.Fatalf("Parse() title = %q, want before---after", parsed.Title)
	}
	body := string(StripFrontmatter(src))
	if strings.Contains(body, "secret") {
		t.Errorf("StripFrontmatter leaked frontmatter into the body: %q", body)
	}
}

// TestSlugifyTable pins the slug algebra. These outputs are load-bearing: they
// are the live URLs of every published note, so a change here silently breaks
// external links. Rows come from PV-SLUG-020 design doc section 3.1.
func TestSlugifyTable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"underscores_survive", "The_Algebra_of_Intervention_Fields", "the_algebra_of_intervention_fields"},
		{"nested_path_survives",
			"Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields",
			"transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields"},
		{"spaces_to_single_dash", "hello    world", "hello-world"},
		{"dots_become_dashes", "v1.2.3 release", "v1-2-3-release"},
		{"ampersand", "Cats & Dogs", "cats-dogs"},
		{"apostrophe", "Manuel's Notes", "manuel-s-notes"},
		{"accents_are_mangled", "Café Münster", "caf-m-nster"},
		{"cyrillic_is_empty", "Привет мир", ""},
		{"cjk_is_empty", "日本語ノート", ""},
		{"emoji_dropped", "done ✅ shipped 🚀", "done-shipped"},
		{"trailing_slash_survives", "a/b/", "a/b/"},
		{"double_slash_survives", "a//b", "a//b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	// Idempotence is what lets the vault build a normalized fallback index on
	// top of Slugify without risking a redirect loop.
	t.Run("idempotent", func(t *testing.T) {
		for _, tt := range tests {
			once := Slugify(tt.in)
			if twice := Slugify(once); twice != once {
				t.Errorf("Slugify not idempotent for %q: %q -> %q", tt.in, once, twice)
			}
		}
	})
}

// TestStripNoteExtension pins the exact boundary of the ".md" strip: only a
// trailing ".md", case-insensitively, and never at the cost of an empty target.
func TestStripNoteExtension(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Note.md", "Note"},
		{"Note.MD", "Note"},
		{"Folder/Note.md", "Folder/Note"},
		{"Note", "Note"},
		{"Note.md.md", "Note.md"},
		{"readme.markdown", "readme.markdown"},
		{"pic.png", "pic.png"},
		{"md", "md"},
		{".md", ".md"},
		{"", ""},
		{"Notes on foo.md and bar", "Notes on foo.md and bar"},
	}
	for _, c := range cases {
		if got := StripNoteExtension(c.in); got != c.want {
			t.Errorf("StripNoteExtension(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestWikiLinkTargetDropsMarkdownExtension is the regression test for
// PV-WIKILINK-021: [[X.md]] must produce the same slug, href and backlink target
// as [[X]]. Without the strip, "…thesis.md" slugifies to "…thesis-md", which
// matches no note — or, worse, matches an unrelated note named "… md".
func TestWikiLinkTargetDropsMarkdownExtension(t *testing.T) {
	src := []byte("# Zoo\n\n" +
		"| a | [[Transcripts/2026/08/06/RAG DSL for Retrieval/thesis.md#Identity is an API decision]] |\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(parsed.WikiLinks) != 1 {
		t.Fatalf("expected 1 wiki link, got %#v", parsed.WikiLinks)
	}
	wl := parsed.WikiLinks[0]
	if want := "Transcripts/2026/08/06/RAG DSL for Retrieval/thesis"; wl.Target != want {
		t.Errorf("WikiLink.Target = %q, want %q", wl.Target, want)
	}
	if want := "Identity is an API decision"; wl.Heading != want {
		t.Errorf("WikiLink.Heading = %q, want %q", wl.Heading, want)
	}

	want := `href="/note/transcripts/2026/08/06/rag-dsl-for-retrieval/thesis#identity-is-an-api-decision"`
	if !contains(parsed.HTML, want) {
		t.Fatalf("expected %s in HTML, got: %s", want, parsed.HTML)
	}
	if contains(parsed.HTML, "thesis-md") {
		t.Fatalf("extension leaked into the slug: %s", parsed.HTML)
	}
}

// TestWikiLinkMarkdownExtensionVariants covers the forms the strip has to keep
// working alongside: an explicit alias, a note embed, and an image embed whose
// extension must survive untouched.
func TestWikiLinkMarkdownExtensionVariants(t *testing.T) {
	src := []byte("# T\n\n" +
		"[[Folder/Note.md|Custom Alias]]\n\n" +
		"![[Folder/Embedded.MD]]\n\n" +
		"![[Attachments/pic.png]]\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !contains(parsed.HTML, `data-target="folder/note"`) || !contains(parsed.HTML, `>Custom Alias</a>`) {
		t.Errorf("aliased .md link mishandled: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `<div class="wiki-embed" data-target="folder/embedded"`) {
		t.Errorf("note embed did not lose its .MD: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `data-asset="Attachments/pic.png"`) {
		t.Errorf("image embed target was rewritten: %s", parsed.HTML)
	}

	for _, wl := range parsed.WikiLinks {
		if strings.HasSuffix(strings.ToLower(wl.Target), ".md") {
			t.Errorf("extension survived into WikiLinks: %#v", wl)
		}
	}
}

// TestSelfHeadingLinkUsesRenderedHeadingID is the regression test for the
// [[#Heading]] bug. These links used to render as
// `<a href="/note/#heading"></a>`: empty text, and a destination pointing at
// the vault root. They must become same-page anchors carrying the heading as
// their text — and the fragment must be goldmark's *actual* id, which is not
// what slugify would produce for any of these headings.
func TestSelfHeadingLinkUsesRenderedHeadingID(t *testing.T) {
	src := []byte("# Zoo\n\n" +
		"1. [[#Pattern 1 — Semantic Identity]]\n" +
		"2. [[#9.2 Kernel K0: canonical identity]]\n\n" +
		"## Pattern 1 — Semantic Identity\n\n" +
		"## 9.2 Kernel K0: canonical identity\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// The point of resolving against the rendered HTML rather than slugifying:
	// these two ids differ from Slugify's answer.
	for _, want := range []string{
		`href="#pattern-1--semantic-identity"`,
		`href="#92-kernel-k0-canonical-identity"`,
		`>Pattern 1 — Semantic Identity</a>`,
		`>9.2 Kernel K0: canonical identity</a>`,
	} {
		if !contains(parsed.HTML, want) {
			t.Errorf("expected %s in HTML, got: %s", want, parsed.HTML)
		}
	}
	if contains(parsed.HTML, `href="/note/#`) {
		t.Fatalf("self link still routed through /note/: %s", parsed.HTML)
	}
	if contains(parsed.HTML, `href="#9-2-kernel-k0-canonical-identity"`) {
		t.Fatalf("fragment was slugified instead of read back from the heading: %s", parsed.HTML)
	}
	// A same-note anchor is not a link to a note, so it must stay out of the
	// backlink graph.
	if len(parsed.WikiLinks) != 0 {
		t.Fatalf("self heading links leaked into WikiLinks: %#v", parsed.WikiLinks)
	}
}

// TestSelfHeadingLinkMatchingRules covers how a target is matched against the
// rendered headings: case- and whitespace-insensitively, first-heading-wins on
// duplicates (as Obsidian does), with the already-slugified form accepted as a
// fallback and no match at all rendered visibly broken.
func TestSelfHeadingLinkMatchingRules(t *testing.T) {
	src := []byte("# T\n\n" +
		"- case: [[#SOME   Heading]]\n" +
		"- id form: [[#some-heading]]\n" +
		"- alias: [[#Some Heading|call it that]]\n" +
		"- dupe: [[#Notes]]\n" +
		"- missing: [[#nowhere]]\n\n" +
		"## Some Heading\n\n## Notes\n\n## Notes\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := strings.Count(parsed.HTML, `href="#some-heading"`); got != 3 {
		t.Errorf("case, id-form and aliased links should all reach #some-heading, got %d: %s", got, parsed.HTML)
	}
	if !contains(parsed.HTML, `>call it that</a>`) {
		t.Errorf("alias should win over the heading text: %s", parsed.HTML)
	}
	// goldmark emits "notes" and "notes-1"; the link takes the first.
	if !contains(parsed.HTML, `href="#notes"`) || contains(parsed.HTML, `href="#notes-1"`) {
		t.Errorf("duplicate headings: first should win, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `href="#unresolved-nowhere" class="wiki-link wiki-link-self broken"`) {
		t.Errorf("missing heading should render visibly broken: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `>nowhere</a>`) {
		t.Errorf("a broken self link must still show its text: %s", parsed.HTML)
	}
}

// TestSelfHeadingLinkDegenerateFormsAreLeftAlone pins that a wiki link with
// neither a target nor a heading is passed through as source text rather than
// turned into an empty anchor.
func TestSelfHeadingLinkDegenerateFormsAreLeftAlone(t *testing.T) {
	parsed, err := Parse([]byte("# T\n\nliteral [[#]] here\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !contains(parsed.HTML, "[[#]]") {
		t.Fatalf("degenerate [[#]] should survive as text, got: %s", parsed.HTML)
	}
	if contains(parsed.HTML, "wiki-link") {
		t.Fatalf("degenerate [[#]] should not become a link, got: %s", parsed.HTML)
	}
}

// TestResolveWikiLinkHeadings covers the fragment rewrite in isolation: the
// provisional slugified fragment wikiLinkHTML wrote is replaced with whatever
// the resolver says the target note actually rendered, and dropped when the
// resolver declines rather than left pointing at an id known not to exist.
func TestResolveWikiLinkHeadings(t *testing.T) {
	html := `<p>` +
		`<a href="/note/other#9-2-kernel-k0" class="wiki-link" data-target="other" data-raw="Other" data-heading="9.2 Kernel K0" data-alias="">Other</a>` +
		`<a href="/note/gone#missing" class="wiki-link" data-target="gone" data-raw="Gone" data-heading="missing" data-alias="">Gone</a>` +
		`<a href="/note/plain" class="wiki-link" data-target="plain" data-raw="Plain" data-heading="" data-alias="">Plain</a>` +
		`</p>`

	got := ResolveWikiLinkHeadings(html, func(slug, heading string) (string, bool) {
		if slug == "other" && heading == "9.2 Kernel K0" {
			return "92-kernel-k0", true
		}
		return "", false
	})

	if !contains(got, `href="/note/other#92-kernel-k0"`) {
		t.Errorf("fragment not replaced with the real id: %s", got)
	}
	if !contains(got, `href="/note/gone"`) || contains(got, `href="/note/gone#missing"`) {
		t.Errorf("unresolvable fragment should be dropped, not kept: %s", got)
	}
	if !contains(got, `href="/note/plain"`) {
		t.Errorf("link without a heading should be untouched: %s", got)
	}
	// The rewrite must preserve every other attribute, since later passes key
	// off data-raw and data-alias.
	if !contains(got, `data-raw="Other" data-heading="9.2 Kernel K0" data-alias=""`) {
		t.Errorf("attributes were not preserved: %s", got)
	}
	if !contains(got, `>Other</a>`) {
		t.Errorf("display text was not preserved: %s", got)
	}
}

// TestWikiLinkCarriesHeadingForLaterResolution pins that the parser hands the
// heading text on in an attribute. The fragment alone cannot be resolved later:
// slugify is lossy, so "#9-2-kernel-k0" no longer tells anyone that the heading
// was "9.2 Kernel K0".
func TestWikiLinkCarriesHeadingForLaterResolution(t *testing.T) {
	parsed, err := Parse([]byte("# T\n\nSee [[Other Note#9.2 Kernel K0: canonical identity]].\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !contains(parsed.HTML, `data-heading="9.2 Kernel K0: canonical identity"`) {
		t.Fatalf("heading text not carried on the anchor: %s", parsed.HTML)
	}

	// A link with no heading still carries the attribute, empty — the fragment
	// pass keys off a non-empty value, so it must be there to be absent.
	plain, err := Parse([]byte("# T\n\nSee [[Other Note]].\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !contains(plain.HTML, `data-heading=""`) {
		t.Fatalf("headingless link should carry an empty data-heading: %s", plain.HTML)
	}
}

// TestWikiLinkAttributesCarryTeXNotMathSentinels is the regression test for the
// second P2 on PR #19. Math is lifted out of the source before wiki links are
// replaced, so a heading like `$\sigma$ bound` reaches wikiLinkHTML as a
// sentinel. RestoreMath then rewrites every sentinel in the document — including
// ones sitting inside an attribute value, where the injected
// `<span class="math math-inline">` ends the attribute at its first quote and
// leaves malformed markup behind.
func TestWikiLinkAttributesCarryTeXNotMathSentinels(t *testing.T) {
	src := []byte("# T\n\n" +
		"- self: [[#The $\\sigma$ bound]]\n" +
		"- cross: [[Target#The $\\sigma$ bound]]\n" +
		"- alias: [[Target|see $\\pi$]]\n\n" +
		"## The $\\sigma$ bound\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// The giveaway for the old bug: markup inside an attribute value.
	if contains(parsed.HTML, `data-heading="The <span`) || contains(parsed.HTML, `data-alias="see <span`) {
		t.Fatalf("math markup was injected into an attribute: %s", parsed.HTML)
	}
	for _, want := range []string{
		`data-heading="The \sigma bound"`,
		`data-alias="see \pi"`,
	} {
		if !contains(parsed.HTML, want) {
			t.Errorf("expected %s in HTML, got: %s", want, parsed.HTML)
		}
	}
	// No sentinel may survive anywhere, attribute or not.
	if strings.ContainsRune(parsed.HTML, '') || strings.ContainsRune(parsed.HTML, '') {
		t.Fatalf("math sentinel leaked into the output: %q", parsed.HTML)
	}

	// The same-note link still has to reach the heading, which it can only do
	// once both sides are expressed in TeX rather than in sentinel indices —
	// the heading and the link naming it are separate math spans.
	if contains(parsed.HTML, `class="wiki-link wiki-link-self broken"`) {
		t.Fatalf("self link to a heading containing math did not resolve: %s", parsed.HTML)
	}
	// Display text is element content, where the math element belongs.
	if !contains(parsed.HTML, `>The <span class="math math-inline">\sigma</span> bound</a>`) {
		t.Errorf("display text should still render the math: %s", parsed.HTML)
	}

	// The heading and target also reach the note JSON, where a sentinel is
	// meaningless.
	for _, wl := range parsed.WikiLinks {
		if strings.ContainsRune(wl.Heading, '') || strings.ContainsRune(wl.Target, '') {
			t.Errorf("math sentinel leaked into WikiLinks: %#v", wl)
		}
	}
}

// TestWikiLinksInsideCodeStayLiteral is the regression test for PV-WIKICODE-022.
// The [[...]] substitution runs before goldmark so goldmark cannot parse the
// link text as Markdown, and that ordering used to rewrite code samples too:
// the anchor HTML was injected into the source and goldmark escaped it into the
// code block, so a note documenting the syntax showed raw markup where its
// author wrote a wiki link.
func TestWikiLinksInsideCodeStayLiteral(t *testing.T) {
	src := []byte("# Syntax\n\n" +
		"Inline: `[[Some Note]]` and `![[Diagram.png]]`.\n\n" +
		"```markdown\n" +
		"[[Target Note#Heading]]\n" +
		"[[Other|aliased]]\n" +
		"```\n\n" +
		"~~~\n[[Tilde Fenced]]\n~~~\n\n" +
		"A real link: [[Some Note]].\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, literal := range []string{
		"<code>[[Some Note]]</code>",
		"<code>![[Diagram.png]]</code>",
		"[[Target Note#Heading]]",
		"[[Other|aliased]]",
		"[[Tilde Fenced]]",
	} {
		if !contains(parsed.HTML, literal) {
			t.Errorf("expected %s to survive as text, got: %s", literal, parsed.HTML)
		}
	}
	// The giveaway for the old bug: escaped anchor markup inside a code element.
	if contains(parsed.HTML, "&lt;a href=") || contains(parsed.HTML, "&lt;img class=") {
		t.Fatalf("anchor markup was injected into a code block: %s", parsed.HTML)
	}

	// The link outside code still has to work.
	if !contains(parsed.HTML, `<a href="/note/some-note" class="wiki-link"`) {
		t.Fatalf("real link stopped resolving: %s", parsed.HTML)
	}

	// And a code sample must not give the note it names a backlink.
	if len(parsed.WikiLinks) != 1 || parsed.WikiLinks[0].Target != "Some Note" {
		t.Fatalf("WikiLinks = %#v, want only the real link", parsed.WikiLinks)
	}
}

// TestCodeRegionsBoundaries pins the CommonMark edge cases the scanners handle,
// since a region that ends too early re-exposes the code after it and one that
// ends too late swallows real links.
func TestCodeRegionsBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		literal []string // must survive as text
		linked  []string // must become anchors (data-raw values)
	}{
		{
			name:    "double backtick span containing a single backtick",
			src:     "Use ``[[A]] ` here`` then [[B]].\n",
			literal: []string{"[[A]]"},
			linked:  []string{"B"},
		},
		{
			name:    "info string on the opening fence only",
			src:     "```go\n[[A]]\n```\n\n[[B]]\n",
			literal: []string{"[[A]]"},
			linked:  []string{"B"},
		},
		{
			name:    "longer closing fence still closes",
			src:     "```\n[[A]]\n`````\n\n[[B]]\n",
			literal: []string{"[[A]]"},
			linked:  []string{"B"},
		},
		{
			name:    "indented three spaces still opens a fence",
			src:     "   ```\n   [[A]]\n   ```\n\n[[B]]\n",
			literal: []string{"[[A]]"},
			linked:  []string{"B"},
		},
		{
			name:    "unterminated fence runs to end of document",
			src:     "```\n[[A]]\n[[B]]\n",
			literal: []string{"[[A]]", "[[B]]"},
		},
		{
			name:   "backtick that opens nothing leaves the link alone",
			src:    "A stray ` tick and [[B]].\n",
			linked: []string{"B"},
		},
		{
			name:   "escaped backticks do not open a code span",
			src:    "A \\`[[A]]\\` then [[B]].\n",
			linked: []string{"A", "B"},
		},
		{
			name:    "even backslashes before a backtick still open a code span",
			src:     "A \\\\`[[C]]` then [[D]].\n",
			literal: []string{"[[C]]"},
			linked:  []string{"D"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed, err := Parse([]byte("# T\n\n" + c.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			for _, lit := range c.literal {
				if !contains(parsed.HTML, lit) {
					t.Errorf("%q should have stayed literal, got: %s", lit, parsed.HTML)
				}
			}
			for _, raw := range c.linked {
				if !contains(parsed.HTML, `data-raw="`+raw+`"`) {
					t.Errorf("[[%s]] should have become a link, got: %s", raw, parsed.HTML)
				}
			}
		})
	}
}

// TestFrontmatterBacktickDoesNotSwallowBodyLink pins the first PR #20 review
// finding: extractWikiLinks used to detect code regions over the whole source,
// so a backtick in a frontmatter scalar paired with a body backtick and
// classified the body link between them as code. replaceWikiLinks scans the
// body only, so the link still rendered as an anchor while being silently
// dropped from WikiLinks and the backlink graph. Both passes now detect code on
// the body only, so the link is both rendered and indexed.
func TestFrontmatterBacktickDoesNotSwallowBodyLink(t *testing.T) {
	src := []byte("---\n" +
		"title: \"a `tick\"\n" +
		"---\n" +
		"[[Body Link]] then `[[In Code]]` here.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The body link sits outside the code span, so it renders and is indexed.
	if !contains(parsed.HTML, `data-raw="Body Link"`) {
		t.Fatalf("body link should render, got: %s", parsed.HTML)
	}
	if len(parsed.WikiLinks) != 1 || parsed.WikiLinks[0].Target != "Body Link" {
		t.Fatalf("WikiLinks = %#v, want exactly [Body Link] (the frontmatter "+
			"backtick used to swallow it and leave [[In Code]] instead)", parsed.WikiLinks)
	}
	// The link inside the body code span stays literal and out of the graph,
	// proving the body code span itself still works after the frontmatter fix.
	if !contains(parsed.HTML, "<code>[[In Code]]</code>") {
		t.Errorf("in-code link should stay literal, got: %s", parsed.HTML)
	}
}

// TestFrontmatterWikiLinkStillIndexed guards the deliberate scope of the fix
// above: code-region detection now excludes frontmatter, but wiki-link
// extraction from frontmatter is unchanged (a separate, filed question — see
// PV-WIKICODE-022 task dmoh). A [[X]] in a frontmatter value is not rendered
// (frontmatter is not document body) yet still enters WikiLinks, so existing
// "related:" lists keep their backlinks.
func TestFrontmatterWikiLinkStillIndexed(t *testing.T) {
	src := []byte("---\n" +
		"related: \"[[Frontmatter Note]]\"\n" +
		"---\n" +
		"Body text with no link.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(parsed.WikiLinks) != 1 || parsed.WikiLinks[0].Target != "Frontmatter Note" {
		t.Fatalf("WikiLinks = %#v, want the frontmatter link to still be indexed",
			parsed.WikiLinks)
	}
	if contains(parsed.HTML, `data-raw="Frontmatter Note"`) {
		t.Fatalf("frontmatter link must not render as document body, got: %s",
			parsed.HTML)
	}
}

// TestEscapedBackticksKeepLinkInBacklinkGraph pins the second PR #20 review
// finding: a wiki link written between two escaped backticks (a backslash
// before each backtick) is, in CommonMark, a literal backtick, then the link,
// then a literal backtick — not a code span. The old codeRegions ignored the
// backslash and wrapped the link in a code span, so both passes skipped it:
// the page showed plain [[Target]] and the expected backlink was absent.
// codeRegions now consumes backslash escapes the way ScanMath does, so an
// escaped backtick no longer opens a code span.
func TestEscapedBackticksKeepLinkInBacklinkGraph(t *testing.T) {
	src := []byte("# T\n\nA \\`[[Target]]\\` link.\n")
	parsed, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !contains(parsed.HTML, `data-raw="Target"`) {
		t.Fatalf("escaped-backtick link should render, got: %s", parsed.HTML)
	}
	if contains(parsed.HTML, "[[Target]]") {
		t.Fatalf("link should have been converted, not left as [[Target]]: %s",
			parsed.HTML)
	}
	if len(parsed.WikiLinks) != 1 || parsed.WikiLinks[0].Target != "Target" {
		t.Fatalf("WikiLinks = %#v, want exactly [Target]", parsed.WikiLinks)
	}
}

// TestSplitSourceMatrix pins the single frontmatter boundary that every parser
// consumer now shares. The delimiter rule must mirror goldmark-meta exactly so
// that any preamble the metadata parser accepts is also the boundary the
// pre-passes protect. A four-dash (or one-/two-dash, or whitespace-wrapped)
// preamble that goldmark-meta parses as metadata must be split off here, not
// handed to the math or wiki pre-passes as Markdown body.
func TestSplitSourceMatrix(t *testing.T) {
	cases := []struct {
		name            string
		src             string
		wantFrontmatter bool   // hasFrontmatter()
		wantBody        string // body bytes (exact)
	}{
		{
			name:            "no frontmatter returns the source untouched",
			src:             "# Title\n\nBody.\n",
			wantFrontmatter: false,
			wantBody:        "# Title\n\nBody.\n",
		},
		{
			name:            "three dashes split",
			src:             "---\ntitle: x\n---\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "one dash splits (goldmark-meta accepts it)",
			src:             "-\ntitle: x\n-\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "two dashes split (goldmark-meta accepts it)",
			src:             "--\ntitle: x\n--\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "four dashes split (the metadata-mutation defect)",
			src:             "----\ntitle: x\n----\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "whitespace-wrapped delimiter splits",
			src:             "  ----  \ntitle: x\n \t----\t \nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "CRLF delimiters split",
			src:             "----\r\ntitle: x\r\n----\r\nBody\r\n",
			wantFrontmatter: true,
			wantBody:        "Body\r\n",
		},
		{
			name:            "dashes inside a quoted scalar do not close early",
			src:             "---\ntitle: \"before---after\"\n---\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "indented dash line inside a block scalar does not close",
			src:             "---\nsummary: |\n  a --- b\npublish: true\n---\nBody\n",
			wantFrontmatter: true,
			wantBody:        "Body\n",
		},
		{
			name:            "closing delimiter at EOF yields an empty body",
			src:             "---\ntitle: x\n---",
			wantFrontmatter: true,
			wantBody:        "",
		},
		{
			name:            "body at EOF without a trailing newline is exact",
			src:             "---\ntitle: x\n---\nBody",
			wantFrontmatter: true,
			wantBody:        "Body",
		},
		{
			name:            "unterminated opener leaves the whole source as body",
			src:             "---\ntitle: x\nno closing delimiter\n",
			wantFrontmatter: false,
			wantBody:        "---\ntitle: x\nno closing delimiter\n",
		},
		{
			name:            "a thematic break after a heading is not frontmatter",
			src:             "# Title\n\n---\n\nAfter the rule.\n",
			wantFrontmatter: false,
			wantBody:        "# Title\n\n---\n\nAfter the rule.\n",
		},
		{
			name:            "an opener with trailing non-dash content is not frontmatter",
			src:             "---yaml\ntitle: x\n---\nBody\n",
			wantFrontmatter: false,
			wantBody:        "---yaml\ntitle: x\n---\nBody\n",
		},
		{
			name:            "empty input is not frontmatter",
			src:             "",
			wantFrontmatter: false,
			wantBody:        "",
		},
		{
			name:            "a single dash line with no newline is not an opener",
			src:             "---",
			wantFrontmatter: false,
			wantBody:        "---",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parts := splitSource([]byte(c.src))
			if parts.hasFrontmatter() != c.wantFrontmatter {
				t.Errorf("hasFrontmatter = %v, want %v", parts.hasFrontmatter(), c.wantFrontmatter)
			}
			if got := string(parts.body); got != c.wantBody {
				t.Errorf("body = %q, want %q", got, c.wantBody)
			}
			// The two slices must reconstruct the source byte-for-byte, and the
			// body offset must equal the frontmatter length: extraction uses it
			// to convert whole-source regex offsets into body-relative ones.
			reconstructed := append(append([]byte{}, parts.frontmatter...), parts.body...)
			if !bytes.Equal(reconstructed, []byte(c.src)) {
				t.Errorf("frontmatter+body does not reconstruct the source:\n got  %q\n want %q", reconstructed, c.src)
			}
			if parts.bodyOffset != len(parts.frontmatter) {
				t.Errorf("bodyOffset = %d, len(frontmatter) = %d", parts.bodyOffset, len(parts.frontmatter))
			}
		})
	}
}
