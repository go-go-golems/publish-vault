---
Title: Investigation diary
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
    - Path: scripts/smoke-ssr-hydration.mjs
      Note: Phase A live SSR hydration smoke test implementation
    - Path: web/package.json
      Note: smoke:ssr script and Playwright dependency
    - Path: web/server.mjs
      Note: SSR_DEBUG_RESOLUTION dependency path/version diagnostics
ExternalSources: []
Summary: Chronological investigation log for page title, SSR sidebar, and HTML layout issues.
LastUpdated: 2026-06-07T12:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Track the investigation and planning for fixing page titles, SSR sidebar/SEO, and HTML menu/content layout in Retro Obsidian Publish.

## Step 1: Evidence Gathering

### User Prompt

Investigate page title issues ("current" shown instead of note title), SSR sidecar not rendering sidebars affecting SEO, and improper HTML layout of menu and content. Create a detailed analysis/design/implementation guide for an intern.

### Assistant Interpretation

The user reports that the browser tab title shows "current" (the vault directory symlink name) instead of the actual page/note title. They also express concern that the SSR sidecar does not render sidebars, potentially harming SEO. They want the menu and content properly structured in the HTML. Additionally, they want a comprehensive guide for an intern to understand the full system.

### Inferred User Intent

The user wants a thorough audit of the Retro Obsidian Publish rendering pipeline — from Go server → SSR sidecar → React SPA — to diagnose title, sidebar, and SEO issues, then produce an intern-grade reference document stored in a docmgr ticket and uploaded to reMarkable.

### Commit (code): —

### What I did
- Explored the repository structure: `publish-vault/` with Go backend, `web/` with React+Vite SPA, and SSR sidecar
- Read the Go server (`internal/server/server.go`), API (`internal/api/api.go`), vault (`internal/vault/vault.go`), parser (`internal/parser/parser.go`), and web handler (`internal/web/static.go`)
- Read the SSR entry point (`web/src/entry-server.tsx`), the Express server (`web/server.mjs`), the client entry (`web/src/entry-client.tsx`), and the React app (`web/src/App.tsx`)
- Read the layout components (`VaultLayout.tsx`, `NotePage.tsx`, `Sidebar.tsx`)
- Inspected the live site at `parc.yolo.scapegoat.dev` using Playwright
- Verified SSR output via `curl` showing correct `<title>`, `<meta>` tags, noscript blocks
- Confirmed `/api/config` returns `"pageTitle":"current"` from the vault directory symlink name
- Created ticket `RETRO-SEO-009` with docmgr

### What worked
- SSR sidecar correctly renders note titles in the `<title>` tag (e.g., "Research Institute Guidelines — current")
- Meta tags (description, og:title, og:description, canonical, JSON-LD, breadcrumb) are present in SSR output
- The Go server correctly proxies page requests to the SSR sidecar, with SPA fallback on failure
- React client correctly sets `document.title` from config

### What didn't work
- The vault directory is named "current" (git-sync symlink name), so `pageTitle` falls back to "current" everywhere
- Client-side `document.title` effect only uses config values (vaultName/pageTitle), not the current note title
- The SSR `SSRNotePage` intentionally omits the sidebar for simplicity — content-only focus for SEO
- The client-side `entry-client.tsx` clears SSR content (`root.textContent = ""`) before mounting React, so the client app never benefits from the server-rendered title

### What I learned
- The title "current" is not a code bug — it's a deployment configuration issue. The vault directory symlink is named "current".
- SSR does render page titles correctly. The problem is the fallback value.
- The React client's `useEffect` in `App.tsx` overwrites the SSR title with only config values, losing the note-specific title context.
- SSR intentionally renders minimal content (no sidebar) as a performance and SEO tradeoff. The client hydrates with the full interactive app.

### What was tricky to build
- The SSR architecture uses a "render-to-string" approach where server.mjs pre-fetches all data from the Go API, populates RTK Query's cache via `upsertQueryData`, then calls `renderToString()`. The SSR components (`SSRNotePage`, `SSRHomePage`) are hand-built React.createElement calls, not the real component tree, because Wouter (the router) has no `StaticRouter` for SSR.
- Client hydration is not done via `hydrateRoot` but via `createRoot` with the SSR content cleared, because the SSR components don't match the full client component tree. This avoids React hydration mismatch errors.

### What warrants a second pair of eyes
- Whether adding sidebar content to the SSR output is worth the complexity, or if the current content-only approach is sufficient for SEO
- Whether the client-side title effect should be enhanced to use note-specific titles
- The tradeoff between having the SSR render a full layout vs. minimal content

### What should be done in the future
- Add a `pageTitle` flag to the deploy config (devctl profile or k8s env var)
- Enhance the client-side title effect to include the current note's title in the browser tab
- Consider adding structured navigation breadcrumbs to the SSR output for SEO

### Code review instructions
- Start with `server.mjs` lines ~120-140 for title construction
- Review `App.tsx` `Router` component's `useEffect` for document.title
- Check `VaultLayout.tsx` for menu bar structure
- Review `entry-client.tsx` for the SSR-clearing behavior

### Technical details

**Title flow (current):**
```
Deployment (vault dir = "current")
  → Go server derives vaultName="current", pageTitle="current"
  → /api/config returns {vaultName:"current", pageTitle:"current"}
  → SSR server.mjs reads config, sets title = note.title + " — " + vaultName
    → HTML: <title>Note Title — current</title>  ✓
  → React client mounts, Router useEffect sets document.title = config.pageTitle
    → Title becomes just "current"  ✗
```

**Title flow (desired):**
```
Deployment (vault dir = "current", pageTitle = "PARC")
  → /api/config returns {vaultName:"current", pageTitle:"PARC"}
  → SSR: <title>Note Title — PARC</title>  ✓
  → React client: document.title = noteTitle + " — PARC"  ✓
```

**SSR sidebar tradeoff:**
```
SSR output:
  - ✅ Note title (h1)
  - ✅ Tags (spans)
  - ✅ Full note HTML body (note-prose)
  - ✅ Backlinks section
  - ✅ Meta tags (og:title, description, JSON-LD, breadcrumbs)
  - ✅ Noscript fallback with navigation
  - ❌ No sidebar navigation

SPA output:
  - ✅ Everything above
  - ✅ Sidebar with file tree, search
  - ✅ Resizable panels
  - ✅ Interactive note renderer
```

## Step 2: React Router Hydration Cleanup Plan

This step changes the ticket direction from small SEO patches to a larger cleanup: replacing Wouter with React Router and consolidating SSR and client rendering onto one hydratable React tree. I captured the implementation strategy before touching code so the migration has explicit phases, risk controls, and validation commands.

The key decision is to keep the existing RTK Query data model and Express sidecar prefetching, while changing the routing/rendering layer. The cleanup should remove duplicated SSR page components, stop clearing server-rendered DOM, and eventually allow `hydrateRoot()` to attach to the server-rendered app.

### Prompt Context

**User prompt (verbatim):** "Let's rip off the bandaid, and first tackle the refactor and use react router and reduce the surface to be handled for SSR by consolidating it all. 

Analyze the situation once more, and write a second design/implementation guide to tackle the cleanup, and upload to remarkable as well. Then add a detailed list of tasks to the ticket, and build them one by one, committinga t appropriate intervals, and keeping a diary"

**Assistant interpretation:** Create a second guide for the React Router + full hydration refactor, upload it, expand the ticket task list, then begin implementation in focused commits while maintaining diary entries.

**Inferred user intent:** The user wants to stop carrying duplicate SSR/client trees and perform the foundational router/hydration cleanup before smaller SEO fixes.

**Commit (code):** pending — docs guide created before code changes.

### What I did
- Added design doc `design-doc/02-react-router-ssr-hydration-cleanup-implementation-guide.md`.
- Rewrote `tasks.md` into Track A (original SEO/title cleanup) and Track B (React Router + full hydration refactor).
- Documented phased commits, risks, validation commands, and decision records.

### Why
- Full hydration is a cross-cutting refactor; without a plan it is easy to mix dependency, router, SSR, hydration, and title changes into one unreviewable patch.
- The guide defines the invariant: server and client must render the same component tree for the same URL and preloaded state.

### What worked
- The Wouter usage surface is small and well-contained.
- The Redux store factory and RTK Query preloading are already compatible with SSR-per-request rendering.
- The Express sidecar can keep owning metadata while React Router cleanup focuses on body/root hydration.

### What didn't work
- No code validation has run yet for this new plan; this step is documentation and task setup only.

### What I learned
- The main blocker is component-tree divergence, not data fetching.
- React Router can replace both Wouter route declarations and the custom server-side route rendering approach.

### What was tricky to build
- The tricky part is sequencing. If `hydrateRoot()` lands before the app tree is deterministic, hydration warnings will obscure the real cause. The guide therefore separates dependency installation, client route migration, server tree consolidation, hydrateRoot switch, deterministic render cleanup, and title cleanup.

### What warrants a second pair of eyes
- Whether to keep `server.mjs` route parsing for metadata/prefetch or to extract a shared route matcher later.
- Whether live UI elements such as the clock should be removed entirely or converted to hydration-safe placeholders.

### What should be done in the future
- Start with a docs/safety commit.
- Add React Router dependency.
- Migrate client routes before changing SSR.

### Code review instructions
- Start with `design-doc/02-react-router-ssr-hydration-cleanup-implementation-guide.md`.
- Review `tasks.md` Track B for the implementation sequence.
- Validate docs with `docmgr doctor --ticket RETRO-SEO-009 --stale-after 30`.

### Technical details
- Target server tree: `<Provider><StaticRouter location={url}><AppRoutes /></StaticRouter></Provider>`.
- Target client tree: `<Provider><BrowserRouter><AppRoutes /></BrowserRouter></Provider>`.
- Target hydration call: `hydrateRoot(root, <...same app shell... />)`.

## Step 3: React Router Migration and Hydratable SSR Tree

This step implemented the core cleanup planned in Step 2. The routing layer now uses React Router, the server renders the same `AppRoutes` component tree that the browser hydrates, and the client entry uses `hydrateRoot()` instead of clearing the server-rendered DOM.

The implementation deliberately kept the existing RTK Query cache preloading and Express sidecar metadata pipeline. That kept the refactor focused on routing, render-tree convergence, hydration, deterministic output, and title alignment rather than changing data fetching or head management at the same time.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Build the React Router + full hydration refactor in focused commits after writing the second guide.

**Inferred user intent:** Consolidate SSR and client rendering to reduce maintenance surface and make future SEO/layout work happen in one app tree.

**Commit (code):** 55dbc348b770318be90048dde692b2581161eb55 — "web: add React Router dependency"

**Commit (code):** 6f76a554710a5d3e93df28185831d8bea73c2587 — "web: migrate client routing to React Router"

**Commit (code):** 5ee46f9c4dc17e1535af5438319bb904a5a8c2d4 — "web: render real app tree during SSR hydration"

**Commit (code):** a4e89cb9db9a6834c02c26f9e755ec0f423da2c6 — "web: make layout clock hydration safe"

**Commit (code):** 00600968d7a2a3e513d1f5146a5b0e0155f2b7bb — "web: align SSR and client page titles"

### What I did
- Added `react-router-dom` and direct `react-router` dependency.
- Migrated Wouter route declarations in `App.tsx` to React Router `Routes`/`Route`.
- Migrated Wouter navigation in `VaultLayout`, `NotePage`, `SearchPage`, and `NotFound` to React Router navigation.
- Exported `AppRoutes` so the server and browser can wrap the same route tree in different routers.
- Replaced the custom SSR-only `SSRNotePage`/`SSRHomePage`/`SSRSearchPage` tree with real `AppRoutes` rendered inside React Router `StaticRouter`.
- Replaced `createRoot()` plus `root.textContent = ""` with `hydrateRoot()` in `entry-client.tsx`.
- Removed Wouter and its stale patch file.
- Added home-note prefetching in `server.mjs` so the real home route can SSR the same selected note that `HomeRedirect` renders.
- Added hydration-safe clock rendering to avoid first-render time mismatches.
- Aligned SSR and client titles to use `note.title — siteTitle` where `siteTitle = pageTitle || vaultName`.

### Why
- Full hydration requires the server and client to render identical component trees for the same URL and preloaded state.
- The previous SSR-only page components produced useful crawler HTML but duplicated app structure and forced client remounting.
- React Router provides a server-side router (`StaticRouter`) and browser router (`BrowserRouter`) for the same route declarations.

### What worked
- `pnpm --dir web check` passed after each implementation phase.
- `pnpm --dir web exec vitest run src/entry-server.test.tsx` passed after updating tests for real-app SSR.
- `pnpm --dir web build:all` passed after SSR consolidation, hydration, deterministic clock, and title updates.
- `GOWORK=off go test ./...` passed after the frontend changes.

### What didn't work
- Initial SSR consolidation tried `import { StaticRouter } from "react-router-dom/server"`, but React Router 7.17.0 does not export that path.
  Exact failure:
  `src/entry-server.tsx(10,30): error TS2307: Cannot find module 'react-router-dom/server' or its corresponding type declarations.`
- Fix: add direct `react-router` dependency and import `StaticRouter` from `react-router`.
- After rendering the real app tree, old SSR tests failed because they expected the old simplified home and missing-note behavior. The real app route renders the selected home note only when the sidecar preloads it, and missing note routes are 404ed by `server.mjs` before rendering.

### What I learned
- RTK Query preloading was already robust; the biggest missing piece was routing/render-tree convergence.
- The home route is not a standalone SSR home page anymore: it runs the actual `HomeRedirect` selection logic and then renders `NotePage`, so the sidecar must prefetch the chosen note.
- React Router v7's server API differs from the older `react-router-dom/server` import path.

### What was tricky to build
- The subtle part was keeping home SSR useful after removing the special SSR-only home page. Without home-note prefetching, `/` would render a loading note state because `HomeRedirect` picks a slug from `listNotes`, then `NotePage` needs `getNote(slug)` data. I added a sidecar `chooseHomeSlug()` mirror and taught `renderApp()` to use the home note slug as the RTK Query cache key when the URL is `/`.
- Hydration safety required identifying non-deterministic render output. The menu-bar clock used `new Date()` during render, which would differ between server and client. I replaced it with a component that renders `--:--` before mount and only reads the clock after hydration.

### What warrants a second pair of eyes
- `server.mjs` now duplicates `chooseHomeSlug()` from `App.tsx`; this is acceptable for this phase but should eventually be shared or moved into a tiny pure module consumed by both SSR sidecar and React app.
- Manual browser hydration-console validation has not yet been performed in this step.
- The real app SSR output is much larger than the previous simplified SSR output; review payload size and crawler tradeoffs.

### What should be done in the future
- Run a local SSR sidecar/browser test and confirm there are no hydration warnings.
- Consider extracting shared route/home-selection helpers.
- Consider later moving head metadata into a React-side head manager once hydration is stable.

### Code review instructions
- Start with `web/src/App.tsx` to see the new `AppRoutes` export and route table.
- Review `web/src/entry-server.tsx` next to confirm real app SSR under `StaticRouter`.
- Review `web/src/entry-client.tsx` to confirm `hydrateRoot()` and no DOM clearing.
- Review `web/server.mjs` for home-note prefetch and title formatting.
- Review `web/src/components/pages/VaultLayout/VaultLayout.tsx` for `HydrationSafeClock`.
- Validate with:
  - `pnpm --dir web check`
  - `pnpm --dir web exec vitest run src/entry-server.test.tsx`
  - `pnpm --dir web build:all`
  - `GOWORK=off go test ./...`

### Technical details
- React Router import split:
  - Browser: `BrowserRouter`, `Routes`, `Route`, `useNavigate`, `useParams`, `useLocation` from `react-router-dom`.
  - Server: `StaticRouter` from `react-router`.
- The SSR sidecar still owns `<head>` metadata injection; this refactor only hydrates the root app tree.
- The browser no longer clears SSR content before rendering.

## Step 4: Hard Cutover Cleanup and Live Hydration Test

This step performed the live local test that was still pending after the React Router migration. The initial browser run exposed a real production-class failure: the Go server was falling back to the SPA shell because the SSR sidecar was returning 500. After fixing SSR bundling and a local path issue, the live test passed with zero browser console warnings/errors.

The cleanup also removed the final stale client entry (`web/src/main.tsx`) and made first-render state/date output deterministic so SSR and browser hydration agree.

### Prompt Context

**User prompt (verbatim):** "clean up any legacy / hard cut over. run live test"

**Assistant interpretation:** Remove remaining legacy cutover artifacts, run local backend + SSR + browser validation, and fix anything that prevents real hydration.

**Inferred user intent:** Ensure the refactor is not just type/build-clean, but actually works as a live SSR-hydrated site with no fallback and no hydration warnings.

**Commit (code):** 685f0ba098bf069ac616aa6f62e94b52e851152f — "web: harden SSR hydration cutover"

### What I did
- Removed stale `web/src/main.tsx`, which still used `createRoot()` but was no longer referenced by `web/index.html`.
- Made `uiSlice` initial state deterministic (`sidebarOpen/rightPanelOpen = true`) instead of reading `window.innerWidth` at module initialization.
- Made `FrontmatterPanel` date formatting use UTC so Node SSR and browser hydration do not disagree by timezone.
- Changed `web/vite.config.ts` SSR config to `noExternal: true` so React, React Router, and React component libraries share one bundled React instance in the SSR bundle.
- Made `server.mjs` resolve `dist/index.html` relative to `import.meta.url` instead of process CWD.
- Let `NotePage` own the `/` home-note title by preventing the app shell title effect from overwriting `/`.
- Started local backend and SSR sidecar:
  - backend: `GOWORK=off go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --vault-name TestVault --page-title "Test Vault" --port 18080 --ssr-url http://127.0.0.1:18089 --watch=false`
  - SSR: `SSR_PORT=18089 API_BASE=http://127.0.0.1:18080 BASE_URL=http://127.0.0.1:18080 node web/server.mjs`
- Tested with Playwright:
  - `/` title: `Index — Test Vault`
  - `/note/index` title: `Index — Test Vault`
  - sidebar navigation to `/note/philosophy/epistemology` title: `Epistemology — Test Vault`
  - browser console: `0` errors, `0` warnings.

### Why
- Build/test success alone did not guarantee the SSR sidecar could actually render in a live Node process.
- The Go proxy silently falls back to SPA on SSR 500s, so live testing was necessary to catch sidecar failures.

### What worked
- Raw SSR response now comes through the Go proxy with Express headers and a populated `<div id="root">...`.
- Browser hydration no longer emits React #418.
- Module script loading uses the hashed Vite asset (`/assets/main-...js`) rather than fallback `/assets/index.js`.
- Direct note loads and client-side sidebar navigation both update URL/title correctly.

### What didn't work
- First live test showed React #418. Investigation revealed raw `/` response was the empty SPA shell because Go fell back from SSR.
- SSR sidecar logs showed invalid hook calls due duplicate React instances:
  - first from `react-router` externalization (`Cannot read properties of null (reading 'useContext')`)
  - then from `react-resizable-panels` externalization (`Cannot read properties of null (reading 'useId')`)
- Fix: use `ssr.noExternal: true` so all React-using frontend dependencies are bundled into one SSR module graph.
- Another live test showed module MIME error for `/assets/index.js`; `server.mjs` had read `./dist/index.html` relative to repo root and used its fallback shell. Fix: resolve `dist/index.html` relative to `server.mjs`.

### What I learned
- The SSR proxy fallback can mask SSR-side errors by returning a working SPA shell; always inspect headers/logs/raw root HTML during live tests.
- With Vite SSR, bundling React but externalizing React-using libraries creates duplicate React instances. For this application, full bundling is safer for correctness.
- Head title behavior for `/` needs to account for the fact that `/` renders a selected home note, not a separate home page.

### What was tricky to build
- The first symptom looked like a hydration mismatch, but the root cause was that SSR was not actually being served. The key debugging step was comparing raw `curl http://127.0.0.1:18080/` output, which showed `<div id="root"></div>` instead of SSR content. That led to backend logs showing `SSR proxy unavailable, falling back to SPA`, then sidecar logs showing invalid hook calls.
- The second symptom looked like a static asset issue, but it was caused by `server.mjs` using a CWD-relative `./dist/index.html` path and therefore falling back to a minimal shell. Resolving relative to `import.meta.url` fixed local and Docker behavior consistently.

### What warrants a second pair of eyes
- `ssr.noExternal: true` increases SSR bundle size substantially. It fixes correctness, but reviewer should confirm this is acceptable for the sidecar image/runtime.
- `server.mjs` still duplicates `chooseHomeSlug()` from `App.tsx`; this is now live-tested but remains a maintainability follow-up.

### What should be done in the future
- Consider extracting shared home-route selection logic into a pure module used by both `App.tsx` and `server.mjs`.
- Consider adding an automated smoke script that starts backend+SSR and asserts raw SSR root is populated and browser console is clean.

### Code review instructions
- Review `web/vite.config.ts` first for `ssr.noExternal: true` and the rationale.
- Review `web/server.mjs` for `WEB_DIR` / `import.meta.url` path resolution.
- Review `web/src/store/uiSlice.ts` and `FrontmatterPanel.tsx` for deterministic first-render behavior.
- Review `web/src/App.tsx` title ownership for `/`.
- Validate with:
  - `pnpm --dir web check`
  - `pnpm --dir web exec vitest run src/entry-server.test.tsx`
  - `pnpm --dir web build:all`
  - `GOWORK=off go test ./...`
  - local backend+SSR Playwright test checking console messages.

### Technical details
- Raw SSR header after fix includes `X-Powered-By: Express` and non-empty root markup.
- Browser console after `/`, `/note/index`, and sidebar navigation: `0` errors, `0` warnings.

## Step 5: Obsidian Vault Report

Wrote a comprehensive textbook-style project report for the Obsidian vault at `go-go-parc` documenting the entire React Router SSR hydration cleanup.

**Prompt context:**
- User: "Write a detailed project report for the obsidiani vault about our progress so far, using a textbook writing style, commit and push the vault once done."
- Assistant: Wrote the report covering the problem statement, solution architecture, six implementation phases, live debugging of two production-class failures, files changed, key learnings, and future work.

**What I did:**
- Wrote `publish-vault-ssr-implementation-report.md` (696 lines) in the vault at `~/code/wesen/go-go-golems/go-go-parc`
- Commit: `c92c049` — "docs: publish-vault SSR hydration implementation report"
- Pushed to remote `main`

**Report structure:**
1. Introduction — three-layer architecture overview
2. The Problem — four concrete manifestations of the divergent architecture
3. The Solution — architecture diagrams, route architecture, data loading, title architecture
4. Implementation Phases — six detailed phases with code examples
5. Live Testing — Bug 1 (duplicate React instances) and Bug 2 (MIME type error) with root cause analysis
6. Files Changed — comprehensive table
7. Key Learnings — five numbered insights
8. Future Work — four actionable follow-ups
9. Summary

Commit and push successful.

## Step 6: Add Live SSR Hydration Smoke Test Guardrail

This step turned the manual backend + SSR sidecar + browser validation into a reusable smoke test. The test is intentionally end-to-end: it builds the web client and SSR bundles, starts the Node sidecar and Go server on random local ports, verifies the raw HTML is actually coming from SSR instead of SPA fallback, then opens Chromium and checks that hydration produces no console warnings or errors.

This guardrail is the prerequisite for safely changing the SSR dependency model. Without it, reducing `ssr.noExternal` could silently reintroduce duplicate React or fallback-to-SPA behavior while ordinary type checks and unit tests still pass.

### Prompt Context

**User prompt (verbatim):** "Ok, create new tasks and phases (split into tasks) to the ticket. Then work on those one by one, keeping a detailed diary as you work, committing at appropriate intervals."

**Assistant interpretation:** Add a phased follow-up plan to RETRO-SEO-009, then begin implementing the phases sequentially with diary entries, validation, and focused commits.

**Inferred user intent:** Replace the blunt `ssr.noExternal: true` fix with a cleaner SSR dependency model, but only after adding enough automated live coverage to catch hydration and duplicate-React regressions.

**Commit (code):** cbd3f5635549a0880d5164f21fb7b8fd404e727d — "test: add live SSR hydration smoke test"

### What I did
- Added ticket tasks 25–29 for the follow-up phases:
  - Phase A: live SSR smoke test
  - Phase B: SSR dependency-resolution diagnostics
  - Phase C: principled SSR dependency model
  - Phase D: Docker/runtime dependency packaging
  - Phase E: bundle-size measurement and documentation
- Added `playwright` as a web dev dependency so the smoke test is project-local rather than relying on the Pi/agent harness browser.
- Added `scripts/smoke-ssr-hydration.mjs`.
- Added `web/package.json` script: `pnpm --dir web smoke:ssr`.
- The smoke script:
  - allocates random backend and SSR ports,
  - runs `pnpm --dir web build:all`,
  - starts `node web/server.mjs`,
  - starts `go run ./cmd/retro-obsidian-publish serve ... --ssr-url <sidecar>` with `GOWORK=off`,
  - waits for `/api/config` and sidecar `/health`,
  - asserts raw `/` HTML has `X-Powered-By: Express`, a populated root, no `/assets/index.js` fallback shell, and preloaded state,
  - opens Chromium with Playwright,
  - visits `/` and `/note/index`,
  - clicks the sidebar `Epistemology` tree button,
  - asserts zero browser console warnings/errors.

### Why
- The previous live test found failures that build/test did not catch: SSR 500 fallback, duplicate React hook errors, CWD-relative shell fallback, and hydration title timing.
- A future dependency-model cleanup must be guarded by exactly the behavior that failed before: live sidecar import, real Go proxying, raw SSR HTML, browser hydration, and client-side navigation.

### What worked
- Final full run passed:
  - command: `pnpm --dir web smoke:ssr`
  - raw SSR HTML: `436397 bytes`, `X-Powered-By=Express`
  - browser hydration: `0` console warnings/errors
  - client-side sidebar navigation to `/note/philosophy/epistemology` succeeded after waiting for the destination heading/title.
- The test also prints build output, including current SSR bundle size: `dist/ssr/entry-server.js 4,979.92 kB` with the current `ssr.noExternal: true` state.

### What didn't work
- First attempt failed because the root-level script could not statically resolve Playwright from `web/node_modules`:
  - command: `pnpm --dir web smoke:ssr`
  - error: `Error [ERR_MODULE_NOT_FOUND]: Cannot find package 'playwright' imported from .../scripts/smoke-ssr-hydration.mjs`
  - fix: use `createRequire(join(WEB_DIR, "package.json"))` and dynamically import `web`'s Playwright package.
- Second attempt failed due a smoke-test assertion bug, not an app bug:
  - error: `GET / did not contain substantial server-rendered root markup`
  - cause: the regex stopped at the first nested `</div>` inside the React tree.
  - fix: replace brittle nested-div extraction with simpler checks for populated root marker, expected note content, and `window.__PRELOADED_STATE__`.
- Cleanup initially left risk of `go run` child processes lingering.
  - fix: spawn long-running processes as detached process groups and kill by negative PID in cleanup.
- Sidebar navigation assertion initially clicked an `<a href="/note/philosophy/epistemology">` link, but the real sidebar tree uses buttons.
  - fix: click `page.getByRole("button", { name: /Epistemology/i })`.
- The first button-click assertion read `document.title` too early:
  - error: `Unexpected sidebar navigation title: Index — Test Vault`
  - fix: wait for the `Epistemology` heading and for `document.title.includes("Epistemology")` before asserting.

### What I learned
- The smoke test needs to validate the exact production-like path: Go proxy → SSR sidecar → built Vite shell → browser hydration. Unit-level SSR rendering alone is insufficient.
- Playwright resolution from a repository-level script is different from resolution inside `web`; scripts outside a package should explicitly resolve dependencies from the package that owns them.
- DOM string assertions against nested React markup should avoid naive non-greedy `</div>` regexes.
- Sidebar validation should interact with actual accessible controls, not assumptions about links.

### What was tricky to build
- Process cleanup was subtle because `go run` starts a compiled child process. Killing only the immediate child can leave the actual server running. Spawning detached process groups and killing the process group solves this for the smoke-test harness.
- The raw SSR HTML assertion needed to distinguish three states: real sidecar SSR, SPA fallback, and fallback index shell. The final checks combine response headers, empty-root detection, fallback-asset detection, and preloaded-state detection rather than relying on one brittle marker.
- The client navigation assertion needed to respect React's asynchronous data/title effect. Waiting for the destination heading and then waiting for `document.title` prevents a false negative race.

### What warrants a second pair of eyes
- The smoke test adds `playwright` as a dev dependency; reviewer should confirm this is acceptable for the web package and CI environment.
- The raw SSR HTML checks are deliberately pragmatic rather than a full HTML parser. They should catch the known fallback modes, but future shell changes may require updating the markers.
- The script uses `go run`, which is convenient but slower than testing a prebuilt binary. CI can later decide whether to build once and pass a binary path.

### What should be done in the future
- Add this smoke script to CI after deciding browser installation strategy.
- Consider adding a `--skip-build` documented mode for iterative local runs; the script already supports it.
- Use the smoke test as the acceptance gate before changing `vite.config.ts` SSR externalization.

### Code review instructions
- Start with `scripts/smoke-ssr-hydration.mjs` and review the process lifecycle, raw SSR assertions, and browser assertions.
- Review `web/package.json` for the new `smoke:ssr` script and `playwright` dev dependency.
- Validate with:
  - `pnpm --dir web smoke:ssr`
  - optional faster rerun after building: `pnpm --dir web smoke:ssr -- --skip-build`

### Technical details
- Final full command: `pnpm --dir web smoke:ssr`
- Final raw SSR check: `raw SSR HTML ok (436397 bytes, X-Powered-By=Express)`
- Final browser check: `browser hydration ok (0 console warnings/errors)`
- Build output observed current SSR entry size: `dist/ssr/entry-server.js 4,979.92 kB`

## Step 7: Add SSR Dependency Resolution Diagnostics

This step added an opt-in diagnostic mode to the SSR sidecar. When `SSR_DEBUG_RESOLUTION=1` is set, `server.mjs` prints the resolved module path and package version for React, React DOM, React Router, React Router DOM, and `react-resizable-panels` before importing the SSR bundle.

The diagnostic is intentionally lightweight and disabled by default. It exists to make future duplicate-React investigations concrete: instead of guessing whether the sidecar is resolving one React graph or several, the operator can run the sidecar or smoke test with a single environment variable and inspect the exact paths.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue the phased follow-up work after adding the smoke-test guardrail, now adding observability for the SSR dependency-resolution problem.

**Inferred user intent:** Make the root cause of duplicate React failures inspectable before attempting to shrink the SSR bundle or change Vite externalization.

**Commit (code):** c37606817508702e382034438e20328cf890c935 — "web: add SSR dependency diagnostics"

### What I did
- Added `SSR_DEBUG_RESOLUTION=1` support to `web/server.mjs`.
- Added `createRequire(import.meta.url)` so package versions can be read relative to the web package.
- Added diagnostics for:
  - `react`
  - `react-dom`
  - `react-dom/server`
  - `react-router`
  - `react-router-dom`
  - `react-resizable-panels`
- Ran the smoke test with diagnostics enabled:
  - `SSR_DEBUG_RESOLUTION=1 pnpm --dir web smoke:ssr -- --skip-build`

### Why
- The previous live failure was caused by mixed bundled/external React graphs. Future cleanup needs a fast way to verify where the SSR sidecar resolves React-family dependencies.
- Logging these paths is especially useful with pnpm because package paths encode peer dependency pairings, e.g. `react-router@7.17.0_react-dom@19.2.1_react@19.2.1__react@19.2.1`.

### What worked
- Diagnostic output showed all inspected packages resolving from the same `web/node_modules/.pnpm` tree.
- The smoke test still passed with diagnostics enabled:
  - raw SSR HTML ok
  - browser hydration ok
  - `0` console warnings/errors

### What didn't work
- N/A. This phase was straightforward after the smoke-test harness was in place.

### What I learned
- `import.meta.resolve()` gives the exact runtime module path that Node will use for each specifier.
- Reading package versions through a `createRequire()` rooted at `server.mjs` keeps diagnostics aligned with the sidecar runtime rather than the caller's current working directory.

### What was tricky to build
- The diagnostic needs to run before importing `./dist/ssr/entry-server.js`; otherwise a duplicate-React import failure could occur before diagnostics print. The sidecar now calls `await logSSRDependencyResolution()` before the SSR bundle dynamic import.
- Package versions are not available from `import.meta.resolve()` alone, so the helper maps subpath imports like `react-dom/server` back to their package name (`react-dom`) before reading `package.json`.

### What warrants a second pair of eyes
- The diagnostic currently logs paths only when `SSR_DEBUG_RESOLUTION=1`; reviewer should confirm this is the desired production behavior rather than adding a debug HTTP endpoint.
- The package-name parser is intentionally simple but sufficient for the current inspected specifiers.

### What should be done in the future
- Use this diagnostic output before and after changing `vite.config.ts` so the diary can compare runtime dependency resolution.
- If CI adopts the smoke test, consider one diagnostic run in a separate troubleshooting job rather than every normal test run.

### Code review instructions
- Review `web/server.mjs`, especially `logSSRDependencyResolution()` and the ordering before `await import("./dist/ssr/entry-server.js")`.
- Validate with:
  - `SSR_DEBUG_RESOLUTION=1 pnpm --dir web smoke:ssr -- --skip-build`

### Technical details
- Observed output included:
  - `react@19.2.1 -> .../web/node_modules/.pnpm/react@19.2.1/node_modules/react/index.js`
  - `react-dom/server@19.2.1 -> .../react-dom/server.node.js`
  - `react-router@7.17.0 -> .../react-router/dist/development/index.mjs`
  - `react-resizable-panels@3.0.6 -> .../react-resizable-panels.edge-light.js`
