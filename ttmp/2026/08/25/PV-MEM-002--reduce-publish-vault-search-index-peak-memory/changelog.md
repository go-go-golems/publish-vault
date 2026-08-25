# Changelog

## 2026-08-25

- Initial workspace created


## 2026-08-25

Created evidence-first search-index optimization ticket, published two prerequisite Obsidian deep dives in go-go-parc commit 5f9f18c, and wrote the intern-oriented architecture, profiling, experiment, API, implementation, and validation guide.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/measure/ttmp/2026/08/25/MEASURE-001--standalone-process-memory-measurement-local-optimization-and-metrics-toolkit/artifacts/phase5-baseline/summary.json — Baseline measurements and accepted persistent-index decision
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/search/search.go — Current broad search_index implementation and first profiling target
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/server/runtime.go — Snapshot lifecycle and persistent-index correctness boundary


## 2026-08-25

Validated the complete intern handoff with docmgr doctor and Markdown checks, then uploaded the index, design guide, and strict diary as PV-MEM-002 Search Index Memory Guide.pdf to /ai/2026/08/25/PV-MEM-002.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/design-doc/01-search-index-memory-analysis-design-and-implementation-guide.md — Primary uploaded implementation handoff
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/reference/01-investigation-diary.md — Chronological prompt, report, design, validation, and delivery evidence

## 2026-08-25

Phase 0 refreshed the clean pinned personal-vault baseline: 2030 notes and 76.9 MB source, with 826.1 MB median heap and 1.034 GB median RSS; search_index remains dominant and 4693 canonical events passed the privacy audit.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/baseline-current/01-baseline-report.md — Workload identity, distribution table, cgroup caveat, and audit result
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/01-run-persistent-baseline.sh — Three-run persistent baseline harness

## 2026-08-25

Phase 1 attributed search_index: post-GC retained heap grew only 33.7 MB while search allocated 51.65 GB, expanded HeapSys by 611.9 MB, and drove repeated one-document Scorch segment/merge work; selected bounded Bleve batches for Phase 2.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/attribution/01-attribution-report.md — Evidence and lifetime inventory
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/profile-search-index/main.go — Reproducible private profile harness

## 2026-08-25

Phase 2 confirmed bounded Bleve batching: selected 16 documents/1 MiB after one exploratory run reduced heap 40%, RSS 28%, allocation 62%, duration 31%, and index bytes 15% versus one-document updates; production defaults remain unchanged pending implementation proof.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/search/search.go — Experimental bounded batch path
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/batch-matrix/summary.json — Machine-readable variant comparison

## 2026-08-25

Phase 3 wired internal persistent defaults of 16 documents/1 MiB, preserved in-memory and incremental behavior, compared 20 real-vault queries across 16725 complete results, and passed generation, local CI, race, lint, security, frontend, and Darwin build gates.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/server/runtime.go — Accepted production integration
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/implementation/01-implementation-validation.md — Correctness and validation evidence

## 2026-08-25

Phase 4 repeated proof: three complete candidate runs reduced median heap 34.27%, RSS 21.52%, duration 48.13%, and index size 2.35%; an isolated 1 GiB cgroup run succeeded at 988 MB peak current and all retained events passed privacy audits.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/candidate-current/01-candidate-proof.md — Repeated acceptance report
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/finite-cgroup/result.json — Finite-cgroup operational boundary

## 2026-08-25

Phase 5 changed the generated budget to persistent production indexing, tightened heap from 64 to 32 MiB and race-compatible RSS from 192 to 160 MiB, validated five public scaling cases, built and smoked the production image under a 512 MiB cgroup, removed the obsolete Compose version key, and passed all fresh local gates.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/server/memory_budget_test.go — Persistent generated-fixture setup and explicit index close
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/server/testdata/generated-fixture-memory-budget.json — Tightened reviewed regression thresholds
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/final/01-phase-5-validation.md — Regression packaging failure-triage and local validation evidence
