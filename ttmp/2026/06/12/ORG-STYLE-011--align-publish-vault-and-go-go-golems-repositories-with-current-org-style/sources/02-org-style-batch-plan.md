---
title: Org Style Batch Plan
doc_type: reference
status: active
intent: short-term
topics:
  - tooling
  - ci
  - linting
---

# Org Style Batch Plan

Generated from `sources/org-style-audit.json` on 2026-06-12.

## Summary

- Current or nearly current: 4 repositories
- Safe bump candidates: 70 repositories
- Legacy/manual review: 10 repositories

## Current or nearly current

These repositories already match the target Go and golangci-lint versions in the audit:

- `go-go-app-inventory`
- `go-template`
- `jesus`
- `smailnail`

## First safe batch candidates

These are good first candidates because they are clean working trees and already have modern tooling files. They usually need only a Go patch-version bump, a `.golangci-lint-version` bump, or both:

- `almanach`
- `bobatea`
- `cliopatra`
- `codex-sessions`
- `css-visual-diff`
- `discord-bot`
- `escuse-me`
- `font-util`
- `form-generator`
- `geppetto`

Suggested per-repo process:

```bash
cd ~/code/wesen/go-go-golems/<repo>
git status --short
# stop if dirty
# update go.mod go directive and/or .golangci-lint-version
GOWORK=off go mod tidy
make lint
make test
# if present and reasonable:
make logcopter-check
make ci-check
git add go.mod go.sum .golangci-lint-version Makefile .github/workflows
 git commit -m "Align tooling with org style"
```

## Second safe batch candidates

These are still likely safe, but may have more repo-specific validation because they are larger, central libraries, or have more dependencies:

- `clay`
- `docmgr`
- `glazed`
- `go-go-goja`
- `go-go-mcp`
- `go-minitrace`
- `logcopter`
- `remarquee`
- `sanitize`
- `sqleton`

## Manual review batch

These repositories use old Go directives or are missing modern tooling files. Do not mass-edit them without checking whether they are archived, intentionally old, or constrained by dependencies:

- `barbar`
- `biberon`
- `bubble-table`
- `bucheron`
- `common-sense`
- `ecrivain`
- `plunger`
- `raza`
- `terraform-provider-stytch-b2b`
- `voyage`

## Guardrails

- Do not modify dirty repositories.
- Do not add `glazed-lint` as a required CI step until the repository's Glazed dependency contains the linter package or a deterministic fallback version is chosen.
- Do not enable docs publishing unless the repository has a working help export command and a matching Vault role.
- Commit per repository; do not create a cross-repo mixed commit.
- Record validation failures in the ORG-STYLE-011 diary before moving to the next repository.
