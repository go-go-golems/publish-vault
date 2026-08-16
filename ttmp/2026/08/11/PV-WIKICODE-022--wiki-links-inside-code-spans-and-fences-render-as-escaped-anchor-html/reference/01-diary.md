---
Title: Diary
Ticket: PV-WIKICODE-022
Status: active
Topics:
    - wiki-link
    - parser
    - html-rendering
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/math.go
      Note: skipCodeSpan/fenceOpensAt/skipFencedBlock — the scanners reused, and the indented-block decision inherited
    - Path: repo://internal/parser/parser.go
      Note: |-
        codeRegions, codeCursor, replaceWikiLinksOutsideCode, and the skip in extractWikiLinks
        extractWikiLinks splits frontmatter and shifts offsets; codeRegions consumes backslash escapes — PR #20 review fixes (commit daf23c3)
    - Path: repo://internal/parser/parser_test.go
      Note: 'regression tests for the two PR #20 review findings (commit daf23c3)'
ExternalSources: []
Summary: 'Wiki links inside code spans and fenced blocks were substituted before goldmark ran, so a note documenting the syntax rendered escaped anchor markup where its author wrote a wiki link — and gave the named note a phantom backlink. Fixed by reusing the code-region scanners the math pre-pass already applies. A follow-up commit addressed two PR #20 review findings: extractWikiLinks now detects code regions on the body only (so a frontmatter backtick cannot silently drop a body link from WikiLinks), and codeRegions no longer treats escaped backticks as code-span delimiters.'
LastUpdated: 2026-08-16T21:37:00Z
WhatFor: Review trail for PV-WIKICODE-022
WhenToUse: Read before changing the wiki-link pre-pass or the code-region scanners
---




# Diary

## Goal

Make `[[Some Note]]` inside a code span or a fenced code block render as the
text the author wrote, instead of as escaped anchor markup, and stop such a
sample from entering the backlink graph.

## Step 1: Find it while writing about it, then measure it

This defect was found while writing a report about [[PV-WIKILINK-021]] for the
vault. That report necessarily shows wiki-link syntax inside code blocks, and
rendering it through `publish-vault` produced walls of escaped `<a href="…">`
markup where the examples should have been. The defect is pre-existing and
unrelated to that ticket's changes; it has been in the renderer since the
wiki-link pre-pass was written.

The first job was to establish whether this was one note's problem or the
vault's. It is the vault's: 69 of 1790 notes are affected.

### Prompt Context

**User prompt (verbatim):** "new PR"

**Assistant interpretation:** Fix the code-span/fence linkification defect I had
just measured and flagged, and open it as a separate pull request rather than
adding it to the open PR #19.

**Inferred user intent:** Keep the wiki-link resolution PR reviewable at the
size it already is, while not leaving a newly measured rendering defect
unfixed.

**Commit (code):** 195da91 — "fix(PV-WIKICODE-022): leave wiki links inside code samples alone"

### What I did

- Wrote `scripts/01-code-region-repro`, a minimal note containing the syntax in
  an inline span, in a fence, and as a real link.
- Wrote `scripts/02-vault-code-leak-audit`, which renders every note in a vault
  and counts injected markup.
- Branched `task/wiki-links-in-code` off `task/publish-vault-mathjax` rather than
  off `main` — see below.

### Why

Because "it looked wrong in one note" is not a defect report. The audit turns it
into 341 occurrences across 69 notes, which is also what makes the fix worth
making rather than working around in the one note that surfaced it.

### What worked

The repro, before the fix:

```html
<p>A note refers to another as <code>&lt;a href=&quot;/note/some-note&quot; class=&quot;wiki-link&quot; data-target=&quot;some-note&quot; …&gt;Some Note&lt;/a&gt;</code>, or with a heading:</p>
```

It also showed a second consequence I had not measured: `WikiLinks` contained
`Target Note`, the note named in the fenced example. A code sample was giving
that note a backlink from a note that never linked to it — the same failure the
`Parse` comment already guards against for `[[Foo]]` appearing inside a formula.

### What didn't work

The audit's first version reported 5 residual occurrences after the fix and I
nearly wrote them up as a known limitation. They are false positives: two notes
quote the renderer's own output in an ```` ```html ```` block on purpose, and
that markup is supposed to be escaped and displayed.

The fix for the audit is a two-pass baseline. Each note is parsed twice, once as
written and once with every `[[` replaced by `⟦⟦`, which cannot open a wiki
link. Markup present in both renderings was written by the author; markup
present only in the first was injected by the pre-pass. That is an exact
discriminator rather than a heuristic on the markup's shape, and it took the
"after" number from 5 to 0 without weakening the check.

### What I learned

The measurement's first answer was wrong in the direction that would have
shipped: a small residual is easy to accept as an edge case and write into a
"known limitations" section. The discriminator that resolved it — render the
same input twice, with the mechanism under test disabled in one pass — is
generally applicable to any question of the form "did *my* pass produce this
output?".

### What was tricky to build

**Reusing the scanners rather than writing new ones.** `ScanMath` already skips
code spans and fenced blocks, because `$100` in a code sample is not math. Those
scanners (`skipCodeSpan`, `fenceOpensAt`, `skipFencedBlock`, `fenceClosesAt`)
are CommonMark-careful in ways that are easy to get wrong on a second attempt:
a closing fence must be at least as long as the opening one and carry nothing
after its run, so an info string inside a block does not terminate it early; a
code span does not cross a blank line; up to three spaces of indent still opens
a fence. Reusing them means the two pre-passes now agree about what code is,
which is a stronger property than either being individually correct.

**Replacement had to become index-aware.** `wikiLinkRegex.ReplaceAllFunc` has no
way to skip a match, so `replaceWikiLinksOutsideCode` walks `FindAllIndex` and
copies the gaps itself. `extractWikiLinks` moved from `FindAllSubmatch` to
`FindAllSubmatchIndex` for the same reason.

**Deliberately not handling indented code blocks.** `ScanMath` documents why:
a four-space indent inside a list is a continuation line, not code, so treating
it as code would silently drop links from nested list items — a worse and far
more common failure in a notes vault than a link rendering inside an indented
code block. The wiki-link pass inherits that decision, and the comment says so.

### What warrants a second pair of eyes

- **This is a behaviour change for existing notes.** 69 notes currently render
  anchor markup inside their code blocks and will render the wiki-link syntax
  instead. That is the intent, and it matches what Obsidian shows, but it is a
  visible change to published pages.
- **`codeRegions` scans the buffer its caller passes.** For `replaceWikiLinks`
  that is the body with frontmatter split off; for `extractWikiLinks` it is the
  whole source. A backtick inside a frontmatter scalar therefore opens a code
  span for extraction purposes only. The effect is to skip a `[[X]]` inside a
  quoted frontmatter value, which is desirable, but the asymmetry is worth
  knowing about.
- **`codeCursor` assumes ascending queries.** Both callers walk regex matches in
  order, which regexp guarantees, but a third caller iterating differently would
  silently get wrong answers.

### What should be done in the future

- The static TypeScript vault (`web/src/vault/staticVault.ts`) has the same
  defect and no code-region scanner at all: its `preprocessWikiLinks` runs a
  bare `content.replace(/\[\[([^\]]+)\]\]/g, …)` over the whole document. It is
  the demo build, and `marked` would need a tokenizer-level fix rather than a
  pre-pass, so it is filed rather than fixed.
- `extractWikiLinks` scans the whole source including frontmatter, so a
  `[[X]]` in a frontmatter value still enters `WikiLinks` and the backlink
  graph. Pre-existing, unrelated to code regions, and arguably correct for a
  `related:` list — but it is not a decision anyone made.

### Code review instructions

- `internal/parser/parser.go` — `codeRegions`, `codeCursor`,
  `replaceWikiLinksOutsideCode`, and the skip at the top of `extractWikiLinks`'s
  loop.
- `internal/parser/math.go` — the scanners being reused, unchanged by this
  ticket.
- `go test ./internal/parser/... -count=1 -run 'WikiLinksInsideCode|CodeRegionsBoundaries'`
- `go run ./ttmp/2026/08/11/PV-WIKICODE-022--*/scripts/01-code-region-repro`
- `go run ./ttmp/2026/08/11/PV-WIKICODE-022--*/scripts/02-vault-code-leak-audit -vault /home/manuel/code/wesen/go-go-golems/go-go-parc`
  — must print `injected markup: 0`.
- Before/after: `git checkout HEAD~1 -- internal/parser/parser.go`, re-run the
  audit, then `git checkout HEAD -- internal/parser/parser.go`.

### Technical details

Measured on the go-go-parc vault, 1790 notes:

| | before | after |
|---|---|---|
| injected markup occurrences | 341 | 0 |
| notes affected | 69 | 0 |

The branch was cut from `task/publish-vault-mathjax` (PR #19) rather than from
`main`, because PR #19 changed the signatures of both functions this ticket
edits — `extractWikiLinks` and `replaceWikiLinks` each gained a `[]MathSpan`
parameter — so a branch off `main` would have conflicted on exactly the lines
being changed here.

PR #19 merged while this work was in progress, which removed the question: the
branch was rebased onto `main` and the pull request targets `main` directly.
Creating the stacked pull request failed with `No commits between
task/publish-vault-mathjax and task/wiki-links-in-code, Base ref must be a
branch`, which is what a deleted base branch looks like from `gh` — the merge
had deleted it. Worth recognising, because the message does not say so.

## Step 2: Address the two P2 review findings on PR #20

PR #20 drew a Codex review with two P2 comments, both on the code-region
pre-pass this ticket added. Each is the same class of failure the ticket was
written to stop — a wiki link that is rendered (or not) but silently dropped
from `ParsedNote.WikiLinks`, so the backlink graph is wrong — just reached by a
different path than the code-sample case Step 1 fixed.

The first finding is an asymmetry Step 1's diary flagged as "worth knowing
about" but did not treat as a bug: `extractWikiLinks` built its code cursor over
the whole source while `replaceWikiLinks` split frontmatter off first. A
backtick inside a frontmatter scalar therefore paired with a body backtick and
swallowed the body link between them — but only for extraction, so the link
still rendered as an anchor while vanishing from the backlink graph. The second
finding is that `codeRegions` treated escaped backticks as code-span
delimiters, so a link written between two escaped backticks was wrapped in a
code span and skipped by both passes.

### Prompt Context

**User prompt (verbatim):** "Address code review issues: https://github.com/go-go-golems/publish-vault/pull/20"

**Assistant interpretation:** Read the inline review comments on PR #20, fix
the two P2 issues they raise in `internal/parser/parser.go`, add regression
tests, and keep the ticket's diary and bookkeeping current.

**Inferred user intent:** Get PR #20 to a state where the review findings are
resolved and the change is safe to merge, without expanding scope beyond what
the review asked for.

**Commit (code):** daf23c3 — "fix(PV-WIKICODE-022): address PR #20 review findings (2x P2)"

### What I did

- **Finding 1 (frontmatter code regions):** `extractWikiLinks` now splits
  frontmatter off and builds the code cursor on the body only, matching
  `replaceWikiLinks`. Match offsets from the whole-source regex are shifted by
  `len(frontmatter)` before the `code.contains` check; a match in frontmatter
  (`m[0] < fmLen`) is never code and is not even queried, so the cursor's
  ascending-query invariant still holds.
- **Finding 2 (escaped backticks):** `codeRegions` gained a backslash branch
  that consumes `\` + the next byte, mirroring `ScanMath`'s `case '\\'`. The
  only byte `codeRegions` acts on (other than fences at line start) is `` ` ``,
  so the only observable effect is that a backtick right after a backslash no
  longer opens a code span.
- Added regression tests: `TestFrontmatterBacktickDoesNotSwallowBodyLink`,
  `TestFrontmatterWikiLinkStillIndexed`, `TestEscapedBackticksKeepLinkInBacklinkGraph`,
  and two new cases in `TestCodeRegionsBoundaries` (escaped backticks; even
  backslashes before a backtick still open a span).
- Verified each new test fails against the pre-fix parser by stashing
  `parser.go`, running the tests, and restoring.

### Why

Both findings are silent data loss in the backlink graph, which is the exact
failure mode this ticket exists to prevent. The fix for finding 1 also removes
the asymmetry Step 1's diary called out — the two passes now agree about what
counts as code everywhere, not just in the body — which is the stronger
property the pre-pass relies on.

### What worked

- The narrow reading of finding 1 — exclude frontmatter from *code-region
  detection*, not from wiki-link *matching* — keeps the 123 go-go-parc notes
  that put `[[X]]` in frontmatter "related:" lists on the backlink graph. A
  broader fix (extract body-only) would have removed those backlinks, which is
  the separately-filed question (task dmoh) and not what the review asked.
  `TestFrontmatterWikiLinkStillIndexed` pins that decision.
- Reusing `ScanMath`'s backslash-consumption rule verbatim in `codeRegions`
  makes the "two passes agree about escaped backticks" invariant
  self-evident, the same way reusing `skipCodeSpan`/`fenceOpensAt` did in
  Step 1.
- The vault audit still reports `injected markup: 0 occurrences across 0
  notes`, so the Step 1 fix is intact.

### What didn't work

- A first draft of the `codeRegions` doc comment used inline backtick
  sequences (`` `\` ``) to show "an escaped backtick"; the edit tooling turned
  the backticks into smart double-quotes, producing `` `\“ `` in the file. Go
  still compiled it (comments carry any bytes), but it was unreadable. Rewrote
  both the `codeRegions` and the escaped-backtick test comments in plain prose
  with no inline backtick characters, and grepped the changed files for
  U+201C/U+201D to confirm none remain.

### What I learned

- The "two passes agree about what counts as code" property from Step 1 is
  fragile in a useful way: it surfaces exactly the places a third behaviour
  (here, backslash escapes) is handled in one pass but not the other. Finding
  2 was already handled correctly by `ScanMath` (its `case '\\'` consumes the
  escaped byte before the backtick branch ever runs); `codeRegions` had no such
  branch, so the passes disagreed. Copying the branch over fixed both the bug
  and the disagreement in one move.
- A review comment's title can be a precise spec. "Exclude frontmatter when
  detecting code regions" is about *detection*, not *matching* — following it
  literally kept the fix narrow and avoided a 123-note behaviour change the
  review did not request.

### What was tricky to build

**The offset shift in `extractWikiLinks`.** The regex still runs over the
whole source (so frontmatter `[[X]]` stays matched), but `codeRegions` is now
body-relative. A match in frontmatter has `m[0] < len(frontmatter)`, so the
shift `m[0] - fmLen` would go negative; guarding with `m[0] >= fmLen &&` keeps
frontmatter matches out of the cursor entirely. The cursor's `contains`
advances an internal index assuming ascending queries — frontmatter matches
arrive first and are skipped without querying, so the first body query still
sees the cursor at index 0 and the ascending order is preserved. Getting that
ordering argument right is what makes the shift safe rather than a latent
off-by-one for a future caller that iterates differently (the same caveat
Step 1 recorded about `codeCursor`).

**Proving the backslash branch has no other effect.** `codeRegions` walks
byte-by-byte and only acts on `` ` `` (and fences at line start). Consuming
`\` + next for every backslash therefore changes exactly one thing: a `` ` ``
immediately after a `\` is no longer an opener. A backslash before any other
byte would have fallen through to `i++` anyway, and the line-start fence check
runs before the backslash branch, so fence detection is unaffected — including
after a `\<newline>` hard break, where `body[i-1]` is still the `\n` after the
consume. `TestCodeRegionsBoundaries/even_backslashes_before_a_backtick_still_open_a_code_span`
pins that real code spans after an even backslash run still open.

### What warrants a second pair of eyes

- **`extractWikiLinks` still extracts `[[X]]` from frontmatter.** This is
  deliberate (task dmoh is the filed question of whether it should), and
  `TestFrontmatterWikiLinkStillIndexed` pins it, but a maintainer who believes
  frontmatter links should not backlink will want to flip that test and close
  dmoh. The fix here does not decide that question either way.
- **The offset shift assumes frontmatter is a prefix.** `splitFrontmatter`
  only recognises a leading `---` block, so `len(frontmatter)` is the offset
  of the body in `src`. If `splitFrontmatter` ever learned to strip a leading
  BOM or shebang, `fmLen` would still be correct because it is the *byte length
  of the split-off frontmatter*, not a hardcoded constant — but the body
  returned must stay byte-identical to the body `replaceMathInBody` computed
  the math spans on, or the spans' offsets would drift. Both call
  `splitFrontmatter`, so they agree today.

### What should be done in the future

- Decide task dmoh: should a `[[X]]` in a frontmatter value enter the backlink
  graph? 123 go-go-parc notes do this today. If "no", drop frontmatter from
  `extractWikiLinks`'s regex range and update
  `TestFrontmatterWikiLinkStillIndexed`; if "yes", close dmoh as intended and
  leave the test as the spec. Either way it is now a deliberate decision, not
  the accident the diary previously called out.
- The static TypeScript vault (`web/src/vault/staticVault.ts`) still has the
  underlying code-sample defect from Step 1 and now also lacks these two
  fixes; it remains filed (task 1rvt).

### Code review instructions

- `internal/parser/parser.go` — `extractWikiLinks` (frontmatter split + offset
  shift) and `codeRegions` (backslash branch).
- `internal/parser/parser_test.go` — the three new tests and the two new
  `TestCodeRegionsBoundaries` cases.
- `go test ./internal/parser/... -count=1` — all green.
- To confirm the tests pin the bugs: `git stash push -- internal/parser/parser.go`,
  re-run the four targeted tests (three fail), `git stash pop`.
- `go run ./ttmp/2026/08/11/PV-WIKICODE-022--*/scripts/02-vault-code-leak-audit -vault /home/manuel/code/wesen/go-go-golems/go-go-parc`
  — must print `injected markup: 0 occurrences across 0 notes`.

### Technical details

Codex review comments (PR #20, commit 2cb4ec0, `chatgpt-codex-connector[bot]`):

1. `internal/parser/parser.go:144` — "Exclude frontmatter when detecting code
   regions": a frontmatter backtick pairs with a body backtick and the
   intervening body link is omitted from `ParsedNote.WikiLinks` while still
   rendered by `replaceWikiLinks`.
2. `internal/parser/parser.go:318` — "Do not treat escaped backticks as code
   spans": `\`[[Target]]\`` is wrapped in a code span, so both passes skip it
   and the backlink is absent.

Both are labelled P2. No other review comments were left on the PR.

The narrow-vs-broad choice for finding 1, measured on the go-go-parc vault
(1836 notes): 123 notes put `[[...]]` in frontmatter. The narrow fix
(code-region detection only) leaves those 123 notes' backlinks unchanged; a
broad fix (extract body-only) would have removed them.
