---
Title: 'Per-note publish flag and vault config: analysis and intern implementation guide'
Ticket: RETRO-PUBLISH-009
Status: active
Topics:
    - obsidian-vault
    - config
    - parser
    - retro-obsidian-publish
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: CLI flags; primary Phase 3 edit target, add --config and load vaultconfig
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/ignore/ignore.go
      Note: Legacy .vault-ignore matcher; unchanged, read for parity (IgnoreFile l.25, Match l.149, HasNegations l.187)
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/parser/parser.go
      Note: Markdown parser; unchanged, frontmatter already a generic map (Parse l.56, ParsedNote l.30, normalizeFrontmatter l.479)
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/server/runtime.go
      Note: Snapshot lifecycle; thread VaultConfig into loadSnapshot -> vault.New
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/server/server.go
      Note: server.Config struct and Run; add VaultConfig field
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vault/vault.go
      Note: Core loader; primary Phase 2 edit target (New l.82, LoadAll l.102, loadNote l.152, ReloadNote l.430, ShouldPruneDir l.455)
    - Path: /home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/watcher/watcher.go
      Note: fsnotify watcher; switch IsIgnored call to IsExcluded
ExternalSources: []
Summary: 'Adds per-note `publish: false` frontmatter gating and a YAML vault config file (`.publish/config.yaml`) with gitignore-style folder blacklist patterns. Intern-facing design and implementation guide.'
LastUpdated: 2026-07-26T17:00:00-04:00
WhatFor: Designing and implementing per-note opt-out publishing and a config-driven folder blacklist with full gitignore semantics.
WhenToUse: Read this before touching the vault loader, parser, ignore matcher, watcher, or serve command for the publish/config feature.
---








# Per-note `publish` flag and vault config: analysis and intern implementation guide

> Audience: a new engineer (intern) who has never seen this codebase. This
> document assumes you know Go and basic Markdown/YAML, but nothing about
> Obsidian, this project, or its internal architecture. Read it top to bottom;
> later sections build on earlier ones.

## 1. Executive summary

Publish Vault (binary `retro-obsidian-publish`, Go module
`github.com/go-go-golems/publish-vault`) turns a folder of Obsidian Markdown
files into a self-hosted website. Today, a note is published **unless** the
entire file path is excluded by a `.vault-ignore` file at the vault root.

This ticket adds two complementary ways to keep content *out* of the published
site:

1. **Per-note `publish: false` frontmatter** — a note author can put
   `publish: false` in the YAML frontmatter of a single Markdown file to keep
   that file private while leaving the rest of its folder public. This is
   Obsidian-native (the file is a normal note), granular (per file), and does
   not require any separate ignore file.

2. **A vault config file with a gitignore-style folder blacklist** — a single
   `.publish/config.yaml` (or YAML passed via `--config`) can list glob
   patterns like `Secrets/**` or `Drafts/` that are excluded from publication,
   using **full gitignore semantics** (`**`, nested directories, negations)
   provided by a well-tested Go library instead of the hand-rolled matcher used
   by `.vault-ignore` today.

The two mechanisms compose: a note is published only when it is **not** excluded
by the config blacklist **and** does not carry `publish: false`. The existing
`.vault-ignore` file keeps working unchanged for backward compatibility, but the
config-file blacklist is the recommended, more capable replacement going forward.

This document is the full analysis + design + implementation guide. It maps the
relevant subsystems with file/line evidence, explains every integration point
the change must touch, gives pseudocode for the runtime flows, and lays out a
phased implementation plan an intern can follow step by step.

## 2. Problem statement and scope

### 2.1 What we cannot do today

- **A draft note inside an otherwise-public folder cannot be hidden by a single
  flag.** The only per-file mechanism today is `.vault-ignore`, which lives at
  the vault root and is path-based, not metadata-based. Hiding one draft means
  adding a path pattern there, which couples publishing policy to filesystem
  layout and is easy to forget when moving files.
- **Folder blacklisting is limited.** The current `.vault-ignore` matcher
  (`internal/ignore/ignore.go`) deliberately implements only a *documented
  subset* of gitignore. It explicitly does **not** support `**` (match across
  directory boundaries) or nested `.vault-ignore` files. See the package doc
  comment at the top of `internal/ignore/ignore.go`. This means common patterns
  like `**/node_modules/` or `Private/**/cache/` cannot be expressed.
- **There is no general config file.** The only "config" the vault understands
  is the ignore file. All other server behavior (port, watch, vault name,
  favicon, search index path, widget pages dir) is configured purely through
  CLI flags / env vars in `cmd/.../serve/serve.go`. There is no place to put
  vault-scoped settings that travel *with the vault* (e.g. in the git repo that
  gets pulled onto the server).

### 2.2 What we want

- A note author can write `publish: false` in frontmatter and the note is
  excluded from the API, file tree, search index, backlinks, raw-source
  endpoint, and (when applicable) the agent-markdown / SSR surfaces — everywhere
  `.vault-ignore` already excludes paths.
- An operator can create a `.publish/config.yaml` in the vault (or pass
  `--config <path>`) listing gitignore-style patterns. These use **full**
  gitignore semantics via a library, so `**`, nested negations, etc. all work.
- Both mechanisms integrate cleanly with the existing runtime: the file
  watcher's incremental reload, the admin reload endpoint's atomic snapshot
  swap, and the in-memory search index all stay consistent.

### 2.3 Out of scope (do not build these)

- A `publish: true` that *force-publishes* a note that is otherwise ignored.
  `publish` is only ever a *negative* opt-out. (Rationale in §6, Decision A.)
- A web UI for editing the config or the publish flag. Editing happens in the
  Markdown / YAML files, the normal Obsidian way.
- Migrating or deleting the existing `.vault-ignore` file. It keeps working;
  the config blacklist is additive and preferred for new use.
- Changing how frontmatter is parsed or normalized in general. We reuse the
  existing `goldmark-meta` pipeline (`internal/parser/parser.go`).

## 3. Current-state architecture (read this first)

This section maps the parts of the system the feature touches. Every claim is
anchored to a concrete file and (where useful) a line number so you can open the
code alongside this doc.

### 3.1 The big picture

Publish Vault is a single Go process with two phases (from `README.md`,
"How the publishing pipeline works"):

```text
LOAD TIME (vault.New / LoadAll)                REQUEST TIME (HTTP handlers)
─────────────────────────────────                ─────────────────────────
Markdown files ─┐                                Browser ─> React SPA ─┐
                ├─> parser.Parse ─> Note ─┐                            │
.vault-ignore ──┘   (goldmark + meta)      ├─> Vault snapshot ─> /api/*│
                                          ─┤   (vault + search index)   │
wiki-link suffix index ────────────────────┤                            │
backlinks ─────────────────────────────────┤                            │
rendered HTML (links resolved) ─────────────┤                            │
Bleve search index ─────────────────────────┘                            │
                                                                         ▼
                                                              /vault-assets/ , /note/... , /
```

- **Load time** is expensive (parse everything, build indexes) and runs on
  startup and on every reload.
- **Request time** only reads the prepared in-memory snapshot, so it is fast.

The implication for us: **any publish/config decision must be made at load
time** (and on the watcher's incremental reload), never per request. We add
gating to the loader, not the HTTP handlers.

### 3.2 The ignore package — `internal/ignore/ignore.go`

This is the existing path-exclusion engine. It is the closest existing analog
to what we are building, so study it.

- It parses a `.vault-ignore` file (`IgnoreFile = ".vault-ignore"`, line 25).
- Each line compiles to a `pattern` struct (line 28): `negate`, `dirOnly`,
  `anchored`, `pat` (the glob), `raw`.
- `Match(relPath, isDir)` (line 149) evaluates patterns in file order with
  **last-match-wins**; a later `!` negation overrides an earlier exclusion.
- It does **not** implement strict gitignore: negation cannot re-include a file
  under an excluded *directory* the way real git does — instead it treats `!` as
  a simple last-match override. The package doc comment says so explicitly.
- `HasNegations()` (line 187) exists because of a subtle correctness rule: when
  negations are present, the vault walk must **descend into** ignored
  directories instead of pruning them, otherwise a `!` that re-includes a file
  beneath an ignored dir would silently drop that file. `ShouldPruneDir` (in
  `pkg/vault/vault.go`) consults this.

**Why this matters for our design:** the config-file blacklist is meant to
*replace* the limitations of this matcher for new configs. We will introduce a
library-backed gitignore matcher with full `**` support, used by the config
file, and keep the hand-rolled `Ignore` for `.vault-ignore` (backward compat).
Both feed the same "is this path excluded?" decision in the vault loader.

### 3.3 The parser — `internal/parser/parser.go`

Parses one Markdown file into a `ParsedNote` (line 30):

```go
type ParsedNote struct {
    Frontmatter map[string]interface{}
    HTML        string
    WikiLinks   []WikiLink
    Tags        []string
    Title       string
    Excerpt     string
}
```

- `Parse(src []byte)` (line 56) runs goldmark with the `meta` extension and
  returns frontmatter as a normalized `map[string]interface{}` (via
  `normalizeFrontmatter`, line 479, which flattens `map[interface{}]interface{}`
  from YAML into string-keyed maps).
- `extractTitle` (line 515) reads `frontmatter["title"]` then falls back to the
  first H1. `extractTags` (line 530) reads `frontmatter["tags"]`.

**Why this matters:** frontmatter is already parsed and available as a generic
map. The `publish` flag is just another frontmatter key. We will read
`frontmatter["publish"]` (case-insensitive) as a boolean. The parser itself
needs **no change** for the flag — the read happens in the vault layer where the
`ParsedNote` is consumed. (Decision B in §6 explains why we do not add a typed
field to `ParsedNote`.)

### 3.4 The vault — `pkg/vault/vault.go`

This is the core. It holds notes, the ignore rules, and all the indexes.

```go
// pkg/vault/vault.go (selected, line-anchored)
type Vault struct {                                  // line 68
    mu            sync.RWMutex
    notes         map[string]*Note
    wikiLinkIndex map[string]string
    assetIndex    map[string]string
    root          string
    ignore        *ignore.Ignore                   // line 74 — .vault-ignore
}
```

Key methods and their roles:

- `New(rootDir)` (line 82): loads `.vault-ignore` via `ignore.Load(root)`, then
  calls `LoadAll`. A missing/empty ignore file → publish everything; a malformed
  one → log and ignore nothing (never takes the site down).
- `LoadAll()` (line 102): walks `root`. For each directory it prunes ignored
  dirs via `ShouldPruneDir`. For each `.md` file it calls `isIgnored`; if not
  ignored it calls `loadNote` and stores the result keyed by slug. Non-`.md`
  files that are not ignored become assets (`indexAsset`).
- `loadNote(absPath, info)` (line 152): reads the file, calls `parser.Parse`,
  builds a `Note` with slug, title, frontmatter, tags, wiki links, etc.
- `isIgnored(absPath, isDir)` (internal, near line 470): lock-free, delegates
  to `v.ignore.MatchAbs`. `IsIgnored` (exported) wraps it.
- `ShouldPruneDir(absPath)` (line 455): true only when ignore is non-empty,
  has no negations, and the dir matches — so pruning is safe.
- `ReloadNote(absPath)` (line 430): used by the watcher. Re-parses one file,
  updates indexes, returns the note. Returns `ErrIgnored` (line 24) if the path
  is now excluded, so the watcher treats it as a no-op.
- `RemoveNote(absPath)`: drops a note (used on delete/rename events).
- `RefreshAssetIndex()` (line 278): rebuilds the asset index from disk without
  the vault lock, then swaps atomically.
- `FileTree()` (line ~590): builds the sidebar tree **only from `v.notes`**.
- `ReadRaw(relPath)` (line ~640): serves raw Markdown. It checks `isIgnored`
  again so the ignore rules cannot be bypassed via this endpoint.

**Why this matters:** the loader is the single choke point. If we gate
publishing here — both the initial `LoadAll` and the incremental
`ReloadNote` — then *every downstream consumer* (API, file tree, search,
backlinks, raw endpoint, watcher) automatically respects the rules, because
they all read from `v.notes`. This is the central insight of the design.

### 3.5 The file watcher — `pkg/watcher/watcher.go`

Watches the vault with `fsnotify` and debounces events. On a `.md` change it
calls `vault.ReloadNote` (which re-checks `isIgnored` and will return
`ErrIgnored`). It prunes ignored dirs when adding watch targets using
`v.ShouldPruneDir`. Non-`.md` changes refresh the asset index.

**Why this matters:** the watcher already re-consults ignore rules on every
event. For `publish: false`, the watcher's `ReloadNote` path will re-parse the
file and (with our change) see the flag, so a note toggled to `publish: false`
disappears from the snapshot on the next debounced reload — no server restart
needed. For the config blacklist, see §7 (config changes require a reload, not
incremental updates — same as `.vault-ignore` today).

### 3.6 The server, runtime, and config plumbing

- `pkg/server/server.go` defines `server.Config` (a struct, not a file) holding
  CLI-derived runtime settings (VaultDir, Port, Watch, etc.) and calls
  `NewRuntimeStateWithOptions` to build the snapshot.
- `pkg/server/runtime.go` manages atomic snapshots. `loadSnapshot` resolves
  symlinks (for git-sync deployments), builds the `vault.Vault`, then the search
  index, and returns an immutable `Snapshot`. `Reload()` builds a new snapshot
  and swaps it in; on failure the old snapshot stays.
- `cmd/.../serve/serve.go` wires Glazed flags → `Settings` struct →
  `appserver.Config`. It uses `cli.CobraCommandDefaultMiddlewares`, which (per
  `glazed/pkg/cli/cobra-parser.go` line 32) builds a minimal chain of **cobra
  flags + args + defaults only** — it does **not** load config files. (Glazed
  *can* load config files via `--config-file`, but this command does not opt
  into that middleware.)

**Why this matters:** there is no existing config-file plumbing for the serve
command. We will add an explicit `--config` flag and load the YAML ourselves in
the serve command, before constructing `appserver.Config`. We deliberately do
**not** switch on Glazed's generic config-file middleware: we want a typed,
vault-scoped config struct, not a generic key/value bag. (Decision C in §6.)

### 3.7 The `.publish` directory convention

The codebase already uses a `<vault>/.publish/` directory for widget page
scripts: in `serve.go` the default `--pages-dir` is
`filepath.Join(settings.Vault, ".publish", "pages")`. We follow this convention:
our config file lives at `<vault>/.publish/config.yaml`. This keeps
publish-specific config grouped and avoids cluttering the vault root.

## 4. Gap analysis

| Concern | Current state | Required state | Gap |
|---|---|---|---|
| Per-note opt-out | Not supported; only path-based `.vault-ignore` | `publish: false` frontmatter hides one note | New gating in `loadNote` + `ReloadNote` |
| Full gitignore globs | `internal/ignore` subset, no `**`, no nested files | Config file with `**` and negations | New library-backed matcher in a config package |
| Vault-scoped config file | None; all config via CLI flags/env | `.publish/config.yaml` travels with vault | New config loader + `--config` flag |
| Watcher consistency | Re-checks `.vault-ignore` per event | Must also re-check `publish` flag (free via `ReloadNote`) and config blacklist (reload-only) | Covered by reusing existing flows |
| Reload semantics | `.vault-ignore` changes need reload | Config changes need reload; `publish` flag is incremental | Document; no new reload mechanism needed |

## 5. Proposed architecture and APIs

### 5.1 Component overview

```text
                         ┌───────────────────────────────────────────────┐
                         │  cmd/.../serve/serve.go                        │
                         │  --config <path> (default <vault>/.publish/    │
                         │                   config.yaml)                  │
                         │     │                                          │
                         │     ▼                                          │
                         │  vaultconfig.Load(path) -> *Config  (NEW)      │
                         │     │  + appserver.Config.VaultConfig          │
                         ▼     ▼                                          │
   ┌───────────────────────────────────────────────────────────────────┐ │
   │ pkg/server/runtime.go  loadSnapshot()                              │ │
   │   vault.New(resolvedRoot, vault.WithConfig(cfg))  (NEW ctor opt)   │ │
   └───────────────────────────────────────────────────────────────────┘ │
                         │                                                │
                         ▼                                                │
   ┌───────────────────────────────────────────────────────────────────┐ │
   │ pkg/vault/vault.go                                                  │ │
   │   ignore    *ignore.Ignore      <- .vault-ignore (legacy)          │ │
   │   blacklist *gitignore.Ignore   <- config file (NEW)               │ │
   │   loadNote(): isExcluded(path) || note.publish==false -> skip      │ │
   └───────────────────────────────────────────────────────────────────┘ │
                         │                                                │
        ┌────────────────┼─────────────────┬───────────────────┐          │
        ▼                ▼                 ▼                   ▼          │
   pkg/api/api.go   pkg/search      pkg/watcher        pkg/server         │
   (reads notes)    (indexed at      (ReloadNote re-     (raw endpoint      │
                    load time)       checks rules)       re-checks rules)   │
                                                                            │
   pkg/vaultconfig/ (NEW package)  ── internal/ignore (unchanged) ────────┘
```

### 5.2 New package `pkg/vaultconfig`

A small, focused package. It owns the config file schema and loading, and the
gitignore-style matcher used by the blacklist. (Note: it imports a library; the
internal `ignore` package stays untouched for `.vault-ignore`.)

```go
// Package vaultconfig loads the publish-vault site configuration file
// (.publish/config.yaml) and answers gitignore-style path-exclusion queries
// derived from its `ignore` list.
package vaultconfig

// DefaultConfigPath is the conventional config location inside a vault.
const DefaultConfigPath = ".publish/config.yaml"

// Config is the deserialized vault configuration. Unknown keys are ignored
// so future fields can be added without breaking older binaries.
type Config struct {
    // Ignore is a list of gitignore-style patterns applied to vault-relative
    // paths. Patterns use FULL gitignore semantics (**, negations, nested
    // matches) via github.com/sabhiram/go-gitignore. An empty list means
    // "ignore nothing" (everything is eligible, subject to per-note publish).
    Ignore []string `yaml:"ignore"`
}

// Load reads a config file. A missing file returns an empty Config and nil
// error so callers can treat "no config" and "empty config" identically,
// mirroring ignore.Load for .vault-ignore. A malformed file returns an empty
// Config and the error; callers should log and proceed.
func Load(path string) (*Config, error)

// Matcher answers "is this vault-relative path excluded by the config
// blacklist?" It wraps the gitignore library. A nil/empty matcher excludes
// nothing. isDir indicates whether the path is a directory.
type Matcher struct{ /* unexported: *gitignore.GitIgnore */ }

// NewMatcher compiles the config's Ignore patterns. Returns a no-op matcher
// when cfg is nil or has no Ignore patterns.
func NewMatcher(cfg *Config) (*Matcher, error)

// Match reports whether relPath (vault-relative, slash-separated) is excluded.
func (m *Matcher) Match(relPath string, isDir bool) bool

// Empty reports whether the matcher would exclude nothing.
func (m *Matcher) Empty() bool
```

### 5.3 Vault API changes (`pkg/vault/vault.go`)

Add a functional option to `New` so the config is injected without breaking
existing callers (the library usage in `README.md` §"Using publish-vault as a
library" calls `vault.New(root)` with one arg).

```go
// Option configures a Vault.
type Option func(*Vault)

// WithConfig attaches a vault config (blacklist matcher) to the vault. When
// set, the loader excludes paths matched by the config blacklist in addition
// to .vault-ignore. The matcher is treated as immutable after construction
// (same lifecycle as ignore), so it is safe to read concurrently without a
// lock.
func WithConfig(cfg *vaultconfig.Config) Option {
    return func(v *Vault) {
        m, err := vaultconfig.NewMatcher(cfg)
        if err != nil {
            log.Printf("vault: warning compiling config blacklist: %v; ignoring no paths", err)
            return
        }
        v.configMatcher = m
    }
}

func New(rootDir string, opts ...Option) (*Vault, error) {
    // ... existing ignore.Load ...
    v := &Vault{ /* ... */ }
    for _, opt := range opts {
        opt(v)
    }
    if err := v.LoadAll(); err != nil {
        return nil, err
    }
    return v, nil
}
```

Add a new exported field/method for "is this note publishable?" and unify the
exclusion check:

```go
// IsExcluded reports whether absPath is excluded by EITHER .vault-ignore OR
// the config blacklist. It replaces the old isIgnored-only decision at every
// call site that gates publication. isDir affects directory-only patterns.
func (v *Vault) IsExcluded(absPath string, isDir bool) bool {
    if v.isIgnored(absPath, isDir) {       // .vault-ignore (legacy)
        return true
    }
    if v.configMatcher != nil && !v.configMatcher.Empty() {
        rel, err := filepath.Rel(v.root, absPath)
        if err == nil {
            return v.configMatcher.Match(filepath.ToSlash(rel), isDir)
        }
    }
    return false
}
```

Add per-note `publish` flag handling. The `Note` struct already carries
`Frontmatter`; we add a derived boolean for fast filtering:

```go
type Note struct {
    // ... existing fields ...
    Publish bool `json:"-"` // false => excluded from publication
}
```

The decision is made in `loadNote`:

```go
// inside loadNote, after parsing frontmatter:
publish := true // default: publish
if v, ok := frontmatterBool(frontmatter, "publish"); ok {
    publish = v
}
```

Where `frontmatterBool` is a small helper that:
- does a case-insensitive lookup over the frontmatter map keys (so `publish`,
  `Publish`, `PUBLISH` all work);
- accepts YAML booleans (`true`/`false`) and the strings `"true"`/`"false"`
  (goldmark-meta may surface scalars as strings depending on YAML quoting);
- returns `(false, false)` ("value is false, and it was present") and
  `(true, true)` ("value is true, and it was present"). Absent key →
  `(defaultValue=true, false)`.

### 5.4 The unified exclusion flow (pseudocode)

This is the heart of the design. Every place that decides "should this be
published?" routes through one decision function.

```text
loadNote(absPath) -> Note:
    src = readFile(absPath)
    parsed = parser.Parse(src)
    note = buildNote(parsed, ...)
    note.Publish = frontmatterBool(parsed.Frontmatter, "publish", default=true)
    return note

LoadAll():
    for each path in walk(root):
        if isDir:
            if ShouldPruneDir(path): skipDir
            else: continue
        if isExcluded(path, isDir=false):        # config blacklist OR .vault-ignore
            continue
        if not .md:
            indexAsset(path); continue
        note = loadNote(path)
        if not note.Publish:                     # per-note frontmatter
            continue                             # parsed but not stored
        v.notes[note.Slug] = note
    buildWikiLinkIndex(); buildBacklinks(); rebuildHTML()

# The watcher's incremental path (no full reload):
ReloadNote(absPath) -> (Note, error):
    if isExcluded(absPath, false): return ErrExcluded   # NEW: config blacklist too
    note = loadNote(absPath)
    if not note.Publish:
        RemoveNote(absPath)            # was published, now hidden
        return nil, ErrExcluded        # signal watcher to drop from search index
    store note; rebuild affected indexes
    return note, nil
```

**Key invariant:** `v.notes` only ever contains publishable notes. Because
*every* consumer (API list, file tree, search index build, backlink graph, raw
endpoint) reads from `v.notes` or from a search index built *from* `v.notes`,
hiding a note at load time hides it everywhere automatically.

### 5.5 Config loading in the serve command

```text
serve.go RunIntoGlazeProcessor():
    vaultDir = settings.Vault or VAULT_DIR
    configPath = settings.Config or join(vaultDir, vaultconfig.DefaultConfigPath)
    cfg, err = vaultconfig.Load(configPath)
    if err: log warning; cfg = empty   # never block startup on a bad config
    server.Run(ctx, Config{
        VaultDir:    vaultDir,
        VaultConfig: cfg,              # NEW field on server.Config
        ...
    })

server.Run():
    state = NewRuntimeStateWithOptions(cfg.VaultDir, RuntimeOptions{
        SearchIndexPath: cfg.SearchIndexPath,
        VaultConfig:     cfg.VaultConfig,   # NEW: threaded into loadSnapshot
    })

runtime.loadSnapshot():
    v = vault.New(resolvedRoot, vault.WithConfig(opts.VaultConfig))
    ...
```

### 5.6 Backlink consistency (important subtlety)

Backlinks are computed in `buildBacklinks()` by iterating notes and following
their `WikiLinks`. If note A links to note B, and B has `publish: false`, then
B is simply not in `v.notes`. The existing resolver (`ResolveWikiLink`)
returns `false` for unknown slugs, so the backlink is dropped silently. **This
is the desired behavior** (a hidden note should not appear in another note's
backlinks), and it falls out for free from gating at load time. No special
backlink handling is needed.

One edge case to handle: if note A *embeds* (`![[B]]`) a hidden note B, the
embed renders as an empty `<div data-target="b">`. We should render a visible
"⚠ Note not published" marker instead, mirroring how unresolved image embeds
already render a broken-embed marker in `parser.ReplaceWikiEmbedImages`. This
is a small addition to `vault.rebuildHTML()`.

## 6. Decision records

### Decision A: `publish` is opt-out only (no `publish: true` force-publish)

- **Context:** Should `publish: true` override an ignore rule (path *or*
  config blacklist)?
- **Options considered:**
  1. `publish` is only ever a negative opt-out. Ignore/blacklist always wins.
  2. `publish: true` force-includes a file even if path-ignored.
- **Decision:** Option 1 — opt-out only.
- **Rationale:** A `publish: true` that resurrects an ignored file creates a
  confusing precedence layer (frontmatter vs. path rules) and a real footgun:
  an operator who blacklists `Secrets/` would expect that to be absolute.
  Keeping ignore/blacklist authoritative makes the security boundary clear.
- **Consequences:** `publish: false` hides a note; absence of the key (or
  `publish: true`) means "eligible, subject to path rules". Must be documented.
  Tests must assert that an ignored file with `publish: true` is still hidden.
- **Status:** accepted

### Decision B: Read `publish` from the generic frontmatter map; do not extend `ParsedNote`

- **Context:** The parser returns `ParsedNote.Frontmatter` as a generic
  `map[string]interface{}`. Should we add a typed `Publish bool` field to
  `ParsedNote`?
- **Options considered:**
  1. Add typed field to `ParsedNote` and set it in the parser.
  2. Leave the parser alone; read `frontmatter["publish"]` in the vault layer.
- **Decision:** Option 2.
- **Rationale:** The parser is deliberately a general Markdown engine; it
  already surfaces `tags` and `title` as conveniences but does not model every
  possible frontmatter key. `publish` is a *publishing-policy* concern, not a
  parsing concern, so it belongs in the vault layer that owns publication.
  Keeping the parser untouched avoids coupling and reduces blast radius.
- **Consequences:** A tiny `frontmatterBool` helper lives in the vault package
  (or a shared `internal/frontmatter` helper if reused). The flag never appears
  in `ParsedNote`; it only appears on `Note.Publish`.
- **Status:** accepted

### Decision C: Explicit `--config` + typed struct, not Glazed's generic config-file middleware

- **Context:** Glazed supports a generic `--config-file` middleware. Should we
  use it for the vault config?
- **Options considered:**
  1. Use Glazed's config-file middleware (generic key/value bag).
  2. Add an explicit `--config` flag and load YAML into a typed `vaultconfig.Config`.
- **Decision:** Option 2.
- **Rationale:** The vault config is vault-scoped (travels with the vault in
  git), has a small known schema (`ignore` list, future fields), and benefits
  from a typed struct for validation and IDE help. Glazed's middleware is
  generic and would conflate CLI-arg overrides with vault-file config. The
  serve command already builds `appserver.Config` from a typed `Settings`
  struct; a typed vault config slots in cleanly.
- **Consequences:** We hand-roll a ~40-line YAML loader using `gopkg.in/yaml.v3`
  (already an indirect dependency in `go.mod`). No new config framework.
- **Status:** accepted

### Decision D: New `pkg/vaultconfig` matcher via library; leave `internal/ignore` for `.vault-ignore`

- **Context:** The config blacklist wants full gitignore (`**`, nested). The
  existing `internal/ignore` deliberately lacks these. Replace or augment?
- **Options considered:**
  1. Extend `internal/ignore` to support `**` and use it for both files.
  2. Add a library-backed matcher in a new `pkg/vaultconfig` for the config
     file; leave `internal/ignore` as-is for `.vault-ignore`.
- **Decision:** Option 2.
- **Rationale:** `internal/ignore` is battle-tested and its limitations are
  documented and relied upon by existing tests; changing its semantics risks
  regressions and breaks the documented contract. A library
  (`github.com/sabhiram/go-gitignore` or `github.com/monochromegane/go-gitignore`)
  gives full gitignore semantics for the new config with zero risk to existing
  behavior. Two matchers compose trivially (excluded if *either* matches).
- **Consequences:** Two matchers exist. Both feed `IsExcluded`. `.vault-ignore`
  remains the legacy path; config blacklist is the recommended new path.
  Document the relationship in README. The library must be vendored via
  `go get` and added to `go.mod`.
- **Status:** accepted

### Decision E: Config and `.vault-ignore` changes require reload; `publish` flag is incremental

- **Context:** The watcher reloads individual files but does not re-read config
  or `.vault-ignore` on change (documented behavior for `.vault-ignore`).
- **Options considered:**
  1. Watch the config file and hot-reload the matcher.
  2. Treat config like `.vault-ignore`: changes take effect on next full reload
     (restart or admin reload endpoint). The `publish` flag, being per-file,
     is picked up by the existing incremental `ReloadNote`.
- **Decision:** Option 2.
- **Rationale:** Consistency with `.vault-ignore`. Hot-reloading a matcher that
  is read concurrently by request handlers and the watcher requires careful
  atomic swap logic; the existing reload endpoint already does atomic snapshot
  swaps and is the supported path for policy changes. The `publish` flag gets
  incremental updates for free because it is read inside `loadNote`/`ReloadNote`.
- **Consequences:** Operators editing `.publish/config.yaml` must call the
  admin reload endpoint (or restart). Toggling `publish: false` on a single
  note updates within the debounce window in `--watch` mode. Document both.
- **Status:** accepted

## 7. Pseudocode and key flows

### 7.1 Startup (full load)

```text
main -> serve command -> appserver.Run(ctx, cfg)
  cfg.VaultConfig = vaultconfig.Load(cfgPath)   # nil-safe, never fatal
  state = NewRuntimeStateWithOptions(cfg.VaultDir, {
      SearchIndexPath, VaultConfig: cfg.VaultConfig
  })
  -> loadSnapshot(resolvedRoot, opts)
       v = vault.New(resolvedRoot, vault.WithConfig(opts.VaultConfig))
         -> ignore.Load(root)            # legacy .vault-ignore
         -> vaultconfig.NewMatcher(cfg)  # config blacklist
         -> LoadAll():
              walk; for each dir prune if ShouldPruneDir;
              for each file: if IsExcluded skip; else loadNote;
                            if not note.Publish skip; else store
              buildWikiLinkIndex; buildBacklinks; rebuildHTML
       si = buildSearchIndex(v, ...)   # indexes only stored (published) notes
       return Snapshot{Vault: v, Search: si, ...}
```

### 7.2 Incremental note edit (`--watch`)

```text
fsnotify event on X.md (not ignored)
  -> watcher debounce -> apply(path, op)
       if Remove|Rename: RemoveNote; search.Delete
       else: ReloadNote(path)
            if IsExcluded(path): return ErrExcluded -> watcher drops from search
            note = loadNote(path)
            if not note.Publish:
                RemoveNote(path); return ErrExcluded -> search.Delete
            store note; rebuildWikiLinkIndex; buildBacklinks; rebuildHTML
            search.Index(SearchDocument(note))
```

### 7.3 Config edit (no `--watch` hot reload)

```text
operator edits .publish/config.yaml
  -> operator calls POST /api/admin/reload (or restarts)
       -> runtime.Reload() -> loadSnapshot() -> new Vault with new matcher
       -> atomic swap; old snapshot closed after delay
```

### 7.4 Request-time raw source (cannot bypass rules)

```text
GET /api/notes/{slug}/raw
  -> note = v.GetNote(slug)          # only publishable notes are stored
  -> if !ok: 404
  -> v.ReadRaw(note.Path):
       if IsExcluded(path): os.ErrNotExist   # config blacklist enforced here too
       open via os.OpenRoot (sandboxed); return bytes
```

## 8. Implementation phases

Each phase is independently testable and committable. Do them in order.

### Phase 1 — `pkg/vaultconfig` package (no vault changes yet)

**Goal:** a loadable, testable config + matcher with no integration risk.

1. `go get github.com/sabhiram/go-gitignore` (evaluate `monochromegane/go-gitignore`
   as alternative — see §10; pick one and record why in the diary).
2. Create `pkg/vaultconfig/config.go` with `Config`, `Load`, `DefaultConfigPath`.
   - Missing file → `&Config{}, nil`.
   - Malformed file → `&Config{}, err`.
3. Create `pkg/vaultconfig/matcher.go` with `Matcher`, `NewMatcher`, `Match`,
  `Empty`.
   - Wrap the library's `GitIgnore`.
   - Convert OS paths to slash before matching.
4. Write `pkg/vaultconfig/config_test.go`:
   - missing file is empty;
   - `Secrets/` excludes `Secrets/x.md`;
   - `**/node_modules/` excludes `a/b/node_modules/c` (this is the case the
     legacy matcher cannot do — assert it);
   - negation `!Secrets/Public.md` re-includes;
   - empty config excludes nothing.

**Done when:** `go test ./pkg/vaultconfig/...` passes; package compiles in
isolation.

### Phase 2 — Vault gating (`pkg/vault/vault.go`)

**Goal:** the loader excludes via config blacklist and `publish` flag.

1. Add `configMatcher *vaultconfig.Matcher` field and `WithConfig` option.
2. Add `IsExcluded(absPath, isDir)` combining `isIgnored` + `configMatcher`.
3. Replace `isIgnored` with `IsExcluded` at these call sites (use `grep`):
   - `LoadAll` file/directory checks;
   - `RefreshAssetIndex`;
   - `ReadRaw`;
   - `ReloadNote`;
   - the watcher's per-event check (it calls `v.IsIgnored` — switch to
     `IsExcluded`, or add `IsExcluded` as the new public method and keep
     `IsIgnored` delegating to it for back-compat with the watcher; prefer the
     former plus updating the watcher call).
4. Add `Note.Publish bool` and the `frontmatterBool` helper. Set it in
   `loadNote`. In `LoadAll`, skip notes with `!Publish`. In `ReloadNote`,
   when `!Publish`, call `RemoveNote` and return `ErrIgnored` (reuse the
   existing sentinel so the watcher already handles it).
5. Add the broken-embed marker for links/embeds to hidden notes in
   `rebuildHTML` (small regex addition mirroring the image case).

**Done when:** `go test ./pkg/vault/...` passes, including new tests:
- `TestLoadAllRespectsPublishFalse` — a note with `publish: false` is absent;
- `TestLoadAllRespectsConfigBlacklist` — a folder blacklisted in config is
  absent;
- `TestPublishFalseOverridesIgnore` is **not** asserted (Decision A: ignore
  wins); instead `TestIgnoredNoteWithPublishTrueStillHidden`;
- `TestReloadNoteDropsPublishFalse` — toggling a note to `publish: false`
  removes it from the snapshot and the watcher path returns `ErrIgnored`.

### Phase 3 — Serve command wiring

**Goal:** operators can supply a config file.

1. Add `--config` Glazed flag in `serve.go` (default empty →
   `<vault>/.publish/config.yaml`).
2. Add `Config` field to the serve `Settings` struct and decode it.
3. In `RunIntoGlazeProcessor`, load the config (nil-safe, log on error) and
   pass it on `appserver.Config.VaultConfig` (new field).
4. Add `VaultConfig` to `server.Config` and thread it through
   `NewRuntimeStateWithOptions` → `RuntimeOptions` → `loadSnapshot` →
   `vault.New(resolvedRoot, vault.WithConfig(opts.VaultConfig))`.

**Done when:** `go run ./cmd/retro-obsidian-publish serve --vault ./vault-example`
loads; with a sample `.publish/config.yaml` containing an ignore entry, the
matched folder is absent from `/api/notes` and `/api/tree`.

### Phase 4 — Tests, docs, examples

1. Add `vault-example/.publish/config.yaml` sample and a `publish: false` note
   to the example vault so manual verification is easy.
2. Update `README.md`:
   - new "Per-note `publish` flag" section near "Frontmatter";
   - extend "Excluding paths" to mention the config blacklist as the
     `**`-capable option, keeping `.vault-ignore` docs intact;
   - add `--config` to the server-flags table and environment table.
3. Update `docs/` if any operator runbook references ignore behavior.
4. Run the full validation checklist from README: `go test ./...`,
   `go build -tags embed`, smoke-test `/api/healthz`, `/api/notes`, `/api/tree`.

**Done when:** all tests green, README accurate, example vault demonstrates
both features.

## 9. Test strategy

- **Unit (config):** `pkg/vaultconfig/config_test.go` — file absence, malformed
  YAML, `**` patterns, negations, empty config.
- **Unit (matcher semantics):** pin the library's behavior for the exact
  patterns we document; if the library mishandles an edge case, the test makes
  it visible before users hit it.
- **Unit (vault):** extend `pkg/vault/vault_test.go` (which already has
  `TestLoadAllRespectsVaultIgnore` at line 270) with the publish-flag and
  config-blacklist cases listed in Phase 2.
- **Unit (watcher):** extend `pkg/watcher/watcher_test.go` to assert that a note
  edited to `publish: false` is removed from the snapshot and search index, and
  that `ErrIgnored` is returned.
- **Integration (serve):** a table-driven test that starts the server with a
  sample vault + config and asserts `/api/notes` omits the hidden note and the
  blacklisted folder; `/api/notes/{slug}/raw` returns 404 for both.
- **Regression:** the existing `.vault-ignore` tests must stay green unchanged
  (they are the back-comat contract).
- **Property to verify, not assert:** backlinks to hidden notes are silently
  dropped (free from gating); verify with one test that links a public note to
  a `publish:false` note and checks the public note's backlinks do not list
  the hidden one and the link does not 500.

## 10. Risks, alternatives, and open questions

### Risks

- **Two matchers, subtle precedence.** A path excluded by `.vault-ignore` but
  *re-included* by a config negation, or vice versa, could confuse users.
  Mitigation: `IsExcluded` is "excluded if *either* matcher says so", so
  negation in one file cannot override exclusion in the other. Document this
  explicitly. (If a user needs a re-include, they must remove the exclusion
  from the other file.)
- **Library behavior drift.** `sabhiram/go-gitignore` is lightly maintained.
  If it misbehaves, we absorb the bug. Mitigation: `monochromegane/go-gitignore`
  is the documented faster alternative; pin the choice in the diary with a
  one-paragraph rationale, and keep the matcher interface narrow so swapping
  libraries later is a one-file change.
- **`publish` flag vs. ignore precedence confusion.** See Decision A. The only
  real risk is a user expecting `publish: true` to resurrect an ignored file.
  Mitigation: README + a test asserting ignore wins.
- **Reload semantics surprise.** Users may expect config edits to hot-reload.
  Mitigation: README states config changes need a reload (same as
  `.vault-ignore`); the `publish` flag is incremental (free).
- **Frontmatter type variance.** YAML `publish: false` vs `publish: "false"`.
  Mitigation: `frontmatterBool` accepts both bool and string forms.

### Alternatives considered

- **Extend `internal/ignore` with `**`.** Rejected (Decision D): risks the
  documented contract and existing tests; library gives full semantics free.
- **Single unified ignore file.** Rejected: would break `.vault-ignore` users
  and the documented subset contract. Keep both; config is additive.
- **Glazed config-file middleware.** Rejected (Decision C): too generic for a
  typed, vault-scoped schema.

### Open questions (resolve before/while implementing)

1. **Which gitignore library?** `sabhiram/go-gitignore` (simple, popular) vs
   `monochromegane/go-gitignore` (tree-indexed, fast for many patterns). The
   vault ignore list will usually be short, so simplicity likely wins; confirm
   `**` handling in a spike test and record the choice in the diary.
2. **Should the config file support a `defaultPublish` global?** E.g. a vault
   that defaults to private and requires opt-in. Out of scope for this ticket
   (the `publish` flag is opt-out by design), but the `Config` struct can grow
   a `DefaultPublish *bool` field later without breaking the loader. Leave as
   a documented future extension.
3. **Nested config files?** No — one config at the vault root (or `--config`).
   Nested `.vault-ignore`-style files are explicitly unsupported by the legacy
   matcher and we will not add them.

## 11. References

Key files (absolute paths) and the role each plays in this design:

- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/vault/vault.go`
  — core loader. `New` (l. 82), `LoadAll` (l. 102), `loadNote` (l. 152),
  `isIgnored`/`IsIgnored`, `ShouldPruneDir` (l. 455), `ReloadNote` (l. 430),
  `ReadRaw`, `FileTree`. **Primary edit target for Phase 2.**
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/ignore/ignore.go`
  — legacy `.vault-ignore` matcher. `IgnoreFile` (l. 25), `Match` (l. 149),
  `HasNegations` (l. 187). **Unchanged; read for parity.**
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/internal/parser/parser.go`
  — Markdown parser. `Parse` (l. 56), `ParsedNote` (l. 30),
  `normalizeFrontmatter` (l. 479), `extractTags` (l. 530). **Unchanged; frontmatter
  is already a generic map.**
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/watcher/watcher.go`
  — fsnotify watcher. Calls `IsIgnored`/`ReloadNote`; switch to `IsExcluded`.
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/server/server.go`
  — `server.Config` struct and `Run`. Add `VaultConfig` field.
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/server/runtime.go`
  — snapshot lifecycle. `loadSnapshot` threads config into `vault.New`.
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/cmd/retro-obsidian-publish/commands/serve/serve.go`
  — CLI flags. Add `--config`; load config here. **Primary edit target for Phase 3.**
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/api/api.go`
  — HTTP handlers. **No change needed** — reads from `v.notes`, which is already
  gated.
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/pkg/search/search.go`
  — Bleve index. **No change needed** — indexes from `vault.SearchDocuments()`,
  which iterates stored (published) notes.
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/README.md`
  — user docs. Update in Phase 4. `.vault-ignore` section at the
  "Excluding paths with `.vault-ignore`" heading; frontmatter section at
  "Frontmatter"; flags table under "Configuration".
- `/home/manuel/workspaces/2026-06-22/goja-publish-vault/publish-vault/vault-example/`
  — example vault for manual verification; add a `publish: false` note and a
  `.publish/config.yaml`.

External:

- `github.com/sabhiram/go-gitignore` — candidate library; see
  https://pkg.go.dev/github.com/sabhiram/go-gitignore.
- `github.com/monochromegane/go-gitignore` — faster alternative for many patterns.
- `gopkg.in/yaml.v3` — already an indirect dependency in `go.mod` (line 161).
