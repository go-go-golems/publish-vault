---
Title: "Investigation diary — retro-obsidian-publish prod OOM (RETRO-MEMORY-012)"
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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological investigation diary for diagnosing the retro-obsidian-publish prod OOM and writing the intern deliverable."
LastUpdated: 2026-07-05T00:00:00Z
WhatFor: "Record what was tried, what worked, and what to validate"
WhenToUse: "Read before resuming work on RETRO-MEMORY-012"
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

**Commit (code):** pending — "RETRO-MEMORY-012: lazy-load raw markdown"

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
