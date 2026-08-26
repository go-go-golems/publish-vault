---
Title: Implementation plan and design reference
Ticket: PV-SEARCH-028
Status: active
Topics:
    - search
    - backend
    - frontend
    - architecture
    - regression
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/api/api.go
      Note: Advanced endpoint and legacy adapter
    - Path: repo://pkg/api/search_request.go
      Note: Advanced endpoint param parser and handler
    - Path: repo://pkg/search/request.go
      Note: Typed request and normalization
    - Path: repo://pkg/search/search.go
      Note: |-
        Bleve mapping, noteDoc, query builder, result extraction
        Bleve mapping and query builder
        Bleve mapping and SearchAdvanced
    - Path: repo://pkg/vault/date.go
      Note: Canonical authored-date resolver
    - Path: repo://pkg/vault/vault.go
      Note: Note/loadNote/SearchDocument owners
    - Path: repo://testdata/search-date-cases.json
      Note: Shared cross-language fixture
    - Path: repo://ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/design-doc/01-date-aware-advanced-search-architecture-and-implementation-guide.md
      Note: Merged primary design contract
    - Path: repo://web/src/components/molecules/AdvancedSearchPanel/AdvancedSearchPanel.tsx
      Note: Accessible filter panel
    - Path: repo://web/src/components/pages/SearchPage/SearchPage.tsx
      Note: URL-driven advanced search page
    - Path: repo://web/src/search/noteDate.ts
      Note: TS date resolver mirroring Go
    - Path: repo://web/src/search/searchParams.ts
      Note: URL codec and normalization
    - Path: repo://web/src/vault/staticVault.ts
      Note: |-
        Static parity and JSON_SCHEMA parsing
        staticSearchAdvanced parity
ExternalSources:
    - https://pkg.go.dev/github.com/blevesearch/bleve/v2
Summary: Phased implementation map for PV-SEARCH-028, anchoring each phase to the merged PV-SEARCH-027 design contracts, files, gates, and review risks.
LastUpdated: 2026-08-26T01:46:00-04:00
WhatFor: Knowing which files change in each phase and what gate must pass before the next phase starts.
WhenToUse: Read before starting or reviewing any PV-SEARCH-028 phase.
---







# Implementation plan and design reference

This document is the working map for PV-SEARCH-028. The authoritative contract
is the merged PV-SEARCH-027 design package; this file tracks phase scope,
files, and gates only.

## Phase A: shared date fixtures and Go/TS canonical date domain

Files:

- add `pkg/vault/date.go` and `date_test.go`;
- update `pkg/vault/vault.go` `Note`, `loadNote`, `SearchDocument`,
  `SearchDocument()`, `searchDocumentBytes` accounting;
- add `web/src/search/noteDate.ts` and tests;
- add shared JSON fixture under `testdata/search-date-cases.json` consumable by
  both Go and web tests;
- update `web/src/vault/staticVault.ts` `parseFrontmatter` to use `JSON_SCHEMA`
  and remove the lossy `Date` path from authored-date handling.

Gate:

```text
all date fixture cases pass in Go and TypeScript
no ModTime/current-date fallback in search date path
existing note JSON compatibility tests pass
quoted/unquoted RFC3339 static build fixture preserves instant and precision
```

## Phase B: typed search request and Bleve mapping

Files:

- add `pkg/search/request.go`, `request_test.go`;
- update `pkg/search/search.go` mapping, `noteDoc`, `toNoteDoc`,
  `searchDocumentBytes`, result extraction;
- update `pkg/search/search_test.go`;
- update `pkg/server/runtime_test.go` if stored fields affect reopen/equivalence.

Gate:

```text
legacy search result equivalence passes
exact tag/path/date contract tests pass
persistent reopen/deletion/rollback tests pass
missing date sort is explicit
```

## Phase C: advanced HTTP API

Files:

- update `pkg/api/api.go`;
- add `pkg/api/search_request.go`, tests;
- update `pkg/api/api_test.go`;
- update README/API help.

Gate:

```text
HTTP contract table has one test per validation rule
legacy endpoint remains covered
no raw query values added to logs/metrics
```

## Phase D: shared TS types, URL codec, static mode

Files:

- update `web/src/types/index.ts`;
- add `web/src/search/searchParams.ts` and tests;
- update `web/src/store/vaultApi.ts`;
- update `web/src/vault/staticVault.ts` and tests.

Gate:

```text
URL round-trip/property cases pass
backend and static expected ID/order fixtures pass
legacy #go/fuzzy expected-ID fixtures match in backend and static modes
invalid URL remains visible to UI
filter-only request executes
```

## Phase E: advanced-search UI

Files:

- update `web/src/components/pages/SearchPage/SearchPage.tsx`;
- add filter panel component using existing `web/src/components/ui/dialog.tsx`;
- update `web/src/components/molecules/NoteCard/NoteCard.tsx` for date display;
- add/update Storybook stories and Vitest tests.

Gate:

```text
accessible responsive controls pass
canonical URL drives the page
invalid filters render reset action
```

## Phase F: full validation, measurement, and rollout

Files:

- `make ci-check`, race tests, Vitest, client/SSR, Storybook;
- Docker/Compose smoke at the PV-MEM-002 budgets;
- index-size and memory measurement;
- README and help updates;
- PR, review, merge, image bump, GitOps rollout.

Gate:

```text
all repository gates green
memory/index/query evidence supports deployment
rollback path verified
```

## Review risks carried from the design

- Using `time.Time` alone loses date-only precision; retain precision separately.
- The default `js-yaml` schema creates `Date` objects and the serializer
  truncates them; configure `JSON_SCHEMA` before parsing.
- Static exact-only legacy tag matching must migrate to the pinned dynamic
  prefix/fuzziness contract with shared expected-ID fixtures.
- Converting timestamps to browser-local dates during SSR can shift the
  displayed calendar day and cause hydration mismatch.
- Falling through from invalid high-priority aliases hides author errors.
