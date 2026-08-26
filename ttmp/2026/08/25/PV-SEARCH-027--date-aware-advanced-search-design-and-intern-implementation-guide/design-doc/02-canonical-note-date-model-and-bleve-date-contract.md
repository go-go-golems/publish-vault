---
Title: Canonical note-date model and Bleve date contract
Ticket: PV-SEARCH-027
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
    - Path: repo://internal/parser/parser.go
      Note: Current frontmatter normalization and parser contract
    - Path: repo://pkg/search/search.go
      Note: Future datetime mapping query and stored-result integration
    - Path: repo://pkg/vault/vault.go
      Note: Current filesystem ModTime and future canonical date owner
    - Path: repo://ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/scripts/01-probe-date-frontmatter/main.go
      Note: Reproducible parser scalar-type probe
    - Path: repo://ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/scripts/02-probe-bleve-date-range/main.go
      Note: Reproducible Bleve datetime range and sort probe
    - Path: repo://web/src/vault/staticVault.ts
      Note: Static created-or-today behavior requiring parity correction
ExternalSources:
    - https://pkg.go.dev/github.com/blevesearch/bleve/v2
    - https://github.com/blevesearch/bleve/blob/v2.6.0/search/query/date_range.go
Summary: Defines authored created/updated dates, strict parsing, provenance, display precedence, range semantics, Bleve fields, static parity, and migration behavior without treating checkout ModTime as authored metadata.
LastUpdated: 2026-08-25T20:55:00-04:00
WhatFor: Providing one date contract for vault notes, search indexing, filters, sorting, API results, static builds, and UI rendering.
WhenToUse: Implement or review any date display, date filter, date sort, frontmatter alias, or static/dynamic parity change.
---


# Canonical note-date model and Bleve date contract

## 1. Decision summary

Search dates will represent **authored note metadata**, not Git checkout filesystem timestamps.

The first implementation supports:

- created-date aliases, in precedence order: `created`, then `date`;
- updated-date aliases, in precedence order: `updated`, `modified`, then `last_updated`;
- strict values in `YYYY-MM-DD` or RFC3339 form;
- separate canonical `created`, `updated`, and resolved `display` values;
- display precedence `updated` then `created`, otherwise absent;
- explicit `kind` and `precision` in API results;
- date-only range parameters with inclusive calendar-day semantics;
- Bleve datetime fields for created, updated, and display values;
- no fallback to filesystem `ModTime` in search display, filtering, or sorting;
- no fallback to the current build date in static mode.

`Note.ModTime` remains available to existing note/list APIs for compatibility, but it is not renamed or reinterpreted as an authored date.

## 2. Why filesystem ModTime is not the default

Dynamic mode obtains `ModTime` from `os.FileInfo` during `Vault.loadNote`. Production obtains files through `git-sync`, which creates a new worktree for a repository revision. Git tracks content and commit metadata, not original filesystem modification times. A checkout can assign similar or current timestamps to many notes regardless of when the author created or revised them.

Static mode currently assigns `modTime` from `created` frontmatter and substitutes today's date when `created` is absent. This means the same property has different meaning in each runtime and missing metadata changes whenever the static app rebuilds.

A search result that displays either value without provenance can make false claims. The canonical model therefore uses only explicit note-authored properties. Missing authored dates stay missing.

## 3. Domain types

The Go vault layer should own date resolution because it already owns normalized note semantics.

```go
type DatePrecision string

const (
    DatePrecisionDate      DatePrecision = "date"
    DatePrecisionTimestamp DatePrecision = "timestamp"
)

type NoteDateKind string

const (
    NoteDateCreated NoteDateKind = "created"
    NoteDateUpdated NoteDateKind = "updated"
)

type NoteDate struct {
    Value       time.Time
    Precision   DatePrecision
    SourceKey   string // canonical lower-case key selected by precedence
    Original    string // optional for diagnostics; never exposed in metrics
}

type NoteDates struct {
    Created *NoteDate
    Updated *NoteDate
}

func (d NoteDates) Display() (NoteDateKind, *NoteDate) {
    if d.Updated != nil {
        return NoteDateUpdated, d.Updated
    }
    if d.Created != nil {
        return NoteDateCreated, d.Created
    }
    return "", nil
}
```

`Note` receives `Dates NoteDates`. This avoids adding unrelated nullable fields and gives one resolver a stable output.

The API projection should omit internal source aliases and original values:

```go
type SearchResultDate struct {
    Value     string `json:"value"`     // date literal or UTC RFC3339
    Kind      string `json:"kind"`      // created | updated
    Precision string `json:"precision"` // date | timestamp
}
```

The result field is optional:

```go
type SearchResult struct {
    Slug    string            `json:"slug"`
    Title   string            `json:"title"`
    Excerpt string            `json:"excerpt"`
    Tags    []string          `json:"tags"`
    Path    string            `json:"path"`
    Score   float64           `json:"score"`
    Date    *SearchResultDate `json:"date,omitempty"`
}
```

The field is named `date`, not `modTime`, because it is a resolved display projection with explicit kind.

## 4. Alias precedence

Frontmatter lookup is case-insensitive, matching the existing publication flag convention.

| Canonical concept | Accepted keys, highest precedence first |
|---|---|
| Created | `created`, `date` |
| Updated | `updated`, `modified`, `last_updated` |

Precedence is evaluated within each concept. `updated` does not erase `created`; both are retained when valid. `Display()` selects updated over created because the search result's single date should describe the most recent authored state when available.

Keys such as `last_modified`, `creation_date`, and `mtime` are not accepted implicitly in version 1. Alias growth changes behavior and should require tests and documentation.

When two aliases for the same concept are present:

- the highest-precedence valid value wins;
- a higher-precedence **invalid** value does not silently fall through to a lower alias;
- the concept is omitted and a content-free warning counter/log records the invalid selected key.

Not falling through makes author mistakes visible and avoids a stale lower-priority value masking an invalid intended update.

## 5. Parsing and normalization

The parser probe at `artifacts/date-probe/result.json` proves that current Goldmark metadata normalization returns all tested date-only, RFC3339, quoted, and invalid values as Go strings. No automatic `time.Time` behavior should be assumed.

Accepted forms:

```text
YYYY-MM-DD
RFC3339 with an explicit timezone or Z
```

Examples:

```text
2024-01-15
2024-01-15T13:45:00-05:00
2024-01-15T18:45:00Z
```

Rejected examples:

```text
01/15/2024
January 15, 2024
2024-1-5
2024-01-15 13:45
```

Strict formats make URL, API, SSR, static build, and test behavior deterministic.

### Date-only values

A date-only value represents a calendar date without an author timezone. Preserve the literal `YYYY-MM-DD` for API display. For Bleve indexing, convert it to midnight UTC on that date:

```text
2024-01-15 -> 2024-01-15T00:00:00Z
```

This UTC instant is an indexing representation, not a claim that the author acted at midnight UTC.

### Timestamp values

An RFC3339 timestamp represents an instant. Normalize the indexed and API timestamp to UTC with second or source-supported fractional precision:

```text
2024-01-15T13:45:00-05:00 -> 2024-01-15T18:45:00Z
```

The UI can render a deterministic date label from the canonical value. Server-rendered text must not depend on the browser's local timezone during hydration.

### Pseudocode

```text
resolveNoteDates(frontmatter):
    created = resolveConcept(frontmatter, ["created", "date"])
    updated = resolveConcept(frontmatter, ["updated", "modified", "last_updated"])
    return NoteDates(created, updated)

resolveConcept(frontmatter, keys):
    selected = first case-insensitive key that exists
    if selected is absent:
        return nil
    if value is not a string:
        record invalid-date warning for selected canonical key
        return nil
    if value exactly matches YYYY-MM-DD and calendar date is valid:
        return NoteDate(midnight UTC, precision=date, sourceKey=selected)
    if value parses as strict RFC3339:
        return NoteDate(value converted to UTC, precision=timestamp, sourceKey=selected)
    record invalid-date warning for selected canonical key
    return nil
```

Warnings must not include title, path, slug, or raw value in metrics. A development log may name the frontmatter key and relative path under the repository's existing content-logging policy, but retained public evidence must remain content-free.

## 6. Missing and invalid values

Missing or invalid authored dates do not block vault publication. They produce `nil` canonical values.

Consequences:

- the result card omits the date row;
- date-range filters exclude the note because the selected field is absent;
- newest/oldest sorts place missing values last;
- relevance sort remains available;
- a future “missing date” filter can be added explicitly, but it is outside version 1.

The loader should count invalid values by finite concept (`created` or `updated`) and reason (`wrong_type`, `invalid_format`, `invalid_calendar_date`). It must not create labels from frontmatter key spelling or values.

## 7. SearchDocument and stored representation

Extend the streamed search document with metadata already resolved on `Note`:

```go
type SearchDocument struct {
    Slug      string
    Title     string
    Body      string
    Tags      []string
    Excerpt   string
    Path      string
    CreatedAt *time.Time
    UpdatedAt *time.Time
    DisplayAt *time.Time
    DateKind  string
}
```

The implementation may use a small index-specific struct to avoid pointer aliasing. Date precision is needed in the stored result representation. Two valid options are:

1. Store `date_precision` as a keyword field.
2. Derive precision from a stored API projection copied into `noteDoc`.

The recommended version stores `date_kind` and `date_precision` as keyword fields because search hits can reconstruct the result without a second vault lookup. This preserves snapshot locality and keeps one read path.

`searchDocumentBytes` must count path, date-kind, precision, and formatted timestamp bytes so the 1 MiB batch bound remains honest.

## 8. Bleve mapping

Bleve v2.6.0 provides:

- `bleve.NewDateTimeFieldMapping()`;
- `bleve.NewDateRangeInclusiveQuery(start, end, ...)`;
- `DateRangeQuery.SetField(...)`;
- `SearchRequest.SortBy([]string{"-display_at", "_id"})`;
- `bleve.NewKeywordFieldMapping()` for exact non-analyzed text.

The local probe `scripts/02-probe-bleve-date-range/main.go` and result `artifacts/date-probe/bleve-range.json` prove that a stored datetime field:

- accepts Go `time.Time` values;
- supports an inclusive-start/exclusive-end day interval;
- returns only documents inside that interval;
- sorts descending by date and then by `_id`.

Recommended mapping additions:

```go
createdAt := bleve.NewDateTimeFieldMapping()
createdAt.Store = true
dm.AddFieldMappingsAt("created_at", createdAt)

updatedAt := bleve.NewDateTimeFieldMapping()
updatedAt.Store = true
dm.AddFieldMappingsAt("updated_at", updatedAt)

displayAt := bleve.NewDateTimeFieldMapping()
displayAt.Store = true
dm.AddFieldMappingsAt("display_at", displayAt)

keyword := bleve.NewKeywordFieldMapping()
keyword.Store = true
dm.AddFieldMappingsAt("date_kind", keyword)
dm.AddFieldMappingsAt("date_precision", keyword)
```

The API should not expose Bleve's encoded `hit.Sort` values. They are internal cursor material if offset pagination is later replaced with search-after behavior.

## 9. Range semantics

Version 1 URL and API date filters are date-only:

```text
date=display|created|updated
after=YYYY-MM-DD
before=YYYY-MM-DD
```

Names `after` and `before` can be mistaken for exclusive endpoints. The recommended public names are therefore:

```text
date_field=display|created|updated
date_from=YYYY-MM-DD
date_to=YYYY-MM-DD
```

Both endpoints are inclusive calendar dates.

Internal conversion:

```text
start = date_from at 00:00:00Z, inclusive
end   = (date_to + 1 calendar day) at 00:00:00Z, exclusive
```

For `2024-01-16` through `2024-01-16`, the Bleve interval is:

```text
[2024-01-16T00:00:00Z, 2024-01-17T00:00:00Z)
```

This includes date-only values and every RFC3339 instant whose UTC date is January 16. If user-local calendar semantics are required later, the request must add an explicit timezone. Version 1 uses UTC to remain deterministic across server, static build, SSR, and clients.

Validation:

- `date_field` defaults to `display` when either endpoint exists;
- `date_field` without an endpoint is invalid or ignored by a documented decision; recommended: reject as ineffective input;
- `date_from > date_to` returns HTTP 400 with a field error;
- invalid dates return HTTP 400;
- ranges outside Bleve's RFC3339-compatible interval are rejected;
- absent fields do not match.

## 10. Sort semantics

Version 1 sort values:

```text
relevance
newest
oldest
```

Rules:

- `relevance` is valid when text query or legacy tag discovery is present;
- filter-only requests default to `newest`;
- `newest` sorts `-display_at`, then `_id` ascending;
- `oldest` sorts `display_at`, then `_id` ascending;
- missing display dates sort last;
- when `date_field=created|updated`, range filtering uses that field, but newest/oldest still use `display_at` in version 1 unless a future explicit `sort_date_field` is added.

Keeping sort and filter date concepts separate avoids a hidden rule where changing one dropdown changes result ordering.

## 11. API formatting

`SearchResultDate.Value` serialization:

| Precision | API value | Suggested visible text |
|---|---|---|
| `date` | original canonical `YYYY-MM-DD` | `Created 2024-01-15` |
| `timestamp` | UTC RFC3339 | `Updated 2024-01-15` with full timestamp in `<time datetime>` title/accessible text |

The card should use `<time dateTime={date.value}>`. Initial SSR and hydrated text should be deterministic. Locale formatting may be added after hydration only if it cannot change structural markup or cause mismatch; the simplest first implementation renders ISO dates.

## 12. Static-mode parity

Static mode must reuse a TypeScript translation of the same pure contract:

```ts
type NoteDate = {
  value: string;
  indexInstant: string;
  precision: "date" | "timestamp";
  sourceKey: string;
};

type NoteDates = {
  created?: NoteDate;
  updated?: NoteDate;
};
```

It must:

- apply the same alias order;
- accept the same strict formats;
- normalize timestamps to UTC;
- never use `new Date()` as a missing fallback;
- select updated over created for display;
- compare index instants for range and sort behavior;
- pass a shared JSON fixture matrix also consumed by Go tests.

A cross-language golden fixture is preferable to duplicating prose assumptions in two independent test tables.

## 13. Decision record DR-1: authored dates over filesystem fallback

**Context.** Search results need a meaningful date. Dynamic and static modes currently assign incompatible meanings to `modTime`.

**Options.** Use filesystem `ModTime`; use frontmatter only; use frontmatter with filesystem fallback; query Git history.

**Decision.** Use strict authored frontmatter dates only for search display/filter/sort. Preserve `ModTime` separately for compatibility. Do not query Git history in version 1.

**Rationale.** Frontmatter is portable across checkout, static build, API, and generated fixtures. Filesystem fallback is misleading in Git deployments. Git history would add repository coupling, startup I/O, rename semantics, and unavailable behavior in static mode.

**Consequences.** Notes without authored dates show no date and are excluded from date ranges. This absence is truthful and testable.

**Status.** Proposed for acceptance in the primary guide.

## 14. Decision record DR-2: three indexed date fields

**Context.** Users may filter by created or updated date while result cards need one display date.

**Options.** Index only display date; index created and updated and resolve display after hits; index created, updated, and display.

**Decision.** Index all three fields and store display provenance/precision.

**Rationale.** It supports field-specific filters, deterministic sorting, and self-contained hit reconstruction without a second vault lookup. The added values are small relative to body text but must still be measured.

**Consequences.** Mapping, search-document byte accounting, index size, and memory budgets change. Persistent indexes rebuild automatically with each snapshot.

**Status.** Proposed.

## 15. Test matrix

The implementation should share cases such as:

| Frontmatter | Created | Updated | Display kind | Result |
|---|---|---|---|---|
| none | absent | absent | absent | no date |
| `created: 2024-01-15` | date | absent | created | valid |
| `date: 2024-01-15` | date | absent | created | alias valid |
| created + date | created wins | absent | created | deterministic |
| `updated: ...Z` | absent | timestamp | updated | valid |
| created + updated | both | both | updated | valid |
| invalid `created` + valid `date` | absent | absent | absent | warning; no fallthrough |
| timezone timestamp | normalized UTC | — | created | exact instant |
| non-string value | absent | — | absent | wrong-type warning |

Range tests must cover same-day inclusivity, open endpoints, leap days, invalid order, missing fields, timestamp boundary instants, and deterministic tie sorting.

## 16. Review risks

- Using `time.Time` alone loses date-only precision; retain precision separately.
- Parsing permissive natural-language dates creates backend/static divergence.
- Converting timestamps to browser-local dates during SSR can shift the displayed calendar day and cause hydration mismatch.
- Falling back to `ModTime` reintroduces checkout-time semantics.
- Falling through from invalid high-priority aliases hides author errors.
- Storing only display date prevents created/updated-specific filters.
- Adding fields without updating batch byte estimates invalidates PV-MEM-002's bound accounting.
