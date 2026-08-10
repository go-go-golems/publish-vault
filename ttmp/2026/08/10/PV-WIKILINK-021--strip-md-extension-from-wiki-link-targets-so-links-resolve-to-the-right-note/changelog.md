# Changelog

## 2026-08-10

- Initial workspace created


## 2026-08-10

Step 1: reproduced both failure modes with scripts/01-md-suffix-repro — a .md-suffixed target slugifies to a trailing -md key, which is normally a miss (#unresolved-, heading fragment and title both lost) and, when the vault also holds a note named '... md', a silent hit on the wrong note (commit e8ff03a)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — slugify turns the dot into a hyphen instead of dropping it


## 2026-08-10

Step 2: added parser.StripNoteExtension, called from parseWikiLinkInner so slug, href, display text, backlink graph and search excerpt all agree; ResolveWikiLink normalises too. Three new tests, verified failing against the pre-fix code (commit bfbcab4)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — The strip and its single call site
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/vault/vault.go — ResolveWikiLink normalises the target as written


## 2026-08-10

Step 3: same fix in the static TS resolver; validated on the real 09 - RAG-MATHS Pattern Zoo.md note — 92 unresolved link occurrences before, 0 after, and the outgoing-link graph stopped double-counting (46 -> 40 distinct links) (commit 2fb5955)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/web/src/vault/staticVault.ts — stripNoteExtension/wikiLinkLabel — titleToSlug deleted the dot rather than hyphenating it

