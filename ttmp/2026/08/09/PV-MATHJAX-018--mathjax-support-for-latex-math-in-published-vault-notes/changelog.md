# Changelog

## 2026-08-09

- Initial workspace created


## 2026-08-09

Step 1: Investigated the note pipeline end to end and wrote the intern-facing design/analysis/implementation guide; created 14 phased tasks. Key finding: math must be protected in a pre-goldmark pass mirroring replaceWikiLinks, and typeset client-side mirroring enhanceMermaid.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — Insertion point for the math pre-pass

