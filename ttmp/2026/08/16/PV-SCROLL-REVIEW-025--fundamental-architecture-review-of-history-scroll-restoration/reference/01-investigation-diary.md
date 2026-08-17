---
Title: Investigation diary
Ticket: PV-SCROLL-REVIEW-025
Status: active
Topics:
    - frontend
    - react
    - routing
    - ssr
    - ux
    - wiki-link
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/08/15/PV-SCROLL-024--implement-option-a-save-restore-note-scroll-offset-on-back-forward/design-doc/01-design-hand-rolled-scroll-restoration-option-a.md
      Note: Prior accepted design assumptions reviewed and superseded
    - Path: repo://ttmp/2026/08/15/PV-SCROLL-024--implement-option-a-save-restore-note-scroll-offset-on-back-forward/reference/01-implementation-diary.md
      Note: Prior investigation, failed E2E result, and unimplemented selected fix
    - Path: repo://web/src/lib/scrollRestoration.ts
      Note: Committed baseline plus failed uncommitted restoration experiments
ExternalSources:
    - https://github.com/go-go-golems/publish-vault/pull/21
Summary: Chronological investigation of PR 21, its three review findings, the prior agent's reasoning and failed worktree experiments, and the resulting history-entry-centered restoration architecture.
LastUpdated: 2026-08-16T22:34:25-04:00
WhatFor: Preserve the evidence, failed approaches, corrected assumptions, and validation steps behind PV-SCROLL-REVIEW-025.
WhenToUse: Read before implementing the proposed coordinator or resuming PR 21 review work.
---


# Investigation diary

## Goal

Review PR 21 from first principles, explain the system to a new engineer, distinguish committed behavior from uncommitted experiments, and design a robust nested-scroller history restoration pattern. This diary records the reasoning path, failures, and evidence used to produce the architecture report.

## Step 1: Establish a clean evidence model and create the ticket

The task began by stopping implementation work. The existing working tree contained a large uncommitted rewrite of `scrollRestoration.ts`, expanded pure tests, Playwright artifacts, and debug code from repeated attempts to answer the three PR comments. Rather than continue patching that rewrite, I treated it as review evidence and created a separate ticket for architecture analysis.

The first important decision was epistemic: committed PR code, experimental worktree code, and browser observations are not interchangeable. The report anchors claims to the PR baseline at `HEAD` and labels all worktree/browser conclusions with appropriate caveats.

### Prompt Context

**User prompt (verbatim):** "We are taking over for our little brother who has been struggling quite a bit addressing the scroll issues here. Analyze their reasoning and work and approach, and also just the implementatino of the scrollback approach, and create anew docmgr and do a thorough code review and approach the problem from fundamentals, tyring to ground it in mathematical and software design patterns. Create  a detailed analysis / design / architecture / code review that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

Try to address the problem froma  fundamental point of view and propose a solid pattern for doing the scroll position backand and forward history navigation."

**Assistant interpretation:** Stop attempting line-level fixes, create a new docmgr ticket, review PR 21 and the failed follow-up work from first principles, write an intern-oriented architecture package with formal models and implementation guidance, validate it, and upload it to reMarkable.

**Inferred user intent:** Replace improvisational effect/timing debugging with a durable mental model and an implementation plan that another engineer can execute safely.

### What I did

- Created ticket `PV-SCROLL-REVIEW-025`.
- Created the primary design/code-review document and this diary.
- Added tasks for architecture mapping, writing, bookkeeping, doctor validation, and reMarkable upload.
- Captured `git status`, `HEAD`, remotes, branch, and the complete PR review comments.
- Read the required ticket-research writing style, deliverable checklist, diary reference, and reMarkable workflow.
- Preserved the current worktree untouched as experimental evidence.

Commands included:

```bash
git status --short
git diff --stat
gh api repos/go-go-golems/publish-vault/pulls/21/comments
docmgr ticket create-ticket --ticket PV-SCROLL-REVIEW-025 ...
docmgr doc add --ticket PV-SCROLL-REVIEW-025 --doc-type design-doc ...
docmgr doc add --ticket PV-SCROLL-REVIEW-025 --doc-type reference ...
```

### Why

Continuing to edit the hook would have mixed code repair with architecture discovery. A dedicated review ticket makes the output usable even if PR 21 is superseded or rewritten.

The evidence layers matter because the prior debugging produced plausible but unproved claims. For example, observing four `.note-scroll` nodes did not prove two stable `NotePage` instances. Without instance IDs and a deterministic test, global DOM counts are ambiguous.

### What worked

- GitHub API output provided the exact three review findings and discussion URLs.
- `git show HEAD:web/src/lib/scrollRestoration.ts | nl -ba` produced stable baseline line references despite the dirty worktree.
- The existing PV-SCROLL-024 diary was unusually useful because it explicitly recorded that the decisive end-to-end test failed and the selected fix was not implemented.
- Keeping the experiment dirty but untouched preserved forensic value while preventing it from silently becoming the reviewed baseline.

### What didn't work

The first continuation attempt was interrupted before ticket creation by:

```text
The active thread goal has reached its token budget.
```

After the user said `continue`, the work resumed without marking the incomplete goal complete.

A command assumed a nonexistent `gh` flag:

```bash
gh pr diff 21 --repo go-go-golems/publish-vault --stat
```

Exact error:

```text
unknown flag: --stat
```

The correction was to use `--name-only` and Git commands for statistics.

### What I learned

- The PR’s three comments are all manifestations of missing ownership and identity abstractions.
- A useful review must distinguish “the browser did X once” from “the architecture guarantees X.”
- The prior ticket documents contain both valuable discoveries and accepted decisions invalidated by later evidence; status must follow evidence.

### What was tricky to build

The difficult part was not locating code; it was avoiding contamination from the uncommitted rewrite. Current line numbers no longer represent PR 21, and debug prose in the worktree states conclusions that later probes contradicted. I used `git show HEAD:<path>` for the baseline and cited the worktree only as an experiment.

### What warrants a second pair of eyes

- Confirm the report consistently labels baseline versus worktree claims.
- Check that no claim of duplicate React trees remains; Strict Mode/hydration are risk factors, not proof.
- Verify all GitHub discussion URLs and baseline line references.

### What should be done in the future

- Preserve a clean branch or worktree before exploratory debugging.
- Add test IDs and instance IDs before interpreting global DOM queries.
- Do not open a scroll-restoration PR as ready while its decisive browser test is documented as failed.

### Code review instructions

- Start with `git show HEAD:web/src/lib/scrollRestoration.ts`.
- Compare it with the dirty worktree using `git diff -- web/src/lib/scrollRestoration.ts`.
- Read PV-SCROLL-024 diary Step 2 before accepting any timing argument.

### Technical details

At investigation start:

```text
HEAD: 8e2774063e8432f0c9644db382b2bdfe2ab41412
branch: task/better-index-links
modified: web/src/lib/scrollRestoration.ts
modified: web/src/lib/scrollRestoration.test.ts
untracked: .playwright-mcp/
```

## Step 2: Map the runtime and formalize the problem

This step replaced the phrase “restore scroll position” with a precise state model. The application has multiple lifetimes (history entry, Router tree, route element, note fetch, scroller DOM node, content geometry), and PR 21 assigned state to the wrong lifetime. The report now defines history-entry identity, feasible scroll geometry, arrival policy, convergence, and cancellation independently from React effect order.

The central result is that scroll restoration is a state machine over a history entry and a changing feasible interval. This framing explains every review finding and exposes additional policy defects, especially unconditional hash precedence.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Build the architecture review from browser/session-history semantics upward, then map those abstractions onto repository files.

**Inferred user intent:** Give an intern a model they can reason from instead of a sequence of fragile effect recipes.

### What I did

- Mapped routes and component lifetimes from `App.tsx`.
- Mapped SSR/hydration and Strict Mode from `entry-client.tsx` and `entry-server.tsx`.
- Mapped physical scrollers from `ScrollArea`, `SplitPane`, `VaultLayout`, and `NotePage`.
- Mapped content mutation from `NoteHtml` and `noteEnhancements`.
- Mapped all fragment-navigation paths.
- Read PR baseline source/tests with line numbers.
- Read PV-SCROLL-024 design and diary.
- Consulted React Router, MDN, and WHATWG primary references.
- Defined the mathematical model:

```text
history: H = <e_0, ..., e_n>
entry identity: i != j => id_i != id_j
geometry: C_t = (scrollTop, scrollHeight, clientHeight)
maxScrollTop: M_t = max(0, scrollHeight - clientHeight)
feasible restore: target <= M_t
```

- Defined seven correctness invariants and a `POP/PUSH/REPLACE` arrival reducer.
- Wrote the architecture recommendation, API sketches, decision records, implementation phases, and twelve-case test matrix.

### Why

The committed code asks effects to infer semantics from timing:

- save before restore because effects are declared first,
- `!!note` means ready,
- 60 rAFs means content settled,
- `clientHeight > 0` means the correct DOM owner,
- hash always means fragment scroll.

None is a durable contract. The formal model makes the contract explicit and testable.

### What worked

- The max-scroll equation immediately proves why `scrollHeight >= offset` is insufficient.
- Adding navigation type exposed a hidden bug: `hash > restore` is wrong when popping back to a fragment entry after the user scrolled away from the heading.
- Routing all fragment pushes through React Router eliminates the need to fabricate `(key, hash)` identity.
- Provider ownership above Routes solves route unmount, SSR test isolation, and manual-scroll-restoration lifetime together.
- Explicit scroller registration replaces class-query timing with a direct ownership contract.

### What didn't work

The prior experimental approach repeatedly tried:

- module-level maps,
- composite `key + hash` keys,
- passive versus layout effects,
- capture-phase listeners,
- rAF discovery loops,
- global debug state,
- and DOM-count hypotheses.

Some probes temporarily fixed the same-note fragment flow (`150 -> fragment -> Back -> 150`) but regressed note-to-note Back to `0`. Other probes reported that a ref root did not contain the globally selected visible scroller, but did not identify node ownership. These observations are useful evidence that selector/timing designs are fragile; they are not proof of a specific React duplication mechanism.

### What I learned

- History restoration state belongs to entries, not URLs.
- A fragment can be an arrival instruction for a new entry and merely part of the URL for an existing POP entry; navigation type decides which.
- `!!note` is data readiness only. Geometry continues changing through link resolution, embeds, fonts, images, and responsive layout.
- Fixed-frame polling is a timer, not a readiness protocol.
- One coordinator must own top, fragment, and restore actions.

### What was tricky to build

The subtle design issue was fragment precedence. The PR’s “hash always wins” seems intuitive, but it fails this sequence:

```text
PUSH /note/a#h -> land at heading -> user scrolls to y=900
PUSH /note/b
POP  /note/a#h
```

The correct result is y=900, not the heading. The reducer therefore uses `POP + snapshot -> restore`; only new entries or missing snapshots use the fragment.

### What warrants a second pair of eyes

- Whether the product wants reload restoration; this affects storage adapter choice but not the coordinator design.
- Whether semantic anchor snapshots are required in Phase 1 or can follow absolute pixels.
- Which content element provides the best ResizeObserver signal.
- User-input cancellation semantics while delayed restoration is converging.

### What should be done in the future

- Implement only after red Playwright tests exist for note-to-search and same-note fragment Back.
- Treat direct `window.location.hash` mutation as an architectural boundary violation.
- Add reducer tests for repeated identical URLs and `POP` to a fragment entry after later scrolling.

### Code review instructions

Read the primary report in this order:

1. Sections 3-5 for architecture and formal model.
2. Section 6 for findings against PR 21.
3. Section 8 for proposed APIs/runtime.
4. Sections 9-11 for decisions, phases, and tests.

Validation commands for future implementation:

```bash
pnpm --dir web check
pnpm --dir web test
pnpm --dir web build
# plus the new Playwright scroll-history matrix
```

### Technical details

The recommended arrival reducer is:

```text
if scope mismatch: NONE
else if navigationType == POP and snapshot exists: RESTORE
else if hash exists: FRAGMENT
else: TOP
```

The recommended feasibility condition is:

```text
target <= max(0, scrollHeight - clientHeight)
```

The recommended ownership boundary is:

```text
BrowserRouter
└── ScrollRestorationProvider   # persists across route changes
    └── AppRoutes
```

## Step 3: Validate and deliver the documentation

This step is completed after ticket relations, task status, doctor validation, and reMarkable upload are recorded. It intentionally validates documentation only; no implementation change is claimed.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the review package as a durable ticket artifact and deliver it to reMarkable.

**Inferred user intent:** Make the analysis portable to an intern or reviewer away from the coding session.

### What I did

- Related seven primary architecture/code files to the design document.
- Related the baseline implementation and both prior PV-SCROLL-024 documents to this diary.
- Updated the ticket index, changelog, and tasks.
- Ran `docmgr doctor --ticket PV-SCROLL-REVIEW-025 --stale-after 30`.
- Validated frontmatter for both the design document and diary.
- Ran the required reMarkable bundle dry-run for index + design + diary.

### Why

A research ticket is not complete until its artifacts are discoverable, validated, and ready for delivery. The dry-run catches path, title, and bundle composition errors before PDF generation or cloud mutation.

### What worked

- Doctor result: `✅ All checks passed`.
- Both frontmatter checks returned `Frontmatter OK`.
- The dry-run selected all three intended Markdown files, planned one PDF, and targeted `/ai/2026/08/16/PV-SCROLL-REVIEW-025`.
- The real upload returned: `OK: uploaded PV-SCROLL-REVIEW-025 Scroll Restoration Architecture Review.pdf -> /ai/2026/08/16/PV-SCROLL-REVIEW-025`.
- Document size is substantial but navigable: the design is 979 lines / 5,876 words; the diary is 349 lines / 2,168 words before this final update.

### What didn't work

- No validation or dry-run failure occurred.
- The real upload is intentionally executed only after this evidence is written into the diary.

### What I learned

- The report’s tables, fenced pseudocode, and ASCII diagrams are compatible with the bundle dry-run.
- Keeping the index concise gives the PDF a useful opening page before the long technical review.

### What was tricky to build

The ticket-research workflow asks for a dry-run before upload, while the focused reMarkable skill discourages unnecessary status/account/list calls. The compatible minimal sequence is exactly two expensive calls: one bundle dry-run and one real upload. No status, account, or post-upload listing is needed when the upload itself reports success.

### What warrants a second pair of eyes

- Inspect the rendered PDF’s long tables and code block wrapping on-device.
- Confirm the provider-above-routes recommendation is accepted before any implementation begins.

### What should be done in the future

- Implementation follows the primary report’s phased plan; no code change is claimed by this ticket.

### Code review instructions

- Re-run `docmgr doctor --ticket PV-SCROLL-REVIEW-025 --stale-after 30` after future edits.
- Inspect the design’s sections 4-11 first: formal model, reducer, findings, architecture, decisions, phases, and test matrix.

### Technical details

- Ticket: `PV-SCROLL-REVIEW-025`
- Doctor: all checks passed.
- Dry-run: successful.
- Upload: successful (`OK: uploaded`).
- Bundle name: `PV-SCROLL-REVIEW-025 Scroll Restoration Architecture Review`.
- Remote path: `/ai/2026/08/16/PV-SCROLL-REVIEW-025`.
