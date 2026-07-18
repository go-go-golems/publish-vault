---
Title: Investigation diary
Ticket: PV-VAULT-WIDGETS-016
Status: active
Topics:
    - widget-dsl
    - goja
    - frontend
    - design-system
    - react
    - api
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Diary for PV-VAULT-WIDGETS-016: vault.widgets module, note-domain adapters, NoteHtml, and JS-controlled shell/sidebar."
LastUpdated: 2026-07-17T20:45:49.161469949-04:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Track the implementation of vault.widgets, the note-domain widget adapters, and JS shell/sidebar support.

## Step 1: Ticket and design doc created from conversation decisions

This ticket packages the Horizon-1 increment agreed in the 2026-07-17 conversation on top of PV-WIDGET-DSL-015. The design doc records the three framing decisions verbatim-in-spirit: adapters register existing components (NoteFrontmatter would add nothing to FrontmatterPanel — only NoteHtml is new); a sibling `vault.widgets` module gives first-class-feeling builders without forking rag-eval's widget.dsl (graduating to widget.vault.* when rag-evaluation-system#28 lands RegisterNamespace); and page.shell gets rendered so JS pages control the left sidebar via a VaultLayout sidebar slot. Implementation has not started.

### Prompt Context

**User prompt (verbatim):** "ok, do we have a design doc and ticket?"

**Assistant interpretation:** The vault.widgets / shell increment existed only in conversation; create its ticket + design doc so the work is durable.

**Inferred user intent:** Make the next work item pick-up-able later (or by someone else), consistent with how -015 and issue #28 were documented.

### What I did
- Created PV-VAULT-WIDGETS-016 with design doc (context, target authoring sketch, module/adapter/shell design, 4 decision records, phases A–D, test strategy, risks), 7 tasks, changelog; doctor green.

### What warrants a second pair of eyes
- The §2 `widget.app.shell` builder sketch is UNVERIFIED against v3's actual grammar — Phase C step 3 must true it up before anything depends on it.

### Code review instructions
- Read the design doc §3–§5; cross-check the helper inventory against `internal/vaultdata/vaultdata.go` patterns and the adapter list against `web/src/widgets/defaultRegistry.ts`.

## Step 2: Implement phases A–D (vault.widgets, adapters, shell, dogfood)

All four phases landed in one pass. Phase A: `internal/vaultwidgets` registers `vault.widgets` with six helpers emitting plain IR component nodes; `vw.backlinks` resolves slugs to titles/excerpts server-side via the snapshot provider (D4). Verified the real v3 shell grammar before writing anything: `widget.app.shell(s => s.navigation(nav => nav.placement("sidebar").section(id, label, items => items.item(id, label, action))))` — the design doc's §2 sketch (`.sidebar(...)`) was wrong and the doc/examples now use the real form. A deterministic reader-page golden (UPDATE_GOLDEN=1 regen pattern) pins the full IR; modTime is redacted because the test vault's file mtimes change per run — the first golden run failed on exactly that.

Phase B: six adapters — five over existing components (FrontmatterPanel, BreadcrumbBar, BacklinksPanel, TagCloud with string/TagCount normalization, NoteCard) plus the new `NoteHtml` organism composing NoteBody + the noteEnhancements pipeline with per-flag gating; wiki-link clicks dispatch navigate actions through the render context instead of a hardcoded callback. Phase C: `VaultLayout` gained a `sidebar` slot (desktop panel + mobile drawer), `resolvePageShell` ported into `web/src/widgets/shell.ts` with publish-vault defaults (no shell → vault chrome), `SidebarNav` molecule renders the nav spec retro-style, and a `SidebarSlotContext` lets WidgetPage inject/clear the override. Phase D: pages served from the *default* content-adjacent dir — both `vault-example/.publish/pages` and `~/code/wesen/go-go-golems/go-go-parc/.publish/pages` (981 notes; dot-dir correctly skipped by the vault loader) — with the tmux session now running the reader against the real vault.

### Prompt Context

**User prompt (verbatim):** "go"

**Assistant interpretation:** Execute PV-VAULT-WIDGETS-016 phases A–D per the design doc.

**Inferred user intent:** JS-composable note pages with JS-declared sidebar, proven on the real vault.

### What I did
- Go: `internal/vaultwidgets/` (+5 tests), host registration, `examples/widget-pages/reader.js`, reader-page golden in widgethost (+redaction helper).
- TS: 6 adapters, `NoteHtml` organism + stories, `PvWidgetType`/props extensions, `shell.ts` (+5 tests), `SidebarNav` + story, `sidebarSlot.tsx`, VaultLayout slot, App wiring, WidgetPage shell resolution.
- Fixed two real bugs found by e2e: (1) WidgetPage didn't forward the route query string to the API, so `/w/reader?slug=…` always rendered the default note — now appends `location.search` (mirroring rag-eval); (2) v3 auto-wraps page-level `.view()` nodes in SectionBlocks labeled "Content" — the adapter now renders that structural filler unlabeled so pages keep the article layout.
- Validation: all Go packages green; 21 vitest; tsc; vite client+SSR builds; storybook build; three browser e2e passes (enhancements fire, wiki-link → /note/*, JS sidebar replaces tree and restores on back-nav, sidebar items navigate, ?slug= works); screenshots against go-go-parc.

### What worked
- Verifying the shell grammar from v3.go source before implementing avoided building against the design doc's wrong sketch.
- The reader golden + registry-completeness fixture pair caught nothing new after the first run — the adapter surface matched the IR on the first try because both sides were derived from the same golden.

### What didn't work
- First golden run: nondeterministic `modTime` (temp-vault mtimes) → redaction helper.
- First e2e nav click hit the CSS-hidden mobile-drawer nav instance (`:visible` filter needed) — test artifact, not a product bug; the back-nav sidebar restore it appeared to implicate works correctly.
- The `p.view()` "Content" auto-sections initially rendered three "Content" headings between page-level views.

### What was tricky to build
- Getting the JS-declared sidebar into VaultLayout, which wraps the routes: solved with a setter context owned by AppRoutes; WidgetPage sets the element while mounted with an app/sidebar shell and clears on unmount, so note/search routes always get the tree back.

### What warrants a second pair of eyes
- The "Content" auto-label suppression in SectionBlock.widget.tsx is a heuristic tied to v3's lowering; if v3 ever localizes or renames the auto-label, sections silently regain headings (harmless) — but an authored section literally titled "Content" with no caption also loses its heading (rare; documented in the adapter comment).
- `useSidebarOverride` depends on a stable element identity per page render; the `useMemo` on `[shell, handleAction]` keeps re-set churn down but reruns on every page refetch.

### What should be done in the future
- Remaining -016 tasks: `vw.noteHtml(slug)` convenience decision; NoteView-on-NoteHtml unification (separate ticket).
- `shell: none` full-bleed rendering (v1 renders vault chrome).

### Code review instructions
- Go: `internal/vaultwidgets/vaultwidgets.go`, `internal/widgethost/reader_golden_test.go`.
- TS: `web/src/widgets/shell.ts` + `sidebarSlot.tsx`, `web/src/components/organisms/NoteHtml/NoteHtml.tsx`, `SectionBlock.widget.tsx` (auto-label rule), `WidgetPage.tsx` (query forwarding).
- Validate: `GOWORK=off go test ./... -count=1`; `cd web && pnpm check && npx vitest run && pnpm build && pnpm build-storybook`; serve vault-example (no --pages-dir; uses vault-example/.publish/pages) and open `/w/reader`, click sidebar items and wiki links.
