---
Title: Investigation diary
Ticket: PV-SEARCH-028
Status: active
Topics:
    - search
    - backend
    - frontend
    - diary
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological implementation diary for the date-aware advanced search feature, following the diary skill format.
LastUpdated: 2026-08-26T01:46:00-04:00
WhatFor: Recording each implementation step, failure, and review instruction so work can be resumed and reviewed.
WhenToUse: Read before resuming PV-SEARCH-028 or reviewing a phase.
---

# Diary

## Goal

Implement the merged PV-SEARCH-027 design end to end across the Go vault
layer, persistent Bleve index, HTTP API, static mode, URL state, and React
controls, with strict diary, phase slips, and gates.

## Step 0: ticket setup and master plan

The merged design (PR #26, merge `708685ae`) is the contract. I created
implementation ticket PV-SEARCH-028 on branch `task/pv-search-028-advanced-search-impl`
from merged `origin/main`, added six phase tasks (A–F), wrote the index and
implementation-plan design doc, and created this diary. I confirmed the Go
parser already returns frontmatter date values as strings (probe result),
so only the static TypeScript side needs the `JSON_SCHEMA` change.

### Prompt Context

**User prompt (verbatim):** "Ok, implement. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill) Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done."

**Assistant interpretation:** Implement the PV-SEARCH-027 design as ticket PV-SEARCH-028, committing at intervals, keeping a strict diary, and printing a master plan slip plus a start and done slip for each phase.

**Inferred user intent:** Ship the feature with the same evidence/slip discipline used for the design and PV-MEM-002 work.

### What I did

- Created branch `task/pv-search-028-advanced-search-impl` from `origin/main` (`708685a`).
- Created ticket PV-SEARCH-028 with six phase tasks.
- Wrote index, implementation-plan design doc, and diary.
- Re-read `vault.go`, `search.go`, `parser.go`, and the design probes.

### Why

A separate implementation ticket keeps the merged design package immutable while tracking implementation progress, commits, and gates.

### What worked

- The Go parser probe confirmed frontmatter dates are already strings, simplifying Phase A.
- The existing `searchDocumentBytes` is a single bounded accounting point to extend.

### What didn't work

N/A yet.

### What I learned

- The static-side `JSON_SCHEMA` change is the only parsing-boundary correction needed; Go needs no schema change.

### What was tricky to build

N/A yet.

### What warrants a second pair of eyes

N/A yet.

### What should be done in the future

N/A yet.

### Code review instructions

- Confirm the branch base is the merged design PR (`708685a`).
- Confirm six phase tasks exist and the index links the merged design.

### Technical details

```text
branch base: 708685a (merged PV-SEARCH-027)
ticket: PV-SEARCH-028
phases: A B C D E F
go parser date type: string (no schema change)
static schema change: JSON_SCHEMA required
```

## Step 1: Phase A — shared date fixtures and Go/TS canonical date domain

Phase A establishes the authored-date contract before any indexing, API, or UI
work. The Go vault layer now resolves created/updated/display dates from strict
frontmatter aliases, exposes a content-free invalid-date counter, and streams
the resolved instants through `SearchDocument`. A shared JSON fixture drives both
Go and TypeScript resolver tests so the two runtimes cannot silently diverge.
The static vault now parses frontmatter with `js-yaml` `JSON_SCHEMA` so
unquoted RFC3339 scalars survive as strings instead of being truncated to
`YYYY-MM-DD` by the serializer.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Implement Phase A from the merged design: canonical date domain, shared fixtures, and the static scalar-preservation fix, with tests and a diary.

**Inferred user intent:** Build the verified foundation that Phase B (Bleve mapping) and Phase D (static parity) depend on.

**Commit (code):** pending Phase A commit

### What I did

- Added `pkg/vault/date.go` with `NoteDate`, `NoteDates`, `NoteDateKind`,
  `DatePrecision`, `InvalidDateReason`, `ResolveNoteDates`, and helpers
  (`noteDateInstant`, `noteDateDisplayInstant`, `noteDateKindString`,
  `dateWarningKey`).
- Added `pkg/vault/date_test.go` (fixture-driven + unit tests) and
  `pkg/vault/date_integration_test.go` (note population, SearchDocument dates,
  invalid-date counter).
- Added the shared fixture `testdata/search-date-cases.json` (13 cases).
- Updated `pkg/vault/vault.go`: added `Note.Dates`, `SearchDocument` date
  fields, `loadNote` returns date warnings, both callers aggregate into
  `v.invalidDateCounts`, `InvalidDateCounts()` method, and a load log line.
- Added `web/src/search/noteDate.ts` and `noteDate.test.ts` mirroring the Go
  contract and consuming the same fixture.
- Switched `web/src/vault/staticVault.ts` `parseFrontmatter` to `JSON_SCHEMA`,
  exported `parseFrontmatter`/`serializeFrontmatter`, and added
  `staticVault.frontmatter.test.ts` proving quoted/unquoted scalars survive.

### Why

- Dates must be authored metadata with deterministic precision before any
  index/API/UI consumes them.
- The default js-yaml schema was the P1 review finding: it created `Date`
  objects that the serializer truncated, losing RFC3339 instants.
- A shared fixture is the only way to prove Go and TypeScript agree without
  duplicating prose assumptions.

### What worked

- The Go parser already returns frontmatter dates as strings, so no Go schema
  change was needed; only the static side needed `JSON_SCHEMA`.
- `toUTCSecond` formats JS Dates at second precision to match Go's RFC3339
  output exactly (`2024-01-15T18:45:00Z`, no milliseconds).
- All 13 fixture cases pass in both Go and TypeScript.

### What didn't work

- First build failed: `parseNoteDate` returned a `NoteDate` value but
  `resolveConcept` returned `*NoteDate`; fixed by returning `&nd`.
- The first fixture test compared warnings with `reflect.DeepEqual`, which
  distinguishes nil from non-nil empty slices; replaced with length/element
  comparison and removed the now-unused `reflect` import.
- `golangci-lint nonamedreturns` flagged `lookupAlias`'s named returns;
  switched to unnamed returns.

### What I learned

- `omitempty` on a struct value field never omits the struct; `NoteDates` inner
  pointers carry `omitempty` so absent dates serialize cleanly.
- JS `new Date("01/15/2024")` is valid, so the TS resolver must use a strict
  RFC3339 regex rather than relying on Date parsing to reject non-RFC3339.
- `loadNote` is called outside the lock by `ReloadNote`, so date-warning
  aggregation must happen in the callers under their locks, not inside
  `loadNote`.

### What was tricky to build

The no-fallthrough rule interacts with alias precedence: when the
highest-precedence alias exists but is invalid, the resolver returns nil and a
warning rather than trying the next alias. This makes an author mistake visible
instead of masking it with a stale lower-priority value, but it means a valid
`date` is ignored when an invalid `created` is present. The fixture case
`invalid-created-no-fallthrough` pins this behavior in both runtimes.

The timestamp-precision mismatch was the other sharp edge: Go's `time.RFC3339`
has second precision while JS `toISOString()` includes milliseconds. Without a
shared `toUTCSecond` formatter the two runtimes would emit different API values
for the same input.

### What warrants a second pair of eyes

- Confirm `JSON_SCHEMA` preserves all scalar types the static vault relies on
  (it does for strings, numbers, booleans, arrays, objects per the added test).
- Confirm the invalid-date counter is content-free (it keys only on
  `concept:reason`, never key spelling or value).
- Confirm `Note.Dates` JSON addition to `/api/notes/{slug}` is acceptable as a
  backward-compatible field.

### What should be done in the future

- Phase B should store `CreatedAt`/`UpdatedAt`/`DisplayAt`/`DateKind` in the
  Bleve `noteDoc` and add datetime/keyword mappings.
- Phase D should put resolved `NoteDates` on the static `Note` and remove the
  `new Date()` modTime fallback from the authored-date path.

### Code review instructions

- Start at `pkg/vault/date.go` and `web/src/search/noteDate.ts`.
- Run `GOWORK=off go test ./pkg/vault/` and `pnpm --dir web vitest run`.
- Inspect `loadNote` and both callers for the warning aggregation.
- Verify the shared fixture path works from both package dirs and web.

### Technical details

```text
fixture cases: 13
Go tests: pkg/vault pass (race + count=1)
TS tests: 61/61 (17 noteDate + 6 static frontmatter added)
lint: 0 issues
gosec: 0 issues
static schema: JSON_SCHEMA
date precision: date | timestamp (separate from time.Time)
display precedence: updated > created > absent
invalid-date counter: content-free, keyed concept:reason
```

## Step 2: Phase B — typed request, Bleve mapping, and query builder

Phase B makes the index understand dates, exact tags, paths, and sorting. The
Go search package now has a typed `SearchRequest`/`SearchResponse`, a normalizer
with stable field errors, a Bleve mapping with datetime and keyword fields, a
compound query builder, deterministic sorts, and date-aware result extraction.
The legacy `Search`/`searchByTag` paths were refactored onto the same query and
extraction helpers so there is one search implementation, not two.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Implement Phase B: typed request model, Bleve field mapping, compound query construction, sorting, pagination, and result extraction with dates and path.

**Inferred user intent:** Make the index queryable by structured filters so Phase C (HTTP) and Phase D (static) can build on one typed contract.

**Commit (code):** pending Phase B commit

### What I did

- Added `pkg/search/request.go` with `SearchRequest`, `SearchResponse`,
  `SearchResultDate`, `DateOnly`, `TagMode`/`DateField`/`SearchSort` enums,
  `NormalizeSearchRequest` with stable field errors, and `Effective()`.
- Extended `vault.SearchDocument` with `Path` and `DatePrecision` and populated
  them in `SearchDocument()`.
- Extended `noteDoc` with `tags_kw`, `path`, `path_kw`, `created_at`,
  `updated_at`, `display_at`, `date_kind`, `date_precision`; `toNoteDoc`
  lowercases tags and the path for exact/prefix filtering.
- Extended `buildMapping` with keyword (`tags_kw`, `path`, `path_kw`,
  `date_kind`, `date_precision`) and datetime (`created_at`, `updated_at`,
  `display_at`) field mappings, all stored for hit reconstruction.
- Updated `searchDocumentBytes` to count the new fields so the PV-MEM-002 batch
  bound stays honest.
- Refactored `Search` and `searchByTag` onto shared `textQueryClause`,
  `legacyTagQuery`, and `extractResults` helpers.
- Added `buildSearchQuery`, `dateRangeQuery`, `sortFields`, and `SearchAdvanced`.
- Added `request_test.go` and `search_advanced_test.go` (11 contract tests).

### Why

- One typed request object replaces positional growth and is shared by the
  HTTP and static implementations.
- Exact filters need separate keyword fields so current analyzed `#tag`
  discovery stays unchanged.
- Date ranges and sorts need datetime fields; result cards need stored
  provenance to reconstruct the display date without a second vault lookup.

### What worked

- Bleve's default missing-field sort puts undated notes last for both newest
  and oldest, so no explicit presence field was needed; the contract test pins
  this so a Bleve upgrade cannot silently change it.
- Refactoring legacy search onto shared helpers kept the existing
  equivalence tests green with no behavior change.
- The half-open `[from, to+1day)` range gives correct same-day inclusivity.

### What didn't work

- `golangci-lint exhaustive` required explicit enum cases in
  `dateFieldName`/`sortFields`; added all cases plus a defensive default.
- `bleve.NewDateRangeInclusiveQuery` takes `*bool` inclusive flags, not bool;
  passed `&inclStart`/`&inclEnd`.
- Sort/pagination tests initially used filter-only-empty requests, which are
  correctly not effective; switched them to a text query matching all notes so
  sorting across dated and undated notes is exercised.

### What I learned

- An empty (no text, no filters) request is not effective and returns empty by
  design; the note-list endpoint, not the advanced endpoint, lists everything.
- `tags_kw` must be lowercased explicitly because keyword fields use no
  analyzer, while the analyzed `tags` field is lowercased by the standard
  analyzer.
- `DateOnly.Before` uses calendar order so `date_to < date_from` validation is
  timezone-independent.

### What was tricky to build

The result date is reconstructed from the stored `display_at` instant plus the
`date_kind`/`date_precision` keywords, not from a stored original literal. For
date precision the instant is midnight UTC, which formats back to the original
`YYYY-MM-DD`; for timestamp precision it formats to UTC RFC3339 at second
precision. This keeps one read path and avoids storing a redundant string, but
it depends on the instant being normalized to UTC at index time (Phase A).

The no-fallthrough and alias rules from Phase A mean a note can have a valid
`date` ignored because an invalid `created` took precedence; the date-range
contract therefore filters on resolved display/created/updated instants only.

### What warrants a second pair of eyes

- Confirm `searchDocumentBytes` counting the `tags_kw` copy and `path_kw` copy
  keeps the 1 MiB batch bound honest under the PV-MEM-002 budgets.
- Confirm persistent reopen/equivalence still holds with the new stored fields
  (covered by `pkg/server` race tests).
- Confirm the `date_field` without a range is rejected rather than silently
  ignored.

### What should be done in the future

- Phase C should wire `NormalizeSearchRequest` + `SearchAdvanced` behind
  `/api/search/advanced` with stable 400 field errors and a legacy adapter.
- Phase D should reproduce the exact tag/path/date/sort contract in TypeScript
  with shared expected-ID fixtures.

### Code review instructions

- Start at `pkg/search/request.go` and `SearchAdvanced` in `pkg/search/search.go`.
- Run `GOWORK=off go test ./pkg/search/` (request + advanced contract tests).
- Verify legacy `TestSearchCompatibility` and persistent reopen tests still pass.
- Inspect `buildMapping` for the stored/not-stored field decisions.

### Technical details

```text
request tests: pass
advanced contract tests: 11 pass
legacy search equivalence: pass
persistent reopen: pass (race)
lint: 0 issues
gosec: 0 issues
new fields: tags_kw path path_kw created_at updated_at display_at date_kind date_precision
sorts: relevance=-_score,_id newest=-display_at,_id oldest=display_at,_id
missing date sort: last (Bleve default, pinned by test)
range: half-open [from, to+1day)
```

## Step 3: Phase C — advanced HTTP API

Phase C exposes the typed search over HTTP. The new `/api/search/advanced`
endpoint parses repeated and singleton parameters, rejects unknown keys,
returns a stable 400 field-error envelope, and delegates to the shared
`SearchAdvanced`. The legacy `/api/search` endpoint now delegates to the same
typed implementation so there is one search path during the compatibility window.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Implement Phase C: the advanced HTTP endpoint with validation, error contract, legacy adapter, and tests.

**Inferred user intent:** Give API consumers (and the future frontend) a stable, validated, documented advanced-search contract.

**Commit (code):** pending Phase C commit

### What I did

- Added `pkg/api/search_request.go` with `parseAdvancedParams`, the accepted
  parameter spec, the `advancedError` envelope, `jsonStatusResponse`, and the
  `searchAdvanced` handler.
- Registered `/api/search/advanced` in `Handler.Register`.
- Rewrote `searchNotes` to delegate to `SearchAdvanced` (legacy bare array kept).
- Added `pkg/api/search_advanced_test.go` (9 contract tests: envelope, empty
  array, filter-only, before_date_from, unknown parameter, repeated singleton,
  invalid limit, invalid date, legacy bare array).
- Updated the README API reference table and added an advanced-search curl example.

### Why

- A second endpoint avoids breaking existing bundles/scripts during deployment
  order differences; both handlers share one typed search method.
- Rejecting unknown parameters prevents misspelled filters from returning
  unexpectedly broad results.
- Stable field codes (not messages) are the machine contract.

### What worked

- The legacy endpoint delegating to `SearchAdvanced` kept the existing API test
  green and preserved the bare-array response shape.
- Empty results serialize as `[]` not `null` because the handler nil-checks.

### What didn't work

- First build failed: `encoding/json` was not imported in `search_request.go`.
- A first draft reimplemented `strings.Contains`; replaced with the stdlib call.
- Setting `WriteHeader` before `jsonResponse` would drop the Content-Type; added
  `jsonStatusResponse` to set Content-Type before the status code.

### What I learned

- `http.Header().Set` after `WriteHeader` is a no-op, so error envelopes need a
  helper that sets Content-Type first.
- The legacy and advanced endpoints intentionally return different shapes (bare
  array vs envelope); the test pins that the legacy body has no `"total":` key.

### What was tricky to build

The error envelope must be machine-readable and content-safe: field names and
finite codes are stable, but raw query values are never echoed into the body or
logs. Invalid dates and limits produce parse-time errors with the same field
names as the semantic errors from `NormalizeSearchRequest`, so the two error
lists are concatenated and the caller sees one coherent set.

### What warrants a second pair of eyes

- Confirm unknown-parameter rejection does not break legitimate query-string
  characters handled by gorilla/mux.
- Confirm the 500 `search_unavailable` path does not leak the underlying Bleve
  error message.
- Confirm the legacy adapter returns the same note set as before delegating.

### What should be done in the future

- Add a deprecation header or help note for `/api/search` once the frontend
  migrates fully (Phase E).
- Consider request-scoped context for cancellation so canceled requests do not
  log as server errors.

### Code review instructions

- Start at `pkg/api/search_request.go` and the `searchAdvanced` handler.
- Run `GOWORK=off go test ./pkg/api/`.
- Verify the README API table lists both search endpoints.

### Technical details

```text
api contract tests: 9 pass
legacy endpoint: delegates to SearchAdvanced, returns bare array
advanced endpoint: /api/search/advanced, returns envelope
error envelope: {"error":{"code","message","fields":[{field,code,message}]}}
unknown params: rejected (400)
repeated singletons: rejected (400)
lint: 0 issues
race: pass
```
