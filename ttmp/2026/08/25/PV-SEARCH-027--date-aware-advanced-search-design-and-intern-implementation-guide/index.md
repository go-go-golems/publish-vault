---
Title: Date-aware advanced search design and intern implementation guide
Ticket: PV-SEARCH-027
Status: complete
Topics:
    - search
    - frontend
    - backend
    - architecture
    - performance
    - regression
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Evidence-first design ticket for adding canonical note dates to search results and a typed advanced-search system with metadata filters, URL persistence, accessible UI, and preserved snapshot semantics.
LastUpdated: 2026-08-25T21:02:20.99464385-04:00
WhatFor: Coordinating architecture research, date semantics, API and index contracts, frontend UX, implementation sequencing, testing, and intern handoff.
WhenToUse: Start here before changing publish-vault note metadata, Bleve mappings, search APIs, URL state, result cards, or advanced-search controls.
---


# Date-aware advanced search design and intern implementation guide

## Overview

This ticket designs two connected search improvements: meaningful note dates in result cards and structured advanced filters over date, tags, paths, and justified metadata. The work begins by mapping the complete note-to-index-to-API-to-React path, then makes explicit decisions about date authority, query semantics, URL state, compatibility, testing, and rollout.

The primary audience is a new intern. The final guide must teach the current architecture before prescribing changes and must provide concrete contracts, pseudocode, diagrams, file references, tests, and phased implementation instructions.

## Key Links

- **Primary guide**: [Date-aware advanced search architecture and implementation guide](./design-doc/01-date-aware-advanced-search-architecture-and-implementation-guide.md)
- **Scope and gates**: [Scope, evidence map, and acceptance gates](./analysis/01-scope-evidence-map-and-acceptance-gates.md)
- **Diary**: [Investigation diary](./reference/01-investigation-diary.md)
- **Tasks**: [tasks.md](./tasks.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete — architecture, date, API, index, URL, UX, implementation plan, validation, reMarkable delivery, slips, commits, and push gates passed**

## Topics

- search
- frontend
- backend
- architecture
- performance
- regression

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
