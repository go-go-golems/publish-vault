---
Title: Strip .md extension from wiki-link targets so links resolve to the right note
Ticket: PV-WIKILINK-021
Status: active
Topics:
    - wiki-link
    - parser
    - vault
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-10T06:43:33.634276784-04:00
WhatFor: ""
WhenToUse: ""
---

# Strip .md extension from wiki-link targets so links resolve to the right note

## Overview

Wiki-link targets written with a trailing `.md` — `[[Folder/Note.md#Heading]]`,
the form Obsidian emits under "absolute path in vault" and the form LLM-written
notes produce — did not resolve. `slugify` turns the dot into a hyphen rather
than dropping it, while every slug in the index is built from an extension-less
path, so the target asked for a `…-md` key. Usually that missed, rendering as
`#unresolved-…` with the heading fragment and the target's title both lost;
in a vault that also held a note named "… md" it silently hit the *wrong note*.

Fixed by stripping one trailing `.md` in `parseWikiLinkInner` — the single point
every consumer of a target passes through — plus the same strip in
`ResolveWikiLink` and in the parallel TypeScript resolver `staticVault.ts`.

Validated on the reported note, `Transcripts/Research/09 - RAG-MATHS Pattern
Zoo.md` in the go-go-parc vault: **92 unresolved link occurrences before, 0
after**. Its outgoing-link graph also stopped double-counting (46 → 40 distinct
links), because `[[X.md]]` and `[[X]]` were separate entries before.

Design: [design/01-stripping-the-md-extension-from-wiki-link-targets.md](./design/01-stripping-the-md-extension-from-wiki-link-targets.md).
Diary: [reference/01-diary.md](./reference/01-diary.md).

Two follow-ups are open in tasks.md and were deliberately **not** fixed here:
`[[#Heading]]` same-note links (a different bug in the same function), and the
absence of any unit-test runner under `web/`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- wiki-link
- parser
- vault

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
