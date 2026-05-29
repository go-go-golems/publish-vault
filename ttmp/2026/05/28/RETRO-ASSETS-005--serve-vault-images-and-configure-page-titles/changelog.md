# Changelog

## 2026-05-28

- Initial workspace created


## 2026-05-28

Created ticket, implementation guide, detailed tasks, and initial diary for vault image serving and configurable page titles.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/design-doc/01-image-serving-and-page-title-implementation-guide.md — Primary analysis and implementation guide
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/reference/01-diary.md — Initial investigation diary
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/tasks.md — Detailed phase checklist


## 2026-05-28

Implemented backend image URL rewriting, /assets serving, and pageTitle config (commit 7b2b8c6); implemented frontend document.title consumption (commit aff4713); validated with backend tests, frontend typecheck, build-web, and temp-vault smoke test.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/server/server.go — /assets handler and route registration
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go — Asset URL resolution in note HTML
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/reference/01-diary.md — Implementation diary updates
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/web/src/App.tsx — Document title effect


## 2026-05-28

Resolved docmgr vocabulary warnings for RETRO-ASSETS-005 topics and confirmed docmgr doctor passes.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/tasks.md — Checked final doctor task
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/vocabulary.yaml — Added assets/config/images/page-title topic vocabulary


## 2026-05-28

Fixed real-vault browser test regression: moved vault content assets from /assets/ to /vault-assets/ so Vite CSS/JS under /assets/ are served by the SPA handler (commit 2f9f40f).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/server/server.go — Route prefix changed to /vault-assets/
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go — Rendered vault image URLs changed to /vault-assets/
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/reference/01-diary.md — Recorded CSS/JS 404 diagnosis and fix


## 2026-05-28

Aligned CI/CD, lefthook, golangci-lint, gosec, and local Makefile checks with go-go-golems go-template standards, adapted for backend/ and web/ paths (commit eeb5b70).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/ci.yml — CI setup-node/pnpm and Makefile-backed checks
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/dependency-scanning.yml — govulncheck and gosec workflow
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/lint.yml — Standard golangci-lint workflow
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.golangci.yml — Template-derived lint config
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/Makefile — Standard quality targets
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/lefthook.yml — Active local hooks


## 2026-05-28

Addressed PR #2 review by validating resolved symlink targets before serving /vault-assets files and adding symlink escape regression tests (commit a46173b).

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/server/runtime_test.go — Symlink escape regression tests
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/server/server.go — Resolved symlink target validation in assetHandler
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/28/RETRO-ASSETS-005--serve-vault-images-and-configure-page-titles/reference/01-diary.md — Recorded PR review response

