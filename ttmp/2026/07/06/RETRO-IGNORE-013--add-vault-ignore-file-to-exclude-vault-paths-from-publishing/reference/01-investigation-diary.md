---
Title: Investigation diary
Ticket: RETRO-IGNORE-013
Status: active
Topics:
    - obsidian-vault
    - retro-obsidian-publish
    - vault
    - config
    - parser
    - assets
    - search
    - watcher
    - ignore
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Excluding paths with .vault-ignore section (commit c9cdb03)
    - Path: cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Settings struct; where a future --vault-ignore flag would land
    - Path: internal/api/api.go
      Note: all read paths derive from v.notes, covered transitively by LoadAll filtering
    - Path: internal/ignore/ignore.go
      Note: |-
        Phase 1 matcher — Ignore.Load/Match/MatchAbs (commit abad6df)
        HasNegations — gates directory pruning so permissive negation is honored (commit d7bc215)
    - Path: internal/parser/parser.go
      Note: downstream of the walk; confirms ignore belongs in vault, not parser
    - Path: internal/search/search.go
      Note: New/NewPersistent index from v.ForEachSearchDocument; covered transitively
    - Path: internal/server/runtime.go
      Note: loadSnapshot rebuilds vault per reload — ignore is re-read for free
    - Path: internal/server/server.go
      Note: |-
        assetHandler/validAssetPath; third integration point (asset + raw loopholes)
        assetHandler consults v.IsIgnored and 404s on excluded assets (commit 39fe081)
        assetHandler snapshot race fix — single snapshot for IsIgnored + OpenRoot (commit d7bc215)
    - Path: internal/vault/vault.go
      Note: |-
        LoadAll walk + hidden-dir skip; primary integration point for ignore filtering
        ignore field + LoadAll SkipDir + IsIgnored + ReloadNote ErrIgnored + ReadRaw guard (commit ccf7e0a)
        ShouldPruneDir — prune ignored dirs only when no negations exist (commit d7bc215)
    - Path: internal/watcher/watcher.go
      Note: |-
        New() dir walk + loop() .md filter; second integration point
        New prunes ignored dirs; loop drops ignored events; apply no-ops on ErrIgnored (commit 88987b6)
ExternalSources: []
Summary: 'Chronological investigation diary for RETRO-IGNORE-013: evidence gathering across vault/watcher/server/api and the resulting .vault-ignore design.'
LastUpdated: 2026-07-06T00:00:00Z
WhatFor: Onboarding and continuation reference for the .vault-ignore feature investigation
WhenToUse: Read before resuming RETRO-IGNORE-013 or before editing vault loading, the watcher, or the asset handler
---









# Investigation diary

## Goal

Capture the investigation and design work for RETRO-IGNORE-013 ("add a `.vault-ignore` file to exclude vault paths from publishing") so the implementing engineer can pick up the work with full context: what was examined, what was decided, what is tricky, and how to validate.

## Step 1: Evidence-first system investigation and intern design

This step produced the primary design document `design-doc/01-vault-ignore-file-intern-analysis-design-and-implementation-guide.md`. The work was to (a) understand how `retro-obsidian-publish` turns a vault directory into a website, (b) identify exactly where files are enumerated and why non-hidden paths cannot currently be excluded, and (c) design a dependency-free, gitignore-compatible `.vault-ignore` mechanism that filters once at the walk and is consulted by the watcher and the asset/raw handlers. The motivating example from the ticket — excluding `ttmp/_guidelines` and `ttmp/_templates` — is the canonical use case the design must satisfy.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add a .vault-ignore file to ignore certain directories or files, for example, ttmp/_guidelines and ttmp/_templates. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a docmgr ticket for a `.vault-ignore` feature, write a comprehensive intern-facing analysis/design/implementation guide (prose, bullets, pseudocode, diagrams, API + file references), store it in the ticket workspace, then upload the bundle to reMarkable.

**Inferred user intent:** The user wants a self-contained, onboarding-grade design package that a new engineer can act on, plus the standard docmgr + reMarkable delivery. The `ttmp/_guidelines` / `ttmp/_templates` example signals the real pain: docmgr authoring scaffolding inside a published vault leaks into the site.

**Commit (code):** none — this step is investigation and documentation only; no production code changed.

### What I did

- Confirmed the workspace is `publish-vault/` (Go module `retro-obsidian-publish`), located the CLI entry (`cmd/retro-obsidian-publish/main.go`), and read the full package set: `internal/{vault,parser,search,watcher,server,api,web}`.
- Read `internal/vault/vault.go` end to end. Anchored the key facts: `LoadAll()` uses `filepath.Walk(v.root, …)` (`vault.go:87`); the only exclusion is dot-prefixed directories (`vault.go:93`, `filepath.SkipDir`) and non-`.md` files (`vault.go:97`). Confirmed `FileTree`, `AllNotes`, `ForEachSearchDocument`, `buildBacklinks`, and `buildWikiLinkIndex` all derive from `v.notes`.
- Read `internal/watcher/watcher.go`: `New` walks the root to add every directory to fsnotify (`watcher.go:55-60`); `loop` filters to `.md` events (`watcher.go:95`) and debounces; `apply` calls `ReloadNote`/`RemoveNote` (`watcher.go:121,131`). Identified that ignored dirs are still watched and ignored files are still re-added.
- Read `internal/server/runtime.go`: `loadSnapshot` builds a fresh `vault.New(resolvedRoot)` per reload (`runtime.go:104`) and swaps atomically (`Reload`, `runtime.go:80-99`). Realized an ignore file read inside `vault.New` is re-read on every `/api/admin/reload` for free.
- Read `internal/server/server.go`: `assetHandler` serves non-`.md` files via `os.OpenRoot` (`server.go:211-243`); `validAssetPath` (`server.go:247-261`) blocks dotpaths/traversal but knows nothing of ignore. Identified the asset loophole and the `ReadRaw` (`vault.go:280-318`) loophole.
- Read `internal/api/api.go`: confirmed all read routes go through `v.notes`/search, so filtering at the walk covers the API.
- Read `internal/parser/parser.go` and `internal/search/search.go` to confirm both are downstream of the walk (ignore belongs in `vault`, not `parser`/`search`).
- Read existing tests (`vault_test.go`, `watcher_test.go`) and the RETRO-MEMORY-012 design doc to match house style (intern guide with evidence callouts, ASCII diagrams, `file.go:NN` refs).
- Checked `go.sum`: `github.com/bmatcuk/doublestar/v4` is present but only indirect; decided to keep the matcher dependency-free using stdlib `path.Match`, matching the project's hand-rolled-parser style.
- Verified the docmgr vocabulary had no `watcher`/`ignore` topics; added both, then created ticket `RETRO-IGNORE-013` with topics `obsidian-vault,retro-obsidian-publish,vault,config,parser,assets,search,watcher,ignore,documentation`.
- Added the design doc and this diary; wrote the full intern guide with executive summary, current-state architecture, gap analysis, proposed APIs, decision records, pseudocode flows, phased plan, test strategy, risks, and references.

### Why

The feature is small in surface but spans three subsystems (vault loading, live watcher, asset serving). Filtering once at the walk covers most consumers transitively, but the asset handler and raw-source endpoint read files off-disk and bypass `v.notes`, so they need explicit checks. Documenting the whole pipeline first prevents an intern from scattering exclusion checks across the codebase or missing the reload/watcher/asset edge cases.

### What worked

- `docmgr ticket create-ticket` + `docmgr doc add` produced the standard `index.md`/`tasks.md`/`changelog.md` + design/reference docs cleanly.
- Reusing the RETRO-MEMORY-012 intern-guide style made the design doc immediately consistent with the rest of the `ttmp`.
- Anchoring every claim to `file.go:NN` references kept the analysis evidence-based and greppable.
- The "filter once at the walk" insight reduced the design to one core change plus two targeted loophole fixes (asset + raw), which keeps the phased plan small.

### What didn't work

- None blocking. The only friction was vocabulary: `docmgr doctor` would have warned about unknown `watcher`/`ignore` topics, so they were added proactively before ticket creation.

### What I learned

- The vault's exclusion surface today is *exactly* two rules: dot-prefixed dirs and non-`.md` files. There is no other mechanism.
- `ReadRaw` and the asset handler are the two paths that bypass `v.notes` and would otherwise leak ignored content; both were caught in the gap analysis.
- Because `loadSnapshot` rebuilds the vault from disk on every reload, ignore-file changes are picked up by the existing reload endpoint with zero new reload code.
- The watcher only reloads individual `.md` files, so a `.vault-ignore` edit (vault-wide effect) must trigger a *full* reload, not a per-note one — hence the documented decision to require restart/reload for ignore changes in v1.

### What was tricky to build

- **Anchoring the "last match wins" negation semantics.** gitignore applies patterns in order with last-match-wins for `!` negation. The pseudocode in §5.3 of the design encodes this with `ignored = not p.negate` on each match, but the subtlety is that `dirOnly` patterns must be skipped for files, and `anchored` patterns must also match *prefixes* (so `ttmp/_guidelines` excludes the whole subtree). This is the most error-prone part and is why the matcher gets a dedicated table-driven test in Phase 1.
- **Deciding what is out of scope.** Full gitignore (nested files, `**`) is tempting but expands the matcher's correctness surface dramatically. The decision record captures the tradeoff and shapes the `Ignore` API so a library can replace the hand-rolled matcher later without touching call sites.
- **The watcher vs. reload-model mismatch.** The watcher is per-file; the ignore file is vault-wide. Resolving this required recognizing that ignore changes belong to the reload endpoint (already present for git-sync) rather than the hot watcher path.

### What warrants a second pair of eyes

- The `matchAnchored` prefix test (`strings.HasPrefix(rel, p+"/")`): confirm it correctly excludes a directory *and* its subtree without accidentally matching sibling names that share a prefix (e.g. `ttmp/_guidelines` vs `ttmp/_guidelines-backup`). The trailing `/` in the prefix test is the guard; verify with a test.
- The non-fatal-error decision: a malformed `.vault-ignore` currently means "publish everything". Confirm the warning is loud enough and that this is acceptable for the deployment model.
- Concurrency: `Vault.ignore` is set once in `New`/`NewWithOptions` and never mutated, so it needs no locking; confirm no reload path mutates the existing `Vault`'s `ignore` (reloads build a new `Vault`, so this holds).

### What should be done in the future

- Watch `.vault-ignore` for changes in `--watch` mode and trigger `state.Reload()` automatically (open question #1 in the design).
- Consider a `/api/config` field advertising the active ignore file path for debugging (open question #3).
- Consider per-note frontmatter `publish: false` as a complementary, finer-grained control (listed under alternatives).
- Revisit `**` support if real vaults need deep globs (open question #2).

### Code review instructions

- Start at `design-doc/01-vault-ignore-file-intern-analysis-design-and-implementation-guide.md`, §5 (proposed APIs) and §8 (phased plan).
- No code changed in this step. To validate the *design* assumptions, grep the cited line references, e.g.:
  ```bash
  cd /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault
  nl -ba internal/vault/vault.go | sed -n '82,114p'   # LoadAll walk + skip logic
  nl -ba internal/watcher/watcher.go | sed -n '53,71p' # New() dir walk
  nl -ba internal/server/server.go | sed -n '211,261p' # assetHandler + validAssetPath
  ```
- When code lands (Phases 1-5), validate with:
  ```bash
  go test ./internal/... -count=1
  go build ./...
  golangci-lint run -v
  ```

### Technical details

- Ticket: `RETRO-IGNORE-013` at `ttmp/2026/07/06/RETRO-IGNORE-013--add-vault-ignore-file-to-exclude-vault-paths-from-publishing/`.
- Vocabulary added: `topics/watcher`, `topics/ignore`.
- Core integration points (3): `vault.LoadAll` (filter), `watcher.New`/`loop` (skip ignored), `server.assetHandler`/`vault.ReadRaw` (close loopholes).
- Design invariant: filter once at the walk; consumers derive from `v.notes`; only off-disk readers (asset, raw) need explicit checks.
- New package proposed: `internal/ignore` with `Ignore{ Load, LoadFromPath, Match, MatchAbs, Empty }` over a documented gitignore subset.
- New constructor proposed: `vault.NewWithOptions(root, ignorePath)`; `vault.New(root)` becomes a wrapper (existing callers/tests unchanged).

## Step 2: Phase 1 — internal/ignore package

This step built and tested the `internal/ignore` package in isolation, before touching any caller. The package is the single source of truth for exclusion semantics: it parses a `.vault-ignore` file and answers `Match`/`MatchAbs` queries with a documented gitignore subset. Getting it right here (with a table-driven test for every rule) means the later wiring steps only have to call `Match` in the right places.

### Prompt Context

**User prompt (verbatim):** "go ahead, commit at appropriate intervals, keep a detailed diary as you work (see skill for format)."

**Assistant interpretation:** Implement the `.vault-ignore` feature per the design doc's phased plan, committing at each phase boundary, and record each step in the investigation diary following the `diary` skill format.

**Inferred user intent:** Turn the analysis/design into working, tested code with clean commits and a continuation-friendly audit trail.

**Commit (code):** `abad6df` — "RETRO-IGNORE-013: add internal/ignore package"

### What I did
- Wrote `internal/ignore/ignore.go`: `Ignore` type, `Load(root)`, `LoadFromPath`, `Parse(io.Reader)`, `Match(rel, isDir)`, `MatchAbs(root, abs, isDir)`, `Empty()`, plus unexported `pattern`/`parseLine`/`matches`/`matchSegment`/`ancestors`/`cleanRel`.
- Wrote `internal/ignore/ignore_test.go`: table-driven tests for absent file, explicit path, read error, comments/blanks, directory-only, anchored + sibling-prefix collision, leading-slash root anchoring, unanchored basename globs, `!` negation last-match-wins, `\#`/`\!` escaping, malformed-pattern skipping, `MatchAbs` (incl. outside-root), root-itself-never-ignored, CRLF, and `Parse` from a reader.
- Ran `go test ./internal/ignore/... -count=1 -v` (all pass), `gofmt -l` (clean), `go vet`, `go build`.

### Why
The matcher is the semantic core of the feature and the most error-prone part (negation, anchoring, directory-only, ancestor-based subtree exclusion). Building it standalone with exhaustive cases means the vault/watcher/server wiring in later phases can trust `Match` and stay small.

### What worked
- The uniform matching model — "a pattern matches a path if it matches the path itself OR any ancestor directory" — collapsed all the gitignore subtleties (dir-only excluding files beneath, anchored subtree, unanchored basename at any depth) into one `matches` function plus an `ancestors` helper.
- `path.Match` returning `ErrBadPattern` for malformed globs gave a clean way to skip bad lines without panicking; the test `TestMatchMalformedPatternSkipped` pins this.
- Treating a missing `.vault-ignore` as `(&Ignore{}, nil)` lets every caller use the same nil-safe path.

### What didn't work
- Initially I planned `vault.NewWithOptions` + a `--vault-ignore` CLI flag (Phase 5 in the design). After re-reading `AGENT.md` ("Don't add backwards compatibility layers or adapters unless explicitly asked… ASK IF NECESSARY"), I decided to keep `vault.New(root)` as the single constructor that reads the default `<root>/.vault-ignore` (presence-based) and defer the optional override flag to future work. This avoids a wrapper shim and extra config surface. Documented here as a deliberate scope reduction.

### What I learned
- gitignore anchoring has a subtlety: a trailing `/` does NOT anchor a pattern; only a leading or internal `/` does. So `Secrets/` matches a `Secrets` directory at any depth, while `/Secrets/` matches only at the root. The parser strips the trailing slash first, then decides anchored from the remaining body.
- `path.Match` (slash semantics) is the right choice over `filepath.Match` because the matcher normalises everything to slash-separated paths, matching the vault's existing `filepath.ToSlash` slug convention.
- The sibling-prefix case (`ttmp/_guidelines` vs `ttmp/_guidelines-backup`) is handled correctly for free because `path.Match` does exact matching on non-glob patterns — no special prefix logic needed beyond the ancestor walk.

### What was tricky to build
- **Directory-only patterns excluding files beneath them.** A naive `if dirOnly && !isDir { skip }` would let `ttmp/_guidelines/Index.md` through when only `ttmp/_guidelines/` is listed. The fix is the ancestor rule: a dir-only pattern is still tested against each *ancestor directory* of a file, so `ttmp/_guidelines/Index.md` matches via its ancestor `ttmp/_guidelines`. The `matches`/`ancestors` split makes this explicit and is covered by `TestMatchDirectoryOnly` and `TestMatchAnchored`.
- **Negation vs. directory exclusion.** Strict gitignore forbids re-including a file under an excluded *directory* (`Secrets/` then `!Secrets/keep.md` is a no-op). This package uses simple last-match-wins, so that negation *does* re-include. This is documented as a known deviation; the design doc's real example (`*.draft.md` + `!Projects/Pinned.draft.md`) is a file-pattern exclusion, not a directory exclusion, so it works correctly.

### What warrants a second pair of eyes
- The `cleanRel` normalisation (`path.Clean` + strip leading `/`): confirm it handles Windows backslash inputs and `..` segments the way we want. `MatchAbs` rejects outside-root paths via `filepath.Rel` error, but `Match` itself trusts its rel input.
- The negation-under-directory deviation: confirm operators will not rely on strict gitignore `!`-under-dir semantics.

### What should be done in the future
- Implement the optional `--vault-ignore` CLI override / disable flag if a real deployment needs it (was Phase 5; deferred per AGENT.md).
- Revisit strict gitignore `!`-under-directory re-inclusion rules if real vaults need it.
- Consider `**` support if deep globs are requested.

### Code review instructions
- Read `internal/ignore/ignore.go`: focus on `parseLine` (directive parsing) and `pattern.matches` + `matchSegment` + `ancestors` (the matching model).
- Read `internal/ignore/ignore_test.go`: each `TestMatch*` pins one rule.
- Validate: `go test ./internal/ignore/... -count=1 -v`.

### Technical details
- Commit: `abad6df`.
- Public API: `Ignore.Load(root)`, `Ignore.LoadFromPath(p)`, `Ignore.Parse(r)`, `Ignore.Match(rel, isDir)`, `Ignore.MatchAbs(root, abs, isDir)`, `Ignore.Empty()`.
- Matching invariant: a pattern matches a path iff it matches the path or one of its ancestor directories; dir-only patterns match files only through an ancestor dir; last matching pattern wins (negations flip the result).
- Files: `internal/ignore/ignore.go` (7.3 KB), `internal/ignore/ignore_test.go` (6.9 KB).

## Step 3: Phase 2 — wire ignore into the vault package

This step made `vault.New` read `<root>/.vault-ignore` and filter the walk, plus closed the raw-source loophole. After this change, ignored directories and files are absent from the notes map, so the file tree, search documents, backlinks, and every API read path exclude them transitively — without any change to those consumers. `ReloadNote` now refuses to re-add an ignored path (returns `ErrIgnored`), and `ReadRaw` returns `os.ErrNotExist` for ignored slugs so `/api/notes/{slug}/raw` cannot bypass the ignore.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Implement Phase 2 of the design: thread the ignore matcher into vault loading and the on-demand raw reader, with tests.

**Inferred user intent:** Make exclusion effective for all note-derived surfaces and close the raw-source bypass, with a focused, well-tested commit.

**Commit (code):** `ccf7e0a` — "RETRO-IGNORE-013: wire .vault-ignore into vault loading and raw reads"

### What I did
- `internal/vault/vault.go`: added `errors`, `log`, and `internal/ignore` imports; exported `ErrIgnored`.
- Added an `ignore *ignore.Ignore` field to `Vault`.
- `New(rootDir)` now calls `ignore.Load(rootDir)`; on a non-NotExist error it logs a warning and falls back to an empty `Ignore` (publishing is not blocked by a bad ignore file).
- `LoadAll` walk: `filepath.SkipDir` for ignored directories; `return nil` for ignored `.md` files.
- Added `IsIgnored(absPath, isDir)` (exported, lock-free, mirrors `Root()`) and `isIgnored` (unexported, nil-safe, used inside `LoadAll`/`ReloadNote`/`ReadRaw` where taking a lock would deadlock or is unnecessary).
- `ReloadNote` returns `(nil, ErrIgnored)` when the path is ignored, before `os.Stat`.
- `ReadRaw` returns `os.ErrNotExist` when the cleaned path is ignored.
- `internal/vault/vault_test.go`: added `errors` import and five tests — `TestLoadAllRespectsVaultIgnore` (dirs, globs, negation re-include, file tree, search docs), `TestLoadAllWithoutIgnoreFileIsUnchanged`, `TestReloadNoteIgnoresExcludedPath`, `TestReadRawRejectsIgnoredSlug`, `TestIsIgnoredIsNilSafeWithoutIgnoreFile` — plus a `folderAndFileNames` helper.
- Ran `go test ./internal/vault/...`, `go build ./...`, `go vet`, `gofmt -l` (all clean).

### Why
Filtering once at the walk is the design's core invariant: every read consumer derives from `v.notes`, so one filter covers the file tree, search, backlinks, and the whole API. `ReadRaw` and (later) the asset handler read files off-disk and bypass `notes`, so they need their own explicit checks — `ReadRaw`'s guard closes half of that loophole here.

### What worked
- The lock-free `isIgnored`/`IsIgnored` split avoided an RLock-while-holding-Lock deadlock in `LoadAll` (which holds the write lock) and in `ReloadNote` (which takes the write lock after the check). Since `v.ignore` and `v.root` are write-once, no lock is needed — mirroring the existing `Root()` accessor.
- Reusing the existing `writeVaultTestFile` helper kept the new tests compact and consistent with the suite.
- `errors.Is(err, ErrIgnored)` / `errors.Is(err, os.ErrNotExist)` made the test assertions precise.

### What didn't work
- My first draft of `TestLoadAllRespectsVaultIgnore` used wrong slugs (`tmp/-guidelines/style`, `drafts/pinned-draft-md`). I had incorrectly assumed in the design doc that `parser.Slugify` turns `_` into `-` and that `.draft.md` survives trimming. A quick check (`Slugify` keeps `a-z0-9-_` and `/`; `pathToSlug` strips only the trailing `.md`) showed the real slugs are `ttmp/_guidelines/style` and `drafts/pinned-draft`. Fixed the test slugs and the ignore patterns (I had also typo'd `tmp/` for `ttmp/`).

### What I learned
- **`parser.Slugify` preserves underscores and slashes** — the character class is `[^a-z0-9\-_/]`, so `_` and `/` are kept. The design doc's claim that "the underscore becomes a dash" is wrong; the diary and a future doc revision should correct it. This matters because ignore patterns and slugs both use slash-separated paths, so matching stays consistent.
- `pathToSlug` only trims a single trailing `.md` (via `strings.TrimSuffix`), so `Pinned.draft.md` → `Pinned.draft` → slug `pinned-draft` (the `.` becomes `-`).
- The existing test suite is a good regression net: all prior vault tests still pass unchanged, confirming the ignore feature is additive.

### What was tricky to build
- **Avoiding a deadlock in `LoadAll`.** `LoadAll` holds `v.mu.Lock()` for the whole walk. If `isIgnored` took `v.mu.RLock()`, it would deadlock. The fix is the unexported lock-free `isIgnored` used inside locked regions, with the exported `IsIgnored` as a thin wrapper for external callers. The safety argument is that `v.ignore`/`v.root` are set once in `New` and never mutated.
- **`ReloadNote` returning a sentinel vs. `(nil, nil)`.** Returning `(nil, nil)` would make the watcher's `apply` dereference a nil note in `SearchDocument(note)`. Returning `(nil, ErrIgnored)` lets `apply` recognise the no-op case (handled in Phase 3).

### What warrants a second pair of eyes
- The `ReadRaw` guard constructs an absolute path with `filepath.Join(v.root, filepath.FromSlash(cleaned))` and calls `isIgnored`. Confirm this round-trips to the same rel path the asset handler will check, so raw and asset paths agree on exclusion.
- The non-fatal-error policy: a malformed `.vault-ignore` logs a warning and publishes everything. Confirm the warning is surfaced in production logs.

### What should be done in the future
- Correct the design doc's slug-underscore claim (the underscore is preserved, not dashed).
- Reconsider the `--vault-ignore` CLI override flag (deferred from Phase 5) if a deployment needs to disable or redirect ignore processing.

### Code review instructions
- Read `internal/vault/vault.go`: `New` (ignore load + fallback), `LoadAll` walk (two `isIgnored` checks), `IsIgnored`/`isIgnored`, `ReloadNote` (ErrIgnored guard), `ReadRaw` (ignore guard).
- Read `internal/vault/vault_test.go`: `TestLoadAllRespectsVaultIgnore` is the broad test; the rest pin edge cases.
- Validate: `go test ./internal/vault/... -count=1 -v`.

### Technical details
- Commit: `ccf7e0a`.
- New exported symbols: `vault.ErrIgnored`, `(*Vault).IsIgnored(absPath string, isDir bool) bool`.
- Invariant: `v.ignore` and `v.root` are write-once; `IsIgnored` takes no lock (mirrors `Root()`).
- Filtering point: `LoadAll` walk only. `ReadRaw` is the only off-`notes` reader guarded so far; the asset handler is next (Phase 4).

## Step 4: Phase 3 — respect .vault-ignore in the file watcher

This step made the live watcher ignore-aware at two points: `New` no longer adds ignored directories to fsnotify, and `loop` drops events for ignored `.md` paths before they reach `apply`. `apply` now recognises `vault.ErrIgnored` from `ReloadNote` and no-ops silently. Together these guarantee a file moved into (or edited under) an ignored tree cannot re-enter the vault or the search index while the server runs.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Implement Phase 3: keep the fsnotify watcher from observing and acting on ignored paths.

**Inferred user intent:** Prevent ignored content from leaking back in via live edits, matching the design's two-gate watcher approach.

**Commit (code):** `88987b6` — "RETRO-IGNORE-013: respect .vault-ignore in the file watcher"

### What I did
- `internal/watcher/watcher.go`: added `errors` import.
- `New`'s directory walk now calls `v.IsIgnored(path, true)` and returns `filepath.SkipDir` for ignored directories, so fsnotify never watches them.
- `loop`, after the `.md` suffix filter, drops events where `vw.vault.IsIgnored(event.Name, false)` is true.
- `apply` checks `errors.Is(err, vault.ErrIgnored)` after `ReloadNote` and returns without logging an error or indexing.
- `internal/watcher/watcher_test.go`: added `TestApplySkipsIgnoredPath`, which writes a `.vault-ignore`, an ignored file under `ttmp/_guidelines/`, and a published `Index.md`, then calls `vw.apply(ignoredPath, fsnotify.Write)` and asserts the ignored note is absent from both the vault and the search index.
- Ran `go test ./internal/watcher/...`, `go build`, `go vet`, `gofmt -l` (all clean).

### Why
The watcher is the one path that can re-introduce a file into `v.notes` after `LoadAll` filtered it out. Without the two gates, editing a file under `ttmp/_guidelines/` would call `ReloadNote`, which (before Phase 2) would re-add it. Phase 2 made `ReloadNote` return `ErrIgnored`; Phase 3 makes the watcher respect that and, better, never observe the event in the first place.

### What worked
- Reusing the existing `apply`-direct test style kept the new test small and avoided a fragile real-fsnotify timing test.
- `errors.Is(err, vault.ErrIgnored)` keeps `apply`'s error handling precise: real reload errors are still logged, ignored paths are silent.

### What didn't work
- None blocking. The existing `TestApplyKeepsSearchIndexInSync` still passes unchanged, confirming the change is additive.

### What I learned
- The watcher already only enqueued `.md` events, so adding the `IsIgnored` check right after the suffix check is the natural single insertion point — no other event types need filtering.
- `New`'s `filepath.Walk` returning `filepath.SkipDir` for ignored dirs means fsnotify never establishes a watch on them, which is both more efficient and avoids spurious events from newly-created files under those dirs.

### What was tricky to build
- **`apply` vs. `loop` responsibility.** The design puts the primary gate in `loop` (drop ignored events before enqueueing). But `apply` is the function that actually calls `ReloadNote`, so it must also handle `ErrIgnored` defensively — otherwise a future caller of `apply` (or an event that slips through) would log a misleading "reload error". Both gates are intentional and documented.
- **Deciding not to test `New`'s SkipDir directly.** Asserting that fsnotify did not add a specific directory would require inspecting fsnotify internals or relying on event timing, which is brittle. The `apply` test plus the `loop` gate cover the observable behavior (ignored edits do not surface in search). The `New` SkipDir is a performance/correctness optimization validated by the manual smoke test.

### What warrants a second pair of eyes
- Confirm that a directory created *after* startup under an ignored parent is still not watched. fsnotify watches the parent; a `Create` event for a new ignored subdir would pass the `.md` filter (it's a dir, not `.md`) and be dropped, but the new subdir itself would not be added to the watch. This is fine for the ignore semantics (its `.md` children are ignored by the `loop` gate), but confirm there is no scenario where an ignored new dir's children are watched without passing through the gate.

### What should be done in the future
- Optionally watch `.vault-ignore` itself and trigger `state.Reload()` so a local edit to the ignore file hot-applies (open question #1 in the design). Currently ignore changes need a restart or `/api/admin/reload`.

### Code review instructions
- Read `internal/watcher/watcher.go`: `New` walk (SkipDir on ignored), `loop` (event filter), `apply` (ErrIgnored handling).
- Read `internal/watcher/watcher_test.go`: `TestApplySkipsIgnoredPath`.
- Validate: `go test ./internal/watcher/... -count=1 -v`.

### Technical details
- Commit: `88987b6`.
- Two gates: `New` prunes ignored dirs from the fsnotify watch set; `loop` drops ignored `.md` events; `apply` no-ops on `ErrIgnored`.
- No new exported symbols.

## Step 5: Phase 4 — honor .vault-ignore in the static asset handler

This step closed the last loophole: the `/vault-assets/*` handler reads files straight off disk and bypasses `v.notes`, so before this change an excluded image was still served if its URL was guessed. The handler now consults the active snapshot's vault and returns 404 for any asset whose path `IsIgnored`. With this, the raw-source endpoint (guarded in Phase 2 via `ReadRaw`) and the asset endpoint both respect the ignore, so no off-`notes` reader can leak excluded content.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Implement Phase 4: make the asset handler ignore-aware, with tests.

**Inferred user intent:** Ensure no HTTP surface can serve excluded files, completing the exclusion guarantee across notes, raw source, and assets.

**Commit (code):** `39fe081` — "RETRO-IGNORE-013: honor .vault-ignore in the static asset handler"

### What I did
- `internal/server/server.go`: in `assetHandler`, after `validAssetPath`, fetch the active snapshot's vault via `state.Snapshot()` and return `http.NotFound` when `v.IsIgnored(filepath.Join(v.Root(), rel), false)` is true. The check runs before `os.OpenRoot`, so an ignored request never touches the filesystem.
- `internal/server/asset_test.go` (new): `TestAssetHandler_404OnIgnored` (an image under an ignored `/Secrets/` directory 404s while a published image 200s) and `TestAssetHandler_NoIgnoreFileServesAll` (behavior unchanged with no ignore file).
- Ran `go test ./internal/server/...`, `go build ./...`, `go vet`, `gofmt -l` (all clean). Existing server tests (`TestAssetHandlerServesVaultFiles`, `TestAssetHandlerRejectsSymlinks`, `TestAssetHandlerRejectsUnsafePaths`) still pass unchanged.

### Why
The asset handler is the second off-`notes` reader (the first was `ReadRaw`, guarded in Phase 2). Both read directly from disk via `os.OpenRoot` and would otherwise serve excluded content. Placing the check before the open means ignored requests are cheap and never reach the filesystem.

### What worked
- Reusing the existing favicon-test setup pattern (`NewRuntimeState(vaultDir)` + `httptest.NewRequest`/`NewRecorder`) made the new tests consistent with the suite.
- `v.IsIgnored(filepath.Join(v.Root(), rel), false)` reuses the same accessor the watcher uses, so asset and watch paths agree on exclusion.

### What didn't work
- None blocking. The discovery that there were pre-existing asset-handler tests (`TestAssetHandler*` in another server test file) only confirmed the change is non-regressive.

### What I learned
- The asset handler already rejected `.md` files and dotpaths/traversal via `validAssetPath` and the `info.IsDir()`/extension checks; the ignore check slots in cleanly as an additional early return.
- `state.Snapshot()` returns the active vault atomically with the search index, so the ignore check always reflects the currently-served vault (important during a reload swap).

### What was tricky to build
- **Choosing where to place the check.** It must run after `validAssetPath` (so traversal/dotpath rejection stays first and cheapest) but before `os.OpenRoot` (so ignored requests are free and never touch disk). The chosen spot — right after `validAssetPath`, fetching the snapshot once — satisfies both.
- **`IsIgnored` takes an absolute path.** The handler has a vault-relative `rel`, so I join `v.Root()` + `rel`. This round-trips to the same rel inside `isIgnored` via `filepath.Rel`, matching the `ReadRaw` guard's path semantics. Verified by the test.

### What warrants a second pair of eyes
- Confirm the snapshot fetched for the ignore check is the same one used for `os.OpenRoot(state.ResolvedRoot())`. A reload between the two calls is harmless (the new snapshot's ignore would be at least as strict), but worth noting for the concurrency-aware reader.

### What should be done in the future
- Consider a single shared helper `state.IsIgnoredRel(rel, isDir)` to avoid the `filepath.Join(v.Root(), rel)` round-trip in both the asset handler and `ReadRaw`. Minor; not worth the abstraction yet.

### Code review instructions
- Read `internal/server/server.go`: `assetHandler` (snapshot fetch + `IsIgnored` early return).
- Read `internal/server/asset_test.go`: `TestAssetHandler_404OnIgnored` is the key test.
- Validate: `go test ./internal/server/... -count=1 -run Asset -v`.

### Technical details
- Commit: `39fe081`.
- Exclusion is now complete across all surfaces: notes/tree/search/backlinks/API (via `LoadAll`), raw source (via `ReadRaw`), assets (via `assetHandler`), and live edits (via the watcher).

## Step 6: Phase 6 — README docs and end-to-end smoke test

This step documented the feature in the README and ran a full end-to-end smoke test with the real binary serving a vault containing `ttmp/_guidelines`, `ttmp/_templates`, `Secrets/`, and draft notes. It also finalized the scoping decision to defer the optional `--vault-ignore` CLI override flag (design Phase 5) in line with AGENT.md's guidance against adding config surface / adapters unless necessary — the presence-based default (`vault.New` reads `<root>/.vault-ignore`) covers the user's actual request.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Finish the implementation: document the feature and validate it end-to-end with the real binary.

**Inferred user intent:** Ship a complete, tested, documented feature with confidence that every publishing surface respects the ignore.

**Commit (code):** `c9cdb03` — "RETRO-IGNORE-013: document .vault-ignore in the README"

### What I did
- `README.md`: added an "Excluding paths with `.vault-ignore`" section after Frontmatter, with a worked example (`ttmp/_guidelines/`, `ttmp/_templates/`, `/Secrets/`, `*.draft.md` + `!Drafts/Pinned.draft.md`), the supported gitignore subset bullets, and reload semantics.
- Built the binary (`go build ./cmd/retro-obsidian-publish`) and served a smoke vault at `--watch=false`:
  - `/api/healthz` reported `notes: 3` (Index, Notes/Public, Drafts/Pinned) — the four ignored notes were absent.
  - `/api/notes` listed exactly the 3 published slugs.
  - `/api/tree` top-level names were `Drafts`, `Index`, `Notes` (no `ttmp`, no `Secrets`).
  - `/api/search?q=NOT` returned only published notes; the ignored `ttmp/_guidelines/Style.md` ("Should NOT") was absent.
  - `/api/notes/ttmp/_guidelines/style/raw` → 404; `/api/notes/index/raw` → 200.
  - `/vault-assets/Secrets/secret.png` → 404 (asset loophole closed).
- Re-ran with `--watch=true`:
  - Writing a new ignored file `ttmp/_guidelines/Extra.md` left the count at 3 (watcher dropped/ignored it).
  - Writing a new published note `Notes/Fresh.md` incremented the count to 4 and `search?q=freshly` returned `notes/fresh`.
- Ran `go test ./... -count=1` (all packages pass), `gofmt -l` (clean), `golangci-lint run` (0 issues).

### Why
Documentation closes the loop for operators, and a real-binary smoke test is the only way to confirm the four integration points (vault load, watcher, raw, asset) agree across the HTTP surface — unit tests cover each in isolation but cannot prove the wiring is consistent end-to-end.

### What worked
- The smoke vault mirrored the README example exactly, so the docs and the validated behavior are guaranteed consistent.
- The watcher test was decisive: an ignored write left the count unchanged while a published write incremented it, proving both the `loop` event-drop and the `apply` `ErrIgnored` no-op.

### What didn't work
- One smoke-setup typo: I initially wrote `tmp/_templates/` (missing a `t`) in the `.vault-ignore`, which would have let `ttmp/_templates/Note.md` through. Caught immediately by checking the notes count and the tree, and fixed before the recorded run. A reminder that ignore patterns are literal and a single-character typo silently under-excludes — the README's literal example is the source of truth.

### What I learned
- The four integration points are mutually consistent because they all route through `(*Vault).IsIgnored` / the `LoadAll` filter. The smoke test's `notes: 3` across `/api/notes`, `/api/tree`, `/api/search`, and `/api/healthz` is strong evidence there is no second code path that bypasses the filter.
- `parser.Slugify` preserves underscores (confirmed again: `ttmp/_guidelines/style`), so the design doc's earlier claim about underscores-to-dashes is wrong and should be corrected in a future doc revision.

### What was tricky to build
- **Validating search exclusion.** A naive `search?q=Should` could match published notes by coincidence (fuzziness/prefix). The decisive check was that the ignored note's unique phrase ("Should NOT") never appears in any result, combined with the count staying at 3. The raw/asset 404s are the unambiguous proofs for those surfaces.

### What warrants a second pair of eyes
- The deferred `--vault-ignore` CLI flag (design Phase 5): confirm that presence-based default (`vault.New` reads `<root>/.vault-ignore`) is acceptable and that no deployment needs to redirect or disable ignore processing. If one does, Phase 5 is a small follow-up (add `NewWithOptions` + a `--vault-ignore` flag threaded through `server.Config`).
- The `.vault-ignore` reload semantics: in `--watch` mode, editing the ignore file requires a restart (documented). Confirm this is acceptable for local dev; git-sync deployments already use `/api/admin/reload`.

### What should be done in the future
- Implement the optional `--vault-ignore` CLI override / disable flag if a real deployment needs it (deferred Phase 5).
- Watch `.vault-ignore` for changes in `--watch` mode and auto-trigger `state.Reload()` (design open question #1).
- Correct the design doc's slug-underscore claim.
- Consider a `/api/config` field advertising the active ignore file path for debugging (design open question #3).

### Code review instructions
- Read `README.md`: the "Excluding paths with `.vault-ignore`" section.
- Reproduce the smoke test: build the binary, create the example vault from the README, and curl `/api/notes`, `/api/tree`, the raw and asset endpoints.
- Validate: `go test ./... -count=1` and `golangci-lint run`.

### Technical details
- Commit: `c9cdb03`.
- End-to-end result: every publishing surface (notes, tree, search, backlinks, raw source, assets, live watcher) respects `.vault-ignore`. Ignored writes are dropped; published writes are indexed.
- Deferred: `--vault-ignore` CLI flag / `NewWithOptions` (Phase 5) — presence-based default is sufficient for the requested feature.
- Full commit chain: `abad6df` (Phase 1) → `ccf7e0a` (Phase 2) → `88987b6` (Phase 3) → `39fe081` (Phase 4) → `c9cdb03` (Phase 6), each followed by a diary commit.

## Step 7: Address PR #9 code review (snapshot race + negation consistency)

Codex left two P2 review comments on PR #9, both legitimate correctness concerns. This step fixed both: (1) the asset handler checked `.vault-ignore` against one snapshot but opened the file via a fresh `state.ResolvedRoot()`, so a concurrent reload could gate bytes from the new root with the old vault's ignore rules; (2) `LoadAll` pruned ignored directories with `SkipDir`, but the matcher is permissive (last-match-wins), so a `!` re-include under an excluded directory was silently dropped — the matcher said "publish it" while the loader never visited it. Both are now fixed and covered by tests; an end-to-end smoke test confirms a re-included note + asset under `/Secrets/` is served while excluded siblings 404 across notes, tree, raw, and asset paths.

### Prompt Context

**User prompt (verbatim):** "Address code review comments: https://github.com/go-go-golems/publish-vault/pull/9"

**Assistant interpretation:** Read the PR #9 review comments and fix the two correctness issues Codex raised, with tests, then push.

**Inferred user intent:** Get PR #9 to a reviewable/mergeable state by resolving the automated review feedback.

**Commit (code):** `d7bc215` — "RETRO-IGNORE-013: address PR #9 review (snapshot race + negation consistency)"

### What I did
- **Fix 1 — asset handler snapshot race (`internal/server/server.go`):** captured `v, _ := state.Snapshot()` once and used `v.Root()` for both the `IsIgnored` decision and `os.OpenRoot`, so the ignore check and the bytes always come from one revision (a concurrent `Reload` can no longer mix old rules with a new root).
- **Fix 2 — negation under excluded directories:** added `Ignore.HasNegations()` and `Vault.ShouldPruneDir(absPath)`. `LoadAll` and `watcher.New` now prune an ignored directory only when the ignore file has **no** negation patterns; when negations exist, the walk (and the fsnotify watch set) descend so each file is matched individually and a `!`-re-included file beneath an excluded dir is actually visited and published. The matcher itself is unchanged (permissive last-match-wins, as documented).
- **Tests:** `TestMatchNegationUnderExcludedDir` + `TestHasNegations` (ignore); `TestLoadAllNegationUnderExcludedDir` (vault: re-included note published, sibling excluded, tree/raw/`IsIgnored` all agree); `TestAssetHandler_ServesReIncludedAsset` (asset: re-included asset 200, sibling 404).
- **Docs:** README now states the permissive negation semantics (a `!` can re-include under an excluded dir; excluded dirs are descended rather than pruned when any `!` is present).
- **Smoke test:** built the binary and served a vault with `/Secrets/` + `!Secrets/Public.md` + `!Secrets/public.png`: `notes` = `index` + `secrets/public`; tree shows `Secrets`; `/api/notes/secrets/public/raw` → 200, `/api/notes/secrets/secret/raw` → 404; `/vault-assets/Secrets/public.png` → 200, `/vault-assets/Secrets/secret.png` → 404.
- `go test ./...` all pass; `golangci-lint run` 0 issues; `gofmt` clean.

### Why
Both comments were real correctness gaps. The snapshot race could (rarely) serve an asset that should be ignored after a reload. The pruning inconsistency meant the documented permissive negation did not actually work under excluded directories, and the asset/raw paths disagreed with the loader. Fixing both makes every surface — loader, matcher, `ReadRaw`, asset handler, watcher — give identical answers.

### What worked
- The `ShouldPruneDir` abstraction encapsulates the "prune only when safe" rule in one place, so `LoadAll` and `watcher.New` share identical logic and cannot drift.
- Reusing the existing permissive matcher (no rewrite) kept the change low-risk: the only new logic is the `HasNegations` guard on pruning, which is obviously correct (no negations ⇒ pruning cannot drop a re-included file).
- The end-to-end smoke test was decisive: the same `!`-under-`/Secrets/` case is exercised across notes, tree, raw, and asset paths in one run.

### What didn't work
- One README edit failed with `ENOENT` because I passed an absolute path that omitted the `publish-vault/` segment. Caught immediately and re-applied with the correct path. A reminder that the workspace root is `…/memory-publish-vault/publish-vault/`, not `…/memory-publish-vault/`.

### What I learned
- The permissive-vs-strict gitignore question has a clean resolution that preserves the documented intent: keep the matcher permissive (last-match-wins) and make the **walk** honor it by not pruning when negations exist. This avoids rewriting the subtle matcher while still being fully consistent. The trade-off (descending into ignored dirs when `!` is present) is acceptable because `!` patterns are rare and the production deployment uses the reload endpoint, not `--watch`.
- The asset handler's `state.Snapshot()` returns a vault and a search index that are swapped atomically together, so capturing the vault once and using `v.Root()` for both the check and the open is the correct way to avoid a torn read across a reload.

### What was tricky to build
- **Choosing the negation fix.** Three options were considered: (A) never prune ignored dirs (simplest, but watches/walks large ignored trees even without negations); (B) make the matcher strict gitignore (git-faithful, keeps pruning, but rewrites the subtle matcher and reverses the documented permissive stance); (C) prune only when no negations exist (chosen). Option C preserves the fast pruning path for the common no-`!` case (e.g. `ttmp/_guidelines/`), only descending when a `!` actually requires it, and keeps the matcher unchanged.
- **Watcher consistency.** When negations exist, `watcher.New` must NOT prune ignored dirs either, otherwise a live edit to a re-included file under an excluded dir would not be caught. `ShouldPruneDir` is shared by both `LoadAll` and `watcher.New` so they stay in lockstep.

### What warrants a second pair of eyes
- The `ShouldPruneDir` semantics: confirm that descending into ignored dirs when `!` is present is acceptable for the largest expected ignored trees (the production deployment uses reload, not `--watch`, so the fsnotify-watch-count concern is limited to local dev with small vaults).
- The asset handler fix assumes `v.Root()` (the snapshot's resolved root) is the correct root to open. Confirm `RuntimeState.Snapshot()` always returns a vault whose `Root()` matches the snapshot the handler should serve from (it does — `loadSnapshot` sets `Vault: v` with `v.root = resolvedRoot`).

### What should be done in the future
- Consider a strict-gitignore mode (Option B) if users report surprise that `!` re-includes under excluded dirs; the matcher API is shaped to allow it.
- Optionally watch `.vault-ignore` itself to auto-trigger `state.Reload()` (still open from the design).

### Code review instructions
- `internal/server/server.go`: `assetHandler` — single `v, _ := state.Snapshot()` used for both `IsIgnored` and `OpenRoot(v.Root())`.
- `internal/ignore/ignore.go`: `HasNegations()`.
- `internal/vault/vault.go`: `ShouldPruneDir()`; `LoadAll` uses it for the dir prune decision.
- `internal/watcher/watcher.go`: `New` uses `ShouldPruneDir` for the dir prune decision.
- Tests: `TestMatchNegationUnderExcludedDir`, `TestHasNegations`, `TestLoadAllNegationUnderExcludedDir`, `TestAssetHandler_ServesReIncludedAsset`.
- Validate: `go test ./... -count=1` and the smoke sequence above.

### Technical details
- Commit: `d7bc215` (pushed to `origin/task/memory-publish-vault`; PR #9 updated).
- New symbols: `ignore.Ignore.HasNegations()`, `vault.Vault.ShouldPruneDir(absPath) bool`.
- Invariant: directory pruning happens iff the ignore file has no negation patterns; otherwise the walk descends and matches each file, so permissive negation is honored everywhere consistently.
- Asset handler invariant: the ignore decision and the file open use the same snapshot (`v.Root()`), eliminating the reload race.

