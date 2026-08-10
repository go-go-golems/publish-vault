---
Title: MathJax support for LaTeX math in published vault notes
Ticket: PV-MATHJAX-018
Status: active
Topics:
    - mathjax
    - math
    - latex
    - parser
    - html-rendering
    - frontend
    - ssr
    - bundle
    - obsidian-vault
    - retro-obsidian-publish
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Add LaTeX math rendering to publish-vault: a Go-side pre-goldmark scanner that protects $…$ / $$…$$ / \\(…\\) / \\[…\\] regions as inert HTML placeholders, and a browser-side MathJax (TeX→SVG) enhancement pass mirroring the existing Mermaid pipeline."
LastUpdated: 2026-08-09
WhatFor: "Tracking the MathJax feature from design through implementation."
WhenToUse: "Start here for anything math-related in publish-vault."
---

# MathJax support for LaTeX math in published vault notes

## Overview

Obsidian notes routinely contain LaTeX math written as `$inline$` and
`$$display$$`. Today `publish-vault` has no math support at all, and worse,
Markdown actively destroys TeX on the way through: underscores become `<em>`,
`\\` is eaten, `&` is escaped, and `WithHardWraps()` interleaves `<br/>` into
multi-line `align` environments.

This ticket adds end-to-end math support in two halves, matching the existing
split of responsibilities in the codebase:

- **Go side** — a new `internal/parser/math.go` scans the raw Markdown *before*
  goldmark runs and replaces each math region with an inert
  `<span class="math math-inline">` / `<div class="math math-display">`
  placeholder carrying the verbatim TeX as escaped text content. This is the
  same pre-pass idiom `replaceWikiLinks` already uses. The scanner implements
  Pandoc's whitespace/digit rules so prose about prices (`$30 and $25`) is not
  mistaken for math, and it skips code spans, fenced blocks, indented code, and
  frontmatter.
- **Browser side** — a new `enhanceMath` in the post-hydration enhancement
  pipeline dynamically imports MathJax 4 (`@mathjax/src`, TeX input → SVG
  output) and swaps each placeholder's TeX for rendered SVG. It mirrors
  `enhanceMermaid` exactly: cheap bail-out when a note has no math, lazy chunk,
  cancellable, idempotent, and it leaves the TeX source visible on failure.

See the [design doc](./design/01-mathjax-support-analysis-design-and-implementation-guide.md)
for the full analysis, decision record, pseudocode, diagrams, and the five-phase
implementation plan. The [diary](./reference/01-diary.md) records the work as it
happens.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- mathjax
- math
- latex
- parser
- html-rendering
- frontend
- ssr
- bundle
- obsidian-vault
- retro-obsidian-publish

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
