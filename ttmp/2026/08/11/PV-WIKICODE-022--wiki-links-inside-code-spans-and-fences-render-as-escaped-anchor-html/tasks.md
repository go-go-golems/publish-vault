# Tasks

## TODO

- [x] Skip code spans and fenced blocks in extractWikiLinks and replaceWikiLinks <!-- t:y7w4 -->
- [x] Vault-wide audit distinguishing injected markup from author-written HTML samples <!-- t:d5ig -->
- [ ] FOLLOW-UP: web/src/vault/staticVault.ts has the same defect and no code-region scanner (preprocessWikiLinks runs a bare regex over the whole document) <!-- t:1rvt -->
- [ ] FOLLOW-UP: extractWikiLinks scans frontmatter too, so a [[X]] in a frontmatter value still enters the backlink graph <!-- t:dmoh -->
