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
