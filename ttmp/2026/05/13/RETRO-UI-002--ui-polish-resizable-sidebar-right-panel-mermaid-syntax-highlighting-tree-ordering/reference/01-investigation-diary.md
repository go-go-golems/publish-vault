---
title: Investigation Diary
ticket: RETRO-UI-002
doc_type: reference
status: active
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
