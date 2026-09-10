---
Title: Remove WebKit font pixelation from the published vault UI
Ticket: PV-WEBKIT-FONT-001
Status: complete
Topics:
    - frontend
    - styling
    - ux
    - regression
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: 'Changed the global WebKit font-smoothing policy from none to antialiased, preserving the rest of the retro styling; frontend checks and bundle verification passed.'
LastUpdated: 2026-09-10T16:18:34.760554297-04:00
WhatFor: Tracking the focused CSS fix for readable, non-pixelated WebKit text.
WhenToUse: Read this ticket before changing global font smoothing or retro image-rendering policy.
---


# Remove WebKit font pixelation from the published vault UI

## Overview

The global `html` rule previously set `-webkit-font-smoothing: none`, which made text appear pixelated in WebKit browsers. The fix changes only that value to `antialiased`; font stacks, sizing, retro colors, and the separate `image-rendering: pixelated` policy remain unchanged.

Automated validation passed: TypeScript check, 93 frontend tests, production Vite build, generated CSS inspection, and `git diff --check`. Manual Safari/WebKit visual QA is an optional follow-up because no WebKit runtime is installed in the local Playwright cache.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- frontend
- styling
- ux
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
