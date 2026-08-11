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


## 2026-08-10

Step 6: cross-note [[Note#Heading]] fragments now resolve against the target note's own rendered heading ids. Anchors carry data-heading (the slugified fragment is lossy); ResolveWikiLinkHeadings runs in rebuildHTML after the slug is known, so a heading rename re-resolves inbound links on reload. Pattern Zoo note: 84 of 186 rendered fragments dangled before, 0 after (commit 1a91868)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — HeadingIndex factored out of the same-note pass, plus ResolveWikiLinkHeadings
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/vault/vault.go — rebuildHTML resolves fragments against the target, with a per-pass heading-index cache


## 2026-08-11

Step 7: addressed two P2 findings from the PR #19 review. (1) The vault walk accepts 'Note.MD' but pathToSlug/buildWikiLinkIndex trimmed only lowercase '.md', so the case-insensitive wiki-link strip regressed [[Note.MD]] from working to broken; StripNoteExtension is now the single definition used by the slug, index, title fallback and file tree. (2) Math sentinels in generated attributes were rewritten by RestoreMath into malformed markup; attributes now carry TeX via RestoreMathText, and resolveSelfHeadingLinks moved after RestoreMath because a heading and its link are separate math spans (commit c279a21)

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/math.go — RestoreMathText — sentinel to bare TeX, for attributes and JSON
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — attrTarget/attrAlias/attrHeading in wikiLinkHTML, and the moved self-heading pass
- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/vault/vault.go — pathToSlug, buildWikiLinkIndex, title fallback and FileTree all route through StripNoteExtension

