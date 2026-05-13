---
Title: Retro Obsidian Publish initial assessment and setup plan
Ticket: RETRO-SETUP-001
Status: active
Topics:
    - glazed
    - frontend
    - dagger
    - devctl
    - pnpm
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Updated project architecture and setup documentation referenced by the ticket.
    - Path: backend/cmd/retro-obsidian-publish/main.go
      Note: Current single-binary Glazed CLI entrypoint.
    - Path: backend/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Glazed serve verb replacing the old flag-based backend command.
    - Path: backend/cmd/retro-obsidian-publish/commands/build/web.go
      Note: Dagger-backed build web verb for embedded single-binary assets.
    - Path: web/package.json
      Note: Current pnpm package file after the web/ migration.
    - Path: web/vite.config.ts
      Note: Current Vite configuration after the web/ migration.
    - Path: plugins/retro-obsidian-publish.py
      Note: devctl plugin for local orchestration.
ExternalSources:
    - devctl help user-guide
    - devctl help scripting-guide
    - devctl help plugin-authoring
Summary: "Ticket workspace for the initial setup assessment and intern implementation guide covering Glazed CLI migration, web/pnpm layout, Dagger bundling, and devctl orchestration."
LastUpdated: 2026-05-13T13:45:00-04:00
WhatFor: "Use this ticket to drive the first setup/migration implementation for retro-obsidian-publish."
WhenToUse: "Before implementing or reviewing the developer-experience and build-system migration."
---

# Retro Obsidian Publish initial assessment and setup plan

## Overview

This ticket contains the initial assessment and implementation guide for preparing `retro-obsidian-publish` for a more standard development setup:

- migrate backend CLI flags and verbs into a Glazed/Cobra command structure;
- keep pnpm but move the frontend application into `web/`;
- add Dagger-backed web bundling;
- add devctl support for full-stack local development.

## Primary deliverables

- [Initial assessment and setup implementation guide](./design-doc/01-initial-assessment-and-setup-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)

## Status

Current status: **active**. The assessment and design guide are complete; implementation remains as follow-up work.

## Topics

- glazed
- frontend
- dagger
- devctl
- pnpm

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` - Architecture and design documents
- `reference/` - Investigation diary and reusable context
- `playbooks/` - Command sequences and test procedures
- `scripts/` - Temporary code and tooling
- `various/` - Working notes and research
- `archive/` - Deprecated or reference-only artifacts
