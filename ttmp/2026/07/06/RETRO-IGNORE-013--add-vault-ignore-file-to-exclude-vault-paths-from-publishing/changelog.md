# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Created ticket + intern design/implementation guide for .vault-ignore (gitignore-compatible subset, filter-once-at-walk). Added watcher/ignore vocab topics. No code changes yet.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault.go — LoadAll is the primary integration point for ignore filtering


## 2026-07-06

Phase 1: added internal/ignore package (gitignore subset matcher) with table-driven tests (commit abad6df)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/ignore/ignore.go — Ignore.Load/Match/MatchAbs — the matcher consumed by vault


## 2026-07-06

Phase 2: wired .vault-ignore into vault.New/LoadAll/ReloadNote/ReadRaw + IsIgnored accessor; added tests (commit ccf7e0a)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault.go — LoadAll filters at the walk so all consumers are covered transitively


## 2026-07-06

Phase 3: watcher prunes ignored dirs, drops ignored events, no-ops on ErrIgnored; added test (commit 88987b6)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/watcher/watcher.go — two gates (New dir walk + loop event filter) keep ignored paths out of the hot reload path


## 2026-07-06

Phase 4: assetHandler returns 404 for ignored assets; raw endpoint already covered by ReadRaw guard (commit 39fe081)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/server.go — assetHandler closes the off-notes static-asset loophole


## 2026-07-06

Phase 6: documented .vault-ignore in README; end-to-end smoke test passed (notes/tree/search/raw/asset/watcher all respect ignore) (commit c9cdb03)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/README.md — Excluding paths section documents the supported gitignore subset and reload semantics


## 2026-07-06

Addressed PR #9 review: asset handler snapshot race (single snapshot for check+open) and negation-under-excluded-dir consistency (ShouldPruneDir/HasNegations); added tests + smoke (commit d7bc215)

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/server.go — assetHandler uses v.Root() for both IsIgnored and OpenRoot

