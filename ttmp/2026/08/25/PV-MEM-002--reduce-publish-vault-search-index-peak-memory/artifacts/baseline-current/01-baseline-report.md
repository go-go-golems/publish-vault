---
Title: PV-MEM-002 refreshed persistent-index baseline
Ticket: PV-MEM-002
Status: complete
Topics:
    - memory
    - profiling
    - search
    - bleve
    - performance
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Clean pinned three-run persistent-index baseline and content/privacy validation for the 2030-note 76.9 MB source workload.
LastUpdated: 2026-08-25T18:00:00-04:00
WhatFor: Establishing the before side of PV-MEM-002 memory, duration, and index-size comparisons.
WhenToUse: Read before comparing a candidate optimization or interpreting the retained baseline artifacts.
---

# PV-MEM-002 refreshed persistent-index baseline

## Workload identity

- Vault commit: `5f9f18ca7791ba2ddeb8a2528e3c279e6ae5f75a`
- Vault worktree: clean
- `.vault-ignore` SHA-256: `d39336e4bbba2e47c024bdec30b17659710c04834f8da2d60f52862b81393521`
- `.publish/config.yaml`: absent (empty default configuration)
- Markdown candidates: 2,036
- Published notes: 2,030
- Candidate source bytes: 76,921,819
- publish-vault commit: `8648cfcd1690c086010fbc5a64d27fe0f5ad6a9c`
- measure commit: `a3f4b045b5d204101e17e35458de9b8955d71772`
- Sampling interval: 100 ms

This is the pinned PV-MEM-002 baseline. It supersedes the MEASURE-001 baseline for before/after comparisons because that older run used a dirty vault worktree with 3,396 candidates and 20,938,723 source bytes. The two workloads are not directly comparable.

## Three-run results

| Metric | Minimum | Median | Maximum |
|---|---:|---:|---:|
| Total duration | 130.27 s | 166.42 s | 172.93 s |
| Peak Go heap | 805,533,208 B | 826,146,848 B | 857,792,960 B |
| Peak process RSS | 958,763,008 B | 1,033,994,240 B | 1,051,410,432 B |
| Persistent index size | 204,434,855 B | 210,012,540 B | 211,134,662 B |
| `search_index` duration | 87.92 s | 107.41 s | 115.67 s |
| `search_index` peak heap | 805,533,208 B | 826,146,848 B | 857,792,960 B |
| `search_index` peak RSS | 958,763,008 B | 1,033,994,240 B | 1,051,410,432 B |

`search_index` is the dominant phase for both median heap and median RSS in every run.

The cgroup source is reported as unlimited and describes a shared host-level cgroup. Its 7.18 GB median current peak is retained as truthful source evidence but must not be interpreted as publish-vault process memory. Process RSS is the useful external baseline for this environment. Later container experiments must provide an isolated finite cgroup.

## Artifact and privacy validation

- Three schema-v1 canonical JSONL traces decoded and summarized successfully.
- Three receipts record runtime, process, and smaps sources as available.
- `privacy-audit.json` inspected 4,693 canonical events.
- The only annotation keys are `processed_notes`, `processed_bytes`, and `total_bytes`.
- No path, slug, title, body, Markdown, excerpt, tag, command, environment, repository marker, or absolute home path was found.
- Binary and artifact SHA-256 values are recorded in `artifact-manifest.json`.

Raw note content and heap profiles are not present in this directory.
