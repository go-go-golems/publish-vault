# Changelog

## 2026-08-16

- Initial workspace created


## 2026-08-16

Created the frontmatter boundary implementation guide: confirmed the goldmark-meta v1.1.0 separator contract with a Parse probe (all dash-only forms accepted), documented the defect, proposed one internal splitSource API, test-first commit sequence, acceptance criteria, review checklist, risks, and a ready-to-use PR description.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/ttmp/2026/08/16/PV-FRONTMATTER-024--unify-markdown-frontmatter-boundaries-across-parser-pre-passes/design-doc/01-implementation-guide-canonical-frontmatter-and-body-splitting.md — colleague-ready implementation guide
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/ttmp/2026/08/16/PV-FRONTMATTER-024--unify-markdown-frontmatter-boundaries-across-parser-pre-passes/reference/01-implementation-diary.md — investigation and goldmark contract confirmation


## 2026-08-16

Implemented PV-FRONTMATTER-024: introduced splitSource (the single goldmark-meta-compatible boundary), migrated replaceMathInBody/extractWikiLinks/replaceWikiLinks/StripFrontmatter to it, deleted the duplicate splitFrontmatter and the unused splitLine, and added TestSplitSourceMatrix (16 cases) + TestParseProtectsGoldmarkCompatibleFrontmatter (6 delimiter forms). Regression verified red pre-migration then green post-migration. Full suite + lint clean; frontmatter-link policy unchanged. (commits a84d8fc, 9ce9644)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/math.go — math pre-pass migration
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — canonical boundary and consumer migration

