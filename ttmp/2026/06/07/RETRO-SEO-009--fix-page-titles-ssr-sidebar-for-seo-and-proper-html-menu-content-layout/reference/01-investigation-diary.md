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
Summary: Chronological investigation log for page title, SSR sidebar, and HTML layout issues.
LastUpdated: 2026-06-07T12:00:00Z
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
