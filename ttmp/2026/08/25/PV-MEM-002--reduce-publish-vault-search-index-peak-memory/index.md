---
Title: Reduce publish-vault search-index peak memory
Ticket: PV-MEM-002
Status: complete
Topics:
    - memory
    - profiling
    - search
    - bleve
    - performance
    - reload
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://docker-compose.yml
      Note: Warning-free production Compose configuration
    - Path: repo://pkg/server/memory_budget_test.go
      Note: Persistent generated-fixture production-path regression test
    - Path: repo://pkg/server/testdata/generated-fixture-memory-budget.json
      Note: Reviewed heap and race-compatible RSS thresholds
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/final/02-requirement-evidence-audit.md
      Note: Final objective-to-evidence completion mapping
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/final/pr-state.json
      Note: Exact current-head PR checks mergeability and review snapshot
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/06-run-generated-scaling.sh
      Note: Deterministic generated scaling harness
ExternalSources: []
Summary: Evidence-backed bounded Bleve batching reduced the refreshed workload's median peak heap 34.27 percent and RSS 21.52 percent while preserving search and atomic snapshot behavior; final PR delivery remains active.
LastUpdated: 2026-08-25T19:20:00-04:00
WhatFor: Coordinating baseline refresh, peak attribution, bounded search-index experiments, correctness proof, regression budgets, and rollout guidance.
WhenToUse: Start here before profiling or changing publish-vault search-document generation, Bleve construction, mapping, snapshot publication, or memory limits.
---



# Reduce publish-vault search-index peak memory

## Overview

The refreshed 2,030-note, 76.9 MB workload established a median 826,146,848-byte heap and 1,033,994,240-byte RSS peak. Aligned profiles found 51.65 GB of indexing allocation churn but only 33.7 MB post-GC retained growth, leading to a bounded Bleve batch design rather than an unsafe snapshot-lifetime change. The accepted 16-document/1 MiB persistent policy reduced repeated median heap 34.27%, RSS 21.52%, and duration 48.13% while preserving complete search results and atomic snapshot behavior.

The preferred sub-400 MB RSS target proved unattainable through batching alone because required rendered vault state and file-backed/runtime residency remain. The evidence rejects weakening atomic snapshot replacement, rollback, persistent-index freshness, or query behavior to chase that stretch target.

## Key Links

- **Primary guide**: [Search Index Memory Analysis, Design, and Implementation Guide](./design-doc/01-search-index-memory-analysis-design-and-implementation-guide.md)
- **Diary**: [Investigation Diary](./reference/01-investigation-diary.md)
- **Repeated candidate proof**: [Candidate proof](./artifacts/candidate-current/01-candidate-proof.md)
- **Phase 5 validation**: [Regression and packaging evidence](./artifacts/final/01-phase-5-validation.md)
- **Tasks**: [tasks.md](./tasks.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete — all local, CI, review, privacy, delivery, and evidence gates passed**

## Topics

- memory
- profiling
- search
- bleve
- performance
- reload

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
