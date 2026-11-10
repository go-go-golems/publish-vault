---
Title: Implementation diary
Ticket: PV-MARKDOWN-030
Status: active
Topics:
    - markdown
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/callout.go
      Note: Structural callout parsing and rendering
    - Path: repo://internal/parser/softbreak_test.go
      Note: Soft and explicit line break regression coverage
    - Path: repo://pkg/vault/callout_pipeline_test.go
      Note: Vault resolution integration coverage
ExternalSources: []
Summary: Chronological investigation and implementation record
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation diary

## Step 1: Establish the branch, ticket, and architectural evidence

The user requested an intern-facing design, reMarkable delivery, implementation and a detailed diary. I created a dedicated branch and ticket before touching implementation, and traced the current parse and vault rebuild boundaries.

### Prompt Context
**User prompt (verbatim):** "Ok, create a new docmgr ticket, and then Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\n\n(the document is detailed, the API is elegant, streamlined, composable, modern). \n\nThen, implement and commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"

**Additional user prompt (verbatim):** "create a new branch."

**Assistant interpretation:** Design and implement a composable Markdown pipeline addressing the hard-wrap/callout coupling, without assuming backward compatibility is required; document and deliver the guide.

**Inferred user intent:** Make rendering features structurally composable while fixing the visible wrapping defect.

### What I did
- Created branch `task/markdown-ast-pipeline` from `0c78a5e` and ticket `PV-MARKDOWN-030` with design, diary, and three tasks.
- Inspected `internal/parser/parser.go`, parser tests, `pkg/vault/vault.go`, and callout CSS; located the prior PV-MARKDOWN-023 architecture review.
- Observed unrelated untracked `.claude/`, `.playwright-mcp/`, `search-memory-results.png`, and `ttmp/vocabulary.yaml.orig`; preserved them.

### Why
- The vault resolves cross-note references only after building indexes, so parsing a single note cannot safely resolve all destinations in isolation.
- Callout parsing currently depends on HTML `<br />`, which is generated for every source newline by `html.WithHardWraps()`.

### What worked
- `git switch -c task/markdown-ast-pipeline` and docmgr ticket/document/task commands succeeded.
- Source inspection established that `Vault.rebuildHTML` begins with parser-produced `sourceHTML` and applies vault-specific rewrites.

### What didn't work
- No failures observed in this step.

### What I learned
- A previous callout test contains literal `\\n` characters rather than actual line breaks and does not verify title/body structure. Existing tests are not sufficient for this change.

### What was tricky to build
- The AST design must account for multiple Goldmark parsing stages and preserve vault-level link resolution; replacing the entire pipeline at once would conflate separable risks.

### What warrants a second pair of eyes
- Markdown extension precedence, callout nesting, and resolver behavior across vault reloads.

### What should be done in the future
- Write the design, implement in validated increments, record failures and commit hashes, and deliver the requested document.

### Code review instructions
- Start at `internal/parser/parser.go:56`, then `pkg/vault/vault.go:381` and `:616`.

### Technical details
- Goldmark version: `github.com/yuin/goldmark v1.8.2`; the existing renderer enables both `html.WithHardWraps()` and `html.WithUnsafe()`.

## Step 2: Publish the guide and capture failing behavior

I wrote and delivered the architecture guide, then established executable regression examples. The baseline existing suites pass, but the new targeted tests demonstrate both the soft-break defect and malformed multi-block callout output. The implementation remains outstanding; the failing probe is stored as a non-Go ticket artifact rather than leaving the repository test suite deliberately broken.

### Prompt Context
**User prompt:** See Step 1.
**Assistant interpretation:** Deliver the guide before implementing and keep a record of substantive evidence.
**Inferred user intent:** A sound structural fix, not merely a cosmetic change.
**Commit (documentation):** c166400 — "docs(PV-MARKDOWN-030): design structural markdown rendering".

### What I did
- Wrote the guide with current pipeline, AST proposal, vault integration, API pseudocode, tradeoffs and staged tests.
- `go test ./internal/parser ./pkg/vault` passed before adding new cases.
- Uploaded the guide: `OK: uploaded PV MARKDOWN 030 Markdown Pipeline Design.pdf -> /ai/2026/11/09/PV-MARKDOWN-030`.
- Captured target regression test in `scripts/01-softbreak-regression-test.go.txt`; it can be copied to `internal/parser/softbreak_test.go` to reproduce.
- Added the `markdown` docmgr topic after `docmgr doctor` reported it unknown; subsequent doctor passed.

### Why
- A real source-line break test is necessary because an earlier callout test used literal backslash-n text.
- Multi-paragraph callout structure must be tested independently of the wrapping symptom.

### What worked
- The two existing Go packages passed; reMarkable reported cloud upload success (not proof of device sync).
- Explicit two-space and backslash Markdown breaks passed the probe.

### What didn't work
- `go test ./internal/parser -run 'TestSoftBreaksAndExplicitBreaks|TestCalloutContainsStructuredBody' -count=1` failed as expected. Paragraph and list cases each reported `got 1 breaks, want 0`. The callout case reported `missing "<p>"` and showed `First line<br />\nsecond line</p>` without an opening paragraph. This is a real invalid-structure issue, not just whitespace.

### What I learned
- Goldmark invokes AST transformers after inline parsing; an implementation must preserve the inline tree and first-line boundary, including formatted titles and body content.

### What was tricky to build
- Simply removing `WithHardWraps` breaks the existing callout title/body split. Replacing it requires safely separating inline nodes and preserving nesting, not splitting HTML or assuming every child has a source position.

### What warrants a second pair of eyes
- Extension design and inline-node boundaries, HTML escaping, nested callouts, and frontend toggle behavior.

### What should be done in the future
- Implement and test the structural callout renderer, then remove hard-wrap rendering. Migrate wiki links and math only with their own regression coverage. Deploy separately.

### Code review instructions
- Compare `internal/parser/parser.go:728` with the probe, copy the probe into the parser package, and rerun the command above. Review `web/src/components/organisms/NoteHtml/NoteHtml.tsx` for toggle assumptions.

### Technical details
- `parser.Parse` runs Goldmark block parsing, inline parsing, and then AST transformers in `github.com/yuin/goldmark/parser/parser.go` (v1.8.2). No production parser code was modified in this step.

## Step 3: Implement a structural callout slice and normal soft breaks

I moved callout recognition before Goldmark inline parsing, when the first physical source line is still represented as a segment, and moved rendering to dedicated AST nodes. This separates the callout header from its ordinary block children without using HTML line breaks. The remaining wiki-link and math pipelines have not yet been migrated.

### Prompt Context
**User prompt (verbatim):** "Implement, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill).

Let's get going, we can test things properly once the work is done."
**Assistant interpretation:** Implement the structural design in useful checkpoints, then run meaningful validation at the major boundary rather than repeatedly running broad suites.
**Inferred user intent:** Complete the work, not just produce a plan.

### What I did
- Added `internal/parser/callout.go`: a paragraph transformer extracts the first-line header, inline parsing handles its title, a postorder AST transformer replaces callout blockquotes, and a node renderer composes the wrapper with regular child rendering.
- Removed `html.WithHardWraps()` and the string-based `renderCallouts` post-pass from `internal/parser/parser.go`.
- Promoted the regression probe to `internal/parser/softbreak_test.go` and added `callout_ast_test.go` covering formatted titles, nested callouts, normal blockquotes, math, wiki links, and hard breaks.

### Why
- A paragraph transformer runs before inline parsing, unlike an AST transformer. It can split on source lines without dissecting a formatted inline tree. The AST transformer then handles nested callouts bottom-up.

### What worked
- `go test ./internal/parser -run 'TestSoftBreaksAndExplicitBreaks|TestCalloutContainsStructuredBody|TestCallout|TestCollapsibleCallout|TestOpenCallout' -count=1` passed.
- `go test ./internal/parser ./pkg/vault` passed.
- `go test ./internal/parser -run TestStructuralCalloutComposition -count=1` passed.

### What didn't work
- No unexpected failures during this step. The preceding expected failure is recorded in Step 2.

### What I learned
- Goldmark's paragraph transformer provides a practical narrow structural seam; it lets inline title markup be parsed by the normal parser after the header is identified.

### What was tricky to build
- A blockquote can receive more blocks after its first paragraph is closed, so the paragraph transformer must not replace the entire still-open quote. It inserts a title node and leaves the quote in place; the later AST pass replaces it after all block parsing.

### What warrants a second pair of eyes
- Goldmark paragraph-transform ordering, nested callout traversal, behavior of lazy blockquote continuations, frontend folding, and escaping of source-derived values.

### What should be done in the future
- Review the complete pipeline architecture and implement remaining requested pieces; run full integration validation once feature work reaches a coherent boundary.

### Code review instructions
- Start at `internal/parser/callout.go` and its registration in `Parse`; run `go test ./internal/parser ./pkg/vault` and inspect the rendered output for nested and multi-block callouts.

### Technical details
- The type grammar limits class names to ASCII letters, digits and underscore. Header title children are rendered through Goldmark, not interpolated as attributes. `+` retains the existing always-open behavior and `-` starts collapsed.

## Step 4: Package the extension and validate the vault boundary

I packaged the three callout components as one Goldmark extension and checked that resolved wiki links, heading fragments, math and a list survive inside a callout through the real vault rebuild. A source review caught a subtle traversal bug in the first implementation before release: replacing a blockquote while Goldmark walks its siblings can detach it and prevent later siblings from being visited. The transformer now collects nodes in postorder and replaces them after traversal.

### Prompt Context
**User prompt:** See Step 3.
**Assistant interpretation:** Finish the vertical slice and validate meaningful integration once the work is assembled.
**Inferred user intent:** A composable, robust feature rather than a patch tied to HTML formatting.
**Commit (code):** 073797d — "feat(parser): render callouts structurally and honor soft breaks".

### What I did
- Encapsulated paragraph transformer, AST pass and renderer in `calloutExtension.Extend`.
- Changed AST replacement to collect-first, then mutate in postorder; added a sibling-callout regression.
- Added `pkg/vault/callout_pipeline_test.go`; checked the guide identifies baseline versus implemented slice.
- Rendered the actual JavaScript operating systems note with the existing `01-render-note` script.

### Why
- Registering the extension as a unit makes its passes composable and prevents callers from enabling only half of the feature.
- `ast.walkHelper` advances with `c.NextSibling()` after visiting each child; `ReplaceChild` detaches the old child. Mutating during traversal could skip a second callout.

### What worked
- `go test ./internal/parser ./pkg/vault` passed after the fix; sibling and nested callouts and vault resolution are covered.
- The first code commit's required pre-commit hooks passed `GOWORK=off go test ./...`, the Python plugin unittest, and backend lint with zero issues.
- `go run ./ttmp/2026/08/09/PV-MATHJAX-018--mathjax-support-for-latex-math-in-published-vault-notes/scripts/01-render-note '/home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/javascript-operating-systems.md'` emitted the affected list item with a newline and no generated `<br />` at the source line ending.
- `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30` and `git diff --check` passed.

### What didn't work
- Source review identified the traversal issue in commit 073797d; no failing command was recorded because it was caught before writing the sibling regression. The fix and regression are in the subsequent commit.

### What I learned
- Postorder traversal alone is insufficient if mutation detaches the node while the walker still needs its sibling pointer.

### What was tricky to build
- The nested transformation must run before the containing quote, while sibling traversal must remain stable. A postorder collection followed by replacement satisfies both constraints without mutating the traversal.

### What warrants a second pair of eyes
- Real-world callout syntax beyond the tested subset, `+` fold semantics, trust of raw HTML, and the vault memory cost of a future full-document IR.

### What should be done in the future
- This checkpoint does not implement AST-based wiki links, math, or vault reference resolution; the guide explicitly marks those as future work. Deploying the backend and live browser validation are also separate and have not been performed.

### Code review instructions
- Read `calloutExtension.Extend` and `calloutASTTransformer.Transform`, then run `go test ./internal/parser ./pkg/vault` and inspect `pkg/vault/callout_pipeline_test.go`.

### Technical details
- The user-facing fix applies soft breaks uniformly to paragraph, list and callout body text; explicit hard breaks continue to render as `<br />`.

## Step 5: Record the validated implementation checkpoint

The follow-up code commit packaged the extension and fixed the sibling traversal issue. Its pre-commit checks passed the full Go suite, backend lint, and Python plugin test. No deployment was performed; the broader wiki-link and math AST migration remains future work.

### Prompt Context
**User prompt:** See Step 3.
**Assistant interpretation:** Commit and document the completed vertical slice with evidence.
**Inferred user intent:** Deliver working code with a useful review trail.
**Commit (code):** 70c126b — "fix(parser): preserve sibling traversal in callout extension".

### What I did
- Completed task bookkeeping and recorded the checkpoint.

### Why
- The ticket must distinguish implemented callouts from proposed later AST migrations and deployment.

### What worked
- Commit 70c126b pre-commit hooks: `GOWORK=off go test ./...`, backend lint (zero issues), and `python3 -m unittest plugins/test_retro_plugin.py` passed.

### What didn't work
- No failures in this checkpoint.

### What I learned
- A full passing backend suite does not prove that the deployed server is running this branch.

### What was tricky to build
- No additional implementation in this checkpoint.

### What warrants a second pair of eyes
- Review callout source segment handling and security/trust assumptions, plus behavior of existing notes that relied on hard wrapping.

### What should be done in the future
- Separately deploy and verify the actual page; plan the remaining wiki-link, math and vault-resolution migration if desired.

### Code review instructions
- Review commits 073797d and 70c126b and run `go test ./...`.

### Technical details
- Previously requested reMarkable upload was performed before implementation, so the delivered copy is the design baseline rather than the updated checkpoint note.
