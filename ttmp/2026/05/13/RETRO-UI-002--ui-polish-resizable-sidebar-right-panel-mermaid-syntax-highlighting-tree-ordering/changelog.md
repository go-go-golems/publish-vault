# Changelog

## 2026-05-13

- Initial workspace created


## 2026-05-13

Created design + implementation guide for 5 UI polish features

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/13/RETRO-UI-002--ui-polish-resizable-sidebar-right-panel-mermaid-syntax-highlighting-tree-ordering/design-doc/01-ui-polish-design-and-implementation-guide.md — Primary design doc


## 2026-05-13

Implemented all 5 UI polish features: tree sorting, resizable sidebar, right panel toggle, syntax highlighting, mermaid rendering

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go — sortTree()
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — hljs+mermaid effects
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/pages/VaultLayout/VaultLayout.tsx — ResizablePanelGroup
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/store/uiSlice.ts — rightPanelOpen


## 2026-05-13

Fixed wiki-link resolution: short Obsidian paths now resolve to full vault slugs, enabling correct navigation and backlinks.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/parser/parser.go — ReplaceWikiLinksString
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go — buildWikiLinkIndex


## 2026-05-13

Render callout admonitions and widen article to max-w-4xl

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/parser/parser.go — renderCallouts()
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/index.css — .callout styles


## 2026-05-13

Step 4: Fuzzy search, right-panel resizable, graph edge fix (72ed05b, 581caf8, d0908ec)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/api/api.go — Graph edge resolution using ResolveWikiLink
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/search/search.go — Fuzzy MatchQuery with fuzziness=1
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/pages/NotePage/NotePage.tsx — ResizablePanelGroup for right panel


## 2026-05-13

Step 5: Resolve wiki-link display text to note titles (dc68621)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/parser/parser.go — data-raw attribute and ReplaceWikiLinkDisplay
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go — rebuildHTML calls ReplaceWikiLinkDisplay


## 2026-05-13

Step 6: Filter self-edges and add copy-code button (36258ba, 1040c14)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/api/api.go — Self-edge filter resolved != n.Slug
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Copy button injection


## 2026-05-13

Step 7: Render ![[embeds]] inline and add collapsible callouts (9477854, 945b53d)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/parser/parser.go — Updated calloutRe with fold char
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Embed fetching


## 2026-05-13

Step 8: Heading permalink anchors and hash-scroll navigation (bd03a56)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Heading permalink injection and hash-scroll effect
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/index.css — heading-anchor and callout-collapsible styles


## 2026-05-14

Step 9: Round 3 tasks and frontend node graph removal (213cb65, f334504)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/pages/NotePage/NotePage.tsx — Removed graph panel and graph query
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/pages/VaultLayout/VaultLayout.tsx — Removed graph toggle
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/store/uiSlice.ts — Removed graphVisible state


## 2026-05-14

Step 10: Preserve wiki-link heading fragments and explicit aliases (4554af0)

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/parser/parser.go — Fragment-preserving href resolution and data-alias support
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Preserve hash when navigating wiki links

