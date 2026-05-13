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
