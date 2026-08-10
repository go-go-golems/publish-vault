---
Title: 'Note not found for nested transcripts slug: diagnose slug resolution and vault exclusion'
Ticket: PV-SLUG-020
Status: active
Topics:
    - slug
    - routing
    - parser
    - vault
    - api
    - frontmatter
    - ignore
    - retro-obsidian-publish
    - obsidian-vault
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >-
    The reported note IS correctly slugged, parsed and indexed - slug generation and vault exclusion are not at fault. The user-visible "Note not found" comes from web/server.mjs:242-245, which collapses every failure to fetch /api/notes/<slug> (genuine 404, unreachable backend, non-2xx, unparseable body) into one hard 404 with that literal body. A trailing slash reproduces the exact symptom on production today, because Vault.GetNote is an exact-match map lookup with no normalization.
LastUpdated: 2026-08-09T20:35:30.921160025-04:00
WhatFor: Diagnosing why a note that exists in the vault reports as missing over HTTP, and designing the fix.
WhenToUse: When a /note/<slug> URL 404s, when changing slugify or GetNote, or when a note is silently absent from the published vault.
---

# Note not found for nested transcripts slug: diagnose slug resolution and vault exclusion

## Overview

A user reported that `https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields`
fails with **"Note not found"**, even though the note plainly exists in the vault at
`Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md`.
The obvious suspect was the slugifier: the path is five levels deep, the folder
name has spaces and capitals, and the filename uses underscores. This ticket
proves that suspect innocent and identifies what actually breaks.

Running the real `vault.New` + `LoadAll` + `GetNote` against the real vault
(`scripts/01-slug-probe`) shows the note loads, renders 240 KB of HTML, and is
stored under exactly the slug the URL spells. None of the four exclusion paths
fired: `.vault-ignore` has a single rule (`ttmp/_*/`), `.publish/config.yaml`
does not exist, the note has no `publish` key, and it parsed cleanly. Production
returns HTTP 200 for that URL today, and reports the same 1712 notes as a local
load - so the failure was state-dependent, not a property of the slug.

The root cause is in the Node SSR sidecar. `fetchAPI` (`web/server.mjs:83-91`)
returns `null` for four distinct conditions - a genuine API 404, any non-2xx, a
thrown `fetch` (connection refused, timeout), and a body that fails to parse -
and `web/server.mjs:242-245` renders all of them as HTTP 404 with the literal
body `Note not found`. A backend that is merely unavailable is therefore
reported to users and crawlers as a note that does not exist.
`scripts/03-ssr-conflation-repro.mjs` shows those cases producing byte-identical
output. Separately, because `Vault.GetNote` (`pkg/vault/vault.go:725`) is a bare
exact-match map lookup and `slugify` trims `-` but not `/`, appending a single
trailing slash to the URL reproduces the exact reported symptom on production on
demand.

Deliverables: a design document written for someone new to the codebase (URL-to-note
trace diagram, the measured slug algebra, all four exclusion paths, root cause with
proof, pseudocode, cited API reference, six fix options, a five-phase plan, and
concrete regression-test rows), this diary, and five reproduction scripts under
`scripts/`. No application source was modified - this is an analysis ticket.

### Root cause in one sentence

`web/server.mjs` cannot tell "this note does not exist" apart from "I could not
reach the backend", and reports both as `Note not found`.

### Recommended fix

Do **F1** (make `fetchAPI` return a tagged result so only a genuine 404 yields a
404; unreachable yields 503) and **F4** (log every silently-dropped note with its
reason) first, then **F2** (normalized fallback lookup with a 308 redirect, which
permanently removes the trailing-slash and case-variant class). Do **not** make
slugs lossless - it would break every existing URL.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- slug
- routing
- parser
- vault
- api
- frontmatter
- ignore
- retro-obsidian-publish
- obsidian-vault

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
