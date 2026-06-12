# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created ORG-STYLE-011 with an intern-oriented implementation guide, diary, task list, and initial file relations.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/design-doc/01-org-style-alignment-analysis-and-implementation-guide.md — Primary implementation guide
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-06-12

Uploaded the initial ORG-STYLE-011 implementation guide bundle to reMarkable at /ai/2026/06/12/ORG-STYLE-011 after fixing a pandoc LaTeX escape failure in the diary prompt block.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/reference/01-investigation-diary.md — Updated after upload failure and success


## 2026-06-12

Step 3: Updated infra-tooling GHCR reusable workflow checkout actions from v5 to v6 (commit 63583aa).

### Related Files

- /home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml — Reusable GHCR workflow action pin modernization


## 2026-06-12

Step 4: Updated go-template baseline for Go 1.26.4, golangci-lint v2.12.2, fmt-check/ci-check, optional glazed-lint, and leaner pre-push hooks (commit 4427913).

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-template/.golangci.yml — Removed generic template's Glazed-specific staticcheck exclusion
- /home/manuel/code/wesen/go-go-golems/go-template/Makefile — Added fmt-check ci-check and optional glazed-lint targets
- /home/manuel/code/wesen/go-go-golems/go-template/go.mod — Updated template Go directive to 1.26.4
- /home/manuel/code/wesen/go-go-golems/go-template/lefthook.yml — Removed release build from pre-push hook


## 2026-06-12

Step 5: Updated publish-vault for Go 1.26.4, fmt-check/ci-check, optional glazed-lint, Glazed lint cleanups, raw endpoint errcheck/gosec annotations, and passing ci-check (commit e41b26d).

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/Makefile — Added fmt-check
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/cmd/retro-obsidian-publish/commands/build/web.go — Removed unnecessary Glazed output flags and adjusted env lookup
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/cmd/retro-obsidian-publish/commands/serve/serve.go — Removed unnecessary Glazed output flags and adjusted env lookup
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/go.mod — Updated Go directive to 1.26.4
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/internal/api/api.go — Checked raw markdown write and documented gosec G705 exception
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/internal/server/server.go — Adjusted reload token env lookup


## 2026-06-12

Step 6: Added a read-only org style audit helper and generated Markdown/JSON drift reports for Go and golangci-lint versions.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/scripts/01-audit-org-style.py — Read-only repository style audit helper
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/sources/01-org-style-audit.md — Generated human-readable audit report


## 2026-06-12

Step 7: Classified audit output into first safe batch, second safe batch, and manual-review repository groups.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/sources/02-org-style-batch-plan.md — Repository batch plan derived from audit output


## 2026-06-12

Maintained diary, changelog, task state, and docmgr validation through the first implementation wave; docmgr doctor passes.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/reference/01-investigation-diary.md — Detailed step-by-step implementation diary


## 2026-06-12

Uploaded the final ORG-STYLE-011 bundle including design guide, diary, tasks, audit report, and batch plan to reMarkable at /ai/2026/06/12/ORG-STYLE-011.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/sources/01-org-style-audit.md — Included in final reMarkable bundle
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/sources/02-org-style-batch-plan.md — Included in final reMarkable bundle


## 2026-06-12

Corrected scope to publish-vault and go-template only; reverted accidental infra-tooling change with commit 7e174d9 and removed org-wide audit artifacts.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/design-doc/01-org-style-alignment-analysis-and-implementation-guide.md — Updated design guide scope
- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/tasks.md — Updated task list to narrowed scope


## 2026-06-12

Re-uploaded the scope-corrected ORG-STYLE-011 bundle to reMarkable at /ai/2026/06/12/ORG-STYLE-011.

### Related Files

- /home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/ttmp/2026/06/12/ORG-STYLE-011--align-publish-vault-and-go-go-golems-repositories-with-current-org-style/design-doc/01-org-style-alignment-analysis-and-implementation-guide.md — Scope-corrected bundle source

