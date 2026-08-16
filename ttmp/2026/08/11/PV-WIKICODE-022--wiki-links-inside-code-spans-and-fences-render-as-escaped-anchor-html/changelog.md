# Changelog

## 2026-08-11

- Initial workspace created


## 2026-08-11

Skipped code spans and fenced blocks in the wiki-link pre-pass, reusing ScanMath's CommonMark scanners so both pre-passes agree about what code is. go-go-parc vault: 341 injected-markup occurrences across 69 of 1790 notes before, 0 after (commit 195da91)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — codeRegions, codeCursor, replaceWikiLinksOutsideCode


## 2026-08-16

Addressed two P2 PR #20 review findings: extractWikiLinks now detects code regions on the body only (frontmatter backtick no longer silently drops a body link from WikiLinks), and codeRegions no longer treats escaped backticks as code-span delimiters (mirroring ScanMath). Vault audit unchanged at 0 injected. (commit daf23c3)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — extractWikiLinks frontmatter split + offset shift; codeRegions backslash branch

