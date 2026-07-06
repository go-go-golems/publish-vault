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
    - Path: internal/vault/vault.go
      Note: LoadAll walk + hidden-dir skip; primary integration point for ignore filtering
    - Path: internal/watcher/watcher.go
      Note: New() dir walk + loop() .md filter; second integration point
    - Path: internal/server/server.go
      Note: assetHandler/validAssetPath; third integration point (asset + raw loopholes)
    - Path: internal/server/runtime.go
      Note: loadSnapshot rebuilds vault per reload — ignore is re-read for free
    - Path: internal/api/api.go
      Note: all read paths derive from v.notes, covered transitively by LoadAll filtering
    - Path: internal/parser/parser.go
      Note: downstream of the walk; confirms ignore belongs in vault, not parser
    - Path: internal/search/search.go
      Note: New/NewPersistent index from v.ForEachSearchDocument; covered transitively
    - Path: cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Settings struct; where a future --vault-ignore flag would land
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
