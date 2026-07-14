---
Title: Bundle size reduction — analysis, design and implementation guide
Ticket: PERF-BUNDLE-014
Status: active
Topics:
    - bundle
    - frontend
    - obsidian-vault
    - performance
    - retro-obsidian-publish
    - ssr
    - vite
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://web/server.mjs
      Note: Lines 196-221 unconditional prefetch of notes+tree inlined into every HTML — change in Phase 4
    - Path: repo://web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Lines 9-10 static import hljs + mermaid — the heaviest contributors; change in Phases 2/3
    - Path: repo://web/src/entry-server.tsx
      Note: preloadCache seeds RTK cache; already tolerates null data — safe for Phase 4
    - Path: repo://web/src/store/vaultApi.ts
      Note: RTK Query endpoints and tags; defines the API contract the SSR preload must match
    - Path: repo://web/vite.config.ts
      Note: Single entry, no manualChunks, debug plugins unconditional — the config to change in Phases 1/5
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T17:05:15.035392639-04:00
WhatFor: ""
WhenToUse: ""
---












# Bundle Size Reduction — Analysis, Design and Implementation Guide

> **Audience.** You are a new engineer joining the `publish-vault` project. You have working
> literacy in React, TypeScript, Vite and SSR, but you have *no* prior context on this codebase.
> This document gives you everything you need to understand *why* the production site is heavy,
> *what* the system does, and *exactly how* to fix it in a sequence of safe, reviewable steps.
> Read sections 1–3 for orientation, then 4–8 for the build.

## 1. Executive summary

The published site at `https://parc.yolo.scapegoat.dev/` ships two independent size problems
that compound on every page load:

1. **One giant JavaScript bundle** — `main-CZnSi4zO.js` is **2.03 MB raw / 599 KB gzipped**.
   There is no code splitting at all: the entire application, including two large libraries
   (`mermaid` and a full `highlight.js`), is concatenated into a single chunk that the browser
   must download and parse before any page becomes interactive — even the home page and search,
   which never use those libraries.
2. **SSR inlines the whole vault into every HTML response** — the root document is
   **1.31 MB**, of which ~834 KB is a serialized `window.__PRELOADED_STATE__` blob containing the
   full 934-note list (~550 KB) and the full file tree (~284 KB). This payload is attached to
   every route, including note pages that only need one note.

Both problems are fixable with standard, well-supported Vite/React patterns. The expected
outcome after implementing phases 1–4 is roughly:

- Initial JS gzipped: **~599 KB → ~150–200 KB** (home/search), note routes load the rest on demand.
- Root HTML: **1.31 MB → ~30–60 KB** for note pages (the note body + a lean sidebar payload).
- `highlight.js` languages: loaded **per-language on demand** via `import.meta.glob`, so only
  the languages actually present in a note's code blocks are ever downloaded.

The remainder of this document explains the system, proves each claim with file evidence, and
gives phased, copy-pasteable implementation steps with pseudocode and verification commands.

---

## 2. Problem statement and scope

### 2.1 What is `publish-vault`?

`publish-vault` is a read-only web publisher for an Obsidian vault. It renders markdown notes as a
"Retro macOS 1" themed SPA with backlinks, full-text search, wiki-link resolution, syntax
highlighted code blocks, and Mermaid diagrams. The project is a Go backend plus a React/Vite
frontend with a Node SSR sidecar.

The high-level architecture has three cooperating processes:

```
┌──────────────┐   /api/*   ┌──────────────────┐
│   Browser    │◀───────────│   Go backend     │
│  (React SPA  │            │  (vault index,   │
│   hydrated)  │            │   note render,   │
│              │            │   tree, search)  │
└──────┬───────┘            └────────┬─────────┘
       │ HTML (SSR)                   │
       │ + __PRELOADED_STATE__        │
       ▼                              │
┌──────────────────┐                  │
│  Node SSR sidecar│ pre-fetch /api/* │
│  (server.mjs)    │──────────────────┘
│  renders React   │
│  to string       │
└──────────────────┘
```

- **Go backend** (`internal/api/api.go`) serves JSON endpoints: `/api/config`, `/api/notes`,
  `/api/notes/{slug}`, `/api/tree`, `/api/search`, `/api/tags`. It also proxies HTML/asset
  requests to the SSR sidecar.
- **Node SSR sidecar** (`web/server.mjs`) is an Express app that, for each page request,
  pre-fetches the relevant API data, renders the React tree to an HTML string, and inlines the
  pre-fetched RTK Query cache as `window.__PRELOADED_STATE__` so the client hydrates with the
  same data.
- **React SPA** (`web/src/`) is a Vite-built client that hydrates the SSR HTML. State is managed
  with Redux Toolkit (RTK) Query, which caches API responses.

### 2.2 The scope of this ticket

In scope:

- Splitting the client JavaScript bundle so heavy libraries load only on the routes that use them.
- Replacing the full `highlight.js` import with per-language, on-demand loading.
- Lazy-loading Mermaid so it (and its large d3 dependency) only downloads when a note contains a
  diagram.
- Trimming the SSR `__PRELOADED_STATE__` so note pages do not ship the full notes list and tree.

Out of scope (but noted as future work):

- Replacing Redux Toolkit Query with a lighter data layer.
- Replacing Mermaid with a server-side rendering pipeline.
- Removing the Retro theme's CSS (already small at 20 KB gzipped; not a problem).

---

## 3. Current-state architecture (evidence-based)

This section walks the system end to end. Every claim is anchored to a file so you can read the
code yourself.

### 3.1 The client entry point and the single chunk

The client entry is `web/src/entry-client.tsx`:

```tsx
// web/src/entry-client.tsx (abridged)
import { hydrateRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { AppRoutes } from "./App";
import "./index.css";

const preloadedState = window.__PRELOADED_STATE__;
delete window.__PRELOADED_STATE__;
const store = makeStore(preloadedState);
hydrateRoot(root, <Provider store={store}><BrowserRouter><AppRoutes /></BrowserRouter></Provider>);
```

Two facts matter here. First, **the entire route tree is imported statically** at the top of
`App.tsx`, so Vite puts every page into one chunk. Second, there is **no `React.lazy`** anywhere
in `src/`:

```bash
$ rg -n "React.lazy|lazy\(" web/src/ | wc -l
0   # zero lazy components
$ rg -n "import\(" web/src/ | head -1
web/src/store/vaultApi.ts:31:  const m = await import("../vault/staticVault");
```

The only dynamic `import()` in the codebase is the static-vault demo fallback; production routes
have none.

### 3.2 The Vite build config — no splitting at all

`web/vite.config.ts` defines the build with a single entry and no chunk-splitting configuration:

```ts
// web/vite.config.ts (build section)
build: {
  outDir: path.resolve(WEB_ROOT, "dist"),
  emptyOutDir: true,
  rollupOptions: {
    input: { main: path.resolve(WEB_ROOT, "index.html") },
  },
},
```

There is no `output.manualChunks`, no `splitVendorChunkPlugin`, and no route-level code splitting.
Consequently Vite emits exactly one JS chunk for the entire application. This is verified against
the live site: the network waterfall shows a single `main-CZnSi4zO.js`.

### 3.3 The two heavy static imports

Both heavy libraries are imported at module top level in a single component,
`web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`:

```ts
// web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:9-10
import hljs from "highlight.js";     // imports ALL ~190 languages
import mermaid from "mermaid";       // imports mermaid + d3 + renderer
```

These two lines are the dominant contributor to the 2.03 MB bundle. Note that `NoteRenderer` is
**only used on `/note/*` routes** (it is imported by `web/src/components/pages/NotePage/NotePage.tsx`),
yet its dependencies are shipped to the home page and `/search` because the whole app is one chunk.

A useful negative result: `web/src/components/ui/chart.tsx` does `import * as RechartsPrimitive from "recharts"`,
but **nothing imports `chart.tsx`**, so Rollup tree-shakes it out. Do not chase recharts — it is
not in the bundle. (Verified: `rg -rn "from ['\"].*ui/chart" web/src/` returns nothing.)

### 3.4 How `highlight.js` is used today

`NoteRenderer` uses highlight.js to highlight every `<pre><code>` block after Mermaid has
consumed its own blocks:

```tsx
// web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:181-195
useEffect(() => {
  const el = contentRef.current;
  if (!el) return;
  const codeBlocks = el.querySelectorAll<HTMLElement>("pre code:not(.language-mermaid)");
  codeBlocks.forEach((block) => {
    if (!block.dataset.highlighted) {
      hljs.highlightElement(block);   // auto-detects language
      block.dataset.highlighted = "true";
    }
  });
  // ... copy buttons ...
}, [resolvedHtml]);
```

The import `from "highlight.js"` resolves to the package's main entry, which **registers every
language definition that ships with the library** (around 190 languages, ~700 KB–1 MB minified).
The code only ever calls `hljs.highlightElement(block)` and lets highlight.js auto-detect the
language from the `language-xxx` class. None of this requires all languages to be present up front.

### 3.5 How Mermaid is used today

Mermaid runs inside a `useEffect` that queries for `code.language-mermaid` blocks, initializes
Mermaid once (lazily via a module-level `mermaidInitialized` flag), and replaces each block with
its rendered SVG:

```tsx
// web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:137-179
useEffect(() => {
  const el = contentRef.current;
  if (!el) return;
  const blocks = el.querySelectorAll<HTMLElement>("code.language-mermaid");
  if (blocks.length === 0) return;             // ← early-out when no diagrams
  if (!mermaidInitialized) { mermaid.initialize({ ... }); mermaidInitialized = true; }
  blocks.forEach((block) => {
    // ... mermaid.render(id, src) → replace <pre> with SVG ...
  });
}, [resolvedHtml]);
```

There is already an early-out (`if (blocks.length === 0) return;`) — so the *runtime* cost is
avoided on notes without diagrams, but the *download* cost is not, because the `import mermaid`
is static and top-level. Mermaid pulls in d3 and a substantial renderer; in the built chunk, `d3`
patterns appear 53 times and `mermaid` 17 times.

### 3.6 The SSR preloaded state — the HTML bloat

The SSR sidecar (`web/server.mjs`) pre-fetches three "common" API responses on **every** request,
regardless of route, then passes them into `renderApp`:

```js
// web/server.mjs:196-221 (abridged)
const [config, notes, tree] = await Promise.all([
  fetchAPI("/api/config"),
  fetchAPI("/api/notes"),   // 934-note list
  fetchAPI("/api/tree"),    // full file tree
]);
// ... then route-specific note ...
const { html, preloadedState } = await renderApp(url, { config, notes, tree, note });
const serializedPreloadedState = serializeForInlineScript(preloadedState);
// ... later inlined as:
// <script>window.__PRELOADED_STATE__=${serializedPreloadedState};</script>
```

`renderApp` (in `web/src/entry-server.tsx`) seeds the RTK Query cache with these values via
`vaultApi.util.upsertQueryData`, so the SSR-rendered React tree sees real data. The resulting
Redux state is then serialized and inlined into the HTML.

Measured against the live `https://parc.yolo.scapegoat.dev/` payload:

```
window.__PRELOADED_STATE__ breakdown:
  getConfig(undefined)      →  <1 KB     (vaultName, pageTitle, note count)
  listNotes(undefined)      →  ~550 KB   (934 NoteListItem objects)
  getTree(undefined)        →  ~284 KB   (recursive FileNode tree)
  getNote("index")         →  ~3 KB     (the one note the home route renders)
  ─────────────────────────────────────
  total inlined JSON        →  ~834 KB   (inside a 1.31 MB HTML document)
```

The `notes` and `tree` payloads exist to feed the sidebar (`web/src/components/pages/VaultLayout/VaultLayout.tsx:68`
calls `useGetTreeQuery()`). That is a legitimate need on routes that render the sidebar, but it
means a note page that only displays one note still ships ~834 KB of unrelated data in the HTML.

### 3.7 The data layer — RTK Query

State is managed by Redux Toolkit Query (`web/src/store/vaultApi.ts`). Each endpoint has two
modes: a `queryFn` backed by an in-browser static vault (for the `VITE_STATIC_VAULT` demo
build), or a plain `query` against the Go backend. The relevant endpoints:

| Endpoint | RTK Query call | Backend route | Provides |
|---|---|---|---|
| `getConfig` | `useGetConfigQuery()` | `GET /api/config` | `["Config"]` |
| `listNotes` | `useListNotesQuery()` | `GET /api/notes` | `["Notes"]` |
| `getNote` | `useGetNoteQuery(slug)` | `GET /api/notes/{slug}` | `[{ type: "Note", id: slug }]` |
| `getTree` | `useGetTreeQuery()` | `GET /api/tree` | `["Tree"]` |
| `search` | `useSearchQuery(q)` | `GET /api/search?q=` | — |
| `listTags` | `useListTagsQuery()` | `GET /api/tags` | `["Tags"]` |

Go routes are registered in `internal/api/api.go:60-66`. The SSR sidecar pre-fetches by hitting
these same endpoints directly with `fetch`.

### 3.8 Measured live sizes (reference baseline)

Captured against the production deployment on 2026-07-14:

| Asset | Raw | Gzipped |
|---|---|---|
| `/assets/main-CZnSi4zO.js` | 2,030,018 B (2.03 MB) | 598,884 B (599 KB) |
| `/assets/main-C9LScDXt.css` | 125,549 B (126 KB) | 20,225 B (20 KB) |
| `/` HTML document | 1,305,031 B (1.31 MB) | (not measured) |

Only three network requests are made for the root page (JS + CSS + favicon). The bloat is
concentrated, not fragmented — which makes the diagnosis clean: shrink the single JS chunk and
shrink the inlined SSR state.

---

## 4. Gap analysis

| Gap | Current behavior | Target behavior | Evidence |
|---|---|---|---|
| No JS code splitting | One 599 KB-gzipped chunk for all routes | Route-level chunks; heavy libs only on `/note/*` | `vite.config.ts` build block; `rg "React.lazy"` = 0 |
| Full highlight.js | `import hljs from "highlight.js"` registers ~190 languages | Per-language dynamic import on demand | `NoteRenderer.tsx:9` |
| Static Mermaid import | `import mermaid from "mermaid"` in every bundle | `await import("mermaid")` on first diagram | `NoteRenderer.tsx:10`, d3 appears 53× in chunk |
| SSR ships full vault | `notes` + `tree` inlined on every route | Note pages ship only `config` + `note`; sidebar data fetched by client | `server.mjs:196-221` |
| Debug plugins in prod build | `jsx-loc` and `manus` runtime plugins always enabled | Guarded behind `mode !== "production"` | `vite.config.ts` plugins array |

---

## 5. Proposed architecture and APIs

### 5.1 Bundle splitting strategy

The goal is a chunk graph where the initial load (home/search) is small and the note route's
heavy dependencies load on demand. Three mechanisms work together:

1. **Route-level `React.lazy`** for `NotePage` (and optionally `SearchPage`). This makes Vite
   emit a separate chunk per route that loads when the user navigates there.
2. **`build.rollupOptions.output.manualChunks`** to split large, stable vendors (React, Redux,
   react-router) into their own cacheable chunks, and to isolate the truly heavy libs.
3. **Dynamic `import()` for Mermaid** so it becomes its own chunk loaded only when a note
   contains a `language-mermaid` block.

Target chunk graph:

```
index.html
  └─ main.js              ← app shell, router, layout, RTK Query (~120-150 KB gz)
  └─ vendor-react.js      ← react, react-dom, react-router, redux (cacheable)
  └─ (on navigate to /note/*)
       └─ NotePage.js     ← route chunk
            └─ NoteRenderer.js
                 └─ (on first mermaid block)
                      └─ mermaid.js   ← mermaid + d3, ~300 KB gz, on demand only
                 └─ (per language-xxx block)
                      └─ hl-<lang>.js ← ~5-15 KB gz each, on demand only
```

### 5.2 highlight.js per-language on-demand loading

highlight.js ships two entry points:

- `highlight.js` — the full build with every language registered (~big).
- `highlight.js/lib/core` — the engine with **no** languages registered (~small). You register
  languages individually.

The supported, idiomatic Vite pattern for loading language definitions on demand is
`import.meta.glob` (this is the exact use case described in Vite issue #1903 and the Vite docs'
"Dynamic Import" feature). `import.meta.glob` returns a map of module paths to lazy importers;
Vite splits each matched file into its own chunk by default.

**New module: `web/src/lib/highlightLanguages.ts`** (proposed API):

```ts
// Pseudocode for the on-demand language loader
import hljs from "highlight.js/lib/core";

// Map every language definition to a lazy importer. Vite emits one chunk per language.
// { eager: false } (default) means each language is fetched only when first needed.
const modules = import.meta.glob("../vendor/highlight-languages/*.js");

// Friendly alias → glob path. We curate a small, predictable set instead of
// exposing highlight.js's internal path layout directly.
const ALIASES: Record<string, string> = {
  ts: "typescript", tsx: "typescript", js: "javascript", jsx: "javascript",
  go: "go", py: "python", python: "python", bash: "bash", sh: "bash",
  shell: "bash", json: "json", yaml: "yaml", yml: "yaml",
  html: "xml", xml: "xml", css: "css", sql: "sql", rust: "rust",
  // extend as the vault requires
};

const loaded = new Set<string>();

export async function highlightCodeBlocks(root: HTMLElement) {
  // 1. Discover languages actually present in this note
  const blocks = Array.from(root.querySelectorAll<HTMLElement>("pre code:not(.language-mermaid)"));
  const needed = new Set<string>();
  for (const block of blocks) {
    const lang = detectLanguage(block);          // reads class="language-xxx"
    if (lang) needed.add(lang);
  }

  // 2. Load each needed language exactly once, in parallel
  await Promise.all([...needed].map(loadLanguage));

  // 3. Highlight (registered languages only; unregistered fall back gracefully)
  for (const block of blocks) {
    if (!block.dataset.highlighted) {
      try { hljs.highlightElement(block); } catch { /* unknown language — leave plain */ }
      block.dataset.highlighted = "true";
    }
  }
}

async function loadLanguage(alias: string) {
  const name = ALIASES[alias] ?? alias;
  if (loaded.has(name)) return;
  const importer = modules[`../vendor/highlight-languages/${name}.js`];
  if (!importer) return;                 // not curated — skip, render plain
  const mod = await importer();
  hljs.registerLanguage(name, mod.default);
  loaded.add(name);
}

function detectLanguage(block: HTMLElement): string | null {
  const cls = Array.from(block.classList).find((c) => c.startsWith("language-"));
  return cls ? cls.slice("language-".length).toLowerCase() : null;
}
```

The `vendor/highlight-languages/` directory contains tiny re-export shims so the glob paths are
predictable:

```js
// web/src/vendor/highlight-languages/typescript.js
export { default } from "highlight.js/lib/languages/typescript";
```

> **Why aliases + a curated directory instead of globbing `node_modules` directly?**
> `import.meta.glob` can match `node_modules/highlight.js/lib/languages/*.js`, but depending on
> the package's internal layout and hoisting, the keys are brittle and version-coupled. A small
> `vendor/` directory of explicit re-exports decouples the app from highlight.js's internal path
> layout, makes the curated language set visible in code review, and keeps each language as its
> own Vite chunk. This is the recommended pattern from the Vite maintainers for this exact case.

**Result:** a note with only TypeScript and JSON blocks downloads ~10–20 KB of language
definitions instead of ~1 MB. Languages are cached across navigations (RTK/SPA navigation does
not reload the page), so the second note is instant.

### 5.3 Mermaid lazy import

Replace the static import with a dynamic import inside the existing `useEffect`, gated on the
existing `blocks.length === 0` early-out:

```ts
// Pseudocode — Mermaid on demand
let mermaidInitialized = false;

useEffect(() => {
  const el = contentRef.current;
  if (!el) return;
  const blocks = el.querySelectorAll<HTMLElement>("code.language-mermaid");
  if (blocks.length === 0) return;          // no diagrams → no download

  let cancelled = false;
  (async () => {
    const { default: mermaid } = await import("mermaid");   // ← own chunk
    if (cancelled) return;
    if (!mermaidInitialized) { mermaid.initialize({ ... }); mermaidInitialized = true; }
    blocks.forEach((block) => { /* render → replaceWith(svg) */ });
  })();

  return () => { cancelled = true; };
}, [resolvedHtml]);
```

Because the import is now dynamic, Vite emits `mermaid` (and its d3 subtree) as a separate chunk
that is only fetched when a note actually contains a Mermaid block.

### 5.4 SSR payload trimming

The principle: **only inline data the rendered route actually consumes.** The sidebar (`VaultLayout`)
calls `useGetTreeQuery()`, so the tree is needed wherever the layout renders — which is every
route. However, the **notes list** (`useListNotesQuery()`) is only used by the home route's
`chooseHomeSlug` and by search, not by note pages.

Revised sidecar prefetch contract:

| Route | Prefetch | Inlined in `__PRELOADED_STATE__` |
|---|---|---|
| `/` (home) | `config`, `notes`, `tree`, home `note` | all four |
| `/note/{slug}` | `config`, `tree`, that `note` | `config`, `tree`, `note` — **not** `notes` |
| `/search` | `config`, `notes`, `tree` | `config`, `notes`, `tree` |

The change is localized to `server.mjs`:

```js
// Pseudocode — conditional prefetch
const [config, notesOpt, tree] = await Promise.all([
  fetchAPI("/api/config"),
  route.type === "note" ? null : fetchAPI("/api/notes"),
  fetchAPI("/api/tree"),
]);
const notes = notesOpt ?? null;   // not fetched for /note/*
```

`renderApp` already tolerates `notes: null` (the `if (data.notes)` guard in `preloadCache` skips
it). On the client, if a note page ever needs the notes list later, RTK Query will fetch it lazily.

> **Sidebar note.** `VaultLayout` uses `useGetTreeQuery()` on every route, so the tree stays in
> the preload for now. A future optimization can lazy-load the sidebar and drop the tree from note
> pages entirely, but that changes layout behavior and is out of scope for this ticket.

### 5.5 Guard debug plugins in production

`vite.config.ts` unconditionally includes `jsxLocPlugin()` (adds per-element source-location
attributes), `vitePluginManusRuntime()`, and `vitePluginManusDebugCollector()`. These are
dev/debug aids and should not run in the production build:

```ts
// Pseudocode — conditional plugins
const plugins = [
  react(),
  tailwindcss(),
  process.env.NODE_ENV !== "production" && jsxLocPlugin(),
  process.env.NODE_ENV !== "production" && vitePluginManusRuntime(),
  process.env.NODE_ENV !== "production" && vitePluginManusDebugCollector(),
  vitePluginStorageProxy(),
].filter(Boolean);
```

(`vitePluginStorageProxy` is dev-only too, but it is a server middleware that does not affect the
client bundle; leaving it gated by `command === "serve"` is cleaner — see Phase 5.)

---

## 6. Decision records

### Decision: highlight.js loading strategy

- **Context:** `highlight.js` is the single largest library in the bundle because the bare
  `import hljs from "highlight.js"` entry registers ~190 languages. We need per-language on-demand
  loading, and Vite supports `import.meta.glob` for exactly this (Vite issue #1903).
- **Options considered:**
  1. Switch to `highlight.js/lib/core` and statically import a fixed set of 5–10 languages.
  2. Use `import.meta.glob` to dynamically import language definitions per-note, on demand.
  3. Replace highlight.js with a lighter library (e.g. Prism with selective language registration).
- **Decision:** Option 2 — `highlight.js/lib/core` + `import.meta.glob` over a curated
  `vendor/highlight-languages/` directory of re-export shims.
- **Rationale:** Option 1 still ships every curated language on the note route, even if a given
  note uses only one. Option 2 loads exactly the languages present in each note and caches them
  across SPA navigations. Option 3 is a larger migration with rendering/compat risk and is out of
  scope. The curated `vendor/` directory keeps the language set reviewable and decouples from
  highlight.js internal paths.
- **Consequences:** Adds a small `vendor/highlight-languages/` directory to maintain. Each language
  is a separate HTTP request on first encounter (mitigated by HTTP/2 and RTK's SPA navigation
  caching). Languages not in the curated set render as plain code — the curated set must be kept
  in sync with what the vault actually uses (a one-time audit + occasional additions).
- **Status:** proposed

### Decision: Mermaid loading strategy

- **Context:** Mermaid + d3 is ~300 KB gzipped and is statically imported even though most notes
  have no diagrams and the component already early-outs at runtime.
- **Options considered:**
  1. Keep static import; accept the cost.
  2. Dynamic `import("mermaid")` on first `language-mermaid` block.
  3. Render Mermaid on the server and ship static SVG.
- **Decision:** Option 2.
- **Rationale:** Option 2 is a one-line-ish change that moves Mermaid into its own on-demand
  chunk with no behavior change for notes without diagrams. Option 3 is the ideal end state but is
  a separate, larger project (needs a headless DOM in the Go or Node sidecar) and is out of scope.
- **Consequences:** First mermaid block on a session incurs a one-time ~300 KB download. The
  `mermaidInitialized` module-level flag must remain to avoid re-initializing across notes.
- **Status:** proposed

### Decision: SSR preload contract

- **Context:** `server.mjs` pre-fetches `config` + `notes` + `tree` on every route and inlines
  them, causing ~834 KB of mostly-unused JSON on note pages.
- **Options considered:**
  1. Keep the current "prefetch everything everywhere" contract.
  2. Make prefetch route-conditional: skip the notes list on `/note/*`.
  3. Move all preload data out of HTML into client-side fetches with caching.
- **Decision:** Option 2.
- **Rationale:** Option 2 preserves SSR's main benefit (fast first paint with real data) while
  removing the largest unneeded payload. Option 3 would regress first-paint and SEO; it is a
  valid future direction but out of scope. The tree stays inlined because the sidebar renders on
  every route.
- **Consequences:** Note pages no longer have the notes list pre-seeded; if `chooseHomeSlug` or
  search is reached from a note page, RTK Query fetches `/api/notes` lazily (already cached
  after the first fetch). Must verify no note-route component reads `useListNotesQuery()` at
  render time without a loading state.
- **Status:** proposed

### Decision: Code splitting granularity

- **Context:** The app is a single chunk. We want route-level splitting without a risky rewrite.
- **Options considered:**
  1. Only `manualChunks` for vendors; keep all routes eager.
  2. `React.lazy` on `NotePage` (the only route with heavy deps) + `manualChunks` for vendors.
  3. `React.lazy` on every route.
- **Decision:** Option 2.
- **Rationale:** `NotePage` is the sole consumer of the heavy libs. Lazy-loading just that route
  isolates the cost to `/note/*` with minimal scaffolding. Full route-level splitting (Option 3)
  is marginally better but adds Suspense boundaries everywhere for little gain right now.
- **Consequences:** A `<Suspense>` fallback is needed around the lazy `NotePage`. SSR must
  still render it synchronously — see Phase 1 notes on `React.lazy` + SSR.
- **Status:** proposed

---

## 7. Pseudocode and key flows

### 7.1 Lazy `NotePage` with SSR-safe rendering

`React.lazy` returns a component that suspends. During SSR (`renderToString`), suspense
boundaries are rendered without crashing in React 19, but the simplest safe approach is to keep
the server import eager and only lazy-load on the client. A common pattern:

```tsx
// Pseudocode — App.tsx
import { lazy, Suspense } from "react";

// Client-only lazy; SSR keeps eager import to render synchronously.
const NotePageClient = lazy(() => import("./components/pages/NotePage/NotePage"));

function NoteRoute({ slug }: { slug: string }) {
  // On the server, render the eager component; on the client, the lazy one.
  // A small isServer check (or two entry files) selects which.
  const NotePage = typeof window === "undefined" ? EagerNotePage : NotePageClient;
  return (
    <Suspense fallback={<NoteLoadingFallback />}>
      <NotePage slug={slug} />
    </Suspense>
  );
}
```

> **SSR caveat.** `React.lazy` + `renderToString` does not fetch the chunk on the server, so a
> naive lazy wrapper would render the fallback during SSR and hydrate to the real component on
> the client — causing a hydration mismatch flash. The cleanest fix is to keep the SSR bundle
> importing `NotePage` eagerly (it already does) and apply `React.lazy` only in
> `entry-client.tsx`. This keeps SSR output identical while splitting the client chunk.
> Implement this in Phase 1 and verify `pnpm build:ssr && pnpm smoke:ssr` still passes.

### 7.2 Full client hydration flow (after changes)

```
Browser GET /note/some-slug
  → Go backend proxies to SSR sidecar
  → sidecar fetches /api/config, /api/tree, /api/notes/{slug}  (NOT /api/notes)
  → renderApp() seeds RTK cache with config + tree + note
  → returns HTML with:
       <div id="root">…server-rendered note…</div>
       <script>window.__PRELOADED_STATE__={config, tree, note}</script>  (small!)
  → browser receives ~30-60 KB HTML
  → downloads main.js + vendor-react.js (~150-200 KB gz)
  → hydrates; sidebar reads tree from preloaded state (no fetch)
  → NoteRenderer mounts:
       - scans code blocks → needs typescript, json
       - dynamic-imports hl/typescript.js, hl/json.js (~10-20 KB)
       - highlights
       - if any language-mermaid block → dynamic-imports mermaid chunk (~300 KB)
```

### 7.3 `manualChunks` sketch

```ts
// Pseudocode — vite.config.ts build section
build: {
  rollupOptions: {
    input: { main: path.resolve(WEB_ROOT, "index.html") },
    output: {
      manualChunks(id) {
        if (id.includes("node_modules")) {
          if (id.includes("/react") || id.includes("/scheduler") || id.includes("react-dom"))
            return "vendor-react";
          if (id.includes("react-router")) return "vendor-router";
          if (id.includes("@reduxjs") || id.includes("react-redux")) return "vendor-redux";
          // NOTE: do NOT chunk mermaid/highlight.js here — they are already
          // isolated by dynamic import() and should stay in their own chunks.
        }
      },
    },
  },
},
```

---

## 8. Implementation phases

Each phase is independently shippable and verifiable. Commit after each phase.

### Phase 1: Route-level code splitting (`NotePage`)

**Files:** `web/src/App.tsx`, `web/src/entry-client.tsx`, `web/src/entry-server.tsx`

1. Make `NotePage` a lazy component **on the client only** (see §7.1 SSR caveat).
2. Wrap the lazy route in `<Suspense fallback={<NoteLoadingFallback />}>`.
3. Keep the SSR entry importing `NotePage` eagerly so server output is unchanged.
4. Verify with `pnpm build && du -sh web/dist/assets/*.js` — you should see a new `NotePage-*.js`
   chunk and a smaller `main-*.js`.

**Verify:**
```bash
cd web && pnpm build && ls -la dist/assets/*.js | awk '{print $5, $9}'
pnpm build:ssr && pnpm smoke:ssr
```

### Phase 2: Lazy-load Mermaid

**Files:** `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`

1. Remove `import mermaid from "mermaid"` (line 10).
2. Inside the Mermaid `useEffect` (after the `blocks.length === 0` early-out), add
   `const { default: mermaid } = await import("mermaid");`.
3. Add a `cancelled` flag and clean up in the effect's return.
4. Keep the `mermaidInitialized` module-level flag.

**Verify:**
```bash
cd web && pnpm build
ls dist/assets/ | grep -i mermaid   # expect a mermaid-*.js chunk
# load a note WITHOUT a mermaid block → network tab shows NO mermaid chunk fetched
# load a note WITH a mermaid block → mermaid-*.js fetched once
```

### Phase 3: highlight.js per-language on-demand loading

**Files (new):** `web/src/lib/highlightLanguages.ts`, `web/src/vendor/highlight-languages/*.js`
**Files (edit):** `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`

1. Audit which languages the vault actually uses:
   ```bash
   rg -o 'language-[a-z0-9]+' <vault> | sort | uniq -c | sort -rn
   ```
2. Create `vendor/highlight-languages/<lang>.js` re-export shims for each curated language.
3. Implement `highlightLanguages.ts` as in §5.2.
4. In `NoteRenderer`, remove `import hljs from "highlight.js"` and replace the highlighting
   `useEffect` body with `await highlightCodeBlocks(el)`.
5. Keep the copy-button logic intact (it runs after highlighting).

**Verify:**
```bash
cd web && pnpm build
ls dist/assets/ | grep -iE 'hl-|highlight'   # expect per-language chunks
# load a note with only typescript → only hl-typescript.js fetched, not full hljs
```

### Phase 4: Trim SSR preload on note routes

**Files:** `web/server.mjs`

1. Make the `/api/notes` prefetch conditional on `route.type !== "note"` (see §5.4).
2. Confirm `entry-server.tsx`'s `preloadCache` already skips `null` notes (it does, line 66).
3. Spot-check: a note page's `__PRELOADED_STATE__` should contain `getConfig` + `getTree` + the
   `getNote(slug)` entry, and **not** `listNotes`.

**Verify:**
```bash
curl -s https://parc.yolo.scapegoat.dev/note/<slug> | grep -o 'listNotes(undefined)' | wc -l
# expect 0 on note pages, 1 on home/search
curl -s https://parc.yolo.scapegoat.dev/ -o /tmp/home.html
curl -s https://parc.yolo.scapegoat.dev/note/<slug> -o /tmp/note.html
ls -l /tmp/home.html /tmp/note.html   # note.html should be much smaller
```

### Phase 5: Guard debug plugins in production

**Files:** `web/vite.config.ts`

1. Wrap `jsxLocPlugin()`, `vitePluginManusRuntime()`, and `vitePluginManusDebugCollector()`
   behind `process.env.NODE_ENV !== "production"` (or `mode !== "production"` via the config
   function form).
2. Gate `vitePluginStorageProxy()` behind `command === "serve"` (dev server middleware only).

**Verify:**
```bash
cd web && pnpm build
# build should succeed; production bundle should not contain jsx-loc data attributes
rg -c "data-loc" dist/assets/*.js || echo "no jsx-loc attributes in prod bundle (good)"
```

---

## 9. Test strategy

1. **Build artifacts.** After each phase, `pnpm build` must succeed and `dist/assets/` must show
   the expected new chunk layout (per route / per language / mermaid). Use
   `ls -la dist/assets/*.js | awk '{print $5, $9}'` to record sizes.
2. **SSR smoke.** `pnpm build:ssr && pnpm smoke:ssr` (the existing hydration smoke test) must
   pass after every phase, especially Phase 1 (lazy `NotePage` + SSR interaction).
3. **Hydration correctness.** Load a note page with JS enabled and confirm no React hydration
   warning in the console; the server HTML and client render must match.
4. **Mermaid.** Load a note containing a Mermaid diagram; the diagram must render and the
   `mermaid-*.js` chunk must appear in the network tab exactly once per session.
5. **highlight.js.** Load a note with mixed code blocks (e.g. `typescript`, `json`, `bash`);
   only those three `hl-*.js` chunks should be fetched; an unknown `language-foo` must render as
   plain code without errors.
6. **SSR payload.** `curl -s <note-url> | wc -c` must be substantially smaller than the home
   page, and the inlined state must omit `listNotes` on note pages.
7. **Regression.** Existing unit/vitest tests (`pnpm test`) and Storybook build must pass.

### Verification commands (quick reference)

```bash
# Bundle sizes
cd web && pnpm build && du -sh dist/assets/*.js | sort -h

# Gzipped size of a specific chunk
gzip -c dist/assets/main-*.js | wc -c

# Live payload sizes
curl -s -o /tmp/p.js -w "main.js: %{size_download} bytes\n" https://parc.yolo.scapegoat.dev/assets/main-*.js
curl -s -o /tmp/p.html -w "root.html: %{size_download} bytes\n" https://parc.yolo.scapegoat.dev/

# SSR preload presence per route
curl -s https://parc.yolo.scapegoat.dev/note/<slug> | grep -o 'listNotes(undefined)' | wc -l
```

---

## 10. Risks, alternatives, and open questions

### Risks

- **`React.lazy` + SSR hydration mismatch.** Mitigated by keeping the server import eager and
  lazy-loading only on the client (see §7.1). The `smoke:ssr` test is the guardrail.
- **Curated language set drift.** If a note uses a language not in `vendor/highlight-languages/`,
  it renders as plain code (graceful). Risk is low (audit covers it) but should be monitored.
- **Per-language HTTP requests.** Each language is a separate request on first encounter. Under
  HTTP/2 this is cheap, and RTK/SPA navigation caches them. If it becomes a problem, a Phase 6
  could bundle the top-N languages into one "common languages" chunk.
- **Mermaid first-render flash.** The dynamic import means a Mermaid block shows raw code briefly
  before the chunk loads. Acceptable; can add a minimal placeholder.
- **SSR contract change.** Any component that calls `useListNotesQuery()` on a note route will
  now trigger a client-side fetch instead of reading preloaded state. Must grep for such callers
  before Phase 4.

### Alternatives considered

- **Server-side Mermaid rendering.** Ideal but requires a headless DOM and a larger pipeline.
  Tracked as future work.
- **Replacing Redux Toolkit Query.** Would reduce the vendor chunk but is a large rewrite with no
  bundle benefit on the critical path. Out of scope.
- **Prism instead of highlight.js.** Prism has a smaller core and a well-known selective-import
  story, but migrating changes rendering output and CSS. Out of scope for this ticket.

### Open questions

1. Does any note-route component read `useListNotesQuery()` synchronously at render? (Needs a
   `rg "useListNotesQuery" web/src/components/pages/NotePage` audit before Phase 4.)
2. Should the file tree also be dropped from note-page preload (lazy sidebar)? Out of scope here
   but worth a follow-up ticket.
3. Is the `@builder.io/vite-plugin-jsx-loc` still needed in dev, or can it be removed entirely?

---

## 11. References

### Key files

| File | Role |
|---|---|
| `web/vite.config.ts` | Build config; single entry, no splitting; debug plugins unconditional |
| `web/src/entry-client.tsx` | Client hydration entry; static `AppRoutes` import |
| `web/src/entry-server.tsx` | SSR `renderApp`; `preloadCache` seeds RTK Query cache |
| `web/src/App.tsx` | Route definitions; `HomeRedirect`, `NoteRoute`, `SearchRoute` |
| `web/src/components/pages/NotePage/NotePage.tsx` | The only consumer of `NoteRenderer` |
| `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` | Static `import hljs` (line 9) and `import mermaid` (line 10) |
| `web/src/store/vaultApi.ts` | RTK Query API; endpoints and tags |
| `web/src/store/store.ts` | `makeStore` factory (SSR per-request, client singleton) |
| `web/src/components/pages/VaultLayout/VaultLayout.tsx` | Sidebar; calls `useGetTreeQuery()` (line 68) |
| `web/server.mjs` | SSR sidecar; unconditional prefetch of notes+tree (lines 196–221) |
| `internal/api/api.go` | Go API route registration (lines 60–66) |

### External references

- Vite Dynamic Import / `import.meta.glob` — https://v3.vitejs.dev/guide/features
- Vite issue #1903 (dynamic import of highlight.js languages) — https://github.com/vitejs/vite/issues/1903
- highlight.js usage docs (core vs full) — https://highlightjs.org/usage/
- React `lazy` and Suspense — https://react.dev/reference/react/lazy

### Measured baseline (2026-07-14)

- `main-CZnSi4zO.js`: 2,030,018 B raw / 598,884 B gzipped
- `main-C9LScDXt.css`: 125,549 B raw / 20,225 B gzipped
- `/` HTML: 1,305,031 B (of which ~834 KB is inlined `__PRELOADED_STATE__`)

## 12. Implementation status (2026-07-14)

**Implemented.** The five planned phases landed in commits `88af573`, `f84d634`, `7d1a490`,
`208f105`, and `0b032b5`.

- The browser has a separate `NotePage` chunk; initial home/note hydration resolves that chunk
  before hydration to preserve the SSR component tree, while non-note routes retain lazy loading.
- Mermaid is dynamically imported only after a Mermaid code block is detected.
- highlight.js uses its core engine plus explicit dynamic imports for curated languages. The
  original `import.meta.glob` implementation was replaced with an explicit dynamic-import map and
  SSR no-op alias after it generated invalid Vite SSR output; language chunks remain individual.
- The SSR sidecar no longer serializes `listNotes` on any route. On `/`, it uses the list only on
  the server to choose a note, then serializes a small `window.__HOME_SLUG__` value so the browser
  hydrates the same home note without the index. The tree remains preloaded for the always-visible
  sidebar.
- Development-only JSX-location/Manus/debug/storage plugins run only under `vite serve`.

**Validation passed:** `pnpm check`; 13 SSR unit tests; `pnpm build:all`; and
`pnpm smoke:ssr` (production Vite client + SSR sidecar + Go backend + Chromium, including mobile
sidebar and note navigation), all with zero browser warnings/errors.

**Final local build metrics:** main client chunk 388.78 KB raw / 125.83 KB gzipped; `NotePage`
72.77 KB raw / 25.90 KB gzipped; Mermaid core 145.16 KB gzipped on demand; individual curated
highlight languages 0.32–4.30 KB gzipped. The five-note fixture SSR root response fell to
37,211 B. Deploy to preview and re-measure the live 934-note payload before declaring the live
site result.
