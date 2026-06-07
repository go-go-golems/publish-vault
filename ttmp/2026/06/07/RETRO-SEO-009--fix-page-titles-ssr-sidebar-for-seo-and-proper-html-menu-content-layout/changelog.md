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

