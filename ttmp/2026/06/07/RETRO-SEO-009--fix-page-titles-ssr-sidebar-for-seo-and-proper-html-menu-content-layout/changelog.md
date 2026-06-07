# Changelog

## 2026-06-07

- Initial workspace created


## 2026-06-07

Created ticket RETRO-SEO-009 with comprehensive design/implementation guide for fixing page titles (SSR + React), adding SSR sidebar/breadcrumbs for SEO, and improving HTML semantic structure. Document covers the full architecture: Go server → SSR sidecar → React SPA.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout/design-doc/01-page-title-ssr-sidebar-and-html-layout-design-and-implementation-guide.md — Primary design and implementation guide


## 2026-06-07

Added research logbook (02-research-logbook.md) documenting all 24 resources consulted: 17 codebase files, 2 live site inspections, 1 previous ticket. Each entry records what was researched, why the resource was chosen, what was useful, what wasn't, and what needs updating.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout/reference/02-research-logbook.md — Research logbook with 24 resource entries


## 2026-06-07

Added second design guide for React Router SSR hydration cleanup and expanded tasks into implementation phases.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout/design-doc/02-react-router-ssr-hydration-cleanup-implementation-guide.md — Second guide for full hydration refactor


## 2026-06-07

Implemented React Router hydration cleanup: migrated client routes/navigation, removed Wouter, rendered real AppRoutes under StaticRouter for SSR, switched browser entry to hydrateRoot, added home-note prefetching, made layout clock hydration-safe, and aligned SSR/client titles (commits 55dbc34, 6f76a55, 5ee46f9, a4e89cb, 0060096).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/server.mjs — Home note prefetch and title alignment
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/App.tsx — React Router route table and AppRoutes export
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-client.tsx — Browser now hydrates SSR markup with hydrateRoot
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-server.tsx — SSR now renders real AppRoutes under StaticRouter


## 2026-06-07

Hard-cut hydration cleanup and live test: removed stale main.tsx, fixed deterministic initial UI/date rendering, bundled all SSR dependencies to avoid duplicate React, fixed server.mjs dist path resolution, and verified local backend+SSR+browser with zero console warnings/errors (commit 685f0ba).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/server.mjs — dist/index.html resolved relative to server.mjs for local/Docker consistency
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/components/molecules/FrontmatterPanel/FrontmatterPanel.tsx — UTC date formatting prevents SSR/browser timezone mismatch
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/store/uiSlice.ts — deterministic initial sidebar/right-panel state for hydration
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/vite.config.ts — ssr.noExternal=true prevents duplicate React in SSR sidecar


## 2026-06-07

Phase A complete: added automated live SSR hydration smoke test covering web build, SSR sidecar startup, Go proxying, raw SSR HTML, browser hydration console cleanliness, and sidebar navigation.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/scripts/smoke-ssr-hydration.mjs — End-to-end live SSR hydration guardrail
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/package.json — Adds smoke:ssr script and Playwright dev dependency


## 2026-06-07

Phase B complete: added opt-in SSR dependency-resolution diagnostics for React/React DOM/router packages and validated them with the live smoke test (commit c376068).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/server.mjs — Logs SSR dependency resolution when SSR_DEBUG_RESOLUTION=1


## 2026-06-07

Phase C complete: replaced ssr.noExternal=true with a consistent externalized React SSR dependency graph, reducing entry-server.js from ~4.98 MB to ~72 KB while live smoke hydration stayed clean (commit fcadc3c).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/vite.config.ts — SSR externalization and React dedupe configuration


## 2026-06-07

Phase D complete: updated SSR sidecar Dockerfile to keep production node_modules for the externalized SSR runtime while pruning dev dependencies after build, and validated the built container starts with dependency diagnostics (commit 5ae2a34).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/ssr.Dockerfile — SSR runtime dependency packaging


## 2026-06-07

Phase E complete: documented final SSR dependency model, runtime packaging requirement, validation commands, and bundle-size reduction from ~4.98 MB to 72.39 KB.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout/design-doc/02-react-router-ssr-hydration-cleanup-implementation-guide.md — Final measurement and tradeoff documentation


## 2026-06-07

Closed remaining original tasks: validation is covered by smoke/full checks, and pageTitle is wired through devctl, docker-compose, and the production GitOps deployment (publish-vault commit fadb713, GitOps commit b4cb0a1).

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml — production page title args
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/.devctl.yaml — PAGE_TITLE values for local profiles
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/docker-compose.yml — compose page-title args
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/plugins/retro-obsidian-publish.py — PAGE_TITLE launch plumbing


## 2026-06-07

Ticket closed


## 2026-06-07

Post-close CI fix: removed unconditional Docker COPY of optional web/patches from both image Dockerfiles after GitHub Actions failed because the empty directory is not present in clean checkouts.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/Dockerfile — Same optional patches fix for main app image
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/ssr.Dockerfile — Avoids missing optional web/patches in clean CI checkout


## 2026-06-07

Post-close PR review fix: restored closed mobile sidebar initial state with a hydration-safe post-mount viewport adjustment and smoke-test coverage (commit 11641d2).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/scripts/smoke-ssr-hydration.mjs — Mobile fresh-load sidebar regression check
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/components/pages/VaultLayout/VaultLayout.tsx — Hydration-safe mobile sidebar default

