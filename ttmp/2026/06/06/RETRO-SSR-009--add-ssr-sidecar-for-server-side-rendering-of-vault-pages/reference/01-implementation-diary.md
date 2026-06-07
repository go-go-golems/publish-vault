---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: backend/internal/parser/parser.go
      Note: Final unresolved-link crawl fix described in diary
    - Path: backend/internal/server/agent_markdown.go
      Note: Phase 7 markdown endpoint implementation described in diary
    - Path: backend/internal/vault/vault.go
      Note: Final unresolved-link resolver change described in diary
    - Path: web/server.mjs
      Note: Phase 7 SSR metadata changes described in diary
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the analysis, design, and implementation steps for adding SSR sidecar support to Retro Obsidian Publish, modeled after the existing Glazed SSR implementation.

## Step 1: Evidence Gathering and Architecture Analysis

This step involved a thorough investigation of both the publish-vault codebase and the glazed SSR reference implementation. The goal was to map every component that the SSR sidecar touches — routing, state management, API endpoints, build pipeline, and Go server wiring — and identify the key differences between the two projects.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add SSR to ./publish-vault/, similar to how we did it for ./glazed/web/. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable. Keep a diary as you work."

**Assistant interpretation:** Create a docmgr ticket for the SSR feature, then produce a comprehensive intern-ready design doc covering current architecture, reference implementation analysis, gap analysis, proposed architecture with pseudocode, decision records, phased implementation plan, and testing strategy. Also maintain an implementation diary throughout.

**Inferred user intent:** Have a detailed, self-contained design document that a new team member could use to understand the full SSR system and implement it end-to-end, delivered via docmgr and reMarkable.

### What I did

- Read all key files in `publish-vault/backend/`: `server.go`, `runtime.go`, `api.go`, `vault.go`, `search.go`, `static.go`, `embed.go`, `embed_none.go`, `serve.go`
- Read all key files in `publish-vault/web/`: `App.tsx`, `main.tsx`, `store/store.ts`, `store/vaultApi.ts`, `store/uiSlice.ts`, `vite.config.ts`, `package.json`, `index.html`
- Read page components: `VaultLayout.tsx`, `NotePage.tsx`, `SearchPage.tsx`
- Read the complete glazed SSR implementation: `entry-server.tsx`, `entry-client.tsx`, `server.mjs`, `vite.config.ts`, `store.ts`, `serve.go` (SSR proxy section)
- Read the glazed SSR design doc: `DOCSCTL-SSR/design-doc/01-ssr-sidecar-analysis-and-implementation-guide.md`
- Created docmgr ticket `RETRO-SSR-009` with design doc and diary doc

### Why

A thorough evidence-first investigation ensures the design doc is grounded in the actual codebase structure, not assumptions. The two projects share a pattern but differ in routing library (Wouter vs React Router), URL scheme, store structure, and API shape.

### What worked

- The glazed SSR implementation is a clean reference that maps almost 1:1. Key differences were easy to identify.
- docmgr ticket and doc creation was smooth.

### What didn't work

- N/A — no failures during investigation.

### What I learned

1. **Wouter has no StaticRouter.** This is the biggest architectural difference. The SSR entry must bypass Wouter entirely and render page components directly based on URL parsing.
2. **Store singleton vs factory.** publish-vault's `store.ts` exports a singleton; glazed already has `makeStore()`. This is a simple refactor.
3. **The `vaultApi.util.upsertQueryData` pattern from glazed transfers directly.** RTK Query's cache manipulation API is the same regardless of routing library.

### What was tricky to build

The main design challenge was figuring out how to render page components on the server without Wouter. The solution is to have `entry-server.tsx` parse the URL and render `<NotePageSSR>`, `<SearchPageSSR>`, or `<HomeRedirectSSR>` directly. These SSR wrappers must be thin — they wrap the same page components but skip Wouter's `useLocation()` hook by accepting props instead.

### What warrants a second pair of eyes

- The SSR route parsing in `entry-server.tsx` must exactly match the client routes in `App.tsx`. If someone adds a new route, both must be updated.
- The `makeStore()` refactor must not break existing `store` singleton imports throughout the codebase.
- The Go SSR proxy must correctly exclude `/api/`, `/vault-assets/`, and `/assets/` paths.

### What should be done in the future

- Consider extracting the URL-to-route mapping into a shared module that both `App.tsx` and `entry-server.tsx` import, to reduce the risk of divergence.
- Add an integration test that spins up both Go server and Node sidecar to verify end-to-end SSR.

### Code review instructions

- Start with the design doc: `ttmp/2026/06/06/RETRO-SSR-009--add-ssr-sidecar-for-server-side-rendering-of-vault-pages/design-doc/01-ssr-sidecar-analysis-and-implementation-guide.md`
- Cross-reference with the glazed SSR implementation in `glazed/web/src/entry-server.tsx` and `glazed/web/server.mjs`
- Verify the decision records are sound (especially the Wouter SSR bypass decision)

### Technical details

Key files examined:
- `publish-vault/backend/internal/server/server.go` — Go server routing
- `publish-vault/backend/internal/api/api.go` — API endpoints
- `publish-vault/web/src/App.tsx` — Client routing with Wouter
- `publish-vault/web/src/store/vaultApi.ts` — RTK Query with upsertQueryData
- `glazed/pkg/help/server/serve.go` — Reference Go SSR proxy

## Step 2: Phase 1 — Store refactor + entry-client (2f349f1)

Refactored `store/store.ts` to export a `makeStore(preloadedState?)` factory function alongside the existing `store` singleton. This mirrors the glazed pattern exactly. Created `entry-client.tsx` with `hydrateRoot()` instead of `createRoot().render()`, reading `window.__PRELOADED_STATE__` from the SSR sidecar. Updated `vite.config.ts` with `rollupOptions.input` pointing at `index.html` (which now references `entry-client.tsx`), and added `ssr.noExternal` for React, Redux, RTK Query. Updated `web/index.html` to load `/src/entry-client.tsx` instead of `/src/main.tsx`.

### What I did
- Refactored `store/store.ts`: added `makeStore()` factory, kept `store` singleton, updated type exports
- Created `entry-client.tsx` with `hydrateRoot` + `__PRELOADED_STATE__` support
- Updated `vite.config.ts` with SSR build config and rollupOptions
- Updated `index.html` to reference `entry-client.tsx`
- Verified `pnpm build` succeeds (builds 3932 modules, ~14s)

### Why
The store factory is required so each SSR request gets its own Redux store. `hydrateRoot` reuses server-rendered DOM instead of replacing it.

### What worked
- The `makeStore()` pattern transferred cleanly from glazed.
- `hooks/redux.ts` needed zero changes — the types are still exported from the same location.
- `hydrateRoot` is compatible with an empty `<div id="root">` so local dev still works.

### What didn't work
- Had to `pnpm install` first because `node_modules` was missing in the workspace.

### What was tricky to build
- The `ssr.noExternal` list needed to include `use-sync-external-store` (the glue between React 19's `useSyncExternalStore` and Redux). Without it, the SSR build would leave it as an external import that Node can't resolve.

### What warrants a second pair of eyes
- The `entry-client.tsx` deletes `window.__PRELOADED_STATE__` after reading it. Make sure this doesn't cause issues with React strict mode double-rendering.

### What should be done in the future
- Consider adding `@types/react-dom` for `hydrateRoot` types if not already present.

### Code review instructions
- Compare `store/store.ts` with `glazed/web/src/store.ts` to verify the factory pattern matches
- Verify `entry-client.tsx` correctly reads and deletes `__PRELOADED_STATE__`
- Check that `main.tsx` is still used by `vite dev`

### Technical details
Commit: `2f349f1`
Files: `web/src/store/store.ts`, `web/src/entry-client.tsx` (new), `web/vite.config.ts`, `web/index.html`

## Step 3: Phase 2 — SSR entry point (06ef0e8)

Created `entry-server.tsx` with `renderApp(url, data)` that preloads RTK Query cache and renders React to HTML. Since Wouter doesn't support SSR (no `StaticRouter`), the entry parses the URL manually and renders page components directly. Created simplified SSR components (`SSRNotePage`, `SSRHomePage`, `SSRSearchPage`) that render the content without interactive features.

### What I did
- Created `entry-server.tsx` with `renderApp()`, `preloadCache()`, `parseRoute()`
- Implemented SSR-safe components: `SSRNotePage` (title, tags, HTML body, backlinks), `SSRHomePage` (note list), `SSRSearchPage` (placeholder)
- Added `build:ssr` and `build:all` scripts to `package.json`
- Verified `pnpm build:ssr` produces `dist/ssr/entry-server.js` (1.5MB, 57 modules)
- Verified `pnpm build:all` succeeds

### Why
The SSR entry is the bridge between the Node sidecar and React. It must produce HTML that matches what the client will hydrate.

### What worked
- Using `React.createElement` instead of JSX in the SSR components avoids any Babel/TSX issues in the SSR bundle.
- The `vaultApi.util.upsertQueryData` pattern preloads the cache so RTK Query hooks return data synchronously during `renderToString`.

### What didn't work
- N/A

### What was tricky to build
- Wouter's `useLocation()` hook crashes in Node (no `window.history`). The solution was to completely bypass Wouter in the SSR entry and render content components directly. This means the SSR route table must stay in sync with `App.tsx` — a maintenance burden but the simplest correct solution.
- Used `React.createElement` instead of JSX to ensure the SSR components don't accidentally pull in Wouter or other client-only modules.

### What warrants a second pair of eyes
- The SSR route parsing (`parseRoute`) must match the client routes in `App.tsx`. Currently 3 patterns: `/` → home, `/note/{slug}` → note, `/search` → search.
- The `SSRNotePage` renders `dangerouslySetInnerHTML` with the note's HTML. This is the same pattern used in the client-side `NoteRenderer`, so it's safe, but worth confirming the HTML is already sanitized by the Go parser.

### What should be done in the future
- Add a unit test for `renderApp` with mock data (task 2.6 — skipped for now, will add in a follow-up).

### Code review instructions
- Verify `parseRoute()` matches Wouter routes in `App.tsx`
- Check that `preloadCache` upserts match the RTK Query endpoint names

### Technical details
Commit: `06ef0e8`
Files: `web/src/entry-server.tsx` (new), `web/package.json`

## Step 4: Phase 3 — Node.js sidecar (00c68d2)

Created `server.mjs`, an Express server that receives page requests from the Go reverse proxy, pre-fetches data from the Go API, renders React via `renderApp()`, and assembles complete HTML with preloaded state, meta tags, JSON-LD, and noscript fallback.

### What I did
- Created `server.mjs` with Express
- Implemented URL parsing, data pre-fetching, HTML assembly
- Added JSON-LD (WebPage, BreadcrumbList) and Open Graph meta tags
- Added `<noscript>` fallback with note list for non-JS agents
- Added health check endpoint at `/health`

### Why
This is the core of the SSR sidecar. It bridges the Go server's HTTP responses with React's `renderToString`.

### What worked
- Express was already a dependency in `package.json` so no new deps needed.
- The HTML template replacement pattern (regex on `<div id="root">`) is proven from the glazed sidecar.

### What didn't work
- N/A

### What was tricky to build
- The `serializeForInlineScript` function must escape `<`, `>`, `&`, and Unicode line separators to prevent XSS via `</script>` injection in the serialized state.

### What warrants a second pair of eyes
- The HTML template regex `/<div id="root">([\s\S]*?)<\/div>/` could theoretically match content within the React HTML if it contains a nested `<div id="root">`. This is unlikely but worth noting.

### What should be done in the future
- Consider caching the `index.html` template reload on file change (for now it loads once at startup which is fine for containers).

### Code review instructions
- Compare `server.mjs` with `glazed/web/server.mjs` to verify the pattern matches
- Verify the data pre-fetching matches the RTK Query endpoints used in `entry-server.tsx`

### Technical details
Commit: `00c68d2`
Files: `web/server.mjs` (new)

## Step 5: Phase 4 — Go server SSR proxy (3ad2d4d + 102fef8)

Added `--ssr-url` flag to the Go serve command and implemented `newSSRProxy()` in `server.go`. The proxy reverse-proxies page requests to the SSR sidecar, forwarding relevant headers and falling back to the SPA handler if the sidecar is unavailable.

### What I did
- Added `SSRURL` field to `server.Config` struct
- Added `--ssr-url` flag to `serve.go` with help text
- Implemented `newSSRProxy()` with: request forwarding, header passthrough, SPA fallback on error/5xx, streaming response body
- Added 5 test cases: proxy, 500 fallback, connection-refused fallback, invalid URL fallback, header forwarding
- All tests pass

### Why
The Go server needs to selectively proxy page requests to the Node sidecar while continuing to handle API, vault-assets, and static asset requests itself.

### What worked
- The proxy pattern transferred directly from `glazed/pkg/help/server/serve.go`.
- All 5 test cases pass cleanly.

### What didn't work
- The Go workspace (`go.work`) has a version mismatch with the glazed module. Had to use `GOWORK=off` for building/testing.

### What was tricky to build
- The workspace issue is pre-existing and unrelated to SSR changes. The backend builds and tests fine with `GOWORK=off`.

### What warrants a second pair of eyes
- The proxy only forwards `Accept`, `Accept-Language`, `User-Agent`, and `Cookie` headers. This is intentional (we don't want to forward `Host` or `Connection`), but worth verifying.

### What should be done in the future
- Fix the `go.work` version mismatch to allow workspace-level builds.

### Code review instructions
- Compare `newSSRProxy()` with `glazed/pkg/help/server/serve.go:newSSRProxy()`
- Run `GOWORK=off go test ./internal/server/ -v` to verify all tests

### Technical details
Commits: `3ad2d4d` (implementation), `102fef8` (tests)
Files: `backend/internal/server/server.go`, `backend/cmd/retro-obsidian-publish/commands/serve/serve.go`, `backend/internal/server/ssr_proxy_test.go` (new)

## Step 6: Phase 5 — Docker + deployment (17dbb02)

Created `ssr.Dockerfile` for the Node.js sidecar and updated `docker-compose.yml` to include the sidecar service with proper networking.

### What I did
- Created `web/ssr.Dockerfile`: Node 22 Alpine, pnpm install, `build:all`, runs `server.mjs`
- Updated `docker-compose.yml`: added `ssr` service, wired Go server with `--ssr-url http://ssr:8089`, set `API_BASE=http://app:8080`

### Why
The sidecar needs its own container image with Node.js to run the Express SSR server.

### What worked
- Docker Compose networking lets the Go server (`app`) reach the sidecar (`ssr`) via the service name.

### What didn't work
- N/A

### What was tricky to build
- The Go server's `command` override in docker-compose must include all the original flags plus `--ssr-url`. The default `CMD` in the Go Dockerfile is `serve --port 8080 --serve-web`, so the compose command must replicate those plus add `--ssr-url`.

### What warrants a second pair of eyes
- Verify the Docker Compose networking: `http://ssr:8089` from the Go container, `http://app:8080` from the SSR container.

### What should be done in the future
- End-to-end test with `docker compose up` (Phase 6 manual verification).

### Code review instructions
- Check that the SSR Dockerfile's `build:all` command produces both `dist/` and `dist/ssr/`
- Verify docker-compose networking between services

### Technical details
Commit: `17dbb02`
Files: `web/ssr.Dockerfile` (new), `docker-compose.yml`

## Step 7: Phase 6 — End-to-end verification with devctl (b30da32)

Ran the full stack via `devctl up --profile example` with 3 services (backend, web, ssr). Discovered and fixed three bugs during manual testing.

### What I did
- Updated devctl plugin to add SSR sidecar as third service with `depends_on: ["backend"]`
- Started devctl and verified all 3 services come up healthy
- Tested SSR output directly (`curl :8089/`) — confirmed full HTML with preloaded state, meta tags, JSON-LD
- Tested through Go proxy (`curl :8081/note/zettelkasten-method`) — confirmed correct title, meta tags, rendered content
- Tested SPA fallback (stopped SSR service) — confirmed Go server falls back to `index.html`
- Opened browser and verified rendering with zero console errors

### Why
Phase 6 is the manual verification that everything works end-to-end.

### What worked
- The SSR sidecar correctly fetches data from the Go API and renders React to HTML
- JSON-LD structured data (WebPage + BreadcrumbList) is correctly generated
- Open Graph meta tags are populated with note titles and excerpts
- The `<noscript>` fallback provides text content for non-JS agents
- SPA fallback works correctly when the sidecar is down

### What didn't work
1. **Express 4 wildcard syntax.** `server.mjs` used `{*path}` (Express 5 syntax) but the project uses Express 4. Fixed by changing to `"*"`.
2. **Static assets served as text/html.** The Go SSR proxy caught `/assets/*` requests and forwarded them to the sidecar, which returned HTML instead of JS/CSS. Fixed by adding `r.PathPrefix("/assets/").Handler(spaHandler)` before the SSR proxy in `server.go`.
3. **React hydration mismatch (#418).** The SSR renders simplified components (SSRNotePage) while the client hydrates with the full app (VaultLayout + Wouter routing). The DOM trees don't match, causing React error #418. Fixed by switching from `hydrateRoot` to `createRoot` in `entry-client.tsx`, clearing the SSR content before mounting. The SSR HTML still serves crawlers/agents; the client just replaces it with the full interactive app.

### What was tricky to build
- The hydration mismatch was subtle. The SSR components intentionally render simplified content (no Wouter, no sidebar, no resizable panels). Using `hydrateRoot` would require the SSR and client DOM to be byte-identical, which defeats the purpose of simplified SSR. The `createRoot` approach is the pragmatic choice — SSR HTML for bots, client app for users.
- The devctl `restart` and `start` commands re-resolve ports, breaking the Go→SSR URL mapping. Must use `devctl down && devctl up` instead of `restart`.

### What warrants a second pair of eyes
- The `createRoot` approach means the user briefly sees the SSR content flash, then it's replaced. This is acceptable for SEO/agent use but worth noting.
- The `root.textContent = ""` line clears the SSR content. Make sure this doesn't cause a flash of unstyled content.

### What should be done in the future
- Consider using React 19's streaming SSR with `renderToReadableStream` for progressive loading.
- Consider matching SSR and client component trees more closely to enable true hydration.

### Code review instructions
- Run `devctl up --profile example` and verify all 3 services start
- `curl http://localhost:8081/note/zettelkasten-method | grep "<title>"` should show the note title
- Open in browser, check DevTools console for zero errors

### Technical details
Commit: `b30da32`
Files: `web/src/entry-client.tsx`, `web/server.mjs`, `backend/internal/server/server.go`, `plugins/retro-obsidian-publish.py`

## Step 8: Phase 7 — a14y Markdown Mirrors and Agent Discovery (695aec5, 92af9f7, fddbc10, b2e2c92, b0606d3)

This step applied the a14y workflow after SSR was working. The baseline audit against `http://localhost:8081/` scored 62/100 and showed that SSR solved rendered HTML visibility but the site still lacked the machine-facing discovery and Markdown mirror layer agents expect.

### Prompt Context

**User prompt (verbatim):** "Now apply a14y to the site, now that we have SSR."

**Assistant interpretation:** Run the a14y audit against the SSR-enabled local site, present the gated fix plan, implement approved fixes, keep ticket documentation current, and rerun the audit for score delta.

**Inferred user intent:** Improve agent readability and discoverability now that the site returns useful SSR HTML.

### What I did
- Started the SSR-enabled devctl stack and audited `http://localhost:8081/`.
- Baseline command: `npx -y a14y check http://localhost:8081/ --mode site --output agent-prompt --max-pages 200`.
- Baseline score: `62/100` on scorecard `0.2.0`.
- Wrote `design-doc/02-markdown-mirror-and-a14y-implementation-guide.md` before runtime changes.
- Added Phase 7 tasks to `tasks.md`.
- Saved a14y config/history in `scripts/01-a14y-config.md` per user request instead of `AGENTS.md` or `a14y.md`.
- Implemented Go endpoints for `/AGENTS.md`, `/llms.txt`, `/sitemap.md`, `/sitemap.xml`, `/index.md`, `/note/{slug}.md`, and `Accept: text/markdown` for `/` and `/note/{slug}`.
- Added canonical `Link` headers, required Markdown frontmatter, and `## Sitemap` sections.
- Updated SSR HTML to advertise Markdown mirrors with `<link rel="alternate" type="text/markdown">` and response `Link` headers.
- Added `dateModified` JSON-LD and a hidden glossary/agent guide link.
- Added Go tests for discovery endpoints, sitemap XML, Markdown mirrors, and content negotiation.
- Fixed unresolved wiki links so missing notes do not become crawlable `/note/...` URLs.

### Why
SSR made the HTML useful, but a14y expects agents to discover and consume the site without executing JavaScript. Markdown mirrors and sitemap resources provide that stable text-first surface.

### What worked
- Adding Go-owned Markdown endpoints resolved nearly all Markdown mirror failures.
- The score improved from `62/100` to `99/100`.
- The final audit crawled 16 pages and passed 187 checks.

### What didn't work
- First re-audit scored `97/100`: unresolved wiki link `[[Gettier Problem]]` created crawlable `/note/gettier-problem` with no Markdown mirror.
- Second re-audit scored `98/100`: returning a 404 for the missing note removed Markdown mirror failures but still left the broken page as a crawled URL with HTTP status/content-type failures.
- Final fix: unresolved wiki links are converted to same-page `#unresolved-...` anchors, so a14y no longer crawls them as note pages.

### What I learned
- a14y treats linked missing pages as audit surface if they return HTML, even if the app visually says “not found”. Broken wiki links therefore matter for agent readability.
- `llms.txt` links are expected to point mostly to `.md`/`.mdx` resources; HTML note links in `llms.txt` lower the score.
- Content negotiation and `.md` mirror URLs should share the same rendering code to avoid metadata drift.

### What was tricky to build
- The hard part was distinguishing real published note pages from unresolved Obsidian wiki-link targets. The parser originally generated `/note/{slug}` for every wiki link before the vault knew whether the target existed. The fix was to let the vault resolver return an empty string for unresolved targets and have `ReplaceWikiLinksString` convert those anchors to same-page unresolved fragments.

### What warrants a second pair of eyes
- The Markdown mirror currently derives content from rendered HTML because `vault.Note` does not preserve raw Markdown. This is good enough for a14y and agent reading, but a future raw Markdown field would be cleaner.
- The remaining score issue is `html.headings` on `/`, intentionally skipped in the approved a14y plan.

### What should be done in the future
- Preserve raw Markdown in `vault.Note` so `.md` mirrors can expose the original note body instead of HTML in fenced code.
- Optionally add visually-hidden home-page headings if a perfect 100/100 score is desired.

### Code review instructions
- Start with `backend/internal/server/agent_markdown.go` for endpoint contracts and Markdown rendering.
- Then review `web/server.mjs` for SSR alternate-link/header injection and missing-note 404 behavior.
- Review `backend/internal/parser/parser.go` and `backend/internal/vault/vault.go` for unresolved wiki-link handling.
- Validate with:
  - `cd backend && GOWORK=off go test ./... -count=1`
  - `cd web && pnpm build:all`
  - `devctl down && devctl up --profile example`
  - `npx -y a14y check http://localhost:8081/ --mode site --output agent-prompt --max-pages 200`

### Technical details
- Code commits: `695aec5`, `92af9f7`, `fddbc10`, `b2e2c92`, `b0606d3`.
- Final a14y command: `npx -y a14y check http://localhost:8081/ --mode site --output agent-prompt --max-pages 200`.
- Final a14y result: `99/100`, one remaining failure: `html.headings` on `/`.
- Final JSON audit saved outside the repo at `/tmp/retro-a14y-final2.json`.
