---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: backend/internal/api/api.go
      Note: API handler that serves note HTML
    - Path: backend/internal/parser/parser.go
      Note: Contains the buggy ReplaceWikiLinkDisplay function and all wiki-link processing
    - Path: backend/internal/vault/vault.go
      Note: Calls rebuildHTML which invokes ReplaceWikiLinkDisplay
    - Path: web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Note renderer component that injects resolved HTML
    - Path: web/src/lib/wikiLinks.ts
      Note: Frontend wiki-link post-processor
    - Path: web/src/vault/staticVault.ts
      Note: Static vault wiki-link handling (marked extension)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---







# Wiki-link rendering analysis and implementation guide

## Executive summary

Wiki links (`[[Target]]` syntax) are a core feature of Obsidian vaults. When the retro-obsidian-publish server renders a note containing wiki links inside Markdown tables, the `ReplaceWikiLinkDisplay` function in the Go backend **strips the `<a` opening tag** from the generated HTML anchor element. The result: the browser sees bare text like ` href="/note/..." class="wiki-link" ...` instead of `<a href="/note/..." class="wiki-link" ...>`, which causes the anchor to fail entirely. In some cases, the display text replacement also pulls in content from adjacent table cells because the regex is too greedy.

This document explains every layer of the wiki-link rendering pipeline — from the raw Markdown source through the Go parser, the vault index, the HTML post-processor, the API transport, and the frontend renderer — so that a new intern can understand the system, reproduce the bug, and implement the fix.

## Problem statement

### What the user sees

A note like `Research/KB/Tribal/dmeta-design-system-compiler-pipeline.md` contains a table with wiki links in the first column:

```markdown
| Report | Date | Contribution |
| --- | --- | --- |
| [[ARTICLE - DMETA Design System Factory]] | 2026-05-23 | Canonical description |
```

The rendered HTML should produce a clickable link whose display text is the resolved note title (e.g. "DMETA Design System Factory: From Semantic Schemas to Generated React Widgets"). Instead, the browser renders:

```html
href="/note/projects/2026/05/23/article-dmeta-..." class="wiki-link"
  data-target="projects/2026/05/23/article-dmeta-..."
  data-raw="ARTICLE - DMETA Design System Factory..."
  data-alias="">DMETA Design System Factory: ...  2026-05-23  Canonical description...
```

Notice two problems:

1. **The `<a` opening tag is missing** — the browser treats the `href=...` as raw text, not as an anchor element.
2. **The display text includes adjacent table cell content** — "2026-05-23" and "Canonical description" from neighboring `<td>` elements bleed into the link text.

### Root cause

The `ReplaceWikiLinkDisplay` function in `backend/internal/parser/parser.go` uses a regex to match wiki-link anchors and replace their display text. The regex captures groups between `<a` and `class="wiki-link"`, but the reconstruction **omits the literal `<a` prefix**, producing ` href=...` instead of `<a href=...`.

Additionally, the regex `([^<]*?)` for the display text is not constrained to a single HTML element boundary. When the `<a` tag is already broken (missing), the regex can match further than intended, pulling in content from sibling elements.

## System architecture: the wiki-link rendering pipeline

The rendering of a wiki link passes through five distinct stages. Each stage transforms the data and introduces specific constraints. Understanding all five stages is essential for making correct fixes.

### Stage 1: Raw Markdown → Pre-processed Markdown (Go parser)

**File:** `backend/internal/parser/parser.go` — function `replaceWikiLinks()`

**What happens:** The parser reads raw Markdown bytes. Before goldmark (the Markdown-to-HTML engine) sees the content, `replaceWikiLinks` replaces every `[[Target]]`, `[[Target|Alias]]`, `[[Target#Heading]]`, and `![[Embed]]` with a complete HTML anchor element or embed placeholder.

**Why pre-processing is necessary:** Goldmark does not understand `[[wiki link]]` syntax. If left unprocessed, goldmark would render `[[Target]]` as literal text `[[Target]]`. By replacing wiki links with HTML anchors before goldmark runs, we ensure the links survive as raw HTML in the final output (goldmark is configured with `html.WithUnsafe()` to pass through raw HTML).

**The regex:** `(!?)\[\[([^\[\]]+)\]\]` — matches optional `!` prefix, then `[[...]]` with any characters except `[` and `]` inside.

**The replacement function `wikiLinkHTML`:**

```
Pseudocode for wikiLinkHTML(match):
  isEmbed = (match starts with '!')
  inner = strip brackets → parseWikiLinkInner(inner)
  target, alias, heading = parseWikiLinkInner(inner)
  slug = slugify(target)
  display = alias || target

  if isEmbed:
    return <div class="wiki-embed" data-target=slug data-heading=heading data-raw=target>
  
  href = "/note/" + slug
  if heading: href += "#" + slugify(heading)
  return <a href=href class="wiki-link" data-target=slug data-raw=target data-alias=alias>display</a>
```

**Key invariant:** The HTML anchor produced at this stage is self-contained — it contains all the information needed for later resolution, encoded as data attributes.

**Frontmatter protection:** `splitFrontmatter` ensures that wiki links inside YAML frontmatter (e.g. `related_reports: ["[[Note]]"]`) are NOT replaced with HTML — doing so would corrupt the YAML and cause goldmark-meta to fail.

### Stage 2: Pre-processed Markdown → HTML (Goldmark)

**File:** `backend/internal/parser/parser.go` — `Parse()` function, goldmark conversion step

**What happens:** Goldmark converts the pre-processed Markdown (which now contains raw HTML anchor elements where wiki links used to be) into full HTML. Goldmark's GFM extension handles tables, task lists, strikethrough, and footnotes. The `html.WithUnsafe()` renderer option ensures raw HTML tags pass through unchanged.

**The critical interaction with tables:** When a wiki link appears inside a Markdown table cell, the pre-processing step has already converted `[[Target]]` to `<a href="..." class="wiki-link" ...>Target</a>`. Goldmark's table renderer wraps this anchor inside `<td>...</td>`. At this stage, the HTML is correct:

```html
<td><a href="/note/target" class="wiki-link" data-target="target" data-raw="Target" data-alias="">Target</a></td>
```

### Stage 3: HTML post-processing — wiki-link resolution and display replacement

**File:** `backend/internal/parser/parser.go` — `ReplaceWikiLinksString`, `ReplaceWikiLinkDisplay`, `RewriteImageSources`

**What happens:** After all notes are parsed, the vault builds a `wikiLinkIndex` that maps short slugified targets (like `"tribal/foo"`) to full vault slugs (like `"research/kb/tribal/foo"`). Then `rebuildHTML()` walks every note's HTML and applies three post-processing passes:

1. **`ReplaceWikiLinksString`** — resolves `data-target` and `href` attributes from short slugs to full vault slugs.
2. **`ReplaceWikiLinkDisplay`** — replaces the display text of non-aliased wiki links with the resolved note's title.
3. **`RewriteImageSources`** — rewrites relative image `src` paths to `/vault-assets/...` URLs.

**This is where the bug lives.** Let's examine `ReplaceWikiLinkDisplay` in detail:

```go
// The regex (current, buggy):
wikiLinkRe := regexp.MustCompile(
  `<a([^>]*?)class="wiki-link"([^>]*?)data-raw="([^"]*?)"([^>]*?)>([^<]*?)</a>`
)
```

The five capturing groups:

| Group | Matches | Example capture |
|-------|---------|----------------|
| 1 | Between `<a` and `class="wiki-link"` | ` href="/note/target" ` |
| 2 | Between `class="wiki-link"` and `data-raw="..."` | ` data-target="target" ` |
| 3 | The `data-raw` value | `Target` |
| 4 | Between `data-raw="..."` and `>` | ` data-alias=""` |
| 5 | Display text between `>` and `</a>` | `Target` |

**The reconstruction (current, buggy):**

```go
prefix := sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
```

This produces: ` href="/note/target" class="wiki-link" data-target="target" data-raw="Target" data-alias="">`

**Missing:** The literal `<a` at the beginning! Group 1 captures everything *after* `<a`, so the reconstruction needs to prepend `<a`.

**The fix** is simply:

```go
prefix := `<a` + sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
```

### Stage 4: HTML → JSON API response (Vault → API handler)

**File:** `backend/internal/vault/vault.go` — `Note` struct, `rebuildHTML()`
**File:** `backend/internal/api/api.go` — `getNote` handler

**What happens:** The fully resolved, post-processed HTML is stored in `Note.HTML`. When the frontend requests `GET /api/notes/{slug}`, the API handler serializes the entire `Note` struct as JSON. The `html` field contains the final HTML string.

**No additional transformation** happens at this stage. The HTML that the vault stores is exactly what the frontend receives.

### Stage 5: JSON → DOM rendering (Frontend NoteRenderer)

**File:** `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`
**File:** `web/src/lib/wikiLinks.ts`

**What happens in the frontend:**

1. The `NoteRenderer` component receives the `note.html` string from the API.
2. `resolveWikiLinks()` post-processes the HTML one more time using `DOMParser`:
   - It queries all `a.wiki-link` elements
   - If a link's `data-target` is not in the slug set, it adds class `"broken"` and sets `href="#"`
3. The processed HTML is injected into the DOM via `dangerouslySetInnerHTML`.
4. A click handler intercepts clicks on `.wiki-link` elements for SPA navigation — instead of full page navigation, it calls `onNavigate(slug)` which updates the React Router state.

**Why the frontend step matters for this bug:** If the backend produces HTML with a missing `<a` tag, the frontend's `DOMParser` will NOT find any `a.wiki-link` elements (because there is no `<a>` element — just bare text starting with `href=...`). The broken-link detection will not run, and the click handler will not intercept clicks. The link is completely non-functional.

## Data flow diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                     RAW MARKDOWN SOURCE                            │
│  [[ARTICLE - DMETA Design System Factory]] in a table cell         │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  STAGE 1: replaceWikiLinks() — Pre-processing                      │
│  Regex: (!?)\[\[([^\[\]]+)\]\]                                     │
│  Output: <a href="/note/article-dmeta-..." class="wiki-link"       │
│          data-target="article-dmeta-..." data-raw="ARTICLE -       │
│          DMETA Design System Factory" data-alias="">ARTICLE -       │
│          DMETA Design System Factory</a>                            │
│  ★ At this point, the HTML is CORRECT                              │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  STAGE 2: Goldmark conversion                                      │
│  GFM table renderer wraps anchor in <td>                           │
│  Output: <td><a href="..." class="wiki-link" ...>ARTICLE...</a>    │
│         </td><td>2026-05-23</td><td>Description</td>              │
│  ★ At this point, the HTML is STILL CORRECT                        │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  STAGE 3: rebuildHTML() → ReplaceWikiLinkDisplay()                 │
│  ★ BUG: Regex reconstruction drops the "<a" prefix                │
│  Output: <td> href="/note/..." class="wiki-link" ...>Resolved     │
│          Title</a></td><td>2026-05-23</td>                         │
│  ✗ The <a> tag is GONE — "href=..." is raw text                    │
│  ✗ Browser renders "href=...Resolved Title 2026-05-23 ..."        │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  STAGE 4: JSON serialization /api/notes/{slug}                     │
│  No transformation — bug is frozen into the response                │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  STAGE 5: Frontend NoteRenderer + wikiLinks.ts                     │
│  DOMParser finds NO <a class="wiki-link"> elements (they're        │
│  broken HTML). resolveWikiLinks() does nothing. Click handler      │
│  does nothing. User sees raw "href=...Title Date Description"      │
└─────────────────────────────────────────────────────────────────────┘
```

## Key files reference

| File | Purpose | Lines of interest |
|------|---------|-------------------|
| `backend/internal/parser/parser.go` | Markdown→HTML parser, wiki-link regex, slugify, `ReplaceWikiLinkDisplay` | `wikiLinkHTML` (L~130), `ReplaceWikiLinkDisplay` (L~175), `slugify` (L~140) |
| `backend/internal/parser/parser_test.go` | Parser unit tests | All tests |
| `backend/internal/vault/vault.go` | Vault loader, wiki-link index, `rebuildHTML()`, `ResolveWikiLink` | `buildWikiLinkIndex` (L~120), `rebuildHTML` (L~160) |
| `backend/internal/vault/vault_test.go` | Vault integration tests | — |
| `backend/internal/api/api.go` | HTTP API handlers, JSON serialization | `getNote` (L~100) |
| `web/src/lib/wikiLinks.ts` | Frontend wiki-link post-processor | `resolveWikiLinks` (L~18) |
| `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` | Note renderer component, click handler | `handleClick` (L~50) |
| `web/src/vault/staticVault.ts` | Static/demo mode wiki-link handling (marked extension) | `makeWikiLinkExtension` (L~25) |
| `web/src/index.css` | Wiki-link CSS styles | `a.wiki-link` (L~400) |
| `web/src/types/index.ts` | TypeScript type definitions | `WikiLinkRef`, `Note` |

## The bug in detail

### The `ReplaceWikiLinkDisplay` function

This function is called by `vault.rebuildHTML()` for every note after the wiki-link index is built. Its job is to replace the raw display text of non-aliased wiki links with the resolved note's title.

**Example:**

- Input HTML: `<a href="/note/article-dmeta" class="wiki-link" data-target="article-dmeta" data-raw="ARTICLE - DMETA" data-alias="">ARTICLE - DMETA</a>`
- Title resolver returns: `"DMETA: The Full Title"`
- Expected output: `<a href="/note/article-dmeta" class="wiki-link" data-target="article-dmeta" data-raw="ARTICLE - DMETA" data-alias="">DMETA: The Full Title</a>`
- Actual output: ` href="/note/article-dmeta" class="wiki-link" data-target="article-dmeta" data-raw="ARTICLE - DMETA" data-alias="">DMETA: The Full Title</a>`

**The missing `<a` prefix** causes the browser to treat the entire `href=...` sequence as text content. In a table, this text content fills the `<td>` and visually bleeds into what should be the next cell.

### Why the table context matters

When a wiki link is inside a table, the surrounding `<td>` tags are already part of the HTML. The regex in `ReplaceWikiLinkDisplay` matches across element boundaries because it's matching the entire `<a ...>...</a>` tag. The `([^<]*?)` group for the display text only works correctly when the `<a>` tag is well-formed — if the `<a` is missing, the regex engine's behavior becomes unpredictable.

However, the **primary bug** is simply the missing `<a` prefix. Once that's fixed, the display text replacement will work correctly even inside tables, because the regex correctly matches `>` to `</a>` which bounds the anchor content.

### Edge cases to consider

1. **Wiki links with explicit aliases:** `[[Target|Custom Alias]]` — these have `data-alias="Custom Alias"` and should NOT have their display text replaced. The current code correctly checks for non-empty `data-alias` before replacing.

2. **Wiki links with headings:** `[[Target#Section]]` — these produce an `href` with a fragment (`/note/target#section`). The `ReplaceWikiLinksString` function correctly preserves fragments. Display replacement only affects the text content, not the URL.

3. **Broken wiki links:** Links that don't resolve to any note. The title resolver returns `""`, and the function skips replacement (returns `match` unchanged). The frontend's `resolveWikiLinks` adds the `broken` class.

4. **Wiki links inside frontmatter:** Protected by `splitFrontmatter` — never reach the replacement pipeline.

5. **Embed links:** `![[Target]]` — rendered as `<div class="wiki-embed">`, not `<a>`. Not affected by `ReplaceWikiLinkDisplay`.

## Proposed fix

### Fix 1: Add missing `<a` prefix in `ReplaceWikiLinkDisplay`

**File:** `backend/internal/parser/parser.go`
**Line:** The `prefix :=` assignment in `ReplaceWikiLinkDisplay`

Change:
```go
prefix := sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
```

To:
```go
prefix := `<a` + sub[1] + `class="wiki-link"` + sub[2] + `data-raw="` + sub[3] + `"` + sub[4] + ">"
```

This is a one-character fix (adding `<a` before `sub[1]`). The regex groups already capture everything *after* `<a`, so we just need to re-add the tag name.

### Fix 2: Consider simplifying the regex approach

The current regex-based HTML manipulation is fragile. A more robust approach would be to use a proper HTML parser (like `golang.org/x/net/html`) for `ReplaceWikiLinkDisplay`. However, this is a larger refactor and should be a separate task. For now, the one-character fix is sufficient and low-risk.

### Alternative: Avoid reconstructing the opening tag entirely

Instead of capturing and rebuilding the opening tag, we could:

1. Find the anchor element
2. Replace only the text between `>` and `</a>`
3. Leave the opening tag completely untouched

This avoids the reconstruction bug entirely. Pseudocode:

```
Pattern: (<a[^>]*?class="wiki-link"[^>]*?data-raw="[^"]*?"[^>]*?>)([^<]*?)(</a>)
For each match:
  If link has non-empty data-alias → skip
  Extract data-target → resolve title
  Replace group 2 only (the display text)
  Reassemble: group1 + newTitle + group3
```

This is cleaner because it never reconstructs the opening tag — it just replaces the content. But it requires restructuring the regex, which is more invasive than the one-character fix.

**Recommendation:** Apply Fix 1 (the one-character fix) now. Schedule Fix 2 (HTML parser refactor) for a future task if regex fragility causes more issues.

## Implementation plan

### Phase 1: Fix the bug (this ticket)

1. Add `<a` prefix to the `prefix` line in `ReplaceWikiLinkDisplay`.
2. Add regression tests:
   - Test that `ReplaceWikiLinkDisplay` preserves the `<a>` opening tag.
   - Test that display text replacement works correctly inside table HTML.
   - Test that aliased links are preserved unchanged.
3. Run existing test suite to verify no regressions.
4. Smoke test with the real vault.

### Phase 2: Future hardening (separate ticket)

1. Refactor `ReplaceWikiLinkDisplay` to use `golang.org/x/net/html` for proper HTML parsing.
2. Add fuzz testing for the wiki-link replacement pipeline.
3. Consider adding an HTML validation step after `rebuildHTML()`.

## Testing and validation strategy

### Unit tests (Go)

```go
// Test 1: ReplaceWikiLinkDisplay preserves <a> tag
func TestReplaceWikiLinkDisplayPreservesAnchorTag(t *testing.T) {
    html := `<td><a href="/note/target" class="wiki-link" data-target="target" data-raw="Target" data-alias="">Target</a></td>`
    got := ReplaceWikiLinkDisplay(html, func(slug string) string {
        return "Resolved Title"
    })
    if !strings.HasPrefix(got, "<td><a ") {
        t.Fatalf("Missing <a> tag, got: %s", got)
    }
    if !strings.Contains(got, ">Resolved Title</a>") {
        t.Fatalf("Display text not replaced, got: %s", got)
    }
}

// Test 2: Display replacement in table context
func TestReplaceWikiLinkDisplayInTable(t *testing.T) {
    html := `<table><tbody><tr><td><a href="/note/t" class="wiki-link" data-target="t" data-raw="T" data-alias="">T</a></td><td>Date</td><td>Desc</td></tr></tbody></table>`
    got := ReplaceWikiLinkDisplay(html, func(slug string) string {
        return "Full Title"
    })
    // Adjacent cell content must NOT be inside the anchor
    if strings.Contains(got, "Date</a>") || strings.Contains(got, "Desc</a>") {
        t.Fatalf("Table content bled into anchor: %s", got)
    }
    if !strings.Contains(got, ">Full Title</a>") {
        t.Fatalf("Display not replaced correctly: %s", got)
    }
}

// Test 3: Aliased links are preserved
func TestReplaceWikiLinkDisplayPreservesAlias(t *testing.T) {
    html := `<a href="/note/t" class="wiki-link" data-target="t" data-raw="T" data-alias="My Alias">My Alias</a>`
    got := ReplaceWikiLinkDisplay(html, func(slug string) string {
        return "Resolved Title"
    })
    if !strings.Contains(got, ">My Alias</a>") {
        t.Fatalf("Alias was overwritten: %s", got)
    }
}
```

### Smoke test (live server)

```bash
# 1. Build and start the server against the real vault
cd backend
go build -o retro-obsidian-publish ./cmd/retro-obsidian-publish
./retro-obsidian-publish serve --vault-dir ~/code/wesen/go-go-golems/go-go-parc --port 8090

# 2. Fetch a note with wiki links in tables
curl -s http://localhost:8090/api/notes/research/kb/tribal/dmeta-design-system-compiler-pipeline | jq -r '.html' | grep 'wiki-link' | head -5

# 3. Verify each wiki-link anchor has a proper <a> tag
# Expected: <a href="/note/..." class="wiki-link" data-target="..." ...>Title</a>
# NOT: href="/note/..." class="wiki-link" ...
```

## Risks, alternatives, and open questions

### Risks

- **Risk: The one-character fix might not be the only issue.** If there are other regex-based HTML manipulation functions with similar reconstruction bugs, they should be identified. A quick audit of the parser code shows `ReplaceWikiLinksString` does not reconstruct tags — it only modifies attribute values, so it's not affected.

- **Risk: The regex `([^>]*?)` for attributes between `<a` and `class="wiki-link"` might fail if attributes contain `>` characters.** This is unlikely in practice (HTML attribute values containing `>` would need to be entity-escaped), but a proper HTML parser would handle it.

### Alternatives considered

1. **Do display replacement during Stage 1 (pre-processing).** Instead of replacing wiki links with the raw target text and then fixing it later in `rebuildHTML()`, we could resolve the display text during the initial `wikiLinkHTML()` call. However, at that point we don't have the full vault index yet — notes are being parsed one at a time, and the target note's title isn't known until all notes are loaded.

2. **Use goldmark AST transformations instead of regex post-processing.** We could register a custom goldmark AST walker that modifies the text content of wiki-link nodes. This would be more robust but requires deeper integration with goldmark's AST model.

3. **Move display resolution to the frontend.** Instead of replacing display text in the backend, we could send the raw display text and let the frontend's `resolveWikiLinks()` handle title substitution using the note index. This would simplify the backend but add frontend complexity.

### Open questions

- Should `data-alias=""` (empty string) be treated the same as missing `data-alias`? Currently, `wikiLinkHTML` always sets `data-alias` to the alias (or empty string if no alias), and `ReplaceWikiLinkDisplay` checks for a non-empty alias. Empty-string `data-alias` is effectively "no alias", so the current behavior is correct.
- Should the frontend's `staticVault.ts` also get a `data-raw` and `data-alias` attribute? Currently the static vault's marked extension only emits `data-target` and `href`, not `data-raw` or `data-alias`. This means the static vault doesn't support display text replacement at all. This could be a future improvement.

## References

- Obsidian wiki link syntax: https://help.obsidian.md/Linking+notes+and+other+files
- Goldmark documentation: https://github.com/yuin/goldmark
- Go regexp package: https://pkg.go.dev/regexp
- `golang.org/x/net/html` for future HTML parser approach: https://pkg.go.dev/golang.org/x/net/html
