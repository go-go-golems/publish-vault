---
Title: Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://docker-compose.yml
      Note: Reference deployment wiring --search-index-path to a disk-backed volume
    - Path: repo://pkg/server/memlimit.go
      Note: GOMEMLIMIT derived from the cgroup; note the soft-limit caveat in its doc comment
    - Path: repo://pkg/server/runtime.go
      Note: Reload serialisation and symlink no-op guard implemented in Step 8 (commit 945c2df)
    - Path: repo://publish-vault/pkg/search/search.go
      Note: Step 2/4/6 - search.New (line 46) uses bleve.NewMemOnly (884.7 MB); NewPersistent (line 64) is the already-implemented fix measured in Step 6.
    - Path: repo://publish-vault/pkg/server/runtime.go
      Note: Step 2/5 - Reload (line 100) has no lock around loadSnapshot; oldSnapshotCloseDelay (line 18) retains the old snapshot 30s; buildSearchIndex (line 183) is the sequence the harness mirrors.
    - Path: repo://publish-vault/pkg/vault/vault.go
      Note: Step 2/4 - Note (line 38) holds HTML (line 45) and sourceHTML (line 56); the doc comment at lines 50-56 explains why sourceHTML cannot simply be deleted.
    - Path: repo://publish-vault/ttmp/2026/08/09/PV-MEMORY-019--diagnose-and-fix-oomkill-in-memory-vault-search-index-and-overlapping-reload-memory-growth/scripts/vaultmem/main.go
      Note: The measurement harness written for this ticket; calls vault.New and search.New/NewPersistent and reports MemStats plus /proc/self/status VmHWM.
ExternalSources: []
Summary: 'Chronological investigation diary for the PV-MEMORY-019 OOMKill analysis: how the vault was measured, what the heap profile showed, how the overlapping-reload failure mode was reproduced, and how the on-disk search index turned out to resolve the incident outright.'
LastUpdated: 2026-08-09T00:00:00Z
WhatFor: Following the reasoning and the dead ends behind the memory analysis, and knowing what to re-verify.
WhenToUse: Read alongside the design doc when reviewing the findings or continuing the work.
---






# Diary

## Goal

This diary captures the investigation of a production memory-exhaustion problem
in `publish-vault` (binary `retro-obsidian-publish`): a Kubernetes pod in
CrashLoopBackOff with `exit 137` / OOMKilled against a 1536 MiB memory limit.
It records how the codebase was read, how the real vault was measured, which
hypotheses survived contact with the data, and — importantly — which one turned
out to be far more decisive than expected.

No application source was modified. This is an analysis ticket; the only code
written is the measurement harness under this ticket's `scripts/` directory.

---

## Step 1: Orient in the workspace and locate the real vault

The first job was to establish the geography: which directory is the Go
workspace root (where `docmgr` and `go build` must run), which is the module
under investigation, and where the real Obsidian vault lives so that the
measurements would be against production-shaped data rather than the 20-note
`vault-example/` fixture in the repo. Two candidate vault paths were given and
only one existed, so this had to be checked rather than assumed.

Getting the vault census right up front turned out to matter more than it looked.
A naive `du -sh` on the vault reports 2.6 GiB, which would have anchored every
later estimate to the wrong number. Almost all of that is image attachments that
the application never loads into memory. The number that actually drives memory
consumption is the Markdown byte total, which is 56.5 MiB — a 40× difference.

### Prompt Context

**User prompt (verbatim):**

> You are investigating a production memory-exhaustion problem in the `publish-vault` (binary name `retro-obsidian-publish`) Go application, and producing a docmgr ticket with an exhaustive, intern-readable design/analysis document, then uploading it to reMarkable.
>
> ## Production evidence (from the user, verbatim — put this in the doc)
>
> - App container: CrashLoopBackOff
> - Last termination: exit 137 — killed for exceeding memory (OOMKilled)
> - Memory limit: 1536 MiB
> - Before failure, Go reported ~1.93 GiB heap-system memory
> - Node disk: 75/150 GiB used (52%)
> - /git volume: only 473 MiB
> - Kubernetes reports DiskPressure=False
> - Repeated ~50-second reloads are causing overlapping reloads and memory growth
> - The site may intermittently respond through the SSR sidecar, but the main app is unhealthy
>
> The user's hypothesis: "I suspect there's an in-memory index?"

**Assistant interpretation:** Find the root cause of the OOMKill by reading the
actual code and measuring the actual vault, then produce a long, self-contained,
intern-readable design document in a docmgr ticket, with a phased remediation
plan, and publish it to reMarkable.

**Inferred user intent:** The pod is down and the reporter wants both an
explanation they can act on today and a durable written analysis that another
engineer (or their future self) can pick up without re-deriving anything.

### What I did

- Inspected `/home/manuel/workspaces/2026-08-09/publish-vault-mathjax`: confirmed
  `go.work` uses `./glazed` and `./publish-vault`, and that `.ttmp.yaml` roots
  docmgr at `publish-vault/ttmp`.
- Checked both candidate vault paths. `/home/manuel/code/wesen/go-go-parc` does
  not exist; `/home/manuel/code/wesen/go-go-golems/go-go-parc` does.
- Ran a vault census with `du`, `find`, and `du -cb --files0-from=-`.

### Why

Every memory figure in the final document is expressed as a multiplier against
the on-disk Markdown size. That denominator had to be correct and had to be
measured, not guessed.

### What worked

The census produced clean, unambiguous numbers:

| Metric | Value |
|---|---:|
| Total size incl. `.git` | 2.6 GiB |
| Total size excl. `.git` | 2.3 GiB |
| Markdown files | 1739 |
| **Markdown bytes** | **59,267,905 (56.5 MiB)** |
| Non-Markdown (asset) files | 1970 |
| Asset bytes | 2,271,422,933 (2166.2 MiB) |
| Largest note | 513,615 B (a ChatGPT transcript) |

### What didn't work

Nothing failed, but the first candidate vault path was wrong
(`ls: cannot access '/home/manuel/code/wesen/go-go-parc': No such file or
directory`), which is exactly why both were specified.

### What I learned

**93 % of this vault by bytes is attachments the app never reads.** The
application indexes asset *paths* only (`vault.indexAssetInto`,
`pkg/vault/vault.go:349`) and streams asset bytes from disk per request
(`assetHandler`, `pkg/server/server.go:231`). Anyone sizing this workload from
`du -sh` alone will be wrong by more than an order of magnitude.

### What was tricky to build

Totalling Markdown bytes while excluding `.git` needed
`find ... -name '.git' -prune -o -name '*.md' -print0 | du -cb --files0-from=-`
rather than a naive `du`, because `.git` contains packed objects that would
otherwise be counted and because `du` reports block-allocated size rather than
byte size unless given `-b`.

### What warrants a second pair of eyes

Whether 1739 is the right denominator: the loader excludes files via
`.vault-ignore` and `publish: false`, so the vault actually loaded 1712 notes.
The design doc uses 1739 for the on-disk figure and 1712 for the per-note
figures and says so explicitly; a reviewer should confirm that distinction reads
clearly.

### What should be done in the future

The census script (`scripts/01-measure-vault-on-disk.sh`) is generic. It would be
worth running against other vaults before deploying this app for them, since
memory scales linearly with Markdown bytes.

### Code review instructions

- Start at `scripts/01-measure-vault-on-disk.sh`.
- Validate with:
  `./scripts/01-measure-vault-on-disk.sh /home/manuel/code/wesen/go-go-golems/go-go-parc`

### Technical details

```bash
find "$VAULT" -name '.git' -prune -o -name '*.md' -print0 \
  | du -cb --files0-from=- 2>/dev/null | tail -1
# => 59966268 total   (whole tree, incl. some dotfile dirs)
# harness walk, matching LoadAll's dot-dir pruning => 59267905
```

---

## Step 2: Read the code and form hypotheses

With the data shape known, I read the four subsystems that could plausibly hold
memory: `pkg/vault` (the note model), `pkg/search` (the index), `pkg/server`
(snapshots and the reload path), and `pkg/watcher`. The reporter's hypothesis
("an in-memory index") pointed at `pkg/search`, but the phrase "overlapping
reloads" pointed at `pkg/server/runtime.go`, and those are different bugs with
different fixes, so both needed reading before measuring.

Reading the code produced four concrete, falsifiable hypotheses and — just as
usefully — eliminated several plausible-sounding suspects that turned out to be
innocent.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Before measuring anything, understand what the
program actually retains, so the measurements can be attributed to specific
struct fields and call sites rather than to a vague "the app is big".

**Inferred user intent:** An explanation that names lines of code, not just a
number.

### What I did

- Read `pkg/vault/vault.go` (907 lines) in full, `pkg/search/search.go`,
  `pkg/server/runtime.go`, `pkg/server/server.go`, `pkg/watcher/watcher.go`,
  `pkg/api/api.go`, and the relevant parts of `internal/parser/parser.go`.
- Read `Dockerfile`, `docker-compose.yml`, `deploy/gitops-targets.json`, and the
  reference Kubernetes manifest preserved in the RETRO-DEPLOY-003 ticket.
- Ran targeted greps for runtime tuning and concurrency primitives.

### Why

Attribution. A heap profile tells you *which function* allocated; only reading
the code tells you *why the program still holds it*.

### What worked

Four hypotheses, all confirmed by reading:

1. **`search.New` uses `bleve.NewMemOnly`** (`pkg/search/search.go:47`) — the
   index is entirely on the Go heap. A persistent alternative already exists
   (`NewPersistent`, `:64`) and is fully wired through
   `server.buildSearchIndex` (`pkg/server/runtime.go:183-227`) behind the
   `--search-index-path` flag (`serve.go:101`), which defaults to empty.
2. **Every `Note` holds two full HTML strings.** `Note.HTML` (`vault.go:45`) and
   unexported `Note.sourceHTML` (`vault.go:56`). The doc comment on `sourceHTML`
   explains why: `rebuildHTML` (`:434`) runs a *destructive* final pass
   (`replaceUnresolvedNoteEmbeds`, `:487`) that replaces embed placeholders with
   a "⚠ Note not published" marker, so rebuilding from the previous output would
   bake the first outcome in permanently.
3. **`RuntimeState.Reload` takes no lock around the expensive work**
   (`runtime.go:100`). `s.mu` guards only the pointer swap at `:110-113`. Two
   concurrent `POST /api/admin/reload` requests therefore each run a full
   `loadSnapshot`.
4. **The old snapshot is deliberately retained for 30 seconds**
   (`oldSnapshotCloseDelay`, `runtime.go:18`; `closeSnapshotAfter`, `:229`). The
   goroutine closure captures `snap`, which references `snap.Vault`, so the whole
   old vault stays reachable for the full delay, not just the search index.

Greps that came back empty were as informative as the ones that hit:

```
$ grep -rn "GOMEMLIMIT\|GOGC\|SetMemoryLimit\|SetGCPercent\|debug.FreeOSMemory" \
    --include='*.go' --include='*.yaml' --include='*.yml' --include='Dockerfile*' . | grep -v ttmp/
(no output)
$ grep -rn "pprof" --include='*.go' .
(no output)
$ grep -rn "singleflight" --include='*.go' .
(no output)
```

No `GOMEMLIMIT`, no pprof endpoint, no singleflight — anywhere in the repository.

### What didn't work

Several intuitive suspects turned out to be innocent, and ruling them out took
real reading:

- **Raw Markdown is not retained.** `loadNote` (`vault.go:201`) reads `src`,
  parses, and drops it. `SearchDocument` (`:749`) re-reads from disk via
  `ReadRaw` (`:861`). I initially expected a third copy here; there isn't one.
- **Plain-text bodies are not retained.** `ForEachSearchDocument` (`:765`)
  streams one document at a time and its docstring explicitly says it avoids
  materialising a full-vault slice. That code is already correct.
- **`/api/notes` does not serialise HTML.** `NoteList` (`api.go:103`) copies only
  lightweight fields.
- **The embedded SPA is not on the heap.** `go:embed` data lives in the binary's
  read-only data section.
- I could not find the live Kubernetes manifest at all — `deploy/` contains only
  `gitops-targets.json`, which points at an external repo
  (`wesen/2026-03-27--hetzner-k3s`). The reference manifest in the RETRO-DEPLOY-003
  ticket contains **no `resources:` block whatsoever**, so the 1536Mi limit was
  added outside this repository and I cannot verify it or the probe settings.

### What I learned

The most surprising finding of this step: **the fix for the biggest hypothesis
was already fully implemented and simply not enabled.** `--search-index-path`
exists, `search.NewPersistent` exists, `buildSearchIndex` already does a proper
atomic build → close → rename → reopen, and `closeSnapshotAfter` already cleans
up old index directories. Nobody had turned it on.

Second: the codebase already has good memory instrumentation.
`logMemoryPhase` (`runtime.go:287`) emits `HeapAlloc`/`HeapSys`/`HeapInuse`/
`NextGC`/`NumGC` at eight distinct reload phases, and `/api/healthz`
(`server.go:197`) embeds the same struct. Those log lines from the crashing pod
are the highest-value production artefact available and should be collected
before anything else.

### What was tricky to build

Understanding *why* `sourceHTML` exists. On first read it looks like obvious
redundant state and an easy 68 MiB win. The doc comment at `vault.go:50-56`
documents a real bug that this field already fixed: an embed whose target was
hidden by `publish: false` would keep its broken-embed marker forever after the
target became publishable, because the placeholder it was rendered from is
destroyed by the rewrite. Any patch that just deletes the field reintroduces
that bug. This is why the design doc proposes three graded options (re-parse,
make the last pass non-destructive, or render lazily) rather than "delete it".

### What warrants a second pair of eyes

- The claim that `closeSnapshotAfter`'s closure keeps the old **Vault** (not just
  the search index) alive for 30 s. It follows from the closure capturing `snap`,
  but it deserves confirmation from someone else reading `runtime.go:229-248`.
- The reload-overlap reasoning assumes `net/http` serves concurrent POSTs to the
  same handler concurrently, which it does, but it is worth stating out loud
  because the fix depends on it.

### What should be done in the future

Add `net/http/pprof` behind a `--pprof-addr` flag. The absence of a live profiling
endpoint meant this investigation had to reproduce the problem locally rather
than profile the actual failing pod.

### Code review instructions

- Start at `pkg/server/runtime.go:100-117` (`Reload`) and `:229-248`
  (`closeSnapshotAfter`) — the concurrency and lifetime bugs.
- Then `pkg/search/search.go:46-59` (`New` → `bleve.NewMemOnly`) versus `:64-84`
  (`NewPersistent`).
- Then `pkg/vault/vault.go:38-57` (`Note`) and `:434-467` (`rebuildHTML`).
- Validate the greps yourself; empty output is part of the evidence.

### Technical details

The reload path in brief, with citations:

```
POST /api/admin/reload            -> reloadHandler          server.go:211
  -> stopWatcherBeforeReload()     [sync.Once]              server.go:84
  -> RuntimeState.Reload()                                  runtime.go:100
       -> loadSnapshot()           *** NO LOCK HELD ***     runtime.go:119
            -> filepath.EvalSymlinks(root)                  runtime.go:128
               (result never compared against the active snapshot)
            -> vault.New(root)                              vault.go:122
            -> buildSearchIndex(v, searchIndexPath)         runtime.go:183
                 searchIndexPath == "" -> search.New -> bleve.NewMemOnly
       -> s.mu.Lock(); swap; s.mu.Unlock()                  runtime.go:110
       -> closeSnapshotAfter(old, 30s)                      runtime.go:114
```

---

## Step 3: Build a measurement harness and measure one snapshot

Reading code gives you hypotheses; it does not give you numbers, and this ticket
needed numbers to rank the fixes. I wrote a standalone Go program under the
ticket's `scripts/` directory that calls the exact exported APIs the server calls
— `vault.New()` and `search.New()` — and reports `runtime.MemStats` plus
`/proc/self/status` `VmHWM` after each phase, forcing a GC first so `HeapAlloc`
means "live" rather than "live plus uncollected garbage".

The result was immediate and decisive, and it reframed the whole investigation:
a single cold load, with no reload and no traffic, peaks at **1897.1 MiB RSS**
against a 1536 MiB limit. The application cannot complete its *first* load inside
the limit. Everything about overlapping reloads is an amplifier on top of a
process that was already too big by 24 %.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Write a measurement script under the ticket's
`scripts/` dir that loads the real vault and reports memory, record actual
numbers, and state failures verbatim rather than fabricating.

**Inferred user intent:** Evidence strong enough to justify a production change,
not an estimate.

### What I did

- Wrote `scripts/vaultmem/main.go`, which:
  - walks the vault the way `LoadAll` does (pruning dot-directories) to total
    Markdown and asset bytes;
  - records a phase after baseline, after `vault.New`, and after the search build,
    calling `runtime.GC()` before every `runtime.ReadMemStats`;
  - reads `VmHWM` and `VmRSS` from `/proc/self/status`, because those — not
    `MemStats` — are what a cgroup limit is enforced against;
  - sums the exported per-note fields (`HTML`, `Excerpt`, `Title`, `Path`, `Slug`,
    link/backlink/tag counts) for a field-level budget;
  - can build a second and third snapshot while holding the first;
  - can write a heap profile via `runtime/pprof.WriteHeapProfile`;
  - has `-persist`, `-no-search`, and `-memlimit` flags for testing the fixes.
- Built it from the workspace root so `go.work` resolves the module, and ran it.

### Why

Calling the real exported APIs (rather than reimplementing the load) means the
measurement cannot drift from the server's behaviour. Forcing a GC before reading
`MemStats` is what makes `HeapAlloc` a meaningful "live heap" figure. Reading
`VmHWM` is what makes the result comparable to the container limit.

### What worked

Build and run both succeeded first time.

```
GOMAXPROCS=8 GOGC="" GOMEMLIMIT=""
on-disk: mdFiles=1739 mdBytes=59267905 (56.5 MiB)  assetFiles=1970 assetBytes=2271422933 (2166.2 MiB)

00-baseline        elapsed=0s        heapAlloc=  0.6 MiB  heapInuse=   1.7 MiB  heapSys=   7.4 MiB  sys=  13.6 MiB  nextGC=   4.0 MiB  numGC=2    maxRSS=  19.9 MiB
01-vault1-loaded   elapsed=17.846s   heapAlloc=150.9 MiB  heapInuse= 174.4 MiB  heapSys= 279.5 MiB  sys= 289.9 MiB  nextGC= 302.1 MiB  numGC=215  maxRSS= 299.7 MiB
02-search1-built   elapsed=1m4.52s   heapAlloc=984.9 MiB  heapInuse=1566.0 MiB  heapSys=1823.4 MiB  sys=1890.1 MiB  nextGC=1970.1 MiB  numGC=274  maxRSS=1897.1 MiB

notes = 1712 ; HTML bytes = 71148615 (67.9 MiB) ; HTML/markdown = 1.20x
vault only  = 150.3 MiB ; search only = 834.0 MiB ; one snapshot = 984.3 MiB
multiplier vs on-disk markdown = 17.41x ; peak RSS = 1897.1 MiB
```

The reproduction is faithful to the incident report: the reporter said Go showed
**~1.93 GiB heap-system**; I measured `HeapSys` at **1823.4 MiB (1.78 GiB)** and
`Sys` at **1890.1 MiB (1.85 GiB)**, with a second run reaching **1863.4 MiB**.

### What didn't work

- **Timings are not reproducible run to run.** `search.New` took **64.5 s** in
  the first run and **110.9 s** and **96.4 s** in later runs on the same machine.
  Memory figures were stable to within 1 MiB across all runs; only wall-clock
  varied, because other work was competing for CPU. The design doc therefore
  quotes timing as a range (82–135 s total) and never as a precise figure.
- `Note.sourceHTML` is unexported, so the harness cannot sum it directly. I had
  to infer its size (~78 MB) from the heap profile's `parser.Parse` cumulative
  figure. This is called out explicitly in the design doc's open questions rather
  than presented as a measurement.

### What I learned

**The incident does not require overlapping reloads to explain it.** I went in
expecting the reload path to be the culprit and the base footprint to be
comfortable; the opposite is true. Base footprint is 1897 MiB against a 1536 MiB
limit — already 24 % over — and the reload behaviour multiplies an
already-fatal number.

Second: `VmHWM` and `HeapAlloc` tell very different stories. Live heap is
984.9 MiB; peak RSS is 1897.1 MiB. If you only look at `HeapAlloc` you conclude
the app fits in 1536 MiB, and you are wrong by a factor of two.

### What was tricky to build

Two things.

**Making `HeapAlloc` mean "live".** Without a forced `runtime.GC()` immediately
before `ReadMemStats`, `HeapAlloc` includes garbage the collector has not swept,
and the search-build phase generates enormous transient garbage (every note's raw
bytes and plaintext). Early numbers were noisy until `record()` was changed to
call `runtime.GC()` first.

**Making the harness live inside the module.** The scripts directory sits at
`publish-vault/ttmp/2026/08/09/PV-MEMORY-019--.../scripts/vaultmem`, which is
inside the `github.com/go-go-golems/publish-vault` module, so it is buildable
as a normal package path from the workspace root. Building it as a subdirectory
of `ttmp` with a long, hyphen-heavy directory name worked without complaint, but
it needs quoting in the shell and the glob `PV-MEMORY-019--*` is the practical
way to reference it.

**Keeping snapshots alive to the end.** The overlap measurement is meaningless if
the GC collects snapshot 1 while snapshot 2 is being built. `runtime.KeepAlive`
calls at the end of `main` guarantee every snapshot stays reachable through the
final measurement.

### What warrants a second pair of eyes

- The forced-GC-before-`ReadMemStats` approach: it is the standard technique, but
  it means the reported `HeapAlloc` is a *post-collection* figure and therefore
  optimistic relative to what the process holds mid-cycle. That is deliberate
  (it isolates live data) and the RSS figures capture the pessimistic side, but a
  reviewer should be aware of both.
- `diskStats` in the harness prunes dot-directories the way `LoadAll` does, so its
  Markdown total (59,267,905) differs slightly from a plain `find` over the tree
  (59,966,268). Both numbers appear in the ticket; make sure the distinction is
  clear.

### What should be done in the future

Turn this harness into a `go test -bench`-style memory regression test that fails
CI if per-note resident bytes exceed a threshold, so a future change to `Note` or
the search mapping cannot silently double the footprint.

### Code review instructions

- Start at `scripts/vaultmem/main.go`, functions `record()`, `maxRSSBytes()`, and
  `main()`.
- Validate:
  ```bash
  cd /home/manuel/workspaces/2026-08-09/publish-vault-mathjax
  go build -o /tmp/vaultmem './publish-vault/ttmp/2026/08/09/PV-MEMORY-019--*/scripts/vaultmem'
  /tmp/vaultmem -vault /home/manuel/code/wesen/go-go-golems/go-go-parc
  ```
  Expect ~85 s and a ~1.9 GiB peak; ensure the machine has the room.
- Raw outputs are preserved verbatim under `scripts/results/`.

### Technical details

`VmHWM` is read from `/proc/self/status` because `runtime.MemStats` has no
equivalent — the Go runtime does not know how many of its mapped pages are
currently resident, and the kernel OOM killer does not care about `HeapAlloc`.

---

## Step 4: Heap profile — attribute the 985 MiB to specific call sites

A single aggregate number ("985 MiB") is not actionable. To rank the fixes I
needed to know how much each subsystem contributed, so the harness wrote a heap
profile at peak and I analysed it with `go tool pprof -sample_index=inuse_space`.

The profile settled the ranking immediately and also delivered the direct,
quantified confirmation of the two-copies-of-HTML hypothesis from Step 2.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Optionally write a pprof heap profile and analyse
it with `go tool pprof -top`.

**Inferred user intent:** Know *which code* to change, not just that the process
is big.

### What I did

```bash
go tool pprof -top -sample_index=inuse_space -nodecount=25 /tmp/vaultmem /tmp/snap1.heap
```

### Why

`-sample_index=inuse_space` reports what is *still held* at profile time.
The default, `alloc_space`, reports cumulative allocation since process start and
for this workload is dominated by transient parser garbage — it would have sent
me chasing `parser.PlainText` instead of bleve.

### What worked

```
Showing nodes accounting for 1034.87MB, 98.84% of 1046.97MB total
      flat  flat%   sum%        cum   cum%
  544.16MB 51.97% 51.97%   544.16MB 51.97%  upsidedown_store_api.(*EmulatedBatch).Set
  153.51MB 14.66% 66.64%   340.52MB 32.52%  .../store/gtreap.(*Writer).ExecuteBatch
  150.69MB 14.39% 81.03%   151.20MB 14.44%  regexp.(*Regexp).ReplaceAllStringFunc
   96.50MB  9.22% 90.25%    96.50MB  9.22%  gtreap.(*Treap).split
      83MB  7.93% 98.18%   179.51MB 17.15%  gtreap.(*Treap).union
         0     0% 98.84%   884.68MB 84.50%  bleve/v2.(*indexImpl).Index
         0     0% 98.84%   884.68MB 84.50%  publish-vault/pkg/search.New
         0     0% 98.84%   159.28MB 15.21%  publish-vault/pkg/vault.(*Vault).LoadAll
         0     0% 98.84%    78.07MB  7.46%  publish-vault/pkg/vault.(*Vault).rebuildHTML
         0     0% 98.84%    77.69MB  7.42%  publish-vault/internal/parser.Parse
```

- `search.New` → **884.68 MB cumulative = 84.5 % of live heap**. Hypothesis 1 confirmed.
- `LoadAll` → 159.28 MB, split almost exactly evenly between `parser.Parse`
  (77.69 MB, which becomes `sourceHTML`) and `rebuildHTML` (78.07 MB, the
  rewritten `HTML`). **Two distinct ~78 MB allocations.** Hypothesis 2 confirmed
  with a number.

### What didn't work

Nothing failed. One result needed thought rather than acceptance:
`EmulatedBatch.Set` showing 544 MB as *in-use* long after indexing finished looks
wrong — a "batch" ought to be transient. It is not a leak: gtreap is a persistent
(copy-on-write) treap that stores references to exactly the key/value byte slices
allocated in `EmulatedBatch.Set`, so those slices are the index data itself and
are legitimately reachable.

### What I learned

The 15.6× blowup from Markdown to index is a property of the *storage engine*,
not of the mapping. The mapping is already frugal — `bodyField.Store = false`
(`search.go:323`) means note bodies are analysed but not stored. The cost is one
Go allocation per index row wrapped in a treap node with its own pointers and
priority. A vault of AI transcripts and research notes has an enormous
vocabulary, so postings dominate and per-row overhead dominates postings.

This is precisely why an on-disk index should help disproportionately: the same
logical rows become packed segment files instead of millions of individually
allocated, individually GC-traced Go objects.

### What was tricky to build

Interpreting `regexp.(*Regexp).ReplaceAllStringFunc` at 150.69 MB *flat*. It is
not one call site; it is the physical allocation point for both HTML copies —
`renderCallouts` (`parser.go:359`) inside `Parse`, and
`replaceUnresolvedNoteEmbeds` (`vault.go:487`) inside `rebuildHTML`. Reading it
as a single regex problem would have been wrong.

### What warrants a second pair of eyes

The inference that `Note.sourceHTML` ≈ 78 MB. It rests on `parser.Parse`'s
cumulative in-use figure being retained *only* via `sourceHTML`, which follows
from `loadNote` (`vault.go:240-252`) assigning `parsed.HTML` to both `HTML` and
`sourceHTML` and `rebuildHTML` then replacing `HTML`. A definitive number needs a
temporary exported accessor or an in-package test.

### What should be done in the future

Capture the same profile from the real pod once `net/http/pprof` is available,
to confirm the production allocation shape matches this dev-box reproduction.

### Code review instructions

- Full output: `scripts/results/04-heap-profile-top-inuse-space.txt`.
- Re-derive:
  `go tool pprof -top -sample_index=inuse_space -nodecount=25 /tmp/vaultmem /tmp/snap1.heap`
- Explore interactively: `go tool pprof -http=:8081 /tmp/vaultmem /tmp/snap1.heap`

### Technical details

Always pass `-sample_index=inuse_space` for a "what is holding memory" question.
`alloc_space` answers a different question ("what allocated the most over time")
and for this workload gives a misleading answer.

---

## Step 5: Reproduce the overlapping-reload failure mode

The reporter's description — "repeated ~50-second reloads are causing overlapping
reloads and memory growth" — described a failure mode I had found in the code
(Step 2, hypotheses 3 and 4) but had not yet quantified. I re-ran the harness
with `-second`, which builds a complete second vault and search index while the
first is still referenced, exactly as `RuntimeState.Reload` does when a second
webhook arrives mid-build or during the 30 s `oldSnapshotCloseDelay` window.

The scaling turned out to be exactly linear, and the run surfaced a second-order
effect I had not anticipated: overlapping reloads make *each other slower*, which
makes them overlap more.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Prove the overlapping-reload hypothesis with a
measurement, including a diagram of what happens when reload N+1 starts before
reload N finishes.

**Inferred user intent:** Confirm or refute the reporter's own read of the
symptom.

### What I did

```bash
/tmp/vaultmem -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
              -second -memprofile /tmp/snap2.heap
```

Checked available memory first (`free -m` showed 52 GiB available) since the run
was expected to reach ~4 GiB.

### Why

To turn "overlapping reloads cause memory growth" from a plausible story into a
number that can be compared against the 1536 MiB limit.

### What worked

```
02-search1-built   elapsed=1m50.937s heapAlloc= 984.4 MiB  heapInuse=1591.1 MiB  heapSys=1863.4 MiB  maxRSS=1929.7 MiB
03-vault2-loaded   elapsed=1m10.665s heapAlloc=1134.7 MiB  heapInuse=1798.5 MiB  heapSys=2715.5 MiB  maxRSS=2795.4 MiB
04-search2-built   elapsed=1m46.542s heapAlloc=1967.9 MiB  heapInuse=3195.9 MiB  heapSys=3731.4 MiB  maxRSS=3848.9 MiB
```

| Scenario | Live heap | `HeapSys` | Peak RSS | vs. 1536 MiB |
|---|---:|---:|---:|---|
| 1 snapshot | 984.9 MiB | 1823.4 MiB | 1897.1 MiB | **1.24× over** |
| 2 snapshots | 1967.9 MiB | 3731.4 MiB | **3848.9 MiB** | **2.51× over** |

Exactly linear: a second snapshot costs another 983.8 MiB of live heap. Nothing
is shared between snapshots — not notes, not index rows.

### What didn't work

The run took roughly six minutes of wall clock, much longer than the ~3 minutes
two sequential builds would suggest. That is not a harness problem; it is the
finding.

### What I learned

**Overlapping reloads slow each other down, which makes them overlap more.** The
second `vault.New` took **70.7 s** against the first one's **23.5 s** — 3× slower
for identical work — and the second search build took 106.5 s. The reason is
straightforward once seen: the GC must now scan a 1–2 GiB live heap on every
cycle instead of a 150 MiB one, so every allocation is more expensive.

This is a positive feedback loop, and it explains the reporter's wording
precisely. They described "memory growth", not a stable 2× plateau. A system that
merely double-buffered would plateau. A system where each concurrent build slows
the others, so more builds accumulate before any finishes, grows without bound
until the kernel intervenes.

### What was tricky to build

Ensuring snapshot 1 could not be collected while snapshot 2 was building. Go's GC
is free to reclaim anything unreachable, and a naive harness that assigns
snapshot 1 to a variable it never reads again may find that variable optimised
into unreachability. `runtime.KeepAlive(v1)` / `KeepAlive(s1)` at the very end of
`main` pins them through the final measurement.

### What warrants a second pair of eyes

Whether git-sync actually issues *concurrent* webhook calls, or whether the
overlap in production comes from webhook **retries** against a handler that
blocks for 82–135 s (`reloadHandler`, `server.go:211`, does not return until the
build completes). The design doc treats the retry-storm route as a hypothesis and
flags it as the highest-value unknown, because I could not read the live
manifest's `--webhook-timeout` / `--webhook-backoff`. The fix — serialise the
reload and answer immediately — is the same either way, which is why this
uncertainty does not block the recommendation.

### What should be done in the future

Add a regression test that fires N concurrent `Reload()` calls and asserts
`loadSnapshot` ran exactly once. This is Phase 4b in the plan.

### Code review instructions

- Full output: `scripts/results/02-two-overlapping-snapshots-memonly.txt`.
- Reproduce: `/tmp/vaultmem -vault <vault> -second` (needs ~4 GiB free).
- Three snapshots: add `-third` (needs ~6 GiB free).

### Technical details

The end-to-end reproduction against the real server, without the harness:

```bash
go run ./publish-vault/cmd/retro-obsidian-publish serve \
  --vault <vault> --watch=false --reload-allow-loopback --port 8080
# then, in another shell:
for i in 1 2 3 4 5; do curl -sf -XPOST localhost:8080/api/admin/reload & done; wait
```

On unfixed code this produces five interleaved `memory phase=load_start` lines.
On fixed code it should produce one.

---

## Step 6: Measure the on-disk search index — the result that changed the ranking

Having established that bleve was 84.5 % of the problem, the obvious question was
how much the already-implemented `--search-index-path` option would save. I added
a `-persist` flag to the harness that reproduces `server.buildSearchIndex`'s
exact sequence — `NewPersistent` → `Close` → `OpenPersistent` — and ran it.

I expected a substantial improvement. What I got was a complete resolution of the
incident: peak RSS **1897.1 → 800.3 MiB**, which fits inside the *existing*
1536 MiB limit with 48 % headroom, from a flag and a volume, with no application
code change at all.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Evaluate the on-disk / memory-mapped search index
remediation option with a quantified saving rather than an estimate.

**Inferred user intent:** Know which fix to ship first and what it will actually
buy.

### What I did

- Extended `scripts/vaultmem/main.go` with `-persist <dir>` and a `buildIndex`
  helper that mirrors `runtime.go:188-227`: build into a directory, `Close()` the
  build handle, then `OpenPersistent` the finished index — because it is the
  *reopened* handle, not the build handle, that determines steady-state memory.
- Ran it and measured the resulting index directory with `du -sh`.

### Why

The remediation options had to be ranked by measured saving. Guessing at this one
would have been the weakest link in the whole document, especially since it is
the recommendation that touches production first.

### What worked

```
01-vault1-loaded   elapsed=26.106s   heapAlloc=151.1 MiB  heapSys=279.5 MiB  numGC=213  maxRSS=301.2 MiB
02-search1-built   elapsed=1m36.447s heapAlloc=166.8 MiB  heapSys=943.0 MiB  numGC=404  maxRSS=800.3 MiB

vault only  = 150.4 MiB ; search only = 15.7 MiB ; one snapshot = 166.2 MiB
multiplier vs on-disk markdown = 2.94x
peak RSS = 800.3 MiB ; current RSS = 510.4 MiB
index dir on disk = 155M
```

| Metric | In-memory (today) | On-disk | Change |
|---|---:|---:|---:|
| Search index live heap | 834.0 MiB | **15.7 MiB** | **−98.1 %** |
| One snapshot live heap | 984.3 MiB | **166.2 MiB** | **−83.1 %** |
| Multiplier vs Markdown | 17.41× | **2.94×** | −83 % |
| **Peak RSS** | **1897.1 MiB** | **800.3 MiB** | **−57.8 %** |
| Fits 1536 MiB? | **No** | **Yes, 48 % headroom** | — |
| Build time | 64.5 s | 96.4 s | +49 % |
| Disk | 0 | 155 MB | +155 MB |

### What didn't work

Nothing failed. One number looked contradictory and needed explaining rather than
reporting: **`HeapSys` (943.0 MiB) exceeds peak RSS (800.3 MiB)**. That is not an
error. `HeapSys` counts arena the runtime obtained from the OS *including pages
already handed back* via `MADV_FREE`/`MADV_DONTNEED`, which are no longer
resident. The on-disk build allocates heavily and transiently — `numGC` rose to
**404** versus **274** for the in-memory build — the arena grows, and the
scavenger then returns most of it. Steady-state `VmRSS` settled at 510.4 MiB.

### What I learned

**The fix was already in the codebase, fully implemented, tested, and switched
off.** `--search-index-path` (`serve.go:101`), `search.NewPersistent`
(`search.go:64`), the atomic build/rename/reopen in `buildSearchIndex`
(`runtime.go:183-227`), and old-index-directory cleanup in `closeSnapshotAfter`
(`runtime.go:242-246`) all exist and work. The default is empty, so production
has been running the in-memory path the whole time.

Also: the residual 15.7 MiB is bleve's working state for an open on-disk index.
The bulk now lives in files the kernel pages in on demand. Those pages still
count toward RSS while hot, but they are **file-backed and reclaimable** — under
pressure the kernel evicts them — whereas Go heap pages can only be
garbage-collected, never evicted. That distinction is the whole point of the fix.

### What was tricky to build

Getting the harness to measure the right thing. My first instinct was to measure
right after `NewPersistent` returns, but that handle is the *build* handle, still
holding write buffers. Production closes it and reopens the finished index
(`runtime.go:208-225`), and only the reopened handle reflects steady state. The
`buildIndex` helper replicates that sequence exactly; measuring the build handle
would have overstated the residual footprint substantially.

### What warrants a second pair of eyes

**Search-result equivalence.** `bleve.NewMemOnly` and `bleve.New(path, ...)`
select *different index implementations* — the profile shows the mem-only path
uses `upsidedown`, while the on-disk path uses bleve v2's default persistent
type. Scoring should be equivalent, but before shipping this to production
somebody should run `pkg/search/search_test.go` against both and spot-check
`/api/search?q=` output on the real vault. This is listed as open question 9.

Also worth a look: whether 155 MB of index is acceptable given the reported
`/git` volume is only 473 MiB. It must **not** go on `/git` (that is git-sync's),
and it must **not** be a `medium: Memory` emptyDir (tmpfs is charged to the same
cgroup, which would move the bytes without saving anything).

### What should be done in the future

Consider making `--search-index-path` default to a temp directory rather than
empty, so the memory-hungry path is opt-in rather than the default. That is an
application change and belongs in a follow-up.

### Code review instructions

- Start at `scripts/vaultmem/main.go`, function `buildIndex`, and compare it
  line-for-line against `pkg/server/runtime.go:188-227`.
- Full output: `scripts/results/03-single-snapshot-persistent-index.txt`.
- Reproduce: `/tmp/vaultmem -vault <vault> -persist /tmp/pv-index`

### Technical details

Deployment shape required by this fix:

```yaml
volumes:
  - name: search-index
    emptyDir:
      sizeLimit: 4Gi        # disk-backed; do NOT set medium: Memory
volumeMounts:
  - name: search-index
    mountPath: /var/lib/publish-vault
args:
  - --search-index-path
  - /var/lib/publish-vault/search
```

---

## Step 7: Write up the ticket, the design document, and the plan

With four independent measurements in hand — single snapshot, heap profile,
overlapping snapshots, and on-disk index — the remaining work was to turn them
into something a new engineer could act on without repeating any of it. The
design document is written for somebody who has never opened this repository:
every claim about the code carries a `file.go:line` citation, every number
carries the command that produced it, and everything I could not verify is listed
as an open question rather than smoothed over.

The measured on-disk result forced a late restructuring of the recommendations.
The plan had originally been the conventional one — raise the limit and set
`GOMEMLIMIT` as the immediate fix, then work on the index. Once the on-disk
measurement came back at 800.3 MiB peak RSS, the honest ranking changed: the
"real fix" fits inside the *current* limit, so raising the limit demoted from
"immediate fix" to "optional safety margin".

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Produce the docmgr ticket with a long
intern-readable design doc, a diary, ordered remediation tasks, related files, a
changelog, and measurement scripts; then upload to reMarkable and commit.

**Inferred user intent:** A durable, self-contained artefact that outlives the
session and can be handed to somebody else.

### What I did

- Added six vocabulary topics (`memory`, `oom`, `profiling`, `reload`, `bleve`,
  `kubernetes`) and created ticket **PV-MEMORY-019**.
- Wrote `design/01-memory-usage-analysis-and-remediation-design.md` with: the
  verbatim production evidence; a plain-language explanation of exit 137, cgroup
  limits, and `HeapAlloc` vs `HeapSys` vs `Sys` vs RSS; a mermaid system map and
  an ASCII pipeline annotated with measured sizes; a subsystem walkthrough; a
  "what this code does NOT do" table; all four measurements; a 20-row per-note
  memory budget table; reload timeline diagrams for the serial and overlapping
  cases; pseudocode for the current path and for six proposed fixes; an API
  reference of ~40 application symbols with citations plus a Go runtime/stdlib
  reference; six remediation options each with mechanism, quantified saving,
  cost, risk, and recommendation; a seven-phase implementation plan; local and
  in-cluster reproduction commands; ten open questions; and a file reference table.
- Added nine ordered tasks matching the phases; related the seven most important
  files with explanatory notes; filled in the ticket `index.md`; preserved all raw
  measurement output under `scripts/results/`.

### Why

The brief asked for something an intern could read cold. That rules out a
bullet-only document and rules out unattributed numbers.

### What worked

The document is internally consistent because every figure traces to a file in
`scripts/results/`. Writing the memory budget table as a multiplier against
on-disk Markdown gives a rule of thumb that transfers to other vaults: **~17
bytes of live heap and ~34 bytes of RSS per byte of Markdown** today, **~3 bytes
of live heap and ~14 bytes of RSS** with the on-disk index.

### What didn't work

I had to revise the TL;DR, the projection table, the options summary, the phase
notes, and the closing summary after the Step 6 measurement landed, because the
recommendation ordering had been written against an estimate. That is the right
outcome — the measurement overrode the prior — but it is a reminder not to write
recommendations before the last experiment finishes.

### What I learned

Ruling things out deserves its own section. The "what this code does NOT do"
table (raw Markdown not retained, plaintext not retained, assets not loaded,
`/api/notes` lightweight, `go:embed` not on the heap, no goroutine leak, not a
disk problem) is short, but it is what stops the next engineer re-investigating
the same seven dead ends.

### What was tricky to build

Presenting `Note.sourceHTML` fairly. It is redundant state that costs 68 MiB and
looks like a free win, but the doc comment at `vault.go:50-56` records a real bug
it already fixed. The document had to explain the constraint first and only then
offer three graded options (re-parse on rebuild, make the final pass
non-destructive, or render lazily behind an LRU), so nobody deletes the field on
the strength of the memory number alone.

### What warrants a second pair of eyes

- The Phase 1 vs Phase 3 ordering. I recommend shipping the volume and flag
  (Phase 3) in preference to raising the limit (Phase 1), on the strength of a
  dev-box measurement. Somebody with cluster access should sanity-check that the
  in-cluster footprint is comparable before skipping the limit raise entirely.
- The `GOMEMLIMIT` warning. I argue that setting it at the *current* 1536 MiB
  limit without the on-disk index would trade an OOMKill for a GC death spiral
  (984.9 MiB live heap leaves ~32 % headroom). That reasoning should be checked
  by someone who has tuned Go GC in production.

### What should be done in the future

Collect the `memory phase=` log lines from the actual crashing pod (Phase 2) and
append them to this ticket. They would either confirm the dev-box reproduction or
reveal an in-cluster factor I could not see.

### Code review instructions

- Start at the design doc §1 (TL;DR) and §7 (measurements); everything else
  elaborates those.
- Cross-check any number against `scripts/results/*.txt`.
- Check the citations: pick five `file.go:line` references at random and verify
  they say what the document claims.

### Technical details

Ticket layout:

```
ttmp/2026/08/09/PV-MEMORY-019--diagnose-and-fix-oomkill-.../
├── index.md                     overview + headline numbers + top-3 fixes
├── tasks.md                     9 tasks, phase-ordered
├── changelog.md
├── design/01-memory-usage-analysis-and-remediation-design.md
├── reference/01-diary.md        this file
└── scripts/
    ├── 01-measure-vault-on-disk.sh
    ├── vaultmem/main.go
    └── results/
        ├── 01-single-snapshot-memonly.txt
        ├── 02-two-overlapping-snapshots-memonly.txt
        ├── 03-single-snapshot-persistent-index.txt
        └── 04-heap-profile-top-inuse-space.txt
```

---

## Summary of findings

| # | Finding | Evidence |
|---|---|---:|
| 1 | A single cold load peaks at **1897.1 MiB RSS** against a 1536 MiB limit — the app cannot complete its first load inside the limit | `results/01` |
| 2 | The in-memory bleve index is **884.7 MB = 84.5 %** of live heap | `results/04` |
| 3 | The vault keeps **two** copies of every note's HTML (`HTML` + `sourceHTML`), ~156 MB | `results/04` |
| 4 | Two coexisting snapshots cost **3848.9 MiB peak RSS**; scaling is exactly linear | `results/02` |
| 5 | Overlapping builds slow each other ~3× (23.5 s → 70.7 s for the same vault load), a positive feedback loop | `results/02` |
| 6 | No `GOMEMLIMIT`, no `GOGC`, no pprof, no singleflight anywhere in the repo | grep |
| 7 | `RuntimeState.Reload` holds no lock around `loadSnapshot`; the old snapshot is retained 30 s | `runtime.go:100`, `:18` |
| 8 | **`--search-index-path` already exists and resolves the incident: peak RSS 800.3 MiB, fits the current limit with 48 % headroom, zero code change** | `results/03` |

---

## Step 8: Implement the in-repo fixes (reload serialisation, GOMEMLIMIT, index, pprof)

Turned the investigation into code. The constraint that shaped this step is
that the *measured* biggest win — moving the 884 MiB bleve index off the Go
heap — is a deployment change, and the Kubernetes manifest lives in a different
repository (`wesen/2026-03-27--hetzner-k3s`,
`gitops/kustomize/retro-obsidian-publish/deployment.yaml`). So this step ships
everything that is application code, wires the index change into the one
deployment this repo does own (docker-compose), and leaves the cluster manifest
as an explicit, written-down handoff rather than pretending it is done.

What is application code turned out to be substantial: reloads could overlap,
nothing bounded the Go heap against the container limit, nothing warned when the
index was being built into memory, the replaced snapshot was held for 30s, and
there was no way to profile the running process.

### Prompt Context

**User prompt (verbatim):** "what's next? live urls changing is fine.\n\ndid you fix the memory?"

**Assistant interpretation:** Answer honestly whether the memory work had been
implemented (it had not — only analysed), then implement it, along with the
slug ticket's Phase 4 which the "live urls changing is fine" clause unblocks.

**Inferred user intent:** Stop producing analysis and start landing fixes; the
URL-stability objection that had been holding back one item is withdrawn.

**Commits (code):** `945c2df` — "fix(PV-MEMORY-019): stop overlapping reloads and align the heap with the cgroup"; `292932f` — "feat(PV-MEMORY-019): add --pprof-addr and shorten the old-snapshot hold"

### What I did

- Added `reloadMu` to `RuntimeState` and took it for the whole of `Reload`, so
  two snapshot builds can never be in flight at once.
- Added a no-op guard: when the configured root is a **symlink** whose target is
  unchanged, `Reload` logs `reload_skipped_unchanged` and returns.
- Extracted `resolveRoot` so the guard and `loadSnapshot` agree on resolution.
- Wrote `pkg/server/memlimit.go`: `ApplyMemoryLimit` reads the cgroup v2/v1
  memory limit and calls `debug.SetMemoryLimit(limit * 0.85)`, called first
  thing in `server.Run`.
- Added an in-memory-index warning above 400 notes in `buildSearchIndex`.
- Wired `--search-index-path` into `docker-compose.yml` against a named
  (disk-backed) volume.
- Dropped `oldSnapshotCloseDelay` from 30s to 5s.
- Added `--pprof-addr` serving `net/http/pprof` on a separate listener.
- Tests: `TestReloadSkipsUnchangedSymlinkTarget`, `TestReloadIsSerialised`,
  `TestReadCgroupMemoryMax` (6 rows), `TestApplyMemoryLimitRespectsExplicitEnv`,
  `TestApplyMemoryLimitNoCgroup`.

### Why

Ranked by the measurements in Steps 3–6. Reload overlap is what turns a heavy
but survivable load into a crash loop (1897 MiB for one snapshot, 3849 MiB for
two), and it is entirely fixable in this repo. `GOMEMLIMIT` is the cheapest
mitigation and derives itself from the cgroup so the two numbers cannot drift.
The index is the biggest single win but needs a volume. pprof is what makes the
next investigation cost minutes.

### What worked

- The serialisation is simple and the live check is convincing: five concurrent
  `POST /api/admin/reload` produced five `reload_start`/`reload_swapped` pairs
  with no interleaving.
- The symlink guard behaves exactly as the production deployment needs: three
  webhooks against an unchanged symlink produced three
  `reload_skipped_unchanged` and no build; re-pointing the symlink produced one
  build and served the new revision (`From A` → `From B`).
- Deriving `GOMEMLIMIT` from the cgroup rather than hardcoding it means the
  deployment only has to set the container limit.
- `--search-index-path` genuinely needed no application change — the flag,
  `search.NewPersistent`, and the atomic build/rename/reopen were all already
  there and simply never switched on. Verified an index directory appears on
  disk when the flag is set.

### What didn't work

**The no-op guard broke two existing tests.**

```
--- FAIL: TestReloadRereadsVaultConfig
    runtime_test.go:404: secrets/plan should be published after the exclusion was removed
    runtime_test.go:414: index should be hidden after the config excluded it
--- FAIL: TestExplicitVaultConfigPathIsRereadOnReload
    runtime_test.go:488: secrets/plan should be excluded after the explicit config gained the pattern
```

The design doc's pseudocode skips whenever `EvalSymlinks(configuredRoot)` equals
the active `ResolvedRoot`. That is wrong in general: under a plain directory the
resolved root never changes while the *contents* do, so the guard silently
served stale content — which is exactly what those two tests assert against.
The fix narrows the guard to the case where "unchanged target" actually implies
"unchanged input": a symlink root, which is the git-sync deployment, where each
revision is a fresh immutable checkout. Under a plain directory `Reload` always
rebuilds, as before.

This is the second time in this session that a design document's pseudocode had
a real bug in it, and both times a pre-existing test caught it.

### What I learned

- "The input is unchanged" is a claim about the deployment topology, not about a
  path. Making the guard conditional on the root actually being a symlink is the
  difference between a correct optimisation and a stale-content bug.
- `GOMEMLIMIT` is a soft limit, and that cuts both ways: with a live heap that
  genuinely does not fit, it converts an OOMKill into a GC death spiral. It is
  only safe to ship *because* it is paired with the index change; on its own, at
  the current 1536 MiB with a 985 MiB live heap, it would make things worse. The
  code comment says so, because a future reader will otherwise assume it is a
  complete fix.
- cgroup v1 signals "unlimited" with a sentinel near max int64 rather than a
  keyword, so a naive parse yields a ~9 PiB "limit" and an 85% headroom
  calculation that overflows into nonsense. Both spellings are in the test table.

### What was tricky to build

Deciding where the no-op guard goes relative to the mutex. Putting it *before*
taking `reloadMu` looks cheaper — callers that have nothing to do would not
queue — but it is wrong: a caller that arrives while a build is in flight would
see the *old* resolved root, decide there is work to do, queue, and then rebuild
a revision the in-flight reload was already publishing. Taking the lock first
and re-checking inside means a queued caller sees the result of the reload it
waited on, which is what makes a burst of webhooks collapse to one build.

The second subtlety was the interaction with `closeSnapshotAfter`. Shortening
the delay to 5s is safe only because the swap itself is atomic and readers take
a snapshot pointer under `RLock`; a request that has already obtained the old
snapshot keeps a live pointer to it, so the delay only needs to outlive the
*request*, not the reload.

### What warrants a second pair of eyes

- The symlink-only condition in `canSkipReload`. If any real deployment uses a
  bind-mounted directory that is atomically replaced (rather than a symlink),
  the guard will not fire for it — that is the safe direction, but worth knowing.
- `memLimitHeadroom = 0.85`. It is a guess calibrated to leave ~460 MiB of a
  3 GiB limit for non-heap RSS. With a disk-backed index, page cache for the
  mmap'd bleve segments also lives outside the Go heap and inside the cgroup, so
  0.85 may still be too generous. Worth re-measuring after the index moves.
- `oldSnapshotCloseDelay` 30s → 5s. If any handler holds a snapshot across a
  long-running operation, 5s could close an index underneath it. I could not
  find such a handler; search is the longest and completes in milliseconds.

### What should be done in the future

- **The gitops manifest change, which is the actual biggest win and is not
  done**: a disk-backed `emptyDir` (explicitly *not* `medium: Memory`, which is
  charged to the same cgroup and would change nothing), `--search-index-path`
  pointing at it, the memory limit raised, and probe `initialDelaySeconds`
  raised above the 82s load time so a probe cannot restart the pod mid-load.
- Phase 5 of the plan: eliminate the duplicate per-note HTML (`Note.HTML` +
  `Note.sourceHTML`, ~68 MiB).
- Making `reloadHandler` answer 204 immediately and reload in the background.
  Now that reloads are serialised and idempotent, a client timeout and retry is
  harmless in the symlink deployment, so this is less urgent than it looked.
- Phase 7: incremental reload.

### Code review instructions

- `pkg/server/runtime.go`: `Reload` (the mutex and the guard placement),
  `canSkipReload` (the symlink condition — read its comment), `resolveRoot`.
- `pkg/server/memlimit.go`: `ApplyMemoryLimit` and `readCgroupMemoryMax`.
- `pkg/server/pprof.go`: confirm it is a separate `http.Server`, not routes on
  the public mux.
- Validate:

  ```bash
  go test ./publish-vault/pkg/server/... -count=1
  golangci-lint run ./pkg/...
  ```

  Live checks used here:

  ```bash
  # serialisation: 5 concurrent webhooks against a directory root
  for i in 1 2 3 4 5; do curl -s -X POST localhost:PORT/api/admin/reload & done; wait
  # no-op: 3 webhooks against an unchanged symlink root
  # then advance the symlink and confirm exactly one rebuild
  # pprof isolation
  curl -o /dev/null -w '%{http_code}' localhost:16060/debug/pprof/   # 200
  curl -o /dev/null -w '%{http_code}' localhost:18425/debug/pprof/   # 404
  ```

### Technical details

Observed reload phases, five concurrent webhooks against a directory root:

```
5 phase=reload_start
5 phase=reload_swapped
```

Three webhooks against an unchanged symlink, then one symlink advance:

```
3 phase=reload_skipped_unchanged
1 phase=reload_start
1 phase=reload_swapped
```

Startup warning when the index is built into memory:

```
warning: search index is in memory for N notes and will dominate heap usage;
pass --search-index-path <writable-dir> to keep it on disk (needs a volume, not tmpfs)
```

Still required, in `wesen/2026-03-27--hetzner-k3s`,
`gitops/kustomize/retro-obsidian-publish/deployment.yaml`:

```yaml
volumes:
  - name: search-index
    emptyDir: { sizeLimit: 4Gi }        # disk-backed; NOT medium: Memory
volumeMounts:
  - name: search-index
    mountPath: /var/lib/publish-vault
args: [ ..., "--search-index-path", "/var/lib/publish-vault/search" ]
resources:
  limits: { memory: 3Gi }
  requests: { memory: 2Gi }
```
