# Changelog

## 2026-07-26

- Initial workspace created


## 2026-07-26

Created ticket: per-note publish:false frontmatter + vault config file (.publish/config.yaml) with gitignore-style folder blacklist. Wrote intern-facing analysis/design/implementation guide and investigation diary.

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vault/vault.go — Loader is the unified exclusion choke point


## 2026-07-26

Phase 1: added pkg/vaultconfig package (config.go, matcher.go, config_test.go). Chose sabhiram/go-gitignore for full ** gitignore semantics. All tests green.

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vaultconfig/matcher.go — Wraps sabhiram/go-gitignore; isDir trailing-slash probing for directory-only patterns


## 2026-07-26

Phase 2: vault gating. Added Note.Publish, WithConfig option, IsExcluded (config OR .vault-ignore), ShouldPruneDir consulting both matchers, publish:false skip in LoadAll, ErrIgnored+RemoveNote in ReloadNote, broken-embed marker in rebuildHTML. 7 new tests. Fixed deadlock risk in replaceUnresolvedNoteEmbeds.

### Related Files

- /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vault/vault.go — IsExcluded unifies both matchers; ShouldPruneDir consults negations in both; rebuildHTML uses lock-free note lookup

