---
Title: Investigation diary
Ticket: PV-WIDGET-DSL-015
Status: active
Topics:
    - frontend
    - design-system
    - widget-dsl
    - goja
    - xgoja
    - react
    - api
    - ssr
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/06/22/PV-BACKEND-API-001--design-backend-api-for-dynamic-publish-vault-pages-and-xgoja-servers/design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md
      Note: Prior backend/xgoja design the widget DSL work layers on
    - Path: repo://ttmp/vocabulary.yaml
      Note: Merge conflict resolved (kept both sides additive slugs)
ExternalSources: []
Summary: 'Chronological investigation diary for PV-WIDGET-DSL-015: checking for prior tickets, resolving the vocabulary merge conflict, mapping rag-evaluation-system widget DSL v3 and publish-vault web/, and producing the design/implementation guide.'
LastUpdated: 2026-07-17T15:30:00-04:00
WhatFor: Continuation-friendly record of how the PV-WIDGET-DSL-015 analysis was produced.
WhenToUse: When resuming or reviewing the widget DSL / design-system reorganization work.
---


# Diary

## Goal

Capture the investigation that led to PV-WIDGET-DSL-015: verifying no prior ticket covered the web/ reorganization + widget.dsl-styled go-go-goja JS API, gathering evidence from rag-evaluation-system (widget DSL v3) and publish-vault web/, and writing the intern-facing analysis/design/implementation guide.

## Step 1: Verify no prior ticket exists and create PV-WIDGET-DSL-015

The first task was to establish whether a ticket already covered "reorganize web/ + widget.dsl-styled go-go-goja JS API". I listed all 18 docmgr tickets and grepped ttmp/ for `widget.dsl`, `widget-dsl`, `rag-evaluation`, `reorganiz`, `atom.*molecule`, and `design system`. The only close ticket is PV-BACKEND-API-001 (2026-06-22), which designs the backend `/api/v1` contract and xgoja packaging modes but says nothing about the frontend design-system reorganization or an intent-level widget authoring DSL. The few grep hits in RETRO-WIKILINK-006 and RETRO-SEO-009 were incidental phrasing, not scope overlaps. Conclusion: no existing ticket; create a new one that explicitly builds on PV-BACKEND-API-001.

Along the way I found that `ttmp/vocabulary.yaml` was sitting in an unresolved git merge conflict (`UU` state) between "Updated upstream" (watcher/ignore/bundle/performance/vite slugs from PERF-BUNDLE-014 and RETRO-IGNORE-013) and "Stashed changes" (api/xgoja slugs from PV-BACKEND-API-001). Both sides were purely additive topic lists, so I merged them by keeping all slugs and staged the resolution.

### Prompt Context

**User prompt (verbatim):** "Analyze publish-vault and figure out if a) we have already created a ticket for reorganizing the web/ part and creating a widget.dsl styled go-go-goja JS API for it (see rag-evaluation-system and potential resources /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/widget-dsl.md . If so, reassess since we modified widget.dsl v3 quite a bit and it would be good to align, also on the react design system organization (atom/organism/molecule, IR renderer, etc...). If not, then create a docmgr ticket and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Check whether a docmgr ticket already exists for (a) reorganizing publish-vault's web/ frontend and (b) adding a widget.dsl-v3-style JS API backed by go-go-goja. If yes, reassess it against the current widget DSL v3 and design-system organization in rag-evaluation-system. If no, create a ticket plus an intern-level analysis/design/implementation guide and upload it to reMarkable.

**Inferred user intent:** Bring the widget DSL v3 + design-system patterns proven in rag-evaluation-system into publish-vault in a planned, documented way, with a guide good enough that a new intern could execute the reorganization and DSL work.

### What I did
- `docmgr ticket list` — reviewed all 18 tickets; only PV-BACKEND-API-001 is adjacent.
- `grep -ril "widget.dsl\|widget-dsl\|rag-evaluation\|reorganiz\|atom.*molecule\|design system" ttmp/` — no real hits.
- Read `go-go-parc/Research/KB/Projects/widget-dsl.md` (the Widget DSL MOC) for the layered intent/IR/target architecture and working rules.
- Read PV-BACKEND-API-001's design doc in full (493 lines) to align the new design with the proposed `Backend` interface, `/api/v1` contract, `Page` model, and xgoja packaging modes A/B/C.
- Resolved the `ttmp/vocabulary.yaml` merge conflict by keeping both sides' slugs; `git add`ed the resolution.
- Added vocab topics `widget-dsl`, `design-system`, `goja`.
- `docmgr ticket create-ticket --ticket PV-WIDGET-DSL-015 ...` and added the design doc + this diary.
- Launched two very-thorough Explore agents: one mapping rag-evaluation-system's widget DSL v3 + React design system, one mapping publish-vault's web/ + Go serving path.

### Why
- The user explicitly asked for a ticket-existence check before creating anything.
- The vocabulary conflict would have broken `docmgr doctor` and frontmatter validation later.
- PV-BACKEND-API-001 is the natural substrate: the widget DSL is the UI-authoring layer on top of that ticket's backend/xgoja design, so the two must reference each other rather than conflict.

### What worked
- docmgr ticket listing + targeted grep quickly ruled out prior art.
- The vocabulary conflict was purely additive on both sides, so the resolution was mechanical.

### What didn't work
- N/A so far (no failed commands in this step).

### What I learned
- PV-BACKEND-API-001 already proposes: a `Backend` Go interface, `/api/v1` endpoints, a first-class `Page` model distinct from `Note`, a `POST /api/v1/render` SSR prefetch contract, and three xgoja packaging modes (standalone JS server / Go shell with JS backend / Go shell with JS dynamic routes). The widget DSL design should slot widgets into that `Page` model (`Page.kind: "dynamic"` with widget IR payload) instead of inventing a parallel channel.
- The widget-dsl KB note establishes the vocabulary: intent layer (semantic authoring), IR layer (typed widget instances, slots, actions, data contracts), target layer (React renderer, presets, Storybook), and working rules like "normalize and validate before rendering" and "make actions part of the IR".

### What was tricky to build
- Nothing yet; this step was reconnaissance and bookkeeping.

### What warrants a second pair of eyes
- The vocabulary.yaml conflict resolution (kept all slugs from both branches) — confirm no slug was intentionally being removed by the stash.

### What should be done in the future
- N/A

### Code review instructions
- Check `ttmp/vocabulary.yaml` for the merged slug list (watcher, ignore, bundle, performance, vite, api, xgoja plus new widget-dsl, design-system, goja).
- Ticket workspace: `ttmp/2026/07/17/PV-WIDGET-DSL-015--reorganize-publish-vault-web-into-a-design-system-and-add-a-widget-dsl-styled-go-go-goja-js-api/`.

### Technical details
- Prior ticket: `ttmp/2026/06/22/PV-BACKEND-API-001--design-backend-api-for-dynamic-publish-vault-pages-and-xgoja-servers/design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md`.
- KB source: `/home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/widget-dsl.md`.
- Reference implementation: `/home/manuel/code/wesen/go-go-golems/rag-evaluation-system`.

## Step 2: Map publish-vault web/ (Explore agent results)

The publish-vault frontend map came back and materially reframed the ticket: `web/src/components` is *already* organized as atoms/molecules/organisms/pages with co-located Storybook stories, so the reorganization is not "introduce atomic design" but "finish it": remove the Manus scaffold and ~52 unused shadcn `components/ui/*` primitives, split the 988-line `index.css` token/skin monolith, and decompose the 431-line `NoteRenderer` that mixes `dangerouslySetInnerHTML`, six imperative DOM effects, and two ad-hoc `fetch()` calls. I also confirmed the Go backend has zero goja/xgoja today — the only JS runtime is the Node SSR sidecar (`web/server.mjs`), which duplicates `chooseHomeSlug` from `App.tsx`.

Separately, I read `go-go-goja/modules/uidsl/module.go` myself: go-go-goja already ships a low-level HTML element DSL (`require("ui")` / `ui.div(...)`, tables, tabs, badges, code blocks) with a `Registrar` pattern (`RegisterRuntimeModule(ctx *engine.RuntimeModuleRegistrationContext, reg *require.Registry)`) and an optional `TypeScriptDeclarer` interface for generated .d.ts. This is the module-registration idiom the publish-vault `widget.dsl` module should follow, but the widget DSL itself is semantic (widgets/slots/actions), not element-level.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** (same as Step 1; this step is the web/ evidence-gathering phase)

**Inferred user intent:** (same as Step 1)

### What I did
- Launched an Explore agent over `web/` + Go serving path; received a file/line-anchored map (components, state, SSR, API, build, dead code).
- Read `go-go-goja/modules/uidsl/module.go`, `modules/typing.go` directly for the native-module registration pattern.
- Added a 7-item task list to `tasks.md`; checked off the ticket-existence task.

### Why
- The design doc needs line-anchored current-state evidence, and the intern guide needs the "what already exists vs what is new" distinction to be precise.

### What worked
- Fan-out exploration: the agent produced exact line anchors (e.g. `NoteRenderer.tsx:416` for the `dangerouslySetInnerHTML` injection, `server.mjs:105` for the duplicated home-slug chooser).

### What didn't work
- A zsh heredoc-free `echo ===` in one command was parsed as a glob (`(eval):1: == not found`); reran with quoted strings.

### What I learned
- Key facts for the design: RTK Query is the single data layer (`store/vaultApi.ts`, with a `VITE_STATIC_VAULT` static mode that already proves the backend is swappable at the TS query layer); SSR preloads config/tree/note via `upsertQueryData`; the Go binary embeds `web/dist` behind the `embed` build tag (`internal/web/embed.go`); and `components/ui/sidebar.tsx` (734 lines) is dead scaffold, not the real sidebar.

### What was tricky to build
- Nothing tricky; the main risk is conflating the two `pages/` directories (`web/src/pages/` = dead Manus scaffold vs `web/src/components/pages/` = real route components) — the guide must call this out.

### What warrants a second pair of eyes
- The claim that only `ui/resizable` and `ui/dialog` (plus dead-file imports of button/card) are used from shadcn — verify with an import graph before deleting the other 52 files.

### What should be done in the future
- Verify unused-dependency list (recharts, embla-carousel, react-hook-form, etc.) against `pnpm why` before removal; this belongs in the implementation phase.

### Code review instructions
- Cross-check the current-state section of the design doc against `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`, `web/src/store/vaultApi.ts`, `web/server.mjs`, `internal/web/embed.go`.

### Technical details
- Frontend totals: `web/src` ≈ 12,368 lines of TS/CSS; biggest files: `index.css` 988, `ui/sidebar.tsx` 734 (dead), `staticVault.ts` 462, `NoteRenderer.tsx` 431, `server.mjs` 373.

## Step 3: Map widget DSL v3 and write the design/implementation guide

The rag-evaluation-system Explore agent returned a complete five-layer map of widget DSL v3: JS authors `require("widget.dsl")` (the only production module since the hard cutover — `pkg/widgetdsl/module.go:15-21`, legacy split modules test-only), fluent builders in `pkg/widgetdsl/v3.go` (2417 lines) produce typed specs (`pkg/widgetdsl/spec`) that `Validate()` and lower via `ToWidgetPage()` into serializable Widget IR (`WidgetPage`/three-kind `WidgetNode`), served at `GET /api/widget/pages/{id}` and rendered by `packages/rag-evaluation-site/src/widgets/WidgetRenderer.tsx` against a registry of per-component `defineWidget` adapters; actions POST to `/api/widget/actions/{name}` with `{ok,refresh,toast,patch}` results. The design system there uses five tiers (foundation/atoms/layout/molecules/organisms) with a 6-file-per-component convention including `.widget.tsx` adapters, `.widget.yaml` manifests, and CSS modules over `--rag-*` tokens (with a `--mac-*` bridge — the precedent I reused for publish-vault's token bridge).

With both maps in hand I wrote the primary design doc: intern orientation (goja, xgoja, widget DSL, server-driven UI, atomic tiers), evidence-anchored current-state and reference-architecture sections, a gap table, target architecture (Track A: finish the web/ reorganization in place; Track B: `internal/widgethost` + `internal/vaultdata` + ported `web/src/widgets/` renderer), six decision records (import-not-reimplement the DSL; verbatim `/api/widget/*` contract; token bridge over restyling; separate `vault.data` module; client-render v1/defer SSR; no packages/ split), five implementation phases with validation commands, a test strategy including a cross-repo contract-parity golden test, and risks/open questions.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** (same as Step 1; this step covers the rag-eval evidence and the main deliverable)

**Inferred user intent:** (same as Step 1)

### What I did
- Received the rag-evaluation-system map (file/line-anchored; 43 example/golden pairs identified as the executable grammar spec).
- Wrote `design-doc/01-widget-dsl-and-design-system-reorganization-analysis-design-and-implementation-guide.md` (~640 lines): 12 sections, 3 mermaid diagrams, 6 decision records, 5 phases.
- Aligned the design with PV-BACKEND-API-001 (widget pages = `Page.kind:"dynamic"`; `/api/widget/*` as transport under the future `/api/v1/page`).

### Why
- The user asked for a "very clear and technical" guide "for a new intern" — hence the orientation section, prose + bullets + pseudocode + diagrams + API references + file references structure.

### What worked
- Running both Explore agents in parallel kept wall-clock short; both returned line-anchored evidence directly usable in the doc.
- rag-eval's own `--mac-*` token bridge and in-place tier layout gave precedent-backed answers to the two most contestable choices (D3, D6).

### What didn't work
- N/A (no failed commands this step).

### What I learned
- The single most important porting decision is D1 (import `rag-evaluation-system/pkg/widgetdsl` rather than reimplement): the grammar is ~4,300 lines with 43 golden-tested examples; any reimplementation would drift immediately.
- publish-vault's `VITE_STATIC_VAULT` mode already proves the frontend data layer is backend-agnostic, which is why widget endpoints join RTK Query rather than a bespoke fetch layer.

### What was tricky to build
- Reconciling three documents that could contradict: PV-BACKEND-API-001 (page/render contract), rag-eval's `/api/widget/*` contract, and the reorganization. Resolution: adopt rag-eval's transport verbatim now (renderer compatibility), and declare it the transport detail *beneath* PV-BACKEND-API-001's `Page.kind:"dynamic"` model later — recorded as Decision D2 so it doesn't get re-litigated.

### What warrants a second pair of eyes
- Decision D1's dependency-weight risk: audit `go mod graph` after `go get github.com/go-go-golems/rag-evaluation-system` before committing to it.
- The v1 registry component set (§7.5) — confirm it covers the actual first pages Manuel wants.

### What should be done in the future
- Phase-by-phase execution per the design doc; extraction of `widgetdsl` into a shared module as the D1 follow-up.

### Code review instructions
- Start with the design doc §7 (proposed architecture) and §8 (decision records); check §4/§5 claims against the anchored files if skeptical.
- Validate mermaid rendering and the example page script (`recent.js` sketch) against the real v3 grammar in `pkg/widgetdsl/testdata/v3/examples/`.

### Technical details
- rag-eval shortlist for porting: `pkg/widgetdsl/{module.go,v3.go,spec/}`, `cmd/widgetdsl-v3-preview/main.go` (run wrapper at `:143-176`), `packages/rag-evaluation-site/src/widgets/*`, `src/hooks/useWidgetPage.ts`, `src/theme.css`.

## Step 4: Bookkeeping, validation, and reMarkable delivery

Final housekeeping: related the key evidence files to both docs (docmgr normalized them to repo:// paths and I deduplicated the hand-written frontmatter entries), updated the ticket index with a real overview and links, wrote the changelog entry, and checked all tasks. `docmgr doctor --ticket PV-WIDGET-DSL-015 --stale-after 30` passed with all checks green on the first run (the vocabulary topics added in Step 1 prevented warnings).

Delivery: bundled the design doc + diary into a single PDF with ToC via remarquee (dry-run first, then real upload) to `/ai/2026/07/17/PV-WIDGET-DSL-015` as "PV-WIDGET-DSL-015 Widget DSL + design-system reorg guide"; verified with `remarquee cloud ls --long` (entry present as `[f]`).

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** (same as Step 1; this step is validation + delivery)

**Inferred user intent:** (same as Step 1)

### What I did
- `docmgr doc relate` (6 files → design doc, 2 → diary), `docmgr changelog update`, `docmgr task check` for all 7 tasks.
- `docmgr doctor --ticket PV-WIDGET-DSL-015 --stale-after 30` → ✅ all checks passed.
- `remarquee status` / `cloud account` (user wesen@ruinwesen.com), `upload bundle --dry-run`, real upload, `cloud ls` verification.

### What worked
- Everything on first attempt; docmgr relate auto-normalized absolute paths to repo:// form.

### What didn't work
- Minor: docmgr relate appended repo:// entries alongside the frontmatter RelatedFiles I had hand-written, producing duplicates in the design doc; fixed by removing the hand-written relative-path entries.

### What I learned
- Don't pre-populate RelatedFiles by hand when you intend to run `docmgr doc relate` — let the tool own that block.

### Code review instructions
- `docmgr doctor --ticket PV-WIDGET-DSL-015 --stale-after 30` should stay green.
- reMarkable: `/ai/2026/07/17/PV-WIDGET-DSL-015/PV-WIDGET-DSL-015 Widget DSL + design-system reorg guide` (PDF with ToC).

### Technical details
- Bundle contents: design-doc/01 + reference/01; `--toc-depth 2`.

## Step 5: Implement Phases 0–2 (scaffold removal, style layering, NoteView decomposition)

Executed the first three implementation phases from the design doc. Phase 0: verified the dead-code claims with a Python import-graph scan before deleting — confirmed only `ui/dialog` (LightboxModal) and `ui/resizable` (NotePage, VaultLayout) are live among 53 shadcn files; deleted the other 51 plus the Manus scaffold (`Map.tsx`, `ManusDialog.tsx`, `ErrorBoundary.tsx`, `src/pages/`, `contexts/`, `const.ts`) and three hooks only the deleted files used (`useMobile`, `useComposition`, `usePersistFn`); removed 47 unused runtime dependencies plus `@types/google.maps` and the junk `add` package. Phase 1: split the 988-line `index.css` into `styles/{tokens,bridge,base,chrome,prose}.css` — tokens.css introduces the canonical `--pv-*` palette with the Tailwind `@theme inline` block re-mapping `--color-*` names onto it (so every existing `var(--color-*)` usage keeps working), trims the shadcn variable set to the eight actually consumed, and `bridge.css` maps all 29 `--rag-*` tokens rag-eval CSS modules consume onto retro equivalents.

Phase 2: created `components/foundation/` (Text, Caption, CodeText, VisuallyHidden, moved Divider from atoms with importer updates) and `components/layout/` (Stack, Inline, Panel, SectionBlock, SplitPane wrapping ui/resizable) with stories; decomposed the 431-line `NoteRenderer` into `organisms/NoteView/` — `NoteView.tsx` (composition root + click delegation), `NoteBody.tsx` (pure dangerouslySetInnerHTML host), `noteEnhancements.ts` (idempotent pipeline: enhanceMermaid → enhanceCodeBlocks/addCopyButtons → resolveEmbeds → enhanceHeadingAnchors, all taking injected dependencies), `NoteActions.tsx` (copy/download row). Both raw `fetch()` call sites are gone: embeds load via `vaultApi.endpoints.getNote.initiate` and copy-markdown via a new `getNoteRaw` RTK Query endpoint (with a static-vault queryFn, so embeds/copy now also work in VITE_STATIC_VAULT builds). NotePage switched to NoteView + SplitPane.

### Prompt Context

**User prompt (verbatim):** "Alright, implement it, phase 0 - 2"

**Assistant interpretation:** Execute Phases 0, 1, and 2 from the design doc's implementation plan against web/.

**Inferred user intent:** Land the Track A reorganization so the widget-DSL phases (3–4) can start on a clean design-system base.

### What I did
- Import-graph verification (Python os.walk + regex over src/.storybook/server.mjs/vite configs) before every deletion class.
- `pnpm remove` × 50 deps; deletions listed above.
- CSS split with layer verification via grep over the built `dist/assets/main-*.css` (retro-window, note-prose, --pv-ink, --rag-color-bg all present).
- NoteView decomposition; stories ported with a Redux `Provider` decorator (NoteView now dispatches through RTK).
- Fixed pre-existing Storybook breakage: v8 addons (`@storybook/addon-essentials`, `addon-interactions`, `@storybook/test`) are incompatible with Storybook 10 (`NoMatchingExportError: Icons`); removed them from `.storybook/main.ts` and package.json — `pnpm build-storybook` now succeeds.
- Validation: `tsc --noEmit` clean; `pnpm build` + `build:ssr` clean; vitest 13/13 SSR tests pass; `pnpm smoke:ssr` PASS (Go server + SSR sidecar + Chromium hydration, 0 console errors); `go test ./...` all pass; live Playwright check on `/note/zettelkasten-method` confirmed the enhancement pipeline (4 copy buttons, 8 hljs nodes, 40 heading anchors, 24 wiki-links/0 broken, SPA navigation on wiki-link click, 0 console errors/warnings).

### Why
- Verify-before-delete because the design doc itself flagged the unused-shadcn list as needing an import graph before deletion.
- `chrome.css` added as a fifth layer (design doc said prose.css would hold `.retro-*`): shell chrome and note-content skin are different change surfaces; the design doc was patched to match.

### What worked
- The `@theme inline` re-mapping (`--color-ink: var(--pv-ink)`) preserved every existing Tailwind arbitrary value with zero component edits.
- The enhancement pipeline extraction was behavior-preserving on first try — smoke + live browser checks green without iteration.

### What didn't work
- `pnpm add -D jsdom` fails: the global pnpm store points at `/home/manuel/workspaces/2026-05-03/.../.pnpm-store`, which is a read-only mount in this environment (`ERR_PNPM_EROFS`), so no NEW packages can be installed (removals and installs-from-store work). The planned jsdom unit tests for noteEnhancements are deferred; the failed install also transiently broke node_modules (vitest unresolvable) — fixed with `pnpm install`.
- The Playwright MCP browser (system Chrome) crashes in this sandbox (GPU + read-only crash-report dir); worked around by driving the web package's bundled Playwright chromium with a scratchpad script.
- zsh ate several inline grep/loop one-liners (`bad math expression` from `$b[\"']` subscripts, `== not found` from bare `===`); switched to Python heredocs for the import analysis.

### What I learned
- shadcn `components/ui` files import each other heavily (sidebar → sheet/skeleton/tooltip/separator/input/button), so "unused" must be computed on external importers only.
- `NotePage` imported `ui/resizable` via a relative path (`../../ui/resizable`), which an alias-only grep misses — usage scans need both forms.
- The `getNoteRaw` endpoint doubles as a static-mode fix: the old raw-fetch path 404'd in VITE_STATIC_VAULT builds.

### What was tricky to build
- Keeping SSR/hydration identical through the decomposition: the `resolvedHtml` two-phase dance (raw server HTML for hydration parity, wiki-link resolution in a post-hydration effect, synchronous reset on note swap) had to move into NoteView unchanged — it's load-bearing for the 13 entry-server tests and the hydration smoke.
- Tailwind v4 CSS splitting: `@theme` blocks and `@apply` uses must remain in files reachable from the entry `@import` chain; verified by grepping the built CSS for tokens from every layer.

### What warrants a second pair of eyes
- `styles/tokens.css` trims the shadcn variable set to 8 (background, foreground, muted, muted-foreground, accent, accent-foreground, border, ring); if a future shadcn component is added it may reference dropped variables (card, popover, primary, secondary, destructive, input, sidebar-*).
- The Storybook addon removal: controls/docs behavior in SB10 core should be spot-checked in `pnpm storybook` dev mode.
- `NoteView` retains its former props contract; `VaultLayout` still imports `ui/resizable` directly (left-sidebar split doesn't fit SplitPane's main/side shape) — acceptable or extend SplitPane later.

### What should be done in the future
- Add jsdom (or happy-dom) unit tests for `noteEnhancements.ts` once the pnpm store is writable.
- Phase 3–4 (Go widget host + frontend widget subsystem) per the design doc.

### Code review instructions
- Start: `web/src/components/organisms/NoteView/` (four files) vs the deleted `NoteRenderer.tsx` in git history; `web/src/styles/*.css` vs the old `index.css`; `web/src/store/vaultApi.ts` (`getNoteRaw`).
- Validate: `cd web && pnpm check && pnpm build && pnpm build:ssr && npx vitest run && pnpm build-storybook && pnpm smoke:ssr`; `go test ./... -count=1`.

### Technical details
- Deleted: 51 ui files (~5,900 lines incl. 734-line sidebar.tsx), Manus scaffold, 50 packages from package.json.
- web/src is now ~7,600 lines of TS/CSS (was ~12,368).
- Built CSS after split: 42.7 kB (selector/token spot-checks green).

## Step 6: Implement Phases 3–4 (Go widget host + frontend widget subsystem)

Phase 3 landed the first goja code in publish-vault. Decision D1 turned out even cleaner than designed: `github.com/go-go-golems/rag-evaluation-system` is published and its local checkout HEAD is exactly the released tag v0.1.7 (tagged four hours before this step), so a plain `go get` — no replace directive — pins the current widget DSL v3. `internal/vaultdata` exposes the read-only `vault.data` module (config/notes/note/search/tree/tags) with every value passed through a JSON round-trip so scripts see the exact `/api/*` wire shapes; the list/tags logic was lifted into exported `api.NoteList`/`api.TagCounts` helpers so HTTP handlers and the JS module share one source. `internal/widgethost` discovers `<pagesDir>/*.js`, runs each render in a fresh VM with `widgetdsl.Register` + `vaultdata.Register`, wraps scripts with the same `page.toPage()` convention as rag-eval's preview host, injects `globalThis.request = {pageId, query}`, and serves the rag-eval-compatible contract (`GET /api/widget/pages[/{id}]`, `POST /api/widget/actions/{name}` with `pageId.action` addressing plus bare-name scan). `--pages-dir` defaults to `<vault>/.publish/pages`; routes stay silently disabled when the directory is absent. Two example pages (`examples/widget-pages/{recent,tags}.js`) use only grammar verbs verified against the 43 v3 examples.

Phase 4 ported the renderer stack into `web/src/widgets/`: `ir/` (core trimmed to a `PvWidgetType` v1 union; `actions.ts` ported verbatim; cells trimmed), `registry.ts`, `actions.ts` dispatcher (verbatim — server actions POST to the same paths), `cellRenderers.tsx` restyled with retro classes, `WidgetRenderer.tsx` (UnknownWidget → retro callout), plus `useWidgetPage`. The v1 registry has ten adapters: new retro `DataTable` and `KeyValueStrip` molecules, `.widget.tsx` adapters over the Phase-2 `Stack`/`Inline`/`SectionBlock` layouts and `Text`/`Caption`/`Divider` foundations, a callout-skinned `Panel`, and a `Tag` adapter over the existing atom (proving the registry decouples IR types from any one component set). Route `/w/:pageId` renders through `WidgetPage`, where navigate actions go through React Router and `widget:action-result` events with `refresh:true` re-fetch the IR.

### Prompt Context

**User prompt (verbatim):** "phase 3 - 4"

**Assistant interpretation:** Implement design-doc Phases 3 (Go widget host) and 4 (frontend widget subsystem).

**Inferred user intent:** Make the widget.dsl-styled JS API real end-to-end: JS page scripts against the live vault, rendered in the retro SPA.

### What I did
- Read the actual v3 sources/goldens before coding: `module.go` Register, preview-host run wrapper (`main.go:143-176`), golden IR for tables (`01`), rowSelect/navigate (`02`), action columns (`04`), metrics/KeyValueStrip/Panel (`27`); ported renderer sources read in full.
- Go: `go get rag-evaluation-system@latest` (v0.1.7, module graph now 258 entries), `internal/vaultdata` (+6 script-driven tests), `internal/widgethost` (+6 tests incl. HTTP handlers), `api.NoteList`/`api.TagCounts` refactor, server/serve wiring, example pages.
- Contract-parity test: rag-eval's `01-simple-table.{js,golden.json}` copied into `internal/widgethost/testdata/` and asserted byte-shape-equal through the publish-vault host (provenance comment says re-copy on upgrade).
- Frontend: `web/src/widgets/` subsystem, 10-adapter default registry, `/w/:pageId` route, registry-completeness vitest against a captured live fixture (`__fixtures__/recent-page.json`), stories for WidgetRenderer/DataTable/KeyValueStrip.
- Validation: Go tests all green (13 packages) + vet + gofmt; `tsc` clean; vite client+SSR builds; 15 vitest; Storybook build; live browser e2e — `/w/recent` renders title/metric/5-row table with zero console errors, row click navigates to `/note/zettelkasten-method` with note visible, `/w/tags` renders 12 tags.

### What worked
- The parity golden passed on the first run after the marshal fix — grammar identity via direct dependency worked exactly as D1 predicted.
- Capturing the live server IR as the Storybook/vitest fixture keeps frontend tests honest against the real backend output.

### What didn't work
- First test run failed twice: (1) `json.Marshal` on the wrapper output hit the exported action *functions* (`unsupported type: func(goja.FunctionCall)`) — fixed by keeping the wrapper result as a live `*goja.Object` and marshaling only the `page` member; (2) a variable shadowing bug (`out` redeclared) from that refactor.
- `vault.FileTree()` has no `type` field (`isFolder` bool) — test assertion fixed after reading the struct.
- The e2e browser check initially reported 10 tbody rows for 5 notes — not a bug: `VaultLayout` renders children into desktop AND CSS-hidden mobile panes (pre-existing pattern, same as NotePage).

### What I learned
- widget.dsl v3 lowers `section().metric()` to `KeyValueStrip` components and `widget.ui.callout` to `Panel {title, tone}` — the v1 registry needed both beyond the obvious Stack/SectionBlock/DataTable.
- rag-eval's dispatcher handles navigate by `history.pushState` + synthetic `popstate`, which React Router v7 honors — but routing through `useNavigate` in `WidgetPage.handleAction` is cleaner in-app.

### What was tricky to build
- Action-handler execution: handlers are live JS functions, so the render path (JSON-marshal the page) and the action path (call a function in the same VM) need different lifetimes from one `evalPage`; returning the goja object + VM pair solved both without re-running scripts twice per action.
- Bare action names (`POST /api/widget/actions/ping`) require scanning pages, which re-executes scripts; documented `pageId.action` as the preferred form.

### What warrants a second pair of eyes
- Security posture of `handleAction`: page scripts are trusted, but action payloads are attacker-supplied JSON passed into script functions — size-limited (1 MiB) and JSON-decoded, yet a hostile payload could still make a page script do heavy compute. Acceptable for a personal site; revisit before multi-tenant use.
- The bare-name action scan runs every page script; fine at 2 pages, O(pages) per action otherwise.
- `structuredNavigationTarget` uses `window.location` — SSR-safe only because widget pages are client-only (D5).

### What should be done in the future
- Phase 5 hardening: per-example Go golden tests for the publish-vault example pages; `.widget.yaml` manifests; SSR/agent-markdown for widget pages once PV-BACKEND-API-001 `/api/v1/render` lands; upgrade cadence for the widgetdsl dependency (re-copy parity fixture on bump).

### Code review instructions
- Go: start at `internal/widgethost/widgethost.go` (wrapper + action invocation), then `internal/vaultdata/vaultdata.go` (JSON round-trip boundary), `internal/server/server.go` (route gating).
- Frontend: `web/src/widgets/` (compare `actions.ts` against rag-eval's for verbatim-ness), `web/src/components/molecules/DataTable/DataTable.widget.tsx`, `web/src/components/pages/WidgetPage/WidgetPage.tsx`.
- Validate: `GOWORK=off go test ./... -count=1`; `cd web && pnpm check && npx vitest run && pnpm build && pnpm build-storybook`; then `go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --pages-dir ./examples/widget-pages` and open `/w/recent`.

### Technical details
- New Go deps: rag-evaluation-system v0.1.7 (brings dop251/goja, goja_nodejs).
- Widget HTTP contract byte-compatible with rag-eval (D2): pages list/detail + actions with `{ok, refresh, toast, patch, data, error, fieldErrors}` results.
