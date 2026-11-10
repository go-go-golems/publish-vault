---
Title: Implementation diary
Ticket: PV-MARKDOWN-030
Status: active
Topics: [markdown, backend]
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation and implementation record
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
