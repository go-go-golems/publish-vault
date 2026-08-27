# Changelog

## 2026-08-27

- Initial workspace created


## 2026-08-27

P1: no-stopword custom analyzer (nostop) registered in bleve registry, set on title/body/tags/excerpt in buildMapping; existing search tests pass (commit 6f211ce)

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/analyzer_nostop.go — nostop analyzer (unicode+lowercase, no stop filter) registered via init()
- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/search.go — buildMapping sets nostopAnalyzerName on text fields


## 2026-08-27

P2: set nostop as index DefaultAnalyzer (the _all composite field still used standard); added TestStopwordsAreIndexed; verified live reindex via NewPersistentWithOptions (commit 194c000)

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/search.go — im.DefaultAnalyzer=nostop so _all field drops no stopwords
- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/search_test.go — TestStopwordsAreIndexed regression test


## 2026-08-27

P3: pushed main -> CI built sha-1d9c02d -> GitOps PRs #325 + #326 merged -> Argo synced -> both deployments on sha-1d9c02d; index rebuilt fresh on startup

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml — image bumped to sha-1d9c02d via GitOps PR #325


## 2026-08-27

P4: verified live on parc.yolo — what 1933, this 1919, that 1944, with 1948, from 1909, have 1429, your 1489, they 2000, them 1999 (all were 0); non-stopwords unchanged

