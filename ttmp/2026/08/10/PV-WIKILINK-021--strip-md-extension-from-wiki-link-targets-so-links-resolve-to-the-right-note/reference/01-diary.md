---
Title: Diary
Ticket: PV-WIKILINK-021
Status: active
Topics:
    - wiki-link
    - parser
    - vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: |-
        StripNoteExtension and its single call site in parseWikiLinkInner — the fix (commit bfbcab4)
        StripNoteExtension (commit bfbcab4) and resolveSelfHeadingLinks + the target-less branch of wikiLinkHTML (commit b620b39)
    - Path: repo://pkg/vault/vault.go
      Note: ResolveWikiLink normalises the target; buildWikiLinkIndex explains why the index has no extension-bearing key (commit bfbcab4)
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: 'Click handler returns early for bare #fragment hrefs, so same-page anchors fall through to the browser'
    - Path: repo://web/src/components/organisms/NoteView/noteEnhancements.ts
      Note: enhanceHeadingAnchors makes heading ids shareable URLs — why they were left as goldmark emits them
    - Path: repo://web/src/vault/staticVault.ts
      Note: Parallel TS resolver with the same bug — stripNoteExtension/wikiLinkLabel (commit 2fb5955)
ExternalSources: []
Summary: 'Chronological record of diagnosing and fixing wiki-link targets written with a trailing .md: what the two failure modes actually were, where the strip belongs, and how the fix was validated against a real 40-link note in the go-go-parc vault.'
LastUpdated: 2026-08-10T00:00:00Z
WhatFor: Review trail and continuation notes for PV-WIKILINK-021
WhenToUse: Read before resuming work on wiki-link target resolution
---



# Diary

## Goal

Make `[[Note.md]]` behave exactly like `[[Note]]`, so wiki links copied out of a
file path — the form Obsidian emits under "absolute path in vault", and the form
LLM-written notes overwhelmingly produce — reach the note they name instead of
dying or, worse, silently reaching a different one.

## Step 1: Reproduce, and find out which failure it actually is

The report described the symptom as "linking to the wrong slug", which could
mean two very different things: a dead link, or a live link pointing somewhere
unintended. Those have different severities and, potentially, different fixes,
so the first job was to make the vault produce both under controlled conditions
rather than reason about `slugify` on paper.

I built `scripts/01-md-suffix-repro`, a throwaway vault holding the real path
shape from the report (`Transcripts/2026/08/06/RAG DSL for Retrieval/…`), one
note linking to it both ways, and — deliberately — a decoy note named
`rag-ttc-p01-p03-doctoral-thesis md.md`, whose own slug is exactly what the
buggy target slugifies to. That decoy is what turned a suspicion into a
demonstration.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket about stripping .md from the end of wiki links to avoid linking to the wrong slug. See /home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md for example which has ## Names and sightings

| Name or alias | Exact source sighting |
|---|---|
| Semantic identity; identity projection | [[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md#Identity is an API decision]] |
| Semantic identity and cache fingerprints | [[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md#P01 - Semantic identity and cache fingerprints]] |
| Canonical identity | [[Transcripts/2026/08/09/Designing RAG Abstractions/Compositional_Retrieval_Systems_Thesis.md#9.2 Kernel K0: canonical identity]] |
| Semantic plan identity | [[Transcripts/2026/08/09/Designing RAG Abstractions/Compositional_Retrieval_Systems_Thesis.md#Semantic plan identity]] |
| Domain-separated hash | [[Transcripts/2026/08/09/Designing RAG Abstractions/Compositional_Retrieval_Systems_Thesis.md#7.3 Domain-separated hashes]] |
| Semantic invocation key | [[Transcripts/2026/08/08/Job System Design Thesis/Compositional_RAG_Job_System_Thesis_MathJax.md#7.2 Semantic invocation key]] |
| Behavior identity; build-key projection | [[Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md#13. Behavior identity and causal identity]] |
| Runtime identity; behavior-complete release identity | [[Transcripts/2026/08/09/Designing RAG Abstractions/The_Semantics_and_Dynamics_of_RAG.md#5.4 Runtime identity]] |

Create a small design doc on how to tackle it, and then tackle it, keeping a detailed diary as you work (see skill for diarh format), and committing as you go"

**Assistant interpretation:** Open a docmgr ticket for the `.md`-suffixed
wiki-link bug, write a short design doc for the fix, implement it, keep a diary
of the work, and commit incrementally.

**Inferred user intent:** The go-go-parc vault has notes whose cross-reference
tables are entirely `.md`-suffixed links, and those tables are useless in the
published site. They want the published vault to follow those links.

**Commit (code):** e8ff03a — "docs(PV-WIKILINK-021): open ticket for .md-suffixed wiki-link targets"

### What I did

- Traced the three sites that disagree about extensions: `pathToSlug`
  (`pkg/vault/vault.go:1161`) and `buildWikiLinkIndex` (`:371`) both strip `.md`
  before slugifying; `wikiLinkHTML` and `extractWikiLinks`
  (`internal/parser/parser.go`) do not.
- Wrote `scripts/01-md-suffix-repro/main.go` and ran it against the unpatched
  tree.
- Created the ticket, the design doc, and four tasks.

### Why

`slugify` maps everything outside `[a-z0-9\-_/]` to `-`, so the extension's dot
becomes a hyphen rather than disappearing. The target does not fail to slugify —
it slugifies to a *different, well-formed key*. That is what makes a wrong-note
hit possible rather than merely a miss.

### What worked

The repro produced both failure modes on demand.

Without the decoy — the ordinary case:

```
target="…/rag-ttc-p01-p03-doctoral-thesis.md"
  slugify="…/rag-ttc-p01-p03-doctoral-thesis-md"
  resolved="" ok=false
```

rendering as

```html
<a href="#unresolved-transcripts/…/rag-ttc-p01-p03-doctoral-thesis-md" …>Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md</a>
```

With the decoy added, `ok=true` and the link resolves to
`…/rag-ttc-p01-p03-doctoral-thesis-md` — the decoy — with no broken styling and
nothing in the page to tell the reader.

### What didn't work

Two false starts on the script, both mine: I guessed `vault.New(root)` returned
one value and that `ListNotes` existed. The real signatures are
`New(rootDir string, opts ...Option) (*Vault, error)` and `AllNotes()`. Cheap to
fix, but a reminder to read the constructor before writing against a package.

The editor also flagged `packages.Load error: go.work requires go >= 1.26.5
(running go 1.25.5)` throughout — that is the LSP's toolchain, not the shell's
(`go version` reports go1.26.5), and every `go build`/`go test`/`go run` worked.
Ignored.

### What I learned

The dead-link case loses three things at once, not one: the destination, the
`#Heading` fragment (an unresolved anchor never gets one — `ReplaceWikiLinksString`
rewrites the whole href to `#unresolved-…`), and the display text, because
`ReplaceWikiLinkDisplay` only substitutes a note title for a slug that resolved.
A reader sees a full raw file path where a title should be. That is why these
tables look so wrong in the published site rather than merely being unclickable.

### What was tricky to build

Deciding that "wrong slug" was a real claim and not loose phrasing. The
collision needs a second note whose *name or title* slugifies to `<x>-md`, which
does not happen by accident often — but `buildWikiLinkIndex` also registers
`Slugify(note.Title)`, so a note titled "Doctoral Thesis MD" is enough. Building
the decoy explicitly, instead of arguing about likelihood, settled it and gave
the vault test something concrete to guard.

### What warrants a second pair of eyes

Nothing yet — this step only added a script and documents.

### What should be done in the future

N/A (carried into Step 2).

### Code review instructions

- Start at `internal/parser/parser.go:241` (`slugify`) and
  `pkg/vault/vault.go:371` (`buildWikiLinkIndex`); the disagreement between them
  is the whole bug.
- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/01-md-suffix-repro`

### Technical details

The index registers, per note: the full slug, every progressive path suffix with
`.md` trimmed, and the title slug. Nothing registers an extension-bearing key,
by design — see the design doc for why adding one would be the wrong fix.

---

## Step 2: Strip the extension at the one choke point

Three consumers derive things from a wiki-link target — the anchor HTML, the
`WikiLinks` list that feeds backlinks and the agent view, and the plain-text
search excerpt — and all three call `parseWikiLinkInner`. Putting the strip
there means the slug, the href, the `data-target`, the fallback display text,
the backlink graph and the excerpt cannot drift apart.

The heading survives because `parseWikiLinkInner` splits the alias and the
heading off *before* the target is finalised, so `[[Note.md#Heading]]` still
yields `href="/note/<slug>#heading"`.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the fix described in the design doc.

**Inferred user intent:** (see Step 1)

**Commit (code):** bfbcab4 — "fix(PV-WIKILINK-021): drop a trailing .md from wiki-link targets"

### What I did

- Added `parser.StripNoteExtension` and called it from `parseWikiLinkInner`.
- Made `vault.ResolveWikiLink` strip too, since it is public API documented to
  take a target "as written in the note".
- Added `TestStripNoteExtension`, `TestWikiLinkTargetDropsMarkdownExtension` and
  `TestWikiLinkMarkdownExtensionVariants` to `internal/parser/parser_test.go`,
  and `TestWikiLinkWithMarkdownExtensionResolvesToTheSameNote` (decoy included)
  to `pkg/vault/vault_test.go`.
- Verified the new tests fail against the pre-fix code before committing.

### Why

The alternatives were worse, and the design doc records why: stripping inside
`slugify` would corrupt headings and titles containing "…foo.md"; registering an
extra `…-md` key in `buildWikiLinkIndex` would turn a miss into a *guaranteed*
collision with any note named "… md", i.e. institutionalise the dangerous
failure mode; and fixing only `ResolveWikiLink` would leave the href wrong,
because `rebuildHTML` bypasses it and looks up `v.wikiLinkIndex` directly with
the slug the parser already baked into the HTML.

### What worked

Guard check — with the two call sites reverted to their old form:

```
--- FAIL: TestWikiLinkTargetDropsMarkdownExtension (0.00s)
--- FAIL: TestWikiLinkMarkdownExtensionVariants (0.00s)
--- FAIL: TestWikiLinkWithMarkdownExtensionResolvesToTheSameNote (0.00s)
```

and with them restored, `go test ./... -count=1` is green across all 13 packages
plus `make lint` (0 issues) via the pre-commit hook.

The repro now renders both link forms identically, decoy still present:

```html
href="/note/transcripts/2026/08/06/rag-dsl-for-retrieval/rag-ttc-p01-p03-doctoral-thesis#identity-is-an-api-decision" … >Doctoral thesis</a>
```

### What didn't work

Nothing failed outright. One near-miss: my first instinct was to strip inside
`slugify`, which would have quietly truncated any heading or title ending in
".md" — `[[Note#Reading foo.md]]` would have lost its anchor. Writing the
`Notes on foo.md and bar` case into `TestStripNoteExtension` pinned the boundary
before it could regress.

### What I learned

`isImageTarget` keys off `path.Ext`, so restricting the strip to `.md` leaves
`![[pic.png]]` untouched automatically — no special-casing needed in the embed
path. And because the vault's `ReadRaw` rejects anything whose extension is not
`.md`, ".markdown" genuinely is part of a note's name rather than a suffix, which
is what makes "strip only `.md`" the correct rule and not just the cautious one.

### What was tricky to build

Ordering inside `parseWikiLinkInner`. The strip has to run after the `#` split,
not before: `[[Note#Section.md]]` — a heading that happens to end in ".md" — must
keep its heading intact, and a target-first strip would have hit the wrong half
of the string. The function already splits alias, then heading, then trims, so
the strip goes on the very last line; the tests cover the alias and heading
variants specifically to hold that order in place.

The second sharp edge is the empty-target guard. `[[.md]]` would otherwise strip
to `""`, and an empty target flows into `slugify("")` → `""` → `href="/note/"`,
i.e. a link to the vault root. `len(target) <= len(".md")` returns early instead.

### What warrants a second pair of eyes

- The behaviour change is real and intentional: `[[X.md]]` and `[[X]]` are now
  synonyms. A vault that relied on `[[X.md]]` reaching a note literally named
  `X md` will now reach `X`. That is the bug, not a regression — but it is the
  one place someone could disagree.
- `ResolveWikiLink` now strips as well as slugifies. It is called from
  `buildBacklinks` with targets the parser already stripped, so the second strip
  is a no-op there; the only question is whether the public API should normalise
  at all. I think yes (its doc comment promises "as written in the note").

### What should be done in the future

- `[[#Heading]]` — a link to a heading in the *same* note — is broken
  independently of this ticket. Found while validating (Step 3): the target is
  empty, so it renders as `<a href="/note/#heading" … ></a>`: an **empty** anchor
  (invisible to the reader) pointing at the vault root instead of the heading.
  The Pattern Zoo note has six of them in its table of contents. Filed as a task
  on this ticket but deliberately not fixed here — it is a different bug in the
  same function and deserves its own change and tests.

### Code review instructions

- `internal/parser/parser.go` — `StripNoteExtension` and its single call site at
  the end of `parseWikiLinkInner`.
- `pkg/vault/vault.go` — `ResolveWikiLink`.
- `go test ./internal/parser/... ./pkg/vault/... -count=1`
- To confirm the tests really guard: revert the two call sites to
  `strings.TrimSpace(inner)` / `parser.Slugify(target)` and re-run; three tests
  must fail.

### Technical details

```go
func StripNoteExtension(target string) string {
	if len(target) <= len(".md") {
		return target
	}
	if !strings.EqualFold(target[len(target)-len(".md"):], ".md") {
		return target
	}
	return target[:len(target)-len(".md")]
}
```

`TestStripNoteExtension` pins the boundary cases: `Note.MD` → `Note`,
`Note.md.md` → `Note.md`, `readme.markdown` unchanged, `pic.png` unchanged,
`.md` unchanged, `Notes on foo.md and bar` unchanged.

---

## Step 3: The second implementation, and validation on real data

publish-vault has two wiki-link resolvers. `web/src/vault/staticVault.ts` is the
TypeScript one used by the demo/static build, and it had the same bug wearing a
different mask: its `titleToSlug` *deletes* characters outside `[a-z0-9-/]`
rather than hyphenating them, so `foo.md` was looked up as `foomd`. Same cause,
same fix, different arithmetic.

Then I validated against the note the report actually points at, rather than
against my own fixture — a 40-link, 88-`.md#`-occurrence cross-reference table in
the go-go-parc vault.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the fix everywhere it applies and show that
it works on the reported example.

**Inferred user intent:** (see Step 1)

**Commit (code):** 2fb5955 — "fix(PV-WIKILINK-021): strip .md in the static web vault too"

### What I did

- Added `stripNoteExtension` and `wikiLinkLabel` to `staticVault.ts`, wired into
  `resolveWikiTarget` and both places that compute alias-less display text (the
  `marked` extension renderer and `preprocessWikiLinks`). The backlink graph gets
  it for free — it runs every raw target through `resolveWikiTarget`.
- Wrote `scripts/02-real-note-check`, which parses a real note, copies each of
  its link targets out of the source vault into a temp vault, loads that, and
  counts `#unresolved-` occurrences. This avoids loading go-go-parc in full,
  which is its own memory story (see PV-MEMORY-019).
- Ran it on `Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md`, before and
  after.

### Why

Fixing only the Go side would leave the static build resolving the same links
differently from the server — exactly the kind of divergence that makes a bug
look intermittent later.

### What worked

Before the fix, on the real note:

```
wiki links:  46
staged:      26 target files copied into the temp vault
not on disk: 20
unresolved:  92 links rendered as #unresolved-
```

After:

```
wiki links:  40
staged:      39 target files copied into the temp vault
not on disk: 1
unresolved:  0 links rendered as #unresolved-
```

`make web-check` (`tsc --noEmit`) is clean.

### What didn't work

`pnpm exec prettier --write` on `staticVault.ts` reformatted 70 lines of
unrelated code — the repo has no prettier config, so the installed prettier's
defaults (v3 `arrowParens`, different `printWidth`) disagree with whatever
formatted the file originally. Nothing enforces prettier in the hooks
(`web-check` is `tsc --noEmit` only), so I reverted with
`git checkout web/src/vault/staticVault.ts` and re-applied the three edits by
hand. Diff went from 70 insertions / 26 deletions to 23 / 8.

### What I learned

The link *count* dropping from 46 to 40 is itself a signal, and I did not expect
it. `extractWikiLinks` deduplicates on `target + "|" + alias`, so before the fix
`[[X.md]]` and `[[X]]` were two distinct entries for one note — the outgoing-link
graph double-counted, and a note reachable both ways got two backlink entries
from a single source. Normalising the target fixes the graph as a side effect,
not just the anchors.

The 92-vs-40 gap is the other half of the picture: 40 is *distinct* links, 92 is
occurrences in the rendered HTML. One table row can carry the same target
several times.

### What was tricky to build

Getting an honest before/after on real data without loading a vault that has
previously OOM-killed the server. The trick in `02-real-note-check` is to parse
the note first, use its (already stripped) targets as a copy list, and build a
temp vault containing only the note and its 39 targets. It also means the script
is self-checking: if the strip regresses, the copy list itself goes wrong —
which is exactly what the "not on disk: 20" line in the before-run is showing,
because the script appends `.md` to targets that still carry one.

### What warrants a second pair of eyes

- `staticVault.ts` has **no runtime test coverage** — the web package has only
  `tsc --noEmit` and two SSR smoke scripts, no unit-test runner. The TS change is
  type-checked and is a three-line mirror of the Go logic, but unlike the Go side
  it is not proven by execution. Worth a reviewer's eyes on the logic itself.
- The one remaining "not on disk" entry in the after-run is the empty target from
  the `[[#Heading]]` links (see the follow-up below), not a resolution failure.

### What should be done in the future

- Fix `[[#Heading]]` same-note links (task on this ticket). Current rendering is
  `<a href="/note/#heading" class="wiki-link" data-target="" data-raw="" data-alias=""></a>`:
  empty text, wrong destination. Confirmed by parsing
  `See [[#Pattern 1 — Semantic Identity]] above.` directly.
- Consider giving `web/` a unit-test runner (vitest) so `staticVault.ts` can be
  tested rather than mirrored on trust.

### Code review instructions

- `web/src/vault/staticVault.ts` — `stripNoteExtension`, `wikiLinkLabel`, and the
  three call sites.
- `make web-check`
- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/02-real-note-check -vault /home/manuel/code/wesen/go-go-golems/go-go-parc -note "Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"`
  — must print `unresolved: 0` and exit 0.

### Technical details

The two resolvers mangle the dot differently, which is worth remembering if a
third one ever appears:

| | rule | `foo.md` becomes |
|---|---|---|
| Go `slugify` | `[^a-z0-9\-_/]` → `-` | `foo-md` |
| TS `titleToSlug` | `[^a-z0-9-/]` → `` (deleted) | `foomd` |

Neither matches the extension-less key both index builders register, so both
needed the same strip — but only the Go one could ever collide with a real note.

---

## Step 4: Discovering that the fragment cannot be computed

This step was supposed to be small — give `[[#Heading]]` a same-page href and a
display string — and it stopped being small about five minutes in. Before
writing the fix I checked what the anchor would actually have to point at, and
found that goldmark's auto heading IDs and our `slugify` disagree on most real
headings. So the obvious implementation (`href="#" + slugify(heading)`) would
have produced a link that looks correct in the HTML, passes any test written
against `slugify`, and still does nothing when clicked.

The finding is bigger than this ticket: the same mismatch affects every
cross-note `[[Note#Heading]]` link, which I fixed the *note* half of yesterday
without ever checking the *fragment* half.

### Prompt Context

**User prompt (verbatim):** "fix the 1. and write a detaield diary as you work, according to the diary skill for the format."

**Assistant interpretation:** Fix the first of the two follow-ups I reported —
`[[#Heading]]` same-note links rendering as an empty anchor pointing at the
vault root — and keep a diary in the skill's format while doing it.

**Inferred user intent:** The Pattern Zoo note's table of contents is entirely
`[[#Heading]]` links and is currently invisible and non-functional in the
published site; they want it to work.

**Commit (code):** b620b39 — "fix(PV-WIKILINK-021): make [[#Heading]] a working same-page anchor"

### What I did

- Wrote `scripts/03-heading-id-divergence`, which renders real heading text
  through the parser and prints goldmark's emitted `id` next to
  `parser.Slugify`'s answer for the same string.
- Confirmed the frontend can accept a bare `#id` href: `NoteHtml.tsx` bails out
  of SPA routing with `if (href?.startsWith("#")) return;`, leaving the browser
  to scroll.
- Checked that heading ids are user-facing — `enhanceHeadingAnchors` injects a
  `#` permalink into every heading that has one — before considering any change
  to how they are generated.

### Why

Because "compute the fragment the same way we compute slugs" is the answer a
reasonable person reaches for, and it is wrong here. Establishing that up front
determined the whole shape of the fix.

### What worked

`scripts/03-heading-id-divergence`, on headings taken from the user's own note:

```
OK "Identity is an API decision"                       goldmark="identity-is-an-api-decision"       slugify="identity-is-an-api-decision"
!! "Pattern 1 — Semantic Identity as Explicit Proj…"   goldmark="pattern-1--semantic-identity-as-…" slugify="pattern-1-semantic-identity-as-…"
!! "9.2 Kernel K0: canonical identity"                 goldmark="92-kernel-k0-canonical-identity"   slugify="9-2-kernel-k0-canonical-identity"
!! "7.3 Domain-separated hashes"                       goldmark="73-domain-separated-hashes"        slugify="7-3-domain-separated-hashes"
!! "Entity–Derivation–Observation Separation"          goldmark="entityderivationobservation-sep…"  slugify="entity-derivation-observation-sep…"
```

Five of seven realistic headings disagree. The rule is that goldmark **drops**
punctuation it does not want, while `slugify` **replaces** it with `-`; they
agree only when the heading contains nothing but letters, digits and spaces.

### What didn't work

Nothing was attempted and abandoned here — but the near-miss is worth recording,
because it is the kind of bug that ships: had I written
`href="#" + slugify(heading)` and a test asserting exactly that, the test would
have passed, the HTML would have looked plausible in review, and every link
would still have been dead. The only thing that caught it was rendering a
heading and reading the id back.

### What I learned

Three separate slug algorithms are live in this renderer: `slugify` (note slugs
and wiki-link fragments), goldmark's auto heading ID (heading anchors), and
`titleToSlug` in the static TS vault. Any code that produces a value consumed by
another one of them has to be checked against it rather than assumed compatible.

Also: heading ids are a published URL surface here, not an implementation
detail. `enhanceHeadingAnchors` gives every heading a copyable `#` permalink, so
regenerating ids with a different algorithm would silently break links people
have already shared. That ruled out the "make goldmark use `slugify`" option,
which would otherwise have been the tidiest.

### What was tricky to build

Nothing yet — this step was investigation. The trickiness it uncovered is in
Step 5.

### What warrants a second pair of eyes

The conclusion that heading ids must stay as goldmark emits them. It rests on
`enhanceHeadingAnchors` making them shareable URLs; if nobody has ever shared
one, unifying the algorithms would be a cleaner end state than what I built.

### What should be done in the future

- Cross-note `[[Note#Heading]]` fragments have the same mismatch and are **not**
  fixed by this ticket — quantified in Step 5.

### Code review instructions

- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/03-heading-id-divergence`
  — every `!!` row is a heading whose wiki-link fragment would miss.
- `internal/parser/parser.go:84` (`parser.WithAutoHeadingID()`) is where the
  heading ids come from; `internal/parser/parser.go` `slugify` is the other
  algorithm.

### Technical details

goldmark's auto heading ID keeps `[a-z0-9]`, `-` and `_`, maps spaces to `-`,
and **deletes** everything else, then disambiguates duplicates with a `-1`,
`-2`, … suffix. `slugify` maps every character outside `[a-z0-9\-_/]` to `-`
and collapses runs. Hence `9.2 …` → `92-…` vs `9-2-…`, and
`Entity–Derivation–Observation` → `entityderivationobservation` vs
`entity-derivation-observation`.

---

## Step 5: Resolve the fragment against the rendered HTML

Given that the fragment cannot be derived, the fix reads it back. `wikiLinkHTML`
emits a placeholder for a target-less link — `<a href="#"
class="wiki-link wiki-link-self" data-heading="…">` — and a new post-render pass,
`resolveSelfHeadingLinks`, runs after goldmark, collects every rendered heading's
id and text, and rewrites each placeholder to point at the id that actually
exists. Exact by construction, and immune to goldmark changing its algorithm.

Matching follows Obsidian: heading text, case-insensitive, runs of whitespace
collapsed, first heading wins on duplicates. A target that matches no heading
text is tried against the ids directly, so a link written in already-slugified
form still lands; anything left over becomes a visibly broken link instead of a
silent jump to nowhere.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** (see Step 4)

**Inferred user intent:** (see Step 4)

**Commit (code):** b620b39 — "fix(PV-WIKILINK-021): make [[#Heading]] a working same-page anchor"

### What I did

- Added the `target == ""` branch to `wikiLinkHTML`: display text falls back to
  the heading, and the link never enters the `/note/<slug>` path.
- Added `resolveSelfHeadingLinks` and called it from `Parse` after
  `renderCallouts`, before `RestoreMath`.
- Excluded target-less links from `extractWikiLinks`, so they stop entering
  `WikiLinks` (and the backlink graph, and the agent Markdown view) as
  empty-target entries.
- Four new parser tests and one vault test.
- Gave the static TS vault's `wikiLinkLabel` a heading fallback so the link is at
  least visible there.
- Wrote `scripts/04-self-heading-links` (end-to-end through the vault) and
  `scripts/05-real-self-heading-check` (audits a real note).

### Why

Resolution rather than derivation is the only approach that cannot drift: it
reads the same document the reader will scroll. It also gets duplicate headings
right for free — goldmark emits `notes` and `notes-1`, and picking the first
match is exactly Obsidian's behaviour — which a mirrored algorithm could only
achieve by reimplementing goldmark's collision counter.

### What worked

On the user's actual note, before and after (`scripts/05-real-self-heading-check`,
with the parser checked out at `2fb5955` for the "before" column):

| | before | after |
|---|---|---|
| same-note links | 24 | 24 |
| resolved to a real heading id | 0 | **24** |
| rendered as `/note/#…` (vault root) | 24 | 0 |
| anchors with no visible text | 24 | 0 |
| dangling (id absent from the page) | — | 0 |

`go test ./... -count=1` green across all 13 packages; `make lint` 0 issues;
`make web-check` clean.

### What didn't work

`git stash` to get the "before" numbers was a mistake. The fix was already
committed, so stashing changed nothing relevant, and `git stash pop` then hit

```
Auto-merging ttmp/vocabulary.yaml
CONFLICT (content): Merge conflict in ttmp/vocabulary.yaml
```

against a **stale lefthook autostash** left over from an earlier session (it adds
`api`/`xgoja` topics that HEAD already contains — 87 slugs at HEAD vs 65 in the
stash). Resolved with `git checkout HEAD -- ttmp/vocabulary.yaml`. I left the
stash entry in place rather than dropping someone else's stash; it is
`stash@{0}: autostash` and its content is already in HEAD.

The correct move, used afterwards, is `git checkout HEAD~1 -- internal/parser/parser.go`,
measure, then `git checkout HEAD -- internal/parser/parser.go`.

### What I learned

`marked` v18, which the static TS vault uses, emits **no heading ids at all** —
automatic header ids moved out to `marked-gfm-heading-id`. So the static build
has never supported heading fragments, cross-note ones included, and
`enhanceHeadingAnchors` is a no-op there. `[[#Heading]]` cannot be made to work
in that build without first adding heading ids, which is a feature, not a fix.
Hence the deliberate half-measure there: the link becomes visible, and stays
marked broken, which is at least honest.

### What was tricky to build

**Ordering against the math pass.** `resolveSelfHeadingLinks` compares the
heading text in the rendered HTML against the `data-heading` attribute. Math is
lifted out of the source *before* wiki links are extracted and restored *after*
every HTML pass, so during the window in between both sides carry the same
placeholders — a heading like `## $\sigma$ notes` and a link `[[#$\sigma$ notes]]`
match each other. Run the pass after `RestoreMath` instead and one side has TeX
markup while the other still has a placeholder, and the link silently breaks.
The call sits between `renderCallouts` and `RestoreMath` for exactly this reason,
and the comment there says so.

**Not being undone by the vault.** `rebuildHTML` re-runs every resolution pass
over the parser output on each reload, so a correct anchor can still be rewritten
later. Two passes were candidates to eat it: `ReplaceWikiLinksString`, whose
`hrefNoteRe` only matches `href="/note/…"` (ours is `href="#…"`), and
`ReplaceWikiLinkDisplay`, whose regex requires the literal `class="wiki-link"`
with its closing quote plus a `data-raw` attribute — ours is
`class="wiki-link wiki-link-self"` and carries no `data-raw`. Both miss, but by
accident of their patterns rather than by design, so
`TestSelfHeadingLinksSurviveRebuild` pins it at the vault layer where a future
change to either regex would show up.

**Deciding what an unmatched heading should look like.** Silently leaving
`href="#"` would reproduce the original complaint in a new form. It now renders
`href="#unresolved-<slugified heading>"` with a `broken` class — which
`prose.css` already styles as dotted red, since the static vault has emitted
`wiki-link broken` all along — and, crucially, keeps its text.

### What warrants a second pair of eyes

- `resolveSelfHeadingLinks` parses HTML with regexes. It is operating on our own
  freshly generated markup rather than arbitrary input, and the heading regex
  `(?s)<h[1-6][^>]*\bid="([^"]*)"[^>]*>(.*?)</h[1-6]>` is non-greedy, but a
  heading containing a raw `</h2>` inside an HTML block would confuse it. The
  same style is already used throughout this file.
- The id-fallback rule (`[[#some-heading]]` matching a heading whose id is
  `some-heading`) is **not** Obsidian behaviour. I added it because the notes
  that motivated this ticket are LLM-written and tend to emit slug forms. It
  only fires when the text match fails, so it cannot shadow a real heading.
- Excluding target-less links from `WikiLinks` changes the `wikiLinks` array in
  the note JSON. They were empty-target entries that never resolved, so nothing
  should depend on them — but it is an API-shaped change.

### What should be done in the future

- **Cross-note fragments have the same bug and are not fixed.**
  `scripts/06-cross-note-fragment-audit` measures it on the Pattern Zoo note:
  **8 of 28** cross-note `#Heading` links point at an id that does not exist in
  the target, so they open the right note at the top of the page. Every one is a
  heading with a `.` or a dash in it (`#9.2 Kernel K0: canonical identity` asks
  for `#9-2-kernel-k0-canonical-identity`; the target renders
  `#92-kernel-k0-canonical-identity`). The same read-it-back approach applies,
  but it needs a per-note heading-id index in the vault layer and a fragment
  resolver in `rebuildHTML`, which is a materially larger change than this one.
- `![[#Heading]]` (embedding a section of the current note) still renders as an
  empty `<div class="wiki-embed" data-target="">` — invisible, like the link bug
  just fixed. Transclusion of one's own section is a real feature rather than a
  one-line repair, so it is untouched.
- Block references `[[#^blockid]]` are not supported and now render as visibly
  broken rather than invisibly broken. That is an improvement, not support.
- The static build has no heading ids at all (see above).

### Code review instructions

- Start at `internal/parser/parser.go`: the `target == ""` branch in
  `wikiLinkHTML`, then `resolveSelfHeadingLinks`, then its call site in `Parse`
  (note the comment on why it precedes `RestoreMath`).
- `go test ./internal/parser/... ./pkg/vault/... -count=1`
- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/04-self-heading-links` —
  shows resolution end-to-end through the vault, including duplicate headings and
  a deliberately missing one.
- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/05-real-self-heading-check -note "/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"`
  — must print `resolved: 24`, `broken: 0`, `dangling: 0`.
- To see the "before": `git checkout HEAD~1 -- internal/parser/parser.go`, run
  the same command, then `git checkout HEAD -- internal/parser/parser.go`.

### Technical details

Rendered output for the four interesting cases:

```html
<!-- punctuation goldmark drops but slugify would hyphenate -->
<a href="#92-kernel-k0-canonical-identity" class="wiki-link wiki-link-self"
   data-heading="9.2 Kernel K0: canonical identity" data-alias="">9.2 Kernel K0: canonical identity</a>

<!-- alias wins over heading text -->
<a href="#entityderivationobservation-separation" class="wiki-link wiki-link-self"
   data-heading="Entity–Derivation–Observation Separation" data-alias="call it EDO">call it EDO</a>

<!-- duplicate headings: goldmark emits notes and notes-1; first wins -->
<a href="#notes" class="wiki-link wiki-link-self" data-heading="Notes" data-alias="">Notes</a>

<!-- no such heading: visible, and marked -->
<a href="#unresolved-no-such-heading" class="wiki-link wiki-link-self broken"
   data-heading="no such heading" data-alias="">no such heading</a>
```

Before the fix, all four were `<a href="/note/#…" class="wiki-link" data-target=""
data-raw="" data-alias=""></a>` — note the empty element content.

---

## Step 6: Cross-note fragments, resolved against the note they land on

Step 5 fixed `[[#Heading]]` by reading heading ids back out of the rendered HTML,
and left the cross-note case — `[[Other Note#Heading]]` — alone as a bigger job.
It is the same defect: the parser writes a fragment with `slugify`, the target
note's ids come from goldmark, and the two disagree on any heading containing
punctuation. 84 of the 186 rendered cross-note fragments in the Pattern Zoo note
pointed at an id that does not exist, so those links opened the right note at the
top of the page.

The reason it is bigger is that the fragment cannot be repaired where it is
written. `Parse` sees only one note; the answer lives in a different one. So the
resolution had to move to the vault layer, where every note is loaded — and,
usefully, that also makes it re-run when a heading is renamed.

### Prompt Context

**User prompt (verbatim):** "fix 1."

**Assistant interpretation:** Fix the first of the three items I listed as
shipping-unfixed in the PR readiness check — cross-note `[[Note#Heading]]`
fragments pointing at heading ids that do not exist in the target note.

**Inferred user intent:** They do not want to open the PR with a known-broken
link class in it, having just had the same bug fixed one level down.

**Commit (code):** 1a91868 — "fix(PV-WIKILINK-021): resolve cross-note #Heading fragments against the target"

### What I did

- Factored `HeadingIndex` (+ `BuildHeadingIndex`, `Lookup`) out of
  `resolveSelfHeadingLinks`, so one definition of "which id does this heading
  have" serves both the same-note and cross-note paths.
- Added `data-heading` to cross-note wiki-link anchors.
- Added `parser.ResolveWikiLinkHeadings`, and called it from `rebuildHTML`
  immediately after `ReplaceWikiLinksString`.
- Heading indexes are built lazily per target slug and cached for the duration
  of the pass; a `nil` entry caches "not a published note".
- Rewrote `scripts/06-cross-note-fragment-audit` to read the *rendered* href
  rather than recompute it, so it measures what the reader gets, and taught it
  to recognise the pre-fix markup so the same script produces both columns.
- Two parser tests and two vault tests, including a reload test.

### Why

`data-heading` is not redundant with the fragment: `slugify` is lossy, so
`#9-2-kernel-k0-canonical-identity` no longer says the heading was
`9.2 Kernel K0: canonical identity`. Without carrying the original text there is
nothing left to look up.

Resolution had to happen in `rebuildHTML` rather than in `Parse` for two
reasons. The obvious one is that the target note is not available at parse time.
The second is subtler and turned out to be a feature: `rebuildHTML` re-renders
every note from its parser output on each reload, so a heading rename in one
note automatically re-resolves every link pointing at it, in both directions —
the fragment drops when the heading disappears and comes back when it returns.

### What worked

The audit on the user's note, before (`git checkout HEAD -- internal/parser/parser.go pkg/vault/vault.go`)
and after:

| | before | after |
|---|---|---|
| rendered cross-note fragments | 186 | 186 |
| pointing at a non-existent id | **84** | **0** |
| fragment dropped (heading absent in target) | 0 | 0 |

Zero dropped is worth noting on its own: every heading these links name really
does exist in its target. The links were not typos — the renderer was simply
naming the headings by the wrong scheme.

Guard check against the pre-fix source: both vault tests fail
(`TestCrossNoteHeadingFragmentsUseTheTargetsRenderedIDs`,
`TestCrossNoteHeadingFragmentsFollowTargetEdits`) and the parser test package
fails to build, since `ResolveWikiLinkHeadings` does not exist there.

`go test ./... -count=1` green across all 13 packages, `make lint` 0 issues.

### What didn't work

Nothing failed outright this time. The measurement did trip me up in one
respect: my Step 5 note recorded "8 of 28 dangling", and this step reports "84 of
186". Both are right — the first counts *distinct* `WikiLink` entries, the second
counts *rendered occurrences*, and one table row can carry the same target
several times. The occurrence count is the reader-facing number, so the rewritten
script reports that; the discrepancy is not a change in behaviour.

### What I learned

The same-note fix from Step 5 turned out to be a scaled-down rehearsal for this
one, and factoring `HeadingIndex` out cost almost nothing because the extraction
logic was already written and tested. Had I mirrored goldmark's algorithm in
Step 5 instead (the option I rejected), this step would have had to mirror it
again in the vault layer, or export the mirror and hope both stayed in sync.

### What was tricky to build

**Deciding what to do with a heading the target does not have.** Three options:
keep the slugified fragment (a URL we know points at nothing), drop it (link
opens the note at the top), or mark the link broken. Marking it broken is wrong —
the link does go somewhere useful, only not to the right place in it — and
keeping a known-dangling fragment is dishonest markup. Dropping is what shipped,
and the test asserts both halves: no `#no-such-heading`, but still
`href="/note/target"`.

**Not rewriting links that never resolved.** A link to an unpublished note is
already `href="#unresolved-hidden"` by the time the fragment pass runs.
`crossNoteHeadingLinkRe` requires a literal `/note/` href, so those are invisible
to it — but that is a property of the regexp rather than an explicit check, so
the vault test includes a `publish: false` target specifically to pin it.

**Cost.** `rebuildHTML` already walks every note on every reload, and this adds
another pass over each one. Most notes contain no heading link at all, so
`ResolveWikiLinkHeadings` early-returns on a `strings.Contains` scan before the
regexp ever runs, and heading indexes are built only for slugs that are actually
linked to with a heading — a small minority of the vault — and cached across the
whole pass, so a popular target is indexed once rather than once per inbound
link. This matters here specifically because of PV-MEMORY-019.

### What warrants a second pair of eyes

- `crossNoteHeadingLinkRe` matches the anchor's full attribute sequence in a
  fixed order. That is safe today because `wikiLinkHTML` generates the tag and
  `ReplaceWikiLinksString` only rewrites values in place — but anyone adding an
  attribute to a wiki-link anchor must update this regexp, and the failure mode
  is silent (fragments stop being resolved, links quietly regress to opening the
  note at the top). The tests would catch it; a reader might not.
- A heading containing math, linked from *another* note, will not resolve. Math
  is lifted out per note before wiki links are extracted, so the linking note's
  `data-heading` holds its own placeholder tokens while the target's rendered
  heading holds restored TeX. The fragment is dropped, so the link still works.
  Same-note links are unaffected (both sides carry the same placeholders).
- The pass runs before `ReplaceWikiLinkDisplay`, which rebuilds the anchor open
  tag from captured groups. It preserves attributes, so the order is unchanged
  either way, but the ordering in `rebuildHTML` is deliberate.

### What should be done in the future

- `![[Note#Heading]]` embeds still ignore the heading entirely: the frontend's
  `resolveEmbeds` fetches the whole target note and injects all of it, rather
  than the named section. `data-heading` has been on the embed div all along and
  is unused.
- The static TS vault still emits no heading fragments at all, cross-note
  included, because marked v18 emits no heading ids. Unchanged by this step.
- Block references `[[Note#^blockid]]` remain unsupported; they now cleanly drop
  the fragment instead of producing a bogus one.

### Code review instructions

- `internal/parser/parser.go`: `HeadingIndex` / `BuildHeadingIndex` / `Lookup`
  (factored out of Step 5's pass), the `data-heading` attribute added in
  `wikiLinkHTML`, and `ResolveWikiLinkHeadings`.
- `pkg/vault/vault.go`: the `headingIndexFor` cache at the top of `rebuildHTML`
  and the call between `ReplaceWikiLinksString` and `ReplaceWikiLinkDisplay`.
- `go test ./internal/parser/... ./pkg/vault/... -count=1`
- `go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/06-cross-note-fragment-audit -vault /home/manuel/code/wesen/go-go-golems/go-go-parc -note "Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"`
  — must print `fragment dangles: 0` and exit 0.
- Before/after: `git checkout HEAD~1 -- internal/parser/parser.go pkg/vault/vault.go`,
  re-run the audit (it detects the pre-fix markup and says so), then
  `git checkout HEAD -- internal/parser/parser.go pkg/vault/vault.go`.

### Technical details

The anchor gains one attribute, and the fragment is rewritten in place:

```html
<!-- parser output: a provisional, wrong fragment plus the text to fix it with -->
<a href="/note/target#9-2-kernel-k0-canonical-identity" class="wiki-link"
   data-target="target" data-raw="Target"
   data-heading="9.2 Kernel K0: canonical identity" data-alias="">Target</a>

<!-- after rebuildHTML: the id the target note actually rendered -->
<a href="/note/target#92-kernel-k0-canonical-identity" class="wiki-link" ...>
```

Resolution order in `rebuildHTML` is now: `ReplaceWikiLinksString` (slug) →
`ResolveWikiLinkHeadings` (fragment) → `ReplaceWikiLinkDisplay` (text) →
`RewriteImageSources` → `ReplaceWikiEmbedImages` → `replaceUnresolvedNoteEmbeds`.
The fragment pass has to follow the slug pass, because which note's headings to
consult is not known until the slug is resolved.

---

## Step 7: Two P2 findings from the PR review

Codex left two P2 comments on PR #19. Both are real, both are mine, and both are
the same shape as the bugs this ticket set out to fix: a value produced in one
place, consumed in another, under an assumption the two ends did not share. I
reproduced each against the actual code before touching anything.

The second one is worse than the reviewer's description and worse than what I
had written in Step 6's "warrants a second pair of eyes" — I had it filed as
"math headings don't resolve", when it actually emits malformed HTML.

### Prompt Context

**User prompt (verbatim):** "Address code review comments: https://github.com/go-go-golems/publish-vault/pull/19"

**Assistant interpretation:** Work through the review feedback on PR #19 and fix
what is valid.

**Inferred user intent:** Get the PR to a mergeable state.

**Commit (code):** c279a21 — "fix(PV-WIKILINK-021): address PR #19 review findings (2x P2)"

### What I did

- Reproduced both findings in a throwaway vault before editing anything.
- **P2 #1 — uppercase extensions.** Made `parser.StripNoteExtension` the single
  definition of "strip a note's extension" and routed `pathToSlug`,
  `buildWikiLinkIndex` (both the filename and the progressive path suffixes),
  the title fallback in `loadNote`, and the `FileTree` node label through it.
- **P2 #2 — math in attributes.** Added `parser.RestoreMathText`, used it for
  every generated attribute value (`data-heading`, `data-raw`, `data-alias`,
  `data-asset`, `alt`) and for the values that reach `WikiLinks`; threaded
  `[]MathSpan` into `extractWikiLinks` and `replaceWikiLinks`/`wikiLinkHTML`;
  moved `resolveSelfHeadingLinks` to *after* `RestoreMath`.
- Three regression tests, each verified to fail with only its own fix reverted.

### Why

**#1** was a genuine regression I introduced. The vault walk accepts `Note.MD`
(`strings.HasSuffix(strings.ToLower(name), ".md")`), but `pathToSlug` and
`buildWikiLinkIndex` trimmed a lowercase `".md"` only, so the note was published
at slug `note-md`. Before this PR, `[[Note.MD]]` slugified to `note-md` and
therefore *worked by accident*. My case-insensitive strip turned the target into
`note` and broke it.

The reviewer's framing — "make extension handling consistent" — is the right
call, and consistency in the direction of stripping is strictly better: it also
fixes `[[Note]]`, the natural Obsidian form, which never resolved to a `.MD`
file at all. The cost is that a `.MD` note's slug changes from `note-md` to
`note`, which is the correct URL rather than one with the file extension baked
into it.

**#2** was live in the output all along and I had mis-diagnosed it.

### What worked

The repro for #1, with a `.MD` file whose title differs from its filename (the
title-slug entry in the index masks the bug otherwise):

```
before this PR:  [[Upper.MD]] → /note/upper-md   ✓ (by accident)
                 [[Upper.md]] → /note/upper-md   ✓
                 [[Upper]]    → #unresolved-upper ✗
after Step 2:    all three    → #unresolved-upper ✗   ← the regression
after this fix:  all three    → /note/upper       ✓
```

The repro for #2 — the actual rendered output before the fix:

```html
data-heading="The <span class="math math-inline">\sigma</span> bound"
```

The `"` before `math math-inline` closes `data-heading`, so everything after it
is parsed as further attributes on the `<a>`. After:

```html
<a href="#the-2-bound" class="wiki-link wiki-link-self" data-heading="The \sigma bound"
   data-alias="">The <span class="math math-inline">\sigma</span> bound</a>
```

Well-formed attribute carrying TeX, math still rendering in the display text,
and the link now resolves — it did not before.

`go test ./... -count=1` green across all 13 packages, `make lint` 0 issues.

### What didn't work

My first instinct on #2 was to keep `resolveSelfHeadingLinks` before
`RestoreMath` — the placement Step 5 had argued for at some length, on the
grounds that "both sides carry the same placeholders". That reasoning was
wrong, and the output proves it: the heading and the link naming it are
*separate math spans*, so they carry sentinels `2` and `0` for the same formula
and never matched. Comparing them after restoration, where both sides are the
same TeX, is what actually works. I had written the incorrect justification into
a code comment and into the design doc; both are now corrected.

### What I learned

Two things, both about the shape of my own errors.

The Step 5 comment was confidently wrong in a way tests did not catch, because
I never wrote a test with math in a heading — I reasoned about it in prose and
filed the conclusion under "known limitation" instead of running it. The
reviewer found in minutes what I had talked myself out of checking.

And #1 is a textbook instance of the very failure this ticket is about: the
existing behaviour depended on two functions agreeing about `.md`, they didn't,
and the `.MD` link worked only because both sides were wrong in the same
direction. Changing one side "correctly" broke it. Fixing the pair together is
the only stable answer.

### What was tricky to build

**Which values are element content and which are attributes.** `display` must
stay math-masked, because it is element content and `RestoreMath` turning it
into a `<span class="math…">` is exactly what should happen — that is how math
renders inside a link's text. Every *attribute* must be restored to TeX instead.
So `wikiLinkHTML` now keeps both forms of the same three values side by side
(`heading`/`attrHeading` and so on), and mixing them up would be silent: the
attribute variant renders identically until a note happens to contain math.

**The ordering constraint reversed.** Step 5 needed the self-heading pass before
`RestoreMath`; this step needs it after. The comment at the call site now spells
out why, because the old comment was a plausible-sounding argument for the wrong
thing and would otherwise invite someone to move it back.

**Threading spans without widening the API.** `extractWikiLinks` and
`replaceWikiLinks` both needed `[]MathSpan`, and both are package-private, so
the signatures changed rather than the exported surface. `RestoreMathText` is
exported only because `Parse` and the tests need it; it is deliberately separate
from `RestoreMath` rather than a mode flag on it, since "no markup, ever" is a
different contract, not an option.

### What warrants a second pair of eyes

- **Heading ids for headings that contain math embed a sentinel index**
  (`<h2 id="the-0-bound">` for `## The $\sigma$ bound`), because goldmark
  generates the id while the math is still lifted out. Links now point at that
  id correctly, so nothing is broken — but the id is not stable: adding a
  formula *earlier* in the same note renumbers it and changes the heading's
  permalink. Pre-existing, out of scope here, and worth its own ticket.
- The `.MD` slug change is user-visible: a note at `Note.MD` moves from
  `/note/note-md` to `/note/note`. That URL was wrong, but it was live.
- `wikiLinkHTML` now escapes `data-raw` and `data-alias`, which it did not
  before. That is a correctness fix (a target containing `"` produced broken
  markup), but `ReplaceWikiLinkDisplay` matches `data-raw` textually, so a raw
  target with an entity in it now compares as escaped on both sides.

### What should be done in the future

- File the unstable-heading-id issue above.
- The static TS vault does none of this math handling; it has no math pre-pass
  at all, so the question does not arise there yet.

### Code review instructions

- P2 #1: `parser.StripNoteExtension` and its four new call sites in
  `pkg/vault/vault.go` (`pathToSlug`, `buildWikiLinkIndex` ×2, the `loadNote`
  title fallback, `FileTree`).
- P2 #2: `parser.RestoreMathText` in `internal/parser/math.go`; the
  `attrTarget`/`attrAlias`/`attrHeading` values in `wikiLinkHTML`; the moved
  `resolveSelfHeadingLinks` call in `Parse` and the comment explaining the
  reversal.
- `go test ./internal/parser/... ./pkg/vault/... -count=1`
- To confirm the guards: revert either fix alone and re-run —
  `TestUppercaseMarkdownExtensionIsStrippedEverywhere` for the first,
  `TestWikiLinkAttributesCarryTeXNotMathSentinels` and
  `TestCrossNoteHeadingFragmentWithMath` for the second.

### Technical details

`RestoreMathText` and `RestoreMath` are deliberately different substitutions of
the same sentinels, and the pair is what makes matching work:

| context | substitution | after tag-strip + unescape |
|---|---|---|
| element content (`RestoreMath`) | `<span class="math math-inline">\sigma</span>` | `\sigma` |
| attribute / JSON (`RestoreMathText`) | `\sigma` | `\sigma` |

`BuildHeadingIndex` strips tags and unescapes, so a heading rendered the first
way and a `data-heading` written the second way reduce to the same key.
