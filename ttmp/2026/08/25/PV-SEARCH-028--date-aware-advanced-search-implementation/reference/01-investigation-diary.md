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

## Step 4: Phase D — shared TS types, URL codec, and static parity

Phase D brings the frontend to the same typed contract as the backend. There
are now shared TypeScript request/response types, a pure canonical URL codec, an
RTK Query `searchAdvanced` endpoint with a canonical cache key, and a static
`staticSearchAdvanced` that reproduces the backend's exact tag/path/date/sort/
pagination inclusion and ordering. The static vault resolves authored dates on
its notes and the legacy `#tag` discovery uses the pinned prefix/edit-distance
contract.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Implement Phase D: TS types, URL codec, RTK Query advanced endpoint, and static-mode parity with shared fixtures.

**Inferred user intent:** Make static and dynamic modes include and order the same notes for structured filters, with a shareable canonical URL.

**Commit (code):** pending Phase D commit

### What I did

- Added `SearchRequest`/`SearchResponse`/`SearchResultDate`/`DateOnly`/
  `FieldError`/`TagMode`/`DateField`/`SearchSort` to `web/src/types/index.ts`
  and added `path`/`date` to `SearchResult` and `dates` to `Note`.
- Added `web/src/search/searchParams.ts` with `parseDateOnly`,
  `normalizeSearchRequest`, `canonicalizeSearchRequest`, `encodeSearchParams`,
  `decodeSearchParams`, and `isEffective` mirroring `pkg/search/request.go`.
- Added `web/src/search/searchParams.test.ts` (10 URL codec/normalization tests).
- Refactored `buildVault` into `buildVaultFromRaw(rawFiles)` and resolved
  `Note.dates` in build; extracted `searchAdvancedInNotes` for testability.
- Implemented `staticSearchAdvanced` with exact tag all/any, path prefixes,
  date ranges, deterministic sorts (missing date last), pagination, total, and
  the pinned legacy `#tag` prefix/edit-distance-1 contract.
- Added `web/src/vault/staticVault.advanced.test.ts` (11 parity tests using the
  same Alpha/Beta/Gamma/Plain vault as the Go contract tests).
- Added the `searchAdvanced` RTK Query endpoint (backend + static) with a
  canonical `serializeQueryArgs` cache key and exported `useSearchAdvancedQuery`.
- Updated legacy `staticSearch` results to include `path`.

### Why

- The browser URL is the committed request state; a pure codec keeps it
  shareable and reconstructable.
- Static and dynamic modes must include and order the same notes for metadata
  filters; the same fixture vault proves it.
- A canonical RTK Query cache key prevents equivalent tag orderings from
  creating duplicate cache entries.

### What worked

- The 11 static parity tests use the exact Alpha/Beta/Gamma/Plain vault from
  the Go tests and assert the same expected IDs/order, proving backend ↔ static
  parity for structured filters, dates, sorts, and pagination.
- Extracting `buildVaultFromRaw`/`searchAdvancedInNotes` let tests feed
  controlled notes without the singleton demo vault.
- The pinned legacy `#go` prefix contract matches the backend exactly for
  short queries; edit-distance-1 covers longer tags.

### What didn't work

- `for...of` over `URLSearchParams` needed `Array.from` (tsconfig target).
- `new Set(PARAM_ORDER)` inferred a literal-union Set; typed it as
  `Set<string>`.
- `SearchResult` gained required `path`, which broke the existing
  `staticSearch` result literal; added `path`.
- The empty-request sort default only applies when the raw `sort` is empty, so
  the request type now allows `sort: ""` (and `dateField`/`tagMode` empty) to
  mirror Go zero values.

### What I learned

- RTK Query `serializeQueryArgs` must produce a stable string for equivalent
  requests; canonicalizing before encoding gives that.
- Static text ranking (title/tag/excerpt substring) cannot match Bleve score,
  but inclusion and deterministic tie-break (slug) can.
- Resolving dates once at build time and storing on `Note` avoids re-parsing
  frontmatter on every static search.

### What was tricky to build

The date-range boundary had to match the backend's half-open `[from, to+1day)`
interval exactly: `t >= from && t < nextDayStart`. The static `DateOnly` instant
helpers (`dateOnlyToInstant`, `dateOnlyNextDayInstant`) mirror Go's
`StartUTC`/`NextDayStartUTC` so a single-day filter `[d, d]` includes the whole
calendar day in both runtimes.

The legacy `#tag` fuzzy contract is the one place static mode approximates Bleve:
Bleve uses a fuzziness-1 MatchQuery with the standard analyzer, while static
mode uses exact-or-Levenshtein-1 over normalized complete tags. For typical
single-word tags these agree; the design explicitly does not promise score
parity, and the parity fixtures use prefix (<=3) cases that match exactly.

### What warrants a second pair of eyes

- Confirm `staticSearchAdvanced` and the Go `SearchAdvanced` agree on the
  shared fixture vault for every structured case (the tests assert this).
- Confirm the RTK Query cache key is stable across equivalent tag/path orderings.
- Confirm the static `Note.dates` field does not leak into a serialized shape
  the static `getNote` consumer cannot handle (it is optional).

### What should be done in the future

- Phase E should drive `SearchPage` from the canonical URL via
  `useSearchAdvancedQuery`, add accessible filter controls, and render result
  dates with `<time dateTime>`.
- A follow-up should decide whether to deprecate the legacy `useSearchQuery`.

### Code review instructions

- Start at `web/src/search/searchParams.ts` and `staticSearchAdvanced` in
  `web/src/vault/staticVault.ts`.
- Run `pnpm --dir web vitest run` and `pnpm --dir web build`.
- Compare `staticVault.advanced.test.ts` with `pkg/search/search_advanced_test.go`.

### Technical details

```text
web tests: 82/82 (added 10 searchParams + 11 static advanced)
web build: pass
url codec round-trip: pass
static parity: 11/11 match Go contract
RTK Query: searchAdvanced endpoint + useSearchAdvancedQuery
legacy #tag: pinned prefix (<=3) / edit-distance-1 (>3)
```

## Step 5: Phase E — advanced-search UI and date rendering

Phase E makes the feature user-facing. SearchPage is now driven by the canonical
URL: it decodes URL params into a typed request, queries the shared
`useSearchAdvancedQuery`, and renders results with authored dates and paths. An
accessible filter panel edits tags, tag mode, folder prefixes, and date range;
the header holds the text field, sort, result count, and pagination. Invalid URL
filters render a reset action instead of being silently dropped.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Implement Phase E: URL-driven SearchPage, accessible filter controls, sort, applied chips, pagination, and NoteCard date/path rendering.

**Inferred user intent:** Deliver the visible feature so a user can filter, sort, share a URL, and see truthful authored dates.

**Commit (code):** pending Phase E commit

### What I did

- Rewrote `SearchPage` to decode the URL via `decodeSearchParams` +
  `normalizeSearchRequest`, query `useSearchAdvancedQuery`, and commit changes
  back through a single `commitRequest` that re-encodes the canonical URL.
- Added `AdvancedSearchPanel` (Radix Dialog) with tag, tag-mode, path, date
  field, and date-range inputs; Apply merges with the current query/sort and
  resets offset.
- Added a header sort select, an active-filter chip row with per-chip remove,
  a result count, and Prev/Next pagination.
- Invalid URL filters render a reset-all banner with the field errors.
- Updated `NoteCard` to accept `date` (rendered as `<time dateTime>` with a
  created/updated label) and `path` (rendered as a small breadcrumb), keeping
  `modTime` as a fallback.
- Added NoteCard date/path stories and an AdvancedSearchPanel story.

### Why

- The URL is the committed request; decoding/encoding through one codec keeps
  it shareable and reconstructable.
- Accessible controls (labelled inputs, Radix Dialog) keep filters usable by
  keyboard and screen readers.
- Showing the authored date with `<time>` keeps SSR text deterministic and
  avoids hydration mismatch.

### What worked

- Reusing the Phase D codec meant the page logic is mostly decode then query
  then encode, with no duplicated validation.
- `skip: !effective || errors.length > 0` avoids querying for empty or invalid
  requests and lets the empty state show the TagCloud.
- Storybook, Vitest, client build, and SSR build all pass with the new stories.

### What didn't work

- First tsc pass used icon names "filter" and "x" that the Icon set does not
  expose; switched to "menu" and "close".
- `parseDateOnly` returns `DateOnly | null`; the panel needed `?? undefined` to
  satisfy the `DateOnly | undefined` request field.
- `NoteCard` lacked a `path` prop; added it as an optional breadcrumb.

### What I learned

- One `commitRequest` funnel keeps the URL canonical and prevents partial
  updates from drifting out of sync with Redux.
- Pagination via offset changes the canonical URL, so a shared page link
  includes the page.

### What was tricky to build

The page has two sources of truth risk: Redux `searchQuery` and the URL. The
new SearchPage treats the URL as canonical and only uses Redux for `activeNote`
navigation; the old `searchQuery` slice is no longer read here, so the text
field is controlled by `request.query` (decoded from the URL). This avoids the
dual-source drift the design warned about.

Invalid filters must stay visible: the page decodes errors, skips the query,
and shows a reset-all action rather than silently clearing the URL.

### What warrants a second pair of eyes

- Confirm the filter panel is keyboard-accessible (Radix Dialog focus trap).
- Confirm the `<time>` label does not shift the calendar day across SSR/client.
- Confirm pagination and filter changes reset offset to 0.

### What should be done in the future

- Add a jsdom-based SearchPage integration test once a DOM test environment is
  adopted (the project currently uses node env).
- Deprecate `useSearchQuery` once all consumers migrate.
- Consider a "Load More" alternative to offset pagination for large result sets.

### Code review instructions

- Start at `SearchPage.tsx` and `AdvancedSearchPanel.tsx`.
- Run `pnpm --dir web check`, `pnpm --dir web vitest run`, `pnpm --dir web build`,
  and `pnpm --dir web build-storybook`.
- Verify NoteCard date/path stories render.

### Technical details

```text
web tests: 82/82
web build: pass
storybook build: pass
backend tests: pass (cached)
lint: 0 issues
SearchPage: URL-driven via useSearchAdvancedQuery
NoteCard: <time dateTime> date + path breadcrumb
pagination: offset-based Prev/Next
```

## Step 6: Phase F — full validation, Docker/Compose smoke, and PR

Phase F closes the loop. The full gate suite passes, the production Docker image
builds and serves the new endpoint, and the validation evidence is recorded. The
PR is the gate to merge and then roll out.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Run every repository gate, smoke the Docker/Compose image, record evidence, and open the PR.

**Inferred user intent:** Prove the feature is shippable and ready for review/rollout.

**Commit (code):** 283591f (validation evidence)

### What I did

- Ran `make ci-check` (exit 0), `go test -race ./... -count=1` (all pass),
  `pnpm vitest run` (82/82), `pnpm build`, and `pnpm build-storybook`.
- Built the Docker image and ran `docker compose up -d --build` against the demo
  vault.
- Verified `/api/healthz`, `/api/search/advanced` (tag filter, envelope, path,
  date omitted when absent), the 400 `before_date_from` envelope, and the
  legacy `/api/search` bare array.
- Recorded evidence in `artifacts/final/01-phase-f-validation.md`.
- Marked all six phase tasks complete and pushed the branch.

### Why

- A feature is not done until the production image proves it and the gates are
  green; the smoke catches embed/build issues that unit tests miss.
- The `date` field being omitted for notes without authored dates confirms the
  truthful-absence behavior end to end.

### What worked

- The advanced endpoint, 400 contract, and legacy adapter all worked in the
  production image on first try.
- App memory for the demo vault was ~14.7 MiB, far under budget.

### What didn't work

- `q=zettel` returns empty in both legacy and advanced search because Bleve
  tokenizes `zettelkasten` as one token and `zettel` is edit-distance >1; this
  is existing Bleve behavior, not a regression, and the tag filter matches.

### What I learned

- The demo vault has no authored dates, so the `date` field is `omitempty`-omitted
  in live responses; the date reconstruction path is proven by the Go and static
  unit tests instead.

### What was tricky to build

Nothing new in Phase F; the work was validation and evidence capture.

### What warrants a second pair of eyes

- Run a full-vault memory/index measurement against the private vault before the
  GitOps rollout to confirm the bounded peak stays within 1 GiB request / 2 GiB
  limit (the bounded-batch mechanism is unchanged).

### What should be done in the future

- After merge: publish the optimized image, bump the GitOps tag, and roll out
  with `maxSurge: 0, maxUnavailable: 1`.
- Deprecate `useSearchQuery` once the frontend migration is confirmed stable.

### Code review instructions

- Read `artifacts/final/01-phase-f-validation.md`.
- Re-run `make ci-check` and the Docker/Compose smoke if needed.

### Technical details

```text
make ci-check: exit 0
race: all pass
web tests: 82/82
docker build: pass
compose smoke: pass
advanced endpoint: 200 envelope + 400 contract verified
app memory (demo): ~14.7 MiB
PR: pending
```

## Step 7: PR #27 review corrections

PR #27 review raised five issues: an inclusive day-after upper bound on the
date range (P1), silent acceptance of JS-normalized invalid calendar dates
(P2), an explicit `limit=0` silently becoming the default (P2), partial
numeric parsing in the URL codec (P2), and ignored backend search failures
(P2). All five are addressed with regression tests.

### Prompt Context

**User prompt (verbatim):** "Address code review issues: https://github.com/go-go-golems/publish-vault/pull/27"

**Assistant interpretation:** Inspect every inline review thread on PR #27, fix each with code-backed decisions, add regression tests, validate, and reply to the threads.

**Inferred user intent:** Make PR #27 correct before merge: date boundaries, invalid-date rejection, limit/offset parsing, and error surfacing.

**Commit (code):** pending review-correction commit

### What I did

- **P1 (search.go):** Confirmed with a focused test that `date_to=2024-03-01`
  wrongly included `2024-03-02T00:00:00Z` because `inclEnd` was true on the
  day-after endpoint. Read Bleve's `date_range.go` to confirm a zero time is an
  open bound regardless of the inclusive flag, then set the end exclusive. Added
  `TestSearchAdvancedDateRangeExclusiveUpperBound` as a permanent contract test.
- **P2 dates (searchParams.ts, noteDate.ts):** `parseDateOnly` and
  `parseNoteDate` now construct the instant from the parsed components and
  round-trip the UTC year/month/day against the input, so `2024-02-30` is
  rejected instead of silently shifted to March 1. Added six TS tests.
- **P2 limit=0 (search_request.go):** The HTTP parser now rejects an explicitly
  supplied `limit=0` as out of range before normalization can turn it into the
  default of 30. Added `TestAdvancedSearchExplicitZeroLimitRejected`.
- **P2 numeric parse (searchParams.ts):** `parseIntOpt` now requires the entire
  value to match `^-?\d+$` and takes `min`/`max`, so `10junk`, `2.5`, and `1e2`
  decode as errors instead of a silently different request. Limit uses 1-100,
  offset uses 0-10000. Added TS tests.
- **P2 error surfacing (SearchPage.tsx):** The page now reads `isError` and
  `refetch` from `useSearchAdvancedQuery` and renders a "Search is temporarily
  unavailable" state with a Retry button, distinct from a successful empty
  response.

### Why

- The half-open `[from, to+1day)` range is the documented contract; an inclusive
  day-after endpoint matched a note on the wrong calendar day.
- JS `Date` normalizes nonexistent dates, so without a round-trip check the
  static mode accepted authored dates the Go backend rejected.
- `0` is the internal omitted-value sentinel, so an explicit `limit=0` had to
  be rejected at the boundary that knows the parameter was present.
- `Number.parseInt` accepts a numeric prefix, so a malformed shared URL had to
  be rejected rather than silently executing a different request.
- A backend outage is not an empty search; the page must distinguish them.

### What worked

- The Bleve source confirmed the open-bound semantics, so the exclusive-end fix
  needed no special-casing for the open upper bound.
- Go `ParseDateOnly` already round-tripped, so the date fix was TS-only.
- `refetch` from RTK Query is the correct retry primitive; re-committing the
  same URL would not invalidate the cache.

### What didn't work

- The edit tool rejected several multi-edit calls with "must be object" on
  payloads containing Go string literals with parentheses and template
  backticks; fell back to single edits and a Python insertion for the Go test.

### What I learned

- Bleve treats a zero `time.Time` as an open bound in `parseEndpoints`
  (`min=-Inf`/`max=+Inf`), so the inclusive flags only matter for finite bounds.
- `Number.parseInt("2.5")` returns `2` and `Number.parseInt("1e2")` returns `1`;
  a strict integer regex is the only reliable guard.

### What was tricky to build

The `limit=0` fix had to live at the HTTP boundary, because the typed
`SearchRequest.Limit` field uses zero as its omitted-value sentinel and
`NormalizeSearchRequest` cannot distinguish "omitted" from "explicit zero". The
parser is the only layer that knows the parameter was present, so it owns the
rejection; normalization's range check still guards direct programmatic callers.

### What warrants a second pair of eyes

- Confirm the static `dateOnlyToInstant`/`dateOnlyNextDayInstant` helpers only
  receive already-validated `DateOnly` values (they do; the only callers pass
  `req.DateFrom`/`req.DateTo` from a normalized request).
- Confirm the Retry button's `refetch` re-runs the query against the current
  canonical URL.

### What should be done in the future

- Consider exporting the limit/offset bounds from the search package so the HTTP
  parser and URL codec share one source of truth rather than re-declaring the
  constants.

### Code review instructions

- Run `GOWORK=off go test ./pkg/search/ ./pkg/api/` and
  `pnpm --dir web vitest run`.
- Verify `TestSearchAdvancedDateRangeExclusiveUpperBound`,
  `TestAdvancedSearchExplicitZeroLimitRejected`, the new `parseDateOnly`
  calendar tests, and the partial-numeric-parse tests.

### Technical details

```text
review issues: 5 (1 P1, 4 P2)
date range end: exclusive (inclEnd=false)
invalid calendar dates: rejected in Go and TS via round-trip
explicit limit=0: rejected at HTTP boundary
partial numeric parse: rejected via strict integer regex
search failures: surfaced with isError + refetch Retry
web tests: 91/91
go race: pass
lint: 0 issues
gosec: 0 issues
```
