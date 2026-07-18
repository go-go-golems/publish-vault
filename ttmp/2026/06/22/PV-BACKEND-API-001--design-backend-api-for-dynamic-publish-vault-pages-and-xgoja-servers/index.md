---
Title: Design backend API for dynamic publish-vault pages and xgoja servers
Ticket: PV-BACKEND-API-001
Status: active
Topics:
    - backend
    - api
    - ssr
    - xgoja
    - obsidian-vault
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket workspace for designing a backend API that lets publish-vault serve dynamic pages and future xgoja/JavaScript-backed servers while preserving the current Markdown vault behavior, SSR, and agent-readable mirrors."
LastUpdated: 2026-06-22T21:45:00-04:00
WhatFor: "Use as the entrypoint for PV-BACKEND-API-001 design and implementation planning."
WhenToUse: "When implementing dynamic publish-vault pages, API v1, SSR prefetch changes, or xgoja integration."
---

# Design backend API for dynamic publish-vault pages and xgoja servers

## Overview

This ticket analyzes how `publish-vault` currently works and proposes a backend API for dynamic pages. The main finding is that publish-vault does more than serve Markdown from disk: it parses Markdown, renders HTML, resolves wiki links, builds backlinks and search data, serves safe vault assets, emits agent-readable markdown mirrors, and optionally proxies page HTML rendering to a Node SSR sidecar.

The proposed direction is to formalize this behavior behind a backend contract and `/api/v1` endpoints, then support xgoja/JavaScript backends as either a hybrid Go-owned server or a standalone generated xgoja server.

## Key Links

- [Backend API design for dynamic publish-vault pages](./design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**. Initial investigation and design are complete; implementation is not started.

## Topics

- backend
- api
- ssr
- xgoja
- obsidian-vault

## Deliverables

- Evidence-backed architecture map of current publish-vault behavior.
- Gap analysis for dynamic pages and SSR.
- Proposed backend interface and `/api/v1` API shape.
- xgoja packaging modes and integration strategy.
- Phased implementation and test plan.

## Structure

- `design-doc/` - Primary architecture and API design.
- `reference/` - Chronological investigation diary.
- `playbooks/` - Future implementation/test runbooks.
- `scripts/` - Temporary code and tooling if implementation experiments are added.
