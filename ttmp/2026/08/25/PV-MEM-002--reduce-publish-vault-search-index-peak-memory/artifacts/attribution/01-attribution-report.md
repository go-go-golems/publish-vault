---
Title: PV-MEM-002 search-index peak attribution
Ticket: PV-MEM-002
Status: active
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
Summary: Content-free aggregate attribution showing flat post-GC retained heap, 51.65 GB of search allocation churn, 612 MB of heap-arena growth, and per-document Bleve segment/merge work as the first bounded-batch experiment target.
LastUpdated: 2026-08-25T18:15:00-04:00
WhatFor: Selecting the first PV-MEM-002 optimization from aligned heap, RSS, smaps, cgroup, progress, and pprof evidence.
WhenToUse: Read before implementing or reviewing the bounded Bleve batch experiment.
---

# PV-MEM-002 search-index peak attribution

## Method and limits

A private diagnostic harness loaded the pinned clean vault commit and built one persistent Bleve index. It captured checkpoints at 0%, 25%, 50%, 75%, and 100% of 2,030 documents. At each checkpoint it emitted a measure checkpoint, forced one Go GC, wrote a raw heap profile to a mode-`0700` untracked directory, read runtime/procfs/smaps/cgroup state through `measure/pkg/collector`, and emitted a second measure checkpoint.

The run is intentionally perturbed by five forced collections. Its elapsed time and peak are not a performance baseline. Its purpose is to distinguish reachable heap from allocation churn and runtime/file-backed residency. Raw profiles can contain vault content; they remained under `/tmp`, mode `0600`, and are not present in this ticket or reMarkable. The retained pprof tables contain only aggregate function names and byte counts.

The cgroup is an unlimited shared user scope. Cgroup values are not attributable to this process and are excluded from causal conclusions. Procfs RSS, PSS, anonymous, private dirty, and private clean values describe the diagnostic process.

## Aligned checkpoint results

| Progress | Documents | Indexed bytes | HeapAlloc after GC | HeapSys | RSS | Anonymous | Private clean | Cumulative allocations |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 0% | 0 | 0 | 211,380,552 | 402,063,360 | 411,865,088 | 389,320,704 | 14,282,752 | 6,565,219,928 |
| 25% | 508 | 19,185,020 | 233,599,848 | 913,408,000 | 634,085,376 | 589,631,488 | 21,975,040 | 19,716,875,928 |
| 50% | 1,015 | 39,009,218 | 237,871,536 | 1,014,071,296 | 529,862,656 | 470,573,056 | 41,766,912 | 33,060,006,184 |
| 75% | 1,523 | 56,141,273 | 232,340,152 | 1,013,940,224 | 469,188,608 | 424,873,984 | 20,795,392 | 45,741,447,144 |
| 100% | 2,030 | 72,070,169 | 245,074,792 | 1,013,940,224 | 712,081,408 | 516,022,272 | 195,444,736 | 58,217,854,832 |

From 0% to 100%:

```text
post-GC retained HeapAlloc growth:       33,694,240 bytes
HeapSys growth:                          611,876,864 bytes
RSS growth:                              300,216,320 bytes
cumulative allocation growth:         51,652,634,904 bytes
```

The central finding is that search does **not** retain hundreds of megabytes of additional reachable Go objects after GC. Post-GC `HeapAlloc` remains between 211 MB and 245 MB. The unperturbed baseline nevertheless reaches 826 MB median `HeapAlloc` and 1.034 GB median RSS because the phase performs approximately 51.65 GB of allocations, expands Go heap arenas to roughly 1.014 GB, and faults/creates persistent-index pages.

## Retained heap

At 0%, before any search document is submitted, the heap profile contains approximately 209.4 MB. `regexp.(*Regexp).ReplaceAllStringFunc` accounts for 196.7 MB flat, or 93.9%. Cumulative call paths split this retained output approximately across:

- `Vault.loadNote -> parser.Parse`, including callout/math transformations;
- `Vault.rebuildHTML -> replaceUnresolvedNoteEmbeds`.

These allocations are the retained rendered HTML owned by the runtime snapshot. They exist before search begins and remain throughout. At 100%, the profile total is approximately 227.4 MB; the same 196.7 MB regex-produced output remains, while active Bleve structures account for a comparatively small single-digit-megabyte retained set.

This invalidates a simple corpus-retention hypothesis for Bleve construction. Bleve's persistent builder creates large allocation traffic, but the post-GC profile does not show a full-corpus Go object graph retained by search.

## Allocation traffic

At 100%, the heap profile reports approximately 58.17 GB cumulative allocations for the process. The largest flat and cumulative paths include:

| Allocation site/path | Evidence |
|---|---:|
| `bytes.growSlice` flat | 26.47 GB |
| `bytes.Buffer.grow` cumulative | 27.36 GB |
| `bleve/index/scorch.(*Scorch).Batch` cumulative | 29.80 GB |
| `scorch.planMergeAtSnapshot` cumulative | 15.61 GB |
| `zapx.mergeToWriter` cumulative | 15.34 GB |
| regexp replacement paths | multiple gigabytes |
| `parser.stripMarkdown` cumulative | 5.46 GB |
| Bleve token frequencies, roaring bitmaps, vellum builders, zap chunk coders | repeated 0.3–2.8 GB flat allocations |

Cumulative paths overlap and must not be added together. They establish where allocation traffic passes, not independent byte totals.

`bleve.Index.Index` maps one document and calls the backend's `Update`. Scorch implements each update through a one-document `Batch`, producing a new segment and merge work. With 2,030 calls, repeated segment creation and merge-to-writer activity is the largest controllable allocation path. Plain-text extraction is also expensive, but its 5.46 GB cumulative path is materially smaller than the Scorch batch/merge path.

## RSS and arena interpretation

At 100%, after forced GC:

```text
HeapAlloc:             245,074,792 bytes
HeapSys:             1,013,940,224 bytes
RSS:                   712,081,408 bytes
PSS:                   711,472,128 bytes
anonymous:             516,022,272 bytes
private dirty:         516,022,272 bytes
private clean:         195,444,736 bytes
RSS high-water mark: 1,051,488,256 bytes
```

The difference between `HeapAlloc` and anonymous RSS is consistent with runtime arenas, stacks, metadata, and pages not represented as live heap objects. The private-clean increase at 100% is consistent with persistent-index file-backed pages after construction. These pages may be reclaimable, but they still affect resident and cgroup peaks.

Forced GC reduces reachable heap but does not return all expanded arenas or file-backed residency. Adding repeated forced GC to production would increase CPU cost without removing the underlying 51.65 GB allocation path. It is not selected as the implementation.

## Representation lifetime inventory

| Representation | Creation | Last required use | Observed lifetime/conclusion |
|---|---|---|---|
| Raw Markdown `[]byte` | `Vault.ReadRaw` per note | after `parser.PlainText` | Per-document transient; not present as corpus-scale post-GC retention |
| Plain body string | `parser.PlainText` | after Bleve maps the document | Per-document transient, but contributes to 5.46 GB parser allocation traffic |
| Copied tags | `Vault.SearchDocument` | after mapping | Small per-document copy; not a dominant retained type |
| Flattened tags string | `search.Index` | after mapping | Small per-document allocation; eligible cleanup but not first target |
| `noteDoc` | `search.Index` | after Bleve mapping | Per-document transient |
| Bleve analyzed fields/token frequencies | document mapping and Scorch update | segment construction | High allocation traffic; low retained heap after forced GC |
| Zap/vellum/roaring segment state | Scorch update/merge | segment persist/merge completion | Dominant controllable cumulative allocation path |
| Rendered note HTML | vault parse/rebuild | entire snapshot lifetime | Approximately 196.7 MB retained before search; not a search implementation leak |
| Go heap arenas | runtime allocation | runtime/OS scavenging policy | Grow by 611.9 MB and remain mapped after live heap falls |
| Persistent index pages | Scorch persistence/reopen | search-index lifetime/reclaim | Visible as private-clean/file-backed residency, especially at 100% |

## Hypothesis decisions

### Rejected as primary: corpus-proportional retained Bleve object graph

Post-GC retained heap rises only 33.7 MB from 0% to 100%. The unperturbed 826 MB peak is not a permanently reachable 826 MB search graph.

### Rejected as primary: production forced GC checkpoints

Forced GC demonstrates the distinction between reachable and transient state but leaves expanded HeapSys and substantial RSS. Repeated GC would perturb throughput and treat the symptom rather than reduce allocation work.

### Deferred: optimize `parser.stripMarkdown`

Plain-text extraction accounts for approximately 5.46 GB cumulative allocations and should be reviewed later. The Scorch update/merge path is larger and directly reflects one backend update per document.

### Selected for Phase 2: bounded Bleve batches

Build documents into an explicit Bleve batch with both a document-count and estimated-byte ceiling, then commit the batch. This reduces the number of Scorch segments and repeated merge passes while keeping temporary document retention explicitly bounded.

The experiment must vary batch limits because larger batches can increase peak retained memory. The first matrix will compare:

```text
current: one Index/Update per document
batch 16 documents, 1 MiB estimated source fields
batch 64 documents, 4 MiB estimated source fields
batch 128 documents, 8 MiB estimated source fields
```

It must report peak heap/RSS, cumulative allocation, duration, index size, and search equivalence. No production default changes until the matrix identifies a clear trade-off.

## Privacy validation

The committed attribution directory contains:

- content-free checkpoint JSON;
- schema-v1 diagnostic trace and receipt;
- pprof aggregate top tables with function names and byte counts;
- this report and a machine-readable summary.

It does not contain `.pprof` files, note content, titles, slugs, paths, Markdown, or the detached vault worktree. Raw profiles remain private under `/tmp` until the aggregate analysis is accepted, then must be deleted.
