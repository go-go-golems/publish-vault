# Changelog

## 2026-06-08

- Initial workspace created


## 2026-06-08

Step 1: Investigated codebase, created ticket and design doc for favicon support

### Related Files

- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/ttmp/2026/06/08/FAVICON-001--add-per-site-favicon-ico-support-and-create-go-go-parc-favicon/design-doc/01-favicon-per-site-support-design-and-implementation-guide.md — Comprehensive design doc


## 2026-06-08

Step 2: Implemented favicon handler, CLI flag, router wiring, tests, HTML link tags, and go-go-parc favicon (commits 330838a, 15daa3c)

### Related Files

- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/internal/server/favicon.go — New favicon handler
- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/internal/server/favicon_test.go — 7 unit tests


## 2026-06-08

Step 3: Addressed PR #5 review comments for favicon path safety, bundled fallback, and extension matching (commit eea8482)

### Related Files

- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/internal/server/favicon.go — Whitelisted favicon paths
- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/internal/server/favicon_test.go — Regression tests for review comments
- /home/manuel/workspaces/2026-06-08/publish-vault-favicon/publish-vault/internal/web/static.go — PublicFileExists helper for safe bundled favicon fallback

