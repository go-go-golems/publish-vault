# Changelog

## 2026-08-09

- Initial workspace created


## 2026-08-09

Created PV-MEMORY-019 and added vocabulary topics memory, oom, profiling, reload, bleve, kubernetes.


## 2026-08-09

Measured the go-go-parc vault on disk: 1739 md files, 59,267,905 bytes (56.5 MiB) of markdown, 1970 asset files totalling 2166 MiB. 93% of the vault by bytes is attachments the app never loads.


## 2026-08-09

Built scripts/vaultmem measurement harness; measured a single cold snapshot at 984.9 MiB live heap, 1823.4 MiB HeapSys, 1897.1 MiB peak RSS against a 1536 MiB limit. The app cannot complete its first load inside the limit.


## 2026-08-09

Heap profile (inuse_space) attributes 884.7 MB (84.5%) to the in-memory bleve index and 155.8 MB to the vault's two copies of every note's HTML (parser.Parse 77.69 MB + rebuildHTML 78.07 MB).


## 2026-08-09

Reproduced the overlapping-reload failure mode: two coexisting snapshots cost 1967.9 MiB live heap and 3848.9 MiB peak RSS, scaling exactly linearly. The second vault load took 70.7s vs 23.5s for the first - overlapping builds slow each other, a positive feedback loop.


## 2026-08-09

KEY RESULT: measured the already-implemented --search-index-path option. Search index live heap 834.0 -> 15.7 MiB (-98.1%), one snapshot 984.3 -> 166.2 MiB (-83.1%), peak RSS 1897.1 -> 800.3 MiB (-57.8%). Fits the existing 1536 MiB limit with 48% headroom, zero application code change.


## 2026-08-09

Wrote the design document (incident analysis, system map, memory budget table, reload timelines, pseudocode for six fixes, API and file references, seven-phase plan, reproduction commands, ten open questions) and the investigation diary; added 9 phase-ordered tasks.


## 2026-08-09

Implemented the in-repo fixes (commits 945c2df, 292932f): Reload is serialised on a mutex and skips rebuilds when a symlink root is unchanged; ApplyMemoryLimit derives GOMEMLIMIT from the cgroup; buildSearchIndex warns when indexing a large vault into memory; docker-compose wires --search-index-path to a disk-backed volume; oldSnapshotCloseDelay 30s -> 5s; new --pprof-addr on a separate listener. Verified live: 5 concurrent webhooks = 5 serialised builds, 3 webhooks on an unchanged symlink = 0 builds, symlink advance = 1 build. The gitops manifest change (limit + emptyDir + flag) remains outstanding and is now tracked as two tasks.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/server/memlimit.go — GOMEMLIMIT derived from the cgroup limit
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/server/pprof.go — Separate pprof listener so the next investigation is one command
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/server/runtime.go — Reload serialisation and the symlink no-op guard

