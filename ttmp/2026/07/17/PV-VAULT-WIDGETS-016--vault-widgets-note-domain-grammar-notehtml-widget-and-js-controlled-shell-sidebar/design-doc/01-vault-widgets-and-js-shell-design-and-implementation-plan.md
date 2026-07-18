---
Title: 'vault.widgets and JS shell: design and implementation plan'
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
DocType: design-doc
Intent: long-term
RelatedFiles:
    - Path: repo://internal/vaultdata/vaultdata.go
      Note: Existing vault.data module; vault.widgets follows the same Registrar/JSON-round-trip pattern
    - Path: repo://internal/widgethost/widgethost.go
      Note: Host where the new module registers; newRuntime() gains vault.widgets
    - Path: repo://web/src/widgets/defaultRegistry.ts
      Note: Registry gaining the note-domain adapters
    - Path: repo://web/src/components/organisms/NoteView/noteEnhancements.ts
      Note: Enhancement pipeline reused by the NoteHtml adapter
    - Path: repo://web/src/components/pages/WidgetPage/WidgetPage.tsx
      Note: Route component gaining shell resolution / sidebar override
    - Path: repo://web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: Gains a sidebar slot prop for JS-declared navigation
ExternalSources:
    - https://github.com/go-go-golems/rag-evaluation-system/issues/28 (widget-dsl extraction; supersedes the sibling-module approach when it lands)
    - /home/manuel/code/wesen/go-go-golems/rag-evaluation-system/packages/rag-evaluation-site/src/app/App.tsx (resolvePageShell reference)
Summary: "Focused increment on PV-WIDGET-DSL-015: a vault.widgets goja module providing note-domain building blocks (NoteHtml, frontmatter, backlinks, breadcrumb, tag list) as first-class-feeling builders without forking widget.dsl, plus page.shell support so JS pages control the left sidebar. Ends with dogfooding on the go-go-parc vault."
LastUpdated: 2026-07-17T19:30:00-04:00
WhatFor: "Execution guide for making note pages and navigation expressible from JS page scripts."
WhenToUse: "Before implementing vault.widgets, note-domain adapters, or shell/sidebar rendering for widget pages."
---

# vault.widgets and JS shell: design and implementation plan

## 1. Context and problem statement

PV-WIDGET-DSL-015 (phases 0–4, shipped 2026-07-17) gave publish-vault an embedded goja widget host speaking rag-evaluation-system's `widget.dsl` v3, a read-only `vault.data` module, and a ported IR renderer with ten generic adapters (Stack, SectionBlock, DataTable, KeyValueStrip, …). What it did **not** give us:

1. **Note-domain widgets.** The note page's parts (rendered HTML with enhancements, frontmatter, backlinks, breadcrumb, tags) exist as React components but are not IR-addressable. A JS page script cannot compose a note reader.
2. **First-class authoring for those widgets.** `widget.dsl` is rag-eval's module imported wholesale (PV-WIDGET-DSL-015 Decision D1); v3 has no namespace extension API, so we cannot mount `widget.vault.*` inside it. The sanctioned escape hatch is `widget.raw.component(type, props)` — functional but ugly in user code.
3. **JS-controlled navigation.** `page.shell` (v3's app/none/root-owned shell spec with navigation) is carried as opaque data by our `WidgetPage`; the left sidebar is always the vault file tree.

Three conversation-level decisions (2026-07-17) frame this ticket:

- **No new components where none are needed.** "NoteFrontmatter adds nothing to FrontmatterPanel" — adapters register *existing* molecules under their own names. Only `NoteHtml` is genuinely new.
- **Sibling module now, first-class namespace later.** We ship `require("vault.widgets")` whose Go helpers *emit* raw component nodes so `raw.component` never appears in user code. When rag-evaluation-system#28 lands `RegisterNamespace`, `vault.widgets` graduates to `widget.vault.*` (docs + .d.ts integration) — this ticket is the explicitly "less elegant part just to get going".
- **Article layout, not window chrome**, for top-level composition (already landed: SectionBlock `article` variant, commit 32f5e09).

Out of scope: action write-verbs (`vault.setFrontmatter`, reload — open question from -015, decide separately), the `/api/v1` page model (PV-BACKEND-API-001), SSR/agent-markdown for widget pages, replacing the built-in `/note/*` routes.

## 2. Target authoring experience

```js
const widget = require("widget.dsl");     // rag-eval grammar (unchanged)
const vault  = require("vault.data");     // vault content (unchanged)
const vw     = require("vault.widgets");  // NEW: note-domain building blocks

const note = vault.note(request.query.slug || "index");

const page = widget.page(note.title, (p) =>
  p.id("reader")
   .shell(widget.app.shell((s) =>                        // NEW: rendered, not ignored
      s.sidebar((nav) =>
        nav.section("Reading", (sec) =>
          sec.item("recent", "Recently updated", widget.act.navigate("/w/recent"))
             .item("tags", "Tags", widget.act.navigate("/w/tags"))))))
   .section(note.title, (s) =>
      s.view(vw.breadcrumb(note))
       .view(vw.frontmatter(note))
       .view(vw.noteHtml(note, { embeds: true, anchors: true })))
   .section("Linked mentions", (s) =>
      s.view(vw.backlinks(note, { onSelect: widget.act.navigate("/note/${slug}") }))));
```

Notes on the sketch:
- `vw.*` helpers return plain IR node maps (`{kind:"component", type, props}`) — exactly what `widget.raw.component` produces — so they compose with `.view()`, `use()`, and any v3 builder that accepts nodes.
- Verify the exact `widget.app.shell` builder surface against `pkg/widgetdsl/v3.go` (`v3AppObject`) before implementing; the sketch's `.sidebar/.section/.item` chain must be replaced by whatever v3 actually exposes (check examples `06-admin-course-cms.js`, `13-course-shell-layout.js`).
- Copy/download actions need no widgets: `widget.act.copy(...)` / `widget.act.download(...)` on plain buttons.

## 3. Design

### 3.1 Go: `internal/vaultwidgets` (module `vault.widgets`)

Same shape as `internal/vaultdata` (`Register(reg, provider, config)`; JSON-safe return values). Each helper validates its input map minimally and returns a node map:

```go
// vw.noteHtml(note, {embeds?, anchors?, highlight?, mermaid?}) → ComponentNode
exports.Set("noteHtml", func(note map[string]any, opts ...map[string]any) map[string]any {
    o := firstOpts(opts)
    return node("NoteHtml", map[string]any{
        "html":      note["html"],
        "slug":      note["slug"],
        "embeds":    boolOpt(o, "embeds", true),
        "anchors":   boolOpt(o, "anchors", true),
        "highlight": boolOpt(o, "highlight", true),
        "mermaid":   boolOpt(o, "mermaid", true),
    })
})
```

Helper inventory (v1):

| Helper | Emits type | Props | Renders via |
|---|---|---|---|
| `vw.noteHtml(note, opts)` | `NoteHtml` | html, slug, enhancement flags | NEW adapter (see 3.2) |
| `vw.frontmatter(note)` | `FrontmatterPanel` | frontmatter, tags, modTime, onTagClick action | existing molecule |
| `vw.breadcrumb(note)` | `BreadcrumbBar` | segments[] (derived from note.path) | existing molecule |
| `vw.backlinks(note, {onSelect})` | `BacklinksPanel` | entries[{slug,title,excerpt}], onSelect action | existing organism |
| `vw.tagList(tags, {onSelect})` | `TagCloud` | tags[], onSelect action | existing molecule |
| `vw.noteCard(noteListItem, {onSelect})` | `NoteCard` | title, excerpt, tags, onSelect | existing molecule |

`vw.backlinks(note)` resolves backlink slugs to `{slug,title,excerpt}` server-side via the snapshot provider (Decision D4) instead of forcing scripts to loop `vault.note(slug)` per backlink.

Ship a `TypeScriptDeclarer` (`go-go-goja/modules/typing.go` pattern) even though .d.ts aggregation is manual pre-#28.

### 3.2 Frontend: note-domain adapters

- **Existing components, registered as-is** (10-line `.widget.tsx` each): `FrontmatterPanel`, `BreadcrumbBar`, `BacklinksPanel`, `TagCloud`, `NoteCard`. Callback props become dispatched ActionSpecs with context (`{tag}` for tag clicks, `{slug}` for select) — mirror the DataTable adapter's `ctx.dispatchAction` pattern.
- **`NoteHtml` (new)** — the only real component work. Composes the pieces PV-WIDGET-DSL-015 phase 2 deliberately made injectable:
  - `NoteBody` hosts the HTML;
  - `noteEnhancements.ts` functions run per the boolean props (mermaid → highlight/copy → embeds → anchors, same ordering constraint);
  - embed loading goes through the same RTK `getNote` loader NoteView uses;
  - **wiki-link clicks dispatch actions** instead of calling a hardcoded navigate callback: intercepted `a.wiki-link` clicks become `ctx.dispatchAction({kind:"navigate", to:"/note/" + target}, {slug: target})`. This makes link behavior IR-consistent and overridable by the page's `onAction`.
  - **Lightbox**: v1 keeps a local `LightboxModal` inside the adapter (Decision D3).
  - NoteView itself is NOT rewritten on top of NoteHtml in this ticket (risk containment); a follow-up may unify them.

### 3.3 Shell / sidebar-from-JS

- `VaultLayout` gains an optional `sidebar?: ReactNode` prop; default stays the current search-plus-tree `Sidebar` organism. The mobile drawer renders the same slot.
- Port `resolvePageShell` semantics from rag-eval's `App.tsx` (kind `app` with navigation placement, `none`, `root-owned`; legacy `meta.shell` fallback) into `web/src/widgets/shell.ts` — a trimmed copy with a provenance header, same policy as the other ported files (deleted when #28's headless package ships it).
- New `SidebarNav` molecule renders a navigation spec with retro styling (`retro-tree-item` idiom, `Caption` section headers); item actions dispatch through the page's `onAction`.
- `WidgetPage` resolution:
  - no shell / `root-owned` → current behavior (file-tree sidebar);
  - `app` + sidebar placement → `VaultLayout sidebar={<SidebarNav …/>}`;
  - `none` → render without VaultLayout chrome (full-bleed page; best-effort in v1 if it fights the router layout).
  - The shell element must travel from the routed page up to `VaultLayout` — implement by lifting shell resolution into `AppRoutes` (which owns the `VaultLayout` wrapper) or via a small context; decide at implementation time, prefer whatever keeps SSR tests green.
- Keyboard shortcuts (`page.shortcuts`) stay deferred.

### 3.4 Dogfooding (definition of done)

1. Author real pages in `~/code/wesen/go-go-golems/go-go-parc/.publish/pages/` (reader.js using `vw.*`, plus the recent/tags pages) and serve with the default pages dir.
2. Verify `.publish/` interactions: vault loader must skip the dot-directory (hidden-dir skip in `internal/vault/vault.go`), `.vault-ignore` (RETRO-IGNORE-013) must not need changes, git-sync deployment picks the directory up.
3. tmux session serving go-go-parc with a JS note reader at `/w/reader?slug=index` navigable via a JS-declared sidebar.

## 4. Decision records

### Decision D1: Sibling module `vault.widgets`, not a widget.dsl fork or wait-for-#28

- **Context:** v3 has no namespace extension API; issue #28 will add one but is unscheduled.
- **Options considered:** fork widget.dsl to add a vault namespace; wait for #28; sibling module emitting raw component nodes.
- **Decision:** Sibling module now; graduate to `widget.vault.*` via `RegisterNamespace` when #28 lands.
- **Rationale:** Zero fork risk; user code already reads like the final form (`vw.noteHtml(note)`), so graduation is a require-path change in scripts.
- **Consequences:** Second-class docs/.d.ts until #28.
- **Status:** accepted (user-confirmed 2026-07-17).

### Decision D2: Register existing components under their own names; only NoteHtml is new

- **Context:** "What would NoteFrontmatter add to FrontmatterPanel?" — nothing.
- **Options considered:** new Note* wrapper components vs adapters over existing ones.
- **Decision:** Adapter type names = component names (`FrontmatterPanel`, `BreadcrumbBar`, `BacklinksPanel`, `TagCloud`, `NoteCard`); one new type `NoteHtml`.
- **Rationale:** The adapter's only job is IR addressability (serialized props + action wiring); duplicating components would be pure noise.
- **Consequences:** IR vocabulary matches the design system 1:1; component renames become breaking IR changes (acceptable — we own both sides).
- **Status:** accepted.

### Decision D3: NoteHtml keeps a local lightbox in v1; overlay actions are a follow-up

- **Context:** NoteView owns lightbox state; IR purity says emit `openOverlay` actions, but no overlay host exists yet at WidgetPage level.
- **Options considered:** local LightboxModal in the adapter; build an overlay host now.
- **Decision:** Local `LightboxModal` inside the NoteHtml adapter for v1.
- **Rationale:** Identical UX to note pages with zero new infrastructure.
- **Consequences:** One more thing to migrate when an overlay host lands.
- **Status:** proposed.

### Decision D4: `vault.widgets` takes the snapshot provider (for backlink resolution)

- **Context:** `vw.backlinks(note)` needs titles/excerpts for backlink slugs; making scripts loop `vault.note()` is boilerplate and O(n) VM↔Go crossings.
- **Options considered:** pure transform helpers; provider-backed helpers.
- **Decision:** Module constructor mirrors vaultdata's (`Register(reg, provider, config)`); helpers may consult the snapshot.
- **Rationale:** Server-side resolution is one map lookup per backlink and keeps scripts declarative.
- **Consequences:** Helpers are not pure transforms; they stay read-only and snapshot-consistent per call.
- **Status:** proposed.

## 5. Implementation plan

### Phase A — `vault.widgets` module (~0.5 day)
1. `internal/vaultwidgets/vaultwidgets.go` + script-driven tests (pattern: `internal/vaultdata/vaultdata_test.go`); register in `widgethost.newRuntime()`.
2. Golden test: a page script using every helper → checked-in IR JSON.

### Phase B — adapters + NoteHtml (~1 day)
1. `.widget.tsx` for the five existing components; extend `PvWidgetType` + props; add to `defaultRegistry`; registry-completeness fixture regenerated from the new golden.
2. `NoteHtml` adapter per 3.2 (wiki-link clicks → actions; enhancement flags; RTK embed loader; local lightbox); stories incl. a full reader-page story from the golden.
3. Validation: vitest, builds, storybook; browser check that mermaid/hljs/embeds/anchors fire inside a widget page and wiki-link click navigates.

### Phase C — shell/sidebar (~1 day)
1. `VaultLayout` sidebar slot; `SidebarNav` molecule + story.
2. `web/src/widgets/shell.ts` (ported resolvePageShell, provenance header); wire into the `/w/` route per 3.3.
3. Verify the actual v3 shell-builder grammar against rag-eval examples and true up the §2 sketch and `vw` docs.
4. Validation: SSR tests stay green (shell only affects client-rendered `/w/` routes); browser check sidebar swap + nav item navigation.

### Phase D — dogfood (~0.5 day)
Per 3.4; record findings (especially `vault.data`/`vault.widgets` API gaps) in the diary; file follow-ups.

## 6. Test strategy

- Go: script-driven module tests; one end-to-end golden (reader page) under `internal/widgethost` goldens.
- TS: registry completeness against the new fixture; shell resolution unit tests (kind none/app/root-owned/legacy-meta); NoteHtml enhancement-flag tests if jsdom becomes installable (pnpm store still read-only as of 2026-07-17 — else browser-script checks as in -015).
- E2E: bundled-playwright script — reader page renders note HTML with enhancements, backlink click navigates, JS sidebar replaces the tree and navigates.

## 7. Risks and open questions

- **v3 shell grammar mismatch:** the `widget.app.shell` sketch in §2 is unverified; Phase C step 3 exists precisely to true it up. Worst case, pages set `meta.shell` (the legacy path resolvePageShell already handles).
- **`shell: none` vs App-level VaultLayout wrapper:** may need route-level restructuring; keep `none` best-effort in v1.
- **Backlink resolution cost:** `vw.backlinks` touches N notes per render; fine at vault scale (in-memory lookups), note it in code.
- Open: should `vw.noteHtml` also accept a slug (helper fetches html itself)? Lean yes — trivial with D4's provider.
- Open: unify NoteView on top of the NoteHtml adapter afterwards? (Deduplicates pipeline wiring; separate ticket.)

## 8. References

- Parent: `ttmp/2026/07/17/PV-WIDGET-DSL-015--*/design-doc/01-*.md` (architecture, decisions D1–D6, §7.3 host design) and its diary steps 5–6 (what was copy-ported and why).
- Extraction/graduation path: https://github.com/go-go-golems/rag-evaluation-system/issues/28.
- Shell reference: `rag-evaluation-system/packages/rag-evaluation-site/src/app/App.tsx` (resolvePageShell), examples `06-admin-course-cms.js`, `13-course-shell-layout.js`.
- Module pattern: `internal/vaultdata/vaultdata.go`; adapter pattern: `web/src/components/molecules/DataTable/DataTable.widget.tsx`.
