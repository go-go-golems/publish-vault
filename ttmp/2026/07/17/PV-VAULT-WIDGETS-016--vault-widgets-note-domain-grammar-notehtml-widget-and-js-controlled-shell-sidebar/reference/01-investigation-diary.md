---
Title: Investigation diary
Ticket: PV-VAULT-WIDGETS-016
Status: active
Topics:
    - widget-dsl
    - goja
    - frontend
    - design-system
    - react
    - api
    - obsidian-vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Diary for PV-VAULT-WIDGETS-016: vault.widgets module, note-domain adapters, NoteHtml, and JS-controlled shell/sidebar."
LastUpdated: 2026-07-17T20:45:49.161469949-04:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Track the implementation of vault.widgets, the note-domain widget adapters, and JS shell/sidebar support.

## Step 1: Ticket and design doc created from conversation decisions

This ticket packages the Horizon-1 increment agreed in the 2026-07-17 conversation on top of PV-WIDGET-DSL-015. The design doc records the three framing decisions verbatim-in-spirit: adapters register existing components (NoteFrontmatter would add nothing to FrontmatterPanel — only NoteHtml is new); a sibling `vault.widgets` module gives first-class-feeling builders without forking rag-eval's widget.dsl (graduating to widget.vault.* when rag-evaluation-system#28 lands RegisterNamespace); and page.shell gets rendered so JS pages control the left sidebar via a VaultLayout sidebar slot. Implementation has not started.

### Prompt Context

**User prompt (verbatim):** "ok, do we have a design doc and ticket?"

**Assistant interpretation:** The vault.widgets / shell increment existed only in conversation; create its ticket + design doc so the work is durable.

**Inferred user intent:** Make the next work item pick-up-able later (or by someone else), consistent with how -015 and issue #28 were documented.

### What I did
- Created PV-VAULT-WIDGETS-016 with design doc (context, target authoring sketch, module/adapter/shell design, 4 decision records, phases A–D, test strategy, risks), 7 tasks, changelog; doctor green.

### What warrants a second pair of eyes
- The §2 `widget.app.shell` builder sketch is UNVERIFIED against v3's actual grammar — Phase C step 3 must true it up before anything depends on it.

### Code review instructions
- Read the design doc §3–§5; cross-check the helper inventory against `internal/vaultdata/vaultdata.go` patterns and the adapter list against `web/src/widgets/defaultRegistry.ts`.
