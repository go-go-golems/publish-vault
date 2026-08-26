---
Title: Date-aware advanced search implementation
Ticket: PV-SEARCH-028
Status: active
Topics:
    - search
    - backend
    - frontend
    - architecture
    - regression
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/vault/vault.go
      Note: Note struct, loadNote, SearchDocument, ForEachSearchDocument
    - Path: repo://pkg/search/search.go
      Note: Bleve mapping, noteDoc, query construction, result extraction
    - Path: repo://pkg/api/api.go
      Note: HTTP search handler and future advanced endpoint
    - Path: repo://internal/parser/parser.go
      Note: Frontmatter normalization and scalar types
    - Path: repo://web/src/vault/staticVault.ts
      Note: Static parseFrontmatter/serializeFrontmatter and staticSearch
    - Path: repo://web/src/components/pages/SearchPage/SearchPage.tsx
      Note: Canonical URL and advanced UI target
    - Path: repo://ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/design-doc/01-date-aware-advanced-search-architecture-and-implementation-guide.md
      Note: Merged design contract and phased plan
ExternalSources:
    - https://pkg.go.dev/github.com/blevesearch/bleve/v2
Summary: Implements the merged PV-SEARCH-027 design: canonical authored note dates, typed advanced-search request, Bleve date/tag/path fields, advanced HTTP endpoint, canonical URL codec, static-mode parity, accessible filter UI, and rollout validation.
LastUpdated: 2026-08-26T01:46:00-04:00
WhatFor: Shipping the date-aware advanced search feature end to end from Go domain types through Bleve, HTTP, static mode, URL state, and React controls.
WhenToUse: Read before changing vault note dates, search index fields, query construction, search API routes, RTK Query cache arguments, SearchPage state, or filter controls.
---

# Date-aware advanced search implementation

## Overview

This ticket implements the design merged in PV-SEARCH-027 (PR #26, merge
`708685ae`). The work proceeds in six phases (A–F), each a reviewable commit
slice, with a strict diary, phase start/done slips, and gates between phases.

Canonical contracts (from the design):

- Search dates are authored frontmatter metadata, never filesystem `ModTime`
  and never the static `new Date()` fallback.
- Created aliases: `created`, then `date`. Updated aliases: `updated`,
  `modified`, then `last_updated`. Strict `YYYY-MM-DD` or RFC3339 only.
- Display precedence: updated over created, otherwise absent.
- Static mode parses frontmatter with `js-yaml` `JSON_SCHEMA` so unquoted
  RFC3339 scalars stay strings and survive `buildVault`.
- Legacy `#tag`/`tag:` inclusion uses the deployed dynamic contract (prefix for
  queries of at most three characters, fuzziness one for longer terms); static
  exact-only discovery migrates to that contract. Structured `tag=` filters
  remain exact.
- New `/api/search/advanced` returns a typed envelope; legacy `/api/search`
  remains a thin adapter during a compatibility window.
- The browser URL is the committed search request state.

## Key Links

- **Design**: `ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/design-doc/01-date-aware-advanced-search-architecture-and-implementation-guide.md`
- **Implementation plan**: `design-doc/01-implementation-plan-and-design-reference.md`
- **Diary**: `reference/01-investigation-diary.md`
- **Tasks**: `tasks.md`
- **Changelog**: `changelog.md`

## Status

Current status: **active — Phase C complete, Phase D next**

## Topics

- search
- backend
- frontend
- architecture
- regression

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
