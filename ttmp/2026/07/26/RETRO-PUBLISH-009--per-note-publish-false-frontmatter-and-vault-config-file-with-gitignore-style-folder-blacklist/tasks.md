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
