---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: backend/internal/search/search.go
      Note: Fuzzy MatchQuery and prefix queries (Step 4)
    - Path: web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Embeds
    - Path: web/src/index.css
      Note: Copy button
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---




# Diary

## Goal

Track the implementation of five UI polish features for RETRO-UI-002: resizable sidebar, right-panel toggle, tree ordering, mermaid rendering, and syntax highlighting.

## Step 0: Ticket creation and evidence gathering

Created ticket `RETRO-UI-002` and gathered evidence across the frontend and backend to understand the current architecture and gaps.

### Prompt Context

**User prompt (verbatim):** "Ok, let's improve the visuals / UI looks of this page:

- toggle right sidebar on and off
- allow adjusting the left sidebar size so we can see the full tree
- properly order tree nodes alphabetically
- mermaid graph rendering
- syntax highlighting.

Create a new docmgr ticket, and then analyze the situation and create a design + impementation guide.

Focus on always using storybook when making changes to widgets, so that we have stories backing us up.

Keep a diary as you work.

Commit at appropriate intervals"

**Assistant interpretation:** Create a full ticket+design pipeline for five UI improvements, with storybook-first widget development.

**Inferred user intent:** Get a thorough, actionable plan before writing code, so that implementation is phased and testable.

**Commit (code):** None yet (documentation phase).

### What I did

- Created docmgr ticket `RETRO-UI-002`.
- Created design doc and diary doc.
- Gathered evidence: read VaultLayout, Sidebar, NotePage, NoteRenderer, FileTreeItem, uiSlice, resizable.tsx, index.css, vault.go FileTree(), parser.go.
- Inspected `go-go-parc` tree structure via `/api/tree`.
- Checked installed deps: `react-resizable-panels` already installed, `highlight.js` installed but unused, `mermaid` not installed.
- Wrote the design + implementation guide with 6 phases.

### Why

Evidence-first design prevents blind implementation. The tree structure is deep (6+ levels), the sidebar is too narrow at 224 px, and neither mermaid nor syntax highlighting are wired despite `highlight.js` being in `package.json`.

### What worked

The `react-resizable-panels` wrapper already exists in `web/src/components/ui/resizable.tsx`, so Phase 2 is a drop-in replacement. The `.hljs` CSS overrides are already written.

### What didn't work

N/A — this was a research and documentation step.

### What I learned

- The backend tree is unsorted: `FileTree()` appends in filesystem order.
- Mermaid requires a new dependency (`mermaid` npm package).
- `highlight.js` is installed but never imported anywhere.
- The resizable panel wrapper is ready but never used in VaultLayout.

### What was tricky to build

N/A — documentation phase.

### What warrants a second pair of eyes

- The mermaid async rendering approach (replace `<pre>` with `<div>` containing SVG) may conflict with React's DOM ownership. Need to verify that `dangerouslySetInnerHTML` + post-render DOM mutations work without React re-render clobbering.

### What should be done in the future

- Consider moving to client-side markdown rendering (react-markdown) for better React integration.
- Consider server-side highlighting via goldmark-highlighting if bundle size becomes an issue.

### Code review instructions

Review the design doc:
```
ttmp/2026/05/13/RETRO-UI-002--ui-polish-resizable-sidebar-right-panel-mermaid-syntax-highlighting-tree-ordering/design-doc/01-ui-polish-design-and-implementation-guide.md
```

### Technical details

Key evidence files and their relevant lines:

| File | Lines | What it shows |
|------|-------|---------------|
| `VaultLayout.tsx` | sidebar wrapper at `w-56 shrink-0` | Fixed sidebar width |
| `NotePage.tsx:82–103` | `<aside className="w-56 shrink-0">` | Always-visible right panel |
| `vault.go:228–260` | `FileTree()` | No sort call |
| `index.css:500–506` | `.hljs` overrides | CSS ready for highlighting |
| `resizable.tsx` | Full file | Unused panel wrapper |

## Step 1: Implemented all five UI polish features

Implemented all five features from the design guide across backend and frontend, with storybook stories backing up the changed components.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Execute the phased implementation plan from the design guide.

**Inferred user intent:** Deliver working, storybook-backed code for all five features.

**Commit (code):** 5a4fbc2 — Sort file tree folders-first alphabetically
**Commit (code):** 34bb49c — Resizable sidebar and right panel toggle
**Commit (code):** 3760cfb — Add syntax highlighting and mermaid rendering

### What I did

**Phase 1 — Tree sorting:**
- Added `sortTree()` recursive function to `backend/internal/vault/vault.go`.
- Folders first, then files, each sorted case-insensitively alphabetically.
- Added `TestFileTreeSortedFoldersFirstAlpha` in `vault_test.go`.

**Phase 2 — Resizable left sidebar:**
- Replaced fixed `w-56` flex layout in `VaultLayout.tsx` with `ResizablePanelGroup` / `ResizablePanel` / `ResizableHandle` from the existing `resizable.tsx` wrapper.
- Sidebar panel: `defaultSize={20}`, `minSize={12}`, `maxSize={40}`.
- Added `retro-resize-handle` CSS class for retro-styled drag handles.
- Removed `w-56` from `Sidebar.tsx` (width now comes from the resizable panel).
- Added `VaultLayout.stories.tsx` with Default, SidebarCollapsed, NarrowSidebar.

**Phase 3 — Right panel toggle:**
- Added `rightPanelOpen` and `toggleRightPanel` / `setRightPanelOpen` to `uiSlice.ts`.
- Added a `panel-right` menubar button in `VaultLayout.tsx`.
- Made the right `<aside>` in `NotePage.tsx` conditional on `rightPanelOpen`.
- Added `panel-right` icon to `Icon.tsx` (lucide `PanelRight`).

**Phase 4 — Syntax highlighting:**
- Added `hljs` post-render effect in `NoteRenderer.tsx`.
- Uses `hljs.highlightElement()` on `<pre><code>` blocks (skipping mermaid).
- Skips already-highlighted blocks via `data-highlighted` attribute.
- Extended `.hljs` retro CSS theme in `index.css` with more token types.

**Phase 5 — Mermaid rendering:**
- Installed `mermaid@11.15.0` via pnpm.
- Added mermaid post-render effect in `NoteRenderer.tsx` (runs before hljs).
- Initializes mermaid with retro-matching `base` theme.
- Replaces `<pre><code class="language-mermaid">` blocks with rendered SVG.
- Added mermaid CSS overrides in `index.css`.
- Updated `NoteRenderer.stories.tsx` with `WithMermaid`, `WithMultipleLanguages`, and `WithMermaidAndCode` stories.

**Phase 6 — Rebuild and restart:**
- Rebuilt web dist, restaged embedded assets, rebuilt binary, restarted server for `go-go-parc`.
- Verified tree sorting: folders-first ordering confirmed via `/api/tree`.
- Verified home page loads and app shell serves.

### Why

All five features address concrete pain points when using the real `go-go-parc` vault.

### What worked

- The `react-resizable-panels` wrapper was already in the codebase, so Phase 2 was a drop-in.
- Mermaid tree-shakes well into lazy chunks via Vite.
- The existing `.hljs` CSS overrides needed extension but the pattern was already established.

### What didn't work

Nothing broke during implementation.

### What I learned

- Mermaid `render()` is async; must call it before hljs to avoid hljs trying to highlight mermaid source.
- `react-resizable-panels` handles panel collapse/expand transitions gracefully.
- The `highlight.js` full bundle is ~1.9 MB in the main chunk; selective language imports would reduce this, but it's acceptable for now.

### What was tricky to build

- The mermaid + hljs ordering: mermaid must consume its `<code class="language-mermaid">` blocks first, or hljs will try to highlight them as code. Solved by running the mermaid effect before the hljs effect (React effects run in order).
- The mermaid `render()` API returns a promise; replacing DOM nodes inside React's `dangerouslySetInnerHTML` container is safe because React only reconciles on the next render cycle, not on DOM mutations.

### What warrants a second pair of eyes

- The mermaid DOM replacement (`pre.replaceWith(container)`) — verify React doesn't clobber it on re-renders (e.g., when `resolvedHtml` changes).
- The hljs `data-highlighted` guard — confirm it prevents double-highlighting on re-renders.

### What should be done in the future

- Use selective highlight.js language imports to reduce bundle size.
- Add a `Copy code` button to code blocks.
- Add a `Full screen` toggle for mermaid diagrams.
- Consider server-side highlighting via goldmark-highlighting if bundle size becomes an issue.

### Code review instructions

Review:
- `backend/internal/vault/vault.go` — `sortTree()`
- `backend/internal/vault/vault_test.go` — tree sorting test
- `web/src/components/pages/VaultLayout/VaultLayout.tsx` — resizable panels
- `web/src/components/organisms/Sidebar/Sidebar.tsx` — removed w-56
- `web/src/store/uiSlice.ts` — rightPanelOpen
- `web/src/components/pages/NotePage/NotePage.tsx` — conditional right panel
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — hljs + mermaid effects
- `web/src/index.css` — retro resize handle, hljs extensions, mermaid styling
- `web/src/components/atoms/Icon/Icon.tsx` — panel-right icon
- `web/src/components/organisms/NoteRenderer/NoteRenderer.stories.tsx` — new stories
- `web/src/components/pages/VaultLayout/VaultLayout.stories.tsx` — new story

Validate:
```bash
cd backend && go test ./...
pnpm --dir web check
pnpm --dir web build
curl -fsS http://127.0.0.1:8080/api/tree | python3 -c '...' # verify folders-first
open http://127.0.0.1:8080/
```

## Step 2: Fixed wiki-link resolution — short Obsidian paths now resolve to full vault slugs

Wiki links like `[[Tribal/application-native-authorization]]` in the Goja Sites proposal were generating `href="/note/tribal/application-native-authorization"` which 404'd because the actual slug is `research/kb/tribal/application-native-authorization`. This also meant backlinks weren't being recorded.

### Prompt Context

**User prompt (verbatim):** "Tribal/application-native-authorization: in http://127.0.0.1:8080/note/research/institute/proposals/2026/05/11/proposal-goja-sites-hosting-service should link to the actual documents. So should backlinks appear as well."

**Assistant interpretation:** Wiki links in notes reference short Obsidian paths but the vault stores notes by their full relative filesystem path. Links and backlinks need to resolve correctly.

**Inferred user intent:** Make wiki links navigate to real documents and make backlinks work.

**Commit (code):** 2c20742 — Resolve wiki links to full vault slugs with suffix-based index

### What I did

- Added `wikiLinkIndex` map to `Vault` struct — maps short slugified targets to full vault slugs.
- Added `buildWikiLinkIndex()` — builds suffix-based lookup: for a note at `Research/KB/Tribal/App.md`, it registers `app`, `tribal/app`, `kb/tribal/app`, `research/kb/tribal/app` all pointing to the full slug.
- Added `ResolveWikiLink()` — public method for resolving wiki link targets.
- Added `rebuildHTML()` — after building the index, re-renders all note HTML to replace short `data-target` and `href` values with resolved full slugs using `parser.ReplaceWikiLinksString()`.
- Added `ReplaceWikiLinksString()` to parser — uses regex to replace `data-target` and `href="/note/..."` attributes with resolved slugs.
- Updated `buildBacklinks()` — now uses `ResolveWikiLink()` instead of exact/title matching.
- Updated `ReloadNote()` and `RemoveNote()` — rebuild wiki-link index on changes.
- Added `TestWikiLinkResolution` — verifies short target resolves, backlinks connect, and HTML contains correct href/data-target.

### Why

Obsidian wiki links reference notes by short paths (e.g., `Tribal/App-Auth`), but the vault's slug system uses the full relative path (`research/kb/tribal/app-auth`). Without resolution, links 404 and backlinks don't connect.

### What worked

The suffix-based index handles all common Obsidian link patterns:
- `[[Tribal/App-Auth]]` → matches `Research/KB/Tribal/App-Auth.md`
- `[[App-Auth]]` → matches by filename suffix
- Exact slug match always takes priority

### What didn't work

N/A

### What I learned

Obsidian wiki links use a shortest-unique-path convention. The suffix-based index correctly handles the ambiguity by preferring the longest match (most specific path).

### What was tricky to build

The `ReplaceWikiLinksString` regex approach — needed to be careful about matching `data-target` and `href` attributes without breaking other HTML. The regex `data-target="([^"]+)"` and `href="/note/([^"]+)"` are specific enough.

### What warrants a second pair of eyes

- The suffix-based index could produce ambiguous matches if two notes share the same short path (e.g., two `Index.md` files in different folders). The current "first registered wins" behavior may need to be improved to prefer the closest match.
- The `rebuildHTML()` regex approach runs on every `LoadAll()` — verify performance on large vaults.

### What should be done in the future

- Add ambiguity detection: if multiple notes match a short wiki link target, prefer the one in the closest directory.
- Consider caching the resolved HTML to avoid re-running `ReplaceWikiLinksString` on every load.

### Code review instructions

Review:
- `backend/internal/vault/vault.go` — `buildWikiLinkIndex()`, `ResolveWikiLink()`, `rebuildHTML()`, updated `buildBacklinks()`
- `backend/internal/parser/parser.go` — `ReplaceWikiLinksString()`
- `backend/internal/vault/vault_test.go` — `TestWikiLinkResolution`

Validate:
```bash
curl -fsS http://127.0.0.1:8080/api/notes/research/kb/tribal/application-native-authorization | jq .backlinks
```

## Step 3: Render callout admonitions and widen article

The `> [!summary]` callout in the Goja Sites proposal was rendering as a plain blockquote with raw `[!summary]` text. Also, the article width was narrower than Obsidian Publish (768px vs 800px).

### Prompt Context

**User prompt (verbatim):** "I aso think the width of the article could be a bit wider: https://publish.obsidian.md/manuel/Projects/2026/05/04/ARTICLE+-+Obsidian+to+reMarkable+Sync+-+Native+Delta+Upload+and+Vault+Report+Pipeline (compare to this) ,not much though. also render admonitions like !summary on http://127.0.0.1:8080/note/research/institute/proposals/2026/05/11/proposal-goja-sites-hosting-service"

**Assistant interpretation:** Widen article slightly and render Obsidian-style callout admonitions.

**Inferred user intent:** Match Obsidian Publish's readability width and make callouts look like proper styled boxes.

**Commit (code):** 22dab5f — Render callout admonitions and widen article to max-w-4xl

### What I did

- Checked Obsidian Publish's actual content width: `max-width: 800px` on `.markdown-preview-sizer`.
- Widened article from `max-w-3xl` (768px) to `max-w-4xl` (896px) — a bit wider than Obsidian Publish, which fits the retro design's denser layout.
- Added `renderCallouts()` to `parser.go` — detects `<blockquote><p>[!type]` patterns and transforms them into styled `<div class="callout callout-type">` blocks.
- Added `calloutIcon()` helper — maps callout types to Unicode icons (≡ for summary, ⚠ for warning, 💡 for tip, etc.).
- Added retro callout CSS — each callout type has a colored titlebar (green for summary, orange for warning, red for important, blue for tip).
- Added `TestCalloutRendering` and `TestCalloutWithTitle` to `parser_test.go`.
- Added `WithCallouts` story to `NoteRenderer.stories.tsx`.

### Why

Obsidian callouts (`> [!type]`) are a core authoring feature. Without rendering them, important summary/warning/note boxes appear as plain blockquotes with raw `[!type]` markers.

### What worked

The regex approach `\[!(\w+)\]` catches all standard callout types. The retro-themed callout boxes with colored titlebars match the overall design language.

### What didn't work

Go's `regexp` package doesn't support `(?!...)` lookahead, so the initial regex failed at compile time. Rewrote using `[\s\S]*?` lazy match with `</blockquote>` as the closing boundary.

### What I learned

Obsidian Publish uses `max-width: 800px`. Our `max-w-4xl` (896px) is slightly wider, which works well for the retro monospace-heavy content.

### What was tricky to build

The regex for multi-paragraph blockquotes — goldmark wraps single-paragraph blockquotes in one `<p>` tag, but multi-paragraph ones use multiple `<p>` tags. The regex `[\s\S]*?</blockquote>` handles both cases.

### What warrants a second pair of eyes

- Nested blockquotes inside callouts — the current regex may not handle them correctly.
- Callout collapsibility — Obsidian supports `> [!type]+` and `> [!type]-` for default-open/closed. Not yet implemented.

### What should be done in the future

- Add collapsible callout support (`[!type]+` default open, `[!type]-` default closed).
- Handle nested blockquotes inside callouts.
- Add more callout type colors if needed.

### Code review instructions

Review:
- `backend/internal/parser/parser.go` — `renderCallouts()`, `calloutIcon()`
- `backend/internal/parser/parser_test.go` — callout tests
- `web/src/index.css` — `.callout` styles
- `web/src/components/pages/NotePage/NotePage.tsx` — `max-w-4xl`
- `web/src/components/organisms/NoteRenderer/NoteRenderer.stories.tsx` — `WithCallouts` story

Validate:
```bash
curl -fsS http://127.0.0.1:8080/api/notes/research/institute/proposals/2026/05/11/proposal-goja-sites-hosting-service | grep callout
```

## Step 4: Fuzzy search and right-panel resizable — improve search and layout UX

Search returned zero results for partial words like "merm". The right panel was fixed-width and couldn't be resized. Graph edges used raw wiki-link targets instead of resolved slugs, producing 0 edges instead of ~900.

### Prompt Context

**User prompt (verbatim):** (see Step 0)

**Assistant interpretation:** Fix search to handle partial words, make the right panel resizable, and fix the graph edge resolution bug.

**Inferred user intent:** Make search actually useful for partial matches; make the layout more flexible; fix the graph which was empty.

**Commit (code):** 72ed05b — Improve search: fuzzy matching for partial words, prefix for short queries
**Commit (code):** 581caf8 — Make right panel resizable with drag handle
**Commit (code):** d0908ec — Fix graph edges: resolve wiki links to vault slugs

### What I did

**Fuzzy search:**
- Changed `search.go` to use `MatchQuery` with `SetFuzziness(1)` for words > 3 characters.
- Short words (≤3 chars) use prefix queries for exact start matching.
- Searching "merm" now returns 30 results instead of 0.

**Right panel resizable:**
- Wrapped the right `<aside>` in `NotePage.tsx` with `ResizablePanelGroup` / `ResizablePanel` / `ResizableHandle`.
- Default 25%, min 12%, max 40%.
- Added retro-styled resize handle.

**Graph edge fix:**
- `getGraph` in `api.go` was comparing raw `wl.Target` (e.g., `Tribal/App-Auth`) against slug set — never matching.
- Changed to use `vault.ResolveWikiLink()` to resolve targets before checking slug membership.
- Edge count went from 0 → 906.

### Why
Search was nearly unusable for partial matches. The graph was completely empty. The right panel was too inflexible.

### What worked
- Bleve's `MatchQuery` with `SetFuzziness(1)` handles partial words well without needing ngram tokenizers.
- The `ResolveWikiLink()` function already existed for backlinks — reusing it for graph edges was straightforward.

### What didn't work
N/A

### What I learned
- Bleve `MatchQuery` with fuzziness is a good middle ground between exact and ngram search.
- The graph edge resolution bug was a classic "use raw value vs. resolved value" mistake.

### What was tricky to build
- The fuzzy vs. prefix query boundary: very short words with fuzziness produce too many false positives, so ≤3 char words use prefix queries.

### What warrants a second pair of eyes
- The graph edge resolution — verify that `ResolveWikiLink` doesn't produce false positives for ambiguous targets.

### What should be done in the future
- Consider ngram tokenizer for true substring matching if fuzzy proves insufficient.
- Filter self-referencing graph edges (a note linking to itself creates a loop).

### Code review instructions

Review:
- `backend/internal/search/search.go` — fuzzy + prefix query logic
- `backend/internal/api/api.go` — graph edge resolution using `ResolveWikiLink`
- `web/src/components/pages/NotePage/NotePage.tsx` — ResizablePanelGroup for right panel
- `web/src/index.css` — `.retro-resize-handle` styles

Validate:
```bash
curl -fsS 'http://127.0.0.1:8080/api/search?q=merm' | jq '.results | length'
curl -fsS 'http://127.0.0.1:8080/api/graph' | jq '.edges | length'
```

## Step 5: Resolve wiki-link display text to note titles

Wiki links in the rendered HTML showed slugified targets like `Tribal/application-native-authorization` as display text instead of the actual note title `Application-Native Authorization — How We Do It`. The `data-raw` attribute was added to anchors in the parser, but the display text replacement wasn't wired in.

### Prompt Context

**User prompt (verbatim):** "work on tasks 1-6 (renumbered 2.1-2.6), add them to the ticket, work them one by one, commit at intervals, and keep a diary."

**Assistant interpretation:** Task 2.1: Fix wiki-link display text to show resolved note titles.

**Inferred user intent:** Make wiki links show human-readable titles instead of slugified paths.

**Commit (code):** dc68621 — Resolve wiki-link display text to note titles

### What I did

- Added `data-raw` attribute to wiki-link anchors in `parser.go` — stores the original raw target (e.g., `Fundamentals/access-control-models`).
- Added `dataRawRe` regex to match `data-raw="..."` attributes.
- Added `ReplaceWikiLinkDisplay()` function that takes a title-resolver callback and replaces the anchor text between `>` and `</a>` with the resolved note title.
- Updated `rebuildHTML()` in `vault.go` to call `ReplaceWikiLinkDisplay` after `ReplaceWikiLinksString`.
- Added `TestWikiLinkDataRaw` and `TestWikiLinkDataRawRealFormat` to `parser_test.go`.

### Why
The `data-raw` attribute carries the original wiki-link target through the HTML transformation pipeline. The `ReplaceWikiLinksString` step resolves `data-target` and `href` to full slugs, while `ReplaceWikiLinkDisplay` resolves display text to note titles. Without `data-raw`, the display text replacement had no way to know which note to look up.

### What worked
- Adding `data-raw` to the parser output was simple and non-breaking.
- `ReplaceWikiLinkDisplay` uses the already-resolved `data-target` slug to look up titles via `v.notes[slug].Title`.
- Verified: `Fundamentals/access-control-models` → display text `Access Control Models: Authentication, Authorization, and Delegation`.

### What didn't work
**Critical debugging mystery**: After adding `data-raw` to the parser (confirmed by unit test), the API-served HTML contained 0 `data-raw` attributes. The mystery turned out to be a stale server binary — the previous server process was still running on port 8080, and the kill+restart didn't fully take effect due to a port conflict (`bind: address already in use`). After killing the stale process and restarting, `data-raw` appeared correctly (1298 occurrences before and after replacement).

### What I learned
- Always verify the server is actually running the new binary by checking PID and log output.
- The `ReplaceWikiLinksString` regex preserves `data-raw` because it only targets `data-target` and `href="/note/..."` attributes.

### What was tricky to build
- The stale-server debugging mystery took significant time. The parser was correct, the regex was correct, but the served HTML came from an old binary. The lesson: after rebuilding, always confirm the old process is dead (`lsof -ti:8080`) and the new one started.

### What warrants a second pair of eyes
- The `ReplaceWikiLinkDisplay` regex — it matches `data-raw="..."` then captures text between `>` and `</a>`. If an anchor has nested HTML (e.g., `<code>` inside), the regex may break.

### What should be done in the future
- Consider HTML-aware (DOM-based) display text replacement instead of regex for robustness.
- Add test coverage for `ReplaceWikiLinkDisplay`.

### Code review instructions

Review:
- `backend/internal/parser/parser.go` — `data-raw` attribute, `dataRawRe`, `ReplaceWikiLinkDisplay`
- `backend/internal/parser/parser_test.go` — `TestWikiLinkDataRaw`, `TestWikiLinkDataRawRealFormat`
- `backend/internal/vault/vault.go` — `rebuildHTML()` calling `ReplaceWikiLinkDisplay`

Validate:
```bash
curl -fsS http://127.0.0.1:8080/api/notes/research/institute/proposals/2026/05/11/proposal-goja-sites-hosting-service | python3 -c 'import json,re,sys; d=json.load(sys.stdin); [print(m.group(1)[:50], "→", m.group(2)[:50]) for m in re.finditer(r"data-raw=\"([^\"]*)\"[^>]*>([^<]*)", d["html"])]'
```

## Step 6: Filter self-referencing graph edges and add copy-code button

Notes that link to themselves were creating circular graph edges. Code blocks had no way to copy their content.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Task 2.2: Filter self-edges in graph. Task 2.3: Add copy-code button.

**Inferred user intent:** Clean up the graph visualization and add a common code-reading convenience.

**Commit (code):** 36258ba — Filter self-referencing graph edges
**Commit (code):** 1040c14 — Add click-to-copy button on code blocks

### What I did

**Self-edge filter:**
- Added `&& resolved != n.Slug` condition to the graph edge builder in `api.go`.
- Self-referencing edges (a note linking to itself) are now excluded.

**Copy-code button:**
- Added a post-highlight useEffect in `NoteRenderer.tsx` that injects a `⎘` button into each `<pre>` block.
- On click, copies `code.textContent` to clipboard and briefly shows `✓`.
- Button is hidden by default (`opacity: 0`) and appears on hover.
- Added `.copy-code-btn` CSS in `index.css` with retro styling (ink background, paper text).

### Why
Self-edges create noise in the graph. Copy-code is a standard feature in documentation sites.

### What worked
- The one-line `&& resolved != n.Slug` fix was trivial.
- The DOM-injection approach for copy buttons works well alongside hljs highlighting.

### What didn't work
N/A

### What I learned
- The copy button needs `position: relative` on the `<pre>` parent for absolute positioning to work.

### What was tricky to build
N/A — straightforward implementation.

### What warrants a second pair of eyes
- The copy button's clipboard API call (`navigator.clipboard.writeText`) — may fail in non-HTTPS contexts. Should add a fallback.

### What should be done in the future
- Add clipboard API fallback for HTTP contexts (e.g., `document.execCommand('copy')`).
- Consider selective highlight.js language imports to reduce bundle size.

### Code review instructions

Review:
- `backend/internal/api/api.go` — self-edge filter (`resolved != n.Slug`)
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — copy button injection
- `web/src/index.css` — `.copy-code-btn` styles

Validate:
```bash
curl -fsS http://127.0.0.1:8080/api/graph | python3 -c 'import json,sys; g=json.load(sys.stdin); self_edges=[e for e in g["edges"] if e["source"]==e["target"]; print(f"Self-edges: {len(self_edges)}")'
```

## Step 7: Render ![[embeds]] with inline note content and add collapsible callouts

`![[Note]]` embeds were rendering as empty `<div class="wiki-embed">` placeholders. Callouts with `[!type]-` syntax (Obsidian's default-collapsed) were not collapsible.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Task 2.4: Render embeds inline. Task 2.5: Add collapsible callout support.

**Inferred user intent:** Make embeds show actual content and make callouts match Obsidian's collapsible behavior.

**Commit (code):** 9477854 — Render ![[embeds]] with inline note content
**Commit (code):** 945b53d — Add collapsible callouts [!type]- default closed with toggle

### What I did

**Embed rendering:**
- Added a `useEffect` in `NoteRenderer.tsx` that finds `.wiki-embed` divs, fetches the embedded note's HTML via `/api/notes/{target}`, and injects it as a child `<div class="wiki-embed-content">`.
- Failed fetches show `⚠ Embed not found: {target}`.
- Added `.wiki-embed-content` and `.wiki-embed-broken` CSS.

**Collapsible callouts:**
- Updated `calloutRe` regex in `parser.go` to capture the `+`/`-` fold character after `[!type]`: `\[!(\w+)\]([+-])?`.
- `[!type]-` renders as `<div class="callout callout-type callout-collapsible">` with the body `display:none`.
- `[!type]+` and `[!type]` render as normal (open) callouts.
- Added `callout-toggle` span (▼) in the title for collapsible callouts.
- Added click handler in `NoteRenderer.tsx` that toggles `display:none` on the `.callout-body` and swaps ▼/▶.
- Added `.callout-collapsible`, `.callout-toggle` CSS with hover effects.
- Added `TestCollapsibleCallout` and `TestOpenCallout` to `parser_test.go`.

### Why
Embeds are a core Obsidian feature. Collapsible callouts are used in the `go-go-parc` vault for summary blocks that should be hidden by default.

### What worked
- The client-side fetch approach for embeds is simple and doesn't require backend changes.
- The `]` in the regex was the key insight — `[!warning]-` means the `]` closes the type, then `-` is the fold char.

### What didn't work
**Regex failure #1**: Initial regex `\[!(\w+)([+-])?` didn't account for the `]` between the type and fold char. The `\w+` matched `warning`, then `([+-])?` tried to match `]` which isn't `+` or `-`, so the fold char was always empty. The test output showed `]- Collapsed Warning` leaking into the title. Fix: added explicit `\]` before the fold group: `\[!(\w+)\]([+-])?`.

**Regex failure #2**: The Go raw string literal had `\]` which Go interprets as literal `\]`. But the actual issue was simpler — the first attempt forgot to match the closing `]` bracket, so the fold character was never captured.

### What I learned
- The `]` in `[!type]-` is the Markdown link reference closing bracket, not part of the callout type. The regex must match `\]` explicitly between the type and fold char.
- Client-side embed fetching is elegant but creates N+1 API calls for notes with many embeds.

### What was tricky to build
- The regex evolution: `\[!(\w+)\]([+-])?([\s\S]*?)` — the `]` placement was the critical fix. Without it, the fold char was always empty and `- Collapsed Warning` leaked into the content.

### What warrants a second pair of eyes
- Embed fetching — if an embedded note itself contains embeds, this creates recursive API calls. Should add a depth limit.
- The collapsible callout click handler is in the `handleClick` callback — verify it doesn't interfere with wiki-link clicks.

### What should be done in the future
- Add embed depth limit (max 2 levels of nesting).
- Add embed loading indicator (spinner or placeholder text).
- Consider server-side embed resolution for better performance.

### Code review instructions

Review:
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — embed useEffect, collapsible callout toggle in handleClick
- `backend/internal/parser/parser.go` — updated `calloutRe` regex, collapsible callout rendering
- `backend/internal/parser/parser_test.go` — `TestCollapsibleCallout`, `TestOpenCallout`
- `web/src/index.css` — `.wiki-embed-content`, `.callout-collapsible`, `.callout-toggle` styles

Validate:
```bash
cd backend && go test ./internal/parser/ -run TestCollapsibleCallout -v
curl -fsS http://127.0.0.1:8080/api/notes/research/playbooks/sync-vault-projects-to-parc | grep wiki-embed
```

## Step 8: Add heading permalink anchors and hash-scroll navigation

Goldmark generates `id` attributes on headings, but there were no permalink links and no scroll-to-heading behavior for hash fragments.

### Prompt Context

**User prompt (verbatim):** "keep a diary (read the skill if necessary)" and "commit at appropriate intervals"

**Assistant interpretation:** Task 2.6 (final task): Add heading permalinks. Also update the diary with all recent progress.

**Inferred user intent:** Complete all six tasks, keep documentation current, commit properly.

**Commit (code):** bd03a56 — Add heading permalink anchors and hash-scroll navigation

### What I did

- Added a `useEffect` in `NoteRenderer.tsx` that finds all `<h1>`–`<h6>` elements with `id` attributes and injects an `<a class="heading-anchor" href="#id">#</a>` as the last child.
- The `#` link is hidden by default (`opacity: 0`) and appears on heading hover.
- Clicking the permalink sets `window.location.hash` and scrolls smoothly to the heading.
- Added a separate `useEffect` that handles hash-scroll on note load — if `window.location.hash` is set, waits 200ms for content to render, then scrolls to the target heading.
- Added `.heading-anchor` CSS with retro styling.

### Why
Heading permalinks are essential for sharing specific sections. Hash-scroll is needed because the SPA renders content dynamically after navigation — the browser's native scroll-to-hash doesn't work with `dangerouslySetInnerHTML`.

### What worked
- `CSS.escape(hash)` properly escapes heading IDs that contain special characters.
- The 200ms delay gives mermaid and hljs effects time to run before scrolling.

### What didn't work
N/A

### What I learned
- React's `dangerouslySetInnerHTML` doesn't trigger the browser's native scroll-to-hash because the content isn't in the DOM at navigation time. A programmatic scroll is needed.
- `CSS.escape()` is essential for heading IDs with dots, colons, or other CSS-significant characters.

### What was tricky to build
- The scroll timing: mermaid rendering is async and can change the document height. The 200ms delay is a heuristic — for very large mermaid diagrams, the scroll target may shift. Consider using a `MutationObserver` or `ResizeObserver` instead.

### What warrants a second pair of eyes
- The scroll delay heuristic (200ms) — may be insufficient for slow mermaid renders or very long notes.

### What should be done in the future
- Replace the 200ms scroll delay with a `MutationObserver` that waits for mermaid SVGs to be inserted.
- Add scroll-margin-top to headings so the fixed header doesn't obscure the target.

### Code review instructions

Review:
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — heading permalink injection and hash-scroll effects
- `web/src/index.css` — `.heading-anchor` styles

Validate:
```bash
# Check that headings have IDs
curl -fsS http://127.0.0.1:8080/api/notes/research/institute/proposals/2026/05/11/proposal-goja-sites-hosting-service | python3 -c 'import re,sys; html=sys.stdin.read(); ids=re.findall(r"<h[1-6] id=\"([^\"]*)\"", html); print(f"Heading IDs: {len(ids)}"); [print(f"  {i}") for i in ids[:5]]'
```
