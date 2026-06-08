---
Title: Add per-site favicon.ico support and create go-go-parc favicon
Ticket: FAVICON-001
Status: active
Topics:
    - retro-obsidian-publish
    - assets
    - config
    - html-layout
    - obsidian-vault
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: CLI flags where --favicon will be added
    - Path: internal/server/favicon.go
      Note: New favicon handler with cascading resolution (commit 330838a)
    - Path: internal/server/favicon_test.go
      Note: 7 unit tests for favicon handler (commit 330838a)
    - Path: internal/server/runtime.go
      Note: RuntimeState with ResolvedRoot() needed to locate vault-root favicons
    - Path: internal/server/server.go
      Note: HTTP router where favicon routes are wired; the bug lives here
    - Path: internal/vault/vault.go
      Note: Vault loader that filters to .md files only
    - Path: internal/web/static.go
      Note: SPA handler that serves index.html for missing files including favicon
    - Path: web/index.html
      Note: SPA HTML shell that needs link rel=icon injection
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-08T18:23:02.307576388-04:00
WhatFor: ""
WhenToUse: ""
---









# Add per-site favicon.ico support and create go-go-parc favicon

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- retro-obsidian-publish
- assets
- config
- html-layout
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
