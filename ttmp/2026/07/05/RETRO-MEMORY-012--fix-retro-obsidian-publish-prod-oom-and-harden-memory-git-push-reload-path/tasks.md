# Tasks — RETRO-MEMORY-012

This checklist is the resumable execution plan after the takeover review. It intentionally **does not preserve API backwards compatibility where that would make the design worse**; the only requirement is that the shipped app still works end-to-end.

## Phase A — Planning, review, and baseline

- [x] A1. Create ticket workspace, intern guide, and investigation diary.
- [x] A2. Add second-pass project/design review correcting the naive persistent-index plan.
- [x] A3. Run baseline `go test ./...` before code changes.
- [x] A4. Commit planning docs and baseline state (`596286c`).

## Phase B — Memory/reload instrumentation (first code checkpoint)

Goal: make reload/startup memory measurable before changing storage semantics.

- [x] B1. Add phase-level memory logging around `loadVaultAndSearch`: before symlink resolution, after vault load, after search build, and after reload swap.
- [x] B2. Extend `/api/healthz` with Go memory stats (`heapAllocBytes`, `heapSysBytes`, `heapInuseBytes`, `numGC`, `nextGCBytes`) while keeping `ok`, `notes`, `vaultRoot`, and `configuredRoot`.
- [x] B3. Add `Vault.Count()` so health/config can count notes without allocating `AllNotes()` slices.
- [x] B4. Add/update tests for health JSON and `Vault.Count()`.
- [x] B5. Run `gofmt ./...` and `go test ./...`.
- [ ] B6. Commit Phase B.

## Phase C — Remove raw markdown from the hot storage model

Goal: stop retaining every note's source in memory and keep the app working by fetching raw content only when needed.

- [ ] C1. Remove `RawMarkdown` from `vault.Note`.
- [ ] C2. Add safe raw-file reading API on `Vault` (for example `ReadRaw(note.Path)`), reusing the same path-safety posture as asset serving.
- [ ] C3. Change `GET /api/notes/{slug}/raw` to read raw markdown from disk.
- [ ] C4. Change `GET /api/notes/{slug}` response shape to omit `rawMarkdown`.
- [ ] C5. Update frontend `Note` type to remove `rawMarkdown`.
- [ ] C6. Change the Copy as Markdown button to fetch `/api/notes/{slug}/raw` on demand instead of reading `note.rawMarkdown`.
- [ ] C7. Update SSR/tests/stories/static fixtures that still include or expect `rawMarkdown`.
- [ ] C8. Add backend tests for `/raw` success and deletion/missing-file handling.
- [ ] C9. Run Go tests and frontend type/build checks available in the repo.
- [ ] C10. Commit Phase C.

## Phase D — Decouple search indexing from rendered HTML

Goal: make search consume a dedicated search document instead of `stripHTML(note.HTML)`, reducing transients and preparing for lazy HTML.

- [ ] D1. Introduce a `search.Document` / `SearchDocument` type containing slug, title, plain body, tags, and excerpt.
- [ ] D2. Teach the parser/vault loader to produce or expose plain-text search body from markdown without depending on rendered HTML.
- [ ] D3. Change `search.Index` to index `SearchDocument` rather than `*vault.Note`.
- [ ] D4. Update watcher reload path to re-index the changed note's search document.
- [ ] D5. Add search tests that prove queries still match body, title, tags, and excerpts.
- [ ] D6. Run full tests.
- [ ] D7. Commit Phase D.

## Phase E — Persistent search index with explicit lifecycle and snapshot isolation

Goal: move bleve out of heap safely, without stale deleted documents or inconsistent vault/search snapshots.

- [ ] E1. Add `(*search.Index).Close()` and tests/idempotency expectations.
- [ ] E2. Replace or wrap current `NewPersistent` so production code never reuses a stale active index in place.
- [ ] E3. Introduce explicit `server.Snapshot` containing resolved root/revision, `*vault.Vault`, `*search.Index`, and optional index dir.
- [ ] E4. Build persistent indexes in per-revision staging directories and swap snapshot only after successful build.
- [ ] E5. Close old index handles after swap; clean old index directories safely.
- [ ] E6. Add `--search-index-path` / `SearchIndexPath` plumbing from CLI config into runtime.
- [ ] E7. Add tests: deleted note does not remain searchable after reload; repeated reload closes old indexes; search result slugs resolve in the same snapshot.
- [ ] E8. Run full tests.
- [ ] E9. Commit Phase E.

## Phase F — Deployment and operational hardening

Goal: ship safely to prod and make runtime behavior observable.

- [ ] F1. In hetzner-k3s, raise `app` limit to 3072Mi as an operational hotfix if not already done.
- [ ] F2. In hetzner-k3s, mount `/data/search` and set `--search-index-path` once Phase E is released.
- [ ] F3. Consider `GOMEMLIMIT` and `GOGC` only after measuring Phase B/E behavior.
- [ ] F4. Verify prod pod reaches `READY 3/3`, reload survives several git-sync cycles, and `/api/healthz` reports sane memory stats.
- [ ] F5. Commit deployment changes in hetzner-k3s separately.

## Phase G — Documentation, diary, and delivery upkeep

- [ ] G1. Keep `reference/01-investigation-diary.md` updated after every phase with commands, failures, and commit hashes.
- [ ] G2. Update `changelog.md` after every phase.
- [ ] G3. Run `docmgr doctor --ticket RETRO-MEMORY-012 --stale-after 30` after doc updates.
- [ ] G4. Re-upload the final doc bundle to reMarkable after implementation phases are complete.
