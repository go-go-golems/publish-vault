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

