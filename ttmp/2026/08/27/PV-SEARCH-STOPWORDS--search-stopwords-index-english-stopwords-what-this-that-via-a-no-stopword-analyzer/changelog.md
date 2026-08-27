# Changelog

## 2026-08-27

- Initial workspace created


## 2026-08-27

P1: no-stopword custom analyzer (nostop) registered in bleve registry, set on title/body/tags/excerpt in buildMapping; existing search tests pass (commit 6f211ce)

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/analyzer_nostop.go — nostop analyzer (unicode+lowercase, no stop filter) registered via init()
- /home/manuel/code/wesen/go-go-golems/publish-vault/pkg/search/search.go — buildMapping sets nostopAnalyzerName on text fields

