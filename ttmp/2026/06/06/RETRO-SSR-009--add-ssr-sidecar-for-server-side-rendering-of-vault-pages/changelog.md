# Changelog

## 2026-06-06

- Initial workspace created


## 2026-06-06

Step 1: Created ticket, wrote comprehensive SSR design doc with current-state analysis, reference implementation review, gap analysis, proposed architecture, decision records, pseudocode, and phased implementation plan. Created diary.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/06/RETRO-SSR-009--add-ssr-sidecar-for-server-side-rendering-of-vault-pages/design-doc/01-ssr-sidecar-analysis-and-implementation-guide.md — Primary design document
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/06/RETRO-SSR-009--add-ssr-sidecar-for-server-side-rendering-of-vault-pages/reference/01-implementation-diary.md — Implementation diary


## 2026-06-06

Added research logbook: 27 resources cataloged across 4 categories (backend, frontend, glazed reference, design docs) with status assessment and update requirements


## 2026-06-06

Phase 1 complete: store.ts refactored to makeStore() factory, entry-client.tsx with hydrateRoot created, vite.config.ts updated with SSR noExternal + rollupOptions, index.html points to entry-client.tsx. Build verified (2f349f1).

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-client.tsx — Client hydration entry with __PRELOADED_STATE__
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/store/store.ts — makeStore factory + store singleton
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/vite.config.ts — SSR noExternal + rollupOptions


## 2026-06-06

Phases 1-5 complete: store factory, entry-client, entry-server, server.mjs sidecar, Go SSR proxy with tests, Docker + compose (commits 2f349f1..17dbb02)

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/backend/internal/server/server.go — SSR proxy with fallback
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/backend/internal/server/ssr_proxy_test.go — SSR proxy tests
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/docker-compose.yml — Compose with sidecar service
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/server.mjs — Node.js SSR sidecar
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-server.tsx — SSR entry with renderApp
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/ssr.Dockerfile — SSR sidecar Docker image


## 2026-06-06

Task 2.6: Added entry-server unit tests (11 passing), exported parseRoute, fixed React key warnings in SSR components (commit 78ffb27)

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-server.test.tsx — 11 unit tests for renderApp and parseRoute
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/vitest.config.ts — Vitest config for SSR tests


## 2026-06-06

Phases 3.8 + 6 complete: devctl runs 3 services (backend + web + ssr), SSR verified end-to-end. Fixed Express 4 wildcard syntax, added static asset routes to Go server, switched to createRoot to avoid hydration mismatch. Browser renders correctly with zero console errors (commit b30da32). All 31 tasks done.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/backend/internal/server/server.go — Added /assets/
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/plugins/retro-obsidian-publish.py — Added SSR sidecar service to devctl
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/server.mjs — Fixed Express 4 wildcard from {*path} to *
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/web/src/entry-client.tsx — Switched from hydrateRoot to createRoot to avoid #418 hydration mismatch


## 2026-06-06

Phase 7 planning: added Markdown Mirror and a14y Implementation Guide, detailed tasks, and saved a14y configuration in ticket scripts folder.

### Related Files

- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/06/RETRO-SSR-009--add-ssr-sidecar-for-server-side-rendering-of-vault-pages/design-doc/02-markdown-mirror-and-a14y-implementation-guide.md — Implementation guide for markdown mirrors
- /home/manuel/workspaces/2026-06-04/publish-vault-ssr/publish-vault/ttmp/2026/06/06/RETRO-SSR-009--add-ssr-sidecar-for-server-side-rendering-of-vault-pages/scripts/01-a14y-config.md — a14y configuration and baseline score

