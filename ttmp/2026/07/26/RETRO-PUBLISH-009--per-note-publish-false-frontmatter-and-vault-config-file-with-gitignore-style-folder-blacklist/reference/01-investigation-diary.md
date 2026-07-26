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
