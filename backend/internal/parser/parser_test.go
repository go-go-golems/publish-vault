package parser

import (
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

func TestWikiLinkDataRaw(t *testing.T) {
	parsed, err := Parse([]byte("# Test\n\nSee [[Tribal/App-Auth]] and [[Fundamentals/Access|Custom Alias]].\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !contains(parsed.HTML, `data-raw="Tribal/App-Auth"`) {
		t.Fatalf("HTML should contain data-raw for first link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, `data-raw="Custom Alias"`) {
		t.Fatalf("HTML should contain data-raw for aliased link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, ">Tribal/App-Auth</a>") {
		t.Fatalf("HTML should display raw target for first link, got: %s", parsed.HTML)
	}
	if !contains(parsed.HTML, ">Custom Alias</a>") {
		t.Fatalf("HTML should display alias for second link, got: %s", parsed.HTML)
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
