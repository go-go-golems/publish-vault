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
      Note: codeRegions, codeCursor, replaceWikiLinksOutsideCode, and the skip in extractWikiLinks
ExternalSources: []
Summary: Wiki links inside code spans and fenced blocks were substituted before goldmark ran, so a note documenting the syntax rendered escaped anchor markup where its author wrote a wiki link — and gave the named note a phantom backlink. Fixed by reusing the code-region scanners the math pre-pass already applies.
LastUpdated: 2026-08-11T00:00:00Z
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

The branch is based on `task/publish-vault-mathjax` (PR #19) rather than on
`main`, and the pull request targets that branch. PR #19 changed the signatures
of both functions this ticket edits — `extractWikiLinks` and
`replaceWikiLinks` each gained a `[]MathSpan` parameter — so a branch off `main`
would conflict on exactly the lines being changed here. Stacking keeps this
diff to what it is actually about.
