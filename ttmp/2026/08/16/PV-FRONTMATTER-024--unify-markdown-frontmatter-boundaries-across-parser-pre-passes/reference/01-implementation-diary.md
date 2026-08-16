---
Title: Implementation Diary
Ticket: PV-FRONTMATTER-024
Status: active
Topics:
    - parser
    - frontmatter
    - regression
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/math.go
      Note: replaceMathInBody migrated to splitSource
    - Path: repo://internal/parser/parser.go
      Note: |-
        primary evidence source
        splitSource, sourceParts, migrated extractWikiLinks/replaceWikiLinks/stripFrontmatter; deleted splitFrontmatter and splitLine (commits a84d8fc, 9ce9644)
    - Path: repo://internal/parser/parser_test.go
      Note: TestSplitSourceMatrix and TestParseProtectsGoldmarkCompatibleFrontmatter regressions
    - Path: repo://ttmp/2026/08/16/PV-FRONTMATTER-024--unify-markdown-frontmatter-boundaries-across-parser-pre-passes/scripts/01-goldmark-frontmatter-contract/main.go
      Note: |-
        contract probe
        contract probe confirming all dash forms parse as metadata
ExternalSources: []
Summary: Investigation and design for unifying publish-vault frontmatter boundaries around one goldmark-meta-compatible source split.
LastUpdated: 2026-08-16T22:45:00Z
WhatFor: Handoff context for the colleague implementing PV-FRONTMATTER-024.
WhenToUse: Read before implementing or reviewing the frontmatter boundary fix.
---



# Implementation Diary

## Goal

Produce a colleague-ready implementation guide for the confirmed frontmatter boundary defect so a second engineer can implement it without rediscovering the diagnosis.

## Step 1: Confirm the defect and pin the goldmark contract

The architecture review (PV-MARKDOWN-023) had already demonstrated metadata mutation for four-dash frontmatter. This step verified the exact goldmark-meta v1.1.0 separator contract and confirmed which delimiter forms the configured `Parse` pipeline actually accepts as frontmatter.

### What I did

- Read `goldmark-meta@v1.1.0/meta.go:95–132`: `isSeparator` trims surrounding whitespace and requires one or more `-` bytes; opening is restricted to line zero; closing rejects blank lines.
- Read the duplicate splitters in `internal/parser/parser.go`: `splitFrontmatter` (exact `---`) vs `isFrontmatterDelimiter`/`stripFrontmatter` (any dash run).
- Created and ran `scripts/01-goldmark-frontmatter-contract`, which calls the real `Parse` pipeline against one-, two-, three-, four-dash, whitespace-wrapped, and CRLF delimiters.
- Confirmed all six forms parse as metadata in the configured engine.

### What worked

The probe eliminated the only uncertainty in the implementation guide. One- and two-dash delimiters win goldmark parser priority over Markdown list/thematic-break parsing for these inputs, so the regression matrix can include them without reservation.

### What I learned

`isFrontmatterDelimiter` already mirrors goldmark-meta exactly. The defect is purely the duplicate `splitFrontmatter`, not a wrong predicate. The fix is to stop having two splitters, not to invent a third rule.

### What was tricky

The temptation was to fold this into the larger typed AST/IR architecture. Keeping the fix narrow — one unexported structured split, migrate consumers, delete the duplicate — is what makes it safe to hand off and review independently.

### Code review instructions

```bash
go run ./ttmp/2026/08/16/PV-FRONTMATTER-024--*/scripts/01-goldmark-frontmatter-contract
```

All six cases should report `marker="preserved"`.

### Technical details

Probe output:

```text
one dash             title="Boundary" marker="preserved" html="<p>Body</p>\n"
two dashes           title="Boundary" marker="preserved" html="<p>Body</p>\n"
three dashes         title="Boundary" marker="preserved" html="<p>Body</p>\n"
four dashes          title="Boundary" marker="preserved" html="<p>Body</p>\n"
whitespace wrapped   title="Boundary" marker="preserved" html="<p>Body</p>\n"
four dashes CRLF     title="Boundary" marker="preserved" html="<p>Body</p>\n"
```

## Step 2: Introduce splitSource and pin the delimiter contract

The first implementation commit added the canonical boundary without changing any behavior. It introduced `sourceParts` and `splitSource`, migrated `StripFrontmatter` to delegate to it (the delimiter rule was already `isFrontmatterDelimiter`, so the behavior is byte-identical), deleted the now-unused `splitLine`, and added a 16-case unit matrix that pins the boundary against goldmark-meta's actual contract.

Keeping this commit behavior-neutral was deliberate: it let the unit matrix land while the math and wiki pre-passes still used the old `splitFrontmatter`, so a reviewer can read the boundary in isolation before any behavior changes.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Begin the test-first implementation of PV-FRONTMATTER-024 by introducing the canonical split and pinning its contract, without yet changing what the pre-passes do.

**Inferred user intent:** Land the new boundary in a small, reviewable, behavior-neutral commit before the behavior-changing migration.

**Commit (code):** a84d8fc — "refactor(PV-FRONTMATTER-024): add splitSource, the single frontmatter boundary"

### What I did

- Added `sourceParts` (frontmatter/body/bodyOffset) and `splitSource` in `internal/parser/parser.go`, reusing `isFrontmatterDelimiter` so there is exactly one delimiter predicate.
- Rewrote `stripFrontmatter` to `return splitSource(src).body` and deleted `splitLine`, which had no remaining callers.
- Added `TestSplitSourceMatrix` with 16 cases: no frontmatter, one/two/three/four dashes, whitespace-wrapped, CRLF, dashes inside quoted/block scalars, closing delimiter at EOF, body at EOF, unterminated opener, thematic break after a heading, opener with trailing non-dash content, empty input, and a single dash line with no newline.
- Each matrix case asserts `hasFrontmatter`, exact body bytes, byte-for-byte reconstruction (`frontmatter+body == src`), and `bodyOffset == len(frontmatter)`.

### Why

A behavior-neutral first commit lets the boundary be reviewed on its own merits. The unit matrix is the contract: if a later change breaks a delimiter form, the matrix fails before any `Parse`-level test does.

### What worked

- All 16 matrix cases passed on the first run.
- The existing `TestStripFrontmatterOnlyMatchesDelimiterLines` and `TestStripFrontmatterAgreesWithGoldmark` passed unchanged, confirming `splitSource` reproduces `stripFrontmatter`'s behavior exactly.
- `golangci-lint` reported 0 issues; `splitSource` is used (by `stripFrontmatter`), so the `unused` linter did not flag it.

### What didn't work

The pre-commit hook runs `make test` on every commit, so the strict TDD rhythm of "commit a red test, then commit the fix" would have been blocked. I adapted: the red regression test was written and verified to fail locally in Step 3, and only the green result was committed.

### What I learned

`bytes.SplitAfter(src, "\n")` keeps the trailing newline on each line, so `isFrontmatterDelimiter` (which `TrimSpace`s its input) accepts `"---\n"` and `"---\r"` identically. The CRLF case needed no special handling beyond what the delimiter predicate already did — the boundary is newline-agnostic because the predicate trims whitespace.

### What was tricky to build

The opening-line-at-EOF edge case. A source with no newline on its first line (`"---"`) must not be treated as an opener, because there is no second line to close the block. The `bytes.HasSuffix(lines[0], "\n")` guard handles this and preserves the previous `splitLine`-based behavior, where `splitLine` returned `ok=false` for a line with no newline. Matching that behavior exactly is what kept this commit byte-neutral.

### What warrants a second pair of eyes

The `bytes.HasSuffix(lines[0], "\n")` guard means a single-line file `"----"` is not frontmatter. goldmark-meta's `Open` checks position 0 and could in principle treat it as an opener, but with no closing line the block never closes, so the net effect (no metadata) agrees. Confirm with a `Parse` test if this ever matters.

### What should be done in the future

When the larger typed-document refactor lands, `sourceParts` is the internal basis to promote to an exported `SourceDocument`. The unit matrix then becomes the public contract test.

### Code review instructions

```bash
git show a84d8fc
go test ./internal/parser -count=1 -run TestSplitSourceMatrix -v
```

### Technical details

- `internal/parser/parser.go`: `sourceParts`, `splitSource`, rewritten `stripFrontmatter`, deleted `splitLine`.
- Matrix: 16 cases, all passing. Reconstruction invariant holds for every case.

## Step 3: Migrate the pre-passes and add the Parse regression

The second commit changed behavior. It migrated `replaceMathInBody`, `extractWikiLinks`, and `replaceWikiLinks` from the old exact-`---` `splitFrontmatter` to `splitSource`, deleted the now-dead `splitFrontmatter`, and added `TestParseProtectsGoldmarkCompatibleFrontmatter`. The regression was verified to fail against the pre-migration code before the migration landed, then to pass after it.

The four-dash preamble that goldmark-meta parses as metadata is now protected: it is no longer handed to the math or wiki pre-passes, so the parsed frontmatter holds the author's literal `[[Meta Link]]` and `$x^2$` instead of generated anchor HTML and a math sentinel. The frontmatter-link policy is unchanged: a frontmatter `[[X]]` still enters `WikiLinks` and can still produce a backlink, even though it is not rendered.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Complete the fix by migrating the remaining pre-pass consumers to the canonical boundary, removing the duplicate, and adding the `Parse`-level regression that demonstrates metadata is no longer mutated.

**Inferred user intent:** Land the actual behavior change that protects goldmark-compatible frontmatter, with a regression that proves it.

**Commit (code):** 9ce9644 — "fix(PV-FRONTMATTER-024): protect goldmark-compatible frontmatter from pre-passes"

### What I did

- Migrated `replaceMathInBody` (`internal/parser/math.go`) to `splitSource`, reassembling `parts.frontmatter` + masked body.
- Migrated `extractWikiLinks` to use `parts.body` and `parts.bodyOffset` (replacing the local `fmLen`), keeping the whole-source regex so frontmatter `[[X]]` is still matched and indexed.
- Migrated `replaceWikiLinks` to `splitSource`, reassembling frontmatter + replaced body.
- Deleted `splitFrontmatter` (no callers remain).
- Added `TestParseProtectsGoldmarkCompatibleFrontmatter` across six delimiter forms, asserting `related` and `formula` stay literal, no anchor HTML or math sentinel leaks into metadata, and the body wiki link and math still render.
- Verified the regression fails on the pre-migration code (three-dash passes; one/two/four/whitespace/CRLF fail with `related` = anchor HTML and `formula` = `\ue0000\ue001`), then passes after migration.

### Why

This is the actual defect: pre-passes and the metadata parser disagreed about the boundary. Migrating the pre-passes to the one boundary the metadata parser already uses removes the disagreement by construction, rather than by a test that checks it after the fact.

### What worked

- The regression's red state matched the defect precisely: only the three-dash form (the one the old `splitFrontmatter` recognized) passed; every other form mutated metadata. That is the strongest evidence the test pins the right behavior.
- The edge probe's four-dash case now prints `related="[[Meta Link]]"` and `title="Four Dashes"`, with `Meta Link` still in `WikiLinks` — the policy preserved alongside the fix.
- The full repository test suite (`go test ./...`) and `golangci-lint` are both clean.

### What didn't work

Nothing in this step. The migration was mechanical and the regression passed on the first run after it.

### What I learned

`extractWikiLinks` intentionally runs its regex over the whole source (so frontmatter links are indexed) while detecting code regions on the body only. The migration had to preserve that asymmetry: `parts.bodyOffset` is the threshold that decides whether a match offset is a body match (subject to code-region detection) or a frontmatter match (never code). Using `len(parts.frontmatter)` at the call site would have worked today, but `bodyOffset` is the named invariant the larger refactor will rely on.

### What was tricky to build

Keeping the frontmatter-link policy intact while fixing the boundary. The easy mistake would be to split the body and then run extraction only on the body, which would silently drop 123 vault notes' frontmatter `related:` backlinks. The test `TestFrontmatterWikiLinkStillIndexed` (from PR #20) guarded this, and I confirmed it still passes after the migration. The fix protects frontmatter bytes from being rewritten; it does not change which frontmatter bytes produce graph edges.

### What warrants a second pair of eyes

- `extractWikiLinks` still scans frontmatter for `[[X]]` and still indexes those links. Whether frontmatter links should backlink at all is the separately-filed `dmoh` question; this fix does not decide it.
- The math spans remain body-relative. `RestoreMath`/`RestoreMathText` use sentinel indices, not source offsets, so `bodyOffset` does not need to be added to `MathSpan.Start/End`. Confirm this if math ever gains source-span-aware restoration.

### What should be done in the future

- Decide the `dmoh` frontmatter-backlink policy in its own ticket; this fix deliberately leaves it unchanged.
- The next Phase 0 issue is ambiguity-aware resolution (the first-wins `wikiLinkIndex` populated from Go map iteration).

### Code review instructions

```bash
git show 9ce9644
go test ./internal/parser -count=1 -run 'Frontmatter|StripFrontmatter|SplitSource|ParseProtectsGoldmarkCompatibleFrontmatter'
go test ./... -count=1
go run ./ttmp/2026/08/16/PV-MARKDOWN-023--*/scripts/01-parser-edge-probe
```

The edge probe's `=== four-dash frontmatter delimiter ===` section must show `related="[[Meta Link]]"` and `Meta Link` in `links`.

### Technical details

Pre-migration regression output (red), for the four-dash case:

```text
related = "<a href=\"/note/meta-link\" class=\"wiki-link\" ...>Meta Link</a>"
formula = "\ue0000\ue001"
```

Post-migration (green):

```text
related="[[Meta Link]]"  formula via Frontmatter map  title="Four Dashes"
links = [{Meta Link ...}, {Body Link ...}]   # frontmatter link still indexed
html  = <p>Body <a ... data-raw="Body Link" ...>Body Link</a> with <span class="math math-inline">x</span>.</p>
```

Commits: a84d8fc (boundary + matrix), 9ce9644 (migration + regression + delete duplicate).
