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
