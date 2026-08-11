---
Title: Stripping the .md extension from wiki-link targets
Ticket: PV-WIKILINK-021
Status: active
Topics:
    - wiki-link
    - parser
    - vault
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: parseWikiLinkInner is the chosen choke point; slugify is why the dot becomes a hyphen
    - Path: repo://pkg/vault/vault.go
      Note: pathToSlug and buildWikiLinkIndex are the extension-less side of the disagreement
ExternalSources: []
Summary: 'Wiki-link targets written with a trailing .md slugify to a `-md` slug that does not match any note, so the link either dies as #unresolved-… or silently lands on an unrelated note. Fix: strip a trailing .md from the target once, in parseWikiLinkInner, before anything derives a slug, display text or backlink from it.'
LastUpdated: 2026-08-10T00:00:00Z
WhatFor: Design for the fix to `.md`-suffixed wiki-link targets
WhenToUse: Read before touching wiki-link target parsing or the wiki-link index
---


# Stripping the .md extension from wiki-link targets

## Problem

Obsidian writes wiki links in two interchangeable forms. Both are valid and both
mean the same note:

```
[[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis#Identity is an API decision]]
[[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md#Identity is an API decision]]
```

The second form — with the file extension — is what Obsidian emits under the
"absolute path in vault" link setting, and it is what LLM-generated notes tend to
produce because the model is copying a file path. A real example is
`Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md` in the go-go-parc vault,
whose entire "Names and sightings" table links this way:

| Name or alias | Exact source sighting |
|---|---|
| Semantic identity; identity projection | `[[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md#Identity is an API decision]]` |
| Canonical identity | `[[Transcripts/2026/08/09/Designing RAG Abstractions/Compositional_Retrieval_Systems_Thesis.md#9.2 Kernel K0: canonical identity]]` |

publish-vault resolves the first form and breaks on the second.

## Root cause

Three pieces disagree about whether a target carries an extension.

1. `pathToSlug` (`pkg/vault/vault.go`) computes a note's slug from its path with
   the extension **removed**: `TrimSuffix(rel, ".md")` then `Slugify`.
2. `buildWikiLinkIndex` (`pkg/vault/vault.go`) registers every path suffix, also
   with the extension **removed** (lines 381 and 387).
3. `wikiLinkHTML` / `extractWikiLinks` (`internal/parser/parser.go`) slugify the
   target **verbatim**, extension and all.

`slugify` maps every character outside `[a-z0-9\-_/]` to `-`, so the `.` becomes
a hyphen rather than disappearing:

```
"…/rag-ttc-p01-p03-doctoral-thesis.md" → "…/rag-ttc-p01-p03-doctoral-thesis-md"
```

Nothing in the index answers to that key.

## Observed failure modes

`scripts/01-md-suffix-repro` builds a throwaway vault and renders both link
forms. Two distinct outcomes:

**a) Dead link (the common case).** `ResolveWikiLink` misses, `rebuildHTML`'s
resolver returns `""`, and `ReplaceWikiLinksString` rewrites the href to a
same-page anchor:

```html
<a href="#unresolved-transcripts/2026/08/06/rag-dsl-for-retrieval/rag-ttc-p01-p03-doctoral-thesis-md"
   class="wiki-link" …>Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md</a>
```

Three things are lost at once: the destination, the `#Identity is an API decision`
heading fragment (the anchor never gets one), and the display text — the reader
sees a full raw path instead of the target note's title, because
`ReplaceWikiLinkDisplay` only substitutes titles for slugs that resolved.

**b) Wrong note (the dangerous case).** `foo.md` and `foo md` slugify to the same
`foo-md`. A vault holding both `…/doctoral-thesis.md` and
`…/doctoral-thesis md.md` resolves the `.md`-suffixed link to the *decoy*, with
no broken-link styling and no warning — the reader has no way to tell. The
repro script includes this note precisely to demonstrate it. The same collision
can come from a note *title* (`Slugify(note.Title)` is registered too), so a note
titled "Doctoral Thesis MD" would capture links meant for "Doctoral Thesis".

Failure (a) is what the vault hits today; (b) is what makes this a correctness
bug rather than a cosmetic one.

## Fix

Strip one trailing `.md` (case-insensitive) from the **link target**, in
`parseWikiLinkInner`, immediately after the alias and heading have been split
off and the target trimmed.

That single site is the choke point: every consumer of a wiki-link target goes
through it.

- `wikiLinkHTML` → `slug`, `href`, `data-target`, `data-raw`, and the fallback
  display text
- `extractWikiLinks` → `ParsedNote.WikiLinks` → `vault.WikiLinkRef` → the
  backlink graph (`buildBacklinks`) and the agent Markdown view
- `stripMarkdown` → the plain-text excerpt used for search

The heading survives because it is split off *before* the strip, so
`[[Note.md#Heading]]` still produces `href="/note/<slug>#heading"`.

### Why not elsewhere

- **In `slugify`** — it is generic. Headings and titles go through it, so
  "Notes on reading foo.md" would silently lose its tail.
- **In `buildWikiLinkIndex`, by also registering the `…-md` key** — that turns a
  miss into a *guaranteed* collision with any real note whose name ends in
  " md", i.e. it institutionalises failure mode (b) instead of removing it.
- **In `ResolveWikiLink` only** — `rebuildHTML` bypasses it and hits
  `v.wikiLinkIndex` directly with the slug the parser already baked into the
  HTML, so the href would stay wrong. `ResolveWikiLink` still gets the strip as
  well, since it is public API documented to accept a target "as written in the
  note" and external callers (scripts, server handlers) pass raw targets.

### Scope of the strip

Only `.md`, only in trailing position, only when something remains afterwards
(`[[.md]]` is left alone rather than turned into an empty target). The vault
loads exclusively `.md` files — `ReadRaw` rejects anything else via
`filepath.Ext(name) != ".md"` — so `.markdown` and friends are not notes and
must not be stripped.

Image embeds are unaffected: `isImageTarget` keys off the extension, and `.png`
/ `.jpg` / … are never touched. A note embed `![[Note.md]]` is stripped like any
other note reference, which is correct.

### Second implementation: the static web vault

`web/src/vault/staticVault.ts` is a parallel TypeScript implementation used by
the demo/static build. Its `resolveWikiTarget` has the same bug with a different
shape: `titleToSlug` deletes the dot (`[^a-z0-9-/]` → `""`) rather than
hyphenating it, so `foo.md` becomes `foomd`. It gets the same strip, applied
after the alias/heading split, plus the same fix for the display-text fallback so
`[[Foo.md]]` renders as "Foo".

## Behaviour change

`[[X.md]]` and `[[X]]` become synonyms — which is what Obsidian already does, so
this aligns publish-vault with the source of truth rather than inventing a
convention. A vault that deliberately relies on `[[X.md]]` reaching a note
literally named `X md` will now reach `X` instead; that is the bug being fixed,
not a regression to preserve.

## Tasks

1. `stripNoteExtension` in `internal/parser/parser.go`, wired into
   `parseWikiLinkInner`; unit tests for the link, embed, heading, alias,
   uppercase `.MD`, `[[.md]]`, and image-embed cases.
2. `ResolveWikiLink` in `pkg/vault/vault.go` strips too (public API robustness);
   end-to-end vault test that a `.md`-suffixed nested target resolves to the same
   slug and href as the bare one, and does not reach the "… md" decoy.
3. `resolveWikiTarget` + renderer/preprocessor display text in
   `web/src/vault/staticVault.ts`.
4. Repro script retained under `scripts/01-md-suffix-repro` as executable
   evidence.

## Validation

```bash
go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/01-md-suffix-repro
go test ./internal/parser/... ./pkg/vault/... -count=1
go test ./... -count=1
```

The repro's `== rendered HTML ==` block is the acceptance check: both table rows
must render `href="/note/transcripts/2026/08/06/rag-dsl-for-retrieval/rag-ttc-p01-p03-doctoral-thesis#identity-is-an-api-decision"`
with "Doctoral thesis" as the display text.
