---
Title: Fundamentals-first Markdown and wiki-link parser architecture review
Ticket: PV-MARKDOWN-023
Status: complete
Topics:
    - parser
    - wiki-link
    - frontmatter
    - html-rendering
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket landing page for the algorithmic architecture and code review of publish-vault Markdown, wiki-link, graph, and static-renderer behavior.
LastUpdated: 2026-08-16T17:59:09.323078795-04:00
WhatFor: Orient implementers to the review deliverables, findings, and proposed migration.
WhenToUse: Start here before reading or implementing PV-MARKDOWN-023 recommendations.
---


# Fundamentals-first Markdown and wiki-link parser architecture review

## Overview

This ticket examines the complete path from Markdown bytes to rendered HTML, metadata, outgoing links, backlinks, excerpts, and static-build output. It treats recent parser incidents as evidence of deeper architectural seams: duplicated syntax recognition, loss of typed source context, single-value handling of ambiguous references, and HTML strings used as an internal resolution protocol.

The primary recommendation is to keep goldmark and use custom inline/block AST extensions, preserve typed link occurrences with source spans, resolve references through a deterministic ambiguity-aware index, and render HTML/plain text/graphs from the same parsed document. Immediate correctness fixes are separated from the staged migration so high-severity defects do not wait for a refactor.

## Deliverables

1. **Primary architecture and design review** — `design-doc/01-markdown-and-wiki-link-parsing-algorithms-architecture-and-robust-building-blocks.md`
   - current architecture and invariants;
   - twelve code-review findings;
   - target components and API sketches;
   - six decision records;
   - phased migration, tests, risks, alternatives, and open questions.
2. **Parser API, algorithm, and test inventory** — `reference/02-parser-api-algorithm-and-test-inventory.md`
   - file/API map;
   - representation and pipeline tables;
   - backend/static differences;
   - severity/evidence index;
   - reproducible probe results.
3. **Investigation diary** — `reference/01-investigation-diary.md`
   - chronological commands, corrected assumptions, tricky decisions, review guidance, and follow-ups.
4. **Edge probe** — `scripts/01-parser-edge-probe/main.go`
   - frontmatter mutation;
   - occurrence deduplication loss;
   - cross-line link recognition;
   - alias HTML;
   - broad HTML rewrite scope.

## Highest-priority findings

- Parse-time frontmatter masking recognizes fewer valid delimiters than goldmark-meta and can inject anchor HTML into valid YAML values.
- Ambiguous short links resolve by first insertion into an index populated from Go map iteration, so the selected target is not stable.
- Vault HTML resolution rewrites unrelated authored `data-target` attributes and ordinary `/note/...` anchors.
- Link occurrence deduplication loses heading and link/embed distinctions before consumers can choose their own equivalence relation.
- The static renderer uses parser-context-aware wiki nodes for HTML and a raw regex for graph extraction, so rendering and backlinks can disagree.

## Review order for a new intern

1. Read this index and the executive summary of the primary design.
2. Use the inventory's repository orientation and stage tables.
3. Run the edge probe and the parser/vault tests.
4. Read the detailed findings and decision records.
5. Follow the "Intern review path" in the primary design before implementing a phase.

## Status

The research/design deliverable is complete and validated. Implementation is deliberately deferred to focused follow-up tickets, beginning with Phase 0 correctness guards.

## Tasks and changelog

- See [tasks.md](./tasks.md) for completion state.
- See [changelog.md](./changelog.md) for investigation and delivery entries.
