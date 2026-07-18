---
Title: Diary
Ticket: PV-FRAMEWORK-017
Status: active
Topics:
    - vault
    - widget-dsl
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-18T19:21:24.194042001-04:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Chronological record of the framework-ification work: turning publish-vault from a
standalone binary (module `retro-obsidian-publish`) into an importable Go framework
at `github.com/go-go-golems/publish-vault`, so golem-docs (and future apps) can
depend on it.

## Step 1: Ticket, evidence gathering, and design doc

Created PV-FRAMEWORK-017 and wrote the intern-level design doc. Before writing, I
gathered line-anchored evidence: package line counts, the internal dependency graph
(computed by grepping each package's imports), the embed mechanism, and every
non-Go file that hardcodes the module name. The design landed on five decisions:
rename in place (D1), curated promotion of nine packages to pkg/ with ignore+parser
staying internal (D2), keep the cmd/binary name (D3), tagged-assets release flow
plus a Config-level fs.FS escape hatch for the frontend (D4), and v0.x versioning
(D5).

### Prompt Context

**User prompt (verbatim):** "Ok, create a new docmgr ticket in publish-vault, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

For step 1.

But before you start implementing, create the golem-docs repository under the go-go-golems org, based on ~/code/wesen/go-go-golems/go-template, make sure docmgr's ttmp/ isproperly created in there, and create a new docmgr ticket in golem-docs and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

Then, work on Step 1. 

Keep a detailed diary as you work, and commit at appropriate intervals."

**Assistant interpretation:** Three deliverables in order: (1) PV-FRAMEWORK-017 ticket + intern design doc in publish-vault covering "Step 1" of the docs-server plan (framework-ification), uploaded to reMarkable; (2) a new golem-docs repo from go-template with docmgr initialized and its own ticket + design doc, uploaded to reMarkable; (3) implement the framework-ification, with diary and interval commits.

**Inferred user intent:** Get the glazed/go-go-golems docs server project moving on solid rails: durable onboarding docs for both halves of the work, the new repo scaffolded correctly from the start, and the prerequisite refactor actually done.

### What I did

- `docmgr ticket create-ticket --ticket PV-FRAMEWORK-017 …` + design-doc + diary docs.
- Evidence sweep: per-package `wc -l`; dependency graph via `grep -rh "retro-obsidian-publish/internal"` per package; `grep -rln retro-obsidian-publish` for non-Go references; API surface via `grep -n "func New\|type Config"` over the key packages.
- Wrote the design doc (12 sections, decisions D1–D5, 5 phases, testing matrix).
- Related the six load-bearing files to the doc via `docmgr doc relate`.

### Why

- The rename/promotion is mechanical but wide (15 generated logcopter files bake in
  the module path; Dockerfile embeds from internal/web); an intern-level map of every
  hardcoded reference prevents the classic "compiles locally, breaks in Docker" miss.

### What worked

- The dependency graph came out a clean DAG with exactly two private leaves
  (ignore, parser), which made the curation decision (D2) obvious.
- `server.Run(ctx, Config{…})` already being the single entrypoint means the
  downstream story needs no new API, only visibility.

### What didn't work

- N/A for this step (analysis only).

### What I learned

- Go permits `pkg/` packages to import their own module's `internal/` packages, so
  keeping parser/ignore private costs nothing downstream.
- The Dockerfile/CI reference the *filesystem path* `./cmd/retro-obsidian-publish`,
  not the module path, so the module rename does not touch the deployment surface;
  only the Phase-2 embed-path move does.

### What was tricky to build

- The frontend-delivery decision (D4). go:embed only sees files inside the module
  zip, and the built SPA is deliberately not in git. Committing dist to main causes
  permanent generated-diff noise; making downstream build the SPA pushes Dagger/pnpm
  complexity onto every consumer. The resolution: a dispatch-driven release workflow
  that creates a single assets commit reachable only from the tag ref, so main stays
  clean while tagged versions are self-contained; plus an fs.FS override as the
  escape hatch.

### What warrants a second pair of eyes

- D2's curation list — if any exported symbol in a promoted package references an
  internal type, downstream code cannot name it; Phase 2 has an explicit go-doc
  check for this.

### What should be done in the future

- Implement Phases 1–5 (tracked as ticket tasks).

### Code review instructions

- Read the design doc top to bottom; check §3.5 (hardcoded references) against
  `grep -rln retro-obsidian-publish` output.

### Technical details

- Dependency graph and package inventory are reproduced in design doc §3.1–3.2.
