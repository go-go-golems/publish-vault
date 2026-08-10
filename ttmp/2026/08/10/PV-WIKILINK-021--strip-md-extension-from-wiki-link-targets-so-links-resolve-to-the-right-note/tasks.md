# Tasks

## TODO

- [x] Strip trailing .md in parseWikiLinkInner + parser unit tests <!-- t:1udf -->
- [x] Strip in vault.ResolveWikiLink + end-to-end vault test <!-- t:byb7 -->
- [x] Fix resolveWikiTarget/display text in web/src/vault/staticVault.ts <!-- t:n1ae -->
- [x] Run full go test ./... and the repro script as acceptance check <!-- t:db23 -->
- [x] FOLLOW-UP (separate bug, not fixed here): [[#Heading]] same-note links render as <a href="/note/#heading" ...></a> — empty display text, points at the vault root. Six occurrences in the Pattern Zoo ToC. <!-- t:z5hp -->
- [ ] FOLLOW-UP: web/ has no unit-test runner, so staticVault.ts is type-checked but not executed by any test; consider vitest <!-- t:kt1l -->
- [ ] FOLLOW-UP: cross-note [[Note#Heading]] fragments still use slugify and miss goldmark's heading ids — 8 of 28 dangle in the Pattern Zoo note (scripts/06-cross-note-fragment-audit). Needs a per-note heading-id index in the vault layer. <!-- t:ikll -->
- [ ] FOLLOW-UP: ![[#Heading]] self-embeds still render as an empty invisible div; static build has no heading ids at all (marked v18) <!-- t:olbu -->
