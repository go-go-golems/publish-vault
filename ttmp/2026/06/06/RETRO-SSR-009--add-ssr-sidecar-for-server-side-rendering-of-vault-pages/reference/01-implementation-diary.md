---
ticket: RETRO-SSR-009
doc_type: reference
title: "Implementation Diary"
status: active
intent: long-term
topics: [ssr, react, node-js, seo, a14y, hydration, sidecar]
created: 2026-06-06
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
