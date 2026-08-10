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


## 2026-08-10

Step 4: found that goldmark's auto heading IDs and parser.Slugify disagree on most real headings (goldmark deletes '.' and dashes where slugify hyphenates them), so a [[#Heading]] fragment cannot be computed — it has to be read back out of the rendered HTML

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — parser.WithAutoHeadingID() vs slugify — the two algorithms


## 2026-08-10

Step 5: [[#Heading]] now renders as a same-page anchor carrying the heading as its text, with the fragment resolved against the ids goldmark actually emitted (Obsidian matching rules, first heading wins, visibly broken when unmatched). Same-note links no longer enter WikiLinks. On the Pattern Zoo note: 24 invisible /note/# anchors before, 24 working anchors and 0 dangling after (commit b620b39)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — resolveSelfHeadingLinks and the target-less branch of wikiLinkHTML
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/web/src/vault/staticVault.ts — wikiLinkLabel falls back to the heading so the link is visible in the static build

