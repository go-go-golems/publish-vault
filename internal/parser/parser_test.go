package parser

import (
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
