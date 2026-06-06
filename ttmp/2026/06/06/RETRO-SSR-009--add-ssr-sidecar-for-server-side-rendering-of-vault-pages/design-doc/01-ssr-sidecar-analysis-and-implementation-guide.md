---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/help/server/serve.go
      Note: Reference Go SSR proxy implementation from Glazed
    - Path: ../../../../../../../glazed/web/server.mjs
      Note: Reference Node SSR sidecar from Glazed
    - Path: ../../../../../../../glazed/web/src/entry-server.tsx
      Note: Reference SSR entry point from Glazed
    - Path: backend/internal/api/api.go
      Note: REST API endpoints that sidecar will fetch from
    - Path: backend/internal/server/server.go
      Note: Go server that needs SSR proxy addition
    - Path: web/src/App.tsx
      Note: Client-side Wouter routing that SSR entry must mirror
    - Path: web/src/store/store.ts
      Note: Redux store singleton that needs makeStore() factory
    - Path: web/src/store/vaultApi.ts
      Note: RTK Query API with upsertQueryData for SSR cache preloading
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# SSR Sidecar: Analysis and Implementation Guide

## 1. Executive Summary

Add a Node.js SSR sidecar to the Retro Obsidian Publish deployment so that every
vault page is returned as fully-rendered HTML — headings, note body, backlinks,
metadata — before JavaScript runs in the browser. Today the Go server returns an
empty `<div id="root"></div>` shell for all page requests. Crawlers, search
engines, and AI agents that don't execute JavaScript see a blank page.

The approach mirrors the SSR system already running in production for the Glazed
docs browser (`glazed/web/`), adapted to the publish-vault routing scheme,
Redux store shape, and component tree. The Go server gains an `--ssr-url` flag
that, when set, reverse-proxies page requests to the sidecar. If the sidecar is
unavailable, the server falls back to the existing SPA handler.

**Goal:** Every vault URL (`/`, `/note/some-slug`, `/search`) returns complete
HTML with real text content, proper headings, meta tags, and JSON-LD structured
data. The browser hydrates the server-rendered DOM with zero flash of content.

## 2. Problem Statement and Scope

### 2.1 The problem

Retro Obsidian Publish is a single-page application (SPA) built with React,
Redux, and Wouter. The Go backend serves two things:

1. **API routes** (`/api/*`) — JSON data about vault notes, trees, search, tags.
2. **SPA fallback** — for every non-API route, the Go server returns
   `index.html`, which contains `<div id="root"></div>` and a `<script>` tag
   that loads the React bundle.

This means any agent that doesn't execute JavaScript (search-engine crawlers,
`curl`, AI agents, link previewers) receives an empty page. The actual content
— note titles, rendered markdown, backlinks — only appears after React
downloads, boots, fetches data from `/api/*`, and renders components.

### 2.2 Scope

- Add an SSR sidecar (Node.js + Express) that renders React to HTML on the
  server.
- Modify the Go server to support reverse-proxying page requests to the sidecar.
- Modify the web build to produce an SSR bundle alongside the client bundle.
- Switch the client from `createRoot` to `hydrateRoot` for hydration.
- Ensure local development works without the sidecar (fallback to SPA).

**Out of scope:** Static site generation (pre-rendering all notes at build
time), edge SSR (Cloudflare Workers, etc.), replacing Wouter with React Router.

## 3. Current-State Architecture

### 3.1 Go backend

The Go server lives in `publish-vault/backend/` and is organized into packages:

- **`internal/vault/`** (`vault.go`) — Scans the filesystem for `.md` files,
  parses frontmatter, builds wiki-link indexes, computes backlinks, rewrites
  HTML with resolved links. The `Vault` struct holds all notes keyed by slug.
- **`internal/parser/`** (`parser.go`) — Markdown → HTML conversion, frontmatter
  extraction, wiki-link parsing.
- **`internal/search/`** (`search.go`) — Bleve-based full-text search index over
  note bodies and titles.
- **`internal/api/`** (`api.go`) — HTTP REST API registered on `gorilla/mux`.
  Endpoints: `/api/config`, `/api/notes`, `/api/notes/{slug}`, `/api/tree`,
  `/api/search`, `/api/tags`.
- **`internal/server/`** (`server.go`, `runtime.go`) — Main server entry point.
  `Run()` creates a `RuntimeState` (which holds `*vault.Vault` and
  `*search.Index`), registers API routes, optionally serves the SPA, and
  starts an `http.Server` on the configured port (default `:8080`).
- **`internal/web/`** (`embed.go`, `static.go`, `embed_none.go`) — SPA handler.
  `NewSPAHandler()` serves static assets from `web/dist/` and falls back to
  `index.html` for client-side routes. Build tags control whether assets are
  embedded (`//go:build embed`) or read from disk (`//go:build !embed`).
- **`cmd/retro-obsidian-publish/commands/serve/`** (`serve.go`) — Glazed-based
  Cobra command that parses flags (`--vault`, `--port`, `--serve-web`, etc.)
  and calls `server.Run()`.

The server uses `gorilla/mux` (not `net/http.ServeMux`) for routing.
CORS is applied globally via `rs/cors`.

### 3.2 Web frontend

The frontend lives in `publish-vault/web/` and is a Vite + React + TypeScript
SPA.

**Routing:** Wouter (not React Router). Three routes:

| Path | Component | Description |
|------|-----------|-------------|
| `/` | `HomeRedirect` | Picks the first available note (index/home/readme) |
| `/note/*` | `NotePage` | Displays a single note by slug |
| `/search` | `SearchPage` | Full-text search UI |

**State management:**

- **`store/store.ts`** — `configureStore()` with RTK Query (`vaultApi`) and a
  `uiSlice`. A single exported `store` singleton (not a factory function).
- **`store/vaultApi.ts`** — RTK Query API slice with endpoints: `getConfig`,
  `listNotes`, `getNote`, `getTree`, `search`, `listTags`. Supports two modes:
  - *Backend mode* (default): fetches from `/api/*` on the same origin.
  - *Static mode* (`VITE_STATIC_VAULT=true`): serves data from an in-browser
    static vault module.
- **`store/uiSlice.ts`** — UI state: `sidebarOpen`, `rightPanelOpen`,
  `searchQuery`, `activeNoteSlug`.

**Key components:**

- **`App.tsx`** — Root component. Wraps `<Router>` in `<Provider store={store}>`.
  `Router()` uses Wouter `<Switch>/<Route>` to render page components inside
  `<VaultLayout>`.
- **`VaultLayout`** — Main layout with menubar, resizable sidebar, and content
  pane. Fetches tree data via `useGetTreeQuery()`.
- **`NotePage`** — Fetches a note by slug via `useGetNoteQuery(slug)`, renders
  markdown with `NoteRenderer`, shows backlinks in a right panel.
- **`SearchPage`** — Search input + results list using `useSearchQuery()`.

**Entry point:** `web/src/main.tsx` uses `createRoot().render()` — no hydration.

### 3.3 Build pipeline

- `vite build` produces `web/dist/` with `index.html` + hashed JS/CSS assets.
- `go generate ./backend/internal/web/` runs `retro-obsidian-publish build web`
  which copies `web/dist/` into `backend/internal/web/embed/public/`.
- The `//go:embed embed/public` directive bundles everything into the Go binary.
- With `//go:build !embed`, assets are read from disk (dev mode).

### 3.4 Data flow (current)

```
Browser requests /note/my-note
  → Go server: no /api prefix, not a static asset
  → SPA handler serves index.html (empty <div id="root">)
  → Browser loads JS bundle
  → React boots, creates Redux store
  → Wouter matches /note/my-note → NotePage component
  → NotePage calls useGetNoteQuery("my-note")
  → RTK Query fetches /api/notes/my-note from Go server
  → Go server returns JSON { slug, title, html, backlinks, ... }
  → React renders the note
```

The problem is clear: steps 1–5 produce zero visible content. Only after step 9
does the user (or crawler) see anything.

## 4. Reference Implementation: Glazed SSR

The Glazed docs browser already has a production SSR sidecar. This section
describes how it works so you can understand the pattern before we adapt it.

### 4.1 Architecture

```
Production (k3s pod):
┌──────────────────────────────────────────────┐
│  Pod                                          │
│  ┌──────────────┐     ┌────────────────────┐ │
│  │  Go server    │────▶│  Node SSR sidecar   │ │
│  │  (:8088)     │     │  (:8089)            │ │
│  │              │◀────│                     │ │
│  │  /api/* → own│     │  Calls /api/* for   │ │
│  │  /* → proxy  │     │  data, renders      │ │
│  │    to :8089  │     │  React → full HTML   │ │
│  └──────────────┘     └────────────────────┘ │
└──────────────────────────────────────────────┘
```

The Go server and Node sidecar run in the same Kubernetes pod, communicating
over localhost. The Go server is the entry point; it decides whether to handle
a request itself (API, static assets) or forward it to the sidecar (pages).

### 4.2 Key files in glazed/web/

| File | Purpose |
|------|----------|
| `src/entry-server.tsx` | SSR entry point. Exports `renderApp(url, data)` which creates a fresh Redux store, preloads RTK Query cache with server-fetched data, and calls `renderToString()`. Returns `{ html, preloadedState }`. |
| `src/entry-client.tsx` | Client entry with `hydrateRoot()`. Reads `window.__PRELOADED_STATE__` and uses it to create the Redux store, so React hydrates (reuses) server DOM instead of replacing it. |
| `src/main.tsx` | Dev-only entry with `createRoot()` (no hydration). Used by `vite dev`. |
| `server.mjs` | Express HTTP server. Parses URLs, fetches data from Go API, calls `renderApp()`, injects SSR HTML + preloaded state into the `index.html` shell, adds JSON-LD and meta tags. |
| `vite.config.ts` | Includes `ssr.noExternal` config so the SSR bundle inlines React, RTK, etc. instead of leaving them as external imports. |

### 4.3 SSR request flow (Glazed)

```
1. Browser requests /glazed/v1.3.4/sections/foo
2. Go server: not /api, not a static asset → reverse-proxy to Node :8089
3. Node server.mjs receives the request
4. Parse URL → packageName="glazed", version="v1.3.4", slug="foo"
5. Fetch data from Go API:
   - GET /api/packages
   - GET /api/sections?package=glazed&version=v1.3.4
   - GET /api/sections/foo?package=glazed&version=v1.3.4
6. Call renderApp(url, { packages, sections, section })
   a. Create fresh Redux store (makeStore())
   b. Upsert fetched data into RTK Query cache
   c. renderToString(<Provider store={store}><StaticRouter location={url}><AppRoutes /></StaticRouter></Provider>)
   d. Return { html, preloadedState }
7. Inject html into <div id="root"> in the index.html shell
8. Inject preloadedState as window.__PRELOADED_STATE__ = {...}
9. Add <meta> tags, JSON-LD, <title>
10. Return complete HTML to Go server → browser
11. Browser loads JS, entry-client.tsx reads __PRELOADED_STATE__,
    creates store, hydrates existing DOM
```

### 4.4 Critical design decisions in Glazed SSR

1. **Store factory, not singleton.** The SSR server creates a *new* Redux store
   per request via `makeStore()`. The browser module exports a singleton `store`.
   This prevents request-level data leaking between concurrent renders.

2. **`window` mock.** The SSR bundle's API module (`api.ts`) reads
   `window.__GLAZE_SITE_CONFIG__` at module-load time. The server must set up
   a `globalThis.window` mock *before* importing the SSR bundle. This is done
   with dynamic `await import()` in `server.mjs`.

3. **`StaticRouter` vs `BrowserRouter`.** React Router's `BrowserRouter` reads
   `window.location` (doesn't exist in Node). The SSR entry uses
   `StaticRouter` with a `location` prop instead.

4. **`hydrateRoot` vs `createRoot`.** The client entry uses `hydrateRoot()` to
   reuse the server-rendered DOM nodes. If there's no SSR sidecar (local dev),
   `hydrateRoot` still works — it just creates new DOM from scratch.

5. **Fallback to SPA.** If the SSR sidecar is down, the Go server catches the
   proxy error and serves `index.html` directly. The site degrades gracefully.

## 5. Gap Analysis: What Publish-Vault Needs Differently

Publish-Vault's architecture differs from Glazed in several important ways
that the SSR sidecar must account for.

### 5.1 Routing library: Wouter, not React Router

Glazed uses `react-router-dom` with `<Routes>/<Route>`. Publish-Vault uses
`wouter` with `<Switch>/<Route>`. This matters because:

- Wouter has no `StaticRouter` equivalent. On the server, we need a way to
  render components for a given URL without `window.location`.
- **Solution:** Create a minimal SSR-compatible routing wrapper that accepts a
  URL prop and renders the matching page component. On the server, we bypass
  Wouter's hook-based routing entirely and render the matched component
  directly based on URL parsing.

### 5.2 URL scheme: `/note/{slug}`, not `/{package}/{version}/sections/{slug}`

Glazed's URLs encode package, version, and section slug. Publish-Vault has a
simpler scheme:

| URL | What it shows |
|-----|---------------|
| `/` | Home page (first available note) |
| `/note/{slug}` | Single note |
| `/search` | Search page |

The SSR server must parse these URLs to decide which API calls to make.

### 5.3 Store structure

Glazed uses a factory pattern (`makeStore()`) for the SSR entry and a
singleton for the browser. Publish-Vault's `store.ts` only exports a singleton:

```typescript
// Current (publish-vault)
export const store = configureStore({ ... });
```

For SSR we need to add a `makeStore(preloadedState?)` factory function so each
SSR request gets its own store, just like Glazed does.

### 5.4 API data shape

Glazed's API returns `{ packages, sections, section }`. Publish-Vault's API
returns:

| Endpoint | Returns | Used by |
|----------|---------|----------|
| `GET /api/config` | `{ vaultName, pageTitle, notes }` | `VaultLayout` menubar |
| `GET /api/notes` | `NoteListItem[]` | `HomeRedirect` (picks home note) |
| `GET /api/notes/{slug}` | `Note` (full, with `html`, `backlinks`, `wikiLinks`) | `NotePage` |
| `GET /api/tree` | `FileNode` (hierarchical tree) | `Sidebar` |
| `GET /api/search?q=...` | `SearchResult[]` | `SearchPage` |
| `GET /api/tags` | `TagCount[]` | Tags display |

For SSR, the server needs to:
- For `/`: fetch `config`, `notes` (to pick the home note), and `tree`.
- For `/note/{slug}`: fetch `config`, `notes`, `tree`, and the specific note.
- For `/search`: fetch `config` and `tree` only (search is client-side).

### 5.5 Static vault mode

Publish-Vault supports `VITE_STATIC_VAULT=true` for standalone demo deployments
where data comes from an in-browser static module instead of the API. The SSR
sidecar always runs in backend mode (it has the Go API available), so this
doesn't affect the SSR implementation. But the SSR entry must not import the
static vault module.

### 5.6 Summary of differences

| Aspect | Glazed | Publish-Vault | SSR Impact |
|--------|--------|---------------|------------|
| Router | react-router-dom | wouter | Need custom SSR route matching |
| URLs | `/{pkg}/{ver}/sections/{slug}` | `/note/{slug}` | Simpler URL parsing |
| Store | Factory (`makeStore`) | Singleton | Must add factory |
| API | packages/sections/section | config/notes/note/tree/search/tags | Different pre-fetch logic |
| Static mode | No | Yes | SSR always uses backend mode |

## 6. Proposed Architecture

### 6.1 Component diagram

```
                         ┌──────────────────────┐
                         │  Browser / Crawler   │
                         └──────────┬───────────┘
                                    │ HTTP
                                    ▼
              ┌─────────────────────────────────────────┐
              │  Go Server (:8080)                       │
              │                                          │
              │  /api/*        → API handlers            │
              │  /vault-assets → vault file serving       │
              │  /assets/*     → static JS/CSS            │
              │  /*            → SSR proxy (--ssr-url)    │
              │                  OR SPA fallback          │
              └────────────┬─────────────┬───────────────┘
                           │             │
                    /api/* │             │ /* (pages)
                           │             ▼
                           │  ┌──────────────────────────┐
                           │  │  Node SSR Sidecar (:8089) │
                           │  │                           │
                           │  │  1. Parse URL             │
                           │  │  2. Fetch data from Go    │
                           │  │     /api/notes/{slug}     │
                           │  │     /api/notes            │
                           │  │     /api/config           │
                           │  │     /api/tree             │
                           │  │  3. renderToString()      │
                           │  │  4. Return full HTML      │
                           │  └──────────────────────────┘
                           │
                     JSON API data
```

### 6.2 SSR data flow

```
1. Browser requests /note/my-research-note

2. Go server: path is not /api, not /vault-assets, not /assets/*
   → --ssr-url is set → reverse-proxy to http://localhost:8089/note/my-research-note

3. Node server.mjs receives GET /note/my-research-note

4. URL parsing: route="note", slug="my-research-note"

5. Data pre-fetching from Go API (localhost:8080):
   - GET http://localhost:8080/api/config       → { vaultName, pageTitle, notes }
   - GET http://localhost:8080/api/notes          → [ { slug, title, ... }, ... ]
   - GET http://localhost:8080/api/tree           → { name, children: [...] }
   - GET http://localhost:8080/api/notes/my-research-note → { slug, title, html, backlinks, ... }

6. SSR rendering:
   a. Create fresh Redux store (makeStore())
   b. Upsert fetched data into RTK Query cache:
      - store.dispatch(vaultApi.util.upsertQueryData('getConfig', undefined, config))
      - store.dispatch(vaultApi.util.upsertQueryData('listNotes', undefined, notes))
      - store.dispatch(vaultApi.util.upsertQueryData('getTree', undefined, tree))
      - store.dispatch(vaultApi.util.upsertQueryData('getNote', slug, note))
   c. Call renderToString() with the note page component
   d. Return { html, preloadedState }

7. HTML assembly:
   - Read dist/index.html as template
   - Replace <div id="root"></div> with <div id="root">{html}</div>
   - Inject <script>window.__PRELOADED_STATE__ = {serialized state};</script>
   - Set <title>{note.title} — {vaultName}</title>
   - Add <meta name="description"> and Open Graph tags
   - Add JSON-LD structured data (WebPage, BreadcrumbList)

8. Return complete HTML → Go server → browser

9. Browser receives HTML with real content:
   - Crawlers see headings, text, links immediately
   - Browser loads JS bundle
   - entry-client.tsx reads window.__PRELOADED_STATE__
   - Creates Redux store with preloaded state
   - hydrateRoot() reuses server DOM nodes
   - App is interactive immediately
```

## 7. Decision Records

### Decision: SSR via sidecar, not embedded rendering

- **Context:** We need server-rendered HTML but the backend is Go, which cannot
  natively render React components.
- **Options considered:**
  1. Node.js sidecar container
  2. Embedded V8/Goja runtime in Go
  3. Pre-render all pages at build time (SSG)
  4. Headless Chrome pool
- **Decision:** Node.js sidecar (option 1).
- **Rationale:** This is the same pattern already proven in Glazed. The Go
  binary stays pure Go. The sidecar is a simple Express server that imports the
  SSR bundle. No CGO, no embedded V8, no headless browser. Local dev works
  without it.
- **Consequences:** Requires a second container in deployment. Adds ~50ms
  latency for page requests (Node boot + data fetch + render). The Go server
  must have a fallback for when the sidecar is unavailable.
- **Status:** accepted

### Decision: Custom SSR route matching instead of Wouter StaticRouter

- **Context:** Wouter doesn't provide a `StaticRouter` for server-side rendering.
  Its routing is hook-based (`useLocation()`), which requires `window.history`.
- **Options considered:**
  1. Mock `window.history` and `window.location` to make Wouter work in Node
  2. Switch to React Router (which has `StaticRouter`)
  3. Parse the URL in the SSR entry and render the matching component directly
- **Decision:** Option 3 — parse the URL and render directly.
- **Rationale:** This is the simplest approach. The SSR entry only needs to
  handle three URL patterns (`/`, `/note/{slug}`, `/search`). Parsing is
  trivial. Switching the entire app to React Router is a large refactor with no
  other benefit. Mocking window for Wouter is fragile.
- **Consequences:** The SSR entry has its own route parsing that must stay in
  sync with the client-side routes in `App.tsx`. If routes change, both must be
  updated.
- **Status:** accepted

### Decision: Store factory (makeStore) for SSR

- **Context:** The current store is a singleton module-level variable. SSR needs
  a fresh store per request to prevent data leaking between concurrent renders.
- **Options considered:**
  1. Refactor `store.ts` to export `makeStore()` and keep a browser singleton
  2. Create a separate SSR-only store
- **Decision:** Option 1 — refactor to a factory with a browser singleton.
- **Rationale:** This is the exact pattern used in Glazed (`store.ts` exports
  `makeStore()` plus a `store` singleton). It's minimal change and well-tested.
- **Consequences:** All existing imports of `store` from `store.ts` continue to
  work. The SSR entry calls `makeStore()` instead.
- **Status:** accepted

### Decision: hydrateRoot for client entry

- **Context:** The client must switch from `createRoot().render()` to
  `hydrateRoot()` to reuse server-rendered DOM nodes.
- **Options considered:**
  1. New `entry-client.tsx` with `hydrateRoot`, keep `main.tsx` for dev
  2. Modify `main.tsx` to use `hydrateRoot` always
- **Decision:** Option 1 — separate entry points.
- **Rationale:** `hydrateRoot` works even on an empty `<div id="root">` (it
  just creates new DOM), so technically option 2 would work. But having separate
  entries is cleaner: `main.tsx` for `vite dev`, `entry-client.tsx` for the
  production build. This matches the Glazed pattern.
- **Consequences:** The Vite build config must point to `entry-client.tsx` as
  the client entry. `vite dev` continues using `main.tsx`.
- **Status:** accepted

## 8. File Layout

### New files

```
publish-vault/
  web/
    src/
      entry-client.tsx    # Client entry with hydrateRoot (new)
      entry-server.tsx     # SSR entry with renderToString (new)
    server.mjs             # Node.js SSR HTTP server (new)
    ssr.Dockerfile         # Docker image for the sidecar (new)
```

### Modified files

```
publish-vault/
  web/
    src/store/store.ts     # Add makeStore() factory
    src/main.tsx            # Minimal changes (keep for vite dev)
    vite.config.ts          # Add SSR build config
    package.json            # Add SSR scripts, express dependency
    index.html              # Add <!--SSR_CONTENT--> and /*PRELOADED_STATE*/ markers
  backend/
    internal/server/
      server.go             # Add --ssr-url flag + reverse proxy logic
    cmd/retro-obsidian-publish/commands/serve/
      serve.go              # Add --ssr-url flag definition
```

## 9. Pseudocode and API Sketches

### 9.1 entry-server.tsx

This is the SSR entry point. The Node sidecar imports `renderApp()` from this
module after it's been compiled by Vite into `dist/ssr/entry-server.js`.

```typescript
// web/src/entry-server.tsx
import React from 'react';
import { renderToString } from 'react-dom/server';
import { Provider } from 'react-redux';
import { makeStore } from './store/store';
import { vaultApi } from './store/vaultApi';
import type {
  SiteConfig,
  NoteListItem,
  Note,
  FileNode,
} from './types';

export interface SSRData {
  config?: SiteConfig | null;
  notes?: NoteListItem[] | null;
  tree?: FileNode | null;
  note?: Note | null;       // present for /note/{slug} routes
}

export interface SSRResult {
  html: string;
  preloadedState: unknown;
}

export async function renderApp(url: string, data: SSRData): Promise<SSRResult> {
  const store = makeStore();

  // Preload RTK Query cache so components render with real data
  const preloadActions: Promise<unknown>[] = [];

  if (data.config) {
    preloadActions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData('getConfig', undefined, data.config)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.notes) {
    preloadActions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData('listNotes', undefined, data.notes)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.tree) {
    preloadActions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData('getTree', undefined, data.tree)
      ) as unknown as Promise<unknown>
    );
  }

  if (data.note) {
    preloadActions.push(
      store.dispatch(
        vaultApi.util.upsertQueryData('getNote', data.note.slug, data.note)
      ) as unknown as Promise<unknown>
    );
  }

  await Promise.all(preloadActions);

  // Parse URL and render the matching page
  const pathname = url.split('#')[0]?.split('?')[0] || '/';
  let html: string;

  if (pathname.startsWith('/note/')) {
    const slug = pathname.replace('/note/', '');
    html = renderToString(
      <Provider store={store}>
        <NotePageSSR slug={slug} />
      </Provider>
    );
  } else if (pathname === '/search') {
    html = renderToString(
      <Provider store={store}>
        <SearchPageSSR />
      </Provider>
    );
  } else {
    // Home page
    html = renderToString(
      <Provider store={store}>
        <HomeRedirectSSR />
      </Provider>
    );
  }

  return { html, preloadedState: store.getState() };
}
```

**Important note on routing:** Because Wouter doesn't support SSR, the SSR
entry doesn't use Wouter at all. Instead, it parses the URL and renders the
matching page component directly. Each page component (`NotePage`, `SearchPage`,
`HomeRedirect`) is wrapped in a thin SSR-compatible version that doesn't call
Wouter hooks.

### 9.2 entry-client.tsx

```typescript
// web/src/entry-client.tsx
import React from 'react';
import { hydrateRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import { makeStore } from './store/store';
import App from './App';
import './index.css';

declare global {
  interface Window {
    __PRELOADED_STATE__?: Record<string, unknown>;
  }
}

const preloadedState = window.__PRELOADED_STATE__;
delete window.__PRELOADED_STATE__;

const store = makeStore(preloadedState);

hydrateRoot(
  document.getElementById('root')!,
  <React.StrictMode>
    <Provider store={store}>
      <App />
    </Provider>
  </React.StrictMode>
);
```

### 9.3 store.ts refactor

```typescript
// web/src/store/store.ts (after refactor)
import { configureStore } from '@reduxjs/toolkit';
import { vaultApi } from './vaultApi';
import uiReducer from './uiSlice';

export function makeStore(preloadedState?: unknown) {
  return configureStore({
    reducer: {
      [vaultApi.reducerPath]: vaultApi.reducer,
      ui: uiReducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(vaultApi.middleware),
    ...(preloadedState ? { preloadedState } : {}),
  });
}

// Browser singleton (used by main.tsx and App.tsx)
export const store = makeStore();

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
```

### 9.4 server.mjs

The Node.js SSR sidecar. This is an Express server that:
1. Receives page requests proxied from the Go server
2. Fetches data from the Go API
3. Calls `renderApp()` to render React to HTML
4. Assembles complete HTML with metadata

```javascript
// web/server.mjs
import express from 'express';
import { readFileSync } from 'fs';

const PORT = parseInt(process.env.SSR_PORT || '8089', 10);
const API_BASE = process.env.API_BASE || 'http://localhost:8080';
const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';

// Load SSR bundle (dynamic import after any needed setup)
const { renderApp } = await import('./dist/ssr/entry-server.js');

const app = express();

// Health check
app.get('/health', (_req, res) => res.json({ ok: true }));

// Helper: fetch JSON from Go API
async function fetchAPI(path) {
  try {
    const res = await fetch(`${API_BASE}${path}`);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

// Parse URL to determine route
function parseRoute(pathname) {
  if (pathname.startsWith('/note/')) {
    return { route: 'note', slug: pathname.replace('/note/', '') };
  }
  if (pathname === '/search') {
    return { route: 'search' };
  }
  return { route: 'home' };
}

// Read index.html template
function getIndexHtml() {
  try {
    return readFileSync('./dist/index.html', 'utf-8');
  } catch {
    return `<!doctype html><html><head><title>Vault</title></head>
      <body><div id="root"></div></body></html>`;
  }
}

let indexTemplate = null;

// Catch-all handler
app.get('{*path}', async (req, res) => {
  try {
    const url = req.originalUrl;
    const pathname = req.path;
    const { route, slug } = parseRoute(pathname);

    // Load template once
    if (!indexTemplate) indexTemplate = getIndexHtml();

    // 1. Pre-fetch common data
    const config = await fetchAPI('/api/config');
    const notes = await fetchAPI('/api/notes');
    const tree = await fetchAPI('/api/tree');

    // 2. Pre-fetch route-specific data
    let note = null;
    if (route === 'note' && slug) {
      note = await fetchAPI(`/api/notes/${encodeURIComponent(slug)}`);
    }

    // 3. Render React to HTML
    const { html, preloadedState } = await renderApp(url, {
      config, notes, tree, note,
    });

    // 4. Assemble HTML
    let page = indexTemplate;

    // Inject SSR content
    page = page.replace(
      /<div id="root">([\s\S]*?)<\/div>/,
      `<div id="root">${html}</div>`
    );

    // Determine title
    const vaultName = config?.vaultName || 'Vault';
    const title = note?.title
      ? `${note.title} — ${vaultName}`
      : `${vaultName}`;
    const description = note?.excerpt
      || `${vaultName}: ${notes?.length || 0} notes`;

    // Inject preloaded state
    const serializedState = JSON.stringify(preloadedState)
      .replace(/</g, '\u003c')
      .replace(/>/g, '\u003e');

    page = page.replace(
      '</head>',
      `<script>window.__PRELOADED_STATE__=${serializedState};</script>
       <meta name="description" content="${description.replace(/"/g, '&quot;')}" />
       <meta property="og:title" content="${title.replace(/"/g, '&quot;')}" />
       <meta property="og:description" content="${description.replace(/"/g, '&quot;')}" />
       <title>${title}</title>
       </head>`
    );

    // Add noscript fallback
    const noscriptContent = note
      ? `<h1>${note.title}</h1><p>${note.excerpt || ''}</p>`
      : `<h1>${vaultName}</h1><p>${notes?.length || 0} notes</p>`;
    page = page.replace('</body>', `<noscript>${noscriptContent}</noscript></body>`);

    res.type('html').send(page);
  } catch (err) {
    console.error('SSR render error:', err);
    res.status(500).send('SSR render error');
  }
});

app.listen(PORT, () => {
  console.log(`SSR sidecar listening on :${PORT}`);
  console.log(`  API base: ${API_BASE}`);
});
```

### 9.5 Go server changes: SSR proxy

The Go server needs an `--ssr-url` flag and a reverse proxy for page requests.
This follows the exact pattern from `glazed/pkg/help/server/serve.go`.

**Pseudocode for server.go changes:**

```go
// server.go — add SSRURL to Config

type Config struct {
    VaultDir            string
    VaultName           string
    PageTitle           string
    Port                string
    ServeWeb            bool
    Watch               bool
    ReloadToken         string
    ReloadAllowLoopback bool
    SSRURL              string  // NEW: URL of SSR sidecar (e.g. http://localhost:8089)
}

func Run(ctx context.Context, cfg Config) error {
    // ... existing setup ...

    r := mux.NewRouter()
    h := api.NewWithProvider(state, api.PublicConfig{...})
    h.Register(r)
    r.HandleFunc("/api/healthz", healthHandler(state)).Methods("GET")
    r.PathPrefix("/vault-assets/").Handler(assetHandler(state)).Methods("GET", "HEAD")
    // ... reload handler ...

    if cfg.ServeWeb {
        spaHandler := web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"})

        if cfg.SSRURL != "" {
            // SSR mode: proxy page requests to sidecar
            ssrProxy := newSSRProxy(cfg.SSRURL, spaHandler)
            r.PathPrefix("/").Handler(ssrProxy)
        } else {
            // No SSR: serve SPA directly
            r.PathPrefix("/").Handler(spaHandler)
        }
    }

    // ... rest unchanged ...
}

// newSSRProxy returns a handler that reverse-proxies to the SSR sidecar.
// Falls back to the SPA handler if the sidecar is unavailable.
func newSSRProxy(ssrURL string, spaHandler http.Handler) http.Handler {
    ssrEndpoint, err := url.Parse(ssrURL)
    if err != nil {
        return spaHandler
    }
    proxy := &http.Client{Timeout: 10 * time.Second}

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip API and vault-assets routes
        if strings.HasPrefix(r.URL.Path, "/api/") ||
           strings.HasPrefix(r.URL.Path, "/vault-assets/") ||
           strings.HasPrefix(r.URL.Path, "/assets/") {
            spaHandler.ServeHTTP(w, r)
            return
        }

        // Proxy to SSR sidecar
        proxyURL := ssrEndpoint.ResolveReference(&url.URL{
            Path:     r.URL.Path,
            RawQuery: r.URL.RawQuery,
        })
        proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), nil)
        if err != nil {
            spaHandler.ServeHTTP(w, r)
            return
        }

        resp, err := proxy.Do(proxyReq)
        if err != nil {
            // Sidecar unavailable → fall back to SPA
            spaHandler.ServeHTTP(w, r)
            return
        }
        defer resp.Body.Close()

        if resp.StatusCode >= 500 {
            spaHandler.ServeHTTP(w, r)
            return
        }

        // Copy headers and body
        for k, vs := range resp.Header {
            for _, v := range vs {
                w.Header().Add(k, v)
            }
        }
        w.WriteHeader(resp.StatusCode)
        io.Copy(w, resp.Body)
    })
}
```

**Pseudocode for serve.go flag changes:**

```go
// serve.go — add SSR URL flag

fields.New("ssr-url", fields.TypeString,
    fields.WithDefault(""),
    fields.WithHelp("URL of the SSR sidecar (e.g. http://localhost:8089). "+
        "When set, page requests are reverse-proxied to the SSR server "+
        "for server-side rendering."),
),
```

## 10. Implementation Phases

### Phase 1: Store refactor + entry points (no sidecar yet)

**Goal:** Make the web app SSR-ready without changing runtime behavior.

1. Refactor `store/store.ts` to export `makeStore()` factory + `store` singleton.
2. Create `entry-client.tsx` with `hydrateRoot` (wraps existing `App`).
3. Update `vite.config.ts` to build from `entry-client.tsx` instead of `main.tsx`.
4. Verify `pnpm build` works and the app behaves identically.
5. Run `pnpm check` and `pnpm test`.

**Files changed:** `store/store.ts`, `entry-client.tsx` (new), `vite.config.ts`

### Phase 2: SSR entry point

**Goal:** Add `entry-server.tsx` and the SSR build target.

1. Create `entry-server.tsx` with `renderApp()` function.
2. Handle the Wouter-incompatibility by rendering page components directly
   based on URL parsing (no Wouter hooks on the server).
3. Add `build:ssr` script to `package.json`: `vite build --ssr src/entry-server.tsx --outDir dist/ssr`.
4. Add `ssr.noExternal` config to `vite.config.ts` for React, Redux, RTK Query.
5. Verify `pnpm build:ssr` produces `dist/ssr/entry-server.js`.
6. Write unit test for `entry-server.tsx` (similar to `glazed/web/src/entry-server.test.tsx`).

**Files changed:** `entry-server.tsx` (new), `package.json`, `vite.config.ts`

### Phase 3: Node.js sidecar

**Goal:** Create the Express SSR server.

1. Create `server.mjs` with Express app.
2. Implement URL parsing for `/`, `/note/{slug}`, `/search`.
3. Implement data pre-fetching from Go API.
4. Implement HTML assembly with preloaded state injection.
5. Add health check endpoint (`/health`).
6. Add `<noscript>` fallback content for non-JS agents.
7. Add JSON-LD structured data and `<meta>` tags.
8. Test manually: run Go server + sidecar, verify page source has real content.

**Files changed:** `server.mjs` (new), `package.json` (add express dep)

### Phase 4: Go server SSR proxy

**Goal:** Wire the Go server to proxy page requests to the sidecar.

1. Add `SSRURL` field to `server.Config`.
2. Add `--ssr-url` flag to the serve command.
3. Implement `newSSRProxy()` with fallback to SPA handler.
4. Add tests for the proxy (route to SSR for pages, skip for API/assets,
   fallback on sidecar failure).

**Files changed:** `server.go`, `serve.go`

### Phase 5: Docker and deployment

**Goal:** Deploy the sidecar alongside the Go server.

1. Create `ssr.Dockerfile` for the Node sidecar.
2. Update `docker-compose.yml` to add the sidecar service.
3. Update k3s manifests (if applicable) to add sidecar container.
4. Verify the deployment works end-to-end.

**Files changed:** `ssr.Dockerfile` (new), `docker-compose.yml`, deployment manifests

### Phase 6: SEO and a14y verification

**Goal:** Confirm the SSR output meets SEO and agent-readability goals.

1. Run `curl` against various URLs and verify HTML content.
2. Run a14y audit to check score improvement.
3. Verify search-engine-readable headings and text content.
4. Check hydration correctness (no React hydration warnings in console).

## 11. Testing Strategy

### Unit tests

- `entry-server.test.tsx`: Test `renderApp()` with mock data for each route.
  Verify it returns non-empty HTML and correct preloaded state.
- `store.test.ts`: Verify `makeStore()` creates independent stores.
- Go proxy tests: Test SSR proxy routing and fallback behavior
  (same pattern as `glazed/pkg/help/server/serve_test.go`).

### Integration tests

- Start Go server + Node sidecar. Hit `/note/some-slug`, verify:
  - Response contains the note title in `<h1>` or visible text.
  - Response contains `window.__PRELOADED_STATE__`.
  - Response has `<meta name="description">` and OG tags.
  - After hydration, the page is interactive (no React errors).

### Manual verification checklist

```
1. curl http://localhost:8080/ | grep "<title>"
2. curl http://localhost:8080/note/index | grep "<h1>"
3. curl http://localhost:8080/note/index | grep "__PRELOADED_STATE__"
4. Open browser, navigate to a note, check DevTools console for hydration errors
5. Open browser, disable JS, reload page, verify content is visible
6. Kill sidecar, verify Go server falls back to SPA
```

## 12. Risks and Alternatives

### Risks

1. **Wouter SSR incompatibility.** Wouter uses hooks (`useLocation`) that
   don't work in Node. Mitigation: render page components directly in the SSR
   entry without using Wouter's routing. This means the SSR route table must
   be maintained in sync with `App.tsx`.

2. **Hydration mismatches.** If server-rendered HTML differs from client-rendered
   HTML, React will warn and may re-render. Common causes: timestamps,
   random IDs, `Date.now()` calls. Mitigation: ensure SSR renders the same
   component tree as the client, and avoid non-deterministic rendering.

3. **Sidecar latency.** Each page request adds ~50-100ms for Node to fetch
   API data and render React. Mitigation: acceptable for SEO pages; the SPA
   fallback is instant for interactive users.

4. **Data freshness.** The sidecar fetches data from the Go API on every
   request. If the vault is very large, this could be slow. Mitigation:
   the Go API already caches parsed notes in memory; fetches are fast.

### Alternatives considered

1. **Static site generation (SSG):** Pre-render all notes at build time.
   Rejected because the vault can change at runtime (file watcher, git-sync).
   SSG would require rebuilding on every vault change.

2. **Embedded V8/Goja in Go:** Render React inside Go using an embedded JS
   runtime. Rejected because it adds CGO complexity, has limited ES module
   support, and doesn't match the proven Glazed pattern.

3. **Headless Chrome pool:** Use Puppeteer/Playwright to render pages.
   Rejected because it's heavy, slow, and the sidecar approach is simpler.

## 13. Key File References

| File | Role |
|------|------|
| `publish-vault/web/src/App.tsx` | Root component with Wouter routing |
| `publish-vault/web/src/store/store.ts` | Redux store (needs makeStore refactor) |
| `publish-vault/web/src/store/vaultApi.ts` | RTK Query API slice with upsertQueryData support |
| `publish-vault/web/src/store/uiSlice.ts` | UI state slice |
| `publish-vault/web/src/main.tsx` | Current entry point (will be dev-only) |
| `publish-vault/web/vite.config.ts` | Vite config (needs SSR build target) |
| `publish-vault/web/package.json` | Package config (needs SSR scripts) |
| `publish-vault/backend/internal/server/server.go` | Go server (needs SSR proxy) |
| `publish-vault/backend/internal/server/runtime.go` | RuntimeState with Vault and Search |
| `publish-vault/backend/internal/api/api.go` | REST API endpoints |
| `publish-vault/backend/internal/web/static.go` | SPA handler |
| `publish-vault/backend/cmd/retro-obsidian-publish/commands/serve/serve.go` | Serve command flags |
| `glazed/web/src/entry-server.tsx` | Reference: Glazed SSR entry |
| `glazed/web/src/entry-client.tsx` | Reference: Glazed client hydration entry |
| `glazed/web/server.mjs` | Reference: Glazed Node SSR server |
| `glazed/web/src/store.ts` | Reference: Glazed makeStore pattern |
| `glazed/pkg/help/server/serve.go` | Reference: Go SSR proxy implementation |
