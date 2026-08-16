---
Title: Implementation Diary
Ticket: PV-FRONTMATTER-024
Status: active
Topics:
    - parser
    - frontmatter
    - regression
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: primary evidence source
    - Path: repo://ttmp/2026/08/16/PV-FRONTMATTER-024--unify-markdown-frontmatter-boundaries-across-parser-pre-passes/scripts/01-goldmark-frontmatter-contract/main.go
      Note: contract probe
ExternalSources: []
Summary: Investigation and design for unifying publish-vault frontmatter boundaries around one goldmark-meta-compatible source split.
LastUpdated: 2026-08-16T22:45:00Z
WhatFor: Handoff context for the colleague implementing PV-FRONTMATTER-024.
WhenToUse: Read before implementing or reviewing the frontmatter boundary fix.
---


# Implementation Diary

## Goal

Produce a colleague-ready implementation guide for the confirmed frontmatter boundary defect so a second engineer can implement it without rediscovering the diagnosis.

## Step 1: Confirm the defect and pin the goldmark contract

The architecture review (PV-MARKDOWN-023) had already demonstrated metadata mutation for four-dash frontmatter. This step verified the exact goldmark-meta v1.1.0 separator contract and confirmed which delimiter forms the configured `Parse` pipeline actually accepts as frontmatter.

### What I did

- Read `goldmark-meta@v1.1.0/meta.go:95–132`: `isSeparator` trims surrounding whitespace and requires one or more `-` bytes; opening is restricted to line zero; closing rejects blank lines.
- Read the duplicate splitters in `internal/parser/parser.go`: `splitFrontmatter` (exact `---`) vs `isFrontmatterDelimiter`/`stripFrontmatter` (any dash run).
- Created and ran `scripts/01-goldmark-frontmatter-contract`, which calls the real `Parse` pipeline against one-, two-, three-, four-dash, whitespace-wrapped, and CRLF delimiters.
- Confirmed all six forms parse as metadata in the configured engine.

### What worked

The probe eliminated the only uncertainty in the implementation guide. One- and two-dash delimiters win goldmark parser priority over Markdown list/thematic-break parsing for these inputs, so the regression matrix can include them without reservation.

### What I learned

`isFrontmatterDelimiter` already mirrors goldmark-meta exactly. The defect is purely the duplicate `splitFrontmatter`, not a wrong predicate. The fix is to stop having two splitters, not to invent a third rule.

### What was tricky

The temptation was to fold this into the larger typed AST/IR architecture. Keeping the fix narrow — one unexported structured split, migrate consumers, delete the duplicate — is what makes it safe to hand off and review independently.

### Code review instructions

```bash
go run ./ttmp/2026/08/16/PV-FRONTMATTER-024--*/scripts/01-goldmark-frontmatter-contract
```

All six cases should report `marker="preserved"`.

### Technical details

Probe output:

```text
one dash             title="Boundary" marker="preserved" html="<p>Body</p>\n"
two dashes           title="Boundary" marker="preserved" html="<p>Body</p>\n"
three dashes         title="Boundary" marker="preserved" html="<p>Body</p>\n"
four dashes          title="Boundary" marker="preserved" html="<p>Body</p>\n"
whitespace wrapped   title="Boundary" marker="preserved" html="<p>Body</p>\n"
four dashes CRLF     title="Boundary" marker="preserved" html="<p>Body</p>\n"
```
