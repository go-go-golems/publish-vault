# Tasks

## TODO

- [x] P1. Custom no-stopword analyzer in buildMapping (unicode + lowercase, no stop filter) on title/body/tags/excerpt; remove stopword-dropping standard analyzer.
- [x] P2. Add/extend search tests proving stopwords ("what"/"this"/"that") are indexed and returned; verify reindex path rebuilds the index with the new mapping.
- [ ] P3. Build via CI (publish-image), merge GitOps PRs for retro-obsidian-publish + obsidian-vault-publish (sha-<new>).
- [ ] P4. Verify live on parc.yolo.scapegoat.dev that "what"/"this"/"that" return results; confirm index rebuilt.
