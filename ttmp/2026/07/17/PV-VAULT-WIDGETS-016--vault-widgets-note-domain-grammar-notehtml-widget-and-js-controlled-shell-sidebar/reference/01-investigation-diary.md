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

## Step 3: Hunt and fix the disappearing-enhancements bug (hash links)

The user reported that opening a note URL with a `#heading` anchor made the injected heading-anchor links disappear. The hunt went through five instrumented browser experiments and ended somewhere much bigger than the report: **any React re-render of the note body silently destroyed every DOM enhancement** (heading anchors, copy buttons, mermaid SVGs, resolved embeds). Toggling the right panel alone dropped half the anchors (88→44). Hash navigation was just the most reproducible trigger because same-document navigations fire popstate, which re-renders the router tree.

Two independent defects compounded:

1. **`entry-client.tsx` called `hydrateRoot` unconditionally.** The local/dogfood server runs without the SSR sidecar, so `#root` is empty; React 19 "hydrates" an empty container, throws error #418, and performs a recovery client render — nondeterministic re-render timing that made the symptom flaky between runs. Fixed: `root.hasChildNodes() ? hydrateRoot(...) : createRoot(root).render(...)`.
2. **React 19 re-applies `dangerouslySetInnerHTML` on every re-render.** Instrumenting `Element.prototype.innerHTML` with stack traces showed `commitUpdate → updateProperties → setProp` rewriting the note body ~60ms after each hash change — with `resolvedHtml` UNCHANGED and none of the `[resolvedHtml]`-keyed enhancement effects re-running (dev-mode component logging proved both). The `{__html}` prop object is new each render, React rewrites the DOM, and the enhancements are gone with no effect to restore them. Fixed structurally: `NoteBody` is now `memo()`-ized on the `html` string, with a comment stating the memo is load-bearing, not an optimization — the DOM may only be rewritten when the content changes, because that is exactly when the effects re-run.

This bug predates the NoteView refactor (NoteRenderer had the identical structure), and probably explains any historical "sometimes the copy buttons are missing" weirdness.

### Prompt Context

**User prompt (verbatim):** "consolidate. also:

there's a bug btw where when you have a link with #... then the anchor links actually disappear on the page. http://localhost:8080/note/research/playbooks/filter-git-history-big-files#filtering-git-history-for-big-files--how-we-do-it"

**Assistant interpretation:** Two tasks — do the agreed consolidation (NoteView on NoteHtml, vw.noteHtml(slug)), and fix the anchor-disappearance bug; the bug was diagnosed first so the fix lands once in the consolidated component.

**Inferred user intent:** Anchored deep-links are how vault notes get shared; the reading experience must survive them.

### What I did
- Reproduced with playwright against the live go-go-parc server: fresh-load-with-hash → 0 anchors/0 copy buttons (initially flaky between runs — the #418 recovery render was the randomizer).
- Narrowed to same-document hash navigation; tracked DOM node identity with markers (nodes survived, children count unchanged, injected elements gone → innerHTML rewrite).
- Intercepted the `innerHTML` setter with stack capture: React `commitUpdate` writing the normalized html ~60ms after nav.
- Instrumented NoteView renders/effects on the vite dev server: 8 renders (4 instances × StrictMode double-log), no sync-reset, no resolve-effect, no anchor-effect — proving React rewrote without a state change.
- Falsified the "hash-specific" framing with the right-panel toggle test (88→44).
- Fixed both defects; removed instrumentation; validated: tsc, client+SSR builds, 21 vitest, then the 3-round scenario matrix on the prod build (fresh-with-hash / no-hash / same-doc-hash / anchor-click × 3 → 88 anchors & 116 copy buttons in every cell, zero page errors) and the ui-toggle test (88→88).

### What worked
- The escalation ladder — timeline sampling → DOM identity markers → innerHTML setter interception → dev-server component logging — each step eliminated a whole class of hypotheses; the setter stack trace was the pivot from "which state changed" to "no state changed at all".

### What didn't work
- Two hours of state-machine reasoning about the two-phase resolvedHtml flow: every hypothesis (sync-reset, slugSet identity churn, RTK refetch, remounts) predicted effect re-runs that the logs disproved. The real cause was below the component layer.

### What was tricky to build
- The flakiness had TWO causes layered: the false-hydration recovery render made results differ run-to-run, masking the deterministic dangerouslySetInnerHTML rewrite underneath. Fixing only the hydration bug made the failure deterministic (which is what finally exposed cause 2).

### What warrants a second pair of eyes
- Whether React's rewrite-on-rerender is expected React 19 semantics or a regression worth reporting upstream; either way the memo() is correct and self-documenting.
- SSR mode is unaffected by the entry-client change (hasChildNodes true → hydrate path identical); the 13 SSR tests and smoke stay the guard.

### What should be done in the future
- Consolidation (this session, next step) puts NoteBody behind one component so this class of bug has a single owner.

### Code review instructions
- `web/src/entry-client.tsx` (conditional hydrate) and `web/src/components/organisms/NoteView/NoteBody.tsx` (memo + load-bearing comment).
- Repro harnesses in scratchpad (`check-hash*.mjs`, `check-rerender.mjs`) show the verification matrix.

## Step 4: Consolidation — one note-rendering path

Both open decisions from the phase A–D wrap-up are now closed in code. `vw.noteHtml` accepts a slug string as well as a note object (unknown slugs throw so authors fail loudly; the helper resolves via the snapshot provider, closing -016's open question). And NoteView is rewritten on top of NoteHtml: it keeps only the page furniture (breadcrumb, title, NoteActions, FrontmatterPanel) and delegates the body — HTML hosting, wiki-link resolution, enhancement pipeline, click delegation, lightbox — to the same NoteHtml component the widget registry renders. The ~150 duplicated lines are gone; the disappearing-enhancements class of bug now has exactly one owner, and the fix from Step 3 (memoized NoteBody) covers `/note/*` and `/w/*` pages through the same code.

API cleanup that fell out: NoteView's `allSlugs` prop is removed (NoteHtml resolves broken links via its own `useListNotesQuery`, same RTK cache entry NotePage already subscribes to); NotePage and the stories were updated.

### Prompt Context

**User prompt (verbatim):** (see Step 3 — "consolidate. also: there's a bug…")

**Assistant interpretation:** Execute the agreed consolidation now that the bug fix landed.

**Inferred user intent:** One canonical note-rendering path before any further widget work builds on it.

### What I did
- `internal/vaultwidgets`: noteHtml(slug|note) with typed error paths; +1 test (slug resolution + unknown-slug throw).
- `NoteView.tsx` rewritten (166→86 lines) delegating to NoteHtml; `allSlugs` prop removed from NoteViewProps/NotePage/stories.
- Validation: all Go tests; 21 vitest; tsc; client+SSR+storybook builds; `smoke:ssr` PASS (0 console warnings/errors — SSR/hydration parity held through the rewrite); 3-round hash/enhancement matrix still all-green; live go-go-parc e2e (sidebar → `/w/reader?slug=<real note>` renders with anchors, zero errors).

### What worked
- Doing the bug fix BEFORE the consolidation meant the memoized NoteBody flowed into the unified path automatically — no second fix site.

### What didn't work
- The reader e2e initially failed on stale fixture expectations ("Zettelkasten" is a vault-example note, not go-go-parc); adjusted the check to click whatever latest-note the sidebar actually lists.

### What warrants a second pair of eyes
- NoteView no longer receives `allSlugs`; broken-link marking now depends on NoteHtml's own notes subscription. On SSR, that query has no preloaded data on note routes (same as before — the old path's `allSlugs` was empty there too), so behavior is unchanged, but it's worth knowing the equivalence argument.
- The hash-scroll effect now lives only in NoteHtml (keyed on its `slug` prop); NotePage hash deep-links verified in the matrix.

### What should be done in the future
- Remaining -016 nicety: `shell: "none"` full-bleed rendering (still renders vault chrome in v1).

### Code review instructions
- Diff `NoteView.tsx` against its pre-step-4 version (git) — everything removed should exist in `NoteHtml.tsx`.
- `internal/vaultwidgets/vaultwidgets.go` noteHtml switch; test `TestNoteHtmlAcceptsSlug`.
- Validate: full suites plus `pnpm smoke:ssr`; open a note with `#heading` and toggle the right panel — enhancements must survive both.
