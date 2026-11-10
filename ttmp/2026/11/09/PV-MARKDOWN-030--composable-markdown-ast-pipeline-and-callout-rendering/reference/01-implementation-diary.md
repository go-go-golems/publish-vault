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

## Step 6: Start the broad AST migration

The user requested the broader migration, including coherent structural treatment of wiki links, math and vault-specific resolution. I reopened the work as explicit ticket tasks and traced the existing pipeline and its tests before choosing a first slice.

### Prompt Context
**User prompt (verbatim):** "Now do the broad AST migration.  We want a coherent and strong and simpe codebase."

**Additional user prompt (verbatim):** "Commit at appropriate intervals and keep a detaile ddiary (see diary format skill), backfill if nevessary."

**Assistant interpretation:** Migrate the remaining Markdown features and vault resolution away from placeholder/HTML-string pipelines in validated, committed increments.
**Inferred user intent:** A consistently composable implementation, not just the earlier callout-only fix.

### What I did
- Added explicit tasks for wiki links, math, vault resolution, and end-to-end validation.
- Inspected `Parse`, `wikiLinkHTML`, math placeholders, and `Vault.rebuildHTML`; checked Goldmark v1.8.2 inline parser and reader APIs.

### Why
- The vault requires all notes indexed before cross-note resolution; this requires a separate analysis/index/render lifecycle, not merely replacing a single regex.

### What worked
- Existing branch and three previous implementation commits are present; unrelated untracked files remain untouched.

### What didn't work
- No failures observed in this investigation step.

### What I learned
- Existing tests explicitly expect wiki references inside YAML frontmatter, although frontmatter is not rendered. A structural migration needs an explicit metadata reference policy rather than silently dropping backlinks.

### What was tricky to build
- Math currently masks source before wiki scanning, and Goldmark normally handles code spans and blocks itself. Replacing either scanner changes precedence; reference collection and rendering must use the same nodes.

### What warrants a second pair of eyes
- AST source ownership and memory, source-specific heading IDs, reload invalidation, and preserving frontmatter reference behavior.

### What should be done in the future
- Implement incremental parser and vault migrations, validate affected behavior, and record each meaningful commit.

### Code review instructions
- Start at `internal/parser/parser.go:56`, then `internal/parser/math.go` and `pkg/vault/vault.go:616`.

### Technical details
- Goldmark inline parsers register trigger bytes and consume a `text.Reader`; their registration priorities determine precedence with standard Markdown link parsing.

## Step 7: Parse wiki links into AST nodes

I replaced the wiki-link body pre-pass with a Goldmark inline extension. The same node now drives both HTML emission and reference collection, so Goldmark's own code-block and code-span recognition determines where links exist. Frontmatter links remain a deliberate separate metadata input.

### Prompt Context
**User prompt:** See Step 6.
**Assistant interpretation:** Begin the broad migration with one independently validated structural feature.
**Inferred user intent:** Fewer independent parsers and a simpler source of truth.

### What I did
- Added `internal/parser/wikilink_ast.go`; `Parse` explicitly parses a document, analyzes wiki nodes, then renders it.
- Removed body wiki-link source replacement and its redundant code-region scanner from `parser.go`. Existing math masking and HTML resolution remain for later phases.
- Added `wikilink_ast_test.go` for frontmatter, callout titles/bodies, inline code, fenced code and indented code.

### Why
- The old regex scanner and renderer performed separate analyses. They omitted indented code recognition, which could create links and backlinks from code. Goldmark's syntax tree can own that distinction.

### What worked
- `go test ./internal/parser ./pkg/vault` passed after the migration, including the new code-context test and existing math/wiki cross-feature tests.

### What didn't work
- One edit command did not match the exact original text (`Could not find edits[1] in internal/parser/parser.go`). I corrected the exact match and retried successfully; no partial edit was applied.

### What I learned
- Frontmatter references have an existing behavior separate from rendered content, so they cannot be recovered by walking only the Markdown document AST.

### What was tricky to build
- Math is still masked before wiki parsing; the node renderer passes the existing span context to the legacy output formatter to preserve math in link labels and attributes. This is a transitional boundary, not completion of the requested migration.

### What warrants a second pair of eyes
- Inline parser precedence and source line boundaries, frontmatter reference deduplication, HTML escaping and nested markup.

### What should be done in the future
- Remove remaining math sentinels and migrate vault resolution away from rewriting HTML; expand integration tests.

### Code review instructions
- Review `wikilink_ast.go`, its `Parse` registration and the new tests; run `go test ./internal/parser ./pkg/vault`.

### Technical details
- Link recognition currently uses Goldmark inline trigger bytes `[` and `!` with priority 50. References come from node traversal; YAML metadata references use a separate simple scan.


**Commit-hook failure and recovery:** The initial `git commit -m 'refactor(parser): parse wiki links into AST nodes'` was rejected by backend lint: `internal/parser/wikilink_ast.go:28:71: S1016: should convert e (type wikiLinkExtension) to wikiNodeRenderer instead of using struct literal`. The full Go tests and Python plugin test passed in the same hook. Changed the equivalent struct literal to the suggested explicit type conversion and retried the commit.

## Step 8: Make wiki nodes carry semantic fields and escape display text

I tightened the initial wiki-node implementation so parsing the link syntax happens once: the node carries target, alias, heading and embed status, and both analysis and rendering consume those fields. During review I also found that the old renderer inserted an alias directly into HTML; the renderer now escapes that text.

### Prompt Context
**User prompt:** See Step 6.
**Assistant interpretation:** Eliminate duplicate interpretation and make rendering safer as part of the structural migration.
**Inferred user intent:** A simple, coherent representation with predictable HTML output.
**Commit (code):** 9d47f60 — "refactor(parser): parse wiki links into AST nodes".

### What I did
- Refactored `wikiLinkNode` to hold `WikiLink` semantic fields and updated the existing output function and AST reference visitor to use them.
- Escaped wiki link display text and added `TestWikiLinkDisplayIsEscaped`.

### Why
- An AST node that stores only raw source and reparses it separately in two consumers would preserve an avoidable duplication. Text inside an HTML anchor must not be interpreted as markup.

### What worked
- `go test ./internal/parser ./pkg/vault` passed, including existing math-in-link cases and the new escaping regression.

### What didn't work
- No failures observed in this step.

### What I learned
- Existing math sentinel restoration still participates in both reference analysis and output; semantic wiki nodes alone do not finish the math migration.

### What was tricky to build
- Math placeholders embedded in a link must survive text escaping until the later math restoration pass, whereas raw alias markup must not be emitted as trusted HTML.

### What warrants a second pair of eyes
- Wiki alias escaping and inline-math labels, plus any authored notes that relied on raw HTML in aliases.

### What should be done in the future
- Implement structural math and vault resolution; keep the overall migration tasks open.

### Code review instructions
- Review `wikiLinkNode`, `wikiReferences`, `wikiLinkHTML`, and `TestWikiLinkDisplayIsEscaped`; run `go test ./internal/parser ./pkg/vault`.

### Technical details
- The HTML attribute and text escaping use `html.EscapeString`; the transitional math sentinels are restored after rendering.

## Step 9: Record the requested report and remaining migration boundary

The user requested a detailed textbook-style project report in the Obsidian vault after the implementation is finished, followed by a vault commit and push. I inspected the vault Git status before taking any action; existing untracked research and transcript directories must remain untouched. The report is intentionally not written or pushed at this intermediate checkpoint because the broad migration is still in progress.

### Prompt Context
**User prompt (verbatim):** "once you're finished,  write a detailed project report for the obsidian vault as a deep dive technical analysis blog post using a textbook writing style (no analogies, see skill).      
 Commit and push the bsidian vault when done (go-go-parc vault)."
**Assistant interpretation:** After finishing the migration, write an evidence-backed report in a new vault note and commit/push only the intended vault changes.
**Inferred user intent:** A durable, accurate account of the completed implementation, not a premature success report.
**Commit (code):** 0d184dc — "refactor(parser): store semantic wiki link fields".

### What I did
- Read the Obsidian vault writing and textbook-authoring skills; inspected `go-go-parc` status and remote.
- Confirmed `go-go-parc` has unrelated untracked `Research/2026/11/` and `Transcripts/2026/11/` directories.
- Confirmed commit 0d184dc's hooks passed full Go tests, backend lint and the Python plugin test.

### Why
- The report must describe the actual final architecture and cannot truthfully be called a completed-migration report while math and vault HTML rewriting remain.

### What worked
- `git -C /home/manuel/code/wesen/go-go-golems/go-go-parc status --short` and branch/remote inspection succeeded. The active vault branch is `main`, tracking `origin/main` at the time of inspection.

### What didn't work
- No failures in this checkpoint. The earlier lint failure is recorded in Step 7.

### What I learned
- There is pre-existing untracked vault content; staging the whole vault would risk including unrelated files.

### What was tricky to build
- The existing math scanner can identify regions spanning physical lines, including inline and display forms; replacing it requires preserving precedence with code and wiki links without relying on the old HTML sentinel restoration. Vault rebuilds additionally depend on cross-note heading indexes and reload behavior.

### What warrants a second pair of eyes
- Math precedence and block placement, reference resolution and document/source lifetime, as well as vault staging before any push.

### What should be done in the future
- Complete and validate math and vault AST migration, then write, review, commit and push the requested vault report without staging unrelated files.

### Code review instructions
- Review commits 9d47f60 and 0d184dc, then read `internal/parser/math.go` and `pkg/vault/vault.go:616` for the remaining non-AST pipeline.

### Technical details
- No vault files were changed, committed, or pushed during this checkpoint.

## Step 10: Map the remaining all-at-once migration boundary

The user directed that all remaining features be migrated before running validation, rather than repeatedly testing intermediate edits. I inspected the math scanner, Goldmark inline parsing, and vault rebuild dependencies to identify what must be replaced together. No implementation changes or validation commands were made in this step.

### Prompt Context
**User prompt (verbatim):** "migrate them all, and _then_ validate. easier to do it all at once than do it every couple of edits."
**Assistant interpretation:** Complete math and vault resolution migrations as a coherent feature boundary before running the integration gate.
**Inferred user intent:** Avoid repeated broad validation between small edits while still delivering a complete structural design.

### What I did
- Inspected `internal/parser/math.go`, `pkg/vault/vault.go`, and Goldmark's inline parser loop; identified remaining source sentinels and HTML rewrite functions.
- Verified the existing task list still records math, structural vault resolution, final validation and the post-completion Obsidian report as unfinished.

### Why
- Display math can span lines and block boundaries, while vault links require a complete note and heading index. A simple inline parser or a renamed string pre-pass would not satisfy the requested migration.

### What worked
- Source inspection identified Goldmark's parser trigger rule and the vault's reload-dependent resolution boundary.

### What didn't work
- No commands failed; implementation has not yet been completed.

### What I learned
- Goldmark's inline loop only invokes parsers at selected punctuation, spaces, or the beginning of a line; current private-use Unicode sentinels cannot simply be registered as normal trigger bytes. Display math also needs a block-aware design.

### What was tricky to build
- A structural renderer must retain or recreate document ASTs with their source bytes, preserve exact generated heading IDs, and render repeatedly against changing vault indexes without mutating shared documents. The memory and reload implications require an explicit implementation rather than wrapping the old HTML post-passes.

### What warrants a second pair of eyes
- Multiline math within blockquotes and lists, AST source ownership and retained memory, and correctness under reloads and missing references.

### What should be done in the future
- Implement the complete math and resolver migration together, then run the full validation gate and produce the requested Obsidian report.

### Code review instructions
- Begin with `internal/parser/math.go:54`, Goldmark `parser/parser.go:1200`, and `pkg/vault/vault.go:616`.

### Technical details
- No code or vault changes were made and no tests were run in this step.
