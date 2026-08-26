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

**Commit (code):** `e2d37ac` — `docs(PV-SEARCH-027): define search design scope`

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
P0 completion slip printed after commit: yes
Phase 0 task: complete
implementation changes: none
```

## Step 2: Map dynamic and static search from source metadata to result cards

Phase 1 traced the complete search path in both supported runtime modes. The main architectural finding is that the existing `modTime` contract is already inconsistent: backend mode exposes filesystem modification time, while static mode treats `created` frontmatter as `modTime` and substitutes today's date when it is absent.

The map also identifies a compatibility constraint that shapes the eventual Bleve design. Existing `#tag` search is fuzzy or prefix discovery over an analyzed, flattened tags field. Proper exact advanced filtering should add a distinct filter representation instead of silently changing that behavior.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Build a file- and symbol-backed architecture map before choosing date precedence, query parameters, Bleve fields, or UI controls.

**Inferred user intent:** Give an intern the system model necessary to modify all implementations consistently and preserve reload, static-build, and current search behavior.

**Commit (code):** pending Phase 1 documentation commit

### What I did

- Printed and preserved `scripts/slips/03-phase-1-start.yaml` before inspecting architecture.
- Enumerated Go note, parser, search, API, runtime, watcher, and test symbols.
- Enumerated React types, RTK Query, SearchPage, SearchBar, NoteCard, Redux state, router use, static-vault implementation, stories, and tests.
- Read the prior RETRO-TAG-010 guide to distinguish historical intent from current code.
- Compared dynamic and static date behavior.
- Mapped current Bleve fields, analyzers, stored-field behavior, general search, tag search, limits, and missing sort/pagination/filter contracts.
- Wrote `analysis/02-current-search-and-metadata-architecture-map.md` with four diagrams and an architecture gap matrix.

### Why

The feature crosses two search engines and three state representations: URL, Redux, and RTK Query cache arguments. A backend-only design would leave static builds incompatible. A component-only design would duplicate filter parsing and provide no stable API to other clients.

### What worked

- Current main has explicit `Note.ModTime`, so date transport can be designed without adding filesystem collection.
- `NoteCard` already supports optional date rendering, reducing visual component work.
- Persistent indexes are rebuilt per snapshot, so mapping evolution is a derived-state rebuild rather than an in-place data migration.
- Existing tests already protect tag behavior, batch equivalence, deletion, rollback, reload serialization, and delayed cleanup.

### What didn't work

The first broad repository search included a generated embedded JavaScript artifact under a package directory and returned a massive minified line. The command was syntactically valid but produced poor evidence:

```text
rg -n 'type Note struct|ModTime|Frontmatter|Metadata|SearchDocument|ForEachSearchDocument|func \\(.*SearchDocument|Date' pkg internal -S
```

I narrowed subsequent reads to named source files and used targeted symbol searches. No generated output was used as architectural evidence.

No committed PV-SEARCH-026 document was present on current main, despite prior session context referring to one. Rather than importing an unavailable conclusion, this phase re-established the architecture from current code and the committed RETRO-TAG-010 history.

### What I learned

- Dynamic `Note.ModTime` is `os.FileInfo.ModTime()` from checkout state.
- Static `Note.modTime` is frontmatter `created` or the current date; this is unstable and semantically different.
- The current API returns a bare array with a hard-coded limit of 30 and generic 500 errors.
- A text query shorter than two characters suppresses frontend search, which blocks filter-only requests.
- The current tag field is suited to discovery but not exact multi-value filtering.
- Static search does not search the full body and already has ranking differences from Bleve; inclusion parity is a more realistic contract than score parity.

### What was tricky to build

The architecture map had to distinguish fields that happen to share a TypeScript property name from fields that have the same semantics. `modTime` fails that test. The map therefore follows provenance, fallback, normalization, and transport rather than matching names only.

The proposed future state must also preserve PV-MEM-002's bounded indexing. Adding path, exact tags, and dates increases staged document bytes and persistent index size, so implementation acceptance must include updated memory and index-size evidence.

### What warrants a second pair of eyes

- Confirm static mode remains a supported first-class target rather than being deprecated by this feature.
- Review the conclusion that exact tag filtering needs a new field while `tags` remains analyzed.
- Review whether URL state should fully replace committed Redux search state or whether Redux remains only for draft controls.
- Confirm the result contract should expose date provenance rather than an unlabelled `modTime` string.

### What should be done in the future

Phase 2 must define canonical created, updated, filesystem, and display-date semantics; normalize date-only versus timestamp values; specify precedence and invalid/missing behavior; and verify Bleve date/keyword mapping APIs against v2.6.0.

### Code review instructions

- Read `analysis/02-current-search-and-metadata-architecture-map.md` from sections 1 through 13.
- Verify each named symbol in `pkg/vault/vault.go`, `pkg/search/search.go`, `pkg/api/api.go`, `web/src/store/vaultApi.ts`, `SearchPage.tsx`, and `staticVault.ts`.
- Compare dynamic `loadNote` with static `buildVault` date derivation.
- Confirm the gap matrix does not present a Phase 2 decision as current behavior.

### Technical details

```text
backend search implementation: pkg/search/search.go
static search implementation: web/src/vault/staticVault.ts
backend date source: os.FileInfo.ModTime
static date source: frontmatter.created or current date
Bleve version: v2.6.0
current API: GET /api/search?q=..., hard limit 30
current URL state: q only
current result date field: none
current card date prop: optional modTime
architecture diagrams: 3
implementation changes: none
```
