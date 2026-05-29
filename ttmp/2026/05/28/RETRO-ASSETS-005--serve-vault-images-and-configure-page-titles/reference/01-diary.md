---
Title: Diary
Ticket: RETRO-ASSETS-005
Status: active
Topics:
    - assets
    - images
    - config
    - page-title
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: backend/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: --page-title CLI flag (commit 7b2b8c6)
    - Path: backend/internal/api/api.go
      Note: /api/config pageTitle contract (commit 7b2b8c6)
    - Path: backend/internal/parser/parser.go
      Note: Image src rewrite helper (commit 7b2b8c6)
    - Path: backend/internal/server/server.go
      Note: Safe /assets route and page-title config defaulting (commit 7b2b8c6)
    - Path: backend/internal/vault/vault.go
      Note: Note-relative asset URL resolution and HTML rebuild integration (commit 7b2b8c6)
    - Path: web/src/App.tsx
      Note: Runtime document.title update from config (commit aff4713)
    - Path: web/src/store/vaultApi.ts
      Note: Frontend SiteConfig pageTitle type (commit aff4713)
    - Path: web/src/vault/staticVault.ts
      Note: Static mode pageTitle config (commit aff4713)
ExternalSources: []
Summary: Chronological implementation diary for vault image serving and configurable page titles.
LastUpdated: 2026-05-28T23:55:00Z
WhatFor: Use to resume, review, or validate RETRO-ASSETS-005.
WhenToUse: Before continuing implementation or reviewing the image/page-title changes.
---


# Diary

## Goal

This diary records the investigation, implementation, validation, commits, and follow-up risks for `RETRO-ASSETS-005`: serving vault images correctly and making the browser page title configurable.

## Step 1: Ticket setup, current-state analysis, and implementation plan

I created a new docmgr ticket and wrote an evidence-backed design guide before changing code. The main architectural finding is that the backend already has a clean parsing and HTML rebuild seam in `vault.rebuildHTML`, but it does not currently rewrite Markdown image URLs or expose a vault asset route before the SPA catch-all.

The second finding is that site configuration already flows from the Go API into the React app via `/api/config`; it only carries `vaultName` and note count today. That makes `pageTitle` a small extension to an existing contract rather than a separate frontend configuration mechanism.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to address:

- serving images, for example: https://parc.yolo.scapegoat.dev/note/projects/2026/05/28/article-m5dial-proper-3d-renderer-building-a-z-buffered-planet-and-terrain-on-esp32-s3 has images (from ~/code/wesen/go-go-golems/go-go-parc/), but they are not redirected to the real assets.
- make page title configurable. 

Create a detailed analysis and implementation guide, then add detailed tasks to the ticket, then implement, commit at appropriate intervals, and keep a diary."

**Assistant interpretation:** Create a full docmgr ticket, analyze how this app currently handles note HTML/assets/config, write a detailed implementation plan and tasks, then implement and commit incremental code/docs changes while maintaining a diary.

**Inferred user intent:** The published PARC vault should render local note images correctly and should have deployment-specific browser/tab branding without ad-hoc patches.

**Commit (code):** N/A — planning step only so far.

### What I did

- Ran repository orientation commands:
  - `pwd && git status --short && docmgr status --summary-only`
  - `find . -maxdepth 3 -type f | sed 's#^./##' | sort | head -200`
  - `docmgr ticket list && find ttmp -maxdepth 3 -type f | sort`
- Created the ticket:
  - `docmgr ticket create-ticket --ticket RETRO-ASSETS-005 --title "Serve vault images and configure page titles" --topics assets,images,config,page-title,obsidian-vault`
- Added documents:
  - `docmgr doc add --ticket RETRO-ASSETS-005 --doc-type design-doc --title "Image serving and page title implementation guide"`
  - `docmgr doc add --ticket RETRO-ASSETS-005 --doc-type reference --title "Diary"`
- Inspected source files that define the relevant runtime behavior:
  - `backend/cmd/retro-obsidian-publish/commands/serve/serve.go`
  - `backend/internal/server/server.go`
  - `backend/internal/server/runtime.go`
  - `backend/internal/api/api.go`
  - `backend/internal/parser/parser.go`
  - `backend/internal/vault/vault.go`
  - `web/src/store/vaultApi.ts`
  - `web/src/App.tsx`
  - `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`
  - `web/index.html`
- Wrote the primary design guide with current-state evidence, proposed APIs, pseudocode, implementation phases, tests, risks, and alternatives.
- Replaced the placeholder tasks file with a detailed phase-by-phase checklist.
- Replaced the placeholder reference document with this diary entry.

### Why

- The requested change touches backend routing, parser/vault HTML generation, API contracts, frontend config consumption, and deployment behavior. A design-first pass reduces the risk of adding a route that conflicts with the SPA catch-all or rewriting image URLs in the wrong layer.
- The ticket needs enough detail for review and continuation even if implementation spans multiple commits.

### What worked

- The existing `RuntimeState` abstraction already tracks the active resolved vault root, which is exactly the information an asset route needs after git-sync reloads.
- The existing `/api/config` flow already reaches the top-level React router, so configurable page title can be implemented with a small contract extension.
- The existing `vault.rebuildHTML` post-processing step is a natural place to normalize note-relative image URLs after Markdown parsing.

### What didn't work

- A broad `rg -n "vault|note|asset|image|title|Config|serve|http|static|Page" backend web/src README.md -S` command produced noisy matches inside bundled/generated frontend assets under `backend/internal/web/embed/public/assets/...`. I narrowed the investigation to source files using `rg --files backend web/src | sort` and direct reads of relevant files.
- `git status --short` showed many existing untracked screenshot and `.playwright-mcp/` artifacts before this work began. I did not touch them.

### What I learned

- The Go server registers the SPA catch-all at `PathPrefix("/")`, so `/assets/` must be registered earlier or image requests will receive the SPA shell.
- The backend currently loads only `.md` files into the `Vault`, which means attachments do not need to be indexed as notes. Asset serving can use request-time filesystem lookup instead.
- Frontend title configuration should be runtime data from `/api/config`, not a build-time Vite variable, because the same SPA bundle is served for different vault deployments.

### What was tricky to build

- The main subtlety is path ownership. Markdown image paths are semantically relative to the note file, but browser requests would otherwise resolve them relative to `/note/{slug}`. The design resolves paths during backend HTML rebuild, where both the note path and the vault root are available.
- Another subtlety is reload behavior. Capturing an `http.Dir` at startup could serve stale files after a git-sync symlink flips. The planned handler reads `state.ResolvedRoot()` per request.

### What warrants a second pair of eyes

- The security boundary of `/assets/`: traversal, hidden files, directory serving, and Markdown exposure must be checked carefully.
- The exact Obsidian attachment-resolution semantics are broader than deterministic note-relative paths. Review whether the PARC vault's image references are relative, vault-root-relative, or basename-only before declaring the implementation complete.
- Regex-based image rewriting should be verified against goldmark's actual image tag output.

### What should be done in the future

- If PARC uses Obsidian's attachment-folder search by basename, add a second resolution phase that searches configured attachment directories or a prebuilt non-Markdown asset index.
- Consider an operator-facing note that all non-hidden, non-Markdown files under the vault may be publicly served when linked.

### Code review instructions

- Start with the design doc: `ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/design-doc/01-image-serving-and-page-title-implementation-guide.md`.
- Then review the future implementation in this order:
  1. parser image-source rewrite helper and tests,
  2. vault path resolution and HTML rebuild integration,
  3. server `/assets/` handler and route ordering,
  4. API/CLI page-title config,
  5. frontend `document.title` effect.
- Validation command planned for backend changes: `cd backend && go test ./... -count=1`.
- Validation command planned for frontend changes: `pnpm --dir web typecheck` or the repository's equivalent frontend check.

### Technical details

- Important source routes and seams:
  - `backend/internal/server/server.go`: route registration and SPA catch-all.
  - `backend/internal/server/runtime.go`: active resolved vault root.
  - `backend/internal/api/api.go`: `/api/config` contract.
  - `backend/internal/parser/parser.go`: Markdown-to-HTML conversion and post-processing helpers.
  - `backend/internal/vault/vault.go`: note loading and `rebuildHTML`.
  - `web/src/App.tsx`: central config consumer.
- Proposed public contract:
  - `GET /assets/{vault-relative-path}` serves validated non-Markdown vault files.
  - `GET /api/config` returns `{ "vaultName": string, "pageTitle": string, "notes": number }`.

## Step 2: Backend image rewriting, asset serving, and config contract

I implemented the backend side in one focused code commit. The change adds image-source rewriting to the Markdown HTML post-processing pipeline, resolves note-relative and vault-root-relative image paths to `/assets/...`, serves safe non-Markdown vault files through a new `/assets/` route, and extends `/api/config` with `pageTitle`.

The implementation keeps the asset route tied to `RuntimeState.ResolvedRoot()` at request time, so a future git-sync reload can swap the active worktree without rebuilding the router. It also registers `/assets/` before the SPA catch-all so image requests are not swallowed by `PathPrefix("/")`.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the backend path of the requested image serving and page-title feature, with tests and a focused commit.

**Inferred user intent:** Local Obsidian attachments should become stable web assets, and deployment metadata should flow through runtime config.

**Commit (code):** 7b2b8c6ef5246261607070e5395e64ab01571fd1 — "Serve vault assets and expose page title config"

### What I did

- Added `parser.RewriteImageSources` in `backend/internal/parser/parser.go`.
- Added parser tests for double-quoted, single-quoted, and mismatched-quote image `src` rewriting in `backend/internal/parser/parser_test.go`.
- Added `Vault.ResolveAssetURL` in `backend/internal/vault/vault.go` and called it from `rebuildHTML` after wiki-link rewriting.
- Added vault tests for relative image rewriting, root-relative image rewriting, remote image preservation, and traversal handling in `backend/internal/vault/vault_test.go`.
- Added `api.PublicConfig` and `pageTitle` support in `backend/internal/api/api.go` plus config JSON tests in `backend/internal/api/api_test.go`.
- Added `PageTitle` to `server.Config`, `serve.Settings`, and the `--page-title` Glazed flag.
- Added `assetHandler` and `validAssetPath` to `backend/internal/server/server.go`, registering `/assets/` before the SPA catch-all.
- Added server tests for serving assets and rejecting unsafe/hidden/Markdown paths in `backend/internal/server/runtime_test.go`.
- Ran:
  - `gofmt -w backend/cmd/retro-obsidian-publish/commands/serve/serve.go backend/internal/api/api.go backend/internal/parser/parser.go backend/internal/vault/vault.go backend/internal/vault/vault_test.go backend/internal/parser/parser_test.go backend/internal/api/api_test.go backend/internal/server/server.go backend/internal/server/runtime_test.go`
  - `cd backend && go test ./... -count=1`
- Committed backend changes.

### Why

- The backend owns the filesystem root and note path context, so it is the right layer to convert Markdown image references into public URLs.
- Serving assets through Go rather than embedding them in note JSON allows normal browser caching and avoids bloating note API responses.
- `pageTitle` belongs in the existing public config contract because the frontend already fetches `/api/config` at startup.

### What worked

- The existing `rebuildHTML` hook was enough to integrate image rewriting without changing the parser's `Parse` API.
- The existing dynamic `RuntimeState` provider model worked for both `/api/config` and `/assets/` request-time behavior.
- Backend tests passed after fixing an import-name conflict.

### What didn't work

- First backend test run failed because importing the standard library `html` package collided with the existing `github.com/yuin/goldmark/renderer/html` import:
  - Command: `cd backend && go test ./... -count=1`
  - Error:
    - `internal/parser/parser.go:17:2: html redeclared in this block`
    - `internal/parser/parser.go:8:2: other declaration of html`
    - `internal/parser/parser.go:17:2: "github.com/yuin/goldmark/renderer/html" imported and not used`
    - `internal/parser/parser.go:63:9: undefined: html.WithHardWraps`
- Fixed by aliasing the standard library import as `stdhtml`.

### What I learned

- Go's package import names can collide even when import paths differ; aliasing is required when both packages want the same local name.
- The current server tests live in `runtime_test.go`; there is no separate `server_test.go`, so I added asset handler tests there.
- `url.PathEscape` per path segment gives stable URLs while preserving slashes.

### What was tricky to build

- The key invariant is to resolve image paths with POSIX-style slash semantics because note paths and HTML URLs use `/`, while file serving must later convert to platform paths with `filepath.FromSlash`.
- The asset handler must reject malicious paths before joining them to the root, then still verify the cleaned absolute path is inside the cleaned root. This double check protects against both obvious `..` segments and future path normalization surprises.
- `RewriteImageSources` intentionally preserves mismatched quotes rather than attempting a risky rewrite.

### What warrants a second pair of eyes

- Review the asset exposure policy: it serves any non-hidden, non-Markdown file under the vault if the requester knows the path.
- Review whether rejecting every path component beginning with `.` is acceptable for all real vault attachment paths.
- Review whether PARC has basename-only Obsidian embeds that require an attachment-folder index beyond note-relative and vault-root-relative paths.

### What should be done in the future

- Add attachment-folder search if real PARC notes use Obsidian's basename-only attachment resolution.
- Consider explicit cache-control configuration if assets are served behind a CDN.

### Code review instructions

- Start with `backend/internal/vault/vault.go` (`ResolveAssetURL`, `rebuildHTML`) and `backend/internal/server/server.go` (`assetHandler`, route registration).
- Then review parser/API/CLI integration and tests.
- Validate with `cd backend && go test ./... -count=1`.

### Technical details

- Public config now returns: `{ "vaultName": string, "pageTitle": string, "notes": number }`.
- Relative images resolve against the note's directory; root-relative images resolve against the vault root.
- External, protocol-relative, `data:`, `mailto:`, `tel:`, `/assets/`, `/api/`, and `/note/` URLs are left unchanged.

## Step 3: Frontend page-title consumption

I updated the React app to consume the new `pageTitle` field and set `document.title` after `/api/config` loads. The static vault provider now also returns `pageTitle`, so demo/static mode continues to satisfy the shared `SiteConfig` type.

The static `<title>` in `web/index.html` remains as an initial fallback while the SPA loads. Runtime configuration overrides it once React fetches config.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the frontend side of configurable page titles and validate TypeScript.

**Inferred user intent:** Operators should not need to rebuild or patch HTML to customize the browser tab title for a deployment.

**Commit (code):** aff4713ac2cf957e0a6a9fde56ac129ae49f14aa — "Update frontend document title from config"

### What I did

- Added `pageTitle` to `web/src/store/vaultApi.ts` `SiteConfig`.
- Added a `useEffect` in `web/src/App.tsx` that sets `document.title` from `config.pageTitle`, falling back to `config.vaultName` and then `Retro Obsidian Publish`.
- Added `pageTitle` to `web/src/vault/staticVault.ts` static config.
- Ran `pnpm --dir web check`.
- Committed frontend changes.

### Why

- `App.tsx` already owns top-level config loading and layout rendering, so it is the smallest and most central place to update browser title state.
- Keeping the static provider in sync prevents TypeScript drift between backend and demo/static modes.

### What worked

- `pnpm --dir web check` passed without requiring additional frontend changes.
- The change is isolated to three source files and does not alter routing or layout behavior.

### What didn't work

- N/A.

### What I learned

- The existing static-mode API mirrors backend config closely enough that adding `pageTitle` only required one object-literal update.

### What was tricky to build

- The main subtlety is timing: the document initially uses `web/index.html`'s title and then updates after async config fetch. The fallback chain avoids blank or `undefined` titles during loading.

### What warrants a second pair of eyes

- Confirm whether product expectations are a site-level title only or a dynamic title such as `Note Title — Site Title`.

### What should be done in the future

- If desired, add per-note dynamic document titles once the site-level config behavior is accepted.

### Code review instructions

- Review `web/src/App.tsx` first, then `web/src/store/vaultApi.ts` and `web/src/vault/staticVault.ts`.
- Validate with `pnpm --dir web check`.

### Technical details

- Frontend validation command: `pnpm --dir web check`.

## Step 4: Build and end-to-end validation

I ran the repository's web build/export command and performed a temporary-vault smoke test against the Go server. The smoke test verified that config includes `pageTitle`, note HTML emits `/assets/...` image URLs, and the `/assets/...` route returns the image bytes.

The build command also exercised the Dagger-based frontend production build and exported the generated bundle to the backend embed directory. No tracked embed files changed after the export, so there was no additional embed commit.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the completed implementation across backend, frontend type checking, production web build, and API/asset smoke behavior.

**Inferred user intent:** The implementation should be demonstrably working, not only unit-tested.

**Commit (code):** N/A — validation step after commits.

### What I did

- Ran combined validation:
  - `cd backend && go test ./... -count=1 && cd .. && pnpm --dir web check`
- Ran production web export:
  - `make build-web`
- Ran a temp-vault smoke test:
  - Created a temporary vault with `projects/2026/05/28/demo.md` and `projects/2026/05/28/images/planet.png`.
  - Started `go run ./cmd/retro-obsidian-publish serve --vault "$tmp" --serve-web=false --watch=false --port 18080 --vault-name PARC --page-title "PARC Notes"`.
  - Fetched `/api/config`.
  - Fetched `/api/notes/projects/2026/05/28/demo` and extracted `.html`.
  - Fetched `/assets/projects/2026/05/28/images/planet.png`.

### Why

- Unit tests cover helper behavior, but the smoke test proves that the CLI, runtime config, parser/vault integration, HTTP routes, and asset file serving work together.
- `make build-web` checks that the frontend still builds for embedded single-binary serving.

### What worked

- `cd backend && go test ./... -count=1` passed.
- `pnpm --dir web check` passed.
- `make build-web` completed and exported web dist successfully.
- Smoke-test output showed:
  - Config: `{"vaultName":"PARC","pageTitle":"PARC Notes","notes":1}`
  - HTML: `<h1 id="demo">Demo</h1>\n<p><img src="/assets/projects/2026/05/28/images/planet.png" alt="Planet" /></p>`
  - Asset body: `fakepng`

### What didn't work

- `make build-web` emitted an initial Dagger/Docker resolver `HTTP HEAD` error while resolving `node:22`, but Dagger retried/continued and the overall build succeeded.
- `make build-web` emitted the existing Vite chunk-size warning for large chunks. This is unrelated to the ticket and did not fail the build.
- The Dagger run printed a notice that Dagger `v0.21.0` is available; no upgrade was performed.

### What I learned

- The generated embed export did not leave tracked changes after the frontend source commit, so the committed source changes are enough for this repository state.
- The smoke-test route confirms lower-case paths when the source vault path is lower-case; paths retain whatever case the Markdown source uses in the emitted `/assets/...` URL.

### What was tricky to build

- Backgrounding `go run` for a smoke test requires waiting for `/api/healthz` before issuing assertions; the script looped until the server was ready and then killed the recorded PID.

### What warrants a second pair of eyes

- Manual validation used a temp vault, not the real PARC vault. If PARC has attachment path patterns beyond note-relative paths, test against `~/code/wesen/go-go-golems/go-go-parc/` before deployment.

### What should be done in the future

- Run a production smoke test against the real PARC content after deployment.

### Code review instructions

- Re-run:
  - `cd backend && go test ./... -count=1`
  - `pnpm --dir web check`
  - `make build-web`
- Optionally repeat the temp-vault smoke test from this entry.

### Technical details

- Smoke command summary:
  - `curl -fsS http://127.0.0.1:18080/api/config`
  - `curl -fsS http://127.0.0.1:18080/api/notes/projects/2026/05/28/demo`
  - `curl -fsS http://127.0.0.1:18080/assets/projects/2026/05/28/images/planet.png`

## Step 5: Fix SPA bundle asset route collision discovered during real-vault testing

When the PARC vault was served from tmux, the note images worked but the SPA's own CSS and JavaScript returned 404. The root cause was a route-prefix collision: Vite emits bundled frontend files under `/assets/...`, and the new vault asset handler had been registered at the same `/assets/` prefix before the SPA handler.

I moved vault-served files to `/vault-assets/...` so `/assets/...` remains owned by the embedded SPA bundle. This keeps the application shell and vault attachments in separate URL namespaces.

### Prompt Context

**User prompt (verbatim):** "404 on css and js"

**Assistant interpretation:** The local tmux test server loads the page shell but bundled frontend CSS/JS requests are failing, likely because of the newly added asset route.

**Inferred user intent:** Make the local test server usable in the browser while preserving vault image serving.

**Commit (code):** 2f9f40f29dde11745dc4aed05f623590d1ad4415 — "Avoid SPA asset route conflict for vault files"

### What I did

- Changed the vault asset route from `/assets/` to `/vault-assets/` in `backend/internal/server/server.go`.
- Changed rendered note image URLs from `/assets/...` to `/vault-assets/...` in `backend/internal/vault/vault.go`.
- Updated backend tests in `backend/internal/server/runtime_test.go` and `backend/internal/vault/vault_test.go`.
- Ran `cd backend && go test ./... -count=1`.
- Restarted tmux session `retro-parc`.
- Verified:
  - `/api/healthz` returns 617 notes.
  - `/assets/index-C29_Oxvd.css` returns `HTTP/1.1 200 OK`.
  - `/assets/index-CryKyRHX.js` returns `HTTP/1.1 200 OK`.
  - The target note now emits `/vault-assets/Projects/2026/05/28/assets/m5dial-proper-3d-renderer/poster-terrain.png`.
  - That vault image returns `HTTP/1.1 200 OK`.

### Why

- `/assets/` is already the Vite production bundle namespace. Using it for vault attachments stole frontend bundle requests before they could reach the embedded SPA handler.
- `/vault-assets/` is explicit and avoids ambiguity between application assets and vault content assets.

### What worked

- Backend tests passed after updating the expected prefix.
- Restarted PARC server served both app bundle assets and vault image assets correctly.

### What didn't work

- The original `/assets/` design caused CSS/JS 404s in real browser testing because those paths were routed to the vault asset handler and looked up inside the PARC vault instead of the embedded web bundle.

### What I learned

- Route prefix ownership must account for generated frontend build output, not only handwritten API routes.
- The real-vault tmux test caught a conflict that API-only smoke tests did not catch because those tests did not load the SPA bundle.

### What was tricky to build

- The issue was not with embedded build output; it was route ordering and prefix collision. The fix was to separate namespaces instead of trying to make one handler infer whether a path belongs to Vite or the vault.

### What warrants a second pair of eyes

- Review whether `/vault-assets/` is the final preferred public URL name or whether it should be configurable.
- Update public/operator docs if any external references already mention `/assets/` for vault files.

### What should be done in the future

- Add an integration test that verifies SPA bundle assets and vault assets can both be served in the same router.

### Code review instructions

- Review `backend/internal/server/server.go` for route prefix separation.
- Review `backend/internal/vault/vault.go` for emitted URL prefix.
- Validate by loading `http://127.0.0.1:8080` and checking CSS/JS plus note images.

### Technical details

- Application bundle namespace: `/assets/...`.
- Vault content asset namespace: `/vault-assets/...`.
