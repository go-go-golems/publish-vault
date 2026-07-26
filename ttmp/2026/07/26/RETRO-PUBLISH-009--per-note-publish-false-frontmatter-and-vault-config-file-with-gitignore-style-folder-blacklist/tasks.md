# Tasks

## Phase 1 — `pkg/vaultconfig` package
- [ ] 1. `go get` a gitignore library (sabhiram or monochromegane); record choice in diary
- [ ] 2. Implement `Config`, `Load`, `DefaultConfigPath` in `pkg/vaultconfig/config.go`
- [ ] 3. Implement `Matcher`, `NewMatcher`, `Match`, `Empty` in `pkg/vaultconfig/matcher.go`
- [ ] 4. Write `pkg/vaultconfig/config_test.go` (missing file, `**`, negation, empty)

## Phase 2 — Vault gating (`pkg/vault/vault.go`)
- [ ] 5. Add `configMatcher` field + `WithConfig` option
- [ ] 6. Add `IsExcluded` (config blacklist OR .vault-ignore); replace `isIgnored` call sites
- [ ] 7. Add `Note.Publish` + `frontmatterBool` helper; set in `loadNote`
- [ ] 8. Skip `!Publish` notes in `LoadAll`; in `ReloadNote` return `ErrIgnored` + `RemoveNote`
- [ ] 9. Add broken-embed marker for hidden-note embeds in `rebuildHTML`
- [ ] 10. Extend `pkg/vault/vault_test.go` (publish:false, config blacklist, ignore-wins, reload-drops)

## Phase 3 — Serve command wiring
- [ ] 11. Add `--config` flag to `cmd/.../serve/serve.go`; load config nil-safe
- [ ] 12. Add `VaultConfig` to `server.Config` and thread through runtime → `vault.New(..., WithConfig)`
- [ ] 13. Update watcher to call `IsExcluded` instead of `IsIgnored`

## Phase 4 — Tests, docs, examples
- [ ] 14. Add `vault-example/.publish/config.yaml` + a `publish: false` note
- [ ] 15. Update README (Frontmatter section, Excluding paths section, flags/env tables)
- [ ] 16. Run full validation checklist (`go test ./...`, `go build -tags embed`, smoke `/api/*`)
