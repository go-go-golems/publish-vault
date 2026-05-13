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
