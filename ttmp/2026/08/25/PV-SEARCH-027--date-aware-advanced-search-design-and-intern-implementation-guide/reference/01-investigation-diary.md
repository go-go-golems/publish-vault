---
Title: Investigation diary
Ticket: PV-SEARCH-027
Status: active
Topics:
    - search
    - frontend
    - backend
    - architecture
    - performance
    - regression
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Strict chronological diary for the evidence-first design of note dates and advanced metadata search.
LastUpdated: 2026-08-25T20:35:00-04:00
WhatFor: Preserving exact prompts, investigation commands, failures, decisions, commits, review risks, and continuation instructions.
WhenToUse: Read before resuming or reviewing PV-SEARCH-027.
---

# Investigation Diary

## Goal

Capture how PV-SEARCH-027 moves from user-visible goals to an evidence-backed, intern-ready architecture and implementation guide for date-aware advanced search.

## Step 1: Create the ticket, freeze scope, and print the research plan

The first step converted a broad feature request into six gated research phases. It deliberately did not decide whether filesystem modification time or frontmatter is authoritative, nor did it choose a query grammar before examining the current note model, Bleve mapping, HTTP contract, and frontend route state.

The ticket now has an explicit evidence map and acceptance gates. A physical master-plan slip and Phase 0 start slip were printed before architecture work began, and their YAML sources are preserved under the ticket.

### Prompt Context

**User prompt (verbatim):** "Ok, let's ccreate a new ticket to :

- add date information on the search results
- add a proper advanced search with filters and all.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Additional user instruction (verbatim):** "commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)

Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done.

---

go ahead"

**Assistant interpretation:** Create a new ticket entirely in the existing workspace, research the current system before deciding semantics, write an exhaustive intern-oriented design and implementation guide, keep strict diary/docmgr bookkeeping, commit coherent phase increments, print master/start/done slips in order, and upload a validated bundle to reMarkable.

**Inferred user intent:** Make the next search feature safe to delegate. The intern should receive a precise model of the existing architecture, explicit contracts and decisions, and a reviewable implementation sequence rather than an underspecified feature request.

**Commit (code):** pending Phase 0 documentation commit

### What I did

- Returned the existing workspace to a clean branch based on `origin/main` at merge `416a0db`.
- Created ticket `PV-SEARCH-027` with search, frontend, backend, architecture, performance, and regression topics.
- Created the primary design guide, strict diary, and scope/acceptance analysis documents.
- Added six phase tasks.
- Defined master phases:
  - P0 scope and evidence map;
  - P1 current backend/frontend architecture;
  - P2 canonical date model;
  - P3 advanced query and filter UX;
  - P4 implementation/testing/rollout plan;
  - P5 audit and reMarkable delivery.
- Printed and preserved `scripts/slips/00-master-plan.yaml`.
- Printed and preserved `scripts/slips/01-phase-0-start.yaml` before writing scope conclusions.
- Wrote explicit user behaviors, invariants, unresolved questions, evidence sources, deliverables, and completion gates.

### Why

Date display and advanced filtering cross multiple ownership boundaries. Date values originate in source or filesystem metadata, are normalized by the vault, represented in the search document, mapped into Bleve, serialized through HTTP, persisted in the URL, and rendered by React. Selecting one layer in isolation would create ambiguous precedence or duplicated parsing.

The scope document prevents the design from silently assuming that `ModTime` is a creation date, that all frontmatter uses one key, or that URL parameters and Bleve fields already support structured conjunctions.

### What worked

- `docmgr` created a complete ticket workspace with index, tasks, changelog, design, analysis, and diary documents.
- Both requested thermal slips printed successfully through the remote Almanach service.
- The acceptance gates now make privacy, compatibility, index rebuild, accessibility, performance, and rollout part of the feature rather than later additions.

### What didn't work

Before the final go-ahead, I briefly created a new worktree under a new dated workspace while looking for the previously summarized search ticket. The user corrected the workspace boundary:

```text
wait wjat are you doing? don't work outside the workspace. use it still
```

No project changes existed in that temporary worktree. I verified it was clean, removed it, deleted its temporary branch, and created `task/pv-search-027-advanced-search` inside the existing workspace at:

```text
/home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault
```

No source or user changes were discarded.

The first Phase 0 commit attempt was also stopped by the required staged diff check:

```text
ttmp/2026/08/25/PV-SEARCH-027--date-aware-advanced-search-design-and-intern-implementation-guide/changelog.md:15: new blank line at EOF.
```

`docmgr changelog update` had left two terminal newlines. I removed only the extra terminal blank line, staged the updated diary and changelog, and reran `git diff --cached --check` before committing.

### What I learned

- The existing workspace is part of the task contract, not merely a convenient checkout.
- Prior ticket summaries may refer to artifacts that are not committed on current main; any useful conclusion must be re-established from files available in this workspace or explicitly cited external evidence.
- Date authority is the first high-risk semantic decision because Git checkout timestamps can differ from note-authored dates.

### What was tricky to build

The scope must be detailed enough to prevent hidden assumptions without pre-deciding architecture. The solution was to separate observable outcomes and hard invariants from questions that Phase 1 and Phase 2 must answer with file-backed evidence.

The slip order is also an invariant: master plan first, phase-start before phase work, phase-completion only after the phase gate. YAML sources use monotonically numbered names so the physical workflow remains auditable.

### What warrants a second pair of eyes

- Confirm that the requested filter scope includes date, tags, and paths, with other metadata added only when the current model supports it.
- Review whether facets/counts belong in the first implementation or should remain a compatible extension.
- Ensure the final date decision does not call Git checkout `ModTime` a note creation date.
- Confirm the advanced UI remains usable without JavaScript-only hidden query semantics.

### What should be done in the future

Phase 1 must enumerate concrete backend and frontend symbols, existing contracts, tests, and gaps. No final API or mapping recommendation should be accepted before that map exists.

### Code review instructions

- Start with `analysis/01-scope-evidence-map-and-acceptance-gates.md`.
- Compare the six ticket tasks with the master-plan slip.
- Verify `00-master-plan.yaml` and `01-phase-0-start.yaml` exist and were generated by the work-slip tool.
- Run `docmgr doctor --ticket PV-SEARCH-027 --stale-after 30` and `git diff --check`.

### Technical details

```text
ticket: PV-SEARCH-027
branch: task/pv-search-027-advanced-search
base: 416a0db
workspace: /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault
phase count: 6
master slip printed: yes
P0 start slip printed: yes
implementation changes: none
```
