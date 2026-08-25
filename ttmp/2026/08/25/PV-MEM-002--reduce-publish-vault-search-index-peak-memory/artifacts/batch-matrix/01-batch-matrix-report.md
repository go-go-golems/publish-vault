---
Title: PV-MEM-002 bounded Bleve batch matrix
Ticket: PV-MEM-002
Status: active
Topics:
    - memory
    - search
    - bleve
    - performance
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Seven-variant exploratory matrix selecting a 16-document and 1 MiB bounded Bleve batch after reducing observed heap 40%, RSS 28%, cumulative allocation 62%, and duration 31% relative to one-document updates.
LastUpdated: 2026-08-25T18:35:00-04:00
WhatFor: Selecting the production batch default for Phase 3 implementation and repeated proof.
WhenToUse: Read before reviewing batch defaults, implementation semantics, or final before/after runs.
---

# PV-MEM-002 bounded Bleve batch matrix

## Experiment

Each variant ran in a fresh process against the detached clean vault worktree at `5f9f18c`. The harness loaded the same 2,030-note snapshot, started a 100 ms measure recorder immediately before persistent search construction, built a fresh temporary index, closed it, recorded directory size, and removed it. No forced GC was used.

The current path calls `Index` once per document, which becomes one Scorch update/backend batch. Experimental variants retain mapped documents only until either the document ceiling or estimated source-field byte ceiling is reached, then commit one Bleve batch. A single document larger than the byte ceiling is committed alone because it cannot be split safely at this layer.

These are one-run exploratory measurements. They select the candidate; they do not constitute final proof. Phase 4 requires three comparable candidate runs through the complete server lifecycle.

## Results

| Variant | Document limit | Byte limit | Peak heap | Peak RSS | Search duration | Allocation delta | GC cycles | Index bytes |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| current | 1 effective | n/a | 869,712,904 | 1,068,277,760 | 89.73 s | 50,352,073,512 | 192 | 233,101,871 |
| batch 4 | 4 | 256 KiB | 758,835,360 | 956,723,200 | 70.32 s | 28,763,335,192 | 115 | 216,496,974 |
| batch 8 | 8 | 512 KiB | 635,282,728 | 813,395,968 | 67.72 s | 22,587,590,632 | 98 | 198,417,449 |
| **batch 16** | **16** | **1 MiB** | **521,084,376** | **766,619,648** | **61.78 s** | **19,273,219,336** | **83** | **199,028,353** |
| batch 32 | 32 | 2 MiB | 585,288,968 | 843,472,896 | 57.81 s | 18,207,673,408 | 77 | 198,617,238 |
| batch 64 | 64 | 4 MiB | 641,511,216 | 870,866,944 | 94.12 s | 16,751,985,056 | 71 | 199,368,997 |
| batch 128 | 128 | 8 MiB | 801,219,720 | 980,045,824 | 44.84 s | 14,351,344,296 | 55 | 198,146,912 |

Relative to current, batch 16 changed:

```text
peak heap:       -40.09%
peak RSS:        -28.24%
duration:        -31.15%
allocation:      -61.72%
index size:      -14.62%
GC cycles:       192 -> 83
```

## Interpretation

The allocation hypothesis is confirmed. Grouping documents sharply reduces repeated segment and merge work. Every batch variant reduced cumulative allocation and index size. The relationship between batch size and peak memory is not monotonic: larger batches reduce total allocations further but retain more mapped/analyzed document state at each commit and can overlap differently with Scorch merge work.

Batch 128 is fastest and has the lowest cumulative allocation, but its 801 MB heap and 980 MB RSS peaks are too close to the current path. Batch 32 is somewhat faster than batch 16 but uses 64 MB more heap and 77 MB more RSS in this run. Batch 16 provides the lowest observed heap and RSS while also improving duration and index size substantially.

Batch 64's duration is worse than current despite lower allocation. This single run likely encountered an expensive merge schedule or system variation. It demonstrates why final acceptance cannot use duration from one exploratory run.

## Correctness evidence

Unit/integration tests now establish:

- zero/zero options preserve the one-document update path;
- both bounds must be set together and document count cannot be negative;
- invalid options fail before deleting a target index path;
- committed progress is monotonic and advances at batch boundaries;
- the final partial batch is flushed;
- an oversized single document is committed alone;
- baseline and batched persistent indexes return the same result set, stored title/excerpt/tags, and scores for exact, multi-word, prefix, fuzzy, and tag query fixtures;
- tied equal-score results may appear in either order because the existing search API does not specify a secondary tie key.

Full real-vault query equivalence and deleted-note/reload behavior remain mandatory in the implementation/proof phases.

## Selected candidate

Use these production defaults for the next implementation phase:

```text
BatchDocuments = 16
BatchBytes     = 1 MiB estimated search-document fields
```

The bounds should remain internal constants initially. Exposing them as operator flags would create an unsupported tuning surface before a second workload demonstrates a need. The public `search.Options` fields remain useful for tests and controlled experiments.

Progress semantics change from one callback per document to one callback per successfully committed batch. `ProcessedDocuments` continues to mean documents successfully accepted by Bleve; it does not count documents only staged in memory. The final value remains exactly the published note count.

## Remaining proof

Before accepting the implementation:

1. Wire the selected defaults only into persistent full-snapshot construction; retain in-memory/single-document behavior unless separately justified.
2. Run complete search, persistent reload, stale deletion, close/reopen, failure cleanup, measure, and race tests.
3. Run a real-vault content-free query-equivalence harness.
4. Capture three complete candidate server runs against the same pinned commit.
5. Run an isolated finite-cgroup Docker scenario.
6. Compare medians and ranges rather than this exploratory point.
