# Changelog

## 2026-07-17

- Initial workspace created


## 2026-07-17

Investigation + design complete: verified no prior ticket (only adjacent PV-BACKEND-API-001), resolved ttmp/vocabulary.yaml merge conflict, mapped publish-vault web/ and rag-evaluation-system widget DSL v3, wrote the intern-level analysis/design/implementation guide (2 tracks, 6 decision records, 5 phases) and investigation diary.

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/ttmp/2026/07/17/PV-WIDGET-DSL-015--reorganize-publish-vault-web-into-a-design-system-and-add-a-widget-dsl-styled-go-go-goja-js-api/design-doc/01-widget-dsl-and-design-system-reorganization-analysis-design-and-implementation-guide.md — Primary deliverable
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/ttmp/vocabulary.yaml — Conflict resolution + new topics widget-dsl/design-system/goja


## 2026-07-17

Implemented Phases 0-2: deleted Manus/shadcn scaffold (51 ui files, 50 deps), split index.css into tokens/bridge/base/chrome/prose layers with --pv-* canonical palette and --rag-* bridge, added foundation/ and layout/ tiers with stories, decomposed NoteRenderer into NoteView (+NoteBody/noteEnhancements/NoteActions) with all network via RTK Query (new getNoteRaw endpoint), fixed pre-existing Storybook 10 addon incompatibility. Validated: tsc, vite build+SSR, 13 vitest, storybook build, smoke:ssr PASS, go test, live browser enhancement check.

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/components/organisms/NoteView/NoteView.tsx — Decomposition composition root
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/store/vaultApi.ts — New getNoteRaw endpoint
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/styles/tokens.css — Canonical --pv-* token layer


## 2026-07-17

Implemented Phases 3-4: goja widget host (internal/widgethost + read-only vault.data module in internal/vaultdata, rag-evaluation-system pkg/widgetdsl v0.1.7 imported per D1), /api/widget/{pages,actions} contract verbatim per D2, --pages-dir flag defaulting to <vault>/.publish/pages, two example pages; frontend widgets/ subsystem ported (IR types, registry, WidgetRenderer, action dispatcher, cellRenderers) with 10-adapter retro registry, new DataTable/KeyValueStrip molecules, /w/:pageId route. Tests: 12 new Go tests incl. contract-parity vs rag-eval golden, registry-completeness vitest on captured live IR; e2e browser check green (row-click navigates to note, zero console errors).

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/examples/widget-pages/recent.js — Example widget page
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/vaultdata/vaultdata.go — vault.data native module
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/widgethost/widgethost.go — Widget host core
- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/web/src/widgets/WidgetRenderer.tsx — Ported IR renderer

