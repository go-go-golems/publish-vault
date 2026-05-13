---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: backend/internal/vault/vault.go
      Note: FileTree sortTree()
    - Path: web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Mermaid and hljs hooks
    - Path: web/src/components/pages/NotePage/NotePage.tsx
      Note: Right panel toggle
    - Path: web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: Layout structure
    - Path: web/src/components/ui/resizable.tsx
      Note: Existing panel wrapper
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---




## 1. Executive Summary

The retro-obsidian-publish UI has five concrete gaps when rendering the real `go-go-parc` vault:

1. **No right-panel toggle** — the backlinks+graph aside is always visible, eating content width on smaller screens.
2. **Fixed left-sidebar width** — hardcoded `w-56` (224 px) cannot show deep tree paths like `Research/Institute/Research/2026/04/16/…`.
3. **Unordered tree** — the backend `FileTree()` inserts children in filesystem order with no sorting; folders and leaf notes interleave unpredictably.
4. **No mermaid rendering** — `go-go-parc` notes contain ` ```mermaid ` blocks; the goldmark backend emits them as raw `<pre><code class="language-mermaid">` text.
5. **No syntax highlighting** — `highlight.js` and `rehype-highlight` are in `package.json` but never wired; the backend emits raw `<pre><code class="language-*">` without language classes or highlighted tokens.

This guide proposes phased fixes for all five, backed by Storybook stories for every changed component.

---

## 2. Problem Statement and Scope

### 2.1 Right-panel toggle

**Current:** `NotePage.tsx:82–103` renders an `<aside className="w-56 shrink-0">` containing graph + backlinks unconditionally. There is no UI to hide it.

**Desired:** A toggle in the menubar (or a button on the panel itself) that collapses the right panel. State persisted in Redux `uiSlice`.

### 2.2 Resizable left sidebar

**Current:** `VaultLayout.tsx` wraps `Sidebar` with `{sidebarOpen && <Sidebar ... className="... w-56 shrink-0" />}`. The width is fixed.

**Desired:** Replace the fixed-width sidebar with `react-resizable-panels` (`ResizablePanelGroup` / `ResizablePanel` / `ResizableHandle`), which is already installed in `web/src/components/ui/resizable.tsx`. The sidebar panel should have a min/max size and persist its width in Redux.

### 2.3 Tree ordering

**Current:** `backend/internal/vault/vault.go:FileTree()` (lines 228–260) appends children to `parentNode.Children` in filesystem iteration order. No `sort.Slice` call exists.

**Desired:** Folders first, then files, each group sorted alphabetically (case-insensitive). Sorting should happen both in the backend (for the `/api/tree` endpoint) and as a client-side guard.

### 2.4 Mermaid graph rendering

**Current:** Goldmark emits mermaid code fences as `<pre><code class="language-mermaid">graph TD\nA-->B</code></pre>`. `NoteRenderer.tsx` injects this raw HTML via `dangerouslySetInnerHTML`. Nothing transforms mermaid blocks.

**Desired:** After the note HTML is set, scan for `<code class="language-mermaid">` blocks and replace each with an SVG rendered by the `mermaid` JS library. This should be a post-render effect in `NoteRenderer`.

### 2.5 Syntax highlighting

**Current:** The backend goldmark pipeline has no highlighting extension. The frontend has `highlight.js` (^11.11.1) installed but never imports it. `index.css:500–506` already contains `.hljs` retro theme overrides. `rehype-highlight` (^7.0.2) is also installed but unused.

**Desired:** Since the backend emits pre-rendered HTML consumed via `dangerouslySetInnerHTML`, the cleanest approach is client-side highlighting: after mount, `hljs.highlightAll()` or targeted `hljs.highlightElement()` on `<pre><code>` blocks. This avoids backend CGO dependencies and keeps the Go binary simple.

---

## 3. Current-State Architecture (Evidence)

### 3.1 Layout structure

```
VaultLayout.tsx
  ├── <header> retro-menubar
  ├── <div> flex body
  │   ├── {sidebarOpen && <Sidebar>}   ← w-56 fixed, no resize
  │   └── <main>                        ← ScrollArea
  │       └── {children}                ← NotePage, SearchPage, etc.
```

`NotePage.tsx` renders:
```
<div className="flex h-full">
  ├── <ScrollArea>          ← note content (NoteRenderer)
  └── <aside className="w-56 shrink-0">  ← always visible right panel
      ├── {graphVisible && <GraphView>}
      └── <BacklinksPanel>
```

### 3.2 Key files

| Component | File | Lines |
|-----------|------|-------|
| VaultLayout | `web/src/components/pages/VaultLayout/VaultLayout.tsx` | ~120 |
| Sidebar | `web/src/components/organisms/Sidebar/Sidebar.tsx` | ~70 |
| FileTreeItem | `web/src/components/molecules/FileTreeItem/FileTreeItem.tsx` | ~60 |
| NotePage | `web/src/components/pages/NotePage/NotePage.tsx` | ~113 |
| NoteRenderer | `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` | ~110 |
| BacklinksPanel | `web/src/components/organisms/BacklinksPanel/BacklinksPanel.tsx` | ~55 |
| GraphView | `web/src/components/organisms/GraphView/GraphView.tsx` | ~280 |
| uiSlice | `web/src/store/uiSlice.ts` | ~35 |
| Resizable UI | `web/src/components/ui/resizable.tsx` | ~54 |
| HLJS CSS | `web/src/index.css:500–506` | ~7 |
| Backend tree | `backend/internal/vault/vault.go:228–260` | ~33 |
| Backend parser | `backend/internal/parser/parser.go` | ~274 |

### 3.3 Dependencies already installed

| Package | Version | Used? |
|---------|---------|-------|
| `react-resizable-panels` | ^3.0.6 | No (ui/resizable.tsx wrapper exists but unused) |
| `highlight.js` | ^11.11.1 | No (CSS only) |
| `rehype-highlight` | ^7.0.2 | No |
| `mermaid` | — | Not installed |

### 3.4 Storybook coverage

Existing stories:

- `Sidebar.stories.tsx` — Default, Loading
- `NoteRenderer.stories.tsx` — Default (includes code block)
- `GraphView.stories.tsx` — exists
- `BacklinksPanel.stories.tsx` — exists
- `FileTreeItem.stories.tsx` — exists
- `FrontmatterPanel.stories.tsx` — exists

Missing stories (need creation):

- `NotePage.stories.tsx` — currently no story
- `VaultLayout.stories.tsx` — currently no story

---

## 4. Gap Analysis

| # | Gap | Root Cause | Impact |
|---|-----|-----------|--------|
| 1 | Right panel always visible | No toggle mechanism in uiSlice or menubar | Wastes ~224 px on small screens |
| 2 | Sidebar width fixed | `w-56` hardcoded in Sidebar.tsx:8 | Cannot read deep paths |
| 3 | Tree not sorted | `FileTree()` appends without `sort.Slice` | Folders/files interleaved arbitrarily |
| 4 | Mermaid not rendered | No mermaid library; no post-render transform | Diagrams show as raw text |
| 5 | Code not highlighted | highlight.js imported nowhere; no hljs initialization | Code blocks are plain monospace |

---

## 5. Proposed Architecture and APIs

### 5.1 Right-panel toggle

**Redux state change** (`uiSlice.ts`):
```ts
interface UIState {
  // ... existing
  rightPanelOpen: boolean;   // NEW — default true
}
// Add: toggleRightPanel, setRightPanelOpen
```

**Menubar button** (`VaultLayout.tsx`):
Add a menubar item after the existing "Search" button:
```tsx
<button
  type="button"
  className="retro-menubar-item"
  onClick={() => dispatch(toggleRightPanel())}
  title="Toggle right panel"
>
  <Icon name="panel-right" size={13} />
</button>
```

**NotePage conditional** (`NotePage.tsx`):
```tsx
const rightPanelOpen = useAppSelector((s) => s.ui.rightPanelOpen);
// ...
{rightPanelOpen && <aside>...</aside>}
```

### 5.2 Resizable left sidebar

Replace the `VaultLayout` body from:
```tsx
{sidebarOpen && <Sidebar className="w-56 shrink-0" ... />}
<main className="flex-1">...</main>
```

To:
```tsx
<ResizablePanelGroup direction="horizontal">
  {sidebarOpen && (
    <ResizablePanel defaultSize={20} minSize={12} maxSize={35} order={1}>
      <Sidebar ... />
    </ResizablePanel>
  )}
  <ResizablePanel defaultSize={80} order={2}>
    <main>...</main>
  </ResizablePanel>
  {sidebarOpen && <ResizableHandle withHandle />}
</ResizablePanelGroup>
```

The `ResizableHandle` only appears when the sidebar is open, letting the user drag to resize. The sidebar width persists per-session via the panel layout (no extra Redux key needed; `react-resizable-panels` handles its own layout state).

**Styling the handle:** Override the default `resizable.tsx` handle to match the retro design language (1 px black border, no rounded corners, small grip dots).

### 5.3 Tree ordering (backend)

In `backend/internal/vault/vault.go:FileTree()`, add a recursive sort after building the tree:

```go
func sortTree(node *FileNode) {
    if node == nil {
        return
    }
    sort.SliceStable(node.Children, func(i, j int) bool {
        a, b := node.Children[i], node.Children[j]
        if a.IsFolder != b.IsFolder {
            return a.IsFolder // folders first
        }
        return strings.ToLower(a.Name) < strings.ToLower(b.Name)
    })
    for _, child := range node.Children {
        sortTree(child)
    }
}
```

Call `sortTree(root)` before returning.

Also add a client-side sort guard in `FileTreeItem.tsx` (sort `node.children` before rendering) for safety.

### 5.4 Mermaid rendering

**Install:**
```bash
pnpm --dir web add mermaid
```

**NoteRenderer post-render effect:**

```tsx
import mermaid from "mermaid";

// Inside NoteRenderer:
useEffect(() => {
  if (!contentRef.current) return;
  const blocks = contentRef.current.querySelectorAll<HTMLElement>(
    "code.language-mermaid"
  );
  if (blocks.length === 0) return;

  mermaid.initialize({
    startOnLoad: false,
    theme: "base",
    themeVariables: {
      // Match retro design: dark bg, light text
      primaryColor: "#1a1a1a",
      primaryTextColor: "#faf8f4",
      lineColor: "#1a1a1a",
      fontSize: "12px",
    },
  });

  blocks.forEach(async (block) => {
    const pre = block.parentElement;
    if (!pre || pre.tagName !== "PRE") return;
    const src = block.textContent ?? "";
    const id = `mermaid-${nanoid(6)}`;
    try {
      const { svg } = await mermaid.render(id, src);
      const container = document.createElement("div");
      container.className = "mermaid-svg retro-inset my-2";
      container.innerHTML = svg;
      pre.replaceWith(container);
    } catch (err) {
      console.warn("mermaid render failed:", err);
      // Leave the raw <pre> as fallback
    }
  });
}, [resolvedHtml]);
```

**Storybook:** Add a `WithMermaid` story to `NoteRenderer.stories.tsx` with a sample `graph TD` diagram.

### 5.5 Syntax highlighting

**NoteRenderer post-render effect:**

```tsx
import hljs from "highlight.js";

// Inside NoteRenderer, after mermaid effect:
useEffect(() => {
  if (!contentRef.current) return;
  const codeBlocks = contentRef.current.querySelectorAll<HTMLElement>(
    "pre code:not(.language-mermaid)"
  );
  codeBlocks.forEach((block) => {
    hljs.highlightElement(block);
  });
}, [resolvedHtml]);
```

This is minimal — no re-render cost after the initial highlight pass. The `.hljs` CSS overrides in `index.css:500–506` already theme the tokens retro-style.

**Important:** Import the highlight.js CSS or a subset. Since we have retro overrides, import only the base:
```tsx
import "highlight.js/styles/github.css"; // minimal base; retro overrides layer on top
```

Or skip the stylesheet entirely and rely on the custom `.hljs` rules in `index.css`.

**Storybook:** Update the existing `Default` story (which has a Python code block) and add a `WithMultipleLanguages` story containing Go, TypeScript, and bash blocks.

---

## 6. Phased Implementation Plan

### Phase 1: Backend tree sorting (simplest, zero-frontend risk)

**Files:**
- `backend/internal/vault/vault.go` — add `sortTree()` and call it
- `backend/internal/vault/vault_test.go` — add tree ordering test

**Validation:**
```bash
cd backend && go test ./...
curl -fsS http://127.0.0.1:8080/api/tree | python3 -c '...' # verify folders-first, alpha order
```

**Commit:** `Sort file tree folders-first alphabetically`

### Phase 2: Resizable left sidebar

**Files:**
- `web/src/components/pages/VaultLayout/VaultLayout.tsx` — replace flex with ResizablePanelGroup
- `web/src/components/organisms/Sidebar/Sidebar.tsx` — remove `w-56`, accept width from panel
- `web/src/components/ui/resizable.tsx` — add retro-styled handle variant

**Storybook:**
- Add `VaultLayout.stories.tsx` with Default, CollapsedSidebar, NarrowSidebar

**Validation:**
```bash
pnpm --dir web check
pnpm --dir web build
```

**Commit:** `Make left sidebar resizable`

### Phase 3: Right-panel toggle

**Files:**
- `web/src/store/uiSlice.ts` — add `rightPanelOpen`, `toggleRightPanel`
- `web/src/components/pages/VaultLayout/VaultLayout.tsx` — add menubar toggle button
- `web/src/components/pages/NotePage/NotePage.tsx` — conditional aside rendering

**Storybook:**
- Add `NotePage.stories.tsx` with RightPanelOpen, RightPanelClosed

**Validation:**
```bash
pnpm --dir web check
pnpm --dir web build
```

**Commit:** `Add right panel toggle`

### Phase 4: Syntax highlighting

**Files:**
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — add hljs useEffect
- `web/src/index.css` — ensure hljs retro theme covers all token types

**Storybook:**
- Update `NoteRenderer.stories.tsx` Default (verify Python block highlights)
- Add `WithMultipleLanguages` story (Go, TypeScript, bash)

**Validation:**
```bash
pnpm --dir web check
pnpm --dir web build
# Visual: open storybook, verify highlighted code blocks
```

**Commit:** `Add syntax highlighting to note renderer`

### Phase 5: Mermaid rendering

**Files:**
- `web/package.json` — add `mermaid`
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` — add mermaid useEffect (before hljs)
- `web/src/index.css` — add `.mermaid-svg` retro styling

**Storybook:**
- Add `WithMermaid` story to `NoteRenderer.stories.tsx`

**Validation:**
```bash
pnpm --dir web check
pnpm --dir web build
# Visual: storybook WithMermaid story
```

**Commit:** `Add mermaid diagram rendering`

### Phase 6: Rebuild, restart, verify

- Rebuild web dist, restage embedded assets, rebuild binary, restart server.
- Hard-refresh browser and verify all five features on `go-go-parc`.

**Commit:** `Rebuild embedded UI with UI polish`

---

## 7. Testing and Validation Strategy

| Feature | Automated | Manual |
|---------|-----------|--------|
| Tree sorting | `vault_test.go`: verify folders-first alpha | `curl /api/tree` spot check |
| Resizable sidebar | Storybook: VaultLayout stories | Drag handle in browser |
| Right-panel toggle | Storybook: NotePage stories + Redux | Click menubar toggle |
| Syntax highlighting | Storybook: NoteRenderer stories with code blocks | Open note with code in browser |
| Mermaid | Storybook: NoteRenderer WithMermaid | Open note with mermaid in browser |

Cross-browser: Firefox + Chromium. Responsive: test at 1024 px and 1920 px widths.

---

## 8. Risks, Alternatives, and Open Questions

### Risks

1. **Mermaid SSR/async rendering:** Mermaid `render()` is async. Multiple diagrams on one page must use unique IDs. Error handling must leave the raw `<pre>` as fallback.
2. **hljs bundle size:** Importing `highlight.js/lib/core` + only needed languages reduces bundle from ~1.4 MB to ~100 KB. We should use selective imports.
3. **Resizable panels + sidebar toggle:** When the sidebar is toggled off/on, the panel layout must re-render correctly. `react-resizable-panels` handles this, but we need to verify the transition is smooth.
4. **Mermaid + hljs interaction:** The mermaid effect must run before hljs so that `<code class="language-mermaid">` blocks are removed before hljs tries to highlight them.

### Alternatives

1. **Server-side highlighting:** Use `goldmark-highlighting` (chroma) in the backend. Pro: zero JS cost. Con: adds CGO/chroma dependency to Go binary, harder to theme dynamically.
2. **react-markdown client-side:** Replace `dangerouslySetInnerHTML` with `react-markdown` + `rehype-highlight` + `remark-mermaid`. Pro: React-native rendering. Con: major rewrite of NoteRenderer, re-parsing markdown client-side.
3. **CSS-only resize:** Use `resize: horizontal` on the sidebar. Pro: zero deps. Con: no min/max constraints, no visual handle, no panel group coordination.

### Open questions

1. Should the right panel be resizable too, or just togglable? (Start with toggle; add resize later if requested.)
2. Should sidebar width persist across sessions? (Start with per-session; add localStorage later if requested.)
3. Which highlight.js languages to bundle? (Start with: python, go, javascript, typescript, bash, yaml, json, sql, rust, css, xml, markdown, dockerfile.)

---

## 9. References

- `web/src/components/pages/VaultLayout/VaultLayout.tsx`
- `web/src/components/organisms/Sidebar/Sidebar.tsx`
- `web/src/components/molecules/FileTreeItem/FileTreeItem.tsx`
- `web/src/components/pages/NotePage/NotePage.tsx`
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`
- `web/src/components/ui/resizable.tsx`
- `web/src/store/uiSlice.ts`
- `web/src/index.css:500–506`
- `backend/internal/vault/vault.go:228–260`
- `backend/internal/parser/parser.go`
