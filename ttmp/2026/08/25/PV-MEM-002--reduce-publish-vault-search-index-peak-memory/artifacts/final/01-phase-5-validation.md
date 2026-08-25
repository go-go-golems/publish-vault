---
Title: PV-MEM-002 Phase 5 validation and packaging evidence
Ticket: PV-MEM-002
Status: active
Topics:
    - memory
    - regression
    - deployment
    - ci
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Tightened persistent-fixture budgets, generated scaling results, container packaging smoke, and final local validation gates.
LastUpdated: 2026-08-25T19:20:00-04:00
WhatFor: Reviewing Phase 5 regression, packaging, and local delivery evidence before PR completion.
WhenToUse: Read during final PV-MEM-002 review or when changing fixture budgets and container memory guidance.
---

# PV-MEM-002 Phase 5 validation and packaging evidence

## Regression budgets

`TestGeneratedFixtureMemoryBudget` now constructs the same persistent search path used by production full-snapshot reloads and closes its index explicitly. The 160-document × 8 KiB fixture passed ten consecutive normal runs and the full race suite. Final limits are:

```text
run peak heap <32 MiB
run peak RSS <160 MiB
search_index peak heap <32 MiB
search_index peak RSS <160 MiB
```

The heap limit is half the previous 64 MiB. An initial proposal to halve RSS from 192 MiB to 96 MiB passed ten normal runs but failed under race instrumentation at 139,022,336 bytes. It was rejected rather than weakening race coverage. The final 160 MiB limit is 16.7% tighter than the old value and retains 20.7% headroom over that observed race maximum. A focused rerun under `-race` passed at 124,477,440 bytes RSS.

Five public generated scaling cases span 100–1,000 documents and 1–32 KiB payloads. The content-free results and privacy audit are in `../generated-scaling/`; all 428 retained events passed. These results demonstrate that total retained application state scales with source bytes while the accepted search batch bounds staged work.

## Docker and Compose

A production image built successfully from the repository Dockerfile:

```text
image: publish-vault:pv-mem-002
image id: sha256:6cb62f01f84c4237cb8fe1e4f6e75e7c5d00d9750d005333614efeda5c9d49cc
```

The image ran with a hard 512 MiB memory and swap limit, read-only example-vault mount, persistent search directory, private metrics listener, and embedded web UI. Evidence observed:

```text
/api/healthz: ok=true, notes=6
/metrics: measure_run* metrics present
/: embedded HTML present
container OOMKilled=false
container-derived GOMEMLIMIT=456,340,275 bytes (85% of 512 MiB)
```

The container was removed after inspection. `docker compose config` initially emitted the standard warning that top-level Compose `version: "3.9"` is obsolete. Removing only that ignored key changed no service behavior and made configuration validation warning-free.

## Final local gates

The fresh final sequence recorded in `local-validation.txt` passed:

- `go generate ./...`;
- `make ci-check` (format, unit/integration tests, lint, frontend typecheck/build, and repository checks);
- `GOWORK=off go test -race ./... -count=1`;
- Linux amd64 and Darwin arm64 CGO-disabled builds;
- `docker compose config`;
- ten repeated persistent-fixture budget runs;
- fifty repetitions of `pkg/search` tests;
- `git diff --check`.

Earlier Phase 3 evidence separately records direct `gosec ./...`, `govulncheck ./...`, frontend gates, and the full persistent lifecycle/failure test set. The final `make ci-check` reran the repository-owned lint and test gates after the Phase 5 budget edits.

## Review boundary

This report proves local Phase 5 gates and packaging. PR checks, final ticket doctor, clean-worktree state, commit identity, and merge state must be recorded in the final completion audit after the Phase 5 commit is pushed. Until then this ticket remains active and no completion slip may be printed.
