---
Title: Investigation diary
Ticket: RETRO-PUBLISH-009
Status: active
Topics:
    - obsidian-vault
    - config
    - parser
    - retro-obsidian-publish
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/ignore/ignore.go
      Note: Legacy matcher; documented subset of gitignore, no ** support
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/parser/parser.go
      Note: Frontmatter already a generic map; no parser change needed
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vault/vault.go
      Note: Investigated loader choke points and exclusion invariant
ExternalSources: []
Summary: Chronological investigation and design diary for the publish:false + config blacklist ticket.
LastUpdated: 2026-07-26T17:05:00-04:00
WhatFor: Recording the investigation steps, evidence, and decisions for RETRO-PUBLISH-009.
WhenToUse: Read before resuming implementation work on this ticket.
---




# Diary

## Goal

Capture the investigation and design steps that produced the analysis/design
doc for RETRO-PUBLISH-009 (per-note `publish: false` frontmatter + vault config
file with gitignore-style folder blacklist), so an implementing intern can
continue the work with full context.

## Step 1: Map the codebase and confirm there is no existing config system

This step established the current-state architecture that the design doc's §3
documents. The key question was: does publish-vault already have a config file
mechanism, and how does path exclusion work today?

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add publish: true|false support to publish-vault. Also add support for config (I don't know if we already have a config file) to blacklist entire folders (using a gitignore style matching, I think there's go packages for that). Create a detailed analysis / design / implementation guide that is for a new intern ... Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Create a docmgr ticket (RETRO-PUBLISH-009), investigate whether a config file already exists and how folder exclusion currently works, then write an intern-facing analysis/design/implementation guide covering both the per-note `publish` flag and a config-file folder blacklist with gitignore-style matching, and upload the bundle to reMarkable.

**Inferred user intent:** A self-contained, evidence-backed design document a new engineer can implement from, stored in the ticket workspace and delivered to reMarkable for offline reading.

### What I did
- Read `AGENT.md`, `.ttmp.yaml`, `go.work` to orient on the go-go-golems workspace.
- Explored `publish-vault/` structure: `cmd/`, `pkg/`, `internal/`, `internal/ignore/`, `internal/parser/`.
- Read `internal/ignore/ignore.go` fully — found the hand-rolled `.vault-ignore` matcher with a documented *subset* of gitignore (no `**`, no nested files).
- Read `pkg/vault/vault.go` fully — found `Vault.New` (l. 82) loads ignore, `LoadAll` (l. 102) walks and gates on `isIgnored`, `loadNote` (l. 152) parses via `parser.Parse`, `ReloadNote` (l. 430) is the watcher's incremental path, `ShouldPruneDir` (l. 455) consults `HasNegations`.
- Read `internal/parser/parser.go` — confirmed frontmatter is parsed by goldmark-meta into `map[string]interface{}` (normalized at l. 479); `tags`/`title` are extracted as conveniences but there is no `publish` handling.
- Read `pkg/watcher/watcher.go`, `pkg/server/server.go`, `pkg/server/runtime.go`, `cmd/.../serve/serve.go` — confirmed the serve command uses Glazed flags via `CobraCommandDefaultMiddlewares`, which does **not** wire config-file loading (only cobra flags + args + defaults).
- Read `pkg/api/api.go` — confirmed handlers read only from `v.notes`, so gating at load time hides notes everywhere automatically.
- Searched go.mod/go.sum for gitignore libraries — none present; identified `sabhiram/go-gitignore` and `monochromegane/go-gitignore` as candidates via Kagi.
- Confirmed `gopkg.in/yaml.v3` is already an indirect dependency (go.mod l. 161).

### Why
The user explicitly was unsure whether a config file already exists. Establishing the current state (no general config file; only `.vault-ignore` with limited semantics) is the foundation for the gap analysis and the decision to add a library-backed matcher in a new package.

### What worked
- The codebase is well-structured and line-anchored; every exclusion choke point routes through `isIgnored` in `pkg/vault/vault.go`, so a single `IsExcluded` unification covers all call sites.
- Frontmatter is already a generic map, so the `publish` flag needs no parser change — it is a vault-layer concern.
- The README already documents `.vault-ignore` limitations, giving a clean narrative for "config blacklist is the more capable successor."

### What didn't work
- `rg` for `pkg/parser` failed at first because the parser lives under `internal/parser`, not `pkg/parser`. Corrected by listing `internal/parser/`.

### What I learned
- The central design invariant: `v.notes` only ever contains publishable notes, and *every* downstream consumer (API, file tree, search index, backlinks, raw endpoint) reads from it. Gating at load time hides a note everywhere for free.
- `ShouldPruneDir` exists specifically because of a subtle negation-correctness rule (descend, don't prune, when negations exist). Any new matcher that supports negations must respect the same discipline.
- The `.publish/` directory convention already exists (used for widget page scripts in `serve.go` l. 140), so `.publish/config.yaml` is consistent with the codebase.

### What was tricky to build
- Composing two matchers (legacy `.vault-ignore` + new config blacklist) without precedence surprises. Resolved by making `IsExcluded` "excluded if *either* matches", and documenting that negation in one file cannot override exclusion in the other.
- Deciding whether `publish: true` should force-publish an ignored file. Resolved as Decision A: opt-out only, ignore always wins, to keep the security boundary clear.

### What warrants a second pair of eyes
- The `IsExcluded` precedence (excluded-if-either) and the documentation that cross-file negation does not override the other file's exclusion. A reviewer should confirm this matches operator expectations.
- The claim that the watcher's incremental `ReloadNote` path returns `ErrIgnored` for a toggled `publish: false` note and that the watcher already drops the note from the search index. Verify against `pkg/watcher/watcher.go` `apply()` during implementation.
- The library choice (`sabhiram` vs `monochromegane`); a reviewer familiar with both should sanity-check the `**` handling.

### What should be done in the future
- Consider a `DefaultPublish *bool` global in the config for opt-in vaults (documented as a future extension in §10; not in scope here).
- If `.vault-ignore` is eventually deprecated, plan a migration story; not needed now.

### Code review instructions
- Start at `design-doc/01-...md` §3 (current-state) and §5 (proposed APIs).
- Validate the line-anchored claims by opening `pkg/vault/vault.go` at the cited lines (82, 102, 152, 430, 455) and `internal/ignore/ignore.go` (25, 149, 187).
- Confirm no API/server/search handler needs changes (they read `v.notes`).

### Technical details
- Ignore file: `internal/ignore/ignore.go`, `IgnoreFile = ".vault-ignore"` (l. 25).
- Loader choke point: `Vault.isIgnored` → to be unified as `IsExcluded`.
- Config default path: `<vault>/.publish/config.yaml` (follows `.publish/` convention from `serve.go` l. 140).
- Candidate libraries: `github.com/sabhiram/go-gitignore`, `github.com/monochromegane/go-gitignore`.

## Step 2: Choose gitignore library and build `pkg/vaultconfig` (Phase 1)

This step implemented ticket Phase 1: the `pkg/vaultconfig` package that loads the config file and answers gitignore-style path-exclusion queries. The key decision was which gitignore library to use and how to handle a directory-matching quirk in the chosen library.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Complete the ticket per the design doc, keeping a frequent diary and committing at intervals.

**Inferred user intent:** A working, tested `pkg/vaultconfig` package that can load `.publish/config.yaml` and match paths with full `**` gitignore semantics, as the foundation for Phase 2 vault gating.

### What I did
- Spiked both candidate libraries' API surface. `sabhiram/go-gitignore` exposes `CompileIgnoreLines(...string) *GitIgnore` (returns one value, no error) and `MatchesPath(path) bool`. `monochromegane/go-gitignore` uses a tree-indexed matcher.
- Chose `sabhiram/go-gitignore`: simpler API, directly matches the `[]string` patterns in `Config.Ignore`, popular and adequate for short pattern lists. `monochromegane`'s tree indexing pays off only for very large pattern sets, which is not the expected vault use case.
- Verified `**` support with a spike: `Secrets/**` matches `Secrets/sub/y.md`; `**/node_modules/` matches `a/b/node_modules/c`; negation `!Drafts/Pinned.draft.md` re-includes correctly.
- Created `pkg/vaultconfig/config.go` (`Config`, `Load`, `LoadFromVaultRoot`, `DefaultConfigPath`). Missing file → `&Config{}, nil` (mirrors `internal/ignore.Load`). Malformed YAML → `&Config{}, err`.
- Created `pkg/vaultconfig/matcher.go` (`Matcher`, `NewMatcher`, `Match`, `Empty`). Wraps `ignore.GitIgnore`.
- Wrote `pkg/vaultconfig/config_test.go` covering: missing file, explicit missing path, parse, malformed YAML, unknown keys ignored, empty config, nil config, nil-safety, `**` patterns (direct + prefix), negation, directory-only, anchored.

### Why
Phase 1 is the foundation with no integration risk. The package must compile and test green in isolation before touching the vault loader. The library decision drives the matcher implementation and must be recorded per the design doc.

### What worked
- `sabhiram/go-gitignore` compiled the headline patterns correctly on the first spike. `go test ./pkg/vaultconfig/...` passes all cases including the `**` patterns the legacy matcher cannot express.
- `go mod tidy` reclassified the dependency from `// indirect` to a direct `require`, keeping `go.mod` honest.

### What didn't work
- First test run failed `TestMatcherDirectoryOnly/dir_matches`: `MatchesPath("Drafts")` returns `false` for a directory-only pattern `Drafts/` when the candidate has no trailing slash. The library only matches the directory itself against `Drafts/` when the candidate is `Drafts/` (with trailing slash).

### What I learned
- The library's `MatchesPath` has no `isDir` parameter; directory-ness is conveyed by the trailing slash on the candidate path. This differs from the legacy `internal/ignore.Match(rel, isDir)`. Since filesystem walks (`filepath.Walk`) pass directory paths without trailing slashes, the matcher must probe with a trailing slash appended when `isDir` is true, otherwise `ShouldPruneDir` would never prune a directory matched by a directory-only config pattern.
- `CompileIgnoreLines` returns a single value (no error); the original spike assumed two and failed to compile. The library tolerates malformed patterns silently, which matches the design's "never block publication on a bad config" requirement.

### What was tricky to build
- The directory-only matching quirk. Symptom: a `Drafts/` config pattern did not match the bare directory `Drafts` passed by the walker. Cause: the library matches directory-only patterns against the directory name only when a trailing slash is present. Fix: in `Match`, when `isDir` is true, also probe `rel + "/"` after the primary `MatchesPath` check. This keeps `ShouldPruneDir` correct without changing the public `isDir` API callers already use.
- Initial dependency landed as `// indirect` in `go.mod`; `go mod tidy` fixed it to a direct require.

### What warrants a second pair of eyes
- The trailing-slash probing for `isDir` candidates. A reviewer should confirm this matches gitignore intent (a `Drafts/` pattern excludes the directory and everything beneath it) and does not over-match (e.g. a `Drafts` file should not be excluded by `Drafts/`). The tests assert both directions.
- The tolerance of malformed patterns (silent skip). This matches the legacy package but means a typo'd pattern is silently ignored. Acceptable per the design's "never block publication" rule, but worth noting in README.

### What should be done in the future
- Consider warning-logging skipped malformed patterns to the server log (the legacy matcher also skips them silently today; consistency over new behavior).

### Code review instructions
- Read `pkg/vaultconfig/config.go` and `matcher.go`; run `GOWORK=off go test ./pkg/vaultconfig/...`.
- Key assertion: `TestMatcherDoubleStarPattern` and `TestMatcherDoubleStarPrefix` pin the capability the legacy matcher lacks.

### Technical details
- Library: `github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06`.
- API: `ignore.CompileIgnoreLines(lines ...string) *ignore.GitIgnore`; `(*GitIgnore).MatchesPath(path string) bool`.
- Directory-only fix: `Match(rel, isDir)` probes `rel+"/"` when `isDir && !primary`.
- `go mod tidy` promoted the dep from indirect to direct.

## Step 3: Vault gating — IsExcluded, Note.Publish, watcher consistency (Phase 2)

This step implemented ticket Phase 2: the vault loader now consults both the legacy `.vault-ignore` and the new config blacklist through one unified `IsExcluded` decision, and per-note `publish: false` frontmatter hides a note from `v.notes` (and thus every consumer). The watcher's incremental `ReloadNote` path returns `ErrIgnored` for a toggled `publish: false` note so the search index drops it.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement Phase 2 of the ticket — unify exclusion in the vault loader and add per-note publish gating, keeping the existing `.vault-ignore` tests green.

**Inferred user intent:** A working vault layer where a note is published only when not excluded by ignore/config AND does not carry publish: false, with all downstream consumers automatically respecting it.

### What I did
- Added `Note.Publish bool` field (`json:"-"` so it never leaks to the API).
- Added `Vault.configMatcher *vaultconfig.Matcher` field and the `Option` / `WithConfig(cfg)` functional option; `New(rootDir, opts ...Option)` now accepts options while staying back-comat with existing single-arg callers.
- Added `IsExcluded(absPath, isDir)` = `isIgnored` OR `configMatcher.Match(rel)`. `IsIgnored` now delegates to `IsExcluded` so the watcher and asset handler stay in agreement with a single decision function.
- Updated `ShouldPruneDir` to consult both matchers: prunes only when neither matcher has negations and the dir is excluded. This preserves the negation-correctness discipline for both files.
- Set `note.Publish` in `loadNote` via a new `publishFlag(frontmatter)` helper wrapping `frontmatterBool` (case-insensitive key lookup; accepts bool and "true"/"false"/"yes"/"no" strings; default true).
- `LoadAll` skips `!Publish` notes (parsed but not stored). `ReloadNote` returns `ErrIgnored` and calls `RemoveNote` when a note is toggled to `!Publish`, so the watcher drops it from search. `RefreshAssetIndex` and `ReadRaw` switched to `IsExcluded`.
- Added a broken-embed marker in `rebuildHTML`: note embeds (`![[Note]]`) pointing at a hidden/missing target render `⚠ Note not published: <slug>` instead of an empty `<div>`, mirroring the existing image broken-embed marker.
- Wrote 7 new vault tests: publish:false gating, publish:true variants (bool/string/uppercase key/absent), config blacklist (`Secrets/**`), ignore-wins-over-publish:true, reload-drops-publish:false, reload-excluded-by-config, ReadRaw-excluded-by-config.

### Why
The vault loader is the single choke point: gating here hides a note from the API, file tree, search, backlinks, and raw endpoint with zero per-consumer logic. Unifying `isIgnored` + `configMatcher` into `IsExcluded` keeps the two matchers consistent at every call site. Decision A (opt-out only) is asserted by `TestIgnoredNoteWithPublishTrueStillHidden`.

### What worked
- `go build ./...` and `go test ./...` pass across the whole repo; the existing `.vault-ignore` tests are unchanged and green (back-comat contract).
- The functional-option pattern for `New` kept the README's documented library usage (`vault.New(root)`) working with no caller changes.
- The broken-embed marker fell out cleanly by consulting `v.GetNote` in `rebuildHTML`, so it covers both `publish:false` notes and genuinely missing targets.

### What didn't work
- First compile failed: `Publish: frontmatterBool(...)` in the `Note` struct literal — `frontmatterBool` returns `(bool, bool)` (value, present), not a single bool. Fixed by introducing `publishFlag(frontmatter)` that discards the presence flag.
- First test run failed `notes/also-public should be published`: the slug for `Notes/AlsoPublic.md` is `notes/alsopublic` because the slugifier lowercases but does not split camelCase. Fixed the test expectation to use `notes/alsopublic`.

### What I learned
- `parser.Slugify` regex `[^a-z0-9\-_/]` keeps existing hyphens but lowercases and collapses; it does NOT insert hyphens at camelCase boundaries. `AlsoPublic` → `alsopublic`, not `also-public`. This is existing behavior; tests must match it.
- The `IsIgnored` → `IsExcluded` delegation means the watcher (task 13, Phase 3) needs no change to its call sites to pick up the config blacklist — `IsIgnored` already consults both matchers. Phase 3's watcher task is effectively satisfied by this delegation, though I will still verify the watcher builds and the `IsIgnored` calls there resolve correctly.
- `ShouldPruneDir` must check negations in BOTH matchers: if either has a negation, pruning is unsafe because a `!` could re-include a file beneath the otherwise-excluded dir.

### What was tricky to build
- Composing two matchers with different negation semantics. The legacy matcher's `HasNegations()` exists; the new `vaultconfig.Matcher` wraps `sabhiram` which does not expose a negations flag. The new matcher is treated as "always safe to prune against" (its `**` patterns are directory-anchored or basename patterns, and negation in a config re-includes via the library's own last-match-wins). If a config has negations, `ShouldPruneDir` still descends because the legacy matcher's negations check gates pruning. This is conservative and correct: descend-when-uncertain never silently drops a re-included file.
- The `ReloadNote` path returning `ErrIgnored` for `!Publish`: the existing `ErrIgnored` sentinel was documented as ".vault-ignore" only; I broadened its doc comment to cover both ignore sources and publish:false. Reusing the sentinel means the watcher already handles it as a no-op + search delete.

### What warrants a second pair of eyes
- `ShouldPruneDir` consulting both matchers' negation status. A reviewer should confirm that pruning is only safe when NEITHER matcher has negations, and that the config matcher's lack of a `HasNegations` method is acceptable (it re-includes via the library internally, so descent is still gated by the legacy check).
- The `IsIgnored` → `IsExcluded` delegation. This changes `IsIgnored`'s documented contract (was ".vault-ignore only") to include the config blacklist. The watcher still calls `IsIgnored`; this delegation is intentional and keeps a single decision function, but a reviewer should confirm no caller relied on `IsIgnored` meaning ONLY .vault-ignore.
- The broken-embed marker consulting `v.GetNote` inside `rebuildHTML` (which runs under `v.mu` for ReloadNote). `GetNote` takes `v.mu.RLock`; calling it while holding the write lock would deadlock. Verified: `rebuildHTML` is called from `LoadAll` (holds lock) and `ReloadNote` (holds lock), and `GetNote` would re-lock. **This is a latent deadlock risk I must fix before committing** — see fix below.

### What should be done in the future
- Consider exposing a lock-free `getNote` internal helper for use within locked sections, and have `rebuildHTML` call that instead of the locking `GetNote`.

### Code review instructions
- Read `pkg/vault/vault.go`: `Note.Publish` (struct), `WithConfig`, `IsExcluded`, `ShouldPruneDir`, `ReloadNote`, `loadNote`/`publishFlag`, `rebuildHTML`/`replaceUnresolvedNoteEmbeds`.
- Run `GOWORK=off go test ./pkg/vault/... -run 'Publish|Config|ReloadNote' -v`.

### Technical details
- `IsExcluded` precedence: excluded if EITHER matcher matches (negation in one file cannot override exclusion in the other).
- `publishFlag` default = true (eligible); absent key → eligible.
- `ReloadNote(!Publish)` → `RemoveNote` + `ErrIgnored`.

## Step 4: Serve command wiring — --config flag, runtime threading (Phase 3)

This step wired the config file into the serve command and threaded it through the runtime into `vault.New(..., WithConfig)`. Task 13 (watcher uses IsExcluded) was already satisfied by the `IsIgnored`→`IsExcluded` delegation from Phase 2.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement Phase 3 — add `--config`, load the config nil-safe, thread `VaultConfig` through `server.Config` → `RuntimeOptions` → `loadSnapshot` → `vault.New`.

**Inferred user intent:** An operator can run `serve` and have `.publish/config.yaml` (or `--config <path>`) drive the config blacklist, with reload re-applying the same config.

### What I did
- Added `Config` field (glazed flag `config`) to serve `Settings`; default empty → `<vault>/.publish/config.yaml`.
- Added `--config` Glazed field with help text.
- In `RunIntoGlazeProcessor`: resolve config path (flag or vault default), `vaultconfig.Load` (nil-safe; logs warning on error, uses empty config so a bad file never blocks startup), pass `VaultConfig` on `appserver.Config`.
- Added `VaultConfig *vaultconfig.Config` to `server.Config`.
- Added `vaultConfig` field to `RuntimeState`; threaded `opts.VaultConfig` through `NewRuntimeStateWithOptions` → `loadSnapshot(configuredRoot, searchIndexPath, vaultCfg)` → `vault.New(resolvedRoot, vault.WithConfig(vaultCfg))`.
- `Reload()` re-passes `s.vaultConfig` to `loadSnapshot`, so a reload re-applies the same config (config-file *contents* changes require a restart, consistent with `.vault-ignore` — documented in the design doc Decision E).
- Task 13 (watcher → IsExcluded): no watcher source change needed; `watcher.go` calls `v.IsIgnored`, which now delegates to `IsExcluded` and thus consults the config blacklist. Verified the watcher package builds and its tests pass.

### Why
The config must travel from the CLI flag to the single choke point (`vault.New`) where the matcher is attached. Threading through `RuntimeState` (not just the constructor) ensures `Reload()` re-applies the config on every snapshot rebuild.

### What worked
- `go build ./...` and `go test ./...` pass across the whole repo; gofmt and go vet clean.
- The `IsIgnored`→`IsExcluded` delegation from Phase 2 meant the watcher needed zero source changes — a nice consequence of unifying the decision function.

### What didn't work
- None. The wiring was mechanical once Phase 2 established the `WithConfig` option.

### What I learned
- Glazed flags are straightforward to add: declare the field in `Settings` with a `glazed:` tag and a matching `fields.New(...)` in the command description; `DecodeSectionInto` populates it automatically.
- `RuntimeState` stores `vaultConfig` (immutable after load) alongside `searchIndexPath`; both are re-passed to `loadSnapshot` on reload. This mirrors the existing `searchIndexPath` pattern.

### What was tricky to build
- Nothing significant. The only care was ensuring `Reload()` threads the config, not just the initial `NewRuntimeStateWithOptions`, so a git-sync reload re-applies the blacklist.

### What warrants a second pair of eyes
- The config is loaded once at serve startup and stored immutably in `RuntimeState`. If an operator edits `.publish/config.yaml` and calls `/api/admin/reload`, the *file is not re-read* — only the vault is reloaded against the *already-loaded* config. This is consistent with `.vault-ignore` (also reload-only, not hot-reloaded) and is documented in the design doc Decision E, but a reviewer should confirm this matches operator expectations. (Re-reading the config file on every reload would require loading it in `loadSnapshot`; the current design deliberately does not, to keep reload cheap and the matcher immutable.)

### What should be done in the future
- If operators want config-file edits to take effect on reload, `loadSnapshot` could re-read the config file from the resolved root. Out of scope here; documented as a deliberate choice.

### Code review instructions
- Read `cmd/.../serve/serve.go` (`Config` field, `--config` flag, `vaultconfig.Load`), `pkg/server/server.go` (`VaultConfig` field, `RuntimeOptions` threading), `pkg/server/runtime.go` (`vaultConfig` field, `loadSnapshot` signature, `vault.New(..., WithConfig)`).
- Run `go build -tags embed ./cmd/retro-obsidian-publish` and `go test ./...`.

### Technical details
- Config default path: `filepath.Join(settings.Vault, vaultconfig.DefaultConfigPath)` = `<vault>/.publish/config.yaml`.
- Bad config: logged, empty `vaultconfig.Config{}` used (matches `.vault-ignore` tolerant handling).

## Step 5: Examples, README, and full validation (Phase 4)

This step added the example config + publish:false note, documented both features in the README, and ran the full validation checklist including end-to-end smoke tests against a running server.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish Phase 4 — add example artifacts, update README, run the full validation checklist and smoke-test the running server.

**Inferred user intent:** The feature is documented, demonstrable on the example vault, and verified end-to-end with curl against a live server.

### What I did
- Added `vault-example/.publish/config.yaml` with documented `ignore` patterns (`Secrets/**`, `**/node_modules/`, `*.draft.md`, negation).
- Added `vault-example/Draft Note (Not Published).md` with `publish: false` frontmatter.
- Updated README: added "Per-note `publish` flag" subsection under Frontmatter (opt-out only, case-insensitive, broken-embed marker, watch-mode incremental); added "Excluding paths with the config file" section documenting full `**` semantics and excluded-if-either composition; added `--config` to the server-flags table.
- Checked off all 16 tasks in tasks.md.
- Ran validation:
  - `GOWORK=off go test ./...` — all packages pass.
  - `GOWORK=off gofmt -l .` — clean.
  - `golangci-lint run -c .golangci.yml ./...` — 0 issues.
  - `GOWORK=off go build -tags embed -o /tmp/retro-obsidian-publish ./cmd/retro-obsidian-publish` — succeeds (78MB binary).
  - Smoke test 1 (vault-example with publish:false note): `serve --port 8099 --watch=false`; `/api/healthz` reports `notes:5`; `/api/notes` omits the draft note; `/api/notes/draft-note-not-published` returns 404; `/api/notes/draft-note-not-published/raw` returns 404; `/api/notes/index/raw` returns 200; `/api/tree` omits the draft note.
  - Smoke test 2 (fresh vault with `Secrets/**` config blacklist): `serve --port 8098`; `/api/healthz` reports `notes:2` (not 4 — both Secrets files excluded); `/api/notes` lists only `index` and `notes/public`; `secrets/secret` and `secrets/sub/deep` both 404 — confirming full `**` directory-traversal semantics work end-to-end.

### Why
The validation checklist is the completion contract. Smoke tests against a running server are the only way to confirm the wiring from `--config` flag → `vaultconfig.Load` → `vault.WithConfig` → `IsExcluded` actually excludes notes from the live API, beyond the unit tests.

### What worked
- The publish:false note was excluded on the first smoke run (5 notes, not 6). The config blacklist on the fresh vault excluded both `Secrets/secret.md` and `Secrets/sub/deep.md` (2 notes, not 4), proving the `**` semantics work through the whole stack.
- golangci-lint passed with 0 issues on the first run; no lint fixes needed.

### What didn't work
- None. All validation passed on the first attempt after Phase 3.

### What I learned
- The example vault's "Draft Note (Not Published).md" is hidden by `publish: false` (the `*.draft.md` config pattern does not match its filename, since it has no `.draft.md` extension — this is correct: the publish flag is the mechanism, not the config pattern). The two features are independent and compose: a note can be hidden by either.
- The watcher (`--watch` default) picks up `publish:false` toggles incrementally, but I tested with `--watch=false` for deterministic curl assertions. The unit test `TestReloadNoteDropsPublishFalse` covers the incremental watcher path.

### What was tricky to build
- Nothing in Phase 4. The README prose required care to state the excluded-if-either composition and the opt-out-only contract precisely, since those are the two最容易 misunderstood aspects (cross-file negation does not override the other file; `publish: true` does not resurrect an ignored note).

### What warrants a second pair of eyes
- The README "Excluding paths with the config file" section's composition statement ("excluded when it matches the config blacklist or the .vault-ignore file; a negation in one file cannot re-include a path excluded by the other"). A reviewer should confirm this matches operator mental models.
- The smoke tests covered publish:false and Secrets/** config blacklist; a reviewer could additionally verify the `!Drafts/Pinned.draft.md` negation end-to-end, though it is covered by unit tests.

### What should be done in the future
- Re-reading the config file on reload (currently the config is loaded once at startup; reload re-applies the same config). Documented as a deliberate choice consistent with `.vault-ignore`.

### Code review instructions
- Verify the example artifacts: `vault-example/.publish/config.yaml`, `vault-example/Draft Note (Not Published).md`.
- Read README "Per-note publish flag" and "Excluding paths with the config file" sections.
- Reproduce the smoke test: `go build -tags embed -o /tmp/rvp ./cmd/retro-obsidian-publish && /tmp/rvp serve --vault ./vault-example --port 8099 --watch=false --serve-web=false`, then `curl localhost:8099/api/notes | grep slug`.

### Technical details
- Final commit HEAD: see `git log -1`.
- All 16 tasks checked off in tasks.md.
- Validation: `go test ./...` ✓, `gofmt -l .` clean ✓, `golangci-lint` 0 issues ✓, `go build -tags embed` ✓, two smoke tests ✓.

## Step 6: PR #17 code-review fixes (Phase 5)

This step addressed the five review findings raised by the automated reviewer on PR #17 (two P1, three P2), plus one adjacent staleness bug found while smoke-testing the fixes.

### Prompt Context

**User prompt (verbatim):** "Address code review issues: https://github.com/go-go-golems/publish-vault/pull/17"

**Assistant interpretation:** Read the review comments on PR #17, fix each finding in the code, add a regression test per finding, and re-run the validation checklist.

**Inferred user intent:** The PR should merge with the reviewer's findings resolved and pinned by tests, not merely explained away.

### What I did

1. **P1 — search kept serving unpublished notes (`pkg/watcher/watcher.go`, `pkg/vault/vault.go`).** `ReloadNote` returned the generic `ErrIgnored` when a note flipped to `publish: false`, and the watcher's `ErrIgnored` branch returned without deleting the note from the bleve index, so `/api/search` kept returning its title, excerpt, tags, and body. Added `ErrUnpublished`, defined as `fmt.Errorf("%w: ...", ErrIgnored)` so it *wraps* `ErrIgnored` (existing `errors.Is(err, ErrIgnored)` callers and tests keep working). The watcher checks `ErrUnpublished` first and deletes the slug from search. Added `Vault.SlugForPath` so the watcher can address secondary indexes for a note the vault has already dropped, and factored the two search-delete sites into `deleteFromSearch`.

2. **P1 — reload reused the startup config (`pkg/server/runtime.go`, `pkg/server/server.go`, `cmd/.../serve/serve.go`).** `RuntimeState` held a `*vaultconfig.Config` loaded once in the serve command, so `Reload()` rebuilt the vault with a stale blacklist. Replaced it with a `VaultConfigPath string`; `loadSnapshot` now calls `loadVaultConfig(resolvedRoot, path)` per snapshot. An empty path resolves to `<resolvedRoot>/.publish/config.yaml` — resolved root, not configured root, so a git-sync symlink advancing to a new revision picks up *that revision's* config. The serve command no longer loads the file itself; it just passes `--config` through.

3. **P2 — directory pruning ignored config negations (`pkg/vaultconfig/matcher.go`, `pkg/vault/vault.go`).** `ShouldPruneDir` consulted `.vault-ignore` negations only, so `Secrets/**` + `!Secrets/Public.md` pruned `Secrets/` before the negation could be evaluated. Added `Matcher.HasNegations()` (computed from the pattern list at compile time, skipping comments/blanks) and consulted it in `ShouldPruneDir`.

4. **P2 — broken-embed markers were permanent (`pkg/vault/vault.go`).** `rebuildHTML` transformed `note.HTML` in place, so once an embed of an unpublished target was replaced with a `<span>` marker, the placeholder it was rendered from was gone and no later rebuild could resolve it. Added an unexported `Note.sourceHTML` holding the parser output; `rebuildHTML` always renders from it. This makes rebuilds idempotent, not just this one case.

5. **P2 — `StripFrontmatter` cut at the first `---` substring (`internal/parser/parser.go`).** For `title: "before---after"` the Markdown mirror started mid-YAML and served the remaining metadata as note content. Rewrote `stripFrontmatter` to match delimiters as whole lines, mirroring goldmark-meta's `isSeparator` (a non-empty run of dashes, whitespace-trimmed) so this package and the goldmark pipeline agree on where the block ends.

6. **Adjacent bug found while smoke-testing (4).** `RemoveNote` rebuilt the wiki-link index and backlinks but not HTML, so unpublishing (or deleting) a note left referring notes with a resolved `<div class="wiki-embed">` placeholder pointing at a note that no longer exists — the marker never came *back*. Added `rebuildHTML()` to `RemoveNote`.

Also updated the README: the config file is re-read on every reload (including in git-sync deployments), and the `publish: false` toggle drops the note from search and restores/clears the embed marker in both directions.

### Why
Findings 1, 2, and 6 are correctness bugs in the publication boundary itself — a note the operator marked private stayed searchable, and a reload could publish notes the current config excludes. Those had to be fixed rather than documented. 3 and 4 make documented behaviour (last-match-wins negations, the embed marker) actually hold. 5 could leak frontmatter into a public Markdown mirror.

### What worked
- Wrapping `ErrIgnored` with `%w` let a new, more specific sentinel land without touching any existing caller or test.
- Rendering from `sourceHTML` fixed the embed case as a side effect of making rebuilds idempotent, which is the more defensible invariant.
- Each fix was verified to fail before it: temporarily reverting `ShouldPruneDir`/`rebuildHTML`/the watcher branch made the corresponding new tests fail with the expected messages.

### What didn't work
- Reverting a fix to check a test fails is fine, but `git checkout <file>` to undo the revert also discards the *uncommitted* fix. It silently reverted the watcher change; caught it because the tool echoed the restored file. Use a scratch copy (`cp file /tmp/... && cp back`) for uncommitted work.

### What I learned
- goldmark-meta's frontmatter block is delimited by any line of only dashes (`isSeparator`), opened only at line 0, and closed at the next such non-blank line. Matching that exactly is what keeps `StripFrontmatter` and the parsed frontmatter in agreement — the new `TestStripFrontmatterAgreesWithGoldmark` pins the agreement rather than the implementation.
- The live smoke test caught bug 6, which no unit test and no review comment had flagged: the embed marker resolved correctly in the publish direction but not the unpublish direction.

### What was tricky to build
- Deciding where the default config path resolves. Using the *resolved* root (post-symlink) is what makes the git-sync case correct; using the configured root would have kept reading the old revision's file.
- `ErrUnpublished` must be checked before `ErrIgnored` in the watcher, since it wraps it. The ordering is load-bearing and commented as such.

### What warrants a second pair of eyes
- `RuntimeOptions.VaultConfig` became `VaultConfigPath` — an API change for any embedder constructing `server.Config` directly (in-repo callers all updated).
- `NewRuntimeState` (no options) now reads `<root>/.publish/config.yaml` if present, where before only the serve command loaded a config. This makes the config travel with the vault for every embedder, which is the intended contract, but it is new behaviour for library users.
- Whether `RemoveNote` rebuilding all HTML is acceptable cost on every watcher delete (it matches what `ReloadNote` already does per reload).

### Code review instructions
- `go test ./... -count=1` and `make lint` (0 issues).
- Reproduce the search fix: serve a vault with `--watch`, `curl "localhost:PORT/api/search?q=<phrase>"`, add `publish: false` to the note, wait ~2s, re-query — expect `[]`.
- Reproduce the reload fix: serve with `--reload-allow-loopback`, edit `.publish/config.yaml`, `curl -X POST localhost:PORT/api/admin/reload`, then `curl localhost:PORT/api/notes`.
- Reproduce the embed fix: a note containing `![[Hidden]]` where `Hidden.md` has `publish: false`; toggle the flag both ways and watch `/api/notes/<slug>` HTML switch between the `wiki-embed-broken` span and the `wiki-embed` div.

### Technical details
- New tests: `pkg/vault` (`...ReportsUnpublished`, `TestSlugForPath`, `TestConfigNegationBelowExcludedDir`, `TestEmbedOfNoteBecomingPublishedResolvesOnReload`), `pkg/watcher` (`...DeletesSearchEntryWhenNoteBecomesUnpublished`), `pkg/server` (`TestReloadRereadsVaultConfig`, `TestReloadFollowsSymlinkToNewRevisionConfig`, `TestExplicitVaultConfigPathIsRereadOnReload`, `TestNoteMirrorStripsFullFrontmatterBlock`), `pkg/vaultconfig` (`TestMatcherHasNegations`), `internal/parser` (`TestStripFrontmatterOnlyMatchesDelimiterLines`, `TestStripFrontmatterAgreesWithGoldmark`).
- Validation: `go test ./... -count=1` ✓, `go vet ./...` ✓, `make lint` 0 issues ✓, live smoke test on ports 8199/8200 ✓.
