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

