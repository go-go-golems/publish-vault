---
Title: Image serving and page title implementation guide
Ticket: RETRO-ASSETS-005
Status: active
Topics:
    - assets
    - images
    - config
    - page-title
    - obsidian-vault
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: backend/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: CLI flag schema and config handoff for vault name/page title
    - Path: backend/internal/api/api.go
      Note: /api/config contract to extend with pageTitle
    - Path: backend/internal/parser/parser.go
      Note: Markdown HTML post-processing seam for image source rewriting
    - Path: backend/internal/server/runtime.go
      Note: Resolved vault root and reload behavior needed for asset serving
    - Path: backend/internal/server/server.go
      Note: Server route ordering
    - Path: backend/internal/vault/vault.go
      Note: Vault note loading and rebuildHTML hook for resolving image paths
    - Path: web/index.html
      Note: Static fallback browser title
    - Path: web/src/App.tsx
      Note: Top-level config consumer for document title
    - Path: web/src/store/vaultApi.ts
      Note: Frontend SiteConfig contract
ExternalSources: []
Summary: Design and implementation guide for serving vault image assets and exposing a configurable browser page title.
LastUpdated: 2026-05-28T23:55:00Z
WhatFor: Use when implementing or reviewing RETRO-ASSETS-005.
WhenToUse: When image URLs in rendered notes 404 or deployments need a configurable document title.
---


# Image serving and page title implementation guide

## Executive summary

The current application parses Markdown notes into server-provided HTML and serves the React SPA from the same Go process, but it only exposes JSON APIs and the bundled SPA. Markdown image tags remain as whatever `goldmark` produced from the source Markdown. For vault notes that reference local Obsidian attachments, those `src` values are not routed to actual files under the configured vault root, so published pages can render broken images.

This ticket adds a safe `/assets/{path}` route that serves non-Markdown files from the active vault root, rewrites relative Markdown image `src` attributes to that public route during vault HTML rebuild, and extends public configuration with a configurable `pageTitle`. The frontend consumes `/api/config` and updates `document.title` after configuration is loaded.

## Problem statement and scope

### Requested outcomes

1. Serve images for pages such as `https://parc.yolo.scapegoat.dev/note/projects/2026/05/28/article-m5dial-proper-3d-renderer-building-a-z-buffered-planet-and-terrain-on-esp32-s3`, where the content originates from `~/code/wesen/go-go-golems/go-go-parc/` and images are present in that vault but not redirected to real assets.
2. Make the browser page title configurable instead of fixed to `Retro Obsidian Publish`.
3. Keep a docmgr ticket with analysis, tasks, implementation diary, file relations, and changelog.

### In scope

- Local vault images referenced by relative Markdown image URLs, including paths with spaces or nested folders.
- Root-relative `/assets/...` URLs emitted by the backend for note HTML.
- A safe HTTP handler that serves vault assets from the current runtime snapshot's resolved vault root.
- `--page-title` CLI flag and public `pageTitle` field in `/api/config`.
- Frontend document-title updates in backend mode and static/demo fallback.
- Unit tests for image URL rewriting, asset handler security/serving, config JSON, and title behavior where practical.

### Out of scope

- Uploading or syncing assets into a CDN.
- Full Obsidian attachment resolution semantics such as searching every attachment folder by basename when a Markdown image path is ambiguous.
- Serving hidden files or directory listings from the vault.
- Per-note dynamic title templates. A single site-level page title is enough for this ticket.

## Current-state architecture with evidence

### Server startup and route ordering

`backend/internal/server/server.go` owns runtime wiring. `Config` contains vault, display name, web-serving, watch, and reload settings, but no asset route or page title field (`server.go:23-32`). `Run` loads `RuntimeState`, derives `vaultName` from the vault directory basename if omitted (`server.go:57-61`), registers API routes (`server.go:76-79`), optionally registers reload, and finally mounts the SPA catch-all with `r.PathPrefix("/")` (`server.go:85-86`).

Because the SPA route is a catch-all, any new asset route must be registered before `PathPrefix("/")`. Otherwise `/assets/foo.png` will be handled as a client-side route and return `index.html` rather than an image.

### Runtime state and resolved roots

`backend/internal/server/runtime.go` resolves configured vault paths with `filepath.EvalSymlinks`, stores the active `resolvedRoot`, and swaps vault/search state atomically on reload. This matters for git-sync deployments because a stable configured path can point at a changing worktree. Asset serving should read from `state.ResolvedRoot()` for every request rather than capture a stale root at startup.

### API configuration

`backend/internal/api/api.go` exposes `/api/config` and currently returns only `vaultName` and `notes` (`api.go:65-78`). `Handler` stores `vaultName` as a string (`api.go:39-43`), and `api.NewWithProvider` accepts only a vault name (`api.go:50-52`). Extending config cleanly means changing this to accept a public site config or adding a `pageTitle` field alongside `vaultName`.

### Markdown parsing and rendered HTML

`backend/internal/parser/parser.go` uses `goldmark` with GFM extensions and emits HTML (`parser.go:48-75`). It already performs post-processing for wiki links through `ReplaceWikiLinksString` and `ReplaceWikiLinkDisplay` (`parser.go:208+`). There is no image-specific post-processing. Therefore Markdown such as `![render](images/foo.png)` remains a normal `<img src="images/foo.png" ... />` in the `Note.HTML` string.

`backend/internal/vault/vault.go` loads only `.md` files during `LoadAll` (`vault.go:74-87`), builds wiki link indexes, backlinks, and then calls `rebuildHTML` (`vault.go:99-101`). `rebuildHTML` is the correct central hook for transforming note HTML after all notes are loaded (`vault.go:210-220`).

### Frontend config and title

`web/src/store/vaultApi.ts` defines `SiteConfig` with only `vaultName` and `notes` (`vaultApi.ts:48-51`) and fetches `/api/config` in backend mode (`vaultApi.ts:60-73`). `web/src/App.tsx` already calls `useGetConfigQuery()` and passes `config?.vaultName` to `VaultLayout` (`App.tsx:10-14`), so adding a document-title effect there is low risk. `web/index.html` currently hard-codes `<title>Retro Obsidian Publish</title>` (`web/index.html:6`), which remains a good initial fallback before the React app loads.

## Gap analysis

1. **No public route for vault assets.** The only registered non-API route is the SPA catch-all. Relative image requests cannot reliably reach files in the vault.
2. **No HTML rewrite for local images.** Even if an asset route existed, rendered Markdown may request paths relative to the current browser route, e.g. `/note/projects/.../images/foo.png`, rather than a stable `/assets/...` URL.
3. **No request-time reload awareness for assets.** In git-sync mode, the active resolved vault root can change after `POST /api/admin/reload`. Asset serving must follow `RuntimeState` rather than a startup-only filesystem root.
4. **No configurable page title.** The page title is fixed in HTML and omitted from `/api/config`, so deployments cannot set a site-specific browser/tab title without rebuilding the frontend.

## Proposed architecture and APIs

### Public URLs

- Notes keep using `/note/{slug}`.
- Vault files use `/assets/{relative-vault-path}`.
- The path segment after `/assets/` is URL-escaped as needed by Go's URL handling and browser serialization. The backend decodes and validates it before serving.

Examples:

```text
Markdown source:       ![Planet](images/planet.png)
Rendered note path:    Projects/2026/05/28/article.md
Stored asset path:     Projects/2026/05/28/images/planet.png
HTML output:           <img src="/assets/Projects/2026/05/28/images/planet.png" alt="Planet" />
HTTP file lookup:      {resolved vault root}/Projects/2026/05/28/images/planet.png
```

If Markdown already uses an absolute HTTP(S), protocol-relative, `data:`, `mailto:`, `#fragment`, `/api/...`, `/note/...`, or `/assets/...` URL, leave it unchanged.

### Backend config contract

Extend `/api/config` from:

```json
{ "vaultName": "parc", "notes": 123 }
```

to:

```json
{ "vaultName": "parc", "pageTitle": "PARC", "notes": 123 }
```

Defaulting rules:

1. `vaultName`: existing behavior; explicit `--vault-name`, else `filepath.Base(vaultDir)`.
2. `pageTitle`: explicit `--page-title`, else `vaultName`.

This preserves existing display-name behavior while giving deployments a single obvious flag for tab titles.

### CLI contract

Add a Glazed flag:

```text
--page-title string   Browser page title returned by /api/config. Defaults to --vault-name or the vault directory basename.
```

The flag maps through `serve.Settings.PageTitle` into `server.Config.PageTitle`.

### Asset-serving rules

Implement a server-side helper with these invariants:

1. Strip `/assets/`, URL-decode, convert to slash-normalized relative path.
2. Reject empty paths, absolute paths, `..` traversal, and path components beginning with `.`.
3. Join to `state.ResolvedRoot()` at request time.
4. Reject directories and `.md` files.
5. Set conservative cache headers, e.g. `Cache-Control: public, max-age=300`, because git-sync reloads can swap content while paths remain stable.
6. Use `http.ServeFile` for content type/range support after validation.

### Image HTML rewriting

Add a parser-level helper to avoid duplicating regex logic in the vault package:

```go
func RewriteImageSources(html string, resolver func(src string) string) string
```

`vault.rebuildHTML` can call:

```go
note.HTML = parser.RewriteImageSources(note.HTML, func(src string) string {
    return v.ResolveAssetPath(note.Path, src)
})
```

Resolution rules:

1. Skip external, fragment-only, protocol-relative, data, mailto, tel, and already-root-routed URLs.
2. For `src` beginning with `/`, only rewrite if it is not already `/assets/`, `/api/`, or `/note/` and if treating it as vault-root-relative produces a file. This is optional; the initial implementation can leave root-relative URLs unchanged if tests document that behavior.
3. For relative paths, resolve relative to the note's containing directory.
4. Clean with `path.Clean`, reject traversal outside the vault, and return `/assets/{cleaned-relative-path}`.

## Pseudocode and key flows

### Server route registration

```go
r := mux.NewRouter()
h := api.NewWithProvider(state, api.SiteConfig{VaultName: vaultName, PageTitle: pageTitle})
h.Register(r)
r.HandleFunc("/api/healthz", healthHandler(state)).Methods("GET")
r.PathPrefix("/assets/").Handler(assetHandler(state))
if cfg.ServeWeb {
    r.PathPrefix("/").Handler(web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"}))
}
```

### Asset handler

```go
func assetHandler(state *RuntimeState) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rel := strings.TrimPrefix(r.URL.Path, "/assets/")
        if !validVaultAssetPath(rel) { http.NotFound(w, r); return }
        root := state.ResolvedRoot()
        abs := filepath.Join(root, filepath.FromSlash(rel))
        if !strings.HasPrefix(abs, root + string(os.PathSeparator)) { http.NotFound(w, r); return }
        info, err := os.Stat(abs)
        if err != nil || info.IsDir() || strings.EqualFold(filepath.Ext(abs), ".md") {
            http.NotFound(w, r); return
        }
        w.Header().Set("Cache-Control", "public, max-age=300")
        http.ServeFile(w, r, abs)
    })
}
```

### Frontend title effect

```tsx
function Router() {
  const { data: config } = useGetConfigQuery();
  useEffect(() => {
    document.title = config?.pageTitle || config?.vaultName || "Retro Obsidian Publish";
  }, [config?.pageTitle, config?.vaultName]);
  return <VaultLayout vaultName={config?.vaultName}>...</VaultLayout>;
}
```

## Implementation plan

### Phase 1: Documentation and ticket setup

- Create `RETRO-ASSETS-005`.
- Write this design guide.
- Add detailed tasks.
- Start diary with the prompt context, evidence gathered, and implementation plan.
- Relate key files to this document.
- Commit documentation before code so the implementation has a reviewable plan.

### Phase 2: Backend asset URL generation

- Add parser tests for image-source rewriting in `backend/internal/parser/parser_test.go` or a focused new test file.
- Implement `RewriteImageSources` with support for double-quoted and single-quoted `src` attributes emitted by Markdown/HTML.
- Add vault tests that create a temporary note plus image and assert the rendered note HTML points at `/assets/...`.
- Implement vault asset path resolution in `backend/internal/vault/vault.go`.
- Run `go test ./...` in `backend/`.
- Commit focused backend HTML-rewrite changes.

### Phase 3: Backend asset serving and config

- Add `PageTitle` to `server.Config` and `serve.Settings`.
- Add `PageTitle` to `api.SiteConfig`, defaulting to `vaultName` when omitted.
- Implement `assetHandler` in `backend/internal/server/server.go` and register it before SPA catch-all.
- Add server tests for serving an image, rejecting traversal, rejecting Markdown files, and page-title config JSON.
- Run `go test ./...` in `backend/`.
- Commit focused backend server/API changes.

### Phase 4: Frontend title consumption

- Add `pageTitle` to `web/src/store/vaultApi.ts` `SiteConfig`.
- Update `web/src/App.tsx` to set `document.title` from config.
- Update static vault config if it returns a config object.
- Run `pnpm --dir web typecheck` or the available frontend validation command.
- Commit focused frontend changes.

### Phase 5: End-to-end validation and diary closeout

- Create or use a temp vault with a Markdown note and image.
- Run the server with `--serve-web=false` for API/asset checks.
- Verify `/api/notes/{slug}` returns `/assets/...` image URLs.
- Verify `/assets/...` returns image bytes and a correct HTTP status.
- Verify `/api/config` includes `pageTitle`.
- If practical, load the SPA and verify browser title.
- Update diary/changelog/tasks and run `docmgr doctor`.

## Test strategy

### Backend unit tests

- `parser.RewriteImageSources`:
  - rewrites `<img src="images/a.png" />` through resolver.
  - rewrites single-quoted attributes.
  - leaves `https://`, `data:`, `/assets/`, `/api/`, `/note/`, and `#fragment` unchanged when resolver returns original values.
- `vault`:
  - note-relative image paths become `/assets/{note-dir}/...`.
  - `../` is cleaned only if it remains inside vault; paths escaping the root stay unchanged.
- `server`:
  - `/assets/image.png` returns bytes.
  - `/assets/note.md` returns 404.
  - `/assets/../secret.png` or encoded traversal returns 404.
  - `/api/config` returns `pageTitle`.

### Manual validation commands

```bash
cd backend
go test ./... -count=1

tmp=$(mktemp -d)
mkdir -p "$tmp/projects/images"
printf '# Demo\n\n![Planet](images/planet.png)\n' > "$tmp/projects/demo.md"
printf 'fakepng' > "$tmp/projects/images/planet.png"
go run ./cmd/retro-obsidian-publish serve --vault "$tmp" --serve-web=false --port 18080 --page-title PARC &
curl -s http://127.0.0.1:18080/api/config
curl -s http://127.0.0.1:18080/api/notes/projects/demo | jq -r .html
curl -i http://127.0.0.1:18080/assets/projects/images/planet.png
```

## Risks, alternatives, and open questions

### Risks

- Regex-based HTML rewriting can miss unusual HTML shapes. The immediate source is goldmark output, so a targeted regex is acceptable for this ticket; a future HTML tokenizer could replace it if image handling grows.
- Serving all non-hidden, non-Markdown files under the vault can expose attachments that were not linked from published pages. That matches the requested static-asset behavior but should be documented for deployment operators.
- Obsidian can resolve attachments by basename from configured attachment folders. This design starts with path-relative and vault-root-relative references, which are deterministic and safer.

### Alternatives considered

1. **Frontend-only rewrite.** Rejected because the browser cannot safely know the vault filesystem layout or resolve note-relative paths after backend parsing.
2. **Embed images as base64 in note JSON.** Rejected because it inflates API responses, breaks caching, and makes large notes expensive.
3. **Copy assets into the bundled web build.** Rejected because live git-sync vault deployments need assets to update with vault content without rebuilding the SPA.
4. **Expose `http.FileServer(http.Dir(root))` directly.** Rejected because it risks directory listings, Markdown exposure, hidden files, and path traversal edge cases unless wrapped carefully.

## References

- `backend/cmd/retro-obsidian-publish/commands/serve/serve.go`: CLI settings and flag schema.
- `backend/internal/server/server.go`: route registration, runtime startup, SPA catch-all.
- `backend/internal/server/runtime.go`: active resolved vault root and reload behavior.
- `backend/internal/api/api.go`: public `/api/config` contract.
- `backend/internal/parser/parser.go`: Markdown-to-HTML parsing and post-processing helpers.
- `backend/internal/vault/vault.go`: note loading and HTML rebuild pipeline.
- `web/src/store/vaultApi.ts`: frontend `SiteConfig` type and `/api/config` endpoint.
- `web/src/App.tsx`: central config consumer and router.
- `web/index.html`: initial static browser title fallback.
