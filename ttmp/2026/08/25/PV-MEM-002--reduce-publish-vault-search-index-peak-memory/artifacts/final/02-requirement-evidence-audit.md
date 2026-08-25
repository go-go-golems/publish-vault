---
Title: PV-MEM-002 final requirement-to-evidence audit
Ticket: PV-MEM-002
Status: complete
Topics:
    - memory
    - search
    - bleve
    - regression
    - ci
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Final mapping of every PV-MEM-002 objective and safety constraint to committed evidence and fresh validation.
LastUpdated: 2026-08-25T19:25:00-04:00
WhatFor: Deciding whether PV-MEM-002 is actually complete without relying on probable or deferred claims.
WhenToUse: Read before merging or reopening the search-index memory optimization.
---

# PV-MEM-002 final requirement-to-evidence audit

Audit target: PR #25 head `5f5600da7265d6aafcfc4514d3041449d70796f2` plus this final bookkeeping commit. Every required technical gate below is **PASS**. No implementation requirement is deferred.

| Requirement | Result | Concrete evidence |
|---|---|---|
| Reproducible workload identity | PASS | `artifacts/baseline-current/workload-identity.json`, `manifest.json`, and `01-baseline-report.md`: detached clean vault `5f9f18c`, 2,030 notes, 76,903,490 source bytes, pinned config and binary hashes. |
| Three-run baseline | PASS | `artifacts/baseline-current/summary.json`: three complete 100 ms persistent runs; median 826,146,848 B heap, 1,033,994,240 B RSS, 166.42 s. |
| Aligned JSONL and receipts | PASS | Baseline, attribution, candidate, and finite-cgroup artifact directories retain versioned measure traces, summaries, receipts, and metadata aligned to named phases. |
| Secure heap attribution | PASS | `artifacts/attribution/01-attribution-report.md` and `summary.json`: forced-GC checkpoints at 0/25/50/75/100%, reviewed aggregate pprof tables, only +33.7 MB retained heap versus 51.65 GB allocation churn. Raw profiles were deleted and a repository scan finds no `.pprof` or `.pb.gz`. |
| RSS, smaps, and cgroup attribution | PASS | Attribution checkpoints retain heap/RSS, smaps rollups, allocator state, and cgroup values; the report distinguishes shared unlimited host cgroup accounting from application ownership. |
| Object-lifetime inventory | PASS | Attribution report maps vault notes/rendered HTML, transient search documents, Bleve batches/segments, old/new snapshots, OS arenas, and file-backed residency to ownership and release boundaries. |
| Falsifiable hypothesis selection | PASS | Phase 1 selected bounded Bleve batching only after evidence; `artifacts/batch-matrix/` compares one-document updates with 4/16/32/64/128-document variants and byte bounds. |
| Largest accepted bounded optimization | PASS | `pkg/search/search.go` and `pkg/server/runtime.go`: persistent full snapshots use batches bounded to 16 documents and 1 MiB. Exploratory 16-document result was best on the reviewed memory/throughput trade-off. |
| Preserve in-memory and incremental behavior | PASS | Defaults on generic/in-memory indexing remain one-document; production bounds are internal to persistent full-snapshot construction. Tests pin this boundary. |
| Search equivalence | PASS | `artifacts/implementation/real-vault-search-equivalence.json`: 20 real queries, 16,725 complete results, equal identity/fields, max score delta `3.47e-18`; unit tests cover tied scores and flush progress. |
| Persistent freshness and deletion | PASS | Existing and Phase 3 runtime tests rebuild full persistent snapshots, observe updated content, remove deleted-note results, and publish only the completed revision. |
| Atomic swap and snapshot pairing | PASS | Runtime tests verify vault/search revision pairing and no partial publication; `RuntimeState.Reload` still swaps one completed snapshot under lock. |
| Rollback and failure paths | PASS | Tests inject document conversion, batch, and index failures; failed candidate indexes close/remove and the prior snapshot remains active. |
| Reload serialization | PASS | `TestReloadIsSerialised` and full race suite pass. No concurrent candidate publication was introduced. |
| Delayed cleanup | PASS | Runtime tests retain old index paths through the grace boundary and then observe cleanup; generated fixture explicitly closes its persistent index with no temporary-directory cleanup failures. |
| Three-run candidate comparison | PASS | `artifacts/candidate-current/comparison.json` and `01-candidate-proof.md`: median heap −34.27%, RSS −21.52%, duration −48.13%, throughput +92.80%, index bytes −2.35%; all baseline/candidate ranges are retained. |
| Finite cgroup evidence | PASS | `artifacts/finite-cgroup/result.json`: isolated 1 GiB hard limit, 912,680,550 B Go soft limit, 623,229,576 B heap, 803,254,272 B RSS, 988,028,928 B cgroup current, 2,030/2,030 documents, successful exit. |
| Operational guidance | PASS | README documents persistent indexing, bounded behavior, measurement, and memory-limit semantics; candidate report states 1 GiB is a proven tight boundary, not comfortable guidance, and sub-400 MB requires a separate lazy-render/lifetime design. |
| Generated scaling | PASS | `artifacts/generated-scaling/summary.json` covers five cases across 100–1,000 documents and 1–32 KiB payloads; report explains source-byte and retained-state scaling. |
| Memory regression budgets | PASS | Persistent generated fixture runs at 160 × 8 KiB; thresholds tighten heap 64→32 MiB and race-compatible RSS 192→160 MiB. Ten normal repeats, focused race, and full race pass. The rejected 96 MiB race failure is documented. |
| Workload privacy | PASS | Baseline audit: 4,693 events; candidate audit: 2,687; finite cgroup: 841; generated scaling: 428. All content audits pass. No private server logs, indexes, vault data, or raw profiles are committed or uploaded. |
| Formatting and generation | PASS | Fresh `go generate ./...`, formatter/linter gates, `git diff --check`, and pre-commit hooks pass; `artifacts/final/local-validation.txt`. |
| Unit/integration/race/stress | PASS | Fresh `make ci-check`, `go test -race ./... -count=1`, fixture ×10, search package ×50, and pre-push tests pass. |
| Lint, typecheck, frontend | PASS | `make ci-check`, pre-commit, and pre-push reran golangci/glazed lint plus `pnpm check`; Phase 3 also records frontend production build. |
| Security | PASS | Direct local gosec/govulncheck passed; fresh PR GoSec, vulnerability, dependency, CodeQL, and TruffleHog checks all passed. |
| Linux/Darwin builds | PASS | Fresh CGO-disabled Linux amd64 and Darwin arm64 builds pass in `local-validation.txt`. |
| Docker and Compose | PASS | Production image `sha256:6cb62f...c9d49cc` built; 512 MiB smoke served health, private metrics, and embedded UI with `OOMKilled=false`; warning-free `docker compose config` passed. |
| PR and review | PASS | `artifacts/final/pr-state.json`: PR #25 is mergeable at `5f5600d`; all 12 checks are SUCCESS/SKIPPED. Codex reviewed that exact commit and reported no major issues. Main has no branch-protection approval requirement. |
| Diary and docmgr bookkeeping | PASS | Strict diary Steps 1–10 preserve prompts, commands, failures, decisions, commits, review instructions, risks, and follow-ups; tasks/changelog/index/relations updated; final `docmgr doctor --ticket PV-MEM-002 --stale-after 30` passes. |
| Master and phase slips | PASS | Preserved `00-master-plan.yaml`, six phase-start slips (`01`, `03`, `05`, `07`, `09`, `11`), and six post-gate completion slips (`02`, `04`, `06`, `08`, `10`, `12`). Each was physically printed through the Almanach service. |
| No prohibited implementation residue | PASS | Fresh searches and diff review find no new TODO/FIXME placeholder, compatibility shim, duplicate indexing path, global/private metric label, or content-bearing trace field. The policy is one internal persistent default with existing generic behavior intact. |
| Coherent commits | PASS | `5932110` baseline; `ceddf95` attribution; `c8450fa` experiments; `34ec4b9` implementation; `a4e4aa1` repeated proof; `5f5600d` regression/packaging; final evidence commit follows. |

## Decision

PV-MEM-002 meets the accepted outcome. The original sub-400 MB RSS preference was a stretch target, not a correctness gate; evidence proves it is not reachable by batch tuning alone. The implemented bounded batch is materially better on heap, RSS, duration, throughput, and index size without weakening search or snapshot semantics.

The branch must be clean and synchronized after this audit/slip commit, then PR #25 can be merged. The merge commit and post-merge clean-main state are delivery evidence, not a reason to mutate the already reviewed implementation.
