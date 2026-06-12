---
title: Tag-based search analysis and implementation guide
doc_type: design-doc
status: active
intent: long-term
topics: [frontend, search, ux]
ticket: RETRO-TAG-010
created: 2026-06-12
---

# Tag-Based Search Analysis and Implementation Guide

## Executive Summary

Tags are already indexed in the bleve search engine but there's no way to perform a tag-specific search. Clicking a tag currently navigates to an empty search page. This ticket adds `#tagname` search syntax: when a query starts with `#`, the backend performs a field-scoped search on the `tags` field only, and the frontend uses `%23tagname` in URLs to avoid hash-fragment conflicts.

## Problem Statement

1. **Tags are clickable but do nothing useful** — `handleTagClick` in `NotePage.tsx` just calls `navigate("/search")` without any query, landing the user on an empty search page.
2. **No tag-specific search** — Searching for "philosophy" matches both body text and tags, returning many irrelevant results. There's no way to say "only show me notes tagged with philosophy."
3. **Tags in search results aren't clickable** — The `NoteCard` component renders tags with the `Tag` atom but doesn't wire `onClick`, so they're display-only.
4. **No tag browsing** — The `/api/tags` endpoint exists and returns `{tag, count}[]`, but the frontend never calls `useListTagsQuery`. There's no tag cloud or tag filter in the search page.

## Current-State Architecture

### Backend: search indexing (Go + Bleve)

| File | Role |
|------|------|
| `internal/search/search.go` | Bleve index wrapper; `Index()` stores tags as space-separated text; `Search()` tokenizes query and uses fuzzy/prefix matching across all fields |
| `internal/api/api.go` | HTTP handler `searchNotes` passes `?q=` directly to `search.Search()`; `listTags` returns tag counts |
| `internal/vault/vault.go` | `Note.Tags []string` populated from YAML frontmatter |

**Key detail:** Tags are stored in bleve as a single space-separated string (`"philosophy stoicism"`). The `tags` field uses the standard analyzer, so it's tokenized by whitespace. This means a bleve field-scoped query like `tags:philosophy` will match notes with that tag.

### Frontend: tag rendering and click handling

| Component | Current behavior |
|-----------|-----------------|
| `Tag` atom | Renders `#label` with optional `onClick` |
| `FrontmatterPanel` | Renders tags with `onTagClick` → opens `/search` with no query |
| `NoteCard` | Renders tags as display-only (no `onClick` on Tag) |
| `NotePage.handleTagClick` | `navigate("/search")` — ignores the tag name entirely |
| `SearchPage` | Reads `?q=` from URL, searches all fields |

### The `#` character in URLs

The `#` character has special meaning in URLs (fragment identifier). In a query parameter value, `?q=#philosophy` would be parsed as `q=""` with fragment `#philosophy`. The solution is to URL-encode `#` as `%23`:

- Search URL: `/search?q=%23philosophy`
- `URLSearchParams.get("q")` returns `#philosophy` (decoded)
- `encodeURIComponent("#philosophy")` produces `%23philosophy`

This works cleanly with React Router's `useSearchParams`.

## Proposed Solution

### 1. Backend: detect `#` prefix and perform field-scoped search

In `internal/search/search.go`, modify `Search()`:

- If the query starts with `#`, strip the `#` and perform a **field-scoped match query** on the `tags` field only
- Use `bleve.NewMatchQuery(tagName).SetField("tags")` for exact tag matching
- For `#` queries with fuzziness (e.g., `#phil`), use prefix query on the tags field
- Otherwise, keep existing behavior (search all fields)

### 2. Frontend: wire tag clicks to `#tagname` search

- `NotePage.handleTagClick`: navigate to `/search?q=${encodeURIComponent("#" + tag)}`
- `NoteCard`: make tags clickable with same `#tagname` search
- `SearchPage`: already reads `?q=` and dispatches `setSearchQuery` — no change needed

### 3. Frontend: add tag cloud to search page (when no query)

When the search page loads with no query, show a tag cloud using `useListTagsQuery`. Each tag is clickable and triggers `#tagname` search.

## Design Decisions

### Decision 1: Use `#` prefix vs. `tag:` prefix

**Options:**
- (A) `#tagname` — familiar from social media and Obsidian; concise
- (B) `tag:tagname` — clearer intent; no URL encoding issues; used by GitHub search
- (C) Both — accept either prefix

**Decision:** Use `#tagname` as the primary syntax. This is what Obsidian uses for tags, and it's what users of a vault app would expect. The URL encoding issue (`%23`) is handled transparently by `URLSearchParams`. We also accept `tag:` as an alias for discoverability.

**Rationale:** `#` is the Obsidian convention. Users will type `#philosophy` naturally. The `%23` encoding in URLs is an implementation detail that React Router handles correctly.

### Decision 2: Tag search strategy

**Options:**
- (A) Field-scoped bleve query — search only in the `tags` field
- (B) Prepend `tag:` to the indexed tag values — search `tag:philosophy` as a token
- (C) Separate `/api/search/tags?q=...` endpoint

**Decision:** (A) Field-scoped bleve query.

**Rationale:** Bleve supports `SetField()` on match queries. Tags are already indexed in a separate `tags` field with the standard analyzer. A field-scoped query is the most direct and efficient approach. No re-indexing needed.

### Decision 3: Tag cloud placement

**Options:**
- (A) Search page, shown when query is empty
- (B) Separate `/tags` page
- (C) Sidebar section

**Decision:** (A) Search page, when query is empty.

**Rationale:** The search page is already the "discovery" page. Showing a tag cloud when idle provides a natural entry point for tag-based browsing without adding a new route or sidebar complexity.

## Implementation Plan

### Step 1: Backend — add `#`-prefix tag search to bleve

**File:** `internal/search/search.go`

- Detect `#` prefix in `Search()`
- Strip `#`, perform `bleve.NewMatchQuery(tagName)` with `.SetField("tags")`
- Also detect `tag:` prefix as alias
- Return results normally (same `SearchResult` shape)
- Add unit tests

### Step 2: Frontend — wire tag clicks to `#tagname` search

**Files:**
- `web/src/components/pages/NotePage/NotePage.tsx` — update `handleTagClick`
- `web/src/components/molecules/NoteCard/NoteCard.tsx` — add `onTagClick` prop, wire to Tag

### Step 3: Frontend — add tag cloud to search page

**File:** `web/src/components/pages/SearchPage/SearchPage.tsx`

- Import `useListTagsQuery`
- When `query.trim().length < 2`, show tag cloud instead of "Type at least 2 characters"
- Each tag click navigates to `/search?q=%23tagname`

### Step 4: Frontend — create `TagCloud` molecule

**File:** `web/src/components/molecules/TagCloud/TagCloud.tsx`

- Renders tags sorted by count (descending)
- Each tag is a clickable `Tag` atom
- Sizing: larger tags for higher counts (optional, can be same size initially)
- Retro styling: use `.retro-tag` classes

## Risks and Open Questions

- **Bleve standard analyzer + `#` prefix**: The standard analyzer may strip `#` as a punctuation character. We need to strip `#` on the Go side before passing to bleve, which is already the plan.
- **Partial tag matching**: `#phil` should probably match `philosophy`. Using a prefix query on the tags field would handle this. Need to decide if we want exact-only or prefix matching.
- **Tag indexing format**: Tags are currently stored as space-separated text. Multi-word tags (with spaces) wouldn't work correctly with this approach. However, Obsidian tags don't support spaces, so this is not a real concern.
