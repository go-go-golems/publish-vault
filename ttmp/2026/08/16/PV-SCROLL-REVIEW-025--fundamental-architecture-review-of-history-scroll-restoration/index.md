---
Title: Fundamental architecture review of history scroll restoration
Ticket: PV-SCROLL-REVIEW-025
Status: complete
Topics:
    - frontend
    - react
    - routing
    - ssr
    - ux
    - wiki-link
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Fundamental review of PR 21 that replaces effect-order patching with a persistent history-entry-centered nested-scroll coordinator design.
LastUpdated: 2026-08-16T18:43:27.182119491-04:00
WhatFor: Track the architecture review, design recommendation, evidence, validation, and reMarkable delivery for PR 21 scroll restoration.
WhenToUse: Start here before reading the detailed review or implementing scroll restoration.
---


# Fundamental architecture review of history scroll restoration

## Overview

This ticket reviews PR 21 and the failed follow-up experiments from first principles. Its recommendation is to replace route-local, effect-timed restoration with a persistent coordinator mounted above the route switch. The coordinator owns one snapshot per browser history entry, distinguishes POP from PUSH/REPLACE, registers the actual nested scroller explicitly, centralizes fragment/top/restore behavior, and converges when dynamic content geometry becomes ready.

The ticket contains analysis and design only. The dirty worktree implementation is explicitly not accepted or committed.

## Key Links

- [Primary architecture and PR code review](./design-doc/01-scroll-restoration-architecture-and-pr-21-code-review.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [PR 21](https://github.com/go-go-golems/publish-vault/pull/21)
- **Tasks**: [tasks.md](./tasks.md)
- **Changelog**: [changelog.md](./changelog.md)

## Status

Current status: **active** — analysis complete; validation and reMarkable delivery pending.

## Topics

- frontend
- react
- routing
- ssr
- ux
- wiki-link

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
