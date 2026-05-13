# Changelog

## 2026-05-13

- Initial workspace created


## 2026-05-13

Created initial assessment guide and investigation diary for Glazed, web/pnpm, Dagger, and devctl migration.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/13/RETRO-SETUP-001--retro-obsidian-publish-initial-assessment-and-setup-plan/design-doc/01-initial-assessment-and-setup-implementation-guide.md — Primary deliverable
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/13/RETRO-SETUP-001--retro-obsidian-publish-initial-assessment-and-setup-plan/reference/01-investigation-diary.md — Investigation record


## 2026-05-13

Implemented web/ migration, single embedded Go binary, Glazed command tree, Dagger-backed build web verb, devctl setup, and README cleanup (commit df40b4e).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/cmd/retro-obsidian-publish/main.go — New single-binary CLI entrypoint
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/web/static.go — SPA handler for embedded web assets
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/plugins/retro-obsidian-publish.py — devctl plugin


## 2026-05-13

Uploaded final implementation-complete ticket bundle to reMarkable.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/13/RETRO-SETUP-001--retro-obsidian-publish-initial-assessment-and-setup-plan/tasks.md — Marked final upload complete

## 2026-05-13

Completed Phase 7 items 1-4: reviewed migration, fixed watcher/search sync, added parser/API/SPA/CLI/devctl tests, and removed Vite analytics placeholder warnings (commit 8f865db).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/watcher/watcher.go — Watcher now updates search index on reload/remove
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/web/static_test.go — SPA fallback and API exclusion tests
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/index.html — Removed analytics placeholders causing Vite warnings


## 2026-05-13

Completed Phase 8: verified Docker build/runtime, fixed CGO-enabled container build, added .dockerignore, documented generated asset policy, and added CI workflow (commit cf6c8a4).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.dockerignore — Docker context pruning and generated asset policy
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/ci.yml — CI workflow for web/backend/plugin/embed/docker validation
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/Dockerfile — CGO-enabled single-binary Docker build


## 2026-05-13

Fixed embedded frontend data mode so same-origin /api is the default and static demo mode requires VITE_STATIC_VAULT=true.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/store/vaultApi.ts — Frontend API/static mode detection fix

