---
Title: Same-note heading links and why the fragment must be read back
Ticket: PV-WIKILINK-021
Status: active
Topics:
    - wiki-link
    - parser
    - html-rendering
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: wikiLinkHTML target-less branch, resolveSelfHeadingLinks, and the call site in Parse
    - Path: repo://web/src/components/organisms/NoteView/noteEnhancements.ts
      Note: enhanceHeadingAnchors makes heading ids shareable, which ruled out regenerating them
ExternalSources: []
Summary: '[[#Heading]] rendered as an empty anchor pointing at the vault root. Fixing it cannot mean computing the fragment: goldmark''s auto heading IDs and parser.Slugify disagree on most real headings. The fragment is resolved against the ids goldmark actually emitted, in a post-render pass.'
LastUpdated: 2026-08-10T00:00:00Z
WhatFor: Design for the [[#Heading]] fix
WhenToUse: Read before changing heading anchors, heading ids, or wiki-link fragments
---


# Same-note heading links and why the fragment must be read back

## Problem

A wiki link with no target — `[[#Pattern 1 — Semantic Identity]]` — names a
heading in the note it sits in. Obsidian renders it as an in-page jump. The
Pattern Zoo note's whole table of contents is written this way.

`wikiLinkHTML` sent it down the ordinary `/note/<slug>` path with an empty
target, producing:

```html
<a href="/note/#pattern-1-semantic-identity" class="wiki-link"
   data-target="" data-raw="" data-alias=""></a>
```

Two defects in one tag. The element content is empty, so the link is **invisible
on the page** — nothing to click, nothing to read. And `/note/` with no slug is
the vault root, so on the rare occasion someone did find it, it navigated away
from the note entirely.

24 such links in `Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md`.

## Why the fragment cannot be computed

The obvious fix is `href="#" + slugify(heading)`. It does not work, because
heading anchors are not made by `slugify`. Goldmark's `WithAutoHeadingID()`
generates them, and the two disagree: goldmark **deletes** punctuation it does
not want, `slugify` **replaces** it with `-`.

| heading | goldmark id | `slugify` |
|---|---|---|
| `Identity is an API decision` | `identity-is-an-api-decision` | same |
| `9.2 Kernel K0: canonical identity` | `92-kernel-k0-canonical-identity` | `9-2-kernel-k0-…` |
| `7.3 Domain-separated hashes` | `73-domain-separated-hashes` | `7-3-domain-…` |
| `Pattern 1 — Semantic Identity` | `pattern-1--semantic-identity` | `pattern-1-semantic-identity` |
| `Entity–Derivation–Observation Separation` | `entityderivationobservation-separation` | `entity-derivation-observation-separation` |

They agree only when the heading is letters, digits and spaces. Goldmark also
suffixes duplicate headings (`notes`, `notes-1`), which no stateless function
can reproduce.

`scripts/03-heading-id-divergence` prints this table from live output.

### Options considered

1. **Mirror goldmark's algorithm.** Keeps existing anchor URLs, but leaves two
   algorithms that must be kept in sync forever, and cannot reproduce the
   duplicate-heading counter without also reimplementing its state.
2. **Make goldmark use `slugify`** via a custom `parser.IDs`. One algorithm
   everywhere — but heading ids are a *published URL surface*:
   `enhanceHeadingAnchors` injects a copyable `#` permalink into every heading,
   so regenerating them breaks links people have already shared.
3. **Read the ids back out of the rendered HTML.** Exact by construction,
   immune to goldmark changing its algorithm, and gets duplicate headings right
   for free. Chosen.

## Fix

`wikiLinkHTML` grows a `target == ""` branch that emits a placeholder instead of
a `/note/` link, with the heading as fallback display text:

```html
<a href="#" class="wiki-link wiki-link-self" data-heading="…" data-alias="…">…</a>
```

`resolveSelfHeadingLinks` then runs over the rendered document, indexes every
`<h1-6 id="…">` by its stripped text, and rewrites each placeholder's href.

Matching rules, following Obsidian: heading text, case-insensitive, runs of
whitespace collapsed, **first** heading wins on duplicates. One addition that is
*not* Obsidian: if no heading text matches, the target is tried against the ids
directly, so `[[#some-heading]]` written in slug form still lands. It only fires
after the text match fails, so it cannot shadow a real heading.

Unmatched targets render `href="#unresolved-<slug>"` with a `broken` class —
already styled dotted-red by `prose.css` — and keep their text. Visibly broken
beats invisibly broken.

### Placement

The call sits in `Parse` between `renderCallouts` and `RestoreMath`. Not
arbitrary: math is lifted out before wiki links are extracted and restored after
every HTML pass, so in that window a heading and a link to it carry the *same*
math placeholders and still match each other. After `RestoreMath` one side would
have TeX and the other a placeholder.

Resolution belongs in `Parse` rather than the vault layer because it depends only
on the note itself. That also makes it survive `rebuildHTML`, which re-renders
from the parser output on every reload.

### Knock-on changes

- Target-less links no longer enter `ParsedNote.WikiLinks`. They were
  empty-target entries that could only fail to resolve, and showed as blank rows
  in the agent Markdown view. This changes the `wikiLinks` array in note JSON.
- Degenerate `[[#]]` is passed through as source text rather than becoming an
  empty anchor.
- The static TS vault cannot do any of this: `marked` v18 emits no heading ids,
  so that build has never supported heading fragments at all. Its
  `wikiLinkLabel` falls back to the heading so the link is at least visible, and
  it stays marked broken.

## Cross-note fragments

`[[Other Note#Heading]]` had the identical mismatch and was fixed in a second
pass — 84 of the 186 rendered cross-note fragments in the Pattern Zoo note
pointed at an id that does not exist, opening the right note at the top of the
page.

It could not be done here in `Parse`, which sees one note: the answer lives in a
different one. So `HeadingIndex` was factored out of this pass, the anchor gained
a `data-heading` attribute (the fragment alone is lossy — `#9-2-kernel-k0` no
longer says the heading was `9.2 Kernel K0`), and `ResolveWikiLinkHeadings` runs
in `rebuildHTML` once the slug is known. Heading indexes are built lazily per
target and cached for the pass.

Living in `rebuildHTML` also means a heading rename re-resolves every link
pointing at it on the next reload. A heading the target does not have drops the
fragment rather than leaving one known to dangle.

## Not fixed here

- `![[#Heading]]` and `![[Note#Heading]]` embeds ignore the heading entirely —
  `resolveEmbeds` injects the whole target note. Self-embeds still render as an
  empty invisible div.
- Block references `[[#^blockid]]` are unsupported; a same-note one renders
  visibly broken, a cross-note one drops its fragment.
- A heading containing math, linked from another note, does not resolve: math is
  lifted per note, so the two sides carry different placeholders. The fragment is
  dropped and the link still opens the note.
- The static TS vault emits no heading fragments at all, because marked v18
  emits no heading ids.

## Validation

```bash
go test ./internal/parser/... ./pkg/vault/... -count=1
go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/04-self-heading-links
go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/05-real-self-heading-check \
  -note "/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"
```

On the real note: 24 same-note links, **0 → 24** resolved, **24 → 0** rendered as
`/note/#…`, **24 → 0** anchors with no visible text, 0 dangling.
