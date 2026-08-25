---
Title: PV-MEM-002 generated-fixture scaling evidence
Ticket: PV-MEM-002
Status: active
Topics:
    - memory
    - search
    - bleve
    - performance
    - regression
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Five generated persistent-index workloads validating the tightened 160-document budget and showing memory growth across document count and payload size.
LastUpdated: 2026-08-25T19:05:00-04:00
WhatFor: Calibrating public CI budgets and distinguishing bounded batch content from total retained vault growth.
WhenToUse: Read before changing generated fixture shape or memory thresholds.
---

# PV-MEM-002 generated-fixture scaling evidence

Five deterministic generated vaults were loaded through the complete candidate server path with fresh persistent indexes and 100 ms measurement. Generated Markdown existed only in temporary directories and was removed after each run. Retained artifacts contain counters and traces but no generated body text.

| Documents | Payload/document | Source payload | Peak heap | Peak RSS | Complete duration | Index bytes |
|---:|---:|---:|---:|---:|---:|---:|
| 100 | 1 KiB | 0.10 MB | 10,587,880 B | 55,193,600 B | 0.13 s | 332,727 B |
| 160 | 8 KiB | 1.31 MB | 22,936,976 B | 76,808,192 B | 1.30 s | 2,952,831 B |
| 500 | 8 KiB | 4.10 MB | 46,827,992 B | 109,006,848 B | 2.79 s | 8,881,044 B |
| 1,000 | 8 KiB | 8.19 MB | 72,926,608 B | 158,937,088 B | 9.08 s | 35,402,180 B |
| 200 | 32 KiB | 6.55 MB | 67,298,640 B | 137,486,336 B | 4.86 s | 15,631,207 B |

The CI fixture remains 160 documents × 8 KiB. Ten consecutive runs in one test process observed maxima of approximately 21.4 MB heap and 76.1 MB RSS overall, and 21.4 MB heap / 69.4 MB RSS inside `search_index`. The reviewed budgets are therefore tightened to:

```text
run peak heap <32 MiB
run peak RSS <160 MiB
search_index peak heap <32 MiB
search_index peak RSS <160 MiB
```

The 32 MiB heap threshold retains at least 49% headroom over the ten-run non-race maximum and halves the old heap threshold. The initial 96 MiB RSS proposal passed ten normal runs but failed the full race suite because race instrumentation raised observed RSS to 139,022,336 bytes. The final 160 MiB RSS threshold retains 20.7% headroom over that fresh race maximum while tightening the old 192 MiB threshold by 16.7%. The test now uses a persistent search path so it exercises the production batch policy, index publication, and close behavior.

Peak memory still grows with total vault content because the runtime snapshot deliberately retains parsed/rendered notes and the persistent index creates file-backed residency. Bounded batching limits staged search documents and segment construction work; it does not make the whole application's memory independent of corpus size. The 200 × 32 KiB case having similar memory to 1,000 × 8 KiB confirms that source payload, not document count alone, is a necessary scaling axis.
