---
Title: Vault-ignore file — intern analysis, design, and implementation guide
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Settings struct + RunIntoGlazeProcessor -> server.Config; where a future --vault-ignore flag would land
    - Path: internal/api/api.go
      Note: |-
        route table; all read paths go through v.AllNotes()/GetNote(), so correct LoadAll filtering covers the API
        all read routes derive from v.notes — covered transitively by LoadAll filtering
    - Path: internal/parser/parser.go
      Note: |-
        Parse() — out of scope for ignore, but called by loadNote; explains why filtering happens at the walk, not the parser
        downstream of the walk; confirms ignore belongs in vault
    - Path: internal/search/search.go
      Note: New()/NewPersistent() index via v.ForEachSearchDocument — derives from notes map, covered transitively
    - Path: internal/server/runtime.go
      Note: |-
        loadSnapshot() builds a fresh Vault per reload (lines 92-130) — ignore file is re-read here for free
        loadSnapshot rebuilds vault per reload — ignore re-read for free
    - Path: internal/server/server.go
      Note: |-
        assetHandler + validAssetPath (lines 213-265) — assets must also respect ignore
        assetHandler + validAssetPath; asset loophole to close (lines 211-261)
    - Path: internal/vault/vault.go
      Note: |-
        LoadAll() filepath.Walk + hidden-dir skip (lines 84-114); loadNote, ReloadNote, RemoveNote, FileTree — all consumers of the notes map the ignore must filter
        LoadAll walk + hidden-dir skip; primary ignore integration point (lines 82-114)
    - Path: internal/watcher/watcher.go
      Note: |-
        New() Walk that adds dirs to fsnotify (lines 53-66); loop() .md filter (line 95); apply() -> ReloadNote/RemoveNote (lines 117-141)
        New() dir walk + loop() .md filter; second integration point
ExternalSources: []
Summary: 'Intern-facing analysis/design/implementation guide for a .vault-ignore file: explains the whole retro-obsidian-publish system, the gap (no way to exclude non-hidden paths), and a phased, dependency-free gitignore-compatible design.'
LastUpdated: 2026-07-06T00:00:00Z
WhatFor: Onboarding a new engineer to retro-obsidian-publish and shipping the .vault-ignore feature
WhenToUse: Read before changing vault loading, the watcher, the asset handler, or adding any vault-path exclusion
---







# Vault-ignore file — intern analysis, design, and implementation guide

**Ticket:** RETRO-IGNORE-013
**Audience:** A new engineer joining the team who needs to understand the whole system before adding the `.vault-ignore` feature.
**Goal of this document:** Explain, end to end, (1) what `retro-obsidian-publish` is and how a vault becomes a website, (2) where files are enumerated and why some paths cannot currently be excluded, (3) exactly what a `.vault-ignore` file is and the rules it should follow, and (4) a concrete, phased implementation plan with pseudocode, diagrams, and tests. Every claim is anchored to a file and (where useful) a line number.

---

## 0. How to read this document

- `monospace` tokens like `LoadAll` refer to code identifiers you can grep.
- `path/to/file.go:NN` is a line reference. Resolve these against the workspace root: `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/`.
- Diagrams are ASCII so they render everywhere, including the reMarkable upload.
- **"Evidence" callouts** quote observed source state captured on 2026-07-06.
- Decision records (`### Decision: ...`) capture the non-obvious choices and are safe to skip on a first read.

There is one repository involved in this task:

| Repo | Path on this machine | Role |
|------|----------------------|------|
| `publish-vault` | `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault` (this workspace) | The Go application source. The Go module is named `retro-obsidian-publish`. Code changes happen here. |

> **Note on names:** the *folder* is `publish-vault/`, the *Go module* is `retro-obsidian-publish`, and the *binary* is `retro-obsidian-publish`. This document uses `retro-obsidian-publish` when talking about the program and `publish-vault/` when talking about the directory.

---

## 1. Executive summary

- **The product.** `retro-obsidian-publish` turns a folder of Obsidian Markdown files (a "vault") into a small self-hosted website: a JSON API plus a retro monochrome React frontend, served from one Go binary. The vault directory is treated as **read-only content**; the app derives the website from it.
- **The gap.** Today the only automatic exclusion is *hidden paths*: any directory whose name starts with `.` is skipped during the filesystem walk (`internal/vault/vault.go:93`). There is no way to exclude **non-hidden** paths such as `ttmp/_guidelines/`, `ttmp/_templates/`, draft notes, or private folders. Users are forced to either prefix folders with a dot (which Obsidian and many tools dislike) or keep excluded content outside the vault.
- **The proposal.** Add an optional `.vault-ignore` file at the vault root, interpreted as a small, well-documented **gitignore-compatible subset**. When present, the vault loader, the file watcher, and the static-asset handler all consult it so that ignored paths are never indexed, served, or watched.
- **The design principle.** Filter **once, at the walk**, so every downstream consumer (search index, file tree, API, backlinks, asset serving) is covered transitively. This keeps the change small and avoids scattering exclusion checks across the codebase.
- **Scope of this ticket.** A new `internal/ignore` package, wiring it into `vault.LoadAll`/`ReloadNote`, `watcher`, and the `server` asset handler; an optional CLI flag; and tests. No changes to the parser, the frontend, or the on-disk vault format.

---

## 2. Problem statement and scope

### 2.1 The motivating problem

A user publishes a vault that also contains a `ttmp/` directory (a `docmgr` ticket workspace) plus `docmgr`-internal folders such as `ttmp/_guidelines/` and `ttmp/_templates/`. These are authoring scaffolding, not content the reader should see. Today they are published as ordinary notes and appear in the file tree, search, and tag cloud.

**Evidence** — the only exclusion logic in the loader (`internal/vault/vault.go:87-99`):

```go
err := filepath.Walk(v.root, func(path string, info os.FileInfo, err error) error {
    if err != nil {
        return nil // skip unreadable entries
    }
    if info.IsDir() {
        // Skip hidden dirs
        if strings.HasPrefix(info.Name(), ".") {
            return filepath.SkipDir
        }
        return nil
    }
    if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
        return nil
    }
    ...
```

That is the entire exclusion surface: dot-prefixed directories and non-`.md` files. There is no user-facing, file-based way to say "exclude `ttmp/_guidelines/`".

### 2.2 Goals

1. Let a vault author exclude directories and files by listing patterns in a `.vault-ignore` file at the vault root.
2. Make exclusion apply consistently to: note indexing, the file tree, full-text search, backlinks, the `/vault-assets/` static handler, and the live file watcher.
3. Use a syntax people already know (gitignore), scoped to a documented subset.
4. Re-read the ignore file on every reload so a git push that edits `.vault-ignore` takes effect without a restart.
5. Keep the change dependency-free and testable in isolation.

### 2.3 Non-goals

- A full gitignore engine (nested `.vault-ignore` files in subdirectories, `**` double-star across directories, character classes beyond what `path.Match` supports). These can come later.
- Changing the parser, the React frontend, or the on-disk vault format.
- Per-note frontmatter `publish: false` flags. (A reasonable future feature, but orthogonal to a path-based ignore file.)

---

## 3. Current-state architecture (evidence-based)

This section exists so a new engineer can orient before changing anything. If you already know the system, skip to [§4 Gap analysis](#4-gap-analysis).

### 3.1 The big picture

`retro-obsidian-publish` is a single Go binary with a Cobra/Glazed CLI. The `serve` command loads a vault directory into memory, builds a Bleve search index, and starts an HTTP server that exposes a JSON API and (optionally) the bundled React SPA.

```
                ┌───────────────────────────────────────────────────────────┐
   vault dir ──▶│  vault.Vault                                              │
   (Markdown)    │   LoadAll() walks the tree, parses each .md into a Note,  │
                 │   builds wiki-link index + backlinks + rendered HTML     │
                 └───────────────┬───────────────────────┬──────────────────┘
                                 │                       │
                                 ▼                       ▼
                     ┌───────────────────┐   ┌───────────────────────────┐
                     │ search.Index      │   │ server.RuntimeState       │
                     │ (Bleve, in-mem or │   │  holds ONE Snapshot:     │
                     │  on-disk snapshot)│   │  {Vault, Search, root}    │
                     └─────────┬─────────┘   └─────────────┬─────────────┘
                               │                           │
                               └──────────┬────────────────┘
                                          ▼
                            ┌──────────────────────────────┐
                            │ http.Server (gorilla/mux)    │
                            │  /api/*  JSON API            │
                            │  /vault-assets/*  static    │
                            │  /  SPA or SSR proxy        │
                            └──────────────────────────────┘
                                          ▲
                                          │ (optional) fsnotify
                            ┌──────────────────────────────┐
                            │ watcher.VaultWatcher          │
                            │  ReloadNote / RemoveNote      │
                            └──────────────────────────────┘
```

### 3.2 Package map

| Package | File | Responsibility |
|---------|------|-----------------|
| `cmd/retro-obsidian-publish` | `cmd/retro-obsidian-publish/main.go` | Process entry; builds the Cobra root command and executes it. |
| `commands` | `cmd/retro-obsidian-publish/commands/root.go` | Wires `serve` + `build` subcommands, Glazed logging/help. |
| `commands/serve` | `cmd/retro-obsidian-publish/commands/serve/serve.go` | Defines `serve` flags and the `Settings` struct; calls `server.Run`. |
| `internal/vault` | `internal/vault/vault.go` | **Core of this ticket.** Loads notes, computes slugs/backlinks/tree. |
| `internal/parser` | `internal/parser/parser.go` | Goldmark-based Markdown → HTML + frontmatter + wiki-link extraction. |
| `internal/search` | `internal/search/search.go` | Bleve full-text index over vault notes. |
| `internal/watcher` | `internal/watcher/watcher.go` | fsnotify watcher that hot-reloads changed `.md` files. |
| `internal/server` | `internal/server/server.go` | HTTP server, routes, health, reload, asset handler. |
| `internal/server` | `internal/server/runtime.go` | Atomic snapshot/reload state machine. |
| `internal/api` | `internal/api/api.go` | JSON API handlers (`/api/notes`, `/api/tree`, `/api/search`, …). |
| `internal/web` | `internal/web/*.go` | `go:embed` SPA handler. |

### 3.3 How a vault becomes notes (`internal/vault/vault.go`)

The `Vault` struct is an in-memory index keyed by slug:

```go
type Vault struct {
    mu            sync.RWMutex
    notes         map[string]*Note  // keyed by slug
    wikiLinkIndex map[string]string // short slug -> full vault slug
    root          string            // absolute path to vault directory
}
```

`New(rootDir)` creates a `Vault` and calls `LoadAll()` (`vault.go:64-78`). `LoadAll()` is the single place that enumerates files (`vault.go:82-114`):

1. It resets `v.notes`.
2. It calls `filepath.Walk(v.root, …)`.
3. For **directories**: if the name starts with `.`, it returns `filepath.SkipDir` (prune the subtree); otherwise it returns `nil` (descend).
4. For **files**: if the lowercased name does not end in `.md`, it is skipped; otherwise `loadNote` parses it.
5. Each parsed note is stored as `v.notes[note.Slug]`.
6. After the walk: `buildWikiLinkIndex()`, `buildBacklinks()`, `rebuildHTML()`.

`loadNote` (`vault.go:118-167`) reads the file, calls `parser.Parse`, derives a slug via `pathToSlug(relPath)`, and fills a `Note` struct (`Slug`, `Title`, `Path`, `Frontmatter`, `Tags`, `Excerpt`, `HTML`, `WikiLinks`, `ModTime`).

> **Evidence** — `pathToSlug` (`vault.go:331-336`) lowercases, replaces non-`[a-z0-9\-_/]` runs with `-`, and strips a trailing `.md`. So `ttmp/_guidelines/Index.md` → slug `ttmp-guidelines-index` (the underscore becomes a dash). That slug is what the API and URLs use.

Because `FileTree()` (`vault.go:288-326`), `AllNotes()`, `Count()`, `ForEachSearchDocument()` (`vault.go:235-262`), and the backlink/wiki-link builders **all derive from `v.notes`**, filtering at the walk is sufficient to exclude a path everywhere except the static-asset handler (see §3.6) and the watcher (see §3.5).

### 3.4 The parser is downstream of the walk (`internal/parser/parser.go`)

`parser.Parse(src)` takes already-read bytes and returns a `ParsedNote`. It knows nothing about the filesystem or whether a file "should" be published. This is why ignore logic belongs in `vault`, not `parser`. The parser also exposes `Slugify` (`parser.go` end), reused by the vault.

### 3.5 The live watcher (`internal/watcher/watcher.go`)

`watcher.New(v, opts)` (`watcher.go:43-71`) does its own `filepath.Walk(v.Root(), …)` to add **every directory** to the fsnotify watcher, then starts a debounce loop (`loop()`, `watcher.go:78-112`):

- It ignores events whose path does not end in `.md` (`watcher.go:95`).
- It debounces events on a 500ms ticker.
- `apply(path, op)` (`watcher.go:114-141`) handles `Remove`/`Rename` by calling `v.RemoveNote(path)`, and `Write`/`Create` by calling `v.ReloadNote(path)` and re-indexing the single note.

Two gaps for ignore:
1. `New` adds **all** directories, including ones an author wants excluded, so edits under an ignored tree still fire events (they are then re-added by `ReloadNote`).
2. `apply` never asks "is this path ignored?"; it unconditionally reloads or removes.

### 3.6 Reloads and the asset handler (`internal/server/`)

**Reloads.** `server.Run` constructs a `RuntimeState` via `NewRuntimeStateWithOptions` (`runtime.go:46-58`), which calls `loadSnapshot` (`runtime.go:92-130`). `loadSnapshot` resolves symlinks, builds a brand-new `vault.New(resolvedRoot)`, builds the search index, and returns a `Snapshot`. `RuntimeState.Reload()` (`runtime.go:80-99`) builds the next snapshot and atomically swaps it in, then closes the old one after a delay. **Implication:** because every reload calls `vault.New` from scratch, an ignore file read inside `vault.New` is naturally re-read on every reload — no extra wiring needed for the git-push reload path.

**Asset handler.** `/vault-assets/*` (`server.go:103`, `server.go:211-243`) serves arbitrary non-`.md` files from the vault root using `os.OpenRoot` (safe against traversal). `validAssetPath` (`server.go:247-261`) currently rejects empty, absolute, `.`/`..`, and dot-prefixed segments. It does **not** know about `.vault-ignore`, so an ignored image would still be served if someone guessed its URL. The design must extend this so ignored assets are also blocked.

### 3.7 The API surface (`internal/api/api.go`)

Routes registered in `Handler.Register` (`api.go:62-70`):

| Method | Route | Handler | Data source |
|--------|-------|---------|-------------|
| GET | `/api/config` | `getConfig` | `v.Count()` |
| GET | `/api/notes` | `listNotes` | `v.AllNotes()` |
| GET | `/api/notes/{slug}` | `getNote` | `v.GetNote(slug)` |
| GET | `/api/notes/{slug}/raw` | `getNoteRaw` | `v.ReadRaw(note.Path)` |
| GET | `/api/tree` | `getTree` | `v.FileTree()` |
| GET | `/api/search` | `searchNotes` | `search.Index.Search` |
| GET | `/api/tags` | `listTags` | `v.AllNotes()` tags |

Every read path goes through `v.notes` (or the search index, which is built from `v.notes`). Correct filtering in `LoadAll` therefore covers the entire API. `ReadRaw` (`vault.go:280-318`) opens via `os.OpenRoot` and validates the path is a clean `.md` — it should also reject ignored paths so a raw source fetch cannot bypass the ignore.

---

## 4. Gap analysis

| Concern | Current behavior | Desired behavior |
|---------|-------------------|------------------|
| Exclude a non-hidden directory (e.g. `ttmp/_guidelines/`) | Impossible; it is indexed as notes. | Excluded by a `.vault-ignore` entry. |
| Exclude a non-hidden file (e.g. `Drafts/WIP.md`) | Impossible. | Excluded by a `.vault-ignore` entry. |
| Re-include an exception under an excluded tree (e.g. keep `ttmp/Index.md`) | N/A | Supported via `!` negation. |
| Live watcher respects ignore | Watches and reloads everything under `.md`. | Does not watch ignored dirs; ignores events for ignored files. |
| Asset serving respects ignore | Serves any non-dotpath, non-`.md` file. | Returns 404 for ignored assets. |
| Raw source endpoint respects ignore | Serves any valid `.md` slug. | Returns 404 for ignored slugs. |
| Reload picks up ignore changes | Reload rebuilds the vault, but no ignore file exists to read. | Re-reads `.vault-ignore` on every reload. |
| Configurability | None. | Default presence-based; optional `--vault-ignore` override; can be disabled. |

---

## 5. Proposed architecture and APIs

### 5.1 New package `internal/ignore`

A small, dependency-free package with one type and a few functions. It owns parsing and matching so the rest of the codebase depends on a single, well-tested abstraction.

**API sketch:**

```go
// Package ignore parses a .vault-ignore file and answers path-exclusion queries.
package ignore

// Ignore holds the compiled patterns from one .vault-ignore file.
type Ignore struct {
    patterns []pattern
    hasDirs  bool // hints whether any pattern targets directories
}

// Load reads <root>/.vault-ignore (path.Join(root, ".vault-ignore")).
// If the file does not exist, Load returns an empty Ignore and a nil error,
// so callers can treat "no ignore file" and "empty ignore file" identically.
func Load(root string) (*Ignore, error)

// LoadFromPath reads an explicit ignore file path. Use when an operator
// passes --vault-ignore /custom/path.
func LoadFromPath(path string) (*Ignore, error)

// Match reports whether relPath (vault-relative, slash-separated) is ignored.
// isDir indicates whether the path is a directory; patterns ending in "/"
// only match directories. Negations (!) are applied in order, last-wins,
// mirroring gitignore semantics.
func (ig *Ignore) Match(relPath string, isDir bool) bool

// MatchAbs converts absPath to a vault-relative path against root and calls Match.
// absPath must be inside root.
func (ig *Ignore) MatchAbs(root, absPath string, isDir bool) bool

// Empty reports whether no patterns were loaded.
func (ig *Ignore) Empty() bool
```

**Pattern model (internal, unexported):**

```go
type pattern struct {
    negate   bool   // true if the line started with "!"
    dirOnly  bool   // true if the line ended with "/"
    anchored bool   // true if the line contains a "/" (anchored to vault root)
    pat      string // compiled glob body (no leading "!", no trailing "/")
}
```

### 5.2 Supported gitignore subset

The file is plain text, one pattern per line. The supported subset is deliberately small and explicit:

- **Blank lines** are ignored.
- **`#` comments**: a line whose first non-space character is `#` is a comment. To match a literal `#`, escape with `\#`.
- **Negation `!`**: a leading `!` re-includes a path that an earlier pattern excluded. Last match wins.
- **Trailing `/`**: the pattern matches directories only (e.g. `ttmp/_guidelines/`).
- **Leading `/` or any internal `/`**: the pattern is **anchored** to the vault root (e.g. `/Secrets/`, `ttmp/_templates`). Without a slash, the pattern matches at any depth (e.g. `*.draft.md`, `Drafts`).
- **Globs**: `*` matches any run of characters **within a single path segment** (does not cross `/`); `?` matches one character; `[abc]` matches one of the listed characters. This is exactly Go's `path.Match`/`filepath.Match` semantics, applied per segment for non-anchored patterns.
- **Escaping**: a leading `\` escapes the next character (so `\!keep.md` matches a file literally named `!keep.md`).
- **Not supported (documented):** nested `.vault-ignore` files in subdirectories, `**` recursion, and per-pattern scope beyond the above. (Rationale in §6.)

**Example `.vault-ignore`:**

```
# --- docmgr authoring scaffolding, never publish ---
ttmp/_guidelines/
ttmp/_templates/
ttmp/_guidelines
ttmp/_templates

# Exclude a whole private folder (anchored)
/Secrets/

# Exclude drafts anywhere in the vault
*.draft.md

# ...but keep this one published even though it ends in .draft.md
!Projects/Pinned.draft.md
```

### 5.3 Matching algorithm (pseudocode)

```
function Match(relPath, isDir):
    rel = normalize(relPath)            # toSlash, clean, strip leading "/"
    ignored = false
    for p in patterns:                  # patterns kept in file order
        if p.dirOnly and not isDir:
            continue
        if matches(p, rel):
            ignored = not p.negate     # last matching pattern wins
    return ignored

function matches(p, rel):
    if p.anchored:
        return matchAnchored(p.pat, rel)   # match against full rel, or a prefix
                                             # when p targets a directory subtree
    else:
        # unanchored: match the pattern against ANY single path segment,
        # and also against the full path (gitignore-ish "basename or path")
        for seg in split(rel, "/"):
            if pathMatch(p.pat, seg): return true
        return pathMatch(p.pat, rel)

function matchAnchored(p, rel):
    # A pattern like "ttmp/_guidelines" (dir-ish) should match the dir itself
    # AND everything beneath it. We test exact match first, then prefix match.
    if pathMatch(p, rel):               return true
    if strings.HasPrefix(rel, p + "/"): return true
    return false
```

`pathMatch` is Go's `path.Match`. `matchAnchored`'s prefix test is what makes `ttmp/_guidelines` exclude the entire `ttmp/_guidelines/` subtree.

### 5.4 Where ignore plugs in

```
                    .vault-ignore (at vault root)
                            │  ignore.Load(root)
                            ▼
                     internal/ignore.Ignore
                            │
       ┌────────────────────┼────────────────────────┐
       ▼                    ▼                        ▼
 vault.LoadAll()     watcher.New()            server.assetHandler
  - SkipDir on        - don't add ignored       - validAssetPath
    ignored dirs        dirs to fsnotify          also checks ignore
  - skip ignored      - loop(): ignore events    vault.ReadRaw()
    files                for ignored paths        - reject ignored slug
  - ReloadNote():
    return early if ignored
```

### 5.5 Wiring into `Vault`

Add an `ignorer` field to `Vault` and load it in `New`:

```go
type Vault struct {
    mu            sync.RWMutex
    notes         map[string]*Note
    wikiLinkIndex map[string]string
    root          string
    ignore        *ignore.Ignore   // NEW; nil-safe (treated as "ignore nothing")
}

func New(rootDir string) (*Vault, error) {
    ig, err := ignore.Load(rootDir)
    if err != nil {
        // Non-fatal: log and proceed with an empty Ignore so the vault still loads.
        log.Printf("vault: warning reading .vault-ignore: %v", err)
        ig = &ignore.Ignore{}
    }
    v := &Vault{
        notes: make(map[string]*Note), wikiLinkIndex: make(map[string]string),
        root: rootDir, ignore: ig,
    }
    if err := v.LoadAll(); err != nil {
        return nil, err
    }
    return v, nil
}
```

`LoadAll`'s walk callback gains two checks:

```go
if info.IsDir() {
    if strings.HasPrefix(info.Name(), ".") {
        return filepath.SkipDir
    }
    if v.isIgnored(path, true) {          // NEW
        return filepath.SkipDir
    }
    return nil
}
// ... file branch ...
if v.isIgnored(path, false) {             // NEW
    return nil
}
```

`ReloadNote` (`vault.go:226-241`) returns early when the changed file is now ignored (a file can move *into* an ignored tree via rename), and `RemoveNote` is already safe. Expose an accessor for the watcher:

```go
// IsIgnored reports whether absPath is excluded by the current .vault-ignore.
// Safe to call with a nil Ignore (returns false).
func (v *Vault) IsIgnored(absPath string, isDir bool) bool
```

`ReadRaw` (`vault.go:280-318`) gains one guard: if the requested slug maps to an ignored path, return `os.ErrNotExist`. This closes the `/api/notes/{slug}/raw` loophole.

### 5.6 Wiring into the watcher (`internal/watcher/watcher.go`)

Two changes:

1. **`New`** — when walking to add directories, skip ignored directories so fsnotify never watches them:

   ```go
   if info.IsDir() {
       if vw.vault.IsIgnored(path, true) {  // NEW
           return filepath.SkipDir
       }
       return fw.Add(path)
   }
   ```

2. **`loop`** — keep the `.md` suffix filter, and add a defensive `IsIgnored` check before `apply`, so a path moved into an ignored tree mid-run is not re-added:

   ```go
   if !strings.HasSuffix(strings.ToLower(event.Name), ".md") { continue }
   if vw.vault.IsIgnored(event.Name, false) { continue }   // NEW
   pending[event.Name] = event.Op
   ```

3. **The ignore file itself.** `.vault-ignore` is not `.md`, so the existing loop already ignores its events. Because changing the ignore file changes the *set* of published notes (a vault-wide concern), the correct response is a **full reload**, not a single-note reload. The design treats this as out of scope for the watcher's hot path: in `--watch` (local dev) the developer restarts the server; in git-sync deployments the `/api/admin/reload` endpoint already does a full reload (see §5.8). This is captured as an open question / future work to optionally watch `.vault-ignore` and trigger `state.Reload()`.

### 5.7 Wiring into the asset handler (`internal/server/server.go`)

`assetHandler` already holds `state` (the `RuntimeState`) and opens via `os.OpenRoot(state.ResolvedRoot())`. Add an ignore check using the active snapshot's vault:

```go
v, _ := state.Snapshot()
rel := strings.TrimPrefix(r.URL.Path, "/vault-assets/")
if !validAssetPath(rel) || v.IsIgnored(filepath.Join(v.Root(), rel), false) {
    http.NotFound(w, r)
    return
}
```

(Construct the absolute path for `IsIgnored`, or extend the `Ignore` API with a `Match(relPath, isDir)` form keyed on the vault-relative path — preferred, since the handler already has `rel`.)

### 5.8 CLI surface (`cmd/.../serve/serve.go`)

The default is **presence-based**: `<root>/.vault-ignore` is read automatically. Add one optional flag so operators can point elsewhere or disable:

```
--vault-ignore string   Path to a vault-ignore file. Defaults to <vault>/.vault-ignore.
                        Pass an empty string to disable ignore processing entirely.
```

`Settings` gains `VaultIgnore string`; `RunIntoGlazeProcessor` passes it into `server.Config`, which threads it to `loadSnapshot` → `vault.NewWithOptions(root, ignorePath)`. To keep `vault.New(root)` simple for tests, prefer a new constructor:

```go
func NewWithOptions(rootDir, ignorePath string) (*Vault, error)
```

where `ignorePath == ""` means "use default `<root>/.vault-ignore`" and a sentinel (e.g. `-`) could mean "disabled". Keep `New(root)` as a thin wrapper that calls `NewWithOptions(root, "")` so existing callers and tests are unchanged.

### 5.9 Reload semantics

No new reload code is required. `RuntimeState.Reload()` → `loadSnapshot()` → `vault.New(resolvedRoot)` rebuilds everything from disk, so a freshly edited `.vault-ignore` is read on the next `/api/admin/reload`. This is the git-sync path's contract already.

---

## 6. Decision records

### Decision: gitignore-compatible subset instead of a full gitignore engine

- **Context:** Users already know `.gitignore`. We need exclusion of directories and files, with negation. Full gitignore (nested files, `**`, char classes) is complex and subtle.
- **Options considered:** (a) full gitignore via a library (`github.com/sabhiram/go-gitignore` or `go-git`); (b) a documented minimal subset implemented by hand using `path.Match`.
- **Decision:** (b) minimal documented subset, hand-rolled in `internal/ignore`.
- **Rationale:** The project's existing style implements parsers by hand (`internal/parser/parser.go`), values zero new dependencies, and the common cases (`dir/`, `/anchored`, `*.glob`, `!negate`) cover >95% of real needs. A small, tested matcher is easy to audit. The `Ignore` API is shaped so a library can be swapped in later without changing call sites.
- **Consequences:** Enables the feature with no new deps; makes semantics explicit; requires documenting exactly what is *not* supported (nested files, `**`). Must be validated with a table of pattern cases (§8).
- **Status:** proposed

### Decision: filter once at the walk, not at each consumer

- **Context:** Notes flow into the file tree, search index, backlinks, API, and assets.
- **Options considered:** (a) check ignore at each consumer; (b) filter in `LoadAll` so consumers see only published notes; plus a targeted check at the asset/raw endpoints (which read files off-disk, bypassing `notes`).
- **Decision:** (b) filter in `LoadAll`, plus explicit checks in `ReadRaw` and the asset handler.
- **Rationale:** Consumers already derive from `v.notes`; one filter covers them all. Asset serving and raw reads bypass `notes`, so they need their own (cheap) check.
- **Consequences:** Smaller, less error-prone change. Requires that `ReloadNote`/`RemoveNote` and the watcher also consult ignore so a file cannot sneak back in.
- **Status:** proposed

### Decision: ignore errors are non-fatal

- **Context:** A malformed `.vault-ignore` should not take the whole site down.
- **Options considered:** (a) fail `vault.New` on a parse error; (b) log a warning and treat as "ignore nothing".
- **Decision:** (b) log and proceed with an empty `Ignore`.
- **Rationale:** Publishing content is more important than enforcing ignore correctness; operators can fix the file and reload.
- **Consequences:** A broken ignore file silently publishes everything — must be loud in logs and covered by a test that asserts the warning is emitted.
- **Status:** proposed

### Decision: `.vault-ignore` changes are not hot-reloaded per-note

- **Context:** The watcher reloads individual `.md` files; the ignore file is vault-wide.
- **Options considered:** (a) watch `.vault-ignore` and trigger `state.Reload()`; (b) treat ignore changes as requiring a server restart (local) or the existing `/api/admin/reload` (git-sync).
- **Decision:** (b) for this ticket; (a) as explicit future work.
- **Rationale:** Keeps the watcher's per-file model simple; the reload endpoint already exists for deployments. Hot-reloading ignore correctly requires a full rebuild anyway.
- **Consequences:** A local developer who edits `.vault-ignore` must restart (or hit reload). Documented as a known limitation.
- **Status:** proposed

---

## 7. Key flows (pseudocode)

### 7.1 Startup → first serve

```
serve --vault ./my-vault --port 8080
  → server.Run(Config{VaultDir, ...})
      → NewRuntimeStateWithOptions(VaultDir, opts)
          → loadSnapshot(configuredRoot, "")
              → absRoot = abs(VaultDir); resolvedRoot = EvalSymlinks(absRoot)
              → vault.NewWithOptions(resolvedRoot, "")        # reads <root>/.vault-ignore
                  → ig = ignore.Load(root)                   # nil error if file absent
                  → v.ignore = ig
                  → v.LoadAll()
                      walk(root):
                        dir  "."-prefixed?  -> SkipDir
                        dir  ig.Match(rel,true)? -> SkipDir    # NEW
                        file not .md?       -> skip
                        file ig.Match(rel,false)? -> skip      # NEW
                        loadNote -> v.notes[slug]
                  → buildWikiLinkIndex / buildBacklinks / rebuildHTML
              → search.New(v) | search.NewPersistent(v, dir)
          → snapshot{Vault:v, Search:si, ...}
      → watcher.New(v, WithSearchIndex(si))                     # skips ignored dirs
      → register routes (api, /vault-assets, /, /api/admin/reload)
      → srv.ListenAndServe()
```

### 7.2 A `.md` file is edited under a watched (non-ignored) tree

```
fsnotify Write event for /vault/Notes/A.md
  → loop: IsSuffix(".md") ✓ ; vault.IsIgnored(abs,false) ✗  -> pending[/vault/Notes/A.md]=Write
  → ticker (500ms) → apply(/vault/Notes/A.md, Write)
      → vault.ReloadNote(abs)
          → loadNote -> parse ; v.notes[slug]=note
          → rebuild indexes
      → search.Index(doc)
```

### 7.3 A `.md` file is created under an ignored tree

```
fsnotify Create for /vault/ttmp/_guidelines/foo.md
  → loop: IsSuffix(".md") ✓ ; vault.IsIgnored(abs,false) ✓  -> continue   # NEW: dropped
```

### 7.4 Git-sync flips the checkout and calls reload

```
POST /api/admin/reload  (bearer token, or loopback)
  → stopWatcherBeforeReload()       # close fsnotify so old dirs aren't watched
  → state.Reload()
      → loadSnapshot(configuredRoot, "")   # re-reads the NEW .vault-ignore from disk
      → swap snapshot; close old after 30s
  → 204 No Content
```

### 7.5 Asset request for an ignored image

```
GET /vault-assets/Secrets/secret.png
  → assetHandler
      rel = "Secrets/secret.png"
      validAssetPath(rel) ✓
      snapshot.Vault.Ignore.Match("Secrets/secret.png", false) ✓  # NEW
      → 404 Not Found
```

---

## 8. Implementation plan (phased, file-level)

### Phase 1 — The `internal/ignore` package (no callers yet)

1. Create `internal/ignore/ignore.go` with the `Ignore` type, `Load`, `LoadFromPath`, `Match`, `MatchAbs`, `Empty`, and the unexported `pattern` + `matches`/`matchAnchored` helpers. Use `path.Match` for globs.
2. Create `internal/ignore/ignore_test.go` with a table-driven test covering:
   - absent file → `Empty()` true, `Match` false for everything;
   - blank lines and `#` comments ignored;
   - `dir/` matches dir but not file of the same name;
   - `/anchored` vs unanchored `*.draft.md`;
   - `!negation` last-wins;
   - `\!` escaping;
   - subtree exclusion (`ttmp/_guidelines` excludes a nested file).
3. `go test ./internal/ignore/...` must pass before touching any caller.

### Phase 2 — Wire ignore into `vault`

1. `internal/vault/vault.go`: add `ignore *ignore.Ignore` field; implement `NewWithOptions(root, ignorePath)` and reduce `New(root)` to a wrapper.
2. In `LoadAll`, add the two `IsIgnored`/`SkipDir` checks (§5.5).
3. Add `IsIgnored(absPath, isDir)` accessor (nil-safe).
4. In `ReloadNote`, return early (`return nil, errIgnored` or `nil, nil`) when the path is ignored, so a moved-in file is not re-added. In `RemoveNote`, the existing delete is already correct.
5. In `ReadRaw`, reject ignored slugs with `os.ErrNotExist`.
6. Extend `internal/vault/vault_test.go`: add `TestLoadAllRespectsVaultIgnore` (write a `.vault-ignore`, assert excluded dir's notes are absent and the file tree omits it) and `TestReloadNoteIgnoresMovedInFile`.

### Phase 3 — Wire ignore into `watcher`

1. `internal/watcher/watcher.go`: in `New`'s walk, `SkipDir` on ignored dirs; in `loop`, drop events for ignored `.md` paths before enqueueing.
2. Extend `internal/watcher/watcher_test.go`: assert that a write under an ignored dir does not surface in the search index (mirror the existing `TestApplyKeepsSearchIndexInSync` shape, but for an ignored path that must NOT update).

### Phase 4 — Wire ignore into asset + raw serving

1. `internal/server/server.go`: in `assetHandler`, consult the active snapshot's ignore before serving.
2. (Covered transitively) `getNoteRaw` already goes through `v.ReadRaw`, now ignore-aware.
3. Add `internal/server` test: request an ignored asset → expect 404.

### Phase 5 — CLI flag + end-to-end

1. `cmd/retro-obsidian-publish/commands/serve/serve.go`: add the `--vault-ignore` field; thread it through `Settings` → `server.Config` → `loadSnapshot` → `vault.NewWithOptions`.
2. `internal/server/runtime.go`: `RuntimeOptions` gains `IgnorePath string`; `loadSnapshot` passes it to `vault.NewWithOptions`.
3. Manual smoke test with `vault-example/` plus a `ttmp/_guidelines/` dir: confirm notes list, tree, search, and asset requests all exclude the ignored tree.

### Phase 6 — Docs

1. Add a "Excluding paths" section to `README.md` with the example from §5.2.
2. Update `AGENT.md` if the flag naming affects agent workflows (it should not).

---

## 9. Test strategy

| Layer | Test | Asserts |
|-------|------|---------|
| `internal/ignore` | table-driven `TestMatch` | every supported subset rule + escaping + subtree |
| `internal/ignore` | `TestLoadAbsentFile` | absent file → empty, match-nothing |
| `internal/vault` | `TestLoadAllRespectsVaultIgnore` | excluded dir's notes absent; file tree omits it; search docs omit it |
| `internal/vault` | `TestReloadNoteIgnoresMovedInFile` | renaming a file into an ignored dir does not re-add it |
| `internal/vault` | `TestReadRawRejectsIgnoredSlug` | `/raw` for ignored slug → `os.ErrNotExist` |
| `internal/watcher` | `TestApplySkipsIgnoredPath` | write under ignored dir → search index unchanged |
| `internal/server` | `TestAssetHandler404OnIgnored` | `GET /vault-assets/<ignored>` → 404 |
| manual | serve `vault-example` + `ttmp/_guidelines/` | tree, search, tags, assets all exclude it |

Run everything with:

```bash
cd /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault
go test ./internal/... -count=1
go build ./...
golangci-lint run -v   # or: make lint
```

---

## 10. Risks, alternatives, and open questions

### Risks

- **Matcher correctness.** A wrong matcher could either over-exclude (hide real content) or under-exclude (leak private content). Mitigation: table-driven tests; the "under-exclude" case is the dangerous one and is explicitly tested.
- **Performance.** `Match` is called once per file during `LoadAll` (fine) and per fsnotify event (very fine). Patterns are few; no caching needed for v1.
- **Cross-platform paths.** Vault paths are normalized to slash (`filepath.ToSlash`) before matching, matching the slug convention. `MatchAbs` must convert with `filepath.Rel` + `ToSlash`.
- **Symlinks.** `loadSnapshot` resolves the vault root symlink, but ignores inside the tree are evaluated against the resolved root. Symlinked subtrees inside the vault are an edge case; document as unsupported (like hidden-dir handling today).

### Alternatives considered

- **Frontmatter `publish: false`.** Good for per-note control, but does not help exclude whole directories of scaffolding (`ttmp/_guidelines`). Complementary, not a replacement. Future work.
- **Convention-based exclusion (e.g. always ignore `_`-prefixed dirs).** Surprising and inflexible; the user should control this explicitly.
- **A full gitignore library.** Rejected for v1 (see decision record); the API is shaped to allow it later.

### Open questions

1. Should `.vault-ignore` itself be watchable to auto-trigger `state.Reload()` in `--watch` mode? (Proposed: yes, as a fast-follow, not in this ticket.)
2. Should `Match` support `**`? (Proposed: no for v1; revisit if real vaults need deep globs.)
3. Should there be a `/api/config` field advertising the active ignore file path for debugging? (Proposed: yes, small addition.)

---

## 11. References

Key files (resolve against `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/`):

- `internal/vault/vault.go` — `LoadAll` (lines 82-114), `loadNote` (118-167), `ReloadNote` (226-241), `RemoveNote` (243-252), `ReadRaw` (280-318), `FileTree` (288-326), `pathToSlug` (331-336).
- `internal/watcher/watcher.go` — `New` walk (53-71), `loop` (78-112), `apply` (114-141).
- `internal/server/runtime.go` — `loadSnapshot` (92-130), `Reload` (80-99).
- `internal/server/server.go` — `Run` (54-170), `assetHandler` (211-243), `validAssetPath` (247-261), reload route (101-106).
- `internal/api/api.go` — route table (62-70), read paths (72-180).
- `internal/parser/parser.go` — `Parse` (45-95), `Slugify` (end).
- `internal/search/search.go` — `New` (33-48), derives from `v.ForEachSearchDocument`.
- `cmd/retro-obsidian-publish/commands/serve/serve.go` — `Settings` struct, `RunIntoGlazeProcessor`.

External standards:

- gitignore specification: https://git-scm.com/docs/gitignore (the subset implemented is documented in §5.2).
- Go `path.Match`: https://pkg.go.dev/path#Match (the glob primitive reused).
- `os.OpenRoot` (Go 1.24+): used for safe vault-root file access in `ReadRaw` and `assetHandler`.

Related tickets:

- `RETRO-MEMORY-012` — prod OOM and reload hardening; shares the `loadSnapshot`/reload path this design depends on.
