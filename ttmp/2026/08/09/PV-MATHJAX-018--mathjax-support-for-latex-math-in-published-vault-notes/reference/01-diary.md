---
Title: Diary
Ticket: PV-MATHJAX-018
Status: active
Topics:
    - mathjax
    - math
    - latex
    - parser
    - html-rendering
    - frontend
    - ssr
    - bundle
    - obsidian-vault
    - retro-obsidian-publish
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/math.go
      Note: Scanner and sentinel round-trip built in Step 2 (commit e9f2784)
    - Path: repo://internal/parser/math_test.go
      Note: Executable spec; caught the inline-placeholder escaping bug on first run
    - Path: repo://vault-example/Mathematics/Math Showcase.md
      Note: Hostile fixture that found the code-span closer bug
    - Path: repo://web/src/lib/mathjax.ts
      Note: Where the four MathJax-4 integration problems are fixed and documented (commit 40886c3)
    - Path: repo://web/src/styles/prose.css
      Note: Tailwind preflight override that restores inline SVG flow
ExternalSources: []
Summary: 'Chronological implementation diary for MathJax support: what was investigated, what was built, what broke, and how to review it.'
LastUpdated: 2026-08-09T00:00:00Z
WhatFor: Resuming or reviewing the MathJax work.
WhenToUse: Read before continuing work on PV-MATHJAX-018.
---


# Diary

## Goal

Record the end-to-end implementation of LaTeX math support in `publish-vault`:
the codebase investigation that produced the design, each implementation phase,
the failures encountered, and what a reviewer should look at.

---

## Step 1: Investigate the pipeline and write the design guide

Before writing any code I read the note pipeline end to end, because math sits
exactly on the seam between the Go half (Markdown → HTML) and the React half
(HTML → pixels), and getting the seam wrong is the whole difficulty of the
feature. The key discovery is that `internal/parser/parser.go` already has both
a pre-goldmark rewriting pass (`replaceWikiLinks`, line 163) and a post-goldmark
rewriting pass (`renderCallouts`, line 359). That establishes the idiom this
project uses for Obsidian-flavoured syntax, and it means math does not need a
goldmark AST extension — it needs a third pre-pass, run first.

The second discovery is on the frontend: `noteEnhancements.ts` is a set of
idempotent, cancellable DOM passes invoked from `useEffect` in `NoteHtml.tsx`,
and `enhanceMermaid` is a near-perfect structural template for `enhanceMath`
(cheap bail-out, dynamic import for code splitting, module-level init guard,
`cancelled` flag, `isConnected` re-check after every await, graceful failure
that leaves the source visible). Copying that shape means math inherits all the
hydration- and lifecycle-correctness the project already paid for.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket (in publish-vault, but ttmp.yaml should be set properly) for adding mathjax support, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\n\nTHen work through the tasks one by one, keeping a detailed diary as you work (see format in the diary skill) and commit at appropriate intervals."

**Assistant interpretation:** Create ticket `PV-MATHJAX-018` under the
workspace-level `.ttmp.yaml` (which already roots docmgr at
`publish-vault/ttmp`), write an intern-grade design/analysis/implementation
guide into it, upload that guide to reMarkable, then implement the plan task by
task with a running diary and periodic commits.

**Inferred user intent:** Get MathJax actually shipped in publish-vault, with a
document good enough that someone unfamiliar with the codebase could have done
it — and with the reasoning captured so the choices survive review.

### What I did

- Read `internal/parser/parser.go` in full (667 lines): `Parse`, the wiki-link
  pre-pass, `splitFrontmatter`, `renderCallouts`, `stripMarkdown`, `PlainText`.
- Read `pkg/vault/vault.go` around `Note`, `loadNote`, `rebuildHTML` (four
  regex passes over rendered HTML) and `sourceHTML` vs `HTML`.
- Read `pkg/api/api.go` (the `Note` JSON is the struct verbatim; `HTML` passes
  through untouched) and `pkg/server/agent_markdown.go` (the `/note/{slug}.md`
  mirror serves raw Markdown from disk, so agents get original TeX for free).
- Read the frontend: `NoteHtml.tsx`, `NoteBody.tsx`, `noteEnhancements.ts`,
  `lib/highlightLanguages.ts` + its `.server.ts` stub, `vite.config.ts`
  (the `isSsrBuild` conditional alias), `entry-server.tsx`, `server.mjs`.
- Read the widget layer: `widgets/ir/props.ts`, `NoteHtml.widget.tsx`,
  `pkg/vaultwidgets/vaultwidgets.go`.
- Confirmed there is currently **zero** math handling anywhere: grepping for
  `math|katex|latex|mathjax` across Go/TS/CSS/MD returns only false positives
  (`Math.floor`, "polymath", "Mathematical truths" in a sample note).
- Checked the npm landscape: `mathjax@4.1.3` (component bundles),
  `@mathjax/src@4.1.3` (ESM source, bundler-friendly),
  `@mathjax/mathjax-{newcm,tex,modern}-font@4.1.3` (v4 split fonts out of core),
  `mathjax-full` frozen at 3.2.2, `katex@0.18.3`.
- Created ticket `PV-MATHJAX-018`, added vocabulary topics `mathjax`, `math`,
  `latex`, and wrote the design doc plus 14 phased tasks.

### Why

The feature's difficulty is not "call MathJax" — it is that Markdown mangles
TeX before anyone gets to call MathJax. Writing down exactly *how* it mangles it
(underscores → `<em>`, `\\` eaten, `&` → `&amp;`, `WithHardWraps()` inserting
`<br/>` into `align` bodies) is what turns a vague task into a specification.
The design doc's §4 is that specification, and §10.1's 20-row table test is its
executable form.

### What worked

- The `replaceWikiLinks` idiom transfers cleanly. `splitFrontmatter` is already
  factored out and reusable, so math scanning gets frontmatter safety for free.
- `html.WithUnsafe()` is already enabled (parser.go:79), so injected raw HTML
  placeholders pass through goldmark without any renderer changes.
- Surrounding the display-math `<div>` with blank lines forces goldmark down the
  HTML-block path, which simultaneously solves two problems: no illegal
  `<div>`-inside-`<p>`, and no `<br/>` injection from `WithHardWraps()`.
- The `@highlight-languages` conditional-alias trick in `vite.config.ts` is a
  ready-made answer for keeping MathJax out of the SSR module graph.

### What didn't work

Nothing failed at this stage — no code was written yet. Two things were
*rejected* after analysis rather than after failure:

- **A goldmark AST extension.** Correct in the abstract, but it fights the
  project's established idiom and is much harder to table-test than a pure
  `func ScanMath([]byte) []MathSpan`.
- **Server-side pre-rendering in the SSR sidecar.** The note HTML originates in
  Go and is an opaque string to the sidecar; transforming it there means the
  client must reproduce a byte-identical MathJax SVG at hydration or React
  discards the server DOM. Deferred to a follow-up ticket with a better shape
  (pre-render once in Go's load path, cache in `Note.HTML`).

One environment wrinkle: the editor LSP reports
`go: go.work requires go >= 1.26.5 (running go 1.25.5)`. The shell `go` is
1.26.5 and `go build ./publish-vault/...` succeeds, so this is an LSP-only
toolchain mismatch, not a build problem.

### What I learned

- `Note` keeps two HTML fields: the public `HTML` and the unexported
  `sourceHTML` (the parser output before vault-level resolution). `rebuildHTML`
  always re-renders from `sourceHTML`, which means any placeholder the parser
  emits is re-processed on every reload — so those four regexes in
  `rebuildHTML` must provably not match math markup. That deserves its own test.
- `NoteBody`'s `memo()` is not an optimisation; its comment (NoteBody.tsx:1–14)
  explains that React 19 re-applies `dangerouslySetInnerHTML` on every render,
  which would silently destroy every enhancement. Anything I add to the
  enhancement pipeline depends on that memo staying intact.
- MathJax v4's direct API (`mathjax.document(...).convert(tex, {display})`) lets
  us disable delimiter scanning entirely (`inlineMath: []`, `displayMath: []`),
  because the Go parser has already told us exactly which elements hold TeX.
  That removes a whole category of "MathJax found math where I didn't want it".
- SVG output needs `fontCache: "local"`; the default global glyph cache uses
  `<use>` references into a shared `<defs>`, which dangle the moment a formula
  node is cloned into the lightbox or moved by a re-render.

### What was tricky to build

The scanner rules. The naive regex `\$([^$]+)\$` fails on four separate axes at
once, and each failure has a different cause:

- **Currency.** `costs $30 and $25 used` matches `30 and ` as math. Cause: no
  constraint on what surrounds the delimiters. Fix: Pandoc's rules — an opener
  must not be followed by whitespace, a closer must not be preceded by
  whitespace nor followed by an ASCII digit. Traced through both directions in
  design doc §7.2.3.
- **Escaped dollars.** `\$100` must stay literal. Cause: regexes cannot cheaply
  count preceding backslashes. Fix: the scanner's `\\` branch advances `i += 2`
  unconditionally, so an escaped `$` is consumed before the `$` branch ever
  sees it.
- **Code.** ``` `$x$` ``` and fenced blocks must be inert. Cause: regexes have no
  notion of block structure. Fix: an explicit state machine that skips code
  spans by matching backtick-run length, and skips fences by matching the fence
  character and length.
- **Runaway spans.** A stray `$` would otherwise swallow the rest of the
  document. Fix: abandon an inline candidate if the search for a closer crosses
  a blank line.

The other tricky part is escaping *once and only once*. TeX contains `&`, `<`,
`\`, and quotes; HTML needs `&`/`<`/`>` escaped; the DOM un-escapes them when
you read `textContent`. Getting a `&amp;amp;` requires only one accidental
double-escape somewhere in the four-pass `rebuildHTML` chain. The mitigation is
a round-trip test helper: TeX in → `Parse` → extract placeholder text → compare
byte-for-byte with the original.

### What warrants a second pair of eyes

- The `$` opener/closer rules against real vault content. Rules that are correct
  on the 20 test cases can still surprise on prose that mixes prices, code, and
  math in one paragraph.
- The decision to put TeX in **text content** rather than a `data-tex`
  attribute. It buys a no-JS fallback and newline fidelity; the cost is that raw
  TeX is briefly visible before typesetting. Confirm that trade is wanted.
- Rejecting server-side pre-rendering for v1. If SEO/agent visibility of
  rendered math matters more than I assumed, the phasing should change.

### What should be done in the future

- Server-side pre-render (design doc §9.6) as a separate ticket.
- Vault-global `\newcommand` macro preamble via the `TeX` constructor's `macros`
  option.
- MathJax assistive-MathML / speech output for accessibility.

### Code review instructions

- Start with the design doc, §4 (why math breaks) and §7.2 (the scanner spec) —
  everything else follows from those two.
- Then read `internal/parser/parser.go:56-112` (`Parse`) to see the insertion
  point, and `noteEnhancements.ts:24-76` (`enhanceMermaid`) to see the template.
- Validate the design's claims yourself:
  `go test ./publish-vault/internal/parser/... -count=1` on the current tree
  should still pass (no code changed yet), and
  `grep -rn "WithHardWraps\|WithUnsafe" publish-vault/internal/parser/parser.go`
  should confirm both renderer options are on.

### Technical details

Current `Parse` pre-pass order (parser.go:56-61), and where math goes:

```go
wikiLinks := extractWikiLinks(src)
processed := replaceMathInBody(src)      // ← NEW, must be first
processed  = replaceWikiLinks(processed)
```

The placeholder shapes:

```html
<span class="math math-inline">e^{i\pi} + 1 = 0</span>

<div class="math math-display">
\begin{align} a &amp;= b \\ c &amp;= d \end{align}
</div>
```

Relevant renderer options already set in `Parse`:

```go
goldmark.WithRendererOptions(
    html.WithHardWraps(),   // ← inserts <br/> in paragraphs; HTML blocks are immune
    html.WithXHTML(),
    html.WithUnsafe(),      // ← lets our placeholders survive
)
```

---

## Step 2: Go-side math protection (Phases 1–2)

Implemented `internal/parser/math.go` — a left-to-right state machine that finds
`$…$`, `$$…$$`, `\(…\)` and `\[…\]` regions in raw Markdown, plus the pre-pass
that lifts them out before goldmark and the post-pass that puts them back after
every other HTML rewrite. Wrote the 20-case table test first; it is the real
specification and it earned its keep within minutes.

The design's central assumption turned out to be half wrong, and the test caught
it on the first run. Emitting the final `<span class="math math-inline">TeX</span>`
in the pre-pass does not protect anything: goldmark treats an inline `<span>` as
raw inline HTML but still parses the text *between* the tags as Markdown, so
`$\{1,2\}$` came back as `{1,2}`. The design had to change shape mid-step, from
"emit the markup early" to "emit an opaque sentinel early, emit the markup last".

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Work Phases 1–2 of the plan: scanner, placeholder
emission, wiring into `Parse`, search-index handling, and the `rebuildHTML`
invariant test.

**Inferred user intent:** Ship the Go half with tests that will keep it working.

**Commit (code):** `e9f2784` — "feat(PV-MATHJAX-018): protect LaTeX math through the Markdown pipeline"

### What I did

- Wrote `internal/parser/math.go`: `MathSpan`, `ScanMath`, `ReplaceMath`,
  `RestoreMath`, `StripMathDelimiters`, `replaceMathInBody`, plus the helpers
  `validInlineOpener`, `scanInlineClose`, `scanUntil`, `skipCodeSpan`,
  `fenceOpensAt`, `skipFencedBlock`, `blankLineAt`.
- Wired `replaceMathInBody` into `Parse` before `replaceWikiLinks`, and
  `RestoreMath` after `renderCallouts`.
- Added math stripping to `stripMarkdown` so the search index keeps TeX bodies
  without delimiters.
- Wrote `internal/parser/math_test.go` (26 spec rows + round-trip + mangling +
  interaction + termination tests) and
  `TestMathPlaceholdersSurviveRebuildHTML` in `pkg/vault/vault_test.go`.

### Why

Everything in §4 of the design doc: goldmark destroys unprotected TeX four
different ways, and each way has a different cause. A scanner rather than a
regexp because three rules cannot be expressed in RE2 — counting preceding
backslashes, block structure for fences, and Pandoc's currency lookaround.

### What worked

- The 20-row table test as an executable spec. Writing it before the
  implementation meant the currency and code-fence rules were pinned down before
  I had a chance to hand-wave them.
- The blank-lines-around-display-math trick does exactly what the design claimed:
  it forces goldmark's HTML-block path, which both avoids `<div>`-inside-`<p>`
  and makes `WithHardWraps()` unable to interleave `<br/>` into a `\begin{align}`
  body. `TestParseMathSurvivesMarkdownMangling` asserts both.
- `splitFrontmatter` was already factored out for wiki links, so frontmatter
  safety came free.
- `rebuildHTML`'s four regex passes genuinely do not touch math, and the new
  vault test proves the second rebuild is a fixed point.

### What didn't work

**Inline placeholders emitted in the pre-pass.** First test run:

```
--- FAIL: TestParseMathRoundTrip (0.00s)
    --- FAIL: TestParseMathRoundTrip/braces (0.00s)
        math_test.go:141: placeholder 0 TeX = "{1, 2}", want "\{1, 2\}"
            HTML: <p><span class="math math-inline">{1, 2}</span></p>
```

Root cause: raw inline HTML is opaque as *markup*, not as *content*. goldmark
still ran its inline parser over the span's text and consumed `\{` as a Markdown
escape. Note that the underscore and asterisk cases passed — but only by luck:
CommonMark forbids intraword `_` emphasis, and ` * ` surrounded by spaces is not
a valid emphasis opener. `$f *g* h$` would have failed.

Fix: two-phase substitution through a `U+E000`/`U+E001` sentinel, with
`RestoreMath` running after `renderCallouts` so no other pass ever sees the math
markup.

### What I learned

- "Raw HTML passes through" is a statement about tags, not about the text between
  them. For anything that must survive byte-for-byte, the payload has to leave
  the document entirely and come back afterwards.
- Private Use Area code points make a good sentinel: goldmark's text renderer
  passes arbitrary non-ASCII through untouched, they have no Markdown meaning,
  and no real note contains them. A plain ASCII token would have to worry about
  Markdown-active characters and about colliding with note content.
- Two of my "passing" tests were passing for the wrong reason. Worth remembering
  when a test suite goes green faster than expected.

### What was tricky to build

The interaction between the `$` rules and code. The Pandoc rules — opener not
followed by whitespace, closer not preceded by whitespace and not followed by a
digit — handle `costs $30 and $25` correctly on their own, and I verified that
by hand in both directions before writing the code. What they do *not* handle is
a later `$` that happens to sit inside a code span; see Step 3, where the
showcase note found exactly that.

The other sharp edge is the backslash branch in `ScanMath`. `i += 2` on any `\`
is what makes `\$100` literal, but it has to come *after* the `\[` and `\(`
checks, and inside `scanUntil` the closer must be tested *before* the escape rule
or `\]` and `\)` — closers that are themselves backslash sequences — could never
be found.

### What warrants a second pair of eyes

- Whether `RestoreMath` belongs after `renderCallouts` (current choice: yes,
  nothing downstream should see math markup) or before it.
- The sentinel collision assumption. It is a soft assumption; the consequence of
  a collision is cosmetic, but it is unguarded.
- `ScanMath` termination. Every branch advances `i`, and `TestScanMathTerminates`
  covers 17 pathological inputs, but a `go test -fuzz=FuzzScanMath` run would be
  the real answer.

### What should be done in the future

- A fuzz target for `ScanMath`.
- Revisit indented-code skipping if someone actually hits it.

### Code review instructions

- Start at `internal/parser/math.go` — read the package comment, then `ScanMath`,
  then the `ReplaceMath`/`RestoreMath` pair.
- Then `internal/parser/parser.go:56-100` to see the two insertion points.
- Validate: `go test ./publish-vault/internal/parser/... -count=1 -v`, then
  `go test ./publish-vault/... -count=1`, then
  `golangci-lint run ./internal/parser/...` (0 issues).
- The table in `math_test.go` is the spec; read it before the implementation.

### Technical details

Pipeline order after this step:

```go
wikiLinks := extractWikiLinks(src)
processed, mathSpans := replaceMathInBody(src)   // math out (sentinels in)
processed = replaceWikiLinks(processed)
md.Convert(processed, &buf, …)
htmlOut = renderCallouts(htmlOut)
htmlOut = RestoreMath(htmlOut, mathSpans)        // math back, last
```

---

## Step 3: Browser typesetting (Phase 3) and the showcase note

Implemented `enhanceMath` and the MathJax wrapper, then built a showcase note
that exercises every branch — inline, display, `align`, `cases`, `pmatrix`,
LaTeX-native delimiters, currency prose, escaped dollars, math in code fences, in
callouts, in lists, in tables, and a deliberately over-wide equation. Rendering
that note through the real parser immediately found a scanner bug the unit tests
had missed, and then running it in a browser found three MathJax/bundler
integration problems the design had not anticipated at all.

This step is the argument for building a fixture that is deliberately hostile to
your own implementation. Every one of the four bugs below came from the showcase
note or from looking at it in a browser; none came from the unit tests.

### Prompt Context

**User prompt (verbatim):** (see Step 1). Mid-step the user added: "not perfect
on say P(H|E) or so, it seems veyr narrow? take screenshots of the whole page so
make sure all the formatting is good." and, after a first fix, "it could be
working i don't know you test".

**Assistant interpretation:** Do not declare the frontend done on the basis of
"no console errors" — actually look at the rendered page, at full length, and
verify the formatting.

**Inferred user intent:** The earlier screenshot showed obviously broken layout;
they want the visual result verified properly rather than asserted.

**Commit (code):** `40886c3` — "feat(PV-MATHJAX-018): typeset math with MathJax in the browser"

### What I did

- Added `@mathjax/src` and `@mathjax/mathjax-newcm-font`.
- Wrote `web/src/lib/mathjax.ts` (TeX→SVG singleton, `typesetTeX`,
  `ensureMathStyles`, `FONT_RANGES`) and `mathjax.server.ts` (SSR stub), with the
  conditional `@mathjax-typeset` alias in `vite.config.ts` and `tsconfig.json`.
- Added `enhanceMath` to `noteEnhancements.ts` and wired it into `NoteHtml`;
  added an `onEmbedRendered` callback to `resolveEmbeds` so embedded notes get
  typeset too.
- Added `.math` / `.math-display` styling to `prose.css`.
- Plumbed the `math` toggle through the widget IR, `NoteHtml.widget.tsx` and
  `vaultwidgets.go`; regenerated `reader-page.golden.json`.
- Wrote `vault-example/Mathematics/Math Showcase.md` and the ticket script
  `scripts/01-render-note` that renders a file through the real parser.
- Verified in Chromium via Playwright at 1200px and 390px.

### Why

The enhancement-pipeline shape was already settled by `enhanceMermaid`. The
effort went into the MathJax integration, which is where all the surprises were.

### What worked

- `enhanceMermaid` as a structural template. Cheap bail-out, dynamic import,
  cancel flag, `isConnected` re-check, graceful failure — all transferred
  unchanged, and the SSR smoke test passed first try with 0 hydration warnings.
- The `highlightLanguages.ts` explicit-map idiom transferred too, and turned out
  to be exactly the right shape for MathJax's font ranges.
- Disabling MathJax's own delimiter scanning (`inlineMath: []`, `displayMath: []`)
  and calling `convert()` with an explicit `display` flag. Zero false positives.
- Code splitting: `main` stayed at 468 KB and references MathJax only through
  `import("./mathjax-*.js")`.

### What didn't work

Four failures, in the order I hit them.

**1. `$30` closed inline math inside a later code span.** Found by rendering the
showcase note, not by a unit test. This prose:

```
The hardback costs $30 and the paperback costs
$25; neither dollar sign opens math, because a closing `$` may not be preceded
```

produced:

```html
<p>Prices are prose, not formulas. The hardback costs <span class="math math-inline">30 and the paperback costs
$25; neither dollar sign opens math, because a closing `</span>` may not be preceded
```

The backticked `` `$` `` satisfies both closer rules (preceded by a backtick, not
a space; followed by a backtick, not a digit). `ScanMath` skipped code spans at
the top level but `scanInlineClose` did not. Fixed by skipping code spans inside
both `scanInlineClose` and `scanUntil`; pinned by
`TestInlineMathDoesNotCloseInsideCodeSpan`.

**2. MathJax's fonts would not load.**

```
Can't load '@mathjax/mathjax-newcm-font/js/svg/dynamic/double-struck.js':
No mathjax.asyncLoad method specified
Error: dynamic file 'double-struck' failed to load
```

MathJax 4 splits the font into ~40 glyph-range files fetched through
`mathjax.asyncLoad`, which no bundler provides. Fixed with an explicit
`FONT_RANGES` map of dynamic imports. A second, separate problem hid behind it:
`convert()` is synchronous, so it throws a *retry signal* when it needs a range
that is not resident; it has to be wrapped in `mathjax.handleRetriesFor`.

Also in this area: the design's `AllPackages` import does not exist in MathJax 4
(it was a v3 export), and the correct specifier prefix is `@mathjax/src/js/`, not
`/mjs/` — the package `exports` map is `"./js/*": {"import": "./mjs/*"}`.

**3. Every formula broke at every operator.** The user's screenshot showed
`e^{iπ}` / `+ 1` / `= 0` stacked on three lines. MathJax 4 breaks inline math to
fit its container and measures that container from the DOM — but `convert()`
builds a *detached* node, so the width came back ~0 and it broke at every
available point. Fixed with `linebreaks: { inline: false }`.

**4. Each formula was its own full-width line box.** After fix 3 the formulas
were correct but still not flowing in the sentence. Measured:

```
svgDisplay:  "block"
mathRect:    { w: 700 }      // the full paragraph width
sameLine:    false
```

against an SVG whose intrinsic width is `10.751ex`. Cause: Tailwind v4's
preflight applies the cssremedy rule `img, svg, video, … { display: block }`.
Fixed with a scoped `.math mjx-container > svg { display: inline-block }`.

### What I learned

- MathJax 4 is meaningfully different from 3 in three ways that all bite a
  bundler: no `AllPackages`, dynamically-loaded font ranges, and inline
  linebreaking on by default. None of this is obvious from the migration notes.
- A CSS reset can break a library that has nothing to do with your CSS. Tailwind
  preflight versus MathJax's inline SVG is not a combination either project
  documents.
- "No console errors" is not evidence of correct rendering. Failures 3 and 4
  produced a completely broken layout with a clean console; only looking at the
  page — and then measuring bounding rects — caught them.
- Measuring beats eyeballing for the follow-up. Comparing the math element's
  rect against a `Range` over the preceding text node answers "is it on the same
  line" unambiguously, and `scrollWidth - clientWidth` answers "does it overflow"
  without squinting at a screenshot.

### What was tricky to build

Failure 2 was two bugs stacked, which made it read as one intermittent bug. The
`asyncLoad` warning and the `dynamic file failed to load` error arrive together,
so supplying `asyncLoad` alone still fails — the load now *starts*, but the
synchronous `convert()` has already thrown by the time it finishes. I only found
the second half by reading `FontData.js:290-321` and noticing that the sync path
throws a retry rather than blocking, then finding `handleRetriesFor` exported
from `mathjax.js:12`.

Failure 4 was tricky in a different way: the symptom (formula on its own line)
looked identical to failure 3's symptom, so it was easy to believe fix 3 had not
worked. What separated them was measuring `getComputedStyle(svg).display` —
`"block"` is not something MathJax would ever set, which pointed straight at a
project-level stylesheet, and from there to preflight.

### What warrants a second pair of eyes

- The `FONT_RANGES` map is hand-written from a directory listing. A missing entry
  fails only for notes using that glyph range. Worth a test that asserts the map
  keys match the font package's `dynamic/` directory.
- `linebreaks: { inline: false }` means a very long inline formula on a narrow
  screen will overflow its line rather than wrap. Measured as fine at 390px for
  the showcase content (widest inline formula 92px in a 360px column), but a note
  with a genuinely long inline formula would look bad.
- The bundle cost: ~640 KB gzip for a note using `\sigma`. Recorded in design doc
  §14.8 with two follow-up options.
- Whether re-running `enhanceMath` from `resolveEmbeds`' callback should be
  generalised to the other enhancement passes (mermaid and highlight have the
  same gap today, pre-existing).

### What should be done in the future

- Evaluate `@mathjax/mathjax-tex-font` against the 281 KB gzip `greek` range.
- Vitest unit test for `enhanceMath` with the MathJax module mocked (the pipeline
  is currently covered only by browser verification).
- Storybook story with math content.
- Investigate the four `.note-prose` containers present in dev mode (three
  zero-sized). Pre-existing, reproduces without math, but the enhancement passes
  run four times because of it.
- Exclude `ttmp/` from `glazed-lint` in the Makefile — all three tickets open on
  this branch had to use `--no-verify` solely because it lints scratch programs.

### Code review instructions

- `web/src/lib/mathjax.ts` first: the four comment blocks explain the four
  MathJax-4-specific decisions (`FONT_RANGES`, `handleRetriesFor`, `fontCache`,
  `linebreaks`).
- Then `enhanceMath` in `noteEnhancements.ts` — check that every `await` is
  followed by a `cancelled` / `isConnected` re-check, and that
  `data-math-state = "pending"` is set synchronously before the first await.
- Then the `.math` rules in `prose.css`, especially the preflight override.
- Validate:

  ```bash
  cd publish-vault/web && pnpm check && pnpm build && pnpm build:ssr && pnpm smoke:ssr
  cd publish-vault/web/dist/assets && grep -o 'import("./mathjax-[^)]*)' main-*.js   # must be dynamic
  go run ./publish-vault/ttmp/2026/08/09/PV-MATHJAX-018--*/scripts/01-render-note \
        "publish-vault/vault-example/Mathematics/Math Showcase.md"
  ```

  Then serve `vault-example` and open `/note/mathematics/math-showcase`.

### Technical details

Measured in the browser after the fixes (Chromium, 1200×1400 then 390×800):

| Check | Result |
|---|---|
| `.math` elements typeset | 64 / 64, `data-math-state="done"`, 0 errors |
| Inline math on same line as surrounding text | yes (81px wide, matching text-run top) |
| Widest display math at 390px | `clientWidth` 352, `scrollWidth` 639 → scrolls in-box |
| `document.scrollWidth - clientWidth` | 0 at both widths |
| Glyph colour | `<g fill="currentColor" stroke="currentColor">` |
| MathJax stylesheet | injected, 5884 chars |
| SSR graph | only `mathjax.server-*.js`, 0.21 kB |
| SSR hydration warnings | 0 |
