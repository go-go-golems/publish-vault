---
Title: Project and design review — memory handling, index building, and search architecture
Ticket: RETRO-MEMORY-012
Status: active
Topics:
    - retro-obsidian-publish
    - search
    - vault
    - deployment
    - git-sync
    - obsidian-vault
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/api/api.go
      Note: Review highlights getNote rawMarkdown API compatibility and lazy response DTO strategy
    - Path: internal/search/search.go
      Note: Review highlights NewPersistent stale-delete risk
    - Path: internal/server/runtime.go
      Note: Review highlights snapshot isolation and reload lifecycle concerns for persistent per-snapshot indexes
    - Path: internal/vault/vault.go
      Note: Review highlights Note as overloaded storage/API/search/render model and recommends DTO/model split
    - Path: internal/watcher/watcher.go
      Note: Watcher delete path shows search.Delete exists
    - Path: web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Frontend depends on note.rawMarkdown for Copy as Markdown; backend-only removal would break UI
    - Path: web/src/types/index.ts
      Note: TypeScript Note contract currently requires rawMarkdown
ExternalSources: []
Summary: 'Second-pass review of the RETRO-MEMORY-012 intern design: confirms the OOM diagnosis but corrects risky assumptions around persistent bleve, snapshot isolation, raw markdown API compatibility, and implementation sequencing.'
LastUpdated: 2026-07-05T00:00:00Z
WhatFor: Project/design review before implementing the memory and search-index changes
WhenToUse: Read after the original intern guide and before coding Phase 1/2 fixes
---


# Project and Design Review: Memory Handling, Index Building, and Search Architecture

## 1. Review verdict

The original intern guide is useful and mostly points in the right direction: the production failure is very likely memory-related, the reload endpoint is correctly identified, the `git-sync` carry-over flow is explained well, and avoiding a SQL/Postgres rewrite is the right default. However, the implementation plan needs correction before we code against it.

The biggest issue is that the guide treats `search.NewPersistent()` as a nearly drop-in memory fix. It is **not** safe as written. A persistent bleve index introduces stale-document bugs, close/lifecycle requirements, file-locking concerns, and — most importantly — can violate the current `RuntimeState` snapshot model if the same index directory is updated in place while old readers are still serving requests.

The second biggest issue is API compatibility. The guide says we can remove `RawMarkdown` while keeping the public API unchanged. Today the frontend TypeScript type requires `rawMarkdown`, and `NoteRenderer` uses it for the "Copy as Markdown" button. We can still stop storing raw markdown in memory, but `GET /api/notes/{slug}` must either lazily read raw markdown into the response DTO or the frontend must change at the same time.

The revised recommendation is:

1. **Hotfix prod separately** by raising the memory limit.
2. **Instrument first** so we stop guessing about heap, RSS, and reload spikes.
3. **Split storage models from API DTOs** so the in-memory vault can be lean while the HTTP response remains compatible.
4. **Lazy-load raw markdown in `getNote` and `/raw`** as the first code memory win.
5. **Fix search indexing deliberately** with either an in-memory lean document model or a persistent per-snapshot index that is built in a staging directory, swapped atomically, closed, and garbage-collected.
6. Only then consider lazy HTML or deeper incremental indexing.

---

## 2. What the original design got right

### 2.1 OOM diagnosis and cluster framing are sound

The production pod reports `OOMKilled` for the `app` container, while `ssr` and `git-sync` are ready. The original guide correctly separates this from cluster-wide memory starvation. Raising the `app` container limit is a valid operational hotfix because the node has spare memory.

**Keep this part.** It is the right immediate operational response.

### 2.2 The current runtime path is mapped correctly

The guide correctly traces:

```text
git push -> git-sync sidecar -> /api/admin/reload -> RuntimeState.Reload -> loadVaultAndSearch -> snapshot swap
```

The code supports that reading:

- `internal/server/server.go:90-91` registers `POST /api/admin/reload` when reload auth is configured.
- `internal/server/server.go:166-180` calls `state.Reload()` and returns `204`.
- `internal/server/runtime.go:62-74` builds a new vault/search pair and swaps it under a lock.
- `internal/server/runtime.go:77-96` resolves the symlink, loads the vault, and builds search.

The snapshot swap is good architecture. It avoids half-built state being exposed to users.

### 2.3 Not introducing SQL/Postgres is the right default

The problem is not primarily relational querying. The current app is a file-backed publisher with a full-text search need. Adding SQL would introduce schema, migrations, backups, and an extra operational dependency. Bleve already exists in the codebase and fits the search problem.

**Keep the conclusion:** do not jump to Postgres. But see §5 for a safer search-index plan than the original document proposes.

---

## 3. Where the original design is too optimistic or incorrect

### 3.1 The reload-triggered OOM is plausible, but still a hypothesis

**Problem:** The original guide says reload is "the most likely trigger" for the OOM. That is plausible, but the captured logs only prove that the app OOMs after startup, not that `git-sync` necessarily fired a webhook in that exact window.

**Where to look:**

- `internal/server/runtime.go:62-96` shows reload can double state.
- Prod app logs showed startup and no panic, but did not show a reload line before death.
- `git-sync` logs were not included in the ticket artifact as timestamp-correlated evidence.

**Why it matters:** If startup itself can exceed the limit due to parser/search transients, then fixing only reload memory might not be sufficient. Conversely, if reload is the trigger, we need to harden the carry-over path specifically.

**Cleanup / verification sketch:**

```bash
# Capture app and git-sync logs around the same wall-clock interval.
kubectl logs -n retro-obsidian-publish POD -c app --previous --timestamps
kubectl logs -n retro-obsidian-publish POD -c git-sync --timestamps --tail=200

# Add temporary structured reload logging before and after state.Reload:
# reload_start revision=<resolvedRoot> heap=<heap> rss=<rss>
# reload_done  revision=<resolvedRoot> heap=<heap> rss=<rss> duration=<d>
```

**Design review decision:** Treat reload doubling as a serious design smell, but add instrumentation before claiming it is the only trigger.

---

### 3.2 The memory estimate underweights build-time transients

**Problem:** The guide counts steady-state copies (`RawMarkdown`, `HTML`, bleve index), but the code also allocates significant temporary buffers during parse and indexing.

**Where to look:**

`internal/parser/parser.go:42-97` does:

```go
wikiLinks := extractWikiLinks(src)
processed := replaceWikiLinks(src)
var buf bytes.Buffer
md.Convert(processed, &buf, parser.WithContext(ctx))
htmlOut := buf.String()
```

`internal/search/search.go:92-98` does:

```go
doc := noteDoc{
    Title:   note.Title,
    Body:    stripHTML(note.HTML),
    Tags:    tags,
    Excerpt: note.Excerpt,
}
return si.idx.Index(note.Slug, doc)
```

`stripHTML` (`internal/search/search.go:300-318`) allocates a new byte slice and a new string for every note body.

**Why it matters:** Startup and reload peak memory may be much higher than steady-state memory. The process can OOM while building even if the final snapshot would fit.

**Cleanup sketch:**

```go
// Add local instrumentation around each phase.
func logMem(label string) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    log.Printf("mem phase=%s heap_alloc=%d heap_sys=%d num_gc=%d", label, m.HeapAlloc, m.HeapSys, m.NumGC)
}

func loadVaultAndSearch(...) (...) {
    logMem("before_eval_symlink")
    v, err := vault.New(resolvedRoot)
    logMem("after_vault_new")
    si, err := search.New(v)
    logMem("after_search_new")
    return ...
}
```

**Design review decision:** Instrument before and after `vault.New`, `buildWikiLinkIndex`, `buildBacklinks`, `rebuildHTML`, and `search.New`. Then choose the first code fix based on measured peaks.

---

### 3.3 `search.NewPersistent` is not safe as a drop-in: stale deletes

**Problem:** `NewPersistent` opens an existing index and re-indexes current notes, but it never deletes documents for notes that no longer exist. Search results can therefore return deleted notes after a git push.

**Where to look:** `internal/search/search.go:55-77`:

```go
func NewPersistent(v *vault.Vault, indexPath string) (*Index, error) {
    ...
    if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
        idx, err = bleve.New(indexPath, buildMapping())
    } else {
        idx, err = bleve.Open(indexPath)
    }
    ...
    // Re-index all notes
    for _, note := range v.AllNotes() {
        if err := si.Index(note); err != nil {
            return nil, err
        }
    }
    return si, nil
}
```

The watcher has a delete path (`internal/watcher/watcher.go:114-123`):

```go
if op&(fsnotify.Remove|fsnotify.Rename) != 0 {
    slug := vw.vault.RemoveNote(path)
    if vw.search != nil {
        _ = vw.search.Delete(slug)
    }
    return
}
```

But full reload does not compute deleted slugs and does not call `Delete`.

**Why it matters:** The original design's Phase 1 can silently corrupt search semantics. Deleted notes remain searchable. A result can link to `/note/<slug>` that returns 404.

**Cleanup sketch, option A: rebuild clean each reload:**

```go
func NewPersistentFresh(v *vault.Vault, indexPath string) (*Index, error) {
    _ = os.RemoveAll(indexPath)        // safe only for a staging dir, never active dir
    idx, err := bleve.New(indexPath, buildMapping())
    if err != nil { return nil, err }
    si := &Index{idx: idx}
    for _, note := range v.AllNotes() {
        if err := si.Index(note); err != nil {
            _ = idx.Close()
            return nil, err
        }
    }
    return si, nil
}
```

**Cleanup sketch, option B: incremental reconcile:**

```go
func ReconcileIndex(idx bleve.Index, current map[string]*vault.Note) error {
    indexedIDs := listAllDocumentIDs(idx)
    for id := range indexedIDs {
        if _, ok := current[id]; !ok {
            idx.Delete(id)
        }
    }
    for slug, note := range current {
        idx.Index(slug, noteToSearchDoc(note))
    }
    return nil
}
```

Option A is simpler and safer initially. Option B is faster but needs more bleve API knowledge and tests.

**Design review decision:** Do not use the existing `NewPersistent` directly in production. Replace or wrap it with a fresh-per-snapshot or reconcile implementation.

---

### 3.4 Persistent search indexes need explicit lifecycle and `Close`

**Problem:** `search.Index` wraps `bleve.Index` but exposes no `Close` method. With in-memory indexes, this mostly leaks until GC. With persistent indexes, not closing can leak file descriptors, locks, mmap state, and disk resources.

**Where to look:** `internal/search/search.go:26-30`:

```go
type Index struct {
    mu  sync.Mutex
    idx bleve.Index
}
```

No `Close()` exists. `RuntimeState.Reload` (`internal/server/runtime.go:62-74`) overwrites `s.search` without closing the old one.

**Why it matters:** A persistent-index rollout that reloads every git push can accumulate open index handles. Depending on the bleve backend, the next open may fail, disk may not be reclaimed, or memory may remain pinned longer than expected.

**Cleanup sketch:**

```go
func (si *Index) Close() error {
    si.mu.Lock()
    defer si.mu.Unlock()
    if si.idx == nil {
        return nil
    }
    err := si.idx.Close()
    si.idx = nil
    return err
}

type Snapshot struct {
    Root string
    Vault *vault.Vault
    Search *search.Index
}

func (s *RuntimeState) Reload() error {
    next, err := LoadSnapshot(...)
    if err != nil { return err }

    s.mu.Lock()
    old := s.snapshot
    s.snapshot = next
    s.mu.Unlock()

    // Close after swap. If request handlers can hold old.Search beyond Snapshot(),
    // consider reference counting or delayed close.
    go closeAfterGrace(old.Search)
    return nil
}
```

**Design review decision:** Add `Index.Close()` before any persistent index is used. Decide whether immediate close after swap is safe enough or whether a short grace period is required.

---

### 3.5 A shared persistent index directory breaks snapshot isolation

**Problem:** The current runtime model returns a vault pointer and a search pointer as one snapshot. If Phase 1 updates `/data/search/index` in place, old readers may observe the new search contents before `RuntimeState` swaps the new vault into place.

**Where to look:**

- `internal/server/runtime.go:39-44` returns the active vault and search pointers together.
- `internal/api/api.go:166-167` obtains the search pointer from that snapshot and searches it.
- The original design proposes `--search-index-path=/data/search/index` as a stable path.

**Why it matters:** Atomic snapshot semantics are one of the good properties of the current design. Reusing a mutable persistent index path means search can be a moving target independent of the vault snapshot. That can produce mismatches:

```text
request A gets old Vault_A + old Search_A handle
reload starts mutating /data/search/index in place
request A searches and sees new document IDs
request A follows result, but old Vault_A cannot find the note
```

Even if bleve handles concurrent mutations correctly, the application-level snapshot is no longer consistent.

**Cleanup sketch: per-snapshot index directories:**

```text
/data/search/
  snapshots/
    <revision-a>/index/        # active with Vault_A
    <revision-b>.building/     # build here, not visible yet
    <revision-b>/index/        # rename/mark complete before swap
```

```go
func BuildSnapshot(configuredRoot, searchBase string) (*Snapshot, error) {
    resolved := filepath.EvalSymlinks(configuredRoot)
    revision := filepath.Base(resolved) // or hash(resolved)
    buildDir := filepath.Join(searchBase, revision+".building")
    finalDir := filepath.Join(searchBase, revision)

    _ = os.RemoveAll(buildDir)
    v := vault.New(resolved)
    si := search.NewPersistentFresh(v, filepath.Join(buildDir, "index"))

    if err := os.Rename(buildDir, finalDir); err != nil { ... }
    return &Snapshot{Root: resolved, Vault: v, Search: si, IndexDir: finalDir}, nil
}
```

**Design review decision:** If using persistent indexes, build them as part of an immutable per-revision snapshot. Do not mutate a single shared active index directory during reload.

---

### 3.6 `RawMarkdown` cannot simply disappear from `GET /api/notes/{slug}`

**Problem:** The original guide says remove `RawMarkdown` from `Note` and keep the public API unchanged. That is contradictory with current frontend code.

**Where to look:**

`internal/vault/vault.go:18-29` includes:

```go
HTML        string `json:"html"`
RawMarkdown string `json:"rawMarkdown"`
```

`internal/api/api.go:124-134` returns the whole `Note` object:

```go
note, ok := v.GetNote(slug)
...
jsonResponse(w, note)
```

The frontend requires and uses `rawMarkdown`:

- `web/src/types/index.ts:26-38` defines `Note.rawMarkdown: string`.
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:273-279` copies `note.rawMarkdown` to the clipboard.
- `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:323-335` also links to `/api/notes/{slug}/raw` for download/view.

**Why it matters:** Removing the JSON field without a coordinated frontend/API DTO change breaks the copy button and TypeScript contract. It may also affect SSR because `web/server.mjs:197-205` prefetches full note JSON for server rendering.

**Cleanup sketch: keep API, slim storage:**

```go
// Storage model: no RawMarkdown.
type Note struct {
    Slug string
    Title string
    Path string
    HTML string
    ...
}

// API DTO: can include RawMarkdown lazily.
type NoteResponse struct {
    Slug string `json:"slug"`
    Title string `json:"title"`
    Path string `json:"path"`
    HTML string `json:"html"`
    RawMarkdown string `json:"rawMarkdown"`
    ...
}

func (h *Handler) getNote(w http.ResponseWriter, r *http.Request) {
    v, _ := h.provider.Snapshot()
    note, ok := v.GetNote(slug)
    raw, err := v.ReadRaw(note.Path) // one file only; bounded allocation
    if err != nil { ... }
    jsonResponse(w, NewNoteResponse(note, string(raw)))
}
```

Alternative frontend change:

```tsx
const handleCopyMarkdown = useCallback(async () => {
  const res = await fetch(`/api/notes/${encodeURIComponent(note.slug)}/raw`);
  navigator.clipboard.writeText(await res.text());
}, [note.slug]);
```

That alternative removes `rawMarkdown` from full note JSON and is a cleaner API, but it is a frontend-visible contract change.

**Design review decision:** For a low-risk backend memory fix, keep `rawMarkdown` in the **response DTO** but remove it from the **in-memory storage model**. A later frontend cleanup can fetch `/raw` on demand.

---

### 3.7 Search currently indexes HTML-derived text, not markdown-derived text

**Problem:** `search.Index` strips tags from rendered HTML (`stripHTML(note.HTML)`) and indexes the result. This forces HTML to exist before search can be built and allocates a full stripped copy per note.

**Where to look:** `internal/search/search.go:92-98`:

```go
doc := noteDoc{
    Title:   note.Title,
    Body:    stripHTML(note.HTML),
    Tags:    tags,
    Excerpt: note.Excerpt,
}
return si.idx.Index(note.Slug, doc)
```

**Why it matters:** This couples search indexing to HTML rendering. It also makes a future lazy-HTML design harder: if we drop `HTML` from memory, search still wants it. Search should index a search document, not UI HTML.

**Cleanup sketch:** introduce a separate parse output / search document:

```go
type Note struct {
    Slug string
    Title string
    Path string
    HTML string          // UI concern; maybe cached, maybe lazy later
    SearchText string    // optional; or pass directly to search without storing
    Tags []string
    Excerpt string
}

type SearchDocument struct {
    Slug string
    Title string
    Body string
    Tags []string
    Excerpt string
}

func (n *Note) SearchDocument() search.Document {
    return search.Document{
        Slug: n.Slug,
        Title: n.Title,
        Body: n.SearchText, // markdown/plain text, not stripped HTML
        Tags: n.Tags,
        Excerpt: n.Excerpt,
    }
}
```

At minimum, move `stripHTML` out of `search.Index` and make the caller supply a `SearchDocument`. That makes the memory cost explicit and testable.

**Design review decision:** Do not design search around `Note.HTML`. Make search consume an explicit `SearchDocument` or `PlainText` representation.

---

### 3.8 Full `AllNotes()` snapshots are small but frequent allocations

**Problem:** `Vault.AllNotes()` allocates a new `[]*Note` slice every time it is called. This is fine at 887 notes, but it appears in health/config/list/search indexing paths and can become visible during reload and crawl traffic.

**Where to look:** `internal/vault/vault.go` (method near the bottom, not shown in the original design snippets):

```go
func (v *Vault) AllNotes() []*Note {
    v.mu.RLock()
    defer v.mu.RUnlock()
    notes := make([]*Note, 0, len(v.notes))
    for _, n := range v.notes {
        notes = append(notes, n)
    }
    return notes
}
```

Call sites include `healthHandler` (`server.go:159-162`), `getConfig` (`api.go:83-89`), `listNotes` (`api.go:103-121`), and search construction (`search.go:47`, `search.go:71`).

**Why it matters:** Not the root cause, but it is easy to avoid in hot paths. Health checks should not allocate an all-notes slice every 10 seconds forever.

**Cleanup sketch:**

```go
func (v *Vault) Count() int {
    v.mu.RLock()
    defer v.mu.RUnlock()
    return len(v.notes)
}

func healthHandler(state *RuntimeState) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        v, _ := state.Snapshot()
        fmt.Fprintf(w, `{"ok":true,"notes":%d}`, v.Count())
    }
}
```

**Design review decision:** Add `Count()` during the cleanup. It is low-risk and removes allocation noise from health/config.

---

## 4. Revised architecture recommendation

### 4.1 Separate the model into three layers

The current `vault.Note` is trying to be all of these at once:

1. internal storage model,
2. public API response model,
3. search indexing document,
4. UI-rendered content cache.

That is why every optimization threatens an API break. Split the responsibilities explicitly.

```text
             filesystem (.md files)
                     |
                     v
           parser.Parse / note loader
                     |
         +-----------+------------+
         |                        |
         v                        v
  VaultNoteMeta              RenderedContent
  (slug, title, path,        (html; maybe cached,
   tags, excerpt,            maybe lazy later)
   wikiLinks, backlinks)
         |
         +-----------> SearchDocument (title, body, tags, excerpt)
         |
         +-----------> API DTO (NoteResponse; may lazily include rawMarkdown)
```

**Concrete shape:**

```go
type Note struct {
    Slug string
    Title string
    Path string
    Frontmatter map[string]interface{}
    Tags []string
    Excerpt string
    HTML string          // keep for now; later can be lazy
    WikiLinks []WikiLinkRef
    Backlinks []string
    ModTime time.Time
}

type NoteResponse struct {
    Slug string `json:"slug"`
    Title string `json:"title"`
    Path string `json:"path"`
    Frontmatter map[string]interface{} `json:"frontmatter"`
    Tags []string `json:"tags"`
    Excerpt string `json:"excerpt"`
    HTML string `json:"html"`
    RawMarkdown string `json:"rawMarkdown"` // response-only, not stored
    WikiLinks []vault.WikiLinkRef `json:"wikiLinks"`
    Backlinks []string `json:"backlinks"`
    ModTime time.Time `json:"modTime"`
}

type SearchDocument struct {
    Slug string
    Title string
    Body string
    Tags []string
    Excerpt string
}
```

This lets us reduce memory without surprising the frontend.

### 4.2 Treat runtime state as an immutable snapshot

Instead of returning independent `vault` and `search` pointers, make the snapshot explicit:

```go
type Snapshot struct {
    Revision string
    ResolvedRoot string
    Vault *vault.Vault
    Search *search.Index
    IndexDir string
    BuiltAt time.Time
}

type RuntimeState struct {
    mu sync.RWMutex
    configuredRoot string
    snapshot *Snapshot
}

func (s *RuntimeState) Snapshot() *Snapshot {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.snapshot
}
```

Then every request uses one coherent snapshot:

```go
snap := h.provider.Snapshot()
note, ok := snap.Vault.GetNote(slug)
results, err := snap.Search.Search(q, 30)
```

This makes the future persistent-index lifecycle much easier to reason about.

### 4.3 If persistent index is used, build it per snapshot

Recommended first implementation:

```text
/data/search/snapshots/<revision>/index
```

Flow:

1. Resolve `/git/root/current` to concrete worktree path.
2. Derive a revision ID from the resolved worktree basename or git SHA.
3. Build the vault in memory.
4. Build a **fresh** bleve index in `/data/search/snapshots/<revision>.building/index`.
5. On success, rename/mark complete and open as the new snapshot's `Search`.
6. Swap `RuntimeState.snapshot`.
7. Close old `Search` after a grace period.
8. Delete old index dirs after no snapshot uses them.

This preserves snapshot isolation and avoids stale deletes. It may use more disk, but the vault is small enough that this is acceptable.

---

## 5. Revised implementation sequence

### Phase 0 — Production hotfix stays the same

Raise the `app` limit to 3072Mi in the hetzner-k3s deployment. This is operationally correct and should happen before code work.

### Phase 1 — Add observability before changing storage

**Why first:** We need to know whether startup, reload, parse, HTML rebuild, or search indexing is the dominant memory peak.

Add:

- `runtime.MemStats` logs around `vault.New`, `rebuildHTML`, `search.New`, `Reload` start/end.
- Optional `/api/debug/memory` behind loopback/admin only, or extended `/api/healthz` with heap numbers.
- Container env `GOMEMLIMIT` once the limit is raised, so Go GC reacts before cgroup death.

Acceptance:

```text
startup log shows phase-by-phase heap
reload log shows phase-by-phase heap
we know measured peak, not just inferred peak
```

### Phase 2 — Lazy raw markdown while preserving API compatibility

This is the safest memory code change.

Implementation:

1. Remove `RawMarkdown` from the in-memory `vault.Note`.
2. Add `Vault.ReadRaw(note.Path)` using `os.OpenRoot` or equivalent safe path opening.
3. Change `getNote` to construct a `NoteResponse` and lazily include raw markdown.
4. Change `getNoteRaw` to call the same raw reader.
5. Keep frontend unchanged.

This removes full-vault raw markdown storage while preserving `note.rawMarkdown` for the copy button.

### Phase 3 — Search document decoupling

Before persistent bleve, decouple search from HTML:

1. Add a `SearchDocument` type.
2. Generate `SearchDocument.Body` from markdown/plain text, not `stripHTML(note.HTML)`.
3. Change `search.Index` to index `SearchDocument`, not `*vault.Note`.
4. Keep the in-memory index temporarily and measure again.

This reduces build-time transient allocations and prepares for lazy HTML later.

### Phase 4 — Persistent bleve, but per-snapshot and closeable

Only after Phases 1–3:

1. Add `Index.Close()`.
2. Add `NewPersistentFresh` or `BuildPersistentSnapshotIndex` that always builds a clean index in a staging dir.
3. Add `Snapshot` with `IndexDir` and revision ID.
4. Swap snapshots atomically.
5. Close old index handles and garbage-collect old index dirs.

Acceptance:

- Deleted notes do not appear in search after reload.
- Search result slugs all resolve in the same snapshot's vault.
- Repeated reloads do not leak file descriptors.
- Pod survives reload under the target memory limit.

### Phase 5 — Optional: lazy HTML / LRU cache

Only if measurements still justify it. HTML is user-facing and tied to wiki-link resolution, image rewriting, SSR, and SEO. Do not do this before fixing raw markdown and search documents.

Possible shape:

```go
type Renderer struct {
    cache *lru.Cache[string, RenderedHTML]
}

func (r *Renderer) RenderNote(snap *Snapshot, note *vault.Note) (string, error) {
    key := snap.Revision + ":" + note.Slug + ":" + note.ModTime.String()
    if html, ok := r.cache.Get(key); ok { return html, nil }
    raw, _ := snap.Vault.ReadRaw(note.Path)
    html := parser.Render(raw, snap.Vault.ResolveWikiLink, snap.Vault.ResolveAssetURL)
    r.cache.Add(key, html)
    return html, nil
}
```

---

## 6. Review issues with concrete fixes

### Issue 1: Persistent index can return deleted notes

Problem: Existing `NewPersistent` re-indexes current notes but never deletes old IDs.

Where to look: `internal/search/search.go:55-77`; watcher delete path at `internal/watcher/watcher.go:114-123`.

Example:

```go
// Re-index all notes
for _, note := range v.AllNotes() {
    if err := si.Index(note); err != nil {
        return nil, err
    }
}
```

Why it matters: Search can show stale notes after a git push removes files.

Cleanup sketch: Build a fresh index per snapshot or implement `ReconcileIndex` that deletes missing document IDs.

---

### Issue 2: Persistent index handle has no lifecycle

Problem: `search.Index` has no `Close`; `RuntimeState.Reload` overwrites old indexes.

Where to look: `internal/search/search.go:26-30`, `internal/server/runtime.go:69-73`.

Example:

```go
s.mu.Lock()
s.vault = v
s.search = si
s.resolvedRoot = resolved
s.mu.Unlock()
```

Why it matters: Persistent bleve indexes need close semantics. Reloads can leak file descriptors/locks.

Cleanup sketch: Add `Index.Close()` and a snapshot lifecycle with delayed close/cleanup.

---

### Issue 3: Shared persistent path violates snapshot consistency

Problem: A stable `/data/search/index` path can be mutated during reload while old vault snapshots are still serving.

Where to look: `internal/server/runtime.go:39-44`, `internal/api/api.go:166-167`.

Example:

```go
v, _ := h.provider.Snapshot()
_, si := h.provider.Snapshot()
results, err := si.Search(q, 30)
```

Why it matters: Search and note lookup can disagree across reload boundaries.

Cleanup sketch: `Snapshot{Vault, Search, Revision, IndexDir}` built in staging dirs and swapped as a unit.

---

### Issue 4: Raw markdown storage removal breaks frontend unless response DTO is added

Problem: `rawMarkdown` is part of `Note` JSON and used by frontend copy-to-clipboard.

Where to look: `web/src/types/index.ts:26-38`, `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx:273-279`, `internal/api/api.go:124-150`.

Example:

```tsx
navigator.clipboard.writeText(note.rawMarkdown)
```

Why it matters: Removing the field breaks the UI.

Cleanup sketch: Store no raw markdown in `vault.Note`, but populate `NoteResponse.RawMarkdown` lazily in `getNote`; later change frontend copy action to fetch `/raw` on demand.

---

### Issue 5: Search indexing should not depend on rendered HTML

Problem: `search.Index` strips HTML to produce body text.

Where to look: `internal/search/search.go:92-98`, `internal/search/search.go:300-318`.

Example:

```go
Body: stripHTML(note.HTML),
```

Why it matters: This creates a large transient allocation per note and makes lazy HTML harder.

Cleanup sketch: Introduce `SearchDocument` generated from markdown/plain text and index that. Keep HTML as a UI concern.

---

### Issue 6: Health/config allocate full note slices

Problem: `len(v.AllNotes())` allocates a full slice just to count notes.

Where to look: `internal/server/server.go:157-163`, `internal/api/api.go:82-89`.

Example:

```go
len(v.AllNotes())
```

Why it matters: Small but needless allocation in hot/periodic paths.

Cleanup sketch: Add `Vault.Count()` and use it in health/config.

---

## 7. Concrete acceptance criteria for the takeover implementation

Do not consider the memory/search work complete until all of these pass:

1. **API compatibility:** Existing frontend still builds and `Copy as Markdown` works.
2. **Search correctness after deletion:** Delete a note, reload, search for a unique term from that note; it must not appear.
3. **Snapshot consistency:** During a reload, a search result slug must resolve in the same snapshot's vault.
4. **Reload survival:** Repeated `POST /api/admin/reload` calls under a production-sized vault do not exceed the memory budget.
5. **Index lifecycle:** Repeated reloads do not grow open file descriptors indefinitely.
6. **Instrumentation:** Logs identify memory usage before/after vault load, HTML rebuild, search build, and swap.
7. **Operational rollback:** The app can still run with in-memory search if persistent search is disabled.

Suggested test names:

```text
TestGetNoteKeepsRawMarkdownCompatibilityWithLazyStorage
TestPersistentIndexDoesNotReturnDeletedNotesAfterReload
TestReloadSnapshotKeepsSearchAndVaultConsistent
TestRepeatedReloadClosesOldIndexes
TestHealthDoesNotAllocateAllNotesForCount
```

---

## 8. Final recommendation

Use the original design as onboarding material, but do **not** implement its Phase 1 literally. The safe path is:

```text
hotfix limit -> measure -> lazy raw with response DTO -> search document split -> persistent per-snapshot index
```

This keeps the good parts of the current architecture — file-backed vaults, git-sync carry-over, atomic snapshot swaps — while avoiding the traps introduced by a naive persistent-index rollout.

The key mental model for the takeover team should be:

> The problem is not "we need a database." The problem is that one object (`vault.Note`) is being used as storage model, API DTO, search source, and render cache. Split those roles, preserve snapshot consistency, and only persist the search index once its lifecycle is explicit.
