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
    - Path: repo://internal/parser/document.go
      Note: Source ownership and structural renderer
    - Path: repo://internal/parser/math_ast.go
      Note: Display and inline math AST
    - Path: repo://internal/parser/softbreak_test.go
      Note: Soft and explicit line break regression coverage
    - Path: repo://internal/parser/table_ast.go
      Note: Malformed table data preservation
    - Path: repo://pkg/vault/callout_pipeline_test.go
      Note: Vault resolution integration coverage
    - Path: repo://pkg/vault/vault.go
      Note: Deterministic indexing and render-time resolution
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

## Step 11: Connect math and vault resolution through document nodes

Following the user's instruction to migrate all components before the validation gate, I started the integrated replacement without running tests. The work in this step is a working-tree implementation, not a qualified checkpoint: the entire migration and its affected tests must be completed and validated together.

### Prompt Context
**User prompt:** See Step 10.
**Assistant interpretation:** Connect structural math and vault resolution before running the full validation gate.
**Inferred user intent:** Avoid repeated test cycles while the pipeline is intentionally between designs.

### What I did
- Added `math_ast.go` with Goldmark inline and block math nodes, parsers, a display-block promotion pass, and renderers.
- Added `document.go` with an immutable source-owned AST, heading index, render-time resolver and structural image rendering; integrated wiki-link rendering with resolver callbacks.
- Reworked `Vault.rebuildHTML` to render from documents against current note, heading and asset indexes rather than regex-rewriting rendered HTML. Removed the old HTML rewrite and math sentinel implementations.
- Started migrating tests that targeted removed helper functions to document/resolver behavior; adapted the historical parser probe to the new API.

### Why
- Cross-note resolution occurs only after indexing, and needs to be re-evaluated after a reload. Source-owned syntax trees retain the information required to do that without serializing and reparsing HTML.

### What worked
- Source edits and formatting completed; no behavioral claim or passing test is asserted at this intermediate point.

### What didn't work
- One `edit` call for `document.go` failed before applying changes because its third old-text match was not unique; I retried the two specific edits without the ambiguous replacement.

### What I learned
- Standard Markdown images and raw HTML image tags have separate Goldmark node types. The latter require tokenizing the raw HTML fragment to change only image source attributes, not regex rewriting the whole page.

### What was tricky to build
- Goldmark parses block structure before inline nodes; display math requires a block parser for multiline blocks and an AST pass to lift inline display nodes out of paragraphs. Vault rebuild callbacks must not reacquire the lock held by the vault.

### What warrants a second pair of eyes
- This unvalidated implementation needs especially close review of multiline/unterminated math, nested blockquotes, heading IDs with formatting/math, source lifetime and memory, URL safety, raw HTML and reload failure semantics.

### What should be done in the future
- Complete all migration edits and test replacements, then run the main validation gate, diagnose and fix failures, update the design and commit. The requested Obsidian report follows a fully verified migration.

### Code review instructions
- Inspect the current diff in `internal/parser/document.go`, `math_ast.go`, `wikilink_ast.go`, and `pkg/vault/vault.go`. Do not treat this intermediate step as tested.

### Technical details
- The document retains its owned source bytes and Goldmark nodes; rendering uses a fresh renderer with optional callbacks for note, heading, wiki-image and ordinary image resolution. Existing test baselines are being migrated to the new document API.


**Working-tree design refinement:** The initial implementation retained every document AST in the vault. To avoid retaining the entire vault's parse trees, the current work instead keeps each source and compact heading index, then parses and renders one note at a time during rebuild. Rendered HTML is accumulated and published only after all notes render. This remains unvalidated pending the requested integrated gate. The previous statement that vault notes retain the syntax tree describes the discarded intermediate edit; `Document` itself still owns its source and tree while it exists.


**Integrated gate failure 1:** `GOWORK=off go test ./...` failed at compile time: `internal/parser/parser.go:396:9: undefined: ast`. A vestigial `var _ = ast.KindDocument` remained after moving AST code into `document.go` and removing the import. Removed that no-longer-needed placeholder before continuing failure triage.


**Integrated gate recovery and additional evidence:** After removing the stale AST import placeholder, `GOWORK=off go test ./internal/parser ./pkg/vault ./pkg/watcher` and then `GOWORK=off go test ./...` passed. `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher` passed. Rendering the actual JavaScript operating systems note showed no forced `<br>` at the reported source line. `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30` passed. A combined `docmgr doctor ...; git diff --check` command exited 2 because `internal/parser/parser.go:394: new blank line at EOF`; ran `gofmt -w internal/parser/parser.go` to remove the excess blank line. Additional validation remains before qualification.


**Integrated CI gate failure 2:** `make ci-check` passed format checking but stopped at backend lint: `pkg/vault/vault_test.go:1019:15: Error return value of v.rebuildHTML is not checked (errcheck)` and `internal/parser/math_ast.go:112:6: variable close has same name as predeclared identifier (predeclared)`. Updated the test to check the newly returned error and renamed the local variable `closer`. The remaining CI stages were not reached by that attempt and must be run after the fixes.


**Integrated CI gate failure 3:** After fixing the lint findings, `make ci-check > /tmp/pv-markdown-030-ci.log 2>&1` passed formatting, lint, logcopter check, all Go/plugin tests and gosec, then failed at `govulncheck ./...` (exit 3): 12 called standard-library vulnerabilities were reported for Go 1.26.6, including GO-2026-6809 fixed in Go 1.26.10, and HTTP/2 findings also identified `golang.org/x/net@v0.57.0`, fixed in v0.60.0. The remaining frontend stages did not run. `GOTOOLCHAIN=go1.26.10 go version` succeeded. Updated the module and Docker builder to Go 1.26.10 and used `GOWORK=off GOTOOLCHAIN=go1.26.10 go get golang.org/x/net@v0.60.0`; Go also upgraded its related x/crypto, x/mod, x/sync, x/sys, x/term, x/text and x/tools dependencies. Revalidation of the dependency changes is required.


**Vulnerability-gate recovery:** The first `GOWORK=off govulncheck ./...` after the Go and dependency bump failed to load packages with `go: updates to go.mod needed; to update it: go mod tidy`. Ran `GOWORK=off go mod tidy` (which removed the now-redundant toolchain directive while retaining `go 1.26.10`) and repeated the scan. It reported `No vulnerabilities found. Your code is affected by 0 vulnerabilities.` It also reported two vulnerabilities in imported packages and two in required modules for which it found no called vulnerable symbols; these are not represented as zero dependency findings. Full CI and build still need a fresh pass.


**Corpus comparison and table regression:** A 2,372-note inventory comparison found unchanged note, wiki-reference, callout and image counts, but differences in math and backlink totals. Inspection showed the baseline wiki suffix index used randomized map iteration; canonical slugs could also be shadowed by aliases. Changed index construction to reserve canonical slugs first and process alias candidates in sorted order; two full-vault inventory runs then produced identical results. The baseline and new indices can assign different ambiguous aliases, a deliberate deterministic behavior change. Some math count changes came from display nodes inside non-paragraph inline containers, so the inline renderer now retains the `math-display` class in those containers. Inspection also found source tables with one header column and additional body cells: the old math pre-pass accidentally broke such tables and rendered later cells as paragraphs, while unmodified Goldmark silently truncated them. Added a structural paragraph transformer that expands only malformed tables with extra cells, preserving all cell text and alignment, and a code-span escape pass.

**Table implementation diagnostics:** `GOWORK=off go test ./internal/parser ./pkg/vault` first failed to compile `table_ast.go` because chained pointer-receiver calls produced non-addressable `text.Segment` temporaries (`cannot call pointer method TrimRightSpace` and `WithStop`). Split them into named assignments. The next run failed `TestUnevenTableKeepsAllCells`: expected `<code>a|b</code>`, got `<code>a\|b</code>`. The test fixture had authored two backslashes (`a\\|b`), for which retaining one is correct; corrected it to one escaped pipe. The targeted tests then passed. This regression and fix are not yet a full delivery gate.

**Performance experiment:** The ticket's `scripts/02-vault-load` loaded the same real vault (2,372 published notes) with the first structural implementation in 32.064s and 212.2 MiB live heap after GC; a separately archived `HEAD` baseline took 17.439s and 215.4 MiB. Avoiding redundant metadata/reference/excerpt analysis during rebuild using `ParseForRendering` reduced a subsequent run to 17.964s and 212.1 MiB. These are single local observations, not statistically controlled benchmarks or peak-memory measurements.


**Unclosed math across containers:** `GOWORK=off go test ./internal/parser -run TestUnclosedMathInQuoteDoesNotConsumeOutsideContent -count=1` first failed because a `$$` in a later container was treated as the closer, producing `<div class="math math-display">unclosed</div>` inside the quote. Restoring unmatched source as a paragraph preserved the delimiter, but a stronger test with a heading inside the quote then failed because the heading was swallowed into the paragraph. Changed lookahead to require the enclosing blockquote markers on every intervening physical line; the targeted test and `TestMathBlockInCalloutAndEscapedCurrency` subsequently passed together. Wider nested-list and quote cases still warrant review and full validation.

## Step 12: Re-read the design and continue the integrated review

The user asked me to read the design and continue with a detailed diary and appropriate commits. I reread the full ticket design, including the explicit distinction between the original baseline, the AST target, resolver/source lifetimes, test matrix and raw-HTML trust policy. The design's implementation-checkpoint banner is now stale relative to the working tree and must be revised after the implementation is qualified.

### Prompt Context
**User prompt (verbatim):** "read design doc and continue keeping a diary. commit at appropriate intervals, and keep a detailed diary as you work (see the diary skill for the diary format. backfill the diary if you forgot to write it so far)"
**Assistant interpretation:** Ground the remaining implementation in the full design and keep an accurate chronological record, including missing validation evidence.
**Inferred user intent:** Complete the migration without losing the architectural rationale or the actual debugging history.

### What I did
- Read the complete `design-doc/01-markdown-pipeline-architecture-and-implementation-guide.md`.
- Inspected the current math and table extensions and open ticket tasks; preserved unrelated untracked files.
- Rechecked the targeted unclosed-quote math test after the container-aware fix; it passed along with the callout/math case.

### Why
- The design explicitly calls out source ownership, cross-note resolution, math/code precedence and truthful separation of proposal versus tested implementation. Those are the critical remaining review boundaries.

### What worked
- `GOWORK=off go test ./internal/parser -run 'TestUnclosedMathInQuoteDoesNotConsumeOutsideContent|TestMathBlockInCalloutAndEscapedCurrency' -count=1` passed after the quote fix.

### What didn't work
- The earlier stronger unclosed-quote test failed before the fix because `## Preserved heading` appeared as paragraph text; its exact diagnostic is in the preceding checkpoint. Current success only covers the named cases, not every nested-list boundary.

### What I learned
- The original guide remains useful as a baseline and rationale, but its prominent claim that math and resolution have not been implemented must not remain the sole current-state account.

### What was tricky to build
- Closing-delimiter lookahead is performed on the original bytes before Goldmark finishes container parsing. A delimiter in a different container cannot be treated as the current block's closer without swallowing syntax.

### What warrants a second pair of eyes
- List-item and nested-quote boundaries, table syntax complexity, new runtime and dependency versions, and the discrepancy between the historical design and actual code.

### What should be done in the future
- Complete a consolidated review and validation, document the actual pipeline, then commit and write the requested vault report.

### Code review instructions
- Read the ticket design, then `internal/parser/math_ast.go`, `table_ast.go`, `document.go` and `pkg/vault/vault.go`.

### Technical details
- The working tree is not yet committed; the active goal and ticket remain open. No deployment or Obsidian report has been performed.

## Step 13: Measure whether cross-container math occurs in the vault

The user asked whether the recently investigated cross-boundary math problem is present in actual notes, and how many documents are affected. I wrote a small lexical source audit and inspected representative matches. It detects potential opener/closer pairs, not Goldmark's complete block structure or the published-note set, so the findings are explicitly bounded.

### Prompt Context
**User prompt (verbatim):** "is math across boundaries an issue in our documents, or is that a broader valid compelxity? how many documents are affected?"
**Assistant interpretation:** Distinguish demonstrated vault data from hypothetical Markdown edge cases, with counts and caveats.
**Inferred user intent:** Decide whether the parser complexity is proportionate to actual needs.

### What I did
- Added `scripts/04-math-container-audit.py`, which scans Markdown outside `.git` and `node_modules`, skips apparent fenced code, and approximates quote/list scope for multiline display delimiters.
- Ran it against `/home/manuel/code/wesen/go-go-golems/go-go-parc` and inspected representative quote and list matches.

### Why
- Real multiline math inside blockquotes is distinct from the malformed case where an opener uses a later container's closer. A lexical scan can bound plausible examples but cannot prove full parser semantics.

### What worked
- The corrected probe scanned 2,432 Markdown files and found 104 same-quote-scope candidate pairs across 9 files, zero quote crossing candidates, and six list crossing candidates in one file. Manual inspection of the six list candidates shows malformed escaped citation syntax in `Research/Institute/Research/2026/04/16/philosophical-foundations-of-mathematics/sources/web/arxiv-lean-atp-interface.md`, not six confirmed genuine display equations spanning containers.

### What didn't work
- The first probe reported 77 quote candidates across 9 files because it incorrectly counted a closing delimiter as a new opener. For example, the ordinary quote in `Transcripts/2026/07/22/CHATGPT TRANSCRIPT - Colon in Type Theory.md` lines 157–165 contains valid separately paired quoted equations. Changed the probe to consume the closer before looking for the next pair and reran it. Neither run is a full Markdown parse or proof of absence.

### What I learned
- Quoted multiline math is a real corpus feature. The specific unclosed-quote/opener-matching-outside-container regression was demonstrated with a synthetic test, not confirmed in a vault file.

### What was tricky to build
- Raw escaped dollar signs, quoted markers and adjacent formulas can make a regex counter confuse closing delimiters with opening ones. The reported list matches are malformed citation text rather than clear mathematical blocks.

### What warrants a second pair of eyes
- The audit does not handle every nested list, lazy blockquote continuation, escaped delimiter or publication filter; count actual production impact only after a full parser-based corpus audit.

### What should be done in the future
- Keep the fix proportionate; avoid further general-purpose parsing machinery solely for a synthetic edge case unless real evidence or a concrete correctness requirement justifies it.

### Code review instructions
- Run `python3 ttmp/2026/11/09/PV-MARKDOWN-030--composable-markdown-ast-pipeline-and-callout-rendering/scripts/04-math-container-audit.py /home/manuel/code/wesen/go-go-golems/go-go-parc` and inspect reported files around their line numbers.

### Technical details
- These are physical source files, not deduplicated published notes; the scanner skipped `.git` and `node_modules` and did not apply `.vault-ignore` or `publish: false`.

## Step 14: Qualify the integrated design and document the implementation

I finished the structural renderer integration and added a separate current-state guide, leaving the original design labeled as historical. I also checked the analogous list-container case after the quote failure: an unclosed math opener could swallow a later list heading when its apparent closer was outside the list. The lookahead now checks list indentation as well as quote markers; this remains a scoped container check rather than a new general Markdown parser.

### Prompt Context
**User prompts:** See Steps 10, 12 and 13. The user explicitly asked whether the edge-case complexity was justified and to avoid repeated broad validation.
**Assistant interpretation:** Finish the migration, keep the solution proportionate to observed documents, then run a consolidated delivery gate.
**Inferred user intent:** A coherent implementation with an honest evidence-backed account, not maximum parser complexity.
**Commit (dependencies):** e4e4b47 — "build: update Go and networking dependencies for security".

### What I did
- Added the implementation guide `design-doc/02-implemented-markdown-document-pipeline-and-operational-guide.md` and marked the original design as a baseline.
- Added list-container opener and valid nested-list/quote math cases. Used Goldmark's list offset and indentation helper in the closer guard.
- Checked the real-vault inventory and runtime behavior; added a deterministic wiki-index collision test and table/data-preservation coverage.
- Built the embedded backend and inspected a local fixture in Playwright, including the rendered paragraph, callout, link, table and MathJax state.

### Why
- The original design described a staged proposal; an intern needs the actual implementation, ownership, API, failure semantics, and explicitly documented behavior changes.
- A matching delimiter in a different list/quote must not cause earlier headings to become paragraph text.

### What worked
- `make ci-check > /tmp/pv-markdown-030-ci-integrated.log 2>&1` passed on the then-current integrated tree: formatting, backend lint, generated logger check, Go and plugin tests, gosec, govulncheck, web type-check and frontend build. Govulncheck reported zero called vulnerabilities but two vulnerabilities in imported packages and two in required modules without called vulnerable symbols.
- `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher` passed on that tree. `GOWORK=off go build -tags embed -o /tmp/pv-030-embed ./cmd/retro-obsidian-publish` and its `--help` passed; Docker registry has `golang:1.26.10-alpine`.
- The embedded server's local fixture had zero hard breaks in the soft-wrapped paragraph, the expected title and `/note/target` link, both table cells, and MathJax state `done` for inline and display math. Screenshot: `reference/assets/embedded-smoke.png`.
- After the list-container fix, `GOWORK=off go test ./internal/parser -run 'TestDisplayMathInNestedList|TestDisplayMathInListWithinQuote|TestUnclosedMathInListDoesNotConsumeHeading' -count=1` passed.

### What didn't work
- The first `TestUnclosedMathInListDoesNotConsumeHeading` run failed: `## Preserved heading` was rendered as paragraph text inside the list. Added ancestor-aware list indentation to the quote-boundary check and reran the targeted cases successfully.
- Playwright refused a screenshot path outside its allowed roots: `File access denied: /tmp/pv-030-smoke.png is outside allowed roots`; saved it under `.playwright-mcp` and copied the final fixture screenshot into the ticket.
- The fixture had no favicon, so browser console showed 404s for `/favicon.svg` and `/favicon.ico`; no other browser errors were reported. This was not a production-page test.
- `git diff --check` reported trailing whitespace in this diary; stripped it and reran successfully. A later exact-text edit failed without applying changes because its match did not match current indentation; reread the file and retried.

### What I learned
- A lexical vault audit found valid multiline quoted math, but did not establish real cross-container mathematical equations. The list guard addresses a concrete synthetic data-loss regression; its test does not prove every possible Markdown nesting pattern.
- The real-vault load after avoiding repeated metadata analysis was 17.964s/212.1 MiB live heap versus an archived baseline 17.439s/215.4 MiB, each a single local run, not a benchmark distribution or peak-RSS measurement.

### What was tricky to build
- `mathCloserInContainer` must inspect ancestors in root-to-leaf order. Goldmark list items carry a continuation offset, and a closing delimiter that is not in the same list or quote cannot justify consuming intervening Markdown blocks.
- `golang.org/x/net/html.Tokenizer.Token()` may mutate its buffer; raw HTML image resolution must copy the token's original bytes before tokenization to avoid changing unrelated HTML attributes.

### What warrants a second pair of eyes
- Ambiguous source delimiters and atypical nested whitespace; behavior of raw trusted HTML; the deliberate deterministic choice for alias collisions; and maintenance cost of the malformed-table extension.

### What should be done in the future
- A final validation pass must cover the latest list and image-renderer edits; earlier CI and browser results predate them. Commit the qualified code and docs, then write and push the requested Obsidian report. Production deployment is separate and has not happened.

### Code review instructions
- Start at the current guide, then `internal/parser/document.go`, `math_ast.go`, `table_ast.go`, `wikilink_ast.go`, and `pkg/vault/vault.go`. Run `make ci-check` and the race tests before qualification.

### Technical details
- `ParseForRendering` assumes the supplied heading index belongs to the exact same source and parser configuration; the vault stores and replaces the source/index pair together.

## Step 15: Complete the integrated validation boundary

I completed the source-owned document, structural math and vault-resolution migration, then ran the consolidated project gate after the final list-boundary and image-resolution changes. The ticket's current-state guide now documents the implementation separately from the original design. Production deployment and the requested post-implementation vault report are separate remaining actions.

### Prompt Context
**User prompts:** See Steps 10, 12 and 13.
**Assistant interpretation:** Qualify the complete implementation once, keep a detailed diary, then publish the separate report.
**Inferred user intent:** A coherent, maintainable pipeline with evidence proportional to the changed integration boundary.

### What I did
- Finished quote/list-aware display-math matching and preserved existing image destinations when an image callback returns empty.
- Updated `design-doc/02-implemented-markdown-document-pipeline-and-operational-guide.md` and marked the earlier proposal historical.
- Marked math and vault structural-resolution tasks complete; retained report and final bookkeeping tasks open.

### Why
- The initial input/newline defect affects callouts, while math and wiki resolution have their own source and vault lifetimes. One combined validation boundary checks those interactions rather than qualifying each in isolation.

### What worked
- `make ci-check > /tmp/pv-markdown-030-ci-qualified.log 2>&1` passed on the final code: format, backend lint (zero issues), logcopter, all Go/plugin tests, gosec, govulncheck, frontend type-check and Vite build. Govulncheck reported zero called vulnerabilities and separately noted two imported-package and two required-module vulnerabilities without called vulnerable symbols.
- `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher` passed; `GOWORK=off go build -tags embed -o /tmp/pv-030-final ./cmd/retro-obsidian-publish` and `/tmp/pv-030-final --help` passed.
- `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30` and `git diff --check` passed. The earlier browser fixture screenshot and its favicon-only errors are described in Step 14; it predates the last narrowly scoped list/image changes.

### What didn't work
- No new validation failures after the fixes; earlier compilation, lint, vulnerability, screenshot path and malformed-container failures are preserved in Steps 11–14.

### What I learned
- On the real vault, the new pipeline preserved aggregate note, reference, callout and image counts; some math differences came from avoiding previously corrupted raw HTML and preserving inline display semantics. Counts alone do not prove full semantic equivalence.

### What was tricky to build
- Tests that once asserted regex rewrites were migrated to source-document and resolver behavior. The current rebuild stages all HTML before publication, but an error after note/index mutation is not a full transactional rollback; this is documented in the guide.

### What warrants a second pair of eyes
- Nested Markdown delimiter boundaries and atypical tabs, trusted raw HTML, the malformed-table extension's maintenance relative to Goldmark, and deliberate alias-collision determinism.

### What should be done in the future
- Commit this qualified checkpoint, then write and push the requested deep-dive Obsidian report. Deploying to the public site is not part of the local validation and has not occurred.

### Code review instructions
- Read the implementation guide and the diff; run `make ci-check` and the race command above. The local comparison probes are in the ticket `scripts/` directory.

### Technical details
- The latest run's full log is `/tmp/pv-markdown-030-ci-qualified.log`; it is local ephemeral evidence rather than a tracked artifact.

## Step 16: Write and deliver the Obsidian technical report

After the implementation commit, I wrote a standalone project report in the vault, reviewed it as a newcomer-facing explanation, rendered its Mermaid diagram through the local embedded application, and pushed only that report. The report distinguishes the implemented branch from an actual production deployment.

### Prompt Context
**User prompt:** See Step 9 for the verbatim report request; Step 12 for the diary and commit request.
**Assistant interpretation:** Produce a detailed technical article without analogies, then commit and push only the intended vault note.
**Inferred user intent:** Preserve durable explanations of the architecture, tradeoffs, evidence and operational status.
**Commit (code):** b4384c3 — "refactor(parser): render vault documents from structural nodes".
**Commit (vault):** 11a6fa1da29c8cea455c8520382ea56da26507aa — "docs: explain Publish Vault structural Markdown pipeline".

### What I did
- Created `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/11/10/PROJECT REPORT - Publish Vault - A Structural Markdown Rendering Pipeline.md` as a new 2,830-word note with frontmatter, concrete examples, a Mermaid diagram, architecture, tradeoffs, tests and source references.
- Read the complete draft and checked that the proposed and implemented designs, source ownership, missing-target behavior, local performance measurements and deployment state were described separately.
- Served a temporary copy through the embedded local application and inspected the rendered diagram. The initial horizontal layout was very small; changing it to vertical improved size. The next image exposed white text on pale fills, so I explicitly set dark text and re-rendered. The final inspected diagram is `reference/assets/report-diagram.png`.
- Fetched `origin/main` before committing; `git rev-list --left-right --count HEAD...origin/main` returned `0 0`. Staged and committed only the new note, then pushed `HEAD:main` without force.

### Why
- The user requested a textbook-style deep dive without analogies and a vault commit/push. A diagram must be inspected as rendered rather than assumed legible from Mermaid syntax alone.

### What worked
- `git -C /home/manuel/code/wesen/go-go-golems/go-go-parc push origin HEAD:main` advanced `b42c6b9..11a6fa1`. `git ls-remote origin refs/heads/main` returned the exact local commit `11a6fa1da29c8cea455c8520382ea56da26507aa`.
- The final local browser showed the revised diagram with readable dark labels. The temporary server was stopped; the vault working tree was clean after push.

### What didn't work
- No command failed in this step. Two visual deficiencies were identified and corrected: an overly compressed horizontal diagram and insufficient text contrast on colored nodes.

### What I learned
- Syntax-valid Mermaid can render at an unusable scale or contrast; both appearance changes required a fresh render and visual review.

### What was tricky to build
- The report describes the initial design's sentinel pipeline as historical, the current document API as implemented, and the live site as not deployed. Conflating those states would mislead future readers.

### What warrants a second pair of eyes
- The report's experimental performance comparison consists of single local samples, not statistical performance claims; the raw HTML trust policy and alias-collision behavior are deliberate technical tradeoffs.

### What should be done in the future
- Deployment is a separate operation. Re-verify the live note after deploying; this report and local tests do not imply production has changed.

### Code review instructions
- Review `b4384c3`, ticket design documents, and the vault note at the path above; verify the vault commit with `git -C /home/manuel/code/wesen/go-go-golems/go-go-parc show --stat 11a6fa1`.

### Technical details
- The source note contains one Mermaid diagram, not an externally linked image. Only the report was staged and pushed to the vault. The local diagram screenshot is investigation evidence in this ticket, not a vault dependency.

## Step 17: Exercise malformed input before the final audit

As a narrow risk check after the structural migration, I added a parser fuzz target seeded with nested math, table syntax, invalid bytes and unterminated delimiters. This is not a replacement for the integrated test suite or a proof that all inputs are correct; it specifically checks that parsing arbitrary bounded input does not panic.

### Prompt Context
**User prompt:** See Steps 10, 12 and 13.
**Assistant interpretation:** Audit the completed implementation with a targeted check for malformed input before declaring completion.
**Inferred user intent:** Robust parsing without an unbounded cycle of repeated broad validation.
**Commit (code):** b4384c3 — "refactor(parser): render vault documents from structural nodes".

### What I did
- Added `FuzzStructuralParse` in `internal/parser/document_test.go`, with a 64 KiB input limit and representative structural seeds.
- Ran `GOWORK=off go test ./internal/parser -run '^$' -fuzz '^FuzzStructuralParse$' -fuzztime=8s -parallel=2`.

### Why
- Math lookahead and table segment handling consume offsets derived from arbitrary authored text, which creates a concrete risk of a panic even when example-based tests pass.

### What worked
- The fuzz command passed after 43,718 executions and discovered 163 additional interesting inputs during the run. The Go fuzz cache is local, not a committed corpus.

### What didn't work
- No failures observed in this step.

### What I learned
- The targeted run provides bounded evidence about parser termination and panics, not exhaustive correctness or proof that every input finishes within a fixed time.

### What was tricky to build
- No production logic changed. The input-size bound limits the test's memory cost; malformed Unicode is included among seeds.

### What warrants a second pair of eyes
- Corpus-specific rendering semantics and adversarial performance remain separate from the absence of panics in this short run.

### What should be done in the future
- Complete the final requirement-to-evidence audit and retain the dated vault report and exact remote push receipt.

### Code review instructions
- Inspect the fuzz target at the end of `internal/parser/document_test.go`; rerun the command above if a reproducible seed is added.

### Technical details
- The previous `make ci-check` qualified production code before this test-only addition; the repository pre-commit hook will rerun its required backend gates for the added test.

## Step 18: Audit and remove duplicate Goldmark registrations

The completion audit found that `newMarkdown` explicitly registered Table, Strikethrough and TaskList in addition to `extension.GFM`, which already registers all three. This duplication came from the baseline but was carried into the structural renderer. I removed the duplicate registrations and added a focused GFM behavior regression.

### Prompt Context
**User prompt:** See Steps 6, 10 and 12; the current tracked goal requires a requirement-by-requirement completion audit without duplicated logic.
**Assistant interpretation:** Inspect the finished code for redundant registration and correct issues rather than declaring completion from earlier test passes alone.
**Inferred user intent:** Keep the migrated codebase simple as well as functional.
**Commit (prior checkpoint):** 291e25d — "test(parser): fuzz structural documents and record delivery".

### What I did
- Confirmed `github.com/yuin/goldmark/extension/gfm.go` registers Linkify, Table, Strikethrough and TaskList.
- Removed the three redundant explicit extensions in `internal/parser/document.go`; retained Footnote, which GFM does not provide.
- Added `TestGFMFeaturesRemainEnabled` and updated stale comments that described removed HTML-rewrite functions as current code.
- Compared a fresh inventory of the same current vault against an archived `HEAD` using the existing ticket probe.

### Why
- Registering the same parser and renderer extensions twice is unnecessary work and makes priority interactions harder to reason about.

### What worked
- `GOWORK=off go test ./internal/parser ./pkg/vault` passed after the change.
- A controlled fresh comparison against archived `HEAD` found 2,373 notes in each run and zero per-note changes in the probe's link, backlink, math, callout, image and broken-marker counts.

### What didn't work
- An initial comparison against `/tmp/pv-030-after4.json` showed 44 changed notes and one added note. That file predated later table/quote changes and the newly created vault report, so it was not a controlled comparison of this edit. I reran the baseline from archived current `HEAD` against the same vault; the controlled result is above. Aggregate equality does not prove byte-for-byte output equivalence.

### What I learned
- A previous local inventory is not a valid baseline after either the implementation or the vault changes. The fresh pair is the relevant evidence for the registration cleanup.

### What was tricky to build
- The user requested fewer repeated broad tests; the narrow tests and corpus inventory address the specific duplication risk before one final project validation gate.

### What warrants a second pair of eyes
- Goldmark extension ordering and priority if the upstream GFM bundle changes in a future version.

### What should be done in the future
- Run the final project validation for this production change, commit it, and finish the goal audit. The vault report remains an accurate account of the architectural implementation; it references the earlier implementation checkpoint rather than this subsequent cleanup.

### Code review instructions
- Compare `newMarkdown` with Goldmark's GFM extension and run `go test ./internal/parser ./pkg/vault`.

### Technical details
- The inventory is a content-free count comparison, not an output diff or a proof of all Markdown behavior.


**Final validation after deduplication:** `make ci-check > /tmp/pv-markdown-030-ci-final-audit.log 2>&1` passed formatting, backend lint, logcopter, all Go/plugin tests, gosec, govulncheck, frontend type-check and the frontend build. Govulncheck reported zero called vulnerabilities and separately noted uncalled findings in dependencies. `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher`, an embedded build and help smoke, `docmgr doctor`, and `git diff --check` also passed. The vault's remote main still points to report commit `11a6fa1da29c8cea455c8520382ea56da26507aa` and the vault working tree is clean.

## Step 19: Audit the completion contract

I mapped each explicit goal requirement to source, tests, validation output, documentation, commits and the verified vault remote. The audit explicitly distinguishes local checks from deployment and bounded corpus measurements from exhaustive equivalence.

### Prompt Context
**User prompt:** See Steps 6, 9, 10 and 12; the active goal requires a fresh completion audit.
**Assistant interpretation:** Verify the requested work and delivery rather than inferring completion from the number of commits or checked tasks.
**Inferred user intent:** A defensible handoff with clear limits.
**Commit (code):** 0185f51 — "refactor(parser): avoid duplicate GFM registrations".

### What I did
- Added `reference/02-completion-audit-and-evidence-map.md` mapping requirements to artifacts and noted limitations.
- Rechecked `docmgr doctor`, ticket tasks, branch status, validation log and the Obsidian vault's `HEAD` and remote main.

### Why
- The user asked for all migrations, validation, a detailed diary and a pushed report; each has a distinct evidence source and must not be inferred from another.

### What worked
- All ticket tasks are checked and doctor passed. `git ls-remote origin refs/heads/main` in `go-go-parc` matched local report commit `11a6fa1da29c8cea455c8520382ea56da26507aa`. Final CI, race, build, local browser and fuzz results are recorded in Steps 15–18.
- Search of the changed production files found no TODO/FIXME/HACK or old sentinel/whole-HTML rewrite function names. The tracked code diff is committed; unrelated pre-existing untracked files remain untouched.

### What didn't work
- No failures observed in this audit step. Prior failed checks and their fixes remain documented in earlier steps.

### What I learned
- Corpus counts, a short fuzz run and a local browser fixture support specific claims but cannot establish exhaustive correctness or production deployment.

### What was tricky to build
- The design report uploaded to reMarkable precedes implementation, whereas the later pushed vault article describes the implemented branch; their delivery evidence and scopes must stay separate.

### What warrants a second pair of eyes
- Raw HTML is deliberately trusted, and unusual Markdown nesting may need future tests if real input demonstrates a new defect. Neither caveat is a claim that production was deployed.

### What should be done in the future
- Deploy and verify the public site only when requested through the deployment workflow.

### Code review instructions
- Start with the evidence map, then follow the linked implementation guide and commits. Check the public page separately after a future deployment.

### Technical details
- The goal requires local implementation, validation, documentation and the vault report push; it does not authorize or require a production deployment.


**Final-revision browser smoke:** Built `/tmp/pv-030-audit` after the GFM deduplication and served the temporary fixture on loopback port 18200. Playwright inspection showed `hardBreaks: 0`, a callout, `/note/target`, both table cells `One` and `Two`, and `done` for the inline and display math elements. Visually inspected `reference/assets/final-audit-smoke.png`; stopped the temporary server. The fixture lacks a favicon and produced the same two expected favicon 404s; this is not a production deployment check.

## Step 20: Share math markup between nodes and wiki labels

While rechecking the completion contract I found one remaining duplicate rendering definition: the math node renderer and the wiki-label renderer each constructed the same MathJax wrapper independently. Both were structural render-time operations, but the duplicated HTML and escaping could diverge later. I moved that output to one `renderMathElement` helper used by both paths.

### Prompt Context
**User prompt:** See Steps 6, 10 and 12; the active goal explicitly requires no duplicated logic.
**Assistant interpretation:** Correct a concrete duplication found during the completion audit before marking the goal complete.
**Inferred user intent:** Keep the final implementation coherent and simple.
**Commit (prior audit):** d868bbf — "docs(PV-MARKDOWN-030): record completion evidence audit".

### What I did
- Replaced duplicate inline/display math tag construction in `internal/parser/math_ast.go` and `wikilink_ast.go` with one helper.
- Ran the focused parser/vault package tests after formatting.

### Why
- MathJax class names and TeX escaping are an output contract; maintaining two independent string constructions risks divergence.

### What worked
- `GOWORK=off go test ./internal/parser ./pkg/vault` and `git diff --check` passed after the change.

### What didn't work
- No failures observed in the focused check. The earlier full CI predates this production edit and must be rerun before the audit can be considered current.

### What I learned
- The first audit's broad search for old sentinel and HTML-rewrite functions did not by itself rule out duplicated markup construction. A focused source review was necessary.

### What was tricky to build
- Wiki-label math is inline even when its source delimiters are interpreted within a label; the helper preserves that existing class/tag choice while allowing block math nodes to request a `div`.

### What warrants a second pair of eyes
- Future changes to MathJax hooks must use the shared helper rather than reintroducing another independent wrapper.

### What should be done in the future
- Run the final project gate and race checks on this change, then update the audit evidence before marking the active goal complete.

### Code review instructions
- Inspect `renderMathElement`, `mathRenderer.RegisterFuncs` and `renderMathLabel`; run `go test ./internal/parser ./pkg/vault`.

### Technical details
- The math delimiter scanner and HTML escaping semantics did not change in this step.


**Shared-math validation:** `make ci-check > /tmp/pv-markdown-030-ci-shared-math.log 2>&1` passed all project stages, including lint, Go/plugin tests, gosec, govulncheck, frontend type-check and build. The scanner reported zero called vulnerabilities and separately noted uncalled dependency findings. `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher`, an embedded build plus help, `docmgr doctor` and `git diff --check` passed. Vault remote main still matched report commit `11a6fa1da29c8cea455c8520382ea56da26507aa`.


**Shared-math final browser check:** Served the freshly built `/tmp/pv-030-shared-math` against the local fixture on loopback port 18201. Playwright reported zero hard breaks in the paragraph, the expected callout and `/note/target` link, both table cells, and `done` for inline and display math. Visually inspected `reference/assets/shared-math-final.png`, then stopped the server. The fixture's absent favicons produce two expected 404s; this is not a live-production check.

## Step 21: Address PR 30 review findings

The review identified wiki links rejected when their math contained brackets, and quadratic copying for many same-line math spans. I reproduced both, then changed wiki recognition and the inline math reader while retaining cross-line code-span behavior.

### Prompt Context
**User prompt (verbatim):** "Address comments: https://github.com/go-go-golems/publish-vault/pull/30"
**Assistant interpretation:** Verify and fix both review findings, validate, push and respond on the PR.
**Inferred user intent:** Make the PR ready for renewed review without merging it.

### What I did
- Introduced math-aware wiki scanning shared by AST links, table protection and frontmatter references, and preserved math-aware wiki parsing in excerpts. Separators inside math no longer split wiki fields.
- Short-circuited math scanning on the current line and stopped multiline collection once the closer is unambiguous; unmatched backticks require further lookahead because a code span can close on a subsequent line.
- Added parser and vault/backlink regressions, a cross-line code-span test and a dense-formula benchmark. Bumped the Go and Docker patch version to 1.26.11 after the security gate detected a newly published standard-library issue.

### Why
- Regex matching treated TeX brackets as wiki syntax; the former math masking hid them. Copying the remaining inline container for each small formula created quadratic allocation.

### What worked
- The regression initially failed for inline, display, heading and table cases; the corrected parser and vault tests pass. `make ci-check` passed after the patch-version update; `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher`, `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30`, and `git diff --check` passed.
- On the same 11th Gen Intel i7-1165G7 with `-benchtime=100ms`, parsing 5,000 `$x$ ` formulas changed from 24.88 ms and 56,841,449 B/op to 6.20 ms and 4,884,864 B/op in the measured runs. These short runs are directional, not a statistical performance study; multiline ambiguous code spans still require lookahead.

### What didn't work
- Initial `go test ./internal/parser -run '^TestWikiLinkMathWithBrackets$' -count=1` failed: all four original cases returned no references and rendered literal wiki delimiters; the table case also split the link into cells.
- A first cross-line test expected an extra source indentation space, but Goldmark's inline container had already removed it; the observed HTML retained the intended later math closer, so the assertion was corrected.
- The first `make ci-check` failed at `govulncheck` with `GO-2026-6859`, HTTP/1 server crash due to a MaxBytesReader header race, `Found in: net/http@go1.26.10`, `Fixed in: net/http@go1.26.11`, and exit 3. The subsequent gate passed with Go 1.26.11.

### What I learned
- Wiki parsing has several consumers beyond rendered links, including frontmatter, excerpts and the table pipe splitter; fixing only the AST parser would leave inconsistent output or lost cells.

### What was tricky to build
- A math closer observed before an unmatched backtick cannot safely terminate early: the backtick can form a code span with a later line, concealing that closer. The reader only stops early when preceding code spans are complete.

### What warrants a second pair of eyes
- Inspect multiline delimiter/code-span boundary semantics and the shared wiki scanner's handling of unusual math and bracket combinations. The benchmark demonstrates the reported dense same-line case, not a worst-case bound for every ambiguous multiline input.

### What should be done in the future
- Review the updated PR, CI results, and deployment separately; do not infer production rollout from local tests.

### Code review instructions
- Start with `internal/parser/wikilink_ast.go`, `math_ast.go`, `table_ast.go` and `pr30_review_test.go`; run `go test ./internal/parser ./pkg/vault`, and the `BenchmarkInlineMathDenseParagraph` benchmark.

### Technical details
- The corpus smoke and browser checks from prior steps predate this review fix. The new evidence covers parser, vault integration, race checks and the project gate, not a new production or browser deployment check.

## Step 22: Address the PR rereview

The subsequent Codex review identified two more regressions: Markdown image destinations reached asset resolution without Goldmark's normal destination processing, and each wiki link copied the remaining source line. I reproduced both before changing the relevant renderer and inline parser.

### Prompt Context
**User prompt (verbatim):** "yes" — in response to the offer to address the two findings on PR 30.
**Assistant interpretation:** Fix both findings, validate the changes, and update the review threads.
**Inferred user intent:** Continue toward a reviewable PR without merging it.

### What I did
- Applied Goldmark `util.URLEscape(destination, true)` before invoking the Markdown image resolver, preserving entity and punctuation unescaping, URL encoding and existing percent escapes.
- Scanned the first wiki-link line without copying it; only multiline fallback buffers input. The AST node still owns its raw link bytes.
- Added parser regressions for named and numeric entities, escaped parentheses and already-encoded URLs, plus a vault image-resolution check and a dense wiki-link benchmark.

### Why
- The previous Goldmark renderer normalized the image URL before the vault's HTML pass. The original wiki parser's buffer held the whole remaining line even when the matching `]]` was only a few bytes away.

### What worked
- `make ci-check`, `GOWORK=off go test -race ./internal/parser ./pkg/vault ./pkg/watcher`, `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30`, and `git diff --check` passed.
- With `-benchtime=100ms` on an 11th Gen Intel i7-1165G7, parsing 5,000 repeated links changed from 90.56 ms and 165,244,936 B/op to 22.19 ms and 16,139,001 B/op. These are short directional measurements, not a worst-case bound on malformed or multiline inputs.

### What didn't work
- Initial `go test ./internal/parser -run '^TestMarkdownImageDestinationNormalization$' -count=1` failed: the resolver received `images/a&amp;b.png`, `images/a\\(b\\).png`, and `images/a&#32;b.png` without normalization. Named entities were double-escaped in rendered output.

### What I learned
- Goldmark's URL normalization includes percent encoding as well as Markdown escapes and HTML character references; using the same helper avoids a subtly different resolver input contract.

### What was tricky to build
- The fast wiki path must copy only the recognized link into the node before advancing the Goldmark reader. The fallback remains necessary for links spanning lines and retains prior recognition behavior.

### What warrants a second pair of eyes
- Check unusual URI schemes and existing percent escapes against Goldmark behavior, and review malformed or multiline wiki-link performance separately from the benchmarked same-line case.

### What should be done in the future
- Await refreshed PR checks and review; deployment remains a separate operation.

### Code review instructions
- Inspect `internal/parser/document.go`, `wikilink_ast.go`, `pr30_review_test.go`, and `pkg/vault/callout_pipeline_test.go`; run parser and vault tests and `BenchmarkWikiLinkDenseParagraph`.

### Technical details
- Raw HTML image handling is unchanged. This work does not establish a browser or production deployment check.
