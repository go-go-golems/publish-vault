---
Title: Scope evidence map and acceptance gates
Ticket: PV-SEARCH-027
Status: complete
Topics:
    - search
    - frontend
    - backend
    - architecture
    - performance
    - regression
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/api/api.go
      Note: Current HTTP search contract evidence target
    - Path: repo://pkg/search/search.go
      Note: Current Bleve mapping and query semantics evidence target
    - Path: repo://pkg/vault/vault.go
      Note: Primary note metadata and search-document evidence target
    - Path: repo://web/src
      Note: Current search controls result rendering and URL-state evidence boundary
ExternalSources: []
Summary: Defines the requested date-aware search outcomes, investigation boundaries, open decisions, source map, and gates for accepting the intern implementation guide.
LastUpdated: 2026-08-25T20:35:00-04:00
WhatFor: Preventing premature API and date-model decisions before current backend, frontend, schema, and URL behavior are mapped.
WhenToUse: Read before interpreting the final design or changing the ticket scope.
---


# Scope, evidence map, and acceptance gates

## Requested outcome

PV-SEARCH-027 must produce an implementation-ready design for two connected user-visible capabilities:

1. Search results display meaningful note date information.
2. Search supports a proper advanced mode with composable filters rather than requiring users to encode every constraint in free text.

The output is a design and implementation handoff for a new intern. It must explain enough of the existing system that the reader can implement the feature without first reconstructing the architecture from unrelated files.

## Observable user behavior in scope

The guide must define:

- which date or dates a result may display;
- where each date originates and how conflicting sources are resolved;
- how dates are serialized, indexed, filtered, sorted, and rendered;
- exact inclusive/exclusive semantics for date ranges;
- filters for tags, paths/folders, date ranges, and other metadata justified by the current note model;
- how free-text search combines with structured filters;
- URL representation, browser history, shareability, and reload behavior;
- desktop and narrow-screen advanced-search interaction;
- loading, empty, invalid-filter, unavailable-field, and backend-error states;
- accessibility and keyboard behavior;
- compatibility with existing `#tag` and `tag:` searches;
- stable API errors and validation boundaries;
- tests, migrations/rebuild behavior, rollout, telemetry, and rollback.

## Constraints that must remain true

- Search results continue to come from the active immutable runtime snapshot.
- Persistent Bleve indexes remain derived per-snapshot state.
- A reload publishes matching vault and search revisions atomically.
- Index rebuild failures leave the previous snapshot available.
- Search traces and metrics do not add note-, query-, path-, or tag-derived high-cardinality labels.
- Existing simple text and tag searches retain documented behavior unless an explicit migration is accepted.
- Raw private vault values are not committed as examples or fixtures.
- Advanced filters must be representable in a typed contract; the UI must not be the sole parser or source of truth.

## Questions that require evidence before decision

### Date authority

- Does the note model currently retain filesystem modification time, frontmatter dates, both, or neither?
- Which frontmatter keys occur in existing parser tests and public fixtures (`date`, `created`, `updated`, aliases)?
- Does Git-backed deployment make filesystem modification time stable, useful, or misleading after checkout?
- Should the contract expose one resolved display date, separate created/updated dates, or explicit source metadata?
- How should date-only frontmatter values and timestamp values normalize across time zones?
- What happens when a note has invalid, ambiguous, or absent date metadata?

### Query and filter contract

- Which constraints belong in URL query parameters versus a compact expression grammar?
- Does Bleve mapping currently support exact keyword fields for tags and paths, sortable date fields, and conjunctions?
- Should the backend return facets/counts in the first version?
- How are multiple tags combined: any, all, or an explicit operator?
- Are folder filters exact, prefix-based, or both?
- Is sorting limited to relevance/newest/oldest, and what deterministic tie-breaker is required?
- What result limit, offset/cursor, and maximum filter counts keep resource use bounded?

### Frontend state

- Where does the current search state live: route parameters, component state, Redux, or generated API state?
- Which component renders result cards and can display optional metadata without layout regressions?
- How does the current router normalize and replace query strings?
- Which existing design-system components can support dialogs, drawers, comboboxes, chips, and date fields?

## Evidence sources

The investigation will anchor decisions to:

- `pkg/vault/` note parsing, metadata, search-document construction, and fixtures;
- `pkg/search/` mapping, query construction, persistence, and equivalence tests;
- `pkg/api/` HTTP request/response contracts;
- `pkg/server/` snapshot and route wiring;
- `web/src/` result rendering, route state, search controls, generated/manual API types, and tests;
- current help/README documentation;
- prior tag-search and URL-state ticket artifacts if present in the repository;
- Bleve v2 mapping and query APIs actually used by the pinned dependency;
- public generated fixtures for date and filter examples.

## Deliverables

1. A current-state architecture map with concrete symbols and data flow.
2. A canonical date model and precedence decision record.
3. A typed advanced-search request/response contract.
4. Bleve mapping and query-composition sketches grounded in current APIs.
5. A URL and frontend state model with responsive and accessible interaction design.
6. A file-by-file implementation sequence suitable for an intern.
7. Unit, integration, equivalence, browser, accessibility, performance, migration, and rollback tests.
8. Security, privacy, cardinality, and operational guidance.
9. A strict chronological diary and synchronized docmgr bookkeeping.
10. A validated reMarkable bundle containing the ticket overview, design guide, and diary.

## Acceptance gates

The ticket is complete only when:

- every major current-state statement has a file/symbol reference;
- date precedence and missing/invalid behavior are explicit;
- request, response, URL, and index-field contracts are concrete enough to implement;
- filter combination and sorting semantics are unambiguous;
- existing simple/tag search compatibility is addressed;
- performance and index-size implications are considered against persistent bounded batching;
- the implementation sequence names files, symbols, tests, and validation commands;
- diagrams cover runtime data flow, query composition, UI state, and reload/index rebuild behavior;
- the diary records commands, failures, decisions, review risks, and continuation guidance;
- all phase slips exist and each completion slip was printed after its gate;
- `docmgr doctor` and Markdown/diff checks pass;
- the bundle dry-run and real reMarkable upload both succeed and are verified;
- coherent ticket commits are pushed without unrelated workspace changes.
