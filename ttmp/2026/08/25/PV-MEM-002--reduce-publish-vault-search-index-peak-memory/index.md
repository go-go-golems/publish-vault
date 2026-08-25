---
Title: Reduce publish-vault search-index peak memory
Ticket: PV-MEM-002
Status: active
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
RelatedFiles: []
ExternalSources: []
Summary: Evidence-first profiling and implementation ticket for reducing the persistent Bleve search-index peak from its current 391 MB heap and 483 MB RSS median while preserving search behavior and atomic snapshot reloads.
LastUpdated: 2026-08-25T18:00:00-04:00
WhatFor: Coordinating baseline refresh, peak attribution, bounded search-index experiments, correctness proof, regression budgets, and rollout guidance.
WhenToUse: Start here before profiling or changing publish-vault search-document generation, Bleve construction, mapping, snapshot publication, or memory limits.
---

# Reduce publish-vault search-index peak memory

## Overview

MEASURE-001 established that persistent Bleve indexing is materially better than in-memory indexing for the 3,395-note personal-vault workload, but `search_index` still dominates at a median 391,219,024-byte heap peak and 482,586,624-byte RSS peak. This ticket attributes that peak to concrete types and lifetimes, tests one falsifiable optimization hypothesis at a time, and accepts a change only after repeated memory, throughput, search-equivalence, snapshot, privacy, and CI validation.

The preferred initial target is median process RSS below 400 MB. The target is not a license to weaken atomic snapshot replacement, rollback, persistent-index freshness, or query behavior.

## Key Links

- **Primary guide**: [Search Index Memory Analysis, Design, and Implementation Guide](./design-doc/01-search-index-memory-analysis-design-and-implementation-guide.md)
- **Diary**: [Investigation Diary](./reference/01-investigation-diary.md)
- **Tasks**: [tasks.md](./tasks.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

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
