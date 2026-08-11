---
Title: Wiki links inside code spans and fences render as escaped anchor HTML
Ticket: PV-WIKICODE-022
Status: active
Topics:
    - wiki-link
    - parser
    - html-rendering
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-11T12:28:37.245495355-04:00
WhatFor: ""
WhenToUse: ""
---

# Wiki links inside code spans and fences render as escaped anchor HTML

## Overview

The `[[...]]` substitution runs before goldmark, because goldmark would
otherwise parse the link text as Markdown. That ordering also rewrote **code
samples**: anchor HTML was injected into the source, goldmark escaped it into
the code block, and a note documenting the syntax rendered

```
<code>&lt;a href=&quot;/note/some-note&quot; class=&quot;wiki-link&quot;...</code>
```

where its author wrote `[[Some Note]]`. The same substitution put the named note
into `WikiLinks`, so a code sample gave it a backlink from a note that never
linked to it.

`extractWikiLinks` and `replaceWikiLinks` now skip matches inside a code span or
a fenced block, reusing the scanners `ScanMath` already applies so the two
pre-passes agree about what counts as code. Indented four-space blocks stay
excluded, for the reason documented on `ScanMath`.

Measured on the go-go-parc vault (1790 notes): **341 injected-markup occurrences
across 69 notes → 0**. The audit distinguishes injected markup from HTML a note
quotes on purpose by parsing each note twice, once with every `[[` neutralised.

Found while writing a report about PV-WIKILINK-021 for that vault — the report
could not be published without it. The defect itself is older and unrelated to
that ticket's changes.

Diary: [reference/01-diary.md](./reference/01-diary.md).

Branched from `task/publish-vault-mathjax` (PR #19), which changed the
signatures of both functions edited here; a branch off `main` would conflict on
exactly those lines.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- wiki-link
- parser
- html-rendering

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
