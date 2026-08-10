---
Title: 'Diagnose and fix OOMKill: in-memory vault/search index and overlapping reload memory growth'
Ticket: PV-MEMORY-019
Status: active
Topics:
    - memory
    - oom
    - profiling
    - reload
    - bleve
    - kubernetes
    - search
    - vault
    - performance
    - git-sync
    - retro-obsidian-publish
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "The app is OOMKilled (exit 137) because a single cold load of the go-go-parc vault peaks at 1897 MiB RSS against a 1536 MiB limit; the in-memory bleve index is 884.7 MB (84.5%) of the live heap, the vault keeps two HTML copies per note, GOMEMLIMIT is unset, and RuntimeState.Reload has no mutual exclusion so reloads overlap (2 snapshots measured at 3849 MiB peak RSS)."
LastUpdated: 2026-08-09T20:21:07.114924698-04:00
WhatFor: ""
WhenToUse: ""
---

# Diagnose and fix OOMKill: in-memory vault/search index and overlapping reload memory growth

## Overview

The production `retro-obsidian-publish` pod is in CrashLoopBackOff with
`exit 137` (OOMKilled) against a 1536 MiB container memory limit, while Go
reports ~1.93 GiB of heap-system memory. This ticket diagnoses the cause with
real measurements against the production vault (`go-go-parc`), quantifies a
per-note memory budget, models the overlapping-reload failure mode, and lays
out a ranked, phased remediation plan.

**Headline finding: the app cannot complete its *first* load inside the limit.**
Loading go-go-parc (1739 Markdown files, 56.5 MiB) through `vault.New()` +
`search.New()` measures:

| Phase | Live heap | `HeapSys` | Peak RSS (`VmHWM`) |
|---|---:|---:|---:|
| after `vault.New()` | 150.9 MiB | 279.5 MiB | 299.7 MiB |
| after `search.New()` | **984.9 MiB** | **1823.4 MiB** | **1897.1 MiB** |
| two overlapping snapshots | 1967.9 MiB | 3731.4 MiB | **3848.9 MiB** |

That is **17.4x the on-disk Markdown size in live heap and 33.6x in resident
memory**, with no reload, no overlap, and no traffic. Heap profiling attributes
**884.7 MB (84.5%)** to the in-memory bleve index (`bleve.NewMemOnly`,
`pkg/search/search.go:46`) and **155.8 MB** to the vault keeping *two* copies of
every note's rendered HTML (`Note.HTML` and `Note.sourceHTML`).

Three amplifiers sit on top: no `GOMEMLIMIT` anywhere in the repo (so GC paces
to 2x live heap, exactly explaining the 1.93 GiB), `RuntimeState.Reload`
(`pkg/server/runtime.go:100`) takes no lock around the expensive build so
concurrent reloads each build a full snapshot, and `oldSnapshotCloseDelay`
(`pkg/server/runtime.go:18`) keeps the previous snapshot alive for 30 seconds
after every swap.

**Top three fixes, ranked:**

1. Set `--search-index-path` to a disk-backed `emptyDir` so bleve builds a
   persistent on-disk index. Zero application code - the flag already exists
   (`serve.go:101`) and the atomic build/rename/reopen path is already
   implemented (`runtime.go:183`). **This was measured, not estimated, and it
   resolves the incident on its own:**

   | | In-memory (today) | On-disk (this fix) | Change |
   |---|---:|---:|---:|
   | Search index live heap | 834.0 MiB | **15.7 MiB** | **-98.1%** |
   | Total live heap | 984.3 MiB | **166.2 MiB** | **-83.1%** |
   | **Peak RSS** | **1897.1 MiB** | **800.3 MiB** | **-57.8%** |
   | Fits the existing 1536 MiB limit? | No | **Yes, 48% headroom** | - |

   Costs 155 MB of disk and +49% index build time. The volume must be
   disk-backed (not `medium: Memory`) and must not be `/git`.
2. Make `Reload()` non-overlapping (mutex/singleflight) **and** a no-op when the
   resolved vault root has not changed; make `reloadHandler` answer `204`
   immediately instead of blocking for 82-135 seconds.
3. Immediate mitigation only, if the volume cannot land today: raise the limit
   to 3Gi and set `GOMEMLIMIT=2600MiB`. This stops the crash loop but fixes
   nothing; `GOMEMLIMIT` alone at 1536Mi, without fix 1, would trade the OOMKill
   for a GC death spiral.

See [the design document](./design/01-memory-usage-analysis-and-remediation-design.md)
for the full analysis, the memory budget table, diagrams, pseudocode for every
proposed fix, and reproduction instructions.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- memory
- oom
- profiling
- reload
- bleve
- kubernetes
- search
- vault
- performance
- git-sync
- retro-obsidian-publish

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
