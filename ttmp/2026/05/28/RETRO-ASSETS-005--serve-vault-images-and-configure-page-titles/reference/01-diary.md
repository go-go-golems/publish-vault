---
Title: Diary
Ticket: RETRO-ASSETS-005
Status: active
Topics:
    - assets
    - images
    - config
    - page-title
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for vault image serving and configurable page titles."
LastUpdated: 2026-05-28T23:55:00Z
WhatFor: "Use to resume, review, or validate RETRO-ASSETS-005."
WhenToUse: "Before continuing implementation or reviewing the image/page-title changes."
---

# Diary

## Goal

This diary records the investigation, implementation, validation, commits, and follow-up risks for `RETRO-ASSETS-005`: serving vault images correctly and making the browser page title configurable.

## Step 1: Ticket setup, current-state analysis, and implementation plan

I created a new docmgr ticket and wrote an evidence-backed design guide before changing code. The main architectural finding is that the backend already has a clean parsing and HTML rebuild seam in `vault.rebuildHTML`, but it does not currently rewrite Markdown image URLs or expose a vault asset route before the SPA catch-all.

The second finding is that site configuration already flows from the Go API into the React app via `/api/config`; it only carries `vaultName` and note count today. That makes `pageTitle` a small extension to an existing contract rather than a separate frontend configuration mechanism.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to address:

- serving images, for example: https://parc.yolo.scapegoat.dev/note/projects/2026/05/28/article-m5dial-proper-3d-renderer-building-a-z-buffered-planet-and-terrain-on-esp32-s3 has images (from ~/code/wesen/go-go-golems/go-go-parc/), but they are not redirected to the real assets.
- make page title configurable. 

Create a detailed analysis and implementation guide, then add detailed tasks to the ticket, then implement, commit at appropriate intervals, and keep a diary."

**Assistant interpretation:** Create a full docmgr ticket, analyze how this app currently handles note HTML/assets/config, write a detailed implementation plan and tasks, then implement and commit incremental code/docs changes while maintaining a diary.

**Inferred user intent:** The published PARC vault should render local note images correctly and should have deployment-specific browser/tab branding without ad-hoc patches.

**Commit (code):** N/A — planning step only so far.

### What I did

- Ran repository orientation commands:
  - `pwd && git status --short && docmgr status --summary-only`
  - `find . -maxdepth 3 -type f | sed 's#^./##' | sort | head -200`
  - `docmgr ticket list && find ttmp -maxdepth 3 -type f | sort`
- Created the ticket:
  - `docmgr ticket create-ticket --ticket RETRO-ASSETS-005 --title "Serve vault images and configure page titles" --topics assets,images,config,page-title,obsidian-vault`
- Added documents:
  - `docmgr doc add --ticket RETRO-ASSETS-005 --doc-type design-doc --title "Image serving and page title implementation guide"`
  - `docmgr doc add --ticket RETRO-ASSETS-005 --doc-type reference --title "Diary"`
- Inspected source files that define the relevant runtime behavior:
  - `backend/cmd/retro-obsidian-publish/commands/serve/serve.go`
  - `backend/internal/server/server.go`
  - `backend/internal/server/runtime.go`
  - `backend/internal/api/api.go`
  - `backend/internal/parser/parser.go`
  - `backend/internal/vault/vault.go`
  - `web/src/store/vaultApi.ts`
  - `web/src/App.tsx`
  - `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`
  - `web/index.html`
- Wrote the primary design guide with current-state evidence, proposed APIs, pseudocode, implementation phases, tests, risks, and alternatives.
- Replaced the placeholder tasks file with a detailed phase-by-phase checklist.
- Replaced the placeholder reference document with this diary entry.

### Why

- The requested change touches backend routing, parser/vault HTML generation, API contracts, frontend config consumption, and deployment behavior. A design-first pass reduces the risk of adding a route that conflicts with the SPA catch-all or rewriting image URLs in the wrong layer.
- The ticket needs enough detail for review and continuation even if implementation spans multiple commits.

### What worked

- The existing `RuntimeState` abstraction already tracks the active resolved vault root, which is exactly the information an asset route needs after git-sync reloads.
- The existing `/api/config` flow already reaches the top-level React router, so configurable page title can be implemented with a small contract extension.
- The existing `vault.rebuildHTML` post-processing step is a natural place to normalize note-relative image URLs after Markdown parsing.

### What didn't work

- A broad `rg -n "vault|note|asset|image|title|Config|serve|http|static|Page" backend web/src README.md -S` command produced noisy matches inside bundled/generated frontend assets under `backend/internal/web/embed/public/assets/...`. I narrowed the investigation to source files using `rg --files backend web/src | sort` and direct reads of relevant files.
- `git status --short` showed many existing untracked screenshot and `.playwright-mcp/` artifacts before this work began. I did not touch them.

### What I learned

- The Go server registers the SPA catch-all at `PathPrefix("/")`, so `/assets/` must be registered earlier or image requests will receive the SPA shell.
- The backend currently loads only `.md` files into the `Vault`, which means attachments do not need to be indexed as notes. Asset serving can use request-time filesystem lookup instead.
- Frontend title configuration should be runtime data from `/api/config`, not a build-time Vite variable, because the same SPA bundle is served for different vault deployments.

### What was tricky to build

- The main subtlety is path ownership. Markdown image paths are semantically relative to the note file, but browser requests would otherwise resolve them relative to `/note/{slug}`. The design resolves paths during backend HTML rebuild, where both the note path and the vault root are available.
- Another subtlety is reload behavior. Capturing an `http.Dir` at startup could serve stale files after a git-sync symlink flips. The planned handler reads `state.ResolvedRoot()` per request.

### What warrants a second pair of eyes

- The security boundary of `/assets/`: traversal, hidden files, directory serving, and Markdown exposure must be checked carefully.
- The exact Obsidian attachment-resolution semantics are broader than deterministic note-relative paths. Review whether the PARC vault's image references are relative, vault-root-relative, or basename-only before declaring the implementation complete.
- Regex-based image rewriting should be verified against goldmark's actual image tag output.

### What should be done in the future

- If PARC uses Obsidian's attachment-folder search by basename, add a second resolution phase that searches configured attachment directories or a prebuilt non-Markdown asset index.
- Consider an operator-facing note that all non-hidden, non-Markdown files under the vault may be publicly served when linked.

### Code review instructions

- Start with the design doc: `ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/design-doc/01-image-serving-and-page-title-implementation-guide.md`.
- Then review the future implementation in this order:
  1. parser image-source rewrite helper and tests,
  2. vault path resolution and HTML rebuild integration,
  3. server `/assets/` handler and route ordering,
  4. API/CLI page-title config,
  5. frontend `document.title` effect.
- Validation command planned for backend changes: `cd backend && go test ./... -count=1`.
- Validation command planned for frontend changes: `pnpm --dir web typecheck` or the repository's equivalent frontend check.

### Technical details

- Important source routes and seams:
  - `backend/internal/server/server.go`: route registration and SPA catch-all.
  - `backend/internal/server/runtime.go`: active resolved vault root.
  - `backend/internal/api/api.go`: `/api/config` contract.
  - `backend/internal/parser/parser.go`: Markdown-to-HTML conversion and post-processing helpers.
  - `backend/internal/vault/vault.go`: note loading and `rebuildHTML`.
  - `web/src/App.tsx`: central config consumer.
- Proposed public contract:
  - `GET /assets/{vault-relative-path}` serves validated non-Markdown vault files.
  - `GET /api/config` returns `{ "vaultName": string, "pageTitle": string, "notes": number }`.
