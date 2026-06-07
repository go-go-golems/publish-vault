---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/web/src/entry-server.tsx
      Note: Primary SSR reference implementation
    - Path: backend/internal/server/server.go
      Note: Needs SSR proxy addition
    - Path: web/src/store/store.ts
      Note: Needs makeStore() factory for SSR
    - Path: web/vite.config.ts
      Note: Needs ssr.noExternal config
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Research Logbook

## Goal

Catalog every source file, design document, and external resource read during the
RETRO-SSR-009 investigation. Each entry records what we were looking for, what
we found useful, what was missing or outdated, and what would need updating if
this research is revisited later.

## How to use this document

- **Status icons:** ✅ current and useful · ⚠️ partially outdated · ❌ stale
- Each entry is self-contained; read in any order.
- When resuming work on this ticket, check the "Needs updating" field first.
- When adding new research, append a new entry using the same template.

---

## Category A: Publish-Vault Backend (Go)

### A01 — `publish-vault/backend/internal/server/server.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | How the Go server routes requests, where to add an SSR reverse proxy |
| **What I was looking for** | The main `Run()` function, router setup, SPA handler wiring, CORS config |
| **Why I chose it** | This is the server entry point — the SSR proxy must be integrated here |
| **How I found it** | Direct file read of the server package |
| **What I found useful** | Clear `Run()` function that creates `gorilla/mux` router, registers API routes, conditionally serves the SPA. The `cfg.ServeWeb` flag pattern is the exact hook point for the SSR proxy. CORS is applied globally. |
| **What I didn't find useful** | The health handler and reload handler are not relevant to SSR. |
| **What was out of date / wrong** | ✅ Current. No issues. |
| **Needs updating** | Must be modified to add `SSRURL` to `Config`, and replace the `r.PathPrefix("/")` handler with a conditional SSR proxy or SPA fallback. The proxy implementation should follow the `newSSRProxy()` pattern from `glazed/pkg/help/server/serve.go`. |

---

### A02 — `publish-vault/backend/internal/server/runtime.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | How vault data is loaded and swapped atomically |
| **What I was looking for** | The `RuntimeState` struct, `Snapshot()` method, reload mechanics |
| **Why I chose it** | Understanding data freshness — does the SSR sidecar always see current data? |
| **How I found it** | Direct file read, imported by server.go |
| **What I found useful** | `RuntimeState` holds `*vault.Vault` + `*search.Index` behind a `sync.RWMutex`. `Snapshot()` returns both. `Reload()` builds a new vault and atomically swaps it. This confirms the API always serves current data — the SSR sidecar's `/api/*` fetches will be fresh. |
| **What I didn't find useful** | The symlink resolution logic is git-sync-specific and not relevant to SSR. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed for SSR. |

---

### A03 — `publish-vault/backend/internal/api/api.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | The REST API contract — what endpoints exist, what data shapes they return |
| **What I was looking for** | Route definitions, response types, URL parameter extraction |
| **Why I chose it** | The SSR sidecar must fetch from these endpoints to pre-populate data |
| **How I found it** | Direct file read, imported by server.go |
| **What I found useful** | Six endpoints with clear response types: `GET /api/config` → `SiteConfig`, `GET /api/notes` → `NoteListItem[]`, `GET /api/notes/{slug}` → `Note` (full), `GET /api/tree` → `FileNode`, `GET /api/search?q=` → `SearchResult[]`, `GET /api/tags` → `TagCount[]`. The `gorilla/mux` `{slug:.*}` pattern allows multi-segment slugs like `research/kb/tribal/foo`. The `SnapshotProvider` interface decouples the API from the runtime. |
| **What I didn't find useful** | The `staticProvider` and `New()` constructor are for non-server use cases. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed. The SSR sidecar will consume these endpoints as-is. The `upsertQueryData` calls in the SSR entry must match the endpoint names used by RTK Query (`getConfig`, `listNotes`, `getNote`, `getTree`). |

---

### A04 — `publish-vault/backend/internal/vault/vault.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | How notes are loaded, parsed, indexed, and how wiki-links resolve |
| **What I was looking for** | The `Note` struct, slug generation, wiki-link index, backlink computation, HTML rewriting |
| **Why I chose it** | Understanding the data model that the SSR sidecar renders |
| **How I found it** | Direct file read, imported by runtime.go |
| **What I found useful** | The `Note` struct has `Slug`, `Title`, `HTML`, `WikiLinks`, `Backlinks`, `Tags`, `Frontmatter`, `Excerpt`. The `buildWikiLinkIndex()` creates short-slug → full-slug mappings (e.g., `tribal/foo` → `research/kb/tribal/foo`). `rebuildHTML()` rewrites wiki-links and image sources. `FileTree()` returns a hierarchical tree of folders and notes. |
| **What I didn't find useful** | `ReloadNote()` and `RemoveNote()` are file-watcher operations, not relevant to SSR design. |
| **What was out of date / wrong** | ✅ Current. Very well-documented code. |
| **Needs updating** | No changes needed for SSR. |

---

### A05 — `publish-vault/backend/internal/search/search.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | Full-text search implementation |
| **What I was looking for** | How search works, whether it's relevant to SSR pre-fetching |
| **Why I chose it** | To understand if SSR needs to pre-fetch search results |
| **How I found it** | Direct file read, imported by runtime.go |
| **What I found useful** | Bleve-based in-memory index. `Search()` returns ranked results with `Slug`, `Title`, `Excerpt`, `Tags`, `Score`. Uses fuzzy matching. The search endpoint is client-side only — SSR doesn't need to pre-fetch search results (the user hasn't typed a query yet). |
| **What I didn't find useful** | The persistent index variant (`NewPersistent`) is not used in the current deployment. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed for SSR. Search is client-side only. |

---

### A06 — `publish-vault/backend/internal/web/static.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | How the SPA handler works — serving static assets and falling back to index.html |
| **What I was looking for** | The `NewSPAHandler()` function, SPA fallback logic, asset path handling |
| **Why I chose it** | The SSR proxy must fall back to this handler when the sidecar is unavailable |
| **How I found it** | Direct file read, used in server.go |
| **What I found useful** | `newSPAHandler()` takes an `fs.FS` and `SPAOptions`. It checks if a path starts with the API prefix (404), serves static assets if the file exists, otherwise falls back to `index.html`. This is the exact fallback the SSR proxy needs to delegate to. |
| **What I didn't find useful** | The `fileExists()` helper is an internal detail. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed. The SSR proxy will wrap this handler with reverse-proxy logic. |

---

### A07 — `publish-vault/backend/internal/web/embed.go` + `embed_none.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | How web assets are embedded into the Go binary vs loaded from disk |
| **What I was looking for** | The `//go:embed` directive, build tag pattern, `PublicFS` variable |
| **Why I chose it** | Understanding the build pipeline — SSR adds new build outputs (dist/ssr/) that are separate from the embedded assets |
| **How I found it** | Direct file read |
| **What I found useful** | Two build-tag-conditional files: `embed.go` (`//go:build embed`) embeds `embed/public/` via `//go:embed`; `embed_none.go` (`//go:build !embed`) reads from `web/dist/` on disk. The SSR sidecar uses its own `dist/ssr/` directory — it doesn't need to be embedded into the Go binary. |
| **What I didn't find useful** | The `mustSub()` helper and `findRepoRoot()` are build plumbing. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed. SSR assets live in `dist/ssr/` which is only consumed by the Node sidecar, not embedded. |

---

### A08 — `publish-vault/backend/cmd/retro-obsidian-publish/commands/serve/serve.go`

| Field | Detail |
|-------|--------|
| **What I was researching** | The CLI command that starts the server — where to add the `--ssr-url` flag |
| **What I was looking for** | The Glazed-based flag definitions, `Settings` struct, `RunIntoGlazeProcessor()` |
| **Why I chose it** | The `--ssr-url` flag must be defined here and threaded into `server.Config` |
| **How I found it** | Direct file read, the Cobra command entry point |
| **What I found useful** | Uses Glazed's `fields.New()` for flag definitions with help text and defaults. The `Settings` struct has `glazed` tags. `RunIntoGlazeProcessor()` decodes settings and calls `appserver.Run()` with a `Config`. The `--ssr-url` flag follows the exact same pattern as existing flags like `--vault`, `--port`, `--serve-web`. |
| **What I didn't find useful** | The Glazed schema and middleware boilerplate is framework plumbing. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Must add `SSRURL string` to `Settings` struct, add a `fields.New("ssr-url", ...)` flag definition, and pass it through to `appserver.Config{SSRURL: settings.SSRURL}`. |

---

## Category B: Publish-Vault Frontend (React/TypeScript)

### B01 — `publish-vault/web/src/App.tsx`

| Field | Detail |
|-------|--------|
| **What I was researching** | The root React component and client-side routing setup |
| **What I was looking for** | Route definitions, how pages are mounted, the Provider/store wiring, the `chooseHomeSlug()` logic |
| **Why I chose it** | The SSR entry must render the same component tree for each route — I need to know exactly what App renders for `/`, `/note/{slug}`, and `/search` |
| **How I found it** | Direct file read — the app entry point |
| **What I found useful** | Three Wouter routes: `/` → `HomeRedirect`, `/note/*` → `NoteRoute`, `/search` → `SearchRoute`. `HomeRedirect` calls `useListNotesQuery()` and picks a home note via `chooseHomeSlug()`. The whole app is wrapped in `<Provider store={store}>`. `NoteRoute` extracts the wildcard param via `props.params?.["*"]`. This mapping is exactly what the SSR entry must mirror. |
| **What I didn't find useful** | The `indexScore()` helper is home-selection logic that doesn't affect SSR design. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | The `store` import will change from a bare singleton to `makeStore()`-created singleton. The import path stays the same. No functional changes to App.tsx itself. |

---

### B02 — `publish-vault/web/src/store/store.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | The Redux store setup — singleton vs factory pattern |
| **What I was looking for** | How the store is created, what middleware is applied, whether a factory exists |
| **Why I chose it** | SSR needs a fresh store per request; a factory function is required |
| **How I found it** | Direct file read |
| **What I found useful** | Currently exports a single `const store = configureStore({...})` with RTK Query middleware. No factory function exists. The `RootState` and `AppDispatch` types are inferred from the store. The reducer has two keys: `[vaultApi.reducerPath]` and `ui`. |
| **What I didn't find useful** | Nothing — everything in this file is relevant. |
| **What was out of date / wrong** | ✅ Current but needs refactoring. |
| **Needs updating** | **Must be refactored** to export `makeStore(preloadedState?)` as a factory function plus a `store` singleton for browser use. Follow the pattern from `glazed/web/src/store.ts`. The `RootState` and `AppDispatch` types will need adjustment. |

---

### B03 — `publish-vault/web/src/store/vaultApi.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | The RTK Query API slice — endpoint names, data shapes, static mode support |
| **What I was looking for** | Endpoint definitions (`getConfig`, `listNotes`, `getNote`, `getTree`, `search`, `listTags`), how `upsertQueryData` would be used to preload the cache |
| **Why I chose it** | The SSR entry must call `vaultApi.util.upsertQueryData()` with matching endpoint names and query arg signatures |
| **How I found it** | Direct file read |
| **What I found useful** | Six endpoints with clear input/output types. The `IS_STATIC` mode branch is interesting but irrelevant to SSR (SSR always uses backend mode). The endpoint names (`getConfig`, `listNotes`, `getNote`, `getTree`, `search`, `listTags`) are the exact keys used by `upsertQueryData`. `getNote` takes a `string` arg (slug), others take `void`. |
| **What I didn't find useful** | The static vault mode imports — SSR always hits the real API. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed to this file. The SSR entry imports from it directly. The `upsertQueryData` calls must match the endpoint names exactly: `'getConfig'` with `undefined` query arg, `'listNotes'` with `undefined`, `'getTree'` with `undefined`, `'getNote'` with the slug string. |

---

### B04 — `publish-vault/web/src/store/uiSlice.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | UI state management — does the SSR entry need to pre-populate UI state? |
| **What I was looking for** | The `UIState` interface, initial state values, whether any state is window-dependent |
| **Why I chose it** | The SSR entry creates a fresh store; the `ui` slice's initial state must be SSR-safe |
| **How I found it** | Direct file read |
| **What I found useful** | Initial state uses `typeof window !== "undefined"` checks for `sidebarOpen` and `rightPanelOpen` — these are already SSR-safe. The `searchQuery` and `activeNoteSlug` start empty/null. The SSR entry doesn't need to pre-populate any UI state. |
| **What I didn't find useful** | The action creators are only used by client components. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed. The `typeof window` guards are already SSR-compatible. |

---

### B05 — `publish-vault/web/src/main.tsx`

| Field | Detail |
|-------|--------|
| **What I was researching** | The current browser entry point — uses `createRoot`, no hydration |
| **What I was looking for** | How React is mounted, whether there's any pre-rendering support |
| **Why I chose it** | This file will become dev-only; the production build will use the new `entry-client.tsx` instead |
| **How I found it** | Direct file read |
| **What I found useful** | Simple `createRoot(document.getElementById('root')!).render(<App />)` call. Imports `./index.css`. No Redux Provider here (that's in App.tsx). |
| **What I didn't find useful** | Nothing — it's 6 lines. |
| **What was out of date / wrong** | ✅ Current but will be superseded for production builds. |
| **Needs updating** | This file stays for `vite dev`. The production build config will point to `entry-client.tsx` instead. No changes to `main.tsx` itself. |

---

### B06 — `publish-vault/web/src/components/pages/NotePage/NotePage.tsx`

| Field | Detail |
|-------|--------|
| **What I was researching** | The main content page component — what it renders, what data it needs |
| **What I was looking for** | Which RTK Query hooks it calls, how it uses Wouter's `useLocation`, layout structure |
| **Why I chose it** | The SSR entry must render this component (or an SSR-compatible wrapper) for `/note/{slug}` routes |
| **How I found it** | Direct file read |
| **What I found useful** | Calls `useGetNoteQuery(slug)` and `useListNotesQuery()`. Uses `useLocation()` from Wouter for navigation (backlinks, tag clicks). Uses `useAppSelector` for `rightPanelOpen`. Has desktop (resizable panels) and mobile (inline backlinks) layouts. The `useEffect` dispatches `setActiveNote`. |
| **What I didn't find useful** | The mobile/desktop layout branching is complex but doesn't affect SSR design. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | The SSR wrapper (`NotePageSSR`) needs to skip Wouter's `useLocation` — either mock it or pass navigation callbacks as props. The `useEffect` that sets the active note will run on hydration but not during `renderToString`. |

---

### B07 — `publish-vault/web/src/components/pages/SearchPage/SearchPage.tsx`

| Field | Detail |
|-------|--------|
| **What I was looking for** | Whether search page needs SSR pre-rendering |
| **Why I chose it** | Search requires a user query — the SSR page would just show an empty search box |
| **How I found it** | Direct file read |
| **What I found useful** | The search page reads `searchQuery` from Redux and calls `useSearchQuery(query)` with a 2-character minimum. On initial load (empty query), it shows a placeholder. SSR for this page would render the layout + empty state, which is still better than an empty `<div id="root">`. |
| **What I didn't find useful** | The search result rendering is client-side only. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Minimal — the SSR entry renders the search page shell (menubar + sidebar + empty search state). No data pre-fetching needed beyond `config` and `tree`. |

---

### B08 — `publish-vault/web/vite.config.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | The Vite build configuration — what needs to change for SSR output |
| **What I was looking for** | Build output settings, plugins, proxy config, SSR-related options |
| **Why I chose it** | Must add SSR build target and `ssr.noExternal` configuration |
| **How I found it** | Direct file read |
| **What I found useful** | Uses `@vitejs/plugin-react`, `@tailwindcss/vite`, custom Manus plugins. Build output goes to `web/dist/`. Has dev proxy for `/api` and `/vault-assets`. Currently has no SSR configuration at all. |
| **What I didn't find useful** | The Manus debug collector and storage proxy plugins are dev-only and irrelevant to SSR. |
| **What was out of date / wrong** | ✅ Current but missing SSR config. |
| **Needs updating** | **Must add:** `ssr: { noExternal: ['react', 'react-dom', '@reduxjs/toolkit', 'react-redux', 'use-sync-external-store'] }` to ensure the SSR bundle inlines these packages instead of leaving them as external imports. Also needs a `build:ssr` script in package.json. |

---

### B09 — `publish-vault/web/package.json`

| Field | Detail |
|-------|--------|
| **What I was researching** | Dependencies and scripts — what's needed for the SSR sidecar |
| **What I was looking for** | Existing React/Redux versions, whether Express is already a dependency, build scripts |
| **Why I chose it** | Need to add Express and SSR build scripts |
| **How I found it** | Direct file read |
| **What I found useful** | React 19, Redux Toolkit 2.11, Wouter 3.3. Express 4.21 is already a dependency. Uses pnpm. Has `dev`, `build`, `preview`, `check`, `storybook` scripts. |
| **What I didn't find useful** | The large Radix UI dependency list is irrelevant to SSR. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | **Must add:** `"build:ssr": "vite build --ssr src/entry-server.tsx --outDir dist/ssr"`, `"build:all": "pnpm build && pnpm build:ssr"`, `"ssr": "node server.mjs"` scripts. Express is already present so no new dependency needed. |

---

### B10 — `publish-vault/web/index.html`

| Field | Detail |
|-------|--------|
| **What I was researching** | The HTML shell that the SSR server uses as a template |
| **What I was looking for** | The `<div id="root">` element, script tags, meta tags |
| **Why I chose it** | The SSR sidecar reads `dist/index.html` as a template and injects SSR content into it |
| **How I found it** | Direct file read |
| **What I found useful** | Minimal shell: `<!doctype html>`, `<div id="root"></div>`, single `<script type="module" src="/src/main.tsx">`. The SSR server replaces the empty `<div id="root">` with server-rendered HTML and adds preloaded state + meta tags before `</head>`. |
| **What I didn't find useful** | Nothing — it's a minimal shell. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | The production build (`vite build`) will replace the `<script>` with hashed asset references. The SSR server reads from `dist/index.html` (the built version), not this source file. No changes needed to the source `index.html`. |

---

### B11 — `publish-vault/web/src/types/index.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | Central type definitions for the vault data model |
| **What I was looking for** | `Note`, `NoteListItem`, `FileNode`, `SearchResult`, `WikiLinkRef` types |
| **Why I chose it** | The SSR entry imports these types for the `SSRData` interface |
| **How I found it** | Direct file read |
| **What I found useful** | Clean TypeScript interfaces: `Note` has `html`, `wikiLinks`, `backlinks`; `NoteListItem` has `slug`, `title`, `tags`, `excerpt`, `modTime`, `path`; `FileNode` is recursive with `children`. These are the exact shapes returned by the Go API and consumed by RTK Query. |
| **What I didn't find useful** | Nothing — all types are relevant. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | No changes needed. The SSR entry's `SSRData` interface references these types directly. |

---

## Category C: Glazed SSR Reference Implementation

### C01 — `glazed/web/src/entry-server.tsx`

| Field | Detail |
|-------|--------|
| **What I was researching** | The SSR entry point in the working Glazed implementation |
| **What I was looking for** | How `renderApp()` works, how RTK Query cache is preloaded, `StaticRouter` usage, the `SSRData`/`SSRResult` interfaces |
| **Why I chose it** | This is the closest reference — publish-vault's entry-server will follow the same pattern with adaptations for Wouter and different API shapes |
| **How I found it** | `grep -rl ssr web/` in glazed directory |
| **What I found useful** | Clean separation: `SSRData` holds pre-fetched API responses, `renderApp(url, data)` creates a fresh store via `makeStore()`, upserts data into RTK Query cache, then calls `renderToString()` with `<Provider>` + `<StaticRouter>`. Returns `{ html, preloadedState }`. The `preloadRTKQueryCache()` function shows exactly how to call `store.dispatch(helpApi.util.upsertQueryData(...))` for each endpoint. |
| **What I didn't find useful** | The `parseDocsRoute()` function is Glazed-specific (packages/versions/sections URL scheme). Publish-vault has simpler `/note/{slug}` routing. |
| **What was out of date / wrong** | ✅ Current and production-tested. |
| **Needs updating** | Use as a reference template. Key adaptation: replace `StaticRouter` with direct component rendering (Wouter has no `StaticRouter`). Replace `helpApi` with `vaultApi`. Replace Glazed's `SSRData` fields (`packages`, `sections`, `section`) with publish-vault's (`config`, `notes`, `tree`, `note`). |

---

### C02 — `glazed/web/src/entry-client.tsx`

| Field | Detail |
|-------|--------|
| **What I was researching** | The client hydration entry point |
| **What I was looking for** | How `window.__PRELOADED_STATE__` is read and used to create the store, the `hydrateRoot()` call |
| **Why I chose it** | Publish-vault's entry-client.tsx will be nearly identical |
| **How I found it** | Same directory as entry-server.tsx |
| **What I found useful** | Pattern: `const preloadedState = window.__PRELOADED_STATE__; delete window.__PRELOADED_STATE__; const store = makeStore(preloadedState); hydrateRoot(rootEl, <Provider store={store}><BrowserRouter>...</BrowserRouter></Provider>)`. The `declare global { interface Window { __PRELOADED_STATE__ } }` type declaration. |
| **What I didn't find useful** | The `BrowserRouter` with `future` flags is React Router specific. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Copy this pattern almost verbatim. Replace `BrowserRouter` with the App component (which contains its own Wouter routing). Replace `makeStore` import path. |

---

### C03 — `glazed/web/server.mjs`

| Field | Detail |
|-------|--------|
| **What I was researching** | The Node.js SSR HTTP server implementation |
| **What I was looking for** | Express setup, data pre-fetching logic, HTML assembly, meta tag injection, noscript fallback, JSON-LD structured data |
| **Why I chose it** | This is the most complex piece — the publish-vault server.mjs will follow the same structure with different URL parsing and data fetching |
| **How I found it** | Direct file read |
| **What I found useful** | Comprehensive implementation: (1) Dynamic `await import()` of the SSR bundle after setting up `window` mock. (2) `fetchAPI()` helper with error handling. (3) `parseDocUrl()` for URL → route mapping. (4) `getIndexHtml()` reads the built `dist/index.html` as template. (5) HTML assembly: injects SSR content into `<div id="root">`, adds `__PRELOADED_STATE__`, meta tags, JSON-LD, `<noscript>` fallback, and visually-hidden headings for a14y. (6) `serializeForInlineScript()` prevents XSS in the inline script. (7) `Link` headers for canonical and alternate URLs. |
| **What I didn't find useful** | The `window.__GLAZE_SITE_CONFIG__` mock is Glazed-specific (publish-vault's API module doesn't read window at import time). The `normalizeStaticAssetPath` logic in the Go server handles this instead. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Adapt the URL parsing from `/{package}/{version}/sections/{slug}` to `/note/{slug}`, `/`, `/search`. Adapt data fetching from `packages/sections/section` to `config/notes/tree/note`. The `window` mock setup may not be needed if publish-vault's `vaultApi.ts` doesn't read window at import time (it uses `import.meta.env` which is resolved at build time). |

---

### C04 — `glazed/web/src/store.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | The Redux store factory pattern used in Glazed SSR |
| **What I was looking for** | How `makeStore()` is defined, how the singleton `store` coexists with the factory |
| **Why I chose it** | This is the exact pattern publish-vault's `store.ts` must adopt |
| **How I found it** | Direct file read |
| **What I found useful** | `export function makeStore(preloadedState?: unknown)` returns `configureStore({...})`. Below it, `export const store = makeStore()` creates the browser singleton. Types are inferred from the factory return type: `type AppStore = ReturnType<typeof makeStore>`. Clean, minimal, well-tested pattern. |
| **What I didn't find useful** | Nothing — this file is the template. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Use as-is. Copy the pattern to publish-vault's `store.ts`. |

---

### C05 — `glazed/pkg/help/server/serve.go` (SSR proxy section)

| Field | Detail |
|-------|--------|
| **What I was researching** | The Go-side SSR reverse proxy implementation |
| **What I was looking for** | `newSSRProxy()`, `WithSSRURL()`, the `--ssr-url` flag, fallback behavior, static asset handling |
| **Why I chose it** | Publish-vault's Go server needs the same proxy pattern |
| **How I found it** | `grep -rn ssr pkg/help/server/` in glazed directory |
| **What I found useful** | The `ServeOption` pattern for functional options. `newSSRProxy()` creates a reverse proxy that: (1) builds the proxy URL from `ssrEndpoint.ResolveReference()`, (2) forwards useful headers, (3) streams the response body, (4) falls back to the SPA handler on connection errors or 5xx responses. The `normalizeStaticAssetPath()` function correctly routes `/assets/*`, `/fonts/*`, and root static files directly to the SPA handler (not through the proxy). |
| **What I didn't find useful** | The well-known handler, markdown content negotiation, and markdown suffix URL handling are Glazed-specific features. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Copy the `newSSRProxy()` function and `serveHandlerConfig` pattern. Add `normalizeStaticAssetPath()` equivalent for `/assets/*` and `/vault-assets/*`. The `--ssr-url` flag follows the same Glazed field definition pattern. |

---

### C06 — `glazed/web/vite.config.ts`

| Field | Detail |
|-------|--------|
| **What I was researching** | The Vite SSR build configuration |
| **What I was looking for** | The `ssr.noExternal` setting, build output configuration |
| **Why I chose it** | Publish-vault's vite.config.ts needs the same SSR settings |
| **How I found it** | Direct file read |
| **What I found useful** | `ssr: { noExternal: ['react', 'react-dom', 'react-router-dom', '@reduxjs/toolkit', 'react-redux', 'use-sync-external-store'] }`. This tells Vite to inline these packages into the SSR bundle instead of leaving them as external imports. Without this, the SSR bundle would try to `import('react')` at runtime and fail. |
| **What I didn't find useful** | The `base: '/'` and dev proxy settings are specific to Glazed's domain setup. |
| **What was out of date / wrong** | ✅ Current. |
| **Needs updating** | Copy the `ssr.noExternal` list, replacing `react-router-dom` with `wouter` (since publish-vault uses Wouter). |

---

### C07 — `glazed/web/package.json`

| Field | Detail |
|-------|--------|
| **What I was researching** | The SSR-related scripts and Express version in Glazed |
| **What I was looking for** | `build:ssr`, `build:all`, `ssr` scripts, Express dependency |
| **Why I chose it** | Template for publish-vault's package.json additions |
| **How I found it** | Direct file read |
| **What I found useful** | Scripts: `"build:ssr": "vite build --ssr src/entry-server.tsx --outDir dist/ssr"`, `"build:all": "pnpm build && pnpm build:ssr"`, `"ssr": "node server.mjs"`. Uses Express 5 (`"express": "^5.2.1"`). Publish-vault already has Express 4 — either version works for our simple server. |
| **What I didn't find useful** | The other scripts are Glazed-specific. |
| **What was out of date / wrong** | ⚠️ Express 5 vs Express 4 difference. Glazed uses Express 5 (`{*path}` wildcard syntax). Publish-vault has Express 4 — the wildcard syntax is different (`*` vs `{*path}`). |
| **Needs updating** | Copy the script names. Note: use Express 4 wildcard syntax (`app.get('*', ...)`) instead of Express 5 (`app.get('{*path}', ...)`). Alternatively, upgrade to Express 5. |

---

## Category D: Existing Design Documents

### D01 — `glazed/ttmp/2026/05/25/DOCSCTL-SSR--server-side-rendering-via-node-js-sidecar-for-docs-browser/design-doc/01-ssr-sidecar-analysis-and-implementation-guide.md`

| Field | Detail |
|-------|--------|
| **What I was researching** | The original Glazed SSR design doc — architecture, decisions, implementation plan |
| **What I was looking for** | High-level architecture diagram, key design decisions, file layout, implementation task list |
| **Why I chose it** | This is the design doc that guided the Glazed SSR implementation — understanding it helps avoid re-litigating decisions that were already made |
| **How I found it** | `find glazed -path '*ssr*'` to locate SSR-related ticket directories |
| **What I found useful** | The architecture diagram (Go server ↔ Node sidecar in same pod). The request flow (10-step sequence). Key design decisions: (1) sidecar not embedded, (2) same App component for SSR/client, (3) `__PRELOADED_STATE__` for hydration, (4) `StaticRouter` on server, (5) `hydrateRoot` on client. The file layout and task list provided a clear implementation checklist. |
| **What I didn't find useful** | The Glazed-specific URL scheme and package/version/section routing details. |
| **What was out of date / wrong** | ✅ Current — the design doc matches the implemented code. |
| **Needs updating** | Use as a structural template for the publish-vault design doc. The section ordering (exec summary → architecture → request flow → decisions → file layout → tasks) worked well and was reused.

---

## Summary: Resource Status Overview

| ID | Resource | Status | Must Change for SSR? |
|----|----------|--------|---------------------|
| A01 | server.go | ✅ Current | Yes — add SSRURL + proxy |
| A02 | runtime.go | ✅ Current | No |
| A03 | api.go | ✅ Current | No |
| A04 | vault.go | ✅ Current | No |
| A05 | search.go | ✅ Current | No |
| A06 | static.go | ✅ Current | No (fallback target) |
| A07 | embed.go/embed_none.go | ✅ Current | No |
| A08 | serve.go | ✅ Current | Yes — add --ssr-url flag |
| B01 | App.tsx | ✅ Current | No (store import changes) |
| B02 | store.ts | ✅ Current | **Yes — add makeStore()** |
| B03 | vaultApi.ts | ✅ Current | No |
| B04 | uiSlice.ts | ✅ Current | No |
| B05 | main.tsx | ✅ Current | No (dev-only) |
| B06 | NotePage.tsx | ✅ Current | SSR wrapper needed |
| B07 | SearchPage.tsx | ✅ Current | Minimal SSR wrapper |
| B08 | vite.config.ts | ✅ Current | **Yes — add ssr.noExternal** |
| B09 | package.json | ✅ Current | **Yes — add SSR scripts** |
| B10 | index.html | ✅ Current | No |
| B11 | types/index.ts | ✅ Current | No |
| C01 | glazed entry-server.tsx | ✅ Current | Reference only |
| C02 | glazed entry-client.tsx | ✅ Current | Reference only |
| C03 | glazed server.mjs | ✅ Current | Reference only |
| C04 | glazed store.ts | ✅ Current | Reference only |
| C05 | glazed serve.go (proxy) | ✅ Current | Reference only |
| C06 | glazed vite.config.ts | ✅ Current | Reference only |
| C07 | glazed package.json | ⚠️ Express 5 vs 4 | Note wildcard syntax |
| D01 | glazed SSR design doc | ✅ Current | Structural template |

**Files that must be created:** `entry-server.tsx`, `entry-client.tsx`, `server.mjs`, `ssr.Dockerfile`

