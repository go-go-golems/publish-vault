---
Title: Reorganize publish-vault web/ into a design system and add a widget.dsl-styled go-go-goja JS API
Ticket: PV-WIDGET-DSL-015
Status: active
Topics:
    - frontend
    - design-system
    - widget-dsl
    - goja
    - xgoja
    - react
    - api
    - ssr
    - obsidian-vault
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /home/manuel/code/wesen/go-go-golems/rag-evaluation-system (widget DSL v3 reference implementation)
    - /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/widget-dsl.md (Widget DSL MOC)
Summary: "Ticket workspace for finishing publish-vault's web/ design-system reorganization (foundation/atoms/layout/molecules/organisms tiers, scaffold removal, token layering) and adding a widget.dsl-v3-styled go-go-goja JS API with Widget IR renderer, aligned with rag-evaluation-system and PV-BACKEND-API-001."
LastUpdated: 2026-07-17T16:45:00-04:00
WhatFor: "Entry point for the widget DSL / design-system reorganization design and implementation work."
WhenToUse: "When reorganizing web/src, embedding goja in the Go server, or adding widget IR rendering."
---

# Reorganize publish-vault web/ into a design system and add a widget.dsl-styled go-go-goja JS API

## Overview

publish-vault's frontend already uses atomic-design folders but carries heavy Manus/shadcn scaffold, a 988-line style monolith, and a concern-mixing 431-line NoteRenderer; the Go backend has no JS runtime. rag-evaluation-system has meanwhile matured widget DSL v3: a single `widget.dsl` goja module producing validated Widget IR rendered by a registry-driven React `WidgetRenderer` with server-round-trip actions. This ticket designs the port: Track A finishes the design system in place (five tiers, token layering, scaffold deletion, NoteRenderer decomposition); Track B embeds goja, imports the v3 DSL, adds a read-only `vault.data` module, serves `/api/widget/pages/{id}` + `/api/widget/actions/{name}`, and ports the IR renderer into `web/src/widgets/`. The design layers on PV-BACKEND-API-001 (widget pages become `Page.kind: "dynamic"`).

## Key Links

- [Primary design/implementation guide](./design-doc/01-widget-dsl-and-design-system-reorganization-analysis-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md) · [Changelog](./changelog.md)
- Prior ticket: [PV-BACKEND-API-001 backend API design](../../../06/22/PV-BACKEND-API-001--design-backend-api-for-dynamic-publish-vault-pages-and-xgoja-servers/design-doc/01-backend-api-design-for-dynamic-publish-vault-pages.md)

## Status

Current status: **active**. Investigation and design complete (2026-07-17); implementation not started. Six decision records (D1–D6) proposed in the design doc; five implementation phases defined.

## Deliverables

- Evidence-anchored map of publish-vault web/ + serving path.
- Evidence-anchored map of rag-evaluation-system widget DSL v3 (grammar, spec/IR, renderer, host, docs lineage).
- Target architecture and decision records for both tracks.
- Phased, file-level implementation plan with validation commands and test strategy.

## Topics

- frontend, design-system, widget-dsl, goja, xgoja, react, api, ssr, obsidian-vault

## Structure

- design-doc/ - Primary analysis/design/implementation guide
- reference/ - Investigation diary
- playbooks/ - (future) implementation runbooks
- scripts/ - (future) experiments
