---
title: "Investigation diary"
doc_type: reference
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

# Investigation Diary

## Goal

Record the investigation and design work for adding per-site favicon support to Retro Obsidian Publish and creating a favicon for the go-go-parc vault.

## Step 1: Codebase investigation and design doc creation

Investigated the full server routing architecture to understand how HTTP requests flow through the system and why `/favicon.ico` currently returns the root HTML page. Traced the path from `gorilla/mux` router through SPA handler, vault loading, SSR proxy, and the embedded web bundle. Created a comprehensive design document covering the favicon resolution strategy, handler implementation, HTML injection, and the go-go-parc favicon design concept.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to handle favicon.ico instead of serving the root page in publish-vault/. We are using it to serve ~/code/wesen/go-go-golems/go-go-parc and we would like to configure a favicon.ico by site (or return the appropriate error warning), and create a favicon for go-go-parc, which is my golem research parc. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable. No need to be too hardcore about this, it doesn't seem like a super complex feature."

**Assistant interpretation:** Create a docmgr ticket for favicon support, write a thorough intern-ready design doc with system explanation and implementation guide, then upload to reMarkable.

**Inferred user intent:** Get a clear implementation guide so someone (including an intern) can pick up and implement favicon support without needing deep prior knowledge of the codebase.

### What I did

- Explored the full project structure and identified all relevant source files
- Read and analyzed `internal/server/server.go` (router setup), `runtime.go` (state management), `vault.go` (vault loading), `web/static.go` (SPA handler), `web/embed.go` and `embed_none.go` (embedded FS), `api/api.go` (REST endpoints), `cmd/.../serve.go` (CLI flags), `web/index.html` (HTML shell)
- Confirmed the bug: `/favicon.ico` routes forward to `spaHandler` which falls through to `index.html`
- Created docmgr ticket `FAVICON-001` with design-doc and diary documents
- Wrote comprehensive design document with architecture diagrams, pseudocode, decision records, implementation phases, and testing strategy

### Why

The user needs an intern-ready guide that explains the entire system architecture relevant to favicon handling, so someone unfamiliar with the codebase can implement the feature correctly.

### What worked

- Tracing the exact routing order in `server.go` made the bug immediately clear
- The existing `assetHandler` for `/vault-assets/` provided a perfect template for the favicon handler pattern
- The `vault.go` file already filters to `.md` files only, so favicon files in the vault root won't interfere with note loading

### What didn't work

- N/A — straightforward investigation

### What I learned

- The SSR mode already has explicit `/favicon.ico` and `/favicon.svg` routes, but they're wired to the SPA handler (which serves `index.html` for missing files). The non-SSR mode has no favicon routes at all.
- The vault root is the natural home for per-site favicons since Obsidian ignores non-Markdown files and the vault loader only processes `.md`.

### What was tricky to build

- Nothing tricky yet — the investigation was clean and the design is straightforward.

### What warrants a second pair of eyes

- The favicon resolution order (CLI override → vault root → embedded → 404) — is this the right priority?
- Whether to expose the favicon URL in `/api/config` or just rely on the HTML `<link>` tag.

### What should be done in the future

- Implement the favicon handler in Go (Phase 1)
- Create the go-go-parc favicon SVG/ICO (Phase 3)

### ReMarkable delivery

- Uploaded bundle: `FAVICON-001 - Per-site favicon support.pdf`
- Remote path: `/ai/2026/06/08/FAVICON-001`
- Verified via `remarquee cloud ls`

### Code review instructions

- Start with the design doc: `ttmp/2026/06/08/FAVICON-001--.../design-doc/01-*.md`
- Key files to understand the current behavior: `internal/server/server.go` (lines ~85-110 for routing), `internal/web/static.go` (full file for SPA handler logic)
- Verify: `docmgr doctor --ticket FAVICON-001`

### Technical details

- Ticket path: `ttmp/2026/06/08/FAVICON-001--add-per-site-favicon-ico-support-and-create-go-go-parc-favicon/`
- Design doc: `design-doc/01-favicon-per-site-support-design-and-implementation-guide.md`

## Step 2: Implement favicon handler, CLI flag, router wiring, tests, and HTML link tags

Implemented the full favicon support feature across 4 files: created a new `favicon.go` handler with cascading resolution (CLI override → vault root → 404), wired it into the router for both SSR and non-SSR modes, added the `--favicon` CLI flag, wrote 7 unit tests, and injected `<link rel="icon">` tags into the SPA HTML shell. Also created the go-go-parc favicon (SVG + ICO) in the vault root.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement all the tasks from the design doc: handler, CLI flag, router wiring, tests, HTML tags, and go-go-parc favicon.

**Inferred user intent:** Ship the favicon feature end-to-end.

**Commit (code):** 330838a — "feat: add per-site favicon handler with cascading resolution"

**Commit (code):** 15daa3c — "feat: add favicon link tags to SPA HTML shell"

### What I did

- Created `internal/server/favicon.go` with `newFaviconHandler` and `serveFileIfExists`
- Created `internal/server/favicon_test.go` with 7 tests
- Updated `internal/server/server.go`: added `FaviconPath` to `Config`, replaced old favicon routes with new handler in both SSR and non-SSR paths
- Updated `cmd/.../serve.go`: added `Favicon` to `Settings`, `--favicon` flag definition, and wired it to `Config.FaviconPath`
- Updated `web/index.html`: added `<link rel="icon">` for both `.ico` and `.svg`
- Created `go-go-parc/favicon.svg` (pixel-art golem with green eyes and park trees) and converted to `favicon.ico` via ImageMagick

### Why

The design doc laid out the implementation plan clearly; following it task-by-task ensured nothing was missed.

### What worked

- The handler pattern from `assetHandler` translated cleanly to the favicon case
- `http.ServeContent` handles MIME detection automatically (`.ico` → `image/x-icon`, `.svg` → `image/svg+xml`)
- All 7 tests passed on first run
- Full test suite (`go test ./...`) remained green
- ImageMagick SVG→ICO conversion worked without issues

### What didn't work

- VLM couldn't preview SVG directly; had to convert to PNG first for visual verification

### What I learned

- `http.ServeContent` sets both Content-Type and ETag automatically — no manual MIME mapping needed
- The router must register favicon handlers before the catch-all (`PathPrefix("/")`) in both SSR and non-SSR branches, otherwise gorilla/mux would never reach them

### What was tricky to build

- The router wiring needed care: the old SSR mode had favicon routes forwarding to `spaHandler`, while non-SSR mode had none. Unifying both under the same `newFaviconHandler` and registering it before the SSR/non-SSR branch simplified the code.

### What warrants a second pair of eyes

- The favicon handler currently skips the embedded web bundle lookup (step 3 in the design doc is a comment-only fallback). This is fine for now since no embedded favicon exists, but if one is added later, the handler should be updated.

### What should be done in the future

- Consider adding a default embedded favicon for deployments without a vault-root favicon
- Consider exposing favicon URL in `/api/config` for frontend consumption

### Code review instructions

- Start with `internal/server/favicon.go` (the handler), then `favicon_test.go` (tests)
- Then `internal/server/server.go` (router wiring — search for `faviconH`)
- Then `cmd/.../serve.go` (new `--favicon` flag — search for `Favicon`)
- Then `web/index.html` (new `<link>` tags)
- Run: `go test ./internal/server/ -run TestFavicon -v`
- Run: `go test ./... -count=1`

### Technical details

- Commits: `330838a` (handler + CLI + router + tests), `15daa3c` (HTML link tags)
- New files: `internal/server/favicon.go`, `internal/server/favicon_test.go`
- Modified files: `internal/server/server.go`, `cmd/retro-obsidian-publish/commands/serve/serve.go`, `web/index.html`
- go-go-parc favicons: `~/code/wesen/go-go-golems/go-go-parc/favicon.svg`, `favicon.ico`
