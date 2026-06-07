---
Title: Research logbook
Ticket: RETRO-SEO-009
Status: active
Topics:
    - seo
    - ssr
    - html-layout
    - page-title
    - meta-tags
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/api/api.go
      Note: Referenced — /api/config contract
    - Path: internal/parser/parser.go
      Note: Referenced — Markdown-to-HTML pipeline
    - Path: internal/server/server.go
      Note: Referenced — Go server routing and SSR proxy
    - Path: internal/vault/vault.go
      Note: Referenced — Note struct and title extraction
    - Path: ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/design-doc/01-image-serving-and-page-title-implementation-guide.md
      Note: Referenced — Previous page title design doc (paths outdated)
    - Path: web/server.mjs
      Note: Referenced — SSR HTML assembly and title injection
    - Path: web/src/App.tsx
      Note: Referenced — document.title bug location
    - Path: web/src/entry-client.tsx
      Note: Referenced — Client hydration and SSR clearing
    - Path: web/src/entry-server.tsx
      Note: Referenced — SSR React components
ExternalSources: []
Summary: Logbook tracking every document and resource read during the RETRO-SEO-009 investigation, with evaluations of usefulness, accuracy, and currency.
LastUpdated: 2026-06-07T13:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Research Logbook

This document tracks every resource consulted during the investigation for **RETRO-SEO-009** (page titles, SSR sidebar/SEO, HTML layout). For each resource, it records what was being researched, why the resource was chosen, what was useful, what wasn't, what's outdated, and what needs updating.

---

## 1. `internal/server/server.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How the Go server routes requests — which URLs go to API, assets, SSR proxy, or SPA fallback |
| **What I was looking for** | The route registration order, SSR proxy setup, SPA catch-all placement, where page title is derived |
| **Why I chose it** | It's the single entry point for all HTTP requests; every routing decision flows through this file |
| **How I found it** | `find . -name server.go | grep internal/server` |
| **What was useful** | - `Config` struct shows all runtime options (`SSRURL`, `ReloadToken`, `ServeWeb`) (lines 22-32) <br>- `Run()` shows the exact route registration order: API → health → assets → reload → web (lines 76-86) <br>- `newSSRProxy()` shows the reverse proxy with SPA fallback on 5xx or connection errors (lines 130-155) <br>- SPA handler is registered as catch-all at `PathPrefix("/")` — explains why asset routes must be registered before it |
| **What wasn't useful** | - The CORS config block at the bottom is boilerplate, not relevant to title/SEO <br>- The `healthHandler` and `reloadHandler` are tangential |
| **What is out of date / wrong** | Nothing observed — code matches the running server |
| **What needs updating** | Route registration would benefit from a comment block explaining the ordering rationale |

---

## 2. `internal/server/runtime.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How vault state is loaded, resolved, and reloaded — particularly symlink resolution for git-sync deployments |
| **What I was looking for** | Whether `RuntimeState` uses a single snapshot or fresh reads per request; how symlinks are handled |
| **Why I chose it** | `server.go` references `RuntimeState` for vault access — understanding its semantics is essential for explaining request flow |
| **How I found it** | Same `find` command; it's the companion to `server.go` |
| **What was useful** | - `NewRuntimeState()` calls `filepath.EvalSymlinks()` to resolve symlinks — explains why "current" appears as vault name <br>- `Snapshot()` returns a thread-safe copy via `sync.RWMutex` <br>- `Reload()` atomically swaps vault/search index — explains hot-reload behavior <br>- The design of "configured root" vs "resolved root" is a good pattern for git-sync deployments |
| **What wasn't useful** | The methods are trivially small — no deep implementation worth analyzing |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A — the implementation is clean |

---

## 3. `internal/server/agent_markdown.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How the Go server serves Markdown mirrors (`/AGENTS.md`, `/llms.txt`, `/sitemap.xml`, `/note/{slug}.md`) |
| **What I was looking for** | Whether these endpoints include page titles, canonical URLs, or SEO-relevant headers |
| **Why I chose it** | The ticket mentions agent readability — these are the agent-facing endpoints. Also checking for SEO-adjacent patterns (canonical URLs, etc.) |
| **How I found it** | Searched for `/AGENTS.md` and `/llms.txt` references in the codebase |
| **What was useful** | - `writeMarkdownResponse()` sets `Link: rel="canonical"` header (line ~97) <br>- `renderNoteMarkdown()` includes frontmatter with `title`, `canonical_url` (line ~173) <br>- `renderSitemapXML()` generates proper XML sitemaps with `<lastmod>` dates <br>- `appendNoteLinks()` creates navigable note lists with markdown and HTML links <br>- The Markdown mirror pattern (`.md` suffix → structured markdown) is a good model for SEO |
| **What wasn't useful** | The `renderHomeMarkdown` and `renderLLMSTxt` functions are straightforward index builders — not directly related to the page title issue |
| **What is out of date / wrong** | The `markdownDocVersion = "1"` constant is set but never validated against consumers |
| **What needs updating** | Consider adding OpenGraph meta tags to the Markdown mirror responses (or noting that these are purely for agent consumption) |

---

## 4. `internal/api/api.go`

| Field | Value |
|-------|-------|
| **What I was researching** | The `/api/config` endpoint — specifically what fields it returns and how `pageTitle` flows through the API |
| **What I was looking for** | The `SiteConfig` / `PublicConfig` structs, the `getConfig` handler, and the defaulting logic for `pageTitle` |
| **Why I chose it** | This is the API endpoint that feeds the React app's `SiteConfig`, and the source of the "current" title problem |
| **How I found it** | `rg "/api/config" --files` and `rg "getConfig" --files` |
| **What was useful** | - `PublicConfig` struct with `VaultName` and `PageTitle` fields (line ~37) <br>- `SiteConfig` struct returned to frontend with `vaultName`, `pageTitle`, `notes` count (line ~52) <br>- `NewWithProvider()` has the defaulting logic: `config.PageTitle = config.VaultName` when empty (line ~50) <br>- `getConfig` handler marshals `SiteConfig` with all three fields (line ~58) <br>- The contract is clean: the frontend gets everything it needs from one endpoint |
| **What wasn't useful** | The `NoteListItem`, `SearchResult`, `TagCount` types are tangential to this ticket |
| **What is out of date / wrong** | Nothing — the code matches the live `/api/config` response |
| **What needs updating** | `PageTitle` has no defaulting beyond `vaultName`. A more descriptive default (e.g., derived from frontmatter, or an empty string with a clear error) would help |

---

## 5. `internal/vault/vault.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How notes are loaded, parsed, and how `Title` is extracted; the `rebuildHTML` pipeline |
| **What I was looking for** | Whether `Title` comes from frontmatter or H1, how `rebuildHTML` transforms HTML, how `Backlinks` and `WikiLinks` are built |
| **Why I chose it** | The `Note` struct is the central data structure — every endpoint serves notes, and the title flows from here |
| **How I found it** | `find . -name vault.go | grep internal` |
| **What was useful** | - `loadNote()` extracts `Title` from `parsed.Title` (frontmatter/H1), falling back to filename (line ~173) <br>- `LoadAll()` walks the vault directory, parses each `.md`, calls `rebuildHTML()` (lines ~74-101) <br>- `rebuildHTML()` chains: `ReplaceWikiLinksString` → `ReplaceWikiLinkDisplay` → `RewriteImageSources` (lines ~210-220) <br>- `ResolveAssetURL()` handles relative image path resolution (lines ~225-260) <br>- `buildWikiLinkIndex()` creates suffix-based short-path lookups — explains how `[[Tribal/foo]]` resolves to full paths <br>- `FileTree()` builds the hierarchical file tree used by the sidebar |
| **What wasn't useful** | `ReloadNote()` and `RemoveNote()` are hot-reload operations — tangential to this ticket's scope |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | `pathToSlug()` slugifies paths with hyphens — note titles with special characters may produce unexpected slugs. Edge-case worth documenting |

---

## 6. `internal/parser/parser.go`

| Field | Value |
|-------|-------|
| **What I was researching** | The Markdown-to-HTML pipeline — how goldmark is configured, how wiki links are extracted and replaced, how images are rewritten |
| **What I was looking for** | The regex patterns used for wiki links, the post-processing pipeline, and the image source rewriting function |
| **Why I chose it** | Wiki links are the core Obsidian feature, and the parser's output becomes the HTML that SSR and the client render |
| **How I found it** | `find . -name parser.go | grep internal` |
| **What was useful** | - `Parse()` pre-processes wiki links before goldmark sees them, replacing `[[Target]]` with `<a href="/note/...">` (lines ~42-75) <br>- `extractWikiLinks()` uses regex `(!?)\[\[([^\[\]]+)\]\]` to find all wiki links (line ~116) <br>- `splitFrontmatter()` prevents wiki-link HTML from being injected into YAML frontmatter (lines ~158-176) <br>- `ReplaceWikiLinksString()` uses regex to update `data-target` and `href` attributes (lines ~198-220) <br>- `RewriteImageSources()` handles `<img src="...">` rewriting (lines ~223-240) <br>- `renderCallouts()` transforms Obsidian callouts to styled divs (lines ~260-330) <br>- `extractTitle()` checks frontmatter first, then falls back to H1 (lines ~355-365) <br>- The goldmark config includes: meta, GFM, Table, Strikethrough, TaskList, Footnote (lines ~56-62) |
| **What wasn't useful** | The `normalizeFrontmatter()` and `normalizeYAMLValue()` helpers are deep YAML handling — not relevant to the title issue |
| **What is out of date / wrong** | Nothing — the regex patterns match the Obsidian wiki link syntax correctly |
| **What needs updating** | The `wikiLinkRegex` only matches `[[]` — Obsidian also supports `![[embed]]`. The regex does handle this via the `(!?)` prefix, so it's actually correct. No update needed. |

---

## 7. `internal/web/static.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How the SPA handler serves static web assets and handles client-side routing |
| **What I was looking for** | Whether the SPA handler is used for page rendering or only static assets; how the catch-all works |
| **Why I chose it** | Understanding the SPA handler is essential for understanding the SSR fallback path |
| **How I found it** | `find . -name static.go` |
| **What was useful** | - `NewSPAHandler()` creates a handler that serves static files and falls back to `index.html` for unknown paths (SPA routing) <br>- API paths are excluded (prefixed with `APIPrefix`) (lines ~35-37) <br>- The handler uses `PublicFS` which is an embedded filesystem (lines ~27-32) <br>- `serveIndex()` writes `index.html` on catch-all (lines ~46-51) |
| **What wasn't useful** | `fileExists()` is a trivial helper |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 8. `internal/web/embed.go`

| Field | Value |
|-------|-------|
| **What I was researching** | How the web bundle is embedded into the Go binary |
| **What I was looking for** | The `//go:embed` directive, the directory layout, and how `PublicFS` is constructed |
| **Why I chose it** | Needed to understand how `index.html` ends up in the binary for the SPA handler |
| **How I found it** | `find . -name embed.go` |
| **What was useful** | - `//go:embed embed/public` embeds the built web assets <br>- `PublicFS = mustSub(embeddedFS, "embed/public")` exposes them at filesystem root <br>- The build step `go generate` runs `retro-obsidian-publish build web` which copies `web/dist` to `internal/web/embed/public` |
| **What wasn't useful** | `mustSub()` is a one-line helper |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 9. `web/server.mjs`

| Field | Value |
|-------|-------|
| **What I was researching** | The SSR Express server — specifically title construction, meta tag injection, noscript generation, and the HTML assembly pipeline |
| **What I was looking for** | How the page title is computed from `config` and `note`, how meta tags are injected, how the noscript block is built |
| **Why I chose it** | This is where the SSR HTML is assembled — the direct source of the `<title>` tag and meta tags |
| **How I found it** | `find web/ -name "*.mjs"` |
| **What was useful** | - Title construction: `note.title + " — " + vaultName` for notes, `config.pageTitle || vaultName` for home (line ~125) <br>- `getIndexHtml()` reads `./dist/index.html` as a template (line ~53) <br>- `serializeForInlineScript()` escapes `</`, `>`, `&`, `\u2028`, `\u2029` for safe inline scripts (line ~75) <br>- `parseRoute()` determines whether the request is for home, note, search, or unknown (line ~45) <br>- JSON-LD structured data for `WebPage` and `BreadcrumbList` schemas (lines ~145-162) <br>- Noscript block includes note links, agent guide, sitemap, and llms.txt links (lines ~135-145) <br>- The page title replacement: `/<title>.*?<\/title>/` → `<title>${title}</title>` (line ~172) <br>- `<link rel="alternate" type="text/markdown">` is set for agent discoverability (line ~168) |
| **What wasn't useful** | The Express `app.get("/health")` endpoint is boilerplate <br>- The port/env config at the top is deployment-specific |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | The fallback `getIndexHtml()` (line ~56) has a hardcoded `<title>Retro Obsidian Publish</title>` — if the production `dist/index.html` is missing, this fallback would be used. Should be updated to use a more meaningful default or the SSR-computed title |

---

## 10. `web/src/entry-server.tsx`

| Field | Value |
|-------|-------|
| **What I was researching** | The React SSR entry — how `renderApp()` works, how the SSR components are structured, and how RTK Query cache is preloaded |
| **What I was looking for** | The SSR component implementations (SSRNotePage, SSRHomePage), the route parsing logic, the `preloadCache()` mechanism |
| **Why I chose it** | This is where the server-rendered HTML string comes from. Understanding these components is essential for improving SSR content |
| **How I found it** | `find web/src -name "entry-server*" ` |
| **What was useful** | - `preloadCache()` dispatches `upsertQueryData` actions to populate RTK Query cache before rendering (lines ~40-70) <br>- `SSRNotePage` renders: `<h1>{note.title}</h1>`, tags, note body via `dangerouslySetInnerHTML`, backlinks (lines ~77-115) <br>- `SSRHomePage` renders: title from `config.pageTitle`, note count, note list (lines ~117-135) <br>- `SSRSearchPage` renders a placeholder noting "Search requires JavaScript" (lines ~137-145) <br>- `parseRoute()` in SSR uses the same logic as `server.mjs` — duplicated (potential inconsistency risk) <br>- Components use `React.createElement` (not JSX) to avoid JSX compiler dependencies <br>- `renderApp()` calls `renderToString(<Provider store={store}>{content}</Provider>)` (lines ~155-170) |
| **What wasn't useful** | The `SSRData` and `SSRResult` interfaces are simple type definitions |
| **What is out of date / wrong** | The comment says "Wouter doesn't support server-side rendering (no StaticRouter)" — Wouter v3 does have SSR support via `<StaticRouter>`. The codebase might be using an older pattern. Worth checking if the comment needs updating. <br>- Route parsing is duplicated between `server.mjs` and `entry-server.tsx` — a maintenance risk if routes change |
| **What needs updating** | 1. Verify if Wouter v3 is installed and supports StaticRouter <br>2. Consider extracting `parseRoute` to a shared module to avoid duplication |

---

## 11. `web/src/entry-client.tsx`

| Field | Value |
|-------|-------|
| **What I was researching** | How the client-side React app mounts — specifically the SSR-to-client transition and the `document.title` issue |
| **What I was looking for** | The hydration strategy, whether `hydrateRoot` or `createRoot` is used, and why SSR content is cleared |
| **Why I chose it** | This file explains why the SSR title is overwritten — the client clears SSR content before mounting |
| **How I found it** | `find web/src -name "entry-client*" ` |
| **What was useful** | - Reads `window.__PRELOADED_STATE__` from SSR and passes it to `makeStore()` (lines ~19-21) <br>- Deletes `__PRELOADED_STATE__` to prevent it from lingering (line ~21) <br>- `root.textContent = ""` clears SSR content before mounting (line ~26) <br>- Uses `createRoot` (not `hydrateRoot`) because SSR components don't match the client component tree (lines ~27-33) <br>- The comment explains React error #418 (hydration mismatch) as the reason for not using `hydrateRoot` |
| **What wasn't useful** | The `declare global` for `Window.__PRELOADED_STATE__` is straightforward |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | The `root.textContent = ""` clearing means the SSR-rendered title is lost. Consider keeping the title from SSR by reading `document.title` before clearing, or using a different SSR title injection strategy |

---

## 12. `web/src/App.tsx`

| Field | Value |
|-------|-------|
| **What I was researching** | How the React app handles routing and document title management |
| **What I was looking for** | The `Router` component's `useEffect` that sets `document.title`, and how routing is structured |
| **Why I chose it** | This is where the document title bug lives — the `useEffect` only reads from config, not the current note |
| **How I found it** | `find web/src -name "App.tsx"` |
| **What was useful** | - `useEffect` sets `document.title` from `config.pageTitle` or `config.vaultName` (lines ~15-17) <br>- The dependency array `[config?.pageTitle, config?.vaultName]` means the title only updates when config changes, not when navigating between notes <br>- `HomeRedirect` chooses a home note slug from available notes (lines ~35-60) <br>- `chooseHomeSlug()` has a sophisticated ranking: preferred slugs → index/home/readme → depth-based score <br>- `NoteRoute` extracts the slug from Wouter's wildcard params <br>- `NotFoundPage` renders a basic 404 |
| **What wasn't useful** | The `indexScore()` function and `preferredHomeSlugs` array are home-page routing details, not relevant to titles |
| **What is out of date / wrong** | **The core bug**: `useEffect` never considers the current note. When a user navigates to `/note/foo`, the effect runs with the same config values — the title stays as "current" even though the note title is "Foo Note — current" in the SSR HTML. |
| **What needs updating** | Add note-title awareness: when on `/note/*` routes, include the note's title in `document.title`. This can be done either in the Router or in the `NotePage` component itself. |

---

## 13. `web/src/components/pages/VaultLayout/VaultLayout.tsx`

| Field | Value |
|-------|-------|
| **What I was researching** | How the menubar and sidebar are structured in the client-side app |
| **What I was looking for** | The menu bar layout, sidebar toggle mechanism, responsive behavior (desktop vs mobile), and what content goes where |
| **Why I chose it** | To understand what the SSR output is missing — the sidebar and menu are key navigation elements only rendered by this component |
| **How I found it** | `find web/src -name "VaultLayout.tsx"` |
| **What was useful** | - Menu bar: toggle button, vault name (clickable home on desktop, non-clickable on mobile), search button, right panel toggle, clock (lines ~55-105) <br>- Sidebar: desktop uses `ResizablePanelGroup` with `ResizablePanel` (20% default) and `ResizableHandle`; mobile uses off-canvas drawer (lines ~115-170) <br>- Mobile detection via `window.innerWidth < 768` <br>- `Sidebar` component receives `tree`, `activeSlug`, `onSelectNote`, `onSearch`, `vaultName` props <br>- CSS classes: `retro-menubar`, `retro-menubar-item`, `retro-resize-handle`, `retro-scroll` |
| **What wasn't useful** | The `useCallback` wrappers for `handleNavigate` and `handleSearch` are standard React patterns |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 14. `web/src/components/pages/NotePage/NotePage.tsx`

| Field | Value |
|-------|-------|
| **What I was researching** | How a note page renders — content, backlinks, right panel |
| **What I was looking for** | The data flow (RTK Query → component), the desktop vs mobile layout, and the backlinks section |
| **Why I chose it** | To understand what a note page contains in the client app, so we can decide what to add to SSR |
| **How I found it** | `find web/src -name "NotePage.tsx"` |
| **What was useful** | - Uses `useGetNoteQuery(slug)` to fetch note data (line ~54) <br>- Builds `backlinkEntries` from `note.backlinks` + `allNotes` index (lines ~65-73) <br>- Desktop layout: `ResizablePanelGroup` with note (75%) + backlinks panel (25%) when `rightPanelOpen` (lines ~125-145) <br>- Mobile layout: full-width note with inline backlinks below (lines ~150-155) <br>- `NoteRenderer` component renders the HTML body |
| **What wasn't useful** | The `allSlugs` memo and `handleTagClick` are implementation details |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 15. `web/src/store/vaultApi.ts`

| Field | Value |
|-------|-------|
| **What I was researching** | The RTK Query API slice — endpoints, types, and the mode detection (backend vs static) |
| **What I was looking for** | The `SiteConfig` type definition, the `getConfig` endpoint, and how static/demo mode works |
| **Why I chose it** | The `SiteConfig` type is what the frontend uses for page title — and it only has `vaultName` and `pageTitle` fields |
| **How I found it** | `find web/src/store -name "vaultApi.ts"` |
| **What was useful** | - `SiteConfig` type: `{ vaultName, pageTitle, notes }` (lines ~48-51) <br>- Three-mode support: backend mode (RTK Query with `fetchBaseQuery`), static mode (`VITE_STATIC_VAULT=true`), custom API (`VITE_API_URL`) <br>- `queryFn` pattern for static mode vs `query` for backend mode (clean separation) <br>- Tag-based cache invalidation: `["Note", "Notes", "Tree", "Tags", "Config"]` <br>- `providesTags` on queries enables automatic cache updates |
| **What wasn't useful** | The static vault lazy-load pattern (`getStatic()`) is clever but not relevant to this ticket |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | Consider adding a `currentNoteTitle` field to `SiteConfig` or creating a separate `PageConfig` type that includes the current note's title |

---

## 16. `web/src/store/store.ts`

| Field | Value |
|-------|-------|
| **What I was researching** | The Redux store factory — how SSR and client stores are created differently |
| **What I was looking for** | Whether the store is a singleton or created per-request, and how preloaded state is passed |
| **Why I chose it** | SSR creates a new store per request via `makeStore(preloadedState)` — understanding this is important for understanding cache behavior |
| **How I found it** | `find web/src/store -name "store.ts"` |
| **What was useful** | - `makeStore(preloadedState?)` creates a new store instance each time (line ~7) <br>- Browser uses singleton `store = makeStore()` (line ~15) <br>- SSR passes `preloadedState` from RTK Query cache serialization (line ~9) <br>- Middleware includes RTK Query's `api.middleware` for cache management |
| **What wasn't useful** | The type exports (`AppStore`, `RootState`, `AppDispatch`) are standard |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 17. `web/src/types/index.ts`

| Field | Value |
|-------|-------|
| **What I was researching** | The TypeScript type definitions for the vault system |
| **What I was looking for** | The `SiteConfig`, `Note`, `NoteListItem`, and `FileNode` types |
| **Why I chose it** | These types define the contract between backend and frontend — essential for understanding what data is available |
| **How I found it** | `find web/src -name "types"` |
| **What was useful** | - `SiteConfig`: `{ vaultName, pageTitle, notes }` <br>- `Note`: `{ slug, title, path, frontmatter, tags, excerpt, html, wikiLinks, backlinks, modTime }` <br>- `NoteListItem`: lightweight `{ slug, title, tags, excerpt, modTime, path }` <br>- `WikiLinkRef`: `{ target, alias?, isEmbed?, heading? }` <br>- `FileNode`: `{ name, slug?, path, isFolder, children? }` <br>- `SearchResult`: `{ slug, title, excerpt, tags, score }` <br>- Central re-export from `vaultApi.ts` — single source of truth |
| **What wasn't useful** | `TagCount` type is simple |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | Consider adding a `canonicalURL` field to `Note` — the markdown mirror has it but the JSON note does not |

---

## 18. `web/index.html`

| Field | Value |
|-------|-------|
| **What I was researching** | The initial HTML shell served by Vite — specifically the static `<title>` fallback |
| **What I was looking for** | The hardcoded title, viewport meta, and any SEO-related meta tags |
| **Why I chose it** | This is the HTML that `server.mjs` reads as a template and modifies for SSR |
| **How I found it** | Direct read — it's the Vite entry point |
| **What was useful** | - `<title>Retro Obsidian Publish</title>` is the static fallback (line ~6) <br>- `<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1, viewport-fit=cover">` is good mobile SEO <br>- No other meta tags present — the SSR sidecar adds them at runtime |
| **What wasn't useful** | The body only contains `<div id="root"></div>` and the module script tag |
| **What is out of date / wrong** | The comment "No web fonts — using system Chicago/Charcoal/system-ui for retro macOS 1 look" is accurate but the `index.html` doesn't reference the CSS file with the font definitions (the CSS is injected by Vite in dev mode) |
| **What needs updating** | No update needed — the static title is intentionally generic and gets replaced by SSR |

---

## 19. `web/ssr.Dockerfile`

| Field | Value |
|-------|-------|
| **What I was researching** | How the SSR sidecar container is built |
| **What I was looking for** | The Node.js version, build steps, and the final command that runs the SSR server |
| **Why I chose it** | Understanding the container build helps diagnose production deployment issues |
| **How I found it** | `find web/ -name "*.Dockerfile"` |
| **What was useful** | - Uses `node:22-alpine` (line ~5) <br>- Enables pnpm via `corepack enable` (line ~8) <br>- Copies `package.json`, `pnpm-lock.yaml`, `patches` first for layer caching (lines ~10-11) <br>- `pnpm build:all` builds both SPA and SSR bundles (line ~14) <br>- Environment variables: `SSR_PORT=8089`, `API_BASE=http://localhost:8080`, `BASE_URL=http://localhost:8080` (lines ~16-18) <br>- Final command: `node server.mjs` (line ~22) |
| **What wasn't useful** | The pnpm patch directory is for Wouter patches |
| **What is out of date / wrong** | Nothing |
| **What needs updating** | N/A |

---

## 20. `web/dist/index.html` (build output)

| Field | Value |
|-------|-------|
| **What I was researching** | The actual Vite build output — whether it differs from the source `index.html` |
| **What I was looking for** | How Vite transforms the `<script>` tag, the hashed filenames, and whether the title was modified by the build process |
| **Why I chose it** | `server.mjs` reads this file for the SSR template — if Vite modifies it, the SSR output depends on the build output |
| **How I found it** | Direct read of the built output |
| **What was useful** | - Vite added `<script type="module" crossorigin src="/assets/main-C-QvIkfC.js"></script>` (line ~7) <br>- Vite added `<link rel="stylesheet" crossorigin href="/assets/main-rKPtWN3C.css">` (line ~8) <br>- The title remains `<title>Retro Obsidian Publish</title>` — not modified by Vite <br>- The `head` also contains a huge embedded React runtime (the "manus-runtime" blob) — this is the Vite build's way of bundling React |
| **What wasn't useful** | The embedded React/Scheduler runtime code is minified and not human-readable |
| **What is out of date / wrong** | Nothing — build output is correct |
| **What needs updating** | N/A |

---

## 21. `web/package.json`

| Field | Value |
|-------|-------|
| **What I was researching** | Project dependencies and available scripts |
| **What I was looking for** | The Vite version, React version, and whether Wouter supports SSR |
| **Why I chose it** | Dependency versions determine what SSR patterns are available |
| **How I found it** | Direct read |
| **What was useful** | - Scripts: `build`, `build:ssr`, `build:all`, `ssr`, `dev`, `check`, `format`, `storybook` <br>- React dependencies (implied by the build) <br>- Wouter is listed as a dependency (for routing) <br>- Radix UI components, Redux/RTK Query, Tailwind CSS |
| **What wasn't useful** | The Storybook dependencies are tangential |
| **What is out of date / wrong** | Need to check Wouter version for StaticRouter support |
| **What needs updating** | Verify Wouter version supports SSR |

---

## 22. Live site: `https://parc.yolo.scapegoat.dev/`

| Field | Value |
|-------|-------|
| **What I was researching** | Actual rendered output — page title, meta tags, sidebar presence, noscript content |
| **What I was looking for** | The browser page title, the SSR-rendered `<title>`, `<meta>` tags, and whether the sidebar is visible |
| **Why I chose it** | The source code shows the *intended* behavior; the live site shows the *actual* behavior |
| **How I found it** | Navigated to the URL via Playwright browser |
| **What was useful** | - Browser page title at `/` shows "current" (the vault directory name) <br>- `document.title` in the browser console shows "current" (React overwrote SSR) <br>- Meta tags ARE present: description, og:title, og:description, JSON-LD, breadcrumbs <br>- `<title>` in SSR shows "Research Institute Guidelines — current" (correct note title + vault name) <br>- No sidebar in SSR rendered content <br>- Note title is visible in the rendered `<h1>`: "Research Institute Guidelines" <br>- `/api/config` returns `{"vaultName":"current","pageTitle":"current","notes":706}` |
| **What wasn't useful** | The visual rendering of the SPA (colors, layout) is not relevant to the SEO/title investigation |
| **What is out of date / wrong** | **Confirmed**: Page title is "current" instead of the vault's proper name. **Confirmed**: SSR includes note title in `<title>` but React overwrites it. **Confirmed**: No sidebar in SSR output. |
| **What needs updating** | 1. Deployment: set `PAGE_TITLE` env var <br>2. React: add note-aware document.title <br>3. SSR: add breadcrumb navigation |

---

## 23. Live site: `https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index`

| Field | Value |
|-------|-------|
| **What I was researching** | SSR output for a specific note page |
| **What I was looking for** | The `<title>`, meta tags, noscript content, and whether the sidebar is present |
| **Why I chose it** | To verify the SSR output on a real note page with actual content |
| **How I found it** | Navigated via Playwright, then used `curl` for raw HTML inspection |
| **What was useful** | - `curl` shows: `<title>Research Institute Guidelines — current</title>` — SSR correctly includes note title <br>- `<meta property="og:title" content="Research Institute Guidelines — current" />` — correct <br>- `<meta name="description" content="Research Institute Guidelines Operating guidelines..." />` — correct <br>- JSON-LD with proper `WebPage` schema <br>- `<noscript>` block includes note links and navigation <br>- No `<main>`, `<article>`, or `<nav>` semantic elements in SSR output |
| **What wasn't useful** | The full note HTML body is irrelevant to the title/SEO investigation |
| **What is out of date / wrong** | The noscript block appears to be present in the raw HTML but is removed by the client-side React app's `root.textContent = ""` clearing |
| **What needs updating** | Add semantic HTML elements (`<main>`, `<article>`, `<nav>`) to SSR output |

---

## 24. External reference: docmgr ticket `RETRO-ASSETS-005`

| Field | Value |
|-------|-------|
| **What I was researching** | Previous work on page titles and image serving — to avoid duplicating effort and understand what was already done |
| **What I was looking for** | The design guide from RETRO-ASSETS-005 to see what was planned/implemented for page titles |
| **Why I chose it** | RETRO-ASSETS-005 was the previous ticket that addressed page titles — understanding what was done helps identify what's still broken |
| **How I found it** | `ls ttmp/` showed the existing ticket directory |
| **What was useful** | - The design doc from RETRO-ASSETS-005 provides a detailed architecture walkthrough that overlaps significantly with this investigation <br>- It already identified the `/api/config` / `pageTitle` issue <br>- It already had a proposed frontend title effect (see `// Frontend title effect` pseudocode) <br>- The image URL rewriting and asset serving design from that ticket is relevant to understanding the vault rendering pipeline |
| **What wasn't useful** | The image-serving and asset route sections are not relevant to this ticket's scope |
| **What is out of date / wrong** | The previous ticket references `backend/` paths (`backend/cmd/...`, `backend/internal/...`) — the codebase has been reorganized and these paths no longer exist. The actual paths are at `publish-vault/` root. |
| **What needs updating** | File paths in the RETRO-ASSETS-005 design doc need to be updated to the new `publish-vault/` layout |

---

## Summary

### Most useful resources

1. **`web/server.mjs`** — Direct source of the title/meta injection pipeline
2. **`internal/api/api.go`** — The `/api/config` contract that feeds the frontend
3. **`web/src/entry-server.tsx`** — SSR React components and render pipeline
4. **`web/src/App.tsx`** — Where the `document.title` bug lives
5. **Live site inspection** — Confirmed all issues in practice

### Outdated resources

1. **RETRO-ASSETS-005 design doc** — File paths reference old `backend/` layout
2. **Wouter comment in `entry-server.tsx`** — May be outdated if Wouter v3 supports StaticRouter

### No updates needed

- `internal/server/server.go`
- `internal/server/runtime.go`
- `internal/vault/vault.go`
- `internal/parser/parser.go`
- `web/src/entry-client.tsx`
- `web/src/store/vaultApi.ts`
- `web/ssr.Dockerfile`
- All other codebase files
