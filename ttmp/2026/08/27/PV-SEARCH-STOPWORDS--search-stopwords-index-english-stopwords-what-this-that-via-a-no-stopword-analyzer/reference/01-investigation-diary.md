---
title: Investigation diary
doc-type: reference
topics: [search, bleve, publish-vault, diary]
status: draft
intent: long-term
---

# Diary

## Goal

Make English stopwords (what/this/that/with/from/have/your/they/them) searchable on the `parc.yolo.scapegoat.dev` site (and the sibling `obsidian-vault-publish` site). The bleve `standard` analyzer drops them at index time, so they return 0 results. Replace it with a custom no-stopword analyzer, reindex, build, deploy, and verify live.

## Step P1: No-stopword custom analyzer in buildMapping

Replace bleve's `standard` analyzer (lowercase + English stop filter) with a custom `nostop` analyzer (unicode tokenizer + lowercase filter, no stop filter, no stemmer) on the `title`, `body`, `tags`, and `excerpt` text fields. The first attempt used `IndexMapping.AddCustomAnalyzer`, which is not enough — bleve validates the mapping against the global registry and rejected `nostop` ("no analyzer with name or type 'nostop' registered"). Fixed by registering the analyzer constructor via `registry.RegisterAnalyzer` in an `init()`, mirroring how bleve's own `standard` package registers itself.

### Prompt Context

**User prompt (verbatim):** "ok, do it"

**Assistant interpretation:** Implement the no-stopword analyzer fix for the stopword search problem (the agreed plan from the prior analysis).

**Inferred user intent:** Make "what"/"this"/"that" etc. searchable on the live site.

**Commit (code):** 6f211ce — "feat(search): index English stopwords via a no-stopword custom analyzer"

### What I did
- `pkg/search/search.go`: dropped the `analysis/analyzer/standard` import; `buildMapping` now sets `Analyzer = nostopAnalyzerName` on `titleField`, `bodyField`, `tagsField`, `excerptField`; removed the `AddCustomAnalyzer` block.
- `pkg/search/analyzer_nostop.go` (new): `registerNostopAnalyzer()` registers a `nostop` analyzer constructor via `registry.RegisterAnalyzer`, building `analysis.DefaultAnalyzer{Tokenizer: unicode, TokenFilters: [lowercase]}` (no stop filter, no stemmer); called from `init()`.
- `gofmt`, `go build ./pkg/search/`, `go vet` clean; `go test ./pkg/search/` — all existing tests pass (they were all failing with "no analyzer ... 'nostop' registered" before the registry fix).

### Why
- The stop filter is a web-scale ranking optimization (at billions of docs "the" matches almost everything), inappropriate for a ~1700-note personal vault where every word is a legitimate query and the index-size cost of keeping stopwords is negligible.
- No stemmer keeps exact token semantics (code identifiers, filenames); fuzziness on MatchQuery already covers typos.
- Aligns the Go bleve path with the static-vault client-side matcher (plain `includes`, no stopword logic), so the two search paths stop diverging.

### What worked
- After the registry fix, all 12 failing search tests passed on the first run; no behavioral test needed changes (none asserted stopword-dropping).

### What didn't work
- `IndexMapping.AddCustomAnalyzer` alone: it stores the analyzer name on the mapping but bleve still resolves it through the global registry at `bleve.New`/`NewMemOnly`, so it failed with "no analyzer with name or type 'nostop' registered" across all 12 search tests. The constructor must be `registry.RegisterAnalyzer`-registered, which the `standard` package does in its `init()`. Added the same pattern in `analyzer_nostop.go`'s `init()`.

### What I learned
- bleve custom analyzers need BOTH `registry.RegisterAnalyzer` (global, so the index can be created) — `AddCustomAnalyzer` on the mapping is insufficient on its own.
- The `standard` analyzer's stop filter is `en.StopName` (Snowball English list) applied as a token filter, not a separate analyzer option.

### What was tricky to build
- The registry indirection: the mapping references an analyzer *name*, and bleve resolves the name to a constructor in the global registry at index-creation time. The failure mode ("no analyzer ... registered") is the same for every test, which made it easy to misread as a single test setup issue rather than a global-registration issue.

### What warrants a second pair of eyes
- That the `nostop` analyzer is registered before any index is created in every code path (the `init()` in the `search` package runs at import; `pkg/search` is imported wherever an index is built). Tests confirmed this.
- That no field still uses `standard` (grep `standard\.` in pkg/search returns nothing).

### What should be done in the future
- Reconsider the `len(words[0]) <= 3` prefix-query special case in `textQueryClause`: now that stopwords are indexed it is no longer a source of inconsistency, but it is still an arbitrary cutoff. Out of scope here.
- Consider exposing analyzer choice as a config knob if other deployments want stopword removal. Out of scope here.

### Code review instructions
- `pkg/search/analyzer_nostop.go` (the `nostop` analyzer + `init()` registration).
- `pkg/search/search.go:buildMapping` (the four `Analyzer = nostopAnalyzerName` lines).
- `go test ./pkg/search/`.

### Technical details
- Analyzer chain: unicode tokenizer → lowercase filter. No stop filter, no stemmer.
- Fields using `nostop`: title, body, tags, excerpt. Keyword fields (tags_kw, path, path_kw, dates) unchanged.
