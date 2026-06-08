---
title: "Favicon per-site support design and implementation guide"
doc_type: design-doc
status: active
intent: long-term
topics:
  - retro-obsidian-publish
  - assets
  - config
  - html-layout
  - obsidian-vault
ticket: FAVICON-001
created: 2026-06-08
---

# Favicon per-site support design and implementation guide

## 1. Executive summary

Retro Obsidian Publish currently serves the root SPA page (`index.html`) for every request that doesn't match a known route — including `/favicon.ico`. This means browsers and agents requesting a favicon receive a full HTML page instead of an image, wasting bandwidth and generating noise in server logs. This document describes how to add per-site favicon support so each vault deployment can have its own `favicon.ico` (and optionally `favicon.svg`), with a clear 404 fallback when no favicon is configured.

The change is small and well-scoped: intercept `/favicon.ico` and `/favicon.svg` before the catch-all SPA handler, look for a favicon file in the vault root (or a configurable override path), serve it with proper MIME types and caching headers, and return a clean 404 when no file exists. We will also create a favicon for the go-go-parc vault (`go-go-parc` is Manuel's golem research park Obsidian vault).

---

## 2. Problem statement and scope

**Problem:** When a browser loads any Retro Obsidian Publish site, it automatically requests `/favicon.ico`. The server's router has no specific handler for this path, so the request falls through to the SPA catch-all (`PathPrefix("/")`), which responds with `index.html` — a full HTML document. This causes:

- Browser console errors (expecting an image, getting HTML).
- Wasted bandwidth on every page load.
- Confusing entries in access logs.
- No way to brand each vault with its own icon.

**Scope:**

- Add a favicon handler to the Go backend router.
- Support per-vault favicons by looking in the vault root directory.
- Optionally support a CLI flag / env var for an explicit favicon path override.
- Return proper HTTP 404 with `Content-Type: text/plain` when no favicon exists (instead of serving HTML).
- Create a concrete `favicon.ico` (and `.svg`) for the go-go-parc vault.
- Inject a `<link rel="icon">` tag in the HTML `<head>` via the SSR pipeline and the SPA `index.html`.

**Out of scope:**

- Favicon generation tooling (we'll create the go-go-parc favicon by hand / with an image tool).
- Apple touch icons, PWA manifests, or other advanced favicon variants (can be added later).
- Dynamic favicon generation from vault metadata.

---

## 3. Current-state architecture

### 3.1 How HTTP routing works today

The server is built on `github.com/gorilla/mux` and runs inside `internal/server/server.go`. The `Run()` function wires up routes in this order:

```
/api/*          → api.Handler (JSON endpoints)
/api/healthz    → health check handler
/vault-assets/* → vault static file handler (serves images etc from vault root)
/api/admin/reload → admin reload endpoint
/assets/*       → SPA static assets (only when SSR is enabled)
/__manus__/*    → SPA static assets (only when SSR is enabled)
/fonts/*        → SPA static assets (only when SSR is enabled)
/favicon.ico    → SPA handler (only when SSR is enabled) ← BUG: serves index.html!
/favicon.svg    → SPA handler (only when SSR is enabled) ← BUG: serves index.html!
/*              → agent page handler → SPA handler (catch-all)
```

**Key observation:** When SSR is enabled, there *are* explicit `/favicon.ico` and `/favicon.svg` routes, but they just forward to `spaHandler` which serves `index.html` because no actual favicon file exists in the embedded web bundle (`internal/web/embed/public/` is empty). When SSR is disabled, there is no favicon route at all — it falls through to the catch-all.

The relevant code in `internal/server/server.go` (around line 95):

```go
if cfg.ServeWeb {
    spaHandler := web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"})

    if cfg.SSRURL != "" {
        // ... static asset routes ...
        r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
            spaHandler.ServeHTTP(w, r)
        })
        r.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
            spaHandler.ServeHTTP(w, r)
        })
        // ... SSR proxy catch-all ...
    } else {
        // No favicon handler at all — falls through to catch-all
        pageHandler := newAgentPageHandler(...)
        r.PathPrefix("/").Handler(pageHandler)
    }
}
```

### 3.2 The SPA handler

The SPA handler (`internal/web/static.go`) uses an `fs.FS` (either embedded at build time via `//go:embed` or read from disk in dev mode). Its logic is simple:

1. If the path starts with `/api`, return 404.
2. If the path matches a real file in the `fs.FS`, serve it.
3. Otherwise, serve `index.html`.

Because there is no `favicon.ico` or `favicon.svg` file in the embedded FS, step 3 always fires — returning the full HTML page.

### 3.3 Vault file serving

The `/vault-assets/*` handler (`assetHandler` in `server.go`) already serves files from the vault root directory safely. It:

- Strips the `/vault-assets/` prefix.
- Validates the path (no `..`, no hidden files, no `.md` files).
- Opens via `os.OpenRoot(state.ResolvedRoot())`.
- Serves with `http.ServeContent` and `Cache-Control: public, max-age=300`.

This pattern is exactly what we need for favicon serving, except favicons live at the vault root rather than under a subpath.

### 3.4 How sites are deployed

Each Retro Obsidian Publish deployment is configured via:

- **CLI flags:** `--vault`, `--vault-name`, `--page-title`, `--ssr-url`, etc. (see `cmd/retro-obsidian-publish/commands/serve/serve.go`).
- **Docker Compose:** mounts the vault directory as `/vault:ro` and passes environment variables.
- **Kubernetes (k3s):** uses git-sync sidecar to pull vault content and the reload endpoint to refresh.

There is currently no per-site "assets directory" concept — only the vault itself and the embedded web bundle. Favicons should naturally live alongside the vault content.

### 3.5 HTML head

The `web/index.html` file is minimal:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="..." />
    <title>Retro Obsidian Publish</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/entry-client.tsx"></script>
  </body>
</html>
```

There is no `<link rel="icon">` tag. The SSR sidecar generates its own HTML, so both paths need the favicon link injected.

---

## 4. Gap analysis

| Gap | Impact | Difficulty |
|-----|--------|------------|
| No `/favicon.ico` handler — serves HTML instead of 404 | Browser errors, wasted bandwidth | Easy |
| No per-vault favicon support | Every deployment looks the same | Easy |
| No `<link rel="icon">` in HTML | Browsers must guess `/favicon.ico` | Easy |
| No favicon for go-go-parc vault | No branding for the golem research park | Easy (design task) |

All gaps are easy to close. The changes are isolated to the server router and the HTML shell.

---

## 5. Proposed architecture and APIs

### 5.1 Favicon resolution order

When a request arrives for `/favicon.ico` or `/favicon.svg`:

1. **CLI override:** If `--favicon` flag is set, serve that file.
2. **Vault root:** Check if `favicon.ico` (or `favicon.svg`) exists in the vault root directory.
3. **Embedded default:** If a favicon exists in the embedded web bundle, serve it.
4. **404:** Return `404 Not Found` with `Content-Type: text/plain`.

```
Request: GET /favicon.ico
   │
   ├─ --favicon flag set? ──→ serve that file (with MIME detection)
   │
   ├─ vault root contains "favicon.ico"? ──→ serve from vault
   │
   ├─ embedded bundle contains "favicon.ico"? ──→ serve from bundle
   │
   └─ none found ──→ 404 Not Found (text/plain)
```

### 5.2 New CLI flag

Add a `--favicon` flag to the serve command:

```
--favicon string   Path to a favicon.ico (or .svg) file. Overrides vault-root lookup.
```

This gets stored in `server.Config.FaviconPath` and passed through to the favicon handler.

### 5.3 Router changes

Replace the current favicon forwarding with a dedicated handler:

```go
// In server.go Run(), inside the ServeWeb block, BEFORE the catch-all:
faviconHandler := newFaviconHandler(state, cfg.FaviconPath, spaHandler)
r.HandleFunc("/favicon.ico", faviconHandler)
r.HandleFunc("/favicon.svg", faviconHandler)
```

Both SSR and non-SSR modes get the same favicon handler.

### 5.4 HTML `<link>` injection

Add a `<link rel="icon">` to `web/index.html`:

```html
<link rel="icon" href="/favicon.ico" />
<link rel="icon" type="image/svg+xml" href="/favicon.svg" />
```

Browsers will request these URLs and get the favicon (or 404). The SVG link is listed second so browsers that support SVG will prefer it, while older browsers fall back to `.ico`.

### 5.5 API config extension

Optionally expose the favicon URL in `/api/config` so the frontend can reference it:

```json
{
  "vaultName": "go-go-parc",
  "pageTitle": "Golem Research Park",
  "notes": 42,
  "faviconUrl": "/favicon.ico"
}
```

This is a minor convenience — the frontend already knows to look at `/favicon.ico` from the HTML `<head>`.

---

## 6. Implementation details

### 6.1 The favicon handler function

Here is the complete handler in pseudocode:

```go
// newFaviconHandler returns an HTTP handler that serves favicon.ico or favicon.svg
// from (in order): CLI override path, vault root, embedded web bundle.
// Returns 404 text/plain if no favicon is found.
func newFaviconHandler(state *RuntimeState, faviconPath string, spaHandler http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Determine expected filename from URL path
        filename := path.Base(r.URL.Path) // "favicon.ico" or "favicon.svg"

        // 1. CLI override
        if faviconPath != "" {
            if serveFileIfExists(w, r, faviconPath) {
                return
            }
            log.Printf("warning: --favicon path %q does not exist, skipping", faviconPath)
        }

        // 2. Vault root lookup
        vaultFavicon := filepath.Join(state.ResolvedRoot(), filename)
        if serveFileIfExists(w, r, vaultFavicon) {
            return
        }

        // 3. Embedded web bundle (delegate to SPA handler's FS)
        //    The spaHandler would serve index.html for missing files,
        //    so we check the FS directly instead.
        //    If the file exists in the embedded FS, serve it.
        //    Otherwise, skip to 404.

        // 4. Not found — clean 404
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintln(w, "favicon not found")
    }
}

// serveFileIfExists serves a file if it exists and is readable.
// Returns true if served, false if not found.
func serveFileIfExists(w http.ResponseWriter, r *http.Request, absPath string) bool {
    info, err := os.Stat(absPath)
    if err != nil || info.IsDir() {
        return false
    }
    file, err := os.Open(absPath)
    if err != nil {
        return false
    }
    defer file.Close()
    w.Header().Set("Cache-Control", "public, max-age=3600")
    http.ServeContent(w, r, info.Name(), info.ModTime(), file)
    return true
}
```

### 6.2 MIME type handling

Go's `http.ServeContent` auto-detects MIME types from file extensions:

- `.ico` → `image/x-icon` (or `image/vnd.microsoft.icon`)
- `.svg` → `image/svg+xml`

No explicit MIME configuration is needed. However, if you want to be explicit:

```go
switch filepath.Ext(absPath) {
case ".ico":
    w.Header().Set("Content-Type", "image/x-icon")
case ".svg":
    w.Header().Set("Content-Type", "image/svg+xml")
}
```

### 6.3 Caching strategy

Favicons change rarely, so use aggressive caching:

- **Cache-Control:** `public, max-age=3600` (1 hour). This balances freshness (after a favicon update) with reducing requests.
- **ETag:** `http.ServeContent` generates ETags automatically from the file's ModTime and size. No manual work needed.

### 6.4 Security considerations

The favicon handler reads from the filesystem, so the same security rules as the `/vault-assets/` handler apply:

- **Never serve `.md` files** — only `.ico` and `.svg` are expected.
- **Validate the path** — the filename is hardcoded (`favicon.ico` or `favicon.svg`), so path traversal is not possible through the URL. For the `--favicon` CLI flag, the operator is trusted (they set the flag).
- **Read-only** — the vault is mounted read-only in Docker, and the handler never writes.

### 6.5 Config changes

Update `server.Config` to include the favicon path:

```go
type Config struct {
    // ... existing fields ...
    FaviconPath string // Optional: explicit path to favicon file
}
```

Update `cmd/retro-obsidian-publish/commands/serve/serve.go`:

```go
// In the Settings struct:
Favicon string `glazed:"favicon"`

// In the flags list:
fields.New("favicon", fields.TypeString,
    fields.WithDefault(""),
    fields.WithHelp("Path to a favicon file (favicon.ico or favicon.svg). "+
        "When set, overrides vault-root lookup. When empty, the server looks "+
        "for favicon.ico and favicon.svg in the vault root directory."),
),
```

---

## 7. go-go-parc favicon design

### 7.1 What is go-go-parc?

go-go-parc is Manuel's Obsidian vault at `~/code/wesen/go-go-golems/go-go-parc`, described as a "golem research park." It contains research notes, project documentation, and knowledge base articles related to the go-go-golems ecosystem.

The vault root contains:
- `Attachments/` — embedded images
- `Projects/` — project-specific notes
- `Research/` — research articles and investigations
- `Logs/` — activity logs
- `index.md` — vault home page
- Various top-level articles (e.g., "NPM Publishing for Go Go Golems Packages with Vault OIDC.md")

### 7.2 Favicon concept

Since go-go-parc is a "golem research park," the favicon should evoke:

- **A golem figure** — blocky, geometric, reminiscent of clay or stone.
- **A park/nature element** — perhaps a tree or leaf combined with a golem silhouette.
- **Retro aesthetic** — the site uses a retro macOS System 1 design language, so a pixel-art style favicon would be on-brand.

**Recommended approach:**

1. Create a 32×32 pixel-art golem head (simple square face with glowing eyes).
2. Export as both `.ico` (multi-resolution: 16×16, 32×32, 48×48) and `.svg`.
3. Place the files at the vault root: `~/code/wesen/go-go-golems/go-go-parc/favicon.ico` and `favicon.svg`.

For the SVG version, a simple geometric design:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <!-- Golem body -->
  <rect x="8" y="10" width="16" height="16" rx="2" fill="#8B7355"/>
  <!-- Eyes -->
  <rect x="11" y="14" width="4" height="4" fill="#4AF626"/>
  <rect x="17" y="14" width="4" height="4" fill="#4AF626"/>
  <!-- Head -->
  <rect x="6" y="4" width="20" height="8" rx="2" fill="#A0926B"/>
</svg>
```

This creates a simple brown golem face with glowing green eyes — fitting the retro/pixel aesthetic.

### 7.3 Generating the .ico file

Use ImageMagick to convert the SVG to a multi-resolution ICO:

```bash
# Install if needed
# brew install imagemagick

# Generate ICO from SVG
convert -background none favicon.svg \
  -define icon:auto-resize=16,32,48 \
  favicon.ico
```

Or use an online tool like https://realfavicongenerator.net/ for more polished results.

---

## 8. Implementation plan

### Phase 1: Backend favicon handler (Go)

**Files to modify:**

1. `internal/server/server.go` — Add `newFaviconHandler`, replace current favicon routes.
2. `internal/server/favicon.go` — New file: `newFaviconHandler`, `serveFileIfExists`.
3. `internal/server/favicon_test.go` — New file: tests for the favicon handler.
4. `cmd/retro-obsidian-publish/commands/serve/serve.go` — Add `--favicon` flag, pass to `Config`.

**Steps:**

1. Create `internal/server/favicon.go` with the handler function.
2. Write tests for all resolution paths (CLI override, vault root, 404).
3. Update `Config` struct to include `FaviconPath`.
4. Wire the handler in `Run()` for both SSR and non-SSR modes.
5. Add the `--favicon` flag to the serve command.
6. Run existing tests: `go test ./internal/server/... -count=1`.
7. Manual test: place a favicon.ico in a test vault and verify it's served.

### Phase 2: HTML favicon link

**Files to modify:**

1. `web/index.html` — Add `<link rel="icon">` tags.

**Steps:**

1. Add the link tags to the `<head>`.
2. Rebuild the web bundle (if applicable).
3. Verify in browser that the favicon tab icon appears.

### Phase 3: go-go-parc favicon

**Files to create:**

1. `~/code/wesen/go-go-golems/go-go-parc/favicon.svg` — SVG favicon.
2. `~/code/wesen/go-go-golems/go-go-parc/favicon.ico` — ICO favicon.

**Steps:**

1. Design the favicon SVG (pixel-art golem).
2. Convert to ICO.
3. Place in vault root.
4. Deploy and verify.

### Phase 4: Optional — API config exposure

**Files to modify:**

1. `internal/api/api.go` — Add `FaviconURL` to `SiteConfig`.

---

## 9. Testing strategy

### Unit tests

Create `internal/server/favicon_test.go`:

```go
func TestFaviconHandler_VaultRoot(t *testing.T) {
    // Setup: create temp dir with a favicon.ico
    // Create RuntimeState pointing to that dir
    // Request GET /favicon.ico
    // Assert: 200, Content-Type: image/x-icon, body matches file
}

func TestFaviconHandler_NotFound(t *testing.T) {
    // Setup: create temp dir WITHOUT a favicon
    // Request GET /favicon.ico
    // Assert: 404, Content-Type: text/plain
}

func TestFaviconHandler_CLIOverride(t *testing.T) {
    // Setup: create two temp dirs, put favicon in override dir
    // Pass override path via FaviconPath
    // Assert: serves from override, not vault root
}

func TestFaviconHandler_SVG(t *testing.T) {
    // Setup: vault root has favicon.svg but not favicon.ico
    // Request GET /favicon.svg
    // Assert: 200, Content-Type: image/svg+xml
}
```

### Integration tests

- Start the server with a test vault that contains a `favicon.ico`.
- `curl -I http://localhost:8080/favicon.ico` → verify 200 and correct Content-Type.
- `curl -I http://localhost:8080/favicon.ico` → verify Cache-Control header.
- Start the server with a test vault that has NO favicon.
- `curl -I http://localhost:8080/favicon.ico` → verify 404.
- `curl -I http://localhost:8080/favicon.ico` → verify body is NOT HTML.

---

## 10. Decision records

### Decision: Favicon resolution order

- **Context:** There are multiple places a favicon could live (CLI override, vault root, embedded bundle). We need a predictable lookup order.
- **Options considered:**
  1. CLI override only — requires flag for every deployment.
  2. Vault root only — simple but can't override without modifying the vault.
  3. CLI override → vault root → embedded → 404 (chosen).
- **Decision:** Option 3 — cascading lookup.
- **Rationale:** Operators can override without modifying the vault (important for read-only deployments). The vault root is the natural location for site-specific assets. The embedded bundle provides a sensible default. 404 is the honest fallback.
- **Consequences:** Slightly more complex handler logic, but well-tested. Each resolution level is a simple file existence check.
- **Status:** proposed

### Decision: 404 response body format

- **Context:** When no favicon exists, we need to return an error. Should it be HTML, JSON, or plain text?
- **Options considered:**
  1. Empty 404 response — minimal but unhelpful.
  2. JSON `{"error": "not found"}` — consistent with API endpoints.
  3. Plain text "favicon not found" (chosen).
- **Decision:** Option 3 — plain text.
- **Rationale:** Browsers don't care about the body of a 404 favicon response. Plain text is simple, doesn't confuse clients expecting HTML, and is easy to debug from curl.
- **Consequences:** None significant.
- **Status:** proposed

### Decision: Favicon file location in vault

- **Context:** Where in the vault should the favicon file live?
- **Options considered:**
  1. Vault root (`favicon.ico`) — simple, visible.
  2. A `.assets/` or `.site/` hidden directory — keeps vault clean.
  3. A dedicated `assets/` visible directory — structured but adds a non-Obsidian directory.
- **Decision:** Option 1 — vault root.
- **Rationale:** Obsidian ignores non-Markdown files in the vault root. A `favicon.ico` at the vault root is the most natural location, matching how static site generators work. No extra directories needed.
- **Consequences:** The favicon file appears in Obsidian's file explorer, but as a binary file it won't be confused with notes. Obsidian's file tree in our app already filters to `.md` files only (see `vault.go` line 75).
- **Status:** proposed

---

## 11. Risks and open questions

### Risks

1. **Cached stale favicons:** Browsers cache favicons aggressively. After updating a favicon, some browsers may serve the old one for days. Mitigation: use a cache-busting query parameter in the `<link>` tag (e.g., `/favicon.ico?v=2`) or accept the 1-hour `Cache-Control` as a reasonable tradeoff.

2. **Vault root reads on every request:** The handler calls `os.Stat` on each favicon request. For a small static file, this is negligible (microsecond-level). If it ever becomes a concern, cache the result in `RuntimeState` and invalidate on reload.

### Open questions

1. **Should we log a warning when `--favicon` points to a missing file?** — Yes, this helps operators catch typos. A single warning line at startup is sufficient.

2. **Should the favicon be served through the SSR proxy?** — No. Favicons are static binary files; the SSR sidecar renders HTML pages. Favicon requests should be handled directly by the Go server, same as `/assets/*` today.

---

## 12. References

### Key source files

| File | Purpose |
|------|---------|
| `internal/server/server.go` | HTTP router setup, where favicon routes are wired |
| `internal/server/runtime.go` | `RuntimeState` holding vault root path |
| `internal/vault/vault.go` | Vault loading (note: only loads `.md` files, ignores `.ico`) |
| `internal/web/static.go` | SPA handler that currently serves `index.html` for missing files |
| `internal/web/embed.go` | Embedded web bundle FS (build tag: `embed`) |
| `internal/web/embed_none.go` | Dev-mode web bundle FS (read from disk) |
| `internal/api/api.go` | REST API handler, `/api/config` endpoint |
| `cmd/retro-obsidian-publish/commands/serve/serve.go` | CLI flags and config wiring |
| `web/index.html` | SPA HTML shell (needs `<link rel="icon">`) |

### Architecture diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Browser / HTTP Client                     │
└─────────────┬───────────────────────────────────────────────┘
              │
              │  GET /favicon.ico
              │  GET /favicon.svg
              ▼
┌─────────────────────────────────────────────────────────────┐
│                  gorilla/mux Router                           │
│                                                               │
│  /api/*            → api.Handler (JSON)                       │
│  /vault-assets/*   → assetHandler (vault files)               │
│  /favicon.ico ─┐                                              │
│  /favicon.svg ─┤→ faviconHandler (NEW)                        │
│                │                                              │
│  /*                → agentPageHandler → spaHandler             │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│              faviconHandler (NEW)                             │
│                                                               │
│  1. --favicon flag set? ──→ serve that file                   │
│  2. vault root has file? ──→ serve from vault                 │
│  3. embedded bundle?      ──→ serve from bundle               │
│  4. none found            ──→ 404 text/plain                  │
│                                                               │
│  Cache-Control: public, max-age=3600                          │
│  MIME: auto-detected (.ico → image/x-icon, .svg → svg+xml)   │
└───────────────────────────────────────────────────────────────┘
```

### Data flow: favicon request

```
Browser                    Server                    Filesystem
   │                         │                          │
   │  GET /favicon.ico       │                          │
   │────────────────────────>│                          │
   │                         │                          │
   │                         │  stat(--favicon path)     │
   │                         │─────────────────────────>│
   │                         │<────────── not found ────│
   │                         │                          │
   │                         │  stat(vault/favicon.ico)  │
   │                         │─────────────────────────>│
   │                         │<──── found! ─────────────│
   │                         │                          │
   │                         │  open + ServeContent      │
   │                         │─────────────────────────>│
   │                         │<──── file bytes ─────────│
   │                         │                          │
   │  200 OK                 │                          │
   │  Content-Type: image/x-icon                       │
   │  Cache-Control: public, max-age=3600              │
   │<────────────────────────│                          │
```
