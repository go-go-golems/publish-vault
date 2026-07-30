# Tasks

## Phase 1 — `pkg/vaultconfig` package
- [x] 1. `go get` a gitignore library (sabhiram or monochromegane); record choice in diary
- [x] 2. Implement `Config`, `Load`, `DefaultConfigPath` in `pkg/vaultconfig/config.go`
- [x] 3. Implement `Matcher`, `NewMatcher`, `Match`, `Empty` in `pkg/vaultconfig/matcher.go`
- [x] 4. Write `pkg/vaultconfig/config_test.go` (missing file, `**`, negation, empty)

## Phase 2 — Vault gating (`pkg/vault/vault.go`)
- [x] 5. Add `configMatcher` field + `WithConfig` option
- [x] 6. Add `IsExcluded` (config blacklist OR .vault-ignore); replace `isIgnored` call sites
- [x] 7. Add `Note.Publish` + `frontmatterBool` helper; set in `loadNote`
- [x] 8. Skip `!Publish` notes in `LoadAll`; in `ReloadNote` return `ErrIgnored` + `RemoveNote`
- [x] 9. Add broken-embed marker for hidden-note embeds in `rebuildHTML`
- [x] 10. Extend `pkg/vault/vault_test.go` (publish:false, config blacklist, ignore-wins, reload-drops)

## Phase 3 — Serve command wiring
- [x] 11. Add `--config` flag to `cmd/.../serve/serve.go`; load config nil-safe
- [x] 12. Add `VaultConfig` to `server.Config` and thread through runtime → `vault.New(..., WithConfig)`
- [x] 13. Update watcher to call `IsExcluded` instead of `IsIgnored`

## Phase 4 — Tests, docs, examples
- [x] 14. Add `vault-example/.publish/config.yaml` + a `publish: false` note
- [x] 15. Update README (Frontmatter section, Excluding paths section, flags/env tables)
- [x] 16. Run full validation checklist (`go test ./...`, `go build -tags embed`, smoke `/api/*`)

## Phase 5 — PR #17 review fixes
- [x] 17. P1: delete unpublished notes from the search index in `--watch` mode (`ErrUnpublished`, `SlugForPath`)
- [x] 18. P1: re-read `.publish/config.yaml` per snapshot so admin reload / git-sync flips honour it
- [x] 19. P2: honour config negations in `ShouldPruneDir` (`Matcher.HasNegations`)
- [x] 20. P2: render HTML from parser output so broken-embed markers are not permanent
- [x] 21. P2: strip only complete frontmatter delimiter lines (`StripFrontmatter`)
- [x] 22. Rebuild HTML in `RemoveNote` so the embed marker returns when a target is unpublished
- [x] 23. Regression tests for each fix + README updates; re-run validation checklist
- [x] P2: handle a closing frontmatter delimiter at EOF without a trailing newline <!-- t:apxt -->
- [x] CI: bump x/text, excelize, otel to clear govulncheck findings <!-- t:awpp -->
