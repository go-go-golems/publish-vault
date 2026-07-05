# Changelog

## 2026-07-05

- Initial workspace created


## 2026-07-05

Created RETRO-MEMORY-012: confirmed prod OOMKilled app container (exit 137, 236 restarts) at 1536Mi limit; root cause is 3-fold in-memory redundancy (RawMarkdown + HTML + bleve NewMemOnly) amplified by Reload transient 2x memory. Wrote intern design doc with phased plan (Phase 0 ops limit raise; Phase 1 persist bleve index; Phase 2 lazy RawMarkdown) and documented the git-sync -> /api/admin/reload carry-over.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/ttmp/2026/07/05/RETRO-MEMORY-012--fix-retro-obsidian-publish-prod-oom-and-harden-memory-git-push-reload-path/design/01-prod-oom-memory-and-reload-architecture-guide.md — Primary intern deliverable
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/ttmp/2026/07/05/RETRO-MEMORY-012--fix-retro-obsidian-publish-prod-oom-and-harden-memory-git-push-reload-path/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-07-05

Added second-pass project/design review for memory handling, index building, and search. Review confirms the OOM diagnosis but corrects the implementation plan: do not use search.NewPersistent as a drop-in; handle stale deletes, Index.Close lifecycle, per-snapshot index dirs, rawMarkdown API compatibility, and search-document decoupling before persistent search rollout.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/ttmp/2026/07/05/RETRO-MEMORY-012--fix-retro-obsidian-publish-prod-oom-and-harden-memory-git-push-reload-path/design/02-project-and-design-review-memory-index-search.md — Second-pass project/design review


## 2026-07-05

Replaced the initial coarse task list with a resumable phase-by-phase implementation plan after takeover review. The new plan explicitly drops backwards-compatibility as a requirement where it simplifies the design, and sequences work as: instrumentation, raw markdown removal + frontend update, search document split, persistent per-snapshot index, deployment hardening, and diary/doc updates.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/ttmp/2026/07/05/RETRO-MEMORY-012--fix-retro-obsidian-publish-prod-oom-and-harden-memory-git-push-reload-path/tasks.md — Resumable phase/task checklist


## 2026-07-05

Implemented Phase B memory/reload instrumentation: added phase-level memory logs around load/reload, extended /api/healthz with Go heap stats, added Vault.Count() to avoid AllNotes allocation for counts, and added focused tests. Validation: gofmt and go test ./... passed.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/runtime.go — Memory stat helpers and load/reload phase logging
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/runtime_test.go — Health memory stats regression test
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/server.go — /api/healthz now returns heap stats and uses Vault.Count()
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault.go — Added Count() for non-allocating note counts
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault_test.go — Vault.Count() regression test


## 2026-07-05

Implemented Phase C raw markdown removal: removed RawMarkdown from vault.Note, added Vault.ReadRaw() for safe on-demand source reads, changed /api/notes/{slug}/raw to read from disk, intentionally removed rawMarkdown from full note JSON, and updated frontend copy-as-markdown to fetch /raw on demand. Validation: go test ./..., pnpm check, and pnpm build passed after installing web deps.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/api/api.go — /raw endpoint now reads source from disk; full note JSON omits rawMarkdown
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/api/api_test.go — Raw endpoint and omitted rawMarkdown regression tests
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault.go — Removed RawMarkdown from hot Note model and added ReadRaw
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Copy as Markdown fetches raw source on demand
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/web/src/types/index.ts — Removed rawMarkdown from Note type


## 2026-07-05

Implemented Phase D search-document split: added parser.PlainText, introduced vault.SearchDocument built from raw Markdown on demand, changed search indexing to consume SearchDocument bodies instead of stripHTML(note.HTML), and updated watcher incremental indexing accordingly. Validation: gofmt and go test ./... passed.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/parser/parser.go — Added PlainText helper for search body generation
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/search/search.go — Index now consumes vault.SearchDocument rather than stripping rendered HTML
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault.go — Added SearchDocument/SearchDocuments built from raw Markdown
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/vault/vault_test.go — Search document plain-Markdown regression test
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/watcher/watcher.go — Watcher reload path builds a search document before re-indexing changed notes


## 2026-07-05

Implemented Phase E persistent per-snapshot search indexes: added search.Index.Close(), made NewPersistent rebuild clean indexes to avoid stale deleted documents, introduced server.Snapshot with per-revision index dirs, added delayed old snapshot close/cleanup, plumbed --search-index-path through serve config, and added tests for stale deletes and cleanup. Real vault measurement: in-memory search build heapAlloc ~311MB; persistent startup heapAlloc ~80MB with ~52MB disk index; persistent reload peak heapAlloc ~144MB and returned 204.

### Related Files

- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/cmd/retro-obsidian-publish/commands/serve/serve.go — --search-index-path CLI flag
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/search/search.go — Closeable clean persistent bleve indexes
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/search/search_test.go — Close/stale-delete persistent index tests
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/runtime.go — Snapshot lifecycle
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/runtime_test.go — Persistent reload consistency and cleanup tests
- /home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/internal/server/server.go — SearchIndexPath runtime config

