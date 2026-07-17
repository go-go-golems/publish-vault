---
Title: Investigation diary
Ticket: PV-BACKEND-API-001
Status: active
Topics:
    - backend
    - api
    - ssr
    - xgoja
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md
      Note: Investigated for xgoja HTTP server packaging
    - Path: internal/server/server.go
      Note: Investigated for current server route and SSR behavior
    - Path: internal/vault/vault.go
      Note: Investigated for Markdown-to-note content pipeline
    - Path: web/server.mjs
      Note: Investigated for SSR prefetch and metadata behavior
ExternalSources: []
Summary: Chronological investigation notes for the publish-vault backend API and xgoja design ticket.
LastUpdated: 2026-06-22T21:45:00-04:00
WhatFor: Use to resume or review the investigation behind PV-BACKEND-API-001.
WhenToUse: Read before implementing the backend API, SSR render endpoint, or xgoja integration.
---


# Diary

## Goal

Capture the investigation and design work for adding a backend API to publish-vault so dynamic pages and future xgoja/JavaScript-backed servers can fuel the existing frontend and SSR flow.

## Step 1: Create ticket and map current backend architecture

I created a new docmgr ticket and inspected publish-vault from the CLI entrypoint through the server, vault loader, API handlers, React API client, SSR sidecar, and agent markdown mirrors. The key finding is that publish-vault is not just serving Markdown files from disk: it parses Markdown into enriched note objects, renders HTML, resolves wiki links, rewrites asset URLs, builds backlinks and search, and exposes that derived data through JSON endpoints.

This step produced a design document that recommends formalizing the current behavior behind a backend interface and a `/api/v1` HTTP contract before adding dynamic pages or xgoja implementations.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add a backend API for publish-vault so that we can build dynamic pages instead of just serving markdown from disk (I think that's how it works).

Analkyze  publish-vault and figure out how it works.

The idea is that I can in the future easily package it with xgoja (see ./go-go-goja/) to quickly write a JS version of a server that fuels a publish-vault, with SSR too."

**Assistant interpretation:** Create a structured docmgr ticket, analyze the existing publish-vault codebase, and write an evidence-backed backend API/xgoja design for future dynamic pages and SSR.

**Inferred user intent:** Establish a technical roadmap and ticket workspace before implementation, with enough codebase understanding to avoid designing an API that fights the current architecture.

**Commit (code):** N/A — documentation-only investigation in progress.

### What I did

- Ran `docmgr status --summary-only` and inspected existing tickets.
- Created ticket `PV-BACKEND-API-001` with topics `backend,api,ssr,xgoja,obsidian-vault`.
- Added a design document and investigation diary.
- Added tasks for mapping current request flow, inspecting xgoja integration points, and writing the design.
- Inspected publish-vault files:
  - `cmd/retro-obsidian-publish/main.go`
  - `cmd/retro-obsidian-publish/commands/root.go`
  - `cmd/retro-obsidian-publish/commands/serve/serve.go`
  - `internal/server/server.go`
  - `internal/server/runtime.go`
  - `internal/server/agent_markdown.go`
  - `internal/api/api.go`
  - `internal/vault/vault.go`
  - `internal/parser/parser.go`
  - `internal/search/search.go`
  - `internal/watcher/watcher.go`
  - `web/src/store/vaultApi.ts`
  - `web/src/App.tsx`
  - `web/src/entry-server.tsx`
  - `web/server.mjs`
- Inspected go-go-goja files and docs relevant to packaging a JS HTTP server:
  - `cmd/xgoja/root.go`
  - `cmd/xgoja/cmd_build.go`
  - `cmd/xgoja/doc/22-http-serve-command-reference.md`
  - `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`
  - `modules/express/express.go`
  - `pkg/gojahttp/host.go`
  - `examples/xgoja/13-http-serve-jsverbs/xgoja.yaml`
  - `examples/xgoja/13-http-serve-jsverbs/verbs/sites.js`
- Wrote `design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md`.

### Why

- The current implementation already exposes a JSON API, but it is implicit and note-centric.
- Dynamic pages require a page/route/render model that is not identical to disk-backed Markdown notes.
- SSR currently duplicates route selection and prefetch logic; this should become a backend-owned render contract before adding dynamic routes.
- xgoja can host Express-like JavaScript route registration, but publish-vault needs a stable API contract or backend interface before a JS implementation can be straightforward.

### What worked

- The codebase has clear boundaries: CLI command, server runtime, vault loader, API handlers, search, watcher, React API client, and SSR sidecar.
- The frontend already centralizes backend calls through RTK Query, which makes a versioned API migration plausible.
- xgoja already has an HTTP provider, an Express-compatible module, generated `serve` commands, and external-host support for hybrid Go/JS servers.
- Existing agent markdown and SSR behavior gave concrete requirements for the new backend API.

### What didn't work

- No command failures occurred during the investigation.
- One expected documentation path did not exist: `pkg/engine/engine.go` in go-go-goja. The command was:
  - `cd go-go-goja && nl -ba pkg/gojahttp/host.go | sed -n '1,280p' && printf '\n--- engine ---\n' && nl -ba pkg/engine/engine.go | sed -n '1,260p'`
  - Error: `nl: pkg/engine/engine.go: No such file or directory`
- This was not blocking because the HTTP host and xgoja docs contained the needed evidence.

### What I learned

- publish-vault parses and enriches Markdown rather than just serving files. The derived `Note` includes rendered HTML, raw Markdown, frontmatter, tags, wiki links, backlinks, and modification time.
- The Go SSR integration is a reverse proxy to a Node sidecar with SPA fallback on sidecar errors.
- The Node SSR sidecar prefetches API data and duplicates home route selection. A backend `render` endpoint would reduce this duplication.
- Agent-readable markdown mirrors are generated in Go from the runtime state and must be generalized for dynamic pages.
- xgoja standalone servers are feasible, but a hybrid mode is lower risk because existing publish-vault owns safe asset serving, reloads, SSR fallback, and markdown mirror aliases.

### What was tricky to build

- The main conceptual sharp edge was separating "note" from "page". The current UI treats most content routes as notes, but dynamic pages need their own route/page model.
- SSR adds another ordering constraint: route resolution must happen before React render so the cache can be preloaded and correct status/canonical/alternate metadata can be emitted.
- xgoja has two viable integration modes: standalone HTTP server and Go-owned host with JS routes. The design had to avoid prematurely committing to one while still recommending a low-risk path.

### What warrants a second pair of eyes

- Whether the proposed `Backend` interface should be internal-only first or public under `pkg/` for xgoja adapters.
- Whether `/api/v1/render` should return fully rendered HTML/markdown, structured page data, or both.
- Whether JS dynamic pages should be trusted to return HTML directly, or whether publish-vault should sanitize or require structured rendering.
- Whether search should be backend-owned or centralized over a feed of search documents.

### What should be done in the future

- Implement Phase 1 from the design: backend interface plus behavior-preserving `/api/v1` aliases.
- Add contract/golden tests before changing frontend or SSR behavior.
- Prototype a minimal xgoja backend that returns `/api/v1/site`, `/api/v1/routes`, and one dynamic page.

### Code review instructions

- Start with `ttmp/2026/06/22/PV-BACKEND-API-001--design-backend-api-for-dynamic-publish-vault-pages-and-xgoja-servers/design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md`.
- Verify the current-state claims against:
  - `internal/server/server.go`
  - `internal/api/api.go`
  - `internal/vault/vault.go`
  - `web/server.mjs`
  - `web/src/store/vaultApi.ts`
  - `go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md`
- Validate docmgr hygiene with:
  - `cd publish-vault && docmgr doctor --ticket PV-BACKEND-API-001 --stale-after 30`

### Technical details

Important observed contracts:

```text
Current publish-vault flow:
serve command
  -> server.Run
  -> RuntimeState loads vault + Bleve index
  -> API handlers expose notes/tree/search/tags/config
  -> Go serves assets and agent markdown mirrors
  -> optional SSR proxy sends page requests to Node sidecar
  -> Node sidecar prefetches API data and renderToString()s React
```

```text
Recommended future flow:
request path
  -> backend.Resolve or /api/v1/render
  -> backend returns Route + Page + navigation/preload data
  -> SSR renders from one prepared payload
  -> frontend hydrates same payload
  -> markdown mirrors call backend markdown/page output
```
