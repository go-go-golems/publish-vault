---
Title: Mobile responsive layout, configurable vault name, remove Obsidian Publish branding
Ticket: RETRO-MOBILE-004
Status: active
Topics:
    - mobile
    - config
    - vault-name
    - glazed
    - responsive
    - css
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: backend/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Added --vault-name Glazed flag
    - Path: backend/internal/api/api.go
      Note: Added /api/config endpoint and SiteConfig type
    - Path: backend/internal/server/server.go
      Note: Pass VaultName to API handler
    - Path: web/src/App.tsx
      Note: Fetch config and pass vaultName to VaultLayout
    - Path: web/src/components/organisms/Sidebar/Sidebar.tsx
      Note: Removed Obsidian Publish footer
    - Path: web/src/components/pages/NotePage/NotePage.tsx
      Note: Mobile inline backlinks
    - Path: web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: Mobile drawer layout
    - Path: web/src/index.css
      Note: Mobile responsive CSS overrides
    - Path: web/src/store/vaultApi.ts
      Note: Added useGetConfigQuery hook and SiteConfig type
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-15T13:43:49.516478407-04:00
WhatFor: ""
WhenToUse: ""
---


# Mobile responsive layout, configurable vault name, remove Obsidian Publish branding

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- mobile
- config
- vault-name
- glazed
- responsive
- css

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
