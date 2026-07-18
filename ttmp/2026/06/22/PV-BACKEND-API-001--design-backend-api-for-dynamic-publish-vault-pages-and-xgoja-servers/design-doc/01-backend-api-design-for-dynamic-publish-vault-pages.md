---
Title: Backend API design for dynamic publish-vault pages
Ticket: PV-BACKEND-API-001
Status: active
Topics:
    - backend
    - api
    - ssr
    - xgoja
    - obsidian-vault
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md
      Note: xgoja HTTP provider serve command and Express route model
    - Path: internal/api/api.go
      Note: Current JSON API contract and handler shapes
    - Path: internal/server/server.go
      Note: Current server routing
    - Path: internal/vault/vault.go
      Note: Disk-backed vault model
    - Path: web/server.mjs
      Note: Current Node SSR prefetch and HTML assembly behavior
    - Path: web/src/store/vaultApi.ts
      Note: Frontend RTK Query API dependency surface
ExternalSources: []
Summary: Design for extracting publish-vault's disk-backed vault pipeline into a backend API that can be implemented by Go or xgoja/JavaScript while preserving SSR and agent-readable markdown mirrors.
LastUpdated: 2026-06-22T21:45:00-04:00
WhatFor: Use when implementing dynamic publish-vault pages or packaging a JS-backed server with xgoja.
WhenToUse: Before changing publish-vault backend contracts, SSR prefetching, note data models, or xgoja integration.
---


# Backend API design for dynamic publish-vault pages

## Executive summary

`publish-vault` is currently a Go server that loads Markdown notes from a filesystem vault, parses them into an in-memory `vault.Vault`, builds a Bleve search index, exposes a small `/api/*` JSON contract, and serves a React SPA with an optional Node SSR sidecar. It is not merely serving Markdown from disk: Markdown files are parsed, rendered to HTML, enriched with frontmatter, tags, backlinks, wiki-link resolution, asset URL rewrites, and search metadata before the frontend or SSR sidecar sees them.

The future-friendly path is to make the current Go vault loader one implementation of a stable backend API, not the hidden source of truth. The React app and SSR layer should consume a `PublishVaultBackend` contract that can be served by:

1. the existing Go disk-backed implementation,
2. a future dynamic Go implementation,
3. an xgoja-packaged JavaScript implementation, or
4. a hybrid Go shell where xgoja JavaScript routes implement page/data behavior while Go continues to own static assets, reloads, and safety-critical filesystem serving.

The recommended design is an incremental backend API v1 with typed resources (`site`, `routes`, `pages`, `notes`, `tree`, `search`, `tags`, `assets`) and a render-oriented endpoint for SSR. Keep compatibility shims for the existing frontend endpoints while moving implementation behind interfaces. Then package a JS backend using xgoja's HTTP provider and `express` module, with optional embedding into a Go-owned host when publish-vault must keep its current listener and mux.

## Problem statement and scope

The desired outcome is: "add a backend API for publish-vault so we can build dynamic pages instead of just serving markdown from disk" and eventually "package it with xgoja to quickly write a JS version of a server that fuels a publish-vault, with SSR too."

This design covers:

- how publish-vault works today,
- what must be abstracted to support dynamic pages,
- the minimum API contract the frontend and SSR need,
- how to preserve markdown mirrors and agent-readable routes,
- how xgoja can host a JS implementation,
- a phased implementation plan with validation points.

This design does not implement the API yet. It intentionally avoids prescribing a database schema or CMS model; the contract should support multiple data sources.

## Current-state architecture

### Server startup and routing

The CLI entrypoint builds the Cobra command tree and delegates `serve` to `internal/server.Run` (`cmd/retro-obsidian-publish/main.go:10-19`, `cmd/retro-obsidian-publish/commands/root.go:16-44`). The `serve` command accepts `--vault`, `--serve-web`, `--watch`, `--ssr-url`, reload flags, page-title/vault-name flags, and favicon configuration (`cmd/retro-obsidian-publish/commands/serve/serve.go:24-36`, `cmd/retro-obsidian-publish/commands/serve/serve.go:57-97`). It requires `--vault` or `VAULT_DIR` (`cmd/retro-obsidian-publish/commands/serve/serve.go:117-122`) and passes a `server.Config` to `appserver.Run` (`cmd/retro-obsidian-publish/commands/serve/serve.go:123-127`).

`server.Run` validates the vault directory and port, loads runtime state, derives public site names, registers API routes, asset routes, optional reload routes, and either serves the SPA directly or proxies page requests to the SSR sidecar (`internal/server/server.go:41-120`). The current route layout is:

- `GET /api/config`
- `GET /api/notes`
- `GET /api/notes/{slug}`
- `GET /api/notes/{slug}/raw`
- `GET /api/tree`
- `GET /api/search?q=...`
- `GET /api/tags`
- `GET /api/healthz`
- `POST /api/admin/reload` when enabled
- `GET/HEAD /vault-assets/*`
- `GET/HEAD /favicon.ico` and `/favicon.svg`
- page catch-all for SPA or SSR
- agent routes such as `/AGENTS.md`, `/llms.txt`, `/sitemap.md`, `/sitemap.xml`, `/index.md`, and `/note/{slug}.md`.

### Runtime state and reloads

`RuntimeState` owns the active `*vault.Vault` and `*search.Index` behind an RW mutex (`internal/server/runtime.go:12-21`). Initial load and reload both call `loadVaultAndSearch`, which resolves symlinks, loads the vault, and builds the search index (`internal/server/runtime.go:77-96`). `Reload` builds a new vault/search pair before swapping it into service, leaving the previous state active if loading fails (`internal/server/runtime.go:60-74`).

The optional watcher updates individual Markdown files. It watches all directories under the vault root, ignores non-`.md` events, debounces events, reloads notes, and keeps the search index in sync (`internal/watcher/watcher.go:36-67`, `internal/watcher/watcher.go:76-137`).

### Content loading and transformation

The vault implementation scans every `.md` file under the root while skipping hidden directories (`internal/vault/vault.go:70-105`). Each note is parsed by `parser.Parse`, receives a slug derived from relative path, and stores frontmatter, tags, excerpt, rendered HTML, raw Markdown, wiki links, and modification time (`internal/vault/vault.go:108-159`).

After the scan, the vault builds a wiki-link index, backlinks, and final HTML (`internal/vault/vault.go:102-105`). Wiki-link resolution maps short Obsidian link targets and suffixes to full vault slugs (`internal/vault/vault.go:162-212`). HTML is rebuilt to rewrite wiki-link hrefs, display text, and image URLs (`internal/vault/vault.go:214-234`). Assets are exposed through `/vault-assets/*` after path cleaning and escaping (`internal/vault/vault.go:237-309`, `internal/server/server.go:183-214`).

The parser uses goldmark with frontmatter, GFM, tables, strikethrough, task lists, footnotes, auto heading IDs, hard wraps, and unsafe HTML for placeholders (`internal/parser/parser.go:41-67`). It extracts wiki links before rendering, normalizes frontmatter, extracts title/tags/excerpt, and returns rendered HTML plus metadata (`internal/parser/parser.go:75-97`).

### API shape and frontend dependence

`internal/api` already describes the public REST API in comments and code (`internal/api/api.go:1-9`, `internal/api/api.go:58-67`). The frontend RTK Query layer depends directly on those endpoints in backend mode (`web/src/store/vaultApi.ts:49-148`). It also has a static demo mode that swaps API calls for an in-browser static vault module (`web/src/store/vaultApi.ts:25-33`, `web/src/store/vaultApi.ts:56-148`). This is useful evidence that a backend abstraction already exists at the TypeScript query layer, but it is not yet formalized as a versioned server contract.

The `Note` JSON model contains `slug`, `title`, `path`, `frontmatter`, `tags`, `excerpt`, `html`, `rawMarkdown`, `wikiLinks`, `backlinks`, and `modTime` (`internal/vault/vault.go:17-30`, `web/src/types/index.ts:26-38`). `NotePage` fetches a note, all notes, and config; renders `note.html`; derives backlink display rows from the note's backlink slug list; and sets the browser title (`web/src/components/pages/NotePage/NotePage.tsx:56-87`, `web/src/components/pages/NotePage/NotePage.tsx:145-208`). Search uses `/api/search` and `/api/tags` (`web/src/components/pages/SearchPage/SearchPage.tsx:43-47`, `web/src/components/pages/SearchPage/SearchPage.tsx:103-139`).

### SSR sidecar behavior

When `--ssr-url` is set, Go serves static frontend assets itself and proxies page requests to the SSR sidecar (`internal/server/server.go:104-116`). The reverse proxy falls back to the SPA if the sidecar URL is invalid, unavailable, or returns a 5xx (`internal/server/server.go:261-294`).

The Node SSR sidecar prefetches `/api/config`, `/api/notes`, `/api/tree`, and route-specific `/api/notes/{slug}` before calling `renderApp` (`web/server.mjs:190-221`). The React SSR entry then seeds RTK Query cache with those data objects and renders the real app with `StaticRouter` (`web/src/entry-server.tsx:46-117`). The sidecar also duplicates `chooseHomeSlug` from `App.tsx` (`web/server.mjs:101-140`, `web/src/App.tsx:58-95`), computes metadata and markdown alternate links, injects JSON-LD, canonical links, preloaded Redux state, and a noscript fallback (`web/server.mjs:224-335`).

### Agent-readable markdown mirrors

The Go page handler intercepts agent routes before SPA/SSR handling (`internal/server/agent_markdown.go:18-69`). It renders `/AGENTS.md`, `/llms.txt`, `/sitemap.md`, `/sitemap.xml`, `/index.md`, and `/note/{slug}.md` from the current vault snapshot (`internal/server/agent_markdown.go:109-218`, `internal/server/agent_markdown.go:220-267`). It also supports content negotiation for `Accept: text/markdown` (`internal/server/agent_markdown.go:51-74`). The note mirror is currently derived from rendered HTML, not raw Markdown (`internal/server/agent_markdown.go:243-247`).

## Gap analysis

1. **The current API is note-centric, not page-centric.** The frontend route `/note/{slug}` maps directly to a note. Dynamic publish-vault pages need non-note routes, route metadata, redirects, generated pages, feeds, collections, and possibly parameterized content.
2. **The server contract is implicit.** Go structs and TS interfaces match today, but there is no versioned schema or backend capability discovery. xgoja/JS implementations would have to reverse-engineer the current endpoints.
3. **SSR prefetch logic is duplicated and route-specific.** The sidecar contains its own route parser and home-note chooser (`web/server.mjs:91-140`), which will get worse when dynamic pages are added.
4. **Markdown mirrors are Go-only.** Agent-readable pages are generated directly from `RuntimeState`. A JS backend could produce dynamic HTML, but the Go mirror code would not know how to render markdown alternatives for non-note dynamic pages.
5. **Search is tied to the Go vault index.** Dynamic pages may need to participate in search, but today only vault notes are indexed (`internal/search/search.go:40-52`, `internal/search/search.go:79-99`).
6. **Assets are filesystem-specific.** `/vault-assets/*` safely serves files from the resolved vault root (`internal/server/server.go:183-214`), but a JS backend may need virtual, embedded, or remote assets.
7. **xgoja can serve HTTP, but publish-vault does not expose a backend provider.** xgoja's HTTP provider can build generated `serve` commands and allow JS route registration (`go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md:19-81`), but publish-vault currently has no runtime module that exposes vault/page primitives to JS.

## Proposed architecture

### 1. Introduce a backend contract package

Add a Go package such as `internal/backend` or `pkg/publishvault/backend` with an interface that represents the data/rendering needs of the UI, SSR, markdown mirrors, and search.

```go
type Backend interface {
    Site(ctx context.Context) (SiteConfig, error)
    Routes(ctx context.Context) ([]Route, error)
    Resolve(ctx context.Context, path string, opts ResolveOptions) (Page, error)
    Note(ctx context.Context, slug string, opts NoteOptions) (Note, error)
    Notes(ctx context.Context, opts ListOptions) ([]NoteListItem, error)
    Tree(ctx context.Context) (FileNode, error)
    Search(ctx context.Context, q string, opts SearchOptions) ([]SearchResult, error)
    Tags(ctx context.Context) ([]TagCount, error)
    Asset(ctx context.Context, path string) (Asset, error)
    Reload(ctx context.Context) error
}
```

Keep the existing disk-backed implementation by wrapping `RuntimeState`, `vault.Vault`, and `search.Index`. The first refactor should be behavior-preserving: existing `/api/notes`, `/api/tree`, `/api/search`, `/api/tags`, and `/vault-assets` should call the backend interface but return identical JSON.

### 2. Version the HTTP API under `/api/v1`

Keep existing endpoints as compatibility aliases, but make `/api/v1` the contract future backends implement.

Recommended v1 endpoints:

| Endpoint | Purpose | Notes |
| --- | --- | --- |
| `GET /api/v1/site` | public site config and capabilities | replaces/aliases `/api/config` |
| `GET /api/v1/routes` | route manifest for frontend/SSR | includes home route, notes, dynamic pages, search, markdown alternates |
| `GET /api/v1/page?path=/...` | resolve any public route to a renderable page | removes SSR route parser duplication |
| `POST /api/v1/render` | batch render/prefetch contract for SSR | accepts URL/path and returns page + common data |
| `GET /api/v1/notes` | note list | compatibility with current `/api/notes` |
| `GET /api/v1/notes/{slug}` | note detail | compatibility with current `/api/notes/{slug}` |
| `GET /api/v1/notes/{slug}/raw` | raw source when available | may return 404/410 for dynamic notes without raw source |
| `GET /api/v1/tree` | navigation tree | can include dynamic folders/items |
| `GET /api/v1/search?q=...` | searchable pages/notes | should include `type` and `url` in v1 |
| `GET /api/v1/tags` | tag counts | includes dynamic tagged pages |
| `GET /api/v1/markdown?path=/...` | agent markdown mirror | generalizes `/index.md` and `/note/*.md` |
| `GET /api/v1/assets/{path...}` | backend-owned assets | optional alternative to `/vault-assets/*` |

The compatibility layer should continue returning the current `Note`, `NoteListItem`, `FileNode`, `SearchResult`, `TagCount`, and `SiteConfig` shapes until the frontend is migrated.

### 3. Add a render contract for SSR

Replace route-specific sidecar prefetch code with one backend call:

```http
POST /api/v1/render
Content-Type: application/json

{
  "url": "/note/research/example?view=full",
  "mode": "ssr",
  "include": ["config", "navigation", "page", "searchHints"],
  "accept": ["html", "markdown"]
}
```

Response sketch:

```json
{
  "status": 200,
  "config": { "vaultName": "go-go-parc", "pageTitle": "go-go-parc", "notes": 123 },
  "route": { "path": "/note/research/example", "kind": "note", "canonicalPath": "/note/research/example", "markdownPath": "/note/research/example.md" },
  "page": {
    "kind": "note",
    "slug": "research/example",
    "title": "Example",
    "description": "...",
    "html": "<article>...</article>",
    "markdown": "# Example\n...",
    "frontmatter": {},
    "tags": [],
    "backlinks": [],
    "modTime": "2026-06-22T...Z"
  },
  "navigation": { "notes": [...], "tree": {...}, "tags": [...] },
  "preloadedState": {
    "legacy": {
      "config": {...},
      "notes": [...],
      "tree": {...},
      "note": {...}
    }
  }
}
```

The Node SSR sidecar or future xgoja SSR server can use the same call. The backend owns route resolution, home-page selection, canonical URLs, markdown alternates, 404 vs 200 decisions, and common navigation preloads.

### 4. Model pages separately from notes

Add a v1 `Page` type. A note can be one kind of page, but not every page is a note.

```ts
type PageKind = "note" | "index" | "search" | "collection" | "dynamic" | "not_found"

interface Page {
  kind: PageKind
  path: string
  canonicalPath: string
  markdownPath?: string
  title: string
  description?: string
  html?: string
  markdown?: string
  data?: unknown
  note?: Note
  status?: number
  headers?: Record<string, string>
  modTime?: string
}
```

This lets dynamic JS backends return generated pages without pretending every route is a Markdown note. The existing React `NotePage` can keep using `Note` initially. New routes can use a `PageRoute` component that renders `Page.html` or dispatches by `kind`.

### 5. Keep markdown mirrors as first-class output

Do not treat markdown mirrors as a side effect of Go's vault implementation. Add backend methods/endpoints for markdown representations:

- `GET /api/v1/markdown?path=/` for home,
- `GET /api/v1/markdown?path=/note/foo` for a note,
- route aliases `/index.md` and `/note/foo.md`,
- optional `/page/foo.md` for dynamic pages.

For disk-backed notes, the backend can continue deriving mirrors from rendered HTML at first. For JS dynamic pages, the backend should be able to return an explicit markdown body. This is important for a14y/agent readability and avoids requiring Go to reverse-render arbitrary dynamic HTML.

### 6. Add capabilities and source metadata

`GET /api/v1/site` should include backend capabilities so the frontend and SSR can adapt.

```json
{
  "vaultName": "go-go-parc",
  "pageTitle": "go-go-parc",
  "notes": 123,
  "apiVersion": "1",
  "backend": { "kind": "disk-go", "name": "publish-vault", "version": "..." },
  "capabilities": {
    "rawMarkdown": true,
    "dynamicPages": false,
    "serverRender": true,
    "markdownMirrors": true,
    "assets": ["vault-assets"],
    "reload": true
  }
}
```

A JS backend can advertise `kind: "xgoja-js"`, `dynamicPages: true`, and `rawMarkdown: false` for generated pages.

## xgoja integration design

### Observed xgoja capabilities

xgoja can build custom Go binaries from v2 specs (`go-go-goja/cmd/xgoja/root.go:17-25`, `go-go-goja/cmd/xgoja/cmd_build.go:81-143`). The HTTP provider contributes a `serve` command where JavaScript route scripts register routes through `require("express")` while the provider owns listener lifecycle and graceful shutdown (`go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md:19-81`). The Express module exposes `express.app()`, planned route builders, static mounts, and handler mounting (`go-go-goja/modules/express/express.go:89-219`). The example `13-http-serve-jsverbs` shows a generated binary selecting the HTTP provider and `express` runtime module (`go-go-goja/examples/xgoja/13-http-serve-jsverbs/xgoja.yaml:10-39`) and registering JS routes (`go-go-goja/examples/xgoja/13-http-serve-jsverbs/verbs/sites.js:9-15`).

xgoja also supports a hybrid Go-owned host: generated runtime packages can receive host services, and the HTTP provider can register JS routes into an external `*gojahttp.Host` with `OwnsListen: false` (`go-go-goja/cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md:275-304`, `go-go-goja/examples/xgoja/14-generated-runtime-package/README.md:45-52`).

### Recommended packaging modes

#### Mode A: standalone JS publish-vault server

A generated xgoja binary owns the HTTP listener. JavaScript implements `/api/v1/*`, SSR/page routes, and optional static mounts.

Use when experimenting quickly or building a fully JS-backed publish-vault:

```yaml
schema: xgoja/v2
name: publish-vault-js
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: http
      name: express
      as: express
sources:
  - id: site
    kind: jsverbs
    from:
      dir: ./server
    language: javascript
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [site]
```

JS sketch:

```js
__package__({ name: "publishVault" })
__verb__("serve", { name: "serve", output: "text" })
function serve() {
  const express = require("express")
  const app = express.app()
  const backend = createBackendFromConfig()

  app.get("/api/v1/site").public().handle((_ctx, res) => res.json(backend.site()))
  app.get("/api/v1/routes").public().handle((_ctx, res) => res.json(backend.routes()))
  app.post("/api/v1/render").public().handle((ctx, res) => res.json(backend.render(ctx.body)))
  app.get("/").public().handle((ctx, res) => res.html(renderHTML(backend.resolve(ctx.path))))
}
```

This mode is fastest for dynamic page experiments but must reimplement safe asset serving, reload semantics, and markdown mirrors unless those are provided as modules.

#### Mode B: Go shell with JS backend plugin

The existing Go `publish-vault` server owns the listener, static assets, admin reloads, CORS, favicon, and agent route aliases. It embeds or loads an xgoja runtime that implements the `Backend` interface.

Use when retaining current deployment behavior matters. This mode should use xgoja generated package host services or a custom native module that exposes publish-vault primitives to JS. JavaScript returns `Page`, `Note`, and search data; Go adapts that to `/api/v1` and compatibility routes.

#### Mode C: Go shell with JS dynamic routes only

Keep disk-backed notes/search in Go, but mount a JS `gojahttp.Host` under a prefix or route fallback for dynamic pages. xgoja docs explicitly support injecting an external host so JavaScript registers routes into a Go-owned host (`go-go-goja/cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md:290-304`).

Use this as the lowest-risk bridge:

- Go keeps `/api/notes`, `/vault-assets`, `/AGENTS.md`, reloads, and SSR proxy.
- JS adds `/api/v1/page`, `/api/v1/render`, and dynamic route handlers.
- React gradually moves from note-only routes to page routes.

## Decision records

### Decision: Make pages first-class rather than overloading notes

- **Context:** Current routes are mostly `/note/{slug}`, but the goal is dynamic pages that may not correspond to Markdown files.
- **Options considered:** Extend `Note` with optional dynamic fields; add a separate `Page` model; let frontend call arbitrary backend-specific endpoints.
- **Decision:** Add a separate `Page` model and keep `Note` as a note-specific resource.
- **Rationale:** This preserves compatibility while allowing generated pages, collection pages, search pages, redirects, and 404s to have their own representation.
- **Consequences:** The frontend needs a new page route/query layer, but existing note UI can remain intact during migration.
- **Status:** proposed

### Decision: Add `/api/v1/render` for SSR prefetch

- **Context:** SSR currently duplicates route parsing and home-note selection in Node (`web/server.mjs:91-140`). Dynamic routes would multiply this duplication.
- **Options considered:** Keep per-route SSR fetches; let SSR import backend logic; add a render/prefetch API.
- **Decision:** Add a backend-owned render/prefetch endpoint consumed by Node SSR, xgoja SSR, and future clients.
- **Rationale:** The backend should own route resolution, canonical URLs, markdown alternates, and status codes. SSR should render a prepared page payload.
- **Consequences:** Requires a new API and frontend preload adapter, but reduces duplication and makes JS backends easier.
- **Status:** proposed

### Decision: Keep existing endpoints as compatibility aliases

- **Context:** The frontend and tests currently depend on `/api/config`, `/api/notes`, `/api/tree`, `/api/search`, and `/api/tags`.
- **Options considered:** Breaking migration to `/api/v1`; maintain old endpoints forever; add aliases during migration.
- **Decision:** Add `/api/v1` as the formal contract and keep existing routes as aliases backed by the same interface.
- **Rationale:** This enables incremental implementation without breaking current deployments.
- **Consequences:** There will be a temporary compatibility layer to test and eventually deprecate.
- **Status:** proposed

### Decision: Prefer hybrid xgoja mode before standalone replacement

- **Context:** xgoja can own a full HTTP server, but publish-vault already has safe vault asset serving, reload, markdown mirror, and SSR fallback behavior.
- **Options considered:** Replace Go server with xgoja server immediately; embed xgoja behind Go; add only API-level compatibility.
- **Decision:** Implement the backend interface in Go first, then add a hybrid xgoja backend/plugin path before attempting standalone replacement.
- **Rationale:** This keeps deployment behavior stable and lets JS dynamic pages reuse the current server's safety and SEO/agent features.
- **Consequences:** Requires a Go<->JS adapter, but avoids reimplementing everything in JavaScript at once.
- **Status:** proposed

## Implementation phases

### Phase 1: Formalize current contract without behavior change

1. Add `internal/backend` types mirroring current API structs.
2. Add a `DiskBackend` wrapper around `RuntimeState`.
3. Move route handlers in `internal/api` to call `backend.Backend` instead of directly accessing `vault.Vault` and `search.Index`.
4. Add `/api/v1/site` aliases for `/api/config` and `/api/v1/notes` aliases for `/api/notes`.
5. Add contract tests that compare old and v1 outputs for the example vault.

Validation:

```bash
cd publish-vault
go test ./... -count=1
curl -fsS http://localhost:8080/api/config
curl -fsS http://localhost:8080/api/v1/site
```

### Phase 2: Add route/page/render resources

1. Add `Route`, `Page`, `RenderRequest`, and `RenderResponse` types.
2. Implement `DiskBackend.Resolve` for `/`, `/note/{slug}`, `/search`, and 404.
3. Implement `/api/v1/routes`, `/api/v1/page`, and `/api/v1/render`.
4. Move `chooseHomeSlug` logic from frontend/SSR into the backend response while leaving TS fallback for compatibility.
5. Update Node SSR to call `/api/v1/render` and seed RTK Query from the legacy section of the render response.

Validation:

```bash
curl -fsS -X POST http://localhost:8080/api/v1/render \
  -H 'content-type: application/json' \
  -d '{"url":"/"}' | jq .route.kind
```

### Phase 3: Generalize markdown mirrors

1. Add backend `Markdown(ctx, path)` or `Resolve(..., AcceptMarkdown)` support.
2. Reimplement `/index.md`, `/note/{slug}.md`, and content negotiation through the backend.
3. Add `/api/v1/markdown?path=...`.
4. Add tests for note, home, missing note, and a dynamic page fixture.

### Phase 4: Add xgoja backend adapter

1. Define a JS-facing backend module contract, for example `require("publish-vault/backend")` helper types or plain exported JS functions.
2. Implement a Go adapter that calls JS functions through xgoja and converts results to backend `Page`/`Note`/`SearchResult` types.
3. Start with read-only functions: `site`, `routes`, `resolve`, `render`, `search`, `tags`.
4. Add a sample `xgoja.yaml` and JS server in `examples/` or `examples/xgoja-publish-vault`.
5. Use hybrid mode first: Go owns listener; JS provides backend responses.

### Phase 5: Standalone xgoja server

1. Package a standalone xgoja app using the HTTP provider.
2. Serve `/api/v1/*` and SSR routes from JavaScript.
3. Decide whether to embed the React build as xgoja assets or continue to use Go/Vite build artifacts.
4. Add smoke tests similar to existing xgoja HTTP examples: health, home HTML, `/api/v1/site`, dynamic page, markdown mirror.

## Test strategy

- **Contract tests:** golden JSON for `/api/config` vs `/api/v1/site`, `/api/notes` vs `/api/v1/notes`, and `/api/notes/{slug}` vs `/api/v1/notes/{slug}`.
- **Route resolution tests:** home, note, search, unknown route, markdown alternate, canonical path.
- **SSR tests:** update `web/src/entry-server.test.tsx` or add sidecar tests to assert `/api/v1/render` data preloads RTK Query correctly.
- **Agent markdown tests:** keep existing `agent_markdown_test.go` behavior and add dynamic page markdown fixtures.
- **xgoja smoke tests:** generated binary starts, registers JS routes, returns valid `/api/v1/site`, renders a dynamic page, and exits cleanly.
- **Security tests:** asset traversal, hidden files, raw Markdown availability flags, JS backend output escaping, SSR state serialization.

## Risks and mitigations

- **Risk: API surface grows too quickly.** Mitigation: start with compatibility aliases and one `render` endpoint before adding optional features.
- **Risk: SSR and frontend state diverge.** Mitigation: backend-owned render response with explicit legacy preload section.
- **Risk: JS backend returns unsafe HTML.** Mitigation: document trust boundary; consider sanitizer/capability flag; keep markdown/source rendering explicit.
- **Risk: xgoja standalone mode reimplements safe asset serving poorly.** Mitigation: build hybrid mode first and expose a reusable asset handler/module.
- **Risk: raw Markdown assumptions break for dynamic pages.** Mitigation: capabilities and per-page `source`/`rawAvailable` fields.

## Open questions

1. Should the formal backend package live under `internal/backend` first, or public `pkg/publishvault/backend` for external xgoja adapters?
2. Should dynamic JS pages return pre-rendered HTML, structured data plus React component IDs, markdown, or all three?
3. Should xgoja SSR use React server rendering inside goja, or should SSR remain a Node sidecar until the API contract is stable?
4. Should search indexing for dynamic pages be backend-supplied (`Search`) or centralized by Go over a `SearchDocument` feed?
5. Should the API be documented as OpenAPI, protobuf, or TypeScript-first schemas?

## References

- `cmd/retro-obsidian-publish/commands/serve/serve.go`: CLI flags and server configuration.
- `internal/server/server.go`: runtime startup, routes, SSR proxy, asset serving.
- `internal/server/runtime.go`: reloadable vault/search state.
- `internal/api/api.go`: current JSON API handlers.
- `internal/vault/vault.go`: note model, vault loading, wiki-link/backlink/asset processing.
- `internal/parser/parser.go`: Markdown parsing and HTML generation.
- `internal/server/agent_markdown.go`: markdown mirror and agent-readable route generation.
- `web/src/store/vaultApi.ts`: frontend API client contract.
- `web/server.mjs`: Node SSR prefetch/render behavior.
- `web/src/entry-server.tsx`: RTK Query cache preloading for SSR.
- `go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md`: xgoja HTTP serve command model.
- `go-go-goja/cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`: xgoja host-service and external-host integration.
