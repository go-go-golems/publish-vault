---
Title: Page title, SSR sidebar, and HTML layout design and implementation guide
Ticket: RETRO-SEO-009
Status: active
Topics:
    - seo
    - ssr
    - html-layout
    - page-title
    - meta-tags
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/api/api.go
      Note: |-
        /api/config endpoint, SiteConfig contract with vaultName/pageTitle
        /api/config endpoint and SiteConfig contract
    - Path: internal/parser/parser.go
      Note: |-
        Markdown→HTML via goldmark, wiki link resolution, image rewriting
        Markdown-to-HTML via goldmark
    - Path: internal/server/agent_markdown.go
      Note: Markdown mirror endpoints (/AGENTS.md, /llms.txt, /sitemap.xml)
    - Path: internal/server/runtime.go
      Note: RuntimeState with vault/search snapshots, atomic reload
    - Path: internal/server/server.go
      Note: |-
        Go server route registration, SSR proxy, SPA fallback
        Go server route registration
    - Path: internal/vault/vault.go
      Note: |-
        Note struct (title, html, backlinks, wikiLinks), LoadAll, rebuildHTML
        Note struct
    - Path: web/index.html
      Note: Static HTML shell with <title> fallback
    - Path: web/server.mjs
      Note: |-
        Express SSR sidecar, title construction, meta injection, noscript
        Express SSR sidecar
    - Path: web/src/App.tsx
      Note: |-
        Router, document.title effect, VaultLayout composition
        Router
    - Path: web/src/components/pages/NotePage/NotePage.tsx
      Note: |-
        Note content rendering, right panel, backlinks
        Note rendering
    - Path: web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: |-
        Menu bar, sidebar toggle, desktop/mobile responsive layout
        Menu bar
    - Path: web/src/entry-client.tsx
      Note: |-
        Client hydration, SSR content clearing, store preloading
        Client hydration
    - Path: web/src/entry-server.tsx
      Note: |-
        SSR renderApp(), SSRNotePage, SSRHomePage, preloadCache
        SSR renderApp
    - Path: web/src/store/store.ts
      Note: |-
        Redux store factory (SSR-per-request vs browser singleton)
        Redux store factory
    - Path: web/src/store/vaultApi.ts
      Note: |-
        RTK Query API slice, SiteConfig type, all endpoint definitions
        RTK Query API slice
ExternalSources: []
Summary: Complete analysis and implementation guide for fixing page titles, adding SSR sidebar navigation for SEO, and improving HTML semantic structure. Written for interns joining the Retro Obsidian Publish project.
LastUpdated: 2026-06-07T12:00:00Z
WhatFor: Use when fixing page titles, improving SSR SEO, or onboarding new engineers to the Retro Obsidian Publish codebase.
WhenToUse: When SEO is hurting, page titles show wrong text, or when adding navigation to SSR output.
---


# Page title, SSR sidebar, and HTML layout design and implementation guide

## Executive summary

Retro Obsidian Publish is a self-hosted, read-only Obsidian vault viewer with a Go backend, a React SPA frontend, and a Node.js SSR sidecar. The system renders vault notes as both server-side rendered HTML (for SEO and crawler readability) and as an interactive single-page application (for human visitors with JavaScript enabled).

Three concrete issues have been identified on the production deployment at `parc.yolo.scapegoat.dev`:

1. **Page titles show "current" instead of the note or vault name.** The vault directory is symlinked as `current`, so the `pageTitle` configuration defaults to that symlink name. For note pages, the SSR renders the correct note title, but the React client overwrites it.
2. **SSR output does not include sidebar navigation.** The SSR-rendered HTML is content-only — it renders the note body, title, tags, and backlinks, but not the sidebar file tree. This is intentional for performance but means crawlers without full JavaScript execution miss the navigation context.
3. **HTML structure could be more semantic.** The SSR output uses `<div>` containers without proper `<main>`, `<article>`, or `<nav>` elements, which reduces accessibility and structured data quality.

This document explains the full architecture of the system — from the Go server through the SSR sidecar to the React SPA — so an intern can understand every component, identify the root causes of these issues, and implement fixes.

## Problem statement and scope

### Requested outcomes

1. **Fix page titles**: Browser tab and document title should show meaningful text — the vault name for the home page and "Note Title — Vault Name" for note pages.
2. **Improve SSR for SEO**: Add sidebar navigation breadcrumbs to the SSR output so crawlers can discover and understand the vault structure without executing JavaScript.
3. **Improve HTML semantic structure**: Use proper `<main>`, `<article>`, and `<nav>` elements in the SSR output.

### In scope

- The Go server's SSR proxy and SPA handler
- The Node.js SSR sidecar's HTML generation
- The React app's title management (`document.title`)
- Deployment configuration (devctl profiles, k8s env vars)
- SEO meta tags (og:title, description, JSON-LD)

### Out of scope

- Rewriting the entire SPA to support full server-side hydration
- Adding client-side search to SSR output
- Changing the goldmark Markdown-to-HTML pipeline
- CDN integration for assets
- Per-note custom title templates

## Current-state architecture

### High-level overview

```
┌─────────────────────────────────────────────────────────┐
│                     Internet / Crawler                  │
└────────────────────────┬────────────────────────────────┘
                         │ HTTP request
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Go Server (retro-obsidian-publish)         │
│                                                         │
│  /api/*          → API endpoints (notes, config, tree)  │
│  /vault-assets/* → serve vault images                   │
│  /* (page)       → SSR proxy OR SPA fallback            │
│  *.md            → Markdown mirror endpoints            │
└────────┬────────────────────────────────────────────────┘
         │ HTTP reverse proxy
         ▼
┌─────────────────────────────────────────────────────────┐
│           Node.js SSR Sidecar (Express)                 │
│                                                         │
│  1. Pre-fetch data from Go API                          │
│  2. Render React → HTML string                          │
│  3. Inject meta tags, title, noscript                   │
│  4. Return complete HTML                                │
└─────────────────────────────────────────────────────────┘
```

For non-note URLs (AGENTS.md, /llms.txt, /sitemap.xml, /note/*.md), the Go server handles them directly without SSR. For note URLs and the home page, requests go through the SSR sidecar when `--ssr-url` is configured.

### Component-by-component breakdown

#### 1. Go Server (`internal/server/server.go`)

The Go server is the main HTTP entry point. It performs these responsibilities:

- **Vault loading**: `RuntimeState` loads the vault from disk (resolving symlinks), builds a search index, and provides atomic snapshots for concurrent requests.
- **API routing**: Mounts `/api/config`, `/api/notes`, `/api/notes/{slug}`, `/api/tree`, `/api/search`, `/api/tags` via the API handler.
- **Asset serving**: Serves vault images at `/vault-assets/{path}`.
- **Markdown mirrors**: Serves `/AGENTS.md`, `/llms.txt`, `/sitemap.xml`, `/sitemap.md`, `/index.md`, `/note/{slug}.md` as raw Markdown for agent readability.
- **SSR proxy**: When `SSRURL` is configured, proxies page requests to the Node.js SSR sidecar. Falls back to the SPA handler on failure.
- **SPA fallback**: Serves the bundled React SPA from `index.html` for client-side navigation.

**Key code (route registration order)**:

```go
// server.go: Run()
r := mux.NewRouter()
h := api.NewWithProvider(state, api.PublicConfig{VaultName: vaultName, PageTitle: pageTitle})
h.Register(r)  // mounts /api/*

r.HandleFunc("/api/healthz", healthHandler(state)).Methods("GET")
r.PathPrefix("/vault-assets/").Handler(assetHandler(state)).Methods("GET", "HEAD")
r.HandleFunc("/api/admin/reload", reloadHandler(...)).Methods("POST")

if cfg.ServeWeb {
    spaHandler := web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"})

    if cfg.SSRURL != "" {
        // Static assets served directly from Go
        r.PathPrefix("/assets/").Handler(spaHandler)
        r.PathPrefix("/__manus__/").Handler(spaHandler)
        r.PathPrefix("/fonts/").Handler(spaHandler)
        r.HandleFunc("/favicon.ico", ...)
        r.HandleFunc("/favicon.svg", ...)

        // Page requests go to SSR sidecar
        ssrProxy := newSSRProxy(cfg.SSRURL, spaHandler)
        pageHandler := newAgentPageHandler(state, apiConfig, ssrProxy)
        r.PathPrefix("/").Handler(pageHandler)
    } else {
        pageHandler := newAgentPageHandler(state, apiConfig, spaHandler)
        r.PathPrefix("/").Handler(pageHandler)
    }
}
```

The **order matters**: asset routes and API routes are registered before the catch-all `PathPrefix("/")` so they aren't intercepted by the SPA handler.

#### 2. API Handler (`internal/api/api.go`)

The API handler exposes JSON endpoints. The most relevant for this ticket is `/api/config`:

```go
// PublicConfig is the operator-configured public site metadata
type PublicConfig struct {
    VaultName string `json:"vaultName"`
    PageTitle string `json:"pageTitle"`
}

// SiteConfig is what the frontend sees
type SiteConfig struct {
    VaultName string `json:"vaultName"`
    PageTitle string `json:"pageTitle"`
    Notes     int    `json:"notes"`
}

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
    v, _ := h.provider.Snapshot()
    jsonResponse(w, SiteConfig{
        VaultName: h.config.VaultName,
        PageTitle: h.config.PageTitle,
        Notes:     len(v.AllNotes()),
    })
}
```

**Production observation**: `/api/config` returns `{"vaultName":"current","pageTitle":"current","notes":706}` because the vault directory symlink is named "current".

#### 3. Vault and Note Loading (`internal/vault/vault.go`)

The vault package loads all `.md` files from the vault directory, parses each one, and builds indexes:

```go
func (v *Vault) LoadAll() error {
    // Walk the vault directory
    filepath.Walk(v.root, func(path string, info os.FileInfo, err error) error {
        // For each .md file:
        note, err := v.loadNote(path, info)
        v.notes[note.Slug] = note
    })
    v.buildWikiLinkIndex()   // Resolve [[wiki links]]
    v.buildBacklinks()       // Find which notes reference each note
    v.rebuildHTML()          // Post-process: resolve wiki links + image paths
    return nil
}
```

Each `Note` struct contains:
- `Title`: from frontmatter or first H1 heading
- `HTML`: goldmark-rendered HTML with wiki links resolved
- `Backlinks`: array of slugs that link to this note
- `WikiLinks`: array of wiki links found in the note
- `Tags`, `Excerpt`, `Frontmatter`, `ModTime`

The `rebuildHTML` method calls `parser.ReplaceWikiLinksString` and `parser.RewriteImageSources` to post-process the HTML.

#### 4. Markdown Parser (`internal/parser/parser.go`)

The parser uses `goldmark` with GFM extensions and custom processing:

```go
func Parse(src []byte) (*ParsedNote, error) {
    // 1. Extract wiki links before goldmark sees them
    wikiLinks := extractWikiLinks(src)  // finds [[Target]] patterns

    // 2. Replace [[wiki links]] with anchor HTML placeholders
    processed := replaceWikiLinks(src)  // replaces [[Target]] with <a href="/note/...">

    // 3. Parse with goldmark
    md := goldmark.New(
        goldmark.WithExtensions(meta.Meta, extension.GFM, ...),
    )
    md.Convert(processed, &buf)

    // 4. Post-process rendered HTML
    htmlOut = renderCallouts(htmlOut)  // > [!note] Title → styled divs

    return &ParsedNote{HTML: htmlOut, Title: title, Tags: tags, ...}
}
```

The wiki link regex `(!?)\[\[([^\[\]]+)\]\]` matches both `[[Target]]` and `![[embed]]`. These are converted to HTML anchor placeholders before goldmark processes the Markdown, ensuring goldmark doesn't mangle them.

#### 5. Node.js SSR Sidecar (`web/server.mjs`)

The SSR sidecar is an Express server that:

1. Receives page requests from the Go server's reverse proxy
2. Pre-fetches data from the Go API (config, notes list, file tree, note content)
3. Calls `renderApp()` from the React SSR entry to produce an HTML string
4. Injects meta tags, page title, noscript fallback, and JSON-LD structured data
5. Returns the complete HTML page

**Title construction** (server.mjs, ~line 120):

```javascript
const vaultName = config?.vaultName || "Vault";
const title = note?.title
  ? `${note.title} — ${vaultName}`
  : `${config?.pageTitle || vaultName}`;
```

For a note page, this produces: `"Research Institute Guidelines — current"`.
For the home page, this produces: `"current"`.

**HTML assembly** (server.mjs, ~line 130):

```javascript
// Inject preloaded state into the SPA index.html shell
htmlPage = htmlPage.replace(
    "<div id=\"root\">",
    `<div id="root">${html}</div>`
);

// Add noscript fallback
htmlPage = htmlPage.replace("</body>", `<noscript>...list of note links...</noscript>\n</body>`);

// Inject meta tags into <head>
htmlPage = htmlPage.replace("</head>", `
    <script>window.__PRELOADED_STATE__=${serializedPreloadedState};</script>
    <meta name="description" content="${description}" />
    <meta property="og:title" content="${title}" />
    <meta property="og:description" content="${description}" />
    <link rel="canonical" href="${BASE_URL}${canonicalPath}" />
    <link rel="alternate" type="text/markdown" href="${markdownPath}" />
    <script type="application/ld+json">${JSON.stringify(jsonLd)}</script>
    <script type="application/ld+json">${JSON.stringify(breadcrumbLd)}</script>
    </head>`
);

// Update the page title
htmlPage = htmlPage.replace(
    /<title>.*?<\/title>/,
    `<title>${title}</title>`
);
```

The `getIndexHtml()` function reads `./dist/index.html`, which has a hardcoded `<title>Retro Obsidian Publish</title>`. The SSR sidecar replaces it with the dynamic title.

#### 6. React SSR Entry (`web/src/entry-server.tsx`)

The SSR entry point defines simplified page components for server-side rendering:

```tsx
// SSRNotePage — renders note title, tags, HTML body, backlinks
function SSRNotePage({ note }: { note: Note }) {
  return React.createElement("div", { className: "ssr-note" }, [
    React.createElement("h1", { className: "text-xl font-bold" }, note.title),
    // tags...
    React.createElement("div", { className: "note-prose", dangerouslySetInnerHTML: { __html: note.html } }),
    // backlinks...
  ]);
}
```

These components use `React.createElement` directly (not JSX) to avoid JSX compiler dependencies. They render only the essential content — no sidebar, no navigation tree, no interactive features.

**Route parsing**:

```tsx
export function parseRoute(url: string): ParsedRoute {
  const pathname = url.split("#")[0]?.split("?")[0] || "/";
  if (pathname === "/search") return { type: "search" };
  if (pathname.startsWith("/note/")) {
    return { type: "note", slug: decodeURIComponent(pathname.replace(/^\/note\//, "")) };
  }
  if (pathname === "/") return { type: "home" };
  return { type: "unknown" };
}
```

**Preloading RTK Query cache**:

```tsx
async function preloadCache(store, data, slug?) {
  if (data.config) store.dispatch(vaultApi.util.upsertQueryData("getConfig", undefined, data.config));
  if (data.notes) store.dispatch(vaultApi.util.upsertQueryData("listNotes", undefined, data.notes));
  if (data.tree) store.dispatch(vaultApi.util.upsertQueryData("getTree", undefined, data.tree));
  if (data.note && slug) store.dispatch(vaultApi.util.upsertQueryData("getNote", slug, data.note));
}
```

This populates RTK Query's cache before `renderToString()`, so components that read from the cache get real data during SSR.

#### 7. React Client Entry (`web/src/entry-client.tsx`)

The client entry mounts the full interactive React app:

```tsx
const preloadedState = window.__PRELOADED_STATE__;
delete window.__PRELOADED_STATE__;
const store = makeStore(preloadedState);
const root = document.getElementById("root")!;

// CRITICAL: Clear SSR content before mounting
root.textContent = "";

createRoot(root).render(
  <React.StrictMode>
    <Provider store={store}>
      <App />
    </Provider>
  </React.StrictMode>
);
```

The comment explains why: "We don't use `hydrateRoot()` because the SSR components are simplified versions that don't match the full client component tree. Using `hydrateRoot` with mismatched DOM causes React error #418."

This clearing means the client app never sees the server-rendered title — it has to set the title itself.

#### 8. React App Router (`web/src/App.tsx`)

```tsx
function Router() {
  const { data: config } = useGetConfigQuery();

  useEffect(() => {
    document.title = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
  }, [config?.pageTitle, config?.vaultName]);

  return (
    <VaultLayout vaultName={config?.vaultName}>
      <Switch>
        <Route path="/" component={HomeRedirect} />
        <Route path="/note/*" component={NoteRoute} />
        <Route path="/search" component={SearchRoute} />
        <Route component={NotFoundPage} />
      </Switch>
    </VaultLayout>
  );
}
```

**The problem**: This `useEffect` sets `document.title` to `config.pageTitle` or `config.vaultName` — never including the current note's title. So after the client hydrates, all pages show just "current" regardless of which note is being viewed.

#### 9. Vault Layout (`web/src/components/pages/VaultLayout/VaultLayout.tsx`)

The layout renders:
- A menu bar (menubar) with sidebar toggle, vault name, search button, right panel toggle, and clock
- A sidebar (desktop: resizable panel; mobile: off-canvas drawer)
- A content area with scrollable note rendering

```tsx
return (
  <div className="flex flex-col h-screen overflow-hidden bg-[var(--color-paper)]">
    {/* Menu Bar */}
    <header className="retro-menubar shrink-0 z-50">
      <button onClick={() => dispatch(toggleSidebar())}><Icon name="menu" /></button>
      <button onClick={() => navigate("/")}>&#9670; {vaultName}</button>
      <button onClick={() => navigate("/search")}>Search</button>
      {/* right panel toggle, clock */}
    </header>

    {/* Sidebar (desktop) or drawer (mobile) */}
    <ResizablePanelGroup direction="horizontal">
      <ResizablePanel>
        <Sidebar tree={tree} onSelectNote={handleNavigate} />
      </ResizablePanel>
      <ResizablePanel>
        <main className="h-full overflow-y-auto retro-scroll">
          {children}
        </main>
      </ResizablePanel>
    </ResizablePanelGroup>
  </div>
);
```

#### 10. Note Page (`web/src/components/pages/NotePage/NotePage.tsx`)

Renders the note content, backlinks, and an optional right panel. Uses `ResizablePanelGroup` for the right panel layout on desktop, and inline backlinks below the content on mobile.

### Request flow diagram

```
Client → GET /note/my-note
  │
  ├─ Go server: is this a page request? → yes → SSR proxy
  │   │
  │   ├─ Go server reverse-proxies to SSR sidecar
  │   │   │
  │   │   ├─ SSR sidecar pre-fetches from Go API:
  │   │   │   GET /api/config     → {vaultName:"current", pageTitle:"current"}
  │   │   │   GET /api/notes      → [706 notes list]
  │   │   │   GET /api/tree       → {root: {...}}
  │   │   │   GET /api/notes/my-note → {title:"My Note", html:"...", backlinks:[...]}
  │   │   │
  │   │   ├─ SSR sidecar calls renderApp("/note/my-note", {config, notes, tree, note})
  │   │   │   → SSRNotePage renders note title + HTML + backlinks
  │   │   │   → renderToString() returns HTML string
  │   │   │
  │   │   ├─ SSR sidecar injects:
  │   │   │   <title>My Note — current</title>
  │   │   │   <meta name="description" ...>
  │   │   │   <meta property="og:title" content="My Note — current" />
  │   │   │   JSON-LD structured data
  │   │   │   noscript block with note links
  │   │   │
  │   │   └─ SSR sidecar sends complete HTML back to Go server
  │   │       → Go server returns HTML to client
  │   │
  └─ Client receives HTML, executes JS
      ├─ React client mounts, clears SSR content
      ├─ Router useEffect: document.title = "current" (overwrites SSR title!)
      ├─ Full SPA with sidebar, interactive note renderer, backlinks panel loads
      └─ (document.title is STILL wrong — only uses config values)
```

### Critical insight: the dual-title problem

```
                         ┌─────────────────────────────────┐
                         │       SSR output (correct)       │
                         │  <title>My Note — current</title>│
                         │  <meta name="description" ...>   │
                         │  <noscript>note links</noscript> │
                         └──────────────┬──────────────────┘
                                        │
                         ┌──────────────▼──────────────────┐
                         │     Client hydration (wrong)     │
                         │  useEffect:                     │
                         │  document.title = config.pageTitle│
                         │  → "current" (note title lost!) │
                         └─────────────────────────────────┘
```

The SSR correctly includes the note title, but the React client's `useEffect` in `Router()` overwrites it with only the config's `pageTitle` value.

## Gap analysis

### Gap 1: `pageTitle` falls back to vault directory name

**Root cause**: In `server.go:Run()`, if no `--page-title` flag is provided:

```go
pageTitle := cfg.PageTitle
if pageTitle == "" {
    pageTitle = vaultName  // vaultName = filepath.Base(vaultDir) = "current"
}
```

And in `api.go:NewWithProvider`:
```go
if config.PageTitle == "" {
    config.PageTitle = config.VaultName
}
```

There is no way for a deployment to set a custom page title without modifying the source or rebuilding the frontend.

**Impact**: All page titles show the vault directory symlink name ("current").

### Gap 2: React client `document.title` ignores the current note

**Root cause**: In `App.tsx:Router()`:

```tsx
useEffect(() => {
  document.title = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
}, [config?.pageTitle, config?.vaultName]);
```

This only reads from `config`. It never considers which note is currently displayed. The `useGetConfigQuery` hook provides site-level metadata, not page-level metadata.

**Impact**: Even if `pageTitle` were set correctly in deployment, the browser tab would always show the same title regardless of which note is viewed.

### Gap 3: SSR output lacks sidebar navigation

**Root cause**: `SSRNotePage` intentionally renders only note content:

```tsx
function SSRNotePage({ note }) {
  return <div class="ssr-note">
    <h1>{note.title}</h1>
    <div class="note-prose" dangerouslySetInnerHTML={{__html: note.html}} />
    {/* backlinks */}
  </div>;
}
```

There is no sidebar component in SSR. The sidebar is only rendered by the client-side `VaultLayout`.

**Impact**: Crawlers that execute JavaScript see the sidebar via the React SPA, but crawlers that don't execute JavaScript (or that read the raw HTML response) see only note content without navigation context.

### Gap 4: HTML lacks semantic elements

The SSR output uses `<div>` containers throughout. There are no `<main>`, `<article>`, `<nav>`, or `<header>` elements.

**Impact**: Reduced accessibility, no landmark landmarks for screen readers, weaker structured data signals for search engines.

## Proposed solutions

### Solution 1: Configuration-driven page title

**Changes needed:**
1. The devctl profile should set `PAGE_TITLE` environment variable
2. The k8s deployment should set the same env var
3. The Go server should pass this through to the API config

**Example devctl profile update:**
```yaml
profiles:
  go-go-parc:
    env:
      VAULT_DIR: /home/manuel/code/wesen/go-go-golems/go-go-parc
      VAULT_NAME: go-go-parc
      PAGE_TITLE: PARC  # ← adds meaningful page title
```

**Go server changes:**
- Add `PageTitle` to `server.Config`
- Pass through to `api.PublicConfig`

**Server.mjs changes:** None needed — it already reads `config.pageTitle` for the title construction.

### Solution 2: Note-aware document title in React

**Changes needed in `App.tsx`:**

```tsx
function Router() {
  const { data: config } = useGetConfigQuery();
  const location = useLocation();

  useEffect(() => {
    let title = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
    // If we're on a note page, prepend the note title
    if (location[0].startsWith("/note/")) {
      const slug = location[0].replace(/^\/note\//, "");
      const noteQuery = useGetNoteQuery(slug);
      if (noteQuery.data) {
        title = `${noteQuery.data.title} — ${title}`;
      }
    }
    document.title = title;
  }, [config?.pageTitle, config?.vaultName, location[0]]);

  return <VaultLayout vaultName={config?.vaultName}>...</VaultLayout>;
}
```

Or more cleanly, move the title logic into the `NotePage` component so it's scoped to the route that needs it.

### Solution 3: Add sidebar navigation to SSR

**Option A (simple): Breadcrumb navigation**
Add a visible breadcrumb bar to the SSR output that links to parent notes.

```html
<nav class="ssr-breadcrumb">
  <a href="/">Home</a> / 
  <a href="/note/research/institute/guidelines">Guidelines</a> / 
  Research Institute Guidelines
</nav>
```

**Option B (full): File tree navigation**
Include a mini file tree in the noscript block or as a visible nav element.

**Option C (structured): Enhanced noscript**
Enhance the existing noscript block with better structured navigation — this is the lowest-risk approach since it only modifies what's already rendered for non-JS users.

### Solution 4: Semantic HTML elements

**Changes in `entry-server.tsx`:**
```tsx
// SSRNotePage with semantic elements
function SSRNotePage({ note }) {
  return React.createElement("article", { className: "ssr-note" }, [
    React.createElement("header", null, [
      React.createElement("h1", ...),
      // tags...
    ]),
    React.createElement("div", { className: "note-prose", ... }),
    React.createElement("nav", { className: "ssr-backlinks" }, [
      // backlinks...
    ]),
  ]);
}
```

**Changes in `server.mjs`:**
Wrap the rendered content in `<main>` tags in the final HTML.

## Implementation plan

### Phase 1: Fix deployment pageTitle (5 minutes)

**File**: `publish-vault/.devctl.yaml`

```yaml
profiles:
  go-go-parc:
    env:
      VAULT_DIR: /home/manuel/code/wesen/go-go-golems/go-go-parc
      VAULT_NAME: go-go-parc
      PAGE_TITLE: PARC
```

**File**: k8s deployment config (in gitops repo `wesen/2026-03-27--hetzner-k3s`)

Add `PAGE_TITLE: PARC` to the Go server container's env vars.

**Validation**:
```bash
curl https://parc.yolo.scapegoat.dev/api/config
# Expected: {"vaultName":"current","pageTitle":"PARC","notes":706}
```

### Phase 2: Fix React client document.title (30 minutes)

**File**: `web/src/App.tsx` — `Router` component

Replace the current `useEffect`:
```tsx
useEffect(() => {
  document.title = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
}, [config?.pageTitle, config?.vaultName]);
```

With logic that:
1. Starts with the config's `pageTitle` or `vaultName`
2. Appends the current note's title if on a `/note/*` route
3. Delegates note-title fetching to the route's component

**Refactored approach**: Move the document.title logic into the `NotePage` component and keep the Router's effect for non-note pages.

### Phase 3: Add breadcrumb navigation to SSR (30 minutes)

**File**: `web/src/entry-server.tsx` — `SSRNotePage`

Add a breadcrumb element before the note title:
```tsx
function SSRNotePage({ note }) {
  const breadcrumbParts = note.slug.split('/').map((part, i, arr) => {
    const partialSlug = arr.slice(0, i + 1).join('/');
    return React.createElement("span", null, [
      i > 0 ? React.createElement("span", null, " / ") : null,
      React.createElement("a", { href: `/note/${partialSlug}`, className: "wiki-link" }, part),
    ]);
  });
  return React.createElement("div", { className: "ssr-note" }, [
    React.createElement("nav", { className: "ssr-breadcrumb" }, breadcrumbParts),
    React.createElement("h1", { className: "text-xl font-bold" }, note.title),
    // ... rest of content
  ]);
}
```

### Phase 4: Semantic HTML (20 minutes)

**File**: `web/src/entry-server.tsx`

Wrap `SSRNotePage` with semantic elements:
```tsx
function SSRNotePage({ note }) {
  return React.createElement("article", { className: "ssr-note" }, [
    React.createElement("header", { className: "ssr-note-header" }, [/* title, tags */]),
    React.createElement("section", { className: "note-prose", ... }),
    React.createElement("nav", { className: "ssr-backlinks" }, [/* backlinks */]),
  ]);
}
```

**File**: `server.mjs`

Wrap the injected content in `<main>`:
```javascript
htmlPage = htmlPage.replace(
    "<div id=\"root\">",
    `<main><div id="root">${html}</div></main>`
);
```

### Phase 5: Testing and validation

1. Verify `/api/config` returns the correct `pageTitle`
2. Verify page titles in SSR output: `curl -s https://parc.yolo.scapegoat.dev/note/slug | grep '<title>'`
3. Verify meta tags are present
4. Verify React client sets correct document.title
5. Verify noscript block has navigation
6. Run `docmgr doctor --ticket RETRO-SEO-009`
7. Upload to reMarkable

## Test strategy

### Manual validation commands

```bash
# 1. Check API config
curl -s https://parc.yolo.scapegoat.dev/api/config | jq

# 2. Check SSR page title
curl -s https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index | grep '<title>'

# 3. Check meta tags
curl -s https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index | grep 'meta name="description"\|meta property="og:'

# 4. Check JSON-LD
curl -s https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index | grep 'application/ld+json'

# 5. Check noscript block
curl -s https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index | grep -o '<noscript>[\s\S]*</noscript>' | head -5

# 6. Check semantic elements
curl -s https://parc.yolo.scapegoat.dev/note/research/institute/guidelines/guidelines-index | grep -oP '<(main|article|nav|header)[^>]*>'
```

### Expected results after fixes

```
# /api/config should show:
{"vaultName":"current","pageTitle":"PARC","notes":706}

# <title> should show:
<title>Research Institute Guidelines — PARC</title>

# meta og:title should show:
<meta property="og:title" content="Research Institute Guidelines — PARC" />

# noscript should include navigation:
<noscript><nav class="ssr-breadcrumb"><a href="/">Home</a> / ...</nav>...</noscript>
```

## Risks, alternatives, and open questions

### Risks

1. **Breaking existing deployments**: Changing the default behavior of `pageTitle` could affect existing sites that rely on the directory-name fallback. Mitigation: the fallback to `vaultName` remains; only when explicitly configured does `pageTitle` change.

2. **SSR performance impact**: Adding sidebar navigation to SSR increases the size of the server-rendered HTML and adds more RTK Query data fetching. The file tree is already fetched during SSR, so this is a minor addition (just rendering).

3. **React hydration mismatch**: If the SSR components become more complex, ensuring they match the client-side component tree for proper hydration becomes harder. Current approach of clearing SSR content and using `createRoot` avoids this, but adds a brief flash of the SSR content before the SPA renders.

### Alternatives considered

1. **Full SSR hydration (hydrateRoot)**: Would require replacing Wouter (which has no StaticRouter) with a router that supports SSR, or building a StaticRouter shim. Too complex for this ticket.

2. **Frontend-only title fix**: Changing only the React client wouldn't help crawlers that read the raw HTML. The SSR title must be correct independently.

3. **No SSR sidebar**: The current approach of content-only SSR is sufficient for basic SEO. The Googlebot and most modern crawlers execute JavaScript and will see the full sidebar. Adding sidebar navigation to SSR is a "nice to have" for less-capable crawlers and for noscript users.

4. **Server-side title from note title alone**: Instead of "Note — Vault", just showing the note title. This loses the vault context in the tab but is simpler. The current "Note — Vault" format is preferred for multi-vault deployments.

## References

- `internal/server/server.go:30-90`: Config struct, Run(), route registration, SSR proxy setup
- `internal/server/server.go:130-160`: `newSSRProxy()` — reverse proxy with SPA fallback
- `internal/api/api.go:36-52`: PublicConfig, SiteConfig, /api/config handler
- `internal/vault/vault.go:74-101`: LoadAll, rebuildHTML, ResolveAssetURL
- `internal/parser/parser.go:42-75`: Parse(), wiki link extraction and replacement
- `web/server.mjs:100-160`: Title construction, meta injection, noscript generation
- `web/src/entry-server.tsx:60-150`: SSRNotePage, SSRHomePage, parseRoute, renderApp
- `web/src/entry-client.tsx`: Client hydration, SSR content clearing
- `web/src/App.tsx:10-30`: Router, document.title effect
- `web/src/components/pages/VaultLayout/VaultLayout.tsx`: Menu bar, sidebar, responsive layout
- `web/src/components/pages/NotePage/NotePage.tsx`: Note rendering, backlinks, right panel
- `web/src/store/vaultApi.ts`: RTK Query endpoints, SiteConfig type
- `web/src/store/store.ts`: Redux store factory (SSR-per-request pattern)
- `web/ssr.Dockerfile`: SSR sidecar container build
