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

## Step P2: Tests + reindex verification

Added a stopword-indexing test and discovered (and fixed) the second half of the bug: the per-field `nostop` analyzer was not enough because a `MatchQuery` with no `SetField` searches bleve's `_all` composite field, whose analyzer is the IndexMapping's `DefaultAnalyzer` — which bleve defaults to `standard` (with the stop filter). Set `im.DefaultAnalyzer` (and the document mapping's) to `nostop`. Verified the live deploy rebuilds the index fresh on startup, so no manual reindex is needed.

### Prompt Context

**User prompt (verbatim):** (see P1)

**Assistant interpretation:** Add tests proving stopwords are indexed/returned and verify the reindex path.

**Inferred user intent:** Lock the fix with a regression test and confirm the live index will pick up the new mapping on deploy.

**Commit (code):** 194c000 — "fix(search): set nostop as the index default analyzer + stopword index test"

### What I did
- `pkg/search/search_test.go`: added `TestStopwordsAreIndexed` — indexes a note containing `what/this/that/with/from` plus an unrelated note, and asserts each stopword is searchable (4+ chars → MatchQuery path, the one that exposed the bug) and that "what" matches the what-note.
- `pkg/search/search.go:buildMapping`: `im.DefaultAnalyzer = nostopAnalyzerName` and `dm.DefaultAnalyzer = nostopAnalyzerName`.
- `go test ./pkg/search/` and `go test ./...` green.

### Why
- A `MatchQuery` with no `SetField` searches the `_all` composite field. `_all`'s analyzer is the IndexMapping's `DefaultAnalyzer`, which bleve's `NewIndexMapping()` sets to `standard.Name` (bleve `mapping/index.go:37 const defaultAnalyzer = standard.Name`). So even with `nostop` on every text field, the query side still dropped stopwords via `_all`. Setting the default to `nostop` makes both indexing and querying consistent.
- The test pins the exact regression: 4+ char stopwords must be searchable via the MatchQuery path.

### What worked
- After setting `DefaultAnalyzer`, "what"/"this"/"that" passed immediately. "with"/"from" initially failed — because my first test note didn't contain them; fixed the note text. The test then caught a real content gap rather than a code gap, which is the right behavior.

### What didn't work
- First test note omitted "with"/"from"; the test correctly reported 0 for them, which looked like a code failure but was a test-data gap.

### What I learned
- bleve's `_all` field analyzer is the IndexMapping `DefaultAnalyzer`, not inherited from the per-field mappings. Any analyzer change must set the default too, or query-time `_all` searches keep the old analyzer.
- The live index is rebuilt fresh on every startup/vault-reload via `buildSearchIndex` → `search.NewPersistentWithOptions` (removes the build dir, calls `bleve.New(buildIndexDir, buildMapping())`), keyed by vault revision. So the new mapping takes effect on the next pod restart — no manual reindex command.

### What was tricky to build
- Distinguishing "the analyzer is wrong" from "the word isn't in the corpus." The test failure message for "with"/"from" was identical to "what"/"this" before the DefaultAnalyzer fix, so the root cause wasn't obvious until I checked the test note content.

### What warrants a second pair of eyes
- That `DefaultAnalyzer` is set on both the IndexMapping and the document mapping (belt and suspenders; the document-mapping default is what `defaultAnalyzerName` walks).
- That no other code path constructs an index without `buildMapping` (grep confirms `buildMapping` is the single mapping factory).

### What should be done in the future
- Reconsider the `<=3` prefix special case now that stopwords are indexed (no longer inconsistent, but still arbitrary). Out of scope.
- Consider an analyzer-config knob. Out of scope.

### Code review instructions
- `pkg/search/search_test.go:TestStopwordsAreIndexed`.
- `pkg/search/search.go:buildMapping` (the two `DefaultAnalyzer = nostopAnalyzerName` lines).
- `go test ./pkg/search/ -run TestStopwordsAreIndexed -v`.

### Technical details
- `_all` analyzer = IndexMapping.DefaultAnalyzer (defaults to "standard").
- Live reindex: `pkg/server/runtime.go:buildSearchIndex` → `search.NewPersistentWithOptions` (removes + rebuilds).
