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
      Note: StripNoteExtension and its single call site in parseWikiLinkInner — the fix (commit bfbcab4)
    - Path: repo://pkg/vault/vault.go
      Note: ResolveWikiLink normalises the target; buildWikiLinkIndex explains why the index has no extension-bearing key (commit bfbcab4)
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
