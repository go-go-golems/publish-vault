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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for MathJax support: what was investigated, what was built, what broke, and how to review it."
LastUpdated: 2026-08-09
WhatFor: "Resuming or reviewing the MathJax work."
WhenToUse: "Read before continuing work on PV-MATHJAX-018."
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
