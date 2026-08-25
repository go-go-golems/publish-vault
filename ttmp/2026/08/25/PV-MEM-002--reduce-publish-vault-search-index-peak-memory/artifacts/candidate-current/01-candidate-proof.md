---
Title: PV-MEM-002 repeated candidate proof
Ticket: PV-MEM-002
Status: complete
Topics:
    - memory
    - search
    - bleve
    - performance
    - reload
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Three-run complete-server proof that bounded persistent batching reduces median heap 34%, RSS 22%, duration 48%, and index bytes 2% on the pinned workload without shifting the dominant phase.
LastUpdated: 2026-08-25T19:00:00-04:00
WhatFor: Final before/after memory and throughput acceptance evidence for the selected implementation.
WhenToUse: Read before changing budgets, resource guidance, or approving PV-MEM-002.
---

# PV-MEM-002 repeated candidate proof

## Workload and method

Baseline and candidate both use:

```text
vault commit: 5f9f18ca7791ba2ddeb8a2528e3c279e6ae5f75a
vault worktree: clean/detached
.vault-ignore SHA-256: d39336e4bbba2e47c024bdec30b17659710c04834f8da2d60f52862b81393521
.publish/config.yaml: absent
Markdown candidates: 2,036
published notes: 2,030
candidate source bytes: 76,921,819
sample interval: 100 ms
runs per side: 3
```

Each run starts the complete server with a fresh persistent search directory, waits for health readiness, records all vault/search/publication/swap phases, records index directory bytes, terminates gracefully, decodes the trace, and removes temporary logs/indexes. Baseline commit is `8648cfc`; candidate implementation commit is `34ec4b9`.

## Median comparison

| Metric | Baseline median | Candidate median | Change |
|---|---:|---:|---:|
| Peak Go heap | 826,146,848 B | 543,066,520 B | **-34.27%** |
| Peak process RSS | 1,033,994,240 B | 811,429,888 B | **-21.52%** |
| Complete load duration | 166.42 s | 86.32 s | **-48.13%** |
| Throughput | 12.20 notes/s | 23.52 notes/s | **+92.80%** |
| Persistent index bytes | 210,012,540 B | 205,077,678 B | **-2.35%** |

Absolute median reductions:

```text
heap:     283,080,328 bytes
RSS:      222,564,352 bytes
duration: 80,103,375,651 ns
index:      4,934,862 bytes
```

## Candidate ranges

| Metric | Minimum | Median | Maximum |
|---|---:|---:|---:|
| Complete duration | 75.69 s | 86.32 s | 95.12 s |
| Peak heap | 534,493,424 B | 543,066,520 B | 551,111,344 B |
| Peak RSS | 792,576,000 B | 811,429,888 B | 829,358,080 B |
| Index bytes | 205,035,119 B | 205,077,678 B | 207,081,006 B |
| `search_index` duration | 40.95 s | 49.35 s | 52.64 s |

The candidate ranges do not overlap the baseline heap range (805.5–857.8 MB) and barely overlap neither RSS range (baseline 958.8–1,051.4 MB). This is larger than sampling/run-order noise.

## Phase shape

`search_index` remains the dominant heap and RSS phase, so the change did not move a larger peak into publication or swap.

Candidate median phase peaks:

```text
search_index: heap 543,066,520 B; RSS 811,429,888 B
index_publish: heap 321,150,176 B; RSS 712,777,728 B
snapshot_swap: heap 328,797,880 B; RSS 596,049,920 B
```

All runs processed exactly 2,030 search documents. `index_publish` completed close, rename, and reopen; `snapshot_swap` completed one snapshot.

## Finite-cgroup proof

The exact candidate binary also ran in `debian:bookworm-slim` with:

```text
memory.max: 1,073,741,824 bytes (1 GiB)
memory.swap.max constrained with --memory-swap=1g
network: none
vault mount: read-only
fresh /tmp persistent index
```

publish-vault derived:

```text
GOMEMLIMIT = 912,680,550 bytes (85% headroom policy)
```

The run completed without OOM or restart:

```text
duration: 81.92 s
search duration: 47.57 s
peak heap: 623,229,576 B
peak RSS: 803,254,272 B
peak cgroup current: 988,028,928 B
cgroup utilization at sampled peak: 92.0%
index bytes: 201,409,031 B
```

This proves finite cgroup collection and successful completion at 1 GiB. It also shows that 1 GiB has limited operational headroom: cgroup current came within 85.7 MB of the hard limit. Do not recommend a production limit below 1 GiB for this workload. Rollout overlap and node scheduling require separate capacity headroom.

## Privacy and artifact validation

- Three candidate traces contain 2,687 canonical events.
- The finite-cgroup trace contains 841 events.
- Only `processed_notes`, `processed_bytes`, and `total_bytes` annotations are present.
- No private path, note slug, title, body, Markdown, excerpt, tag, command, or repository marker was found.
- Raw server logs and search directories were removed.
- Binary and trace/receipt SHA-256 values are retained in the candidate manifest.

## Acceptance conclusion

The selected bounded batching implementation produces a repeatable, material improvement on the pinned real workload. It reduces both reachable/transient heap pressure and externally observed RSS while nearly doubling complete-load throughput. Search semantics and snapshot lifecycle remain covered separately by Phase 3 equivalence and repository tests.

The original preferred sub-400 MB RSS target is not achieved. Phase 1 established approximately 197 MB of retained rendered HTML before search plus substantial runtime arena and file-backed residency. Reaching sub-400 MB would require a separate render-cache/lazy-HTML design or a different index/file-residency strategy, not further unreviewed batch tuning. PV-MEM-002's evidence-backed search-index objective is satisfied by the measured 222.6 MB median RSS reduction.
