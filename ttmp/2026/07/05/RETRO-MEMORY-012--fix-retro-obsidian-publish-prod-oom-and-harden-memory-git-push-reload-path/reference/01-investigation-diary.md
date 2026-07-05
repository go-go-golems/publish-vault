---
Title: Investigation diary — retro-obsidian-publish prod OOM (RETRO-MEMORY-012)
Ticket: RETRO-MEMORY-012
Status: active
Topics:
    - retro-obsidian-publish
    - vault
    - search
    - deployment
    - git-sync
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/parser/parser.go
      Note: Inline code is unwrapped
    - Path: internal/parser/parser_test.go
      Note: Regression test for inline code plaintext
    - Path: internal/search/search.go
      Note: |-
        Streaming index builders and closed-index guard for review fix
        OpenPersistent helper for final-path reopen
    - Path: internal/server/runtime.go
      Note: Close/rename/reopen sequence for final-path persistent indexes
    - Path: internal/server/runtime_test.go
      Note: Regression test for final-path persistent index updates
    - Path: internal/server/server.go
      Note: Watcher shutdown before admin reload for review fix
    - Path: internal/vault/vault.go
      Note: Streaming search document iterator for review fix
    - Path: internal/vault/vault_test.go
      Note: Search document test includes inline code
    - Path: web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Static raw markdown fallback for copy/view/download actions
    - Path: web/src/vault/staticVault.ts
      Note: Static raw markdown retention for review fix
ExternalSources: []
Summary: Chronological investigation diary for diagnosing the retro-obsidian-publish prod OOM and writing the intern deliverable.
LastUpdated: 2026-07-05T00:00:00Z
WhatFor: Record what was tried, what worked, and what to validate
WhenToUse: Read before resuming work on RETRO-MEMORY-012
---



# Investigation Diary — RETRO-MEMORY-012

## Goal

Diagnose why `retro-obsidian-publish` is down in production, confirm the root cause with evidence, and produce an intern-facing analysis/design/implementation guide that also captures the "git push → live site" carry-over flow.

---

## Step 1: Confirm the prod symptom and identify the failing container

### Prompt Context

**User prompt (verbatim):** "There's something off with retro-obsidian-publish in prod. Create a new docmgr ticket and investigate. I also want this to be a nicer carry over when there is a new git push. Create a detailed analysis / design / implementation guide that is for a new intern ... Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Investigate the production `retro-obsidian-publish` deployment, create a docmgr ticket with a comprehensive intern-oriented design doc, and preserve/harden the git-push carry-over.

**Inferred user intent:** Get the site back up, understand the failure deeply, and leave a document that lets anyone new ramp up quickly.

### What I did
- Loaded the workspace at `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault` (worktree on `task/memory-publish-vault`).
- Read the gitops manifests and ArgoCD application for `retro-obsidian-publish`.
- Used the tailnet kubeconfig (`.cache/kubeconfig-tailnet.yaml`) to inspect the live cluster.

### What worked
- `kubectl get pod -n retro-obsidian-publish` immediately showed `2/3` ready, `CrashLoopBackOff`, 439 restarts.
- Container-status jsonpath pinpointed the `app` container as the failing one: `exitCode=137`, `reason=OOMKilled`, 236 restarts. `ssr` and `git-sync` were healthy.

### What didn't work
- `kubectl exec -n retro-obsidian-publish <pod> -c app -- ...` failed with `container not found ("app")` because the `app` container is mid-crash-loop and not running. Worked around by exec-ing into the `git-sync` sidecar, which shares the `/git` emptyDir.

### What I learned
- The `app` container's logs were completely clean (normal startup lines, no error). That is the signature of an OOM kill: the kernel removes the process with no chance to log.
- `git-sync` showed 203 historical restarts but was currently healthy (last error 2026-06-25).

### What was tricky to build
- Distinguishing "cluster is starved" from "container outgrows its own limit". Resolved by reading both `kubectl top nodes` and the per-container `lastState` from the pod status.

### What warrants a second pair of eyes
- Confirm the 1536Mi limit value is read from the live spec, not just the git repo (they matched, but worth re-checking after any deploy).

### What should be done in the future
- Add a memory/regression test (see design doc §9) so this failure mode is caught in CI.

### Code review instructions
- Start at the deployment manifest in the hetzner-k3s repo (`gitops/kustomize/retro-obsidian-publish/deployment.yaml`) — verify the `app` container `resources.limits.memory`.
- Validate: `kubectl get pod -n retro-obsidian-publish -o jsonpath='{.items[*].spec.containers[?(@.name=="app")].resources}'`.

### Technical details
- Pod: `retro-obsidian-publish-5b46b488f5-gx69l`. Node: `k3s-demo-1`.

---

## Step 2: Rule out cluster starvation and map memory consumers

### Prompt Context

**User prompt (verbatim):** "so this is a memory issue? can you look at cluster load and what eats memory?"

**Assistant interpretation:** Confirm the OOM is memory, then show cluster-wide memory usage to determine whether the node is full.

**Inferred user intent:** Decide whether the fix is "give the app more room" vs "free up room on the node".

### What I did
- `kubectl top nodes`, `kubectl top pods -A --sort-by=memory`.
- Aggregated measured RSS per namespace, and summed pod `requests`/`limits` from the pod specs via a small Python script.

### What worked
- Node showed 58% memory used (9152Mi of ~15.6GiB allocatable), 7% CPU. ~6.8GiB free.
- Top consumers: monitoring 1320Mi (Prometheus 796 + Grafana 308), mysql 726, argocd 679, keycloak 623. `retro-obsidian-publish` measured low (~230Mi) only because it gets killed before settling.

### What didn't work
- My first per-namespace aggregation awk used the wrong column index and produced nonsense numbers. Recomputed with correct `$4` indexing.

### What I learned
- Requests total ~9.1GiB (fits), limits total ~24GiB (overcommitted but normal). Nothing is competing the app to death — the app outgrows its own 1536Mi ceiling.

### What was tricky to build
- Aggregating `Mi`/`Gi` units in shell. Switched to Python with an explicit unit parser for the requests/limits table.

### What warrants a second pair of eyes
- The "node has headroom" conclusion depends on the node being single-node. Confirm k3s-demo-1 is the only node (it is; `kubectl get nodes` returned one).

### What should be done in the future
- Add memory requests/limits auditing to the cluster monitoring dashboards.

### Code review instructions
- Re-run `kubectl top nodes` and the Python requests/limits aggregator (see design doc §4.1) to reproduce.

---

## Step 3: Read the application source to find the in-memory redundancy

### Prompt Context

**User prompt (verbatim):** "the code is in ../go-go-golems/publish-vault/"

**Assistant interpretation:** Read the publish-vault source to determine why the process uses so much memory, rather than guessing.

**Inferred user intent:** Decide whether a DB/index change is warranted.

### What I did
- Read `internal/vault/vault.go`, `internal/search/search.go`, `internal/server/server.go`, `internal/server/runtime.go`, `internal/api/api.go`, `cmd/.../serve/serve.go`.
- Gathered line-anchored evidence for the `Note` struct, `LoadAll`, `rebuildHTML`, `bleve.NewMemOnly`, `NewPersistent`, `RuntimeState.Reload`, `loadVaultAndSearch`, `reloadHandler`, `getNoteRaw`.

### What worked
- Found the three-fold redundancy: `RawMarkdown` + `HTML` on each `Note`, plus an in-RAM bleve index.
- Discovered `search.NewPersistent` already exists but is never called.

### What didn't work
- Measuring the exact prod vault byte size via busybox `find` inside the `git-sync` container returned 0 (busybox `find -printf` not supported). Fell back to measuring the local clone of `go-go-parc`: 934 `.md` files, ~22.8MiB.

### What I learned
- The fix is partially already implemented (`NewPersistent`). The root cause is redundant in-memory copies, not a missing query engine.
- `RawMarkdown` is consumed by exactly one endpoint (`/api/notes/{slug}/raw`), so caching it for every note is wasteful.

### What was tricky to build
- Establishing the "three copies" framing precisely from the code without overclaiming. Each copy is anchored to a specific line.

### What warrants a second pair of eyes
- The memory arithmetic in design doc §4.3 is an estimate, not a measurement. After Phase 1 lands, measure actual RSS with `kubectl top pod` to validate.

### What should be done in the future
- Add `GOMEMLIMIT` and expose Go memory metrics on `/api/healthz` so footprint is observable, not inferred.

### Code review instructions
- Open `internal/vault/vault.go:18-29` and `:131-162`, and `internal/search/search.go:41-75`. The three-copy claim and the unused persistent path are directly visible.

### Technical details
- bleve in-RAM index (`NewMemOnly`) holds tokenized bodies + term dictionaries + postings; this is typically the largest single consumer.

---

## Step 4: Confirm the reload/reindex-without-restart mechanism and its OOM tie-in

### Prompt Context

**User prompt (verbatim):** "Can we also potentially trigger a reindex without having to restart the container?"

**Assistant interpretation:** Determine whether a full reindex can be triggered without a container restart.

**Inferred user intent:** Preserve fast content updates from git pushes without rolling the pod.

### What I did
- Traced `POST /api/admin/reload` → `reloadHandler` → `RuntimeState.Reload` → `loadVaultAndSearch`.
- Cross-referenced the `git-sync` webhook args in `deployment.yaml`.

### What worked
- Confirmed reindex-without-restart already exists and is already wired to git pushes (git-sync polls every 60s and POSTs the reload webhook). Auth is loopback-only via `--reload-allow-loopback`.
- Identified that `Reload` builds a full second state before swapping → transient ~2× memory → the most likely OOM trigger on each git-sync cycle.

### What didn't work
- N/A.

### What I learned
- The carry-over the user wants is already implemented; the work is to make each reload cheap enough to stay under the limit.

### What was tricky to build
- Connecting the "reload works" story to the "pod keeps dying" story: the elegant atomic-swap design is also the memory-doubling mechanism. Both are the same code path.

### What warrants a second pair of eyes
- Whether loopback-only auth is acceptable for the team's security posture, or whether a bearer token should also be required.

### What should be done in the future
- Make the on-disk bleve index update incrementally (Phase 1 follow-up) so reload CPU/IO drops too.

### Code review instructions
- `internal/server/runtime.go:62-96` (Reload + loadVaultAndSearch) and `internal/server/server.go:88-94,166` (route + handler). The webhook wiring is in `deployment.yaml` under the `git-sync` container args.

### Technical details
- `git-sync` `--webhook-success-status=204` matches the `http.StatusNoContent` returned by `reloadHandler` on success.

---

## Step 5: Create the ticket and write the intern deliverable

### Prompt Context

**User prompt (verbatim):** "I created a dedicated workspace in /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault . Create the docmgr ticket there and the doc there."

**Assistant interpretation:** Create the docmgr ticket inside the publish-vault workspace and place the design doc there.

**Inferred user intent:** Keep this investigation's artifacts co-located with the publish-vault source on the dedicated `task/memory-publish-vault` branch.

### What I did
- Confirmed workspace docmgr root is `ttmp` (`.ttmp.yaml`).
- Checked existing ticket numbering (highest was 011) → created `RETRO-MEMORY-012`.
- Wrote `design/01-prod-oom-memory-and-reload-architecture-guide.md` with prose, bullet lists, ASCII diagrams, pseudocode, an API reference table, decision records, phased plan, and file/line references.

### What worked
- `docmgr ticket create-ticket` produced the standard directory skeleton.
- Line-anchored evidence (gathered in Step 3) made the design doc precise.

### What didn't work
- N/A.

### What I learned
- The workspace is a git worktree of publish-vault; referencing source files via the workspace path is correct for the intern.

### What was tricky to build
- Keeping the two repos (publish-vault source vs hetzner-k3s gitops) clearly separated in the doc so an intern does not edit the wrong repo for a memory-limit change.

### What warrants a second pair of eyes
- The Phase 0 limit value (3072Mi) and the recommendation to use `emptyDir` vs PVC for the persistent index (Q1).

### What should be done in the future
- Upload the design doc + this diary to reMarkable as a bundled PDF (Step 6).

### Code review instructions
- Review `design/01-prod-oom-memory-and-reload-architecture-guide.md` end to end; verify every `file.go:NN` reference resolves.

### Technical details
- Ticket path: `ttmp/2026/07/05/RETRO-MEMORY-012--fix-retro-obsidian-publish-prod-oom-and-harden-memory-git-push-reload-path/`.

---

## Step 6: Validate and upload to reMarkable

### Prompt Context

**User prompt (verbatim):** (same as Step 1; this step completes the "upload to remarkable" part.)

**Assistant interpretation:** Run docmgr validation, then bundle the design doc + diary and upload to reMarkable.

**Inferred user intent:** Have an offline-readable copy on the reMarkable for review.

### What I did
- (To be executed next.) `docmgr doctor --ticket RETRO-MEMORY-012 --stale-after 30`; resolve any vocab warnings.
- `remarquee status` + `remarquee cloud account --non-interactive`.
- Dry-run bundle upload of the design doc and diary to `/ai/2026/07/05/RETRO-MEMORY-012`, then the real upload.
- Verified with `remarquee cloud ls`.

### What worked
- (filled in after execution)

### What didn't work
- (filled in after execution)

### What I learned
- (filled in after execution)

### What was tricky to build
- (filled in after execution)

### What warrants a second pair of eyes
- The upload destination and bundle name.

### What should be done in the future
- After Phases 0–2 land, update this diary with measured RSS and close the ticket.

### Code review instructions
- Verify `remarquee cloud ls /ai/2026/07/05/RETRO-MEMORY-012 --long` lists the bundle.

### Technical details
- Bundle name: `RETRO-MEMORY-012 retro-obsidian-publish OOM and reload guide`.

---

## Step 7: Take over the design and create the implementation phase plan

The takeover review changed the implementation strategy from a quick persistent-index patch to a safer phase plan. The important shift is that backwards compatibility is no longer a constraint; the app only needs to continue working. That lets Phase C remove `rawMarkdown` from the full note API and update the frontend copy action instead of preserving a response field solely for compatibility.

I converted the previous coarse task list into a detailed, resumable checklist with explicit phase boundaries, validation commands, commit points, and diary/doc upkeep tasks. This makes it possible to stop after any phase and resume from the checklist without reconstructing context from the chat transcript.

### Prompt Context

**User prompt (verbatim):** "Ok, we don't need any backwards compatibility btw, as long as the app works.

Create a detailed list of phases + tasks per phase so we can nicely track our progress (and resume after stops), then implement phase per phase, committing at appropriate intervals, and keeping a detailed diary as you go."

**Assistant interpretation:** Update the ticket plan to be detailed and resumable, then implement in phases with focused commits and diary updates.

**Inferred user intent:** Avoid losing context during a multi-phase refactor and ensure each phase is reviewable and resumable.

### What I did
- Rewrote `tasks.md` into Phases A–G.
- Incorporated the no-backwards-compatibility decision into Phase C: remove `rawMarkdown` from the API/frontend contract and update the app to fetch raw markdown on demand.
- Added explicit commit checkpoints and validation steps per phase.
- Ran baseline `go test ./...` before coding; all Go tests passed.

### Why
- The previous task list still reflected the first design and treated persistent bleve as Phase 1. The review showed that instrumentation and model/API separation should come first.

### What worked
- Baseline tests passed:
  - `go test ./...`

### What didn't work
- N/A.

### What I learned
- The task list itself needs to encode design decisions, not just todos; otherwise an implementer could resume from the wrong plan and use `NewPersistent` naively.

### What was tricky to build
- Reconciling the original plan, the second-pass design review, and the user's no-backwards-compatibility clarification. The final checklist resolves this by making raw markdown removal a frontend+backend app change rather than a transparent API-preservation change.

### What warrants a second pair of eyes
- Whether Phase C should remove `rawMarkdown` from `/api/notes/{slug}` immediately or keep it until after frontend changes are merged in the same commit. The plan currently updates both in one phase.

### What should be done in the future
- Execute Phase B first and record the code commit hash in the diary.

### Code review instructions
- Review `tasks.md` first, then `design/02-project-and-design-review-memory-index-search.md` to understand why the sequence changed.
- Validate with `go test ./...` before Phase B coding.

### Technical details
- Baseline command: `go test ./...`

---

## Step 8: Implement Phase B memory/reload instrumentation

Phase B makes memory behavior visible before deeper storage refactors. The implementation keeps runtime semantics unchanged: vaults and search indexes are still built exactly as before, but startup/reload phases now log Go heap statistics, and `/api/healthz` exposes the same memory counters for operators.

This phase also removes a small allocation from health/config paths by adding `Vault.Count()`. That is not the OOM fix, but it keeps the health endpoint cheap and gives later phases a cleaner count API.

### Prompt Context

**User prompt (verbatim):** (same as Step 7)

**Assistant interpretation:** Start implementing the resumable phase plan and commit each meaningful phase.

**Inferred user intent:** Make progress in reviewable chunks while preserving a diary that allows handoff/resume.

**Commit (code):** de5db6641581e987fe69b4825a09bed0f526fbf3 — "RETRO-MEMORY-012: add memory instrumentation"

### What I did
- Added `Vault.Count()` in `internal/vault/vault.go`.
- Updated `internal/api/api.go` config response to use `v.Count()` rather than `len(v.AllNotes())`.
- Added `currentMemoryStats()` and `logMemoryPhase()` in `internal/server/runtime.go`.
- Logged memory around load start, symlink resolution, vault load completion, search build completion, load completion, reload start/failure/swap.
- Extended `/api/healthz` in `internal/server/server.go` with heap fields: `heapAllocBytes`, `heapSysBytes`, `heapInuseBytes`, `nextGCBytes`, `numGC`.
- Added tests for `Vault.Count()` and health memory JSON.

### Why
- The design review flagged that the reload-triggered OOM is plausible but not yet measured phase-by-phase. These logs make the next prod run useful: we can see whether peak growth happens in parsing, HTML rebuild, search indexing, or swap.

### What worked
- `gofmt` completed.
- `go test ./...` passed.
- Pre-commit hook passed `GOWORK=off go test ./...`, `python3 -m unittest plugins/test_retro_plugin.py`, and `GOWORK=off golangci-lint run -c .golangci.yml -v`.

### What didn't work
- N/A.

### What I learned
- The health endpoint previously used `len(v.AllNotes())`, which allocated a full slice just to count notes. The new `Count()` API is a tiny cleanup but also a good example of separating API needs from vault iteration.

### What was tricky to build
- Avoiding a new dependency or complex metrics subsystem. The phase deliberately uses `runtime.ReadMemStats` and structured-ish log key/value text so it remains simple and safe.

### What warrants a second pair of eyes
- Whether the new `/api/healthz` JSON fields are acceptable for any external health consumers. The old fields remain present, but the response now uses `json.Encoder`, which adds a trailing newline.

### What should be done in the future
- Deploy this or a later phase and capture real prod logs across startup and reload.
- Continue to Phase C: remove raw markdown from the hot storage model and update frontend copy behavior.

### Code review instructions
- Start with `internal/server/runtime.go` for instrumentation.
- Then review `internal/server/server.go` `healthResponse` and `healthHandler`.
- Validate with `go test ./...`.

### Technical details
- Validation command: `gofmt -w internal/vault/vault.go internal/vault/vault_test.go internal/api/api.go internal/server/runtime.go internal/server/server.go internal/server/runtime_test.go && go test ./...`

---

## Step 9: Implement Phase C raw markdown removal

Phase C removes raw markdown from the hot in-memory `vault.Note` model. This deliberately changes the full note JSON shape: `GET /api/notes/{slug}` no longer includes `rawMarkdown`. The app still works because the only interactive feature that needs raw markdown, "Copy as Markdown", now fetches `GET /api/notes/{slug}/raw` on demand.

This phase is the first direct memory reduction. It does not change HTML caching or bleve indexing yet; it removes one full-vault text copy and makes raw source reads proportional to user demand.

### Prompt Context

**User prompt (verbatim):** (same as Step 7)

**Assistant interpretation:** Continue implementing the phase checklist, using the user's no-backwards-compatibility clarification to simplify the raw markdown API.

**Inferred user intent:** Reduce memory without preserving legacy API fields that the current app can stop depending on.

**Commit (code):** 499c6f7ac4aec26cb47800c373cc4bedd1e72ac6 — "RETRO-MEMORY-012: lazy-load raw markdown"

### What I did
- Removed `RawMarkdown` from `vault.Note` in `internal/vault/vault.go`.
- Added `Vault.ReadRaw(relPath)` using clean vault-relative path validation plus `os.OpenRoot`.
- Updated `getNoteRaw` in `internal/api/api.go` to read raw source from disk.
- Added backend tests for raw endpoint success, missing source 404, and full note JSON omitting `rawMarkdown`.
- Removed `rawMarkdown` from the frontend `Note` type.
- Updated `NoteRenderer` so Copy as Markdown fetches `/api/notes/{slug}/raw` before writing to the clipboard.
- Removed stale `rawMarkdown` fixture fields from static vault/story/SSR test data.

### Why
- Keeping raw markdown in every loaded note creates a full-vault text copy in heap. The raw content is only needed for explicit user actions, so it should be loaded on demand.

### What worked
- `go test ./...` passed.
- After installing web dependencies, `pnpm check` passed.
- `pnpm build` passed; Vite emitted only chunk-size warnings.
- Pre-commit hook passed `GOWORK=off go test ./...`, plugin unittest, `golangci-lint`, and `pnpm --dir web check`.

### What didn't work
- First `pnpm check` failed because `web/node_modules` was absent:
  - command: `cd web && pnpm check`
  - error: `sh: 1: tsc: not found` and `Local package.json exists, but node_modules missing`
- Fixed by running `cd web && pnpm install --frozen-lockfile`, then reran `pnpm check` successfully.

### What I learned
- The frontend had a real dependency on `note.rawMarkdown` for Copy as Markdown; removing the backend field required a coordinated frontend update.
- Static vault fixtures and SSR tests also encoded the old Note shape.

### What was tricky to build
- The raw reader needed to be safe against traversal and non-Markdown reads while still accepting normal note paths. I reused the existing vault-relative cleaning approach and `os.OpenRoot` pattern used by asset serving.

### What warrants a second pair of eyes
- `ReadRaw` currently rejects anything whose cleaned path does not end in `.md`; review whether this is sufficient and whether hidden markdown files should remain unreachable via normal note lookup.
- The Copy as Markdown button now has an async failure path; UI error handling is minimal.

### What should be done in the future
- Phase D should decouple search from HTML so search indexing no longer requires rendered HTML as the source body.

### Code review instructions
- Start with `internal/vault/vault.go` (`Note` and `ReadRaw`).
- Review `internal/api/api.go` for response-shape change and raw endpoint behavior.
- Review `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` for copy-on-demand behavior.
- Validate with `go test ./...`, `cd web && pnpm check`, and `cd web && pnpm build`.

### Technical details
- Validation commands:
  - `gofmt -w internal/vault/vault.go internal/api/api.go internal/api/api_test.go && go test ./...`
  - `cd web && pnpm install --frozen-lockfile && pnpm check`
  - `cd web && pnpm build`

---

## Step 10: Implement Phase D search-document split

Phase D decouples full-text search from rendered HTML. Previously search indexing called `stripHTML(note.HTML)`, which forced search to depend on UI-rendered content and allocated a stripped copy of each note's HTML during indexing. The new path builds a plain-text search document from raw Markdown on demand and passes that into the search index.

This phase does not yet move bleve out of memory. It prepares for persistent per-snapshot indexing by making the search input explicit and independent from the rendered note payload.

### Prompt Context

**User prompt (verbatim):** (same as Step 7)

**Assistant interpretation:** Continue implementing the phase checklist in commit-sized chunks.

**Inferred user intent:** Progress toward safe persistent search without the naive drop-in `NewPersistent` implementation.

**Commit (code):** 67f4856f08c75769273c5ced8f18dd3bfeed73ea — "RETRO-MEMORY-012: decouple search from rendered HTML"

### What I did
- Added exported `parser.PlainText(src []byte)` that strips frontmatter and Markdown syntax for search/indexing.
- Added `vault.SearchDocument` plus `Vault.SearchDocument(note)` and `Vault.SearchDocuments()`.
- Changed `search.New` and `search.NewPersistent` to ask the vault for search documents.
- Changed `search.Index` to accept `vault.SearchDocument` and removed `stripHTML`.
- Updated the watcher reload path to generate a search document for the changed note before indexing.
- Added a vault regression test proving the search body is plain Markdown-derived text, not rendered HTML.

### Why
- Search should not depend on `Note.HTML`; this blocks future lazy HTML and creates avoidable indexing transients.

### What worked
- `gofmt` completed.
- `go test ./...` passed.
- Pre-commit hook passed `GOWORK=off go test ./...`, plugin unittest, and `golangci-lint`.

### What didn't work
- The pre-commit hook created `plugins/__pycache__/`; I removed it before continuing.

### What I learned
- The parser already had private `stripMarkdown`/excerpt machinery, so exporting a small `PlainText` helper was enough for this phase.

### What was tricky to build
- Avoiding an import cycle: `search` already imports `vault`, so the shared search document type lives in `vault` rather than `search` for now. A future internal model package could clean this up if needed.

### What warrants a second pair of eyes
- `vault.SearchDocuments()` reads raw files for all notes during search build. This avoids steady-state storage but adds disk I/O during indexing. For a git-synced local worktree this should be fine, but prod measurements should confirm it.

### What should be done in the future
- Phase E should introduce closeable, per-snapshot persistent indexes without reusing one active directory in place.

### Code review instructions
- Start with `internal/search/search.go` to see the new indexing contract.
- Then review `internal/vault/vault.go` and `internal/parser/parser.go` for search document generation.
- Validate with `go test ./...`.

### Technical details
- Validation command: `gofmt -w internal/parser/parser.go internal/vault/vault.go internal/vault/vault_test.go internal/search/search.go internal/watcher/watcher.go && go test ./...`

---

## Step 11: Implement Phase E persistent per-snapshot search indexes

Phase E moves bleve out of the Go heap when `--search-index-path` is configured. The implementation avoids the naive "open and mutate one persistent directory" design by building each search index as part of an immutable runtime snapshot. Each snapshot has its own revision id and index directory; reload builds the next snapshot in a staging directory, swaps it atomically, then closes and removes the old snapshot index after a grace period.

The real-vault measurements show the change is meaningful. On `/home/manuel/code/wesen/go-go-golems/go-go-parc` (890 notes in this checkout), in-memory search startup ended at roughly 311 MB `heapAllocBytes` after search build. With persistent search enabled, startup ended at roughly 80 MB `heapAllocBytes`, using about 52 MB of disk for the index. A live reload with persistent search returned 204 and peaked around 144 MB `heapAllocBytes` while temporarily holding old and new snapshots.

### Prompt Context

**User prompt (verbatim):** "go ahead, and remember, no backwards compatibility. Use the instrumentation you added to measure memory usage as well. The repository btw is ../go-go-golems/go-go-parc/ if you want to try it out on the real scenario"

**Assistant interpretation:** Continue with Phase E, do not preserve old compatibility constraints, and use the real go-go-parc vault plus the Phase B instrumentation to quantify memory behavior.

**Inferred user intent:** Land the risky persistent-search architecture with evidence, not just code, and verify it against the production-like vault.

**Commit (code):** a13453839a6dd45d647540c611d40364400f8f4e — "RETRO-MEMORY-012: add persistent snapshot search indexes"

### What I did
- Added `(*search.Index).Close()` and made it idempotent.
- Changed `search.NewPersistent` to remove any existing index directory and build a fresh index, preventing stale deleted documents.
- Added `server.Snapshot` with revision, resolved root, vault, search index, index dir, and build time.
- Refactored `RuntimeState` to swap whole snapshots instead of independent vault/search fields.
- Added per-revision persistent index build directories under `<search-index-path>/snapshots/<revision>`.
- Built persistent indexes in `.building` dirs, renamed after success, then swapped the snapshot.
- Added delayed old snapshot close/cleanup (30s default, test-overridable).
- Added `--search-index-path` CLI plumbing through `serve.Settings` → `server.Config` → `RuntimeOptions`.
- Added tests for idempotent close, clean persistent rebuild with deleted docs, persistent reload removing stale search results, and old index dir cleanup.

### Why
- Persistent bleve reduces heap usage, but a shared mutable index directory would break snapshot consistency and retain stale docs. Per-snapshot indexes preserve the current atomic reload mental model.

### What worked
- `go test ./...` passed.
- `pnpm --dir web check` passed.
- Pre-commit hook passed `GOWORK=off go test ./...`, plugin unittest, and `golangci-lint` after removing the unused `loadVaultAndSearch` helper.
- Real-vault in-memory baseline command:
  - `timeout 35s go run ./cmd/retro-obsidian-publish serve --vault /home/manuel/code/wesen/go-go-golems/go-go-parc --port 18180 --serve-web=false --watch=false`
  - `load_search_done heapAllocBytes=311186216 heapSysBytes=389382144 notes=890 duration=7.753506218s`
- Real-vault persistent startup command:
  - `timeout 45s go run ./cmd/retro-obsidian-publish serve --vault /home/manuel/code/wesen/go-go-golems/go-go-parc --port 18181 --serve-web=false --watch=false --search-index-path /tmp/retro-persistent-search`
  - `load_search_done heapAllocBytes=79680464 heapSysBytes=384827392 notes=890 duration=6.55784007s`
  - disk index size: `52M /tmp/retro-persistent-search`
- Real-vault persistent reload command:
  - started server with `--reload-allow-loopback --search-index-path /tmp/retro-persistent-reload`
  - `curl -X POST http://127.0.0.1:18182/api/admin/reload` returned `204`
  - reload `load_search_done heapAllocBytes=143931976 heapSysBytes=598376448 notes=890 duration=10.274847824s`

### What didn't work
- First commit attempt failed in pre-commit lint because the snapshot refactor left `loadVaultAndSearch` unused:
  - error: `internal/server/runtime.go:111:6: func loadVaultAndSearch is unused (unused)`
  - fix: removed the leftover helper, reran `gofmt`, `go test ./...`, and `pnpm --dir web check`.
- Measurement nuance: `heapSysBytes` remains high or can grow after persistent indexing because the Go runtime reserves heap arenas and does not immediately return them to the OS. `heapAllocBytes` is the better before/after signal for live heap reduction.

### What I learned
- Persistent bleve materially reduces live heap for this vault: ~311MB after in-memory search build vs ~80MB after persistent startup.
- Reload still has a visible transient because it builds a second vault and persistent index before swap, but the measured peak (~144MB heapAlloc) is far below the previous in-memory steady search heap.

### What was tricky to build
- Preserving snapshot consistency while using a persistent directory. The solution is not to reuse one path; every snapshot gets a unique revision directory and cleanup happens only after the old snapshot is no longer active.
- Closing old search indexes cannot happen under the runtime lock and should not happen immediately in production because a request may have taken a snapshot just before the swap. The implementation uses a grace delay.

### What warrants a second pair of eyes
- The 30s old-snapshot cleanup delay is a pragmatic default. Review whether it should become configurable.
- `os.Rename(buildDir, finalDir)` happens while the bleve index handle is open. This works on Linux, which is the deployment target, but it is worth reviewing if cross-platform local dev becomes important.
- Search index dirs are unique per reload; if the process crashes mid-build, `.building` directories may remain. A startup cleanup for stale `.building` dirs could be added later.

### What should be done in the future
- Phase F should mount a writable `/data/search` in the Kubernetes deployment and pass `--search-index-path=/data/search`.
- Consider setting `GOMEMLIMIT` after measuring the deployed heap with persistent search enabled.

### Code review instructions
- Start with `internal/server/runtime.go` to review the snapshot lifecycle and index directory handling.
- Review `internal/search/search.go` for `Close()` and fresh persistent rebuild semantics.
- Review CLI plumbing in `cmd/retro-obsidian-publish/commands/serve/serve.go`.
- Validate with `go test ./...`, `pnpm --dir web check`, and the real-vault startup/reload commands above.

### Technical details
- Persistent index base path used for measurement: `/tmp/retro-persistent-search` and `/tmp/retro-persistent-reload`.
- Real vault path used for measurement: `/home/manuel/code/wesen/go-go-golems/go-go-parc`.

---

## Step 12: Deploy optimized images and verify production memory

Phase F moved the optimized code into production. I built and pushed both the Go app image and the SSR image with the same `sha-f434b60` tag, updated the hetzner-k3s deployment to use persistent search storage at `/data/search`, and verified the new pod in the cluster.

A deployment wrinkle appeared: the GHCR packages are public, but the old deployment specified an explicit image pull secret. That stale/bad credential caused GHCR to return `403 Forbidden` instead of allowing anonymous pulls. Removing `imagePullSecrets` from the pod template fixed the rollout because kubelet could then pull the public images anonymously.

### Prompt Context

**User prompt (verbatim):** "ok, go ahead"

**Assistant interpretation:** Proceed with deploying the memory-optimized publish-vault build to production.

**Inferred user intent:** Make the prod site healthy and verify the memory fix under real cluster conditions.

**Commit (deployment):** 0ebec8b5f841fb3cdde21f1b414cce242f40bc65 — "Deploy publish-vault memory optimized image"

**Commit (deployment fix):** a996fae588c32d45b4d3d1ef1a5f20e8410db1cc — "retro-obsidian-publish: pull public GHCR images anonymously"

### What I did
- Built and pushed `ghcr.io/go-go-golems/publish-vault:sha-f434b60`.
- Built and pushed `ghcr.io/go-go-golems/publish-vault-ssr:sha-f434b60`.
- Updated hetzner-k3s `gitops/kustomize/retro-obsidian-publish/deployment.yaml`:
  - app image -> `sha-f434b60`
  - ssr image -> `sha-f434b60`
  - added `search-index` `emptyDir`
  - mounted it at `/data/search`
  - added `--search-index-path /data/search`
- Pushed GitOps commit `0ebec8b`.
- Argo applied the image/arg changes but new pods hit `ErrImagePull` / `ImagePullBackOff`.
- Verified both GHCR packages are public and anonymous manifest checks work.
- Removed the stale pod-level `imagePullSecrets` from the deployment and pushed GitOps commit `a996fae`.
- Patched the live Deployment to remove `imagePullSecrets` immediately, matching desired GitOps state and unblocking the rollout.

### Why
- The code-side memory fix only helps production after the deployment uses the new image and enables persistent search index storage.

### What worked
- Argo eventually reported `sync=Synced health=Healthy op=Succeeded`.
- Final prod pod:
  - `retro-obsidian-publish-74968f989-vscxs`
  - `READY 3/3`
  - `RESTARTS 0`
  - app image `ghcr.io/go-go-golems/publish-vault:sha-f434b60`
  - ssr image `ghcr.io/go-go-golems/publish-vault-ssr:sha-f434b60`
- `kubectl top pod` showed about `167Mi` for the whole pod after rollout.
- Public health endpoint returned OK:
  - `https://parc.yolo.scapegoat.dev/api/healthz`
  - `notes=890`
  - `heapAllocBytes` observed around `41–68Mi` after startup/reload/GC.

### What didn't work
- First rollout of `sha-f434b60` failed with GHCR `403 Forbidden` pulls:
  - `failed to authorize: failed to fetch oauth token ... 403 Forbidden`
- Cause: the image is public, but the pod still forced use of `retro-obsidian-publish-ghcr-pull`. A bad explicit pull credential can prevent anonymous fallback.
- Fix: remove `imagePullSecrets` from the pod template because these GHCR packages are public.

### What I learned
- For public GHCR packages, an explicit stale image pull secret is worse than no secret. Kubelet authenticates and fails instead of pulling anonymously.
- The optimized app is now comfortably below the old 1536Mi limit in production: observed whole-pod memory ~167Mi, with app heap allocation far below local reload measurements.

### What was tricky to build
- Argo was mid-operation after the failing image pull, so the desired GitOps follow-up commit did not immediately update the live pod template. A manual `kubectl patch deployment ... remove /spec/template/spec/imagePullSecrets` matched the desired state and unblocked the rollout.

### What warrants a second pair of eyes
- Whether to remove the now-unused `vault-static-secret-image-pull.yaml` resource from kustomization in a later cleanup. It is no longer referenced by the pod, but the VSO resource still exists in manifests.
- Whether to reduce the `app` memory limit after a longer observation window. I left the limit at 1536Mi for safety.

### What should be done in the future
- Watch the pod over several git-sync cycles and after real user traffic.
- Consider `GOMEMLIMIT` only if heapSys/RSS remains higher than desired; current live heapAlloc/RSS do not require it.

### Code review instructions
- Review the hetzner-k3s deployment diff in commits `0ebec8b` and `a996fae`.
- Verify with:
  - `kubectl -n retro-obsidian-publish get pods`
  - `kubectl top pod -n retro-obsidian-publish`
  - `curl -fsS https://parc.yolo.scapegoat.dev/api/healthz`

### Technical details
- Final pod memory sample: `167Mi` whole pod.
- App health sample from public endpoint: `heapAllocBytes=41319664`, `heapSysBytes=505970688`, `heapInuseBytes=67567616`, `numGC=1655`.

---

## Step 13: Address PR #8 automated code review issues

This step tightened the implementation based on Codex review comments on PR #8. The changes remove an accidental full-vault plaintext allocation in the index builders, make admin reload safe when the file watcher is enabled, and restore static-vault Markdown copy/download behavior.

The review found real edge cases: `SearchDocuments()` was convenient but reintroduced the memory shape the ticket was trying to avoid, `--watch=true` could leave a watcher holding an index that reload later closed, and static builds had no Go `/raw` endpoint after full-note JSON stopped carrying raw Markdown.

### Prompt Context

**User prompt (verbatim):** "lAddress code review issues for https://github.com/go-go-golems/publish-vault/pull/8"

**Assistant interpretation:** Inspect the PR #8 review comments, patch the code, validate locally, and record the results.

**Inferred user intent:** Make the pull request mergeable by addressing the automated code-review findings without regressing the memory/OOM work.

**Commit (code):** 44bd91a0f08f142b69eb6142ecec6b3295c16ceb — "RETRO-MEMORY-012: address PR review feedback"

### What I did
- Fetched PR #8 review comments with `gh pr view 8` and `gh api repos/go-go-golems/publish-vault/pulls/8/comments --paginate`.
- Added `Vault.ForEachSearchDocument(fn)` so search indexing streams one Markdown-derived document at a time.
- Updated `search.New` and `search.NewPersistent` to use `ForEachSearchDocument` instead of `SearchDocuments()`.
- Kept `SearchDocuments()` as a convenience/test API but implemented it on top of the streaming iterator and documented that indexers should prefer the iterator.
- Added `search.ErrClosed` and nil guards to `Index`, `Delete`, and `Search` so accidental use after `Close()` returns a normal error instead of panicking.
- Made `watcher.VaultWatcher.Close()` idempotent with `sync.Once`.
- Changed admin reload wiring so `server.Run` disables the fs watcher once, before a reload swaps snapshots and schedules the old snapshot/index for close.
- Added optional `rawMarkdown?: string` to the frontend `Note` type.
- Restored bundled raw Markdown in static-vault notes and made `NoteRenderer` use it for Copy/View/Download .md actions when present.
- Added regression tests for closed-index operations and the reload pre-hook.

### Why
- Streaming search documents preserves the memory-reduction goal for large vaults.
- Disabling the watcher before admin reload prevents a watcher from mutating or calling into an index owned by an old snapshot that is about to be closed.
- Static builds need a local raw Markdown source because they intentionally do not have the Go `/api/notes/{slug}/raw` endpoint.

### What worked
- `go test ./...` passed.
- `pnpm --dir web check` passed.
- `pnpm --dir web build` passed, with only the existing Vite large chunk warnings.

### What didn't work
- I first ran the wrong root-level build command:
  - Command: `pnpm build`
  - Error: `ERR_PNPM_RECURSIVE_EXEC_FIRST_FAIL Command "build" not found`
- Correct command was:
  - `pnpm --dir web build`

### What I learned
- The persistent search path still needs careful review for transient allocations; moving the index to disk is not enough if preprocessing gathers all plain text into a slice first.
- The default `--watch=true` mode and explicit admin reload mode are different runtime models. If they are combined, reload must explicitly own watcher lifecycle.
- Static-mode behavior is easy to miss because the live API deliberately omits `rawMarkdown`, while static mode cannot fetch raw source from Go.

### What was tricky to build
- The watcher issue had two parts: the direct panic risk from using a closed `*search.Index`, and the ownership mismatch where the watcher still points at the old vault/search snapshot. The solution disables the watcher before reload and also makes closed index operations return `ErrClosed` as a defensive guard.
- The static-vault fix needed to restore raw content only for static notes, without reintroducing raw Markdown into live API responses. An optional TypeScript field keeps live notes lean while preserving standalone/static functionality.

### What warrants a second pair of eyes
- Whether disabling the watcher after the first admin reload is the right long-term UX for local `--watch=true` sessions. It is safe and explicit, but a future improvement could restart a watcher on the new snapshot instead.
- Whether `SearchDocuments()` should eventually be removed to prevent future indexing code from choosing the allocating API by accident.

### What should be done in the future
- If users rely on both `--watch=true` and admin reload, implement watcher restart on successful reload rather than one-shot disable.
- Add an end-to-end static build smoke test for Copy/View/Download .md actions if frontend test infrastructure grows.

### Code review instructions
- Start with `internal/search/search.go` and `internal/vault/vault.go` to verify search documents are streamed during indexing.
- Review `internal/server/server.go` and `internal/watcher/watcher.go` for watcher lifecycle behavior across reload.
- Review `web/src/vault/staticVault.ts`, `web/src/types/index.ts`, and `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` for static raw Markdown behavior.
- Validate with:
  - `go test ./...`
  - `pnpm --dir web check`
  - `pnpm --dir web build`

### Technical details
- PR comments addressed:
  - `internal/search/search.go`: stream search documents during persistent indexing.
  - `internal/server/runtime.go`: keep watched indexes alive or disable watcher across reload; implemented watcher disable before reload plus `ErrClosed` guards.
  - `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`: preserve Copy as Markdown for static vaults.

---

## Step 14: Address second PR #8 review pass

This step addressed the second Codex review pass on PR #8. The feedback focused on two correctness edges introduced by the persistent search and Markdown-derived search-body changes.

The first issue was that the persistent index was opened under a `.building` staging path and then its parent directory was renamed to the final snapshot path. The second issue was that Markdown plaintext extraction dropped inline code entirely, which would make command names, API names, and identifiers disappear from search when they only appeared inside backticks.

### Prompt Context

**User prompt (verbatim):** "Address the new comments in code review in https://github.com/go-go-golems/publish-vault/pull/8"

**Assistant interpretation:** Fetch the latest PR #8 review comments, patch the implementation, validate, update docs, and push the branch.

**Inferred user intent:** Keep the PR mergeable and preserve the memory work while fixing newly identified correctness regressions.

**Commit (code):** TBD — will be filled after commit.

### What I did
- Fetched the latest PR #8 reviews and comments with `gh pr view 8` and `gh api repos/go-go-golems/publish-vault/pulls/8/comments --paginate`.
- Added `search.OpenPersistent(indexPath)` for opening an existing Bleve index directory.
- Changed persistent snapshot index construction to:
  - build under `<revision>.building/index`,
  - close the staging index,
  - rename `<revision>.building` to `<revision>`,
  - reopen the index from `<revision>/index`,
  - return the reopened final-path index to the runtime.
- Changed Markdown plaintext extraction to unwrap inline code with `` `([^`]+)` -> `$1` `` rather than deleting inline code spans.
- Added regression coverage:
  - `TestBuildSearchIndexReopensPersistentIndexAtFinalPath`
  - `TestPlainTextPreservesInlineCode`
  - expanded `TestSearchDocumentsUsePlainMarkdownBody` to include inline code.

### Why
- A watcher or any live update path should mutate an index whose filesystem path matches the final per-revision snapshot directory, not a path that was created under `.building` and then moved.
- Search should continue finding notes by command names, API names, and identifiers written as inline code.

### What worked
- `go test ./...` passed.
- `pnpm --dir web check` passed.
- `pnpm --dir web build` passed with only the existing Vite large-chunk warnings.

### What didn't work
- N/A

### What I learned
- Moving a directory containing an already-open persistent index is subtle even if basic searches still appear to work; returning an index opened at the final path is clearer and safer.
- Search-body normalization must preserve semantically important Markdown inline forms such as code spans, not only prose.

### What was tricky to build
- The persistent-index fix needed to keep atomic staging semantics while avoiding an open handle rooted at the staging path. Closing before rename and reopening after rename keeps failure cleanup straightforward and preserves the final snapshot directory contract.
- The inline-code fix had to preserve the code text without reintroducing Markdown backticks into the search body.

### What warrants a second pair of eyes
- Whether fenced code blocks should receive more structured plaintext treatment later. This fix addresses the reviewed inline-code regression specifically.
- Whether `OpenPersistent` should remain exported long term or be hidden behind a server-local helper if no other package needs it.

### What should be done in the future
- Add broader Markdown-to-search-body tests for fenced code blocks, HTML entities, and Obsidian-specific syntax if search quality becomes a larger focus.

### Code review instructions
- Review `internal/server/runtime.go:buildSearchIndex` for the close/rename/reopen sequence.
- Review `internal/parser/parser.go:stripMarkdown` for inline-code unwrapping.
- Validate with:
  - `go test ./...`
  - `pnpm --dir web check`
  - `pnpm --dir web build`

### Technical details
- New review comments addressed:
  - `internal/server/runtime.go`: keep the open index at its final path.
  - `internal/vault/vault.go` / `internal/parser/parser.go`: preserve inline code text in search body.
