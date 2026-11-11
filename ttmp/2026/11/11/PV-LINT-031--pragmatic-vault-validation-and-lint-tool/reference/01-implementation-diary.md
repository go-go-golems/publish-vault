---
Title: Implementation diary
Ticket: PV-LINT-031
Status: active
Topics:
    - vault
    - linting
    - markdown
    - parser
    - diary
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/retro-obsidian-publish/commands/root.go
      Note: Command registration
    - Path: repo://internal/parser/document.go
      Note: Document parsing and heading indexes
    - Path: repo://pkg/vault/vault.go
      Note: Loader publication rules and resolver APIs
ExternalSources: []
Summary: Chronological research and delivery record for the vault validation design.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation diary

## Step 1: Establish the ticket and investigate the current publication path

I created a new ticket for a design-only investigation, leaving tool implementation as an explicitly open future task. I traced the command entry point, vault loader, parser, resolver, configuration, and existing evidence script so the proposed linter can reuse real behavior instead of inventing a parallel publishing model.

### Prompt Context
**User prompt (verbatim):** "ok, create a new docmgr ticket for the lint tool, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\n\nThe document is detailed, the tool design is pragmatic, clean, straight forward, not a rabbit hole of hyperverification."

**Additional user prompt (verbatim):** "commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill) when done"

**Assistant interpretation:** Research and document a practical lint tool, not implement it; maintain a detailed diary, make coherent commits, and upload the finished guide to reMarkable.

**Inferred user intent:** Give an intern enough context and concrete next steps to implement a useful read-only validator without building a general-purpose framework or an exhaustive verification system.

### What I did
- Created ticket `PV-LINT-031`, a design document, this diary, and four tasks, leaving implementation explicitly pending.
- Inspected `cmd/retro-obsidian-publish/commands/root.go`, `commands/serve/serve.go`, `pkg/vault/vault.go`, `pkg/vaultconfig/config.go`, `internal/ignore/ignore.go`, `internal/parser/document.go`, and the prior inventory script at revision `469a4bf02707d299103de1d1a41df694d75a2e72`.
- Checked the diary, docmgr, ticket research, Glazed command authoring, and reMarkable delivery guidance.

### Why
- Existing loader APIs decide which notes are published and how links resolve; a separate directory walk with approximated semantics would create misleading diagnostics.

### What worked
- `docmgr ticket create-ticket`, `docmgr doc add`, and task creation succeeded. The current working tree initially contained only the previously identified unrelated untracked artifacts.

### What didn't work
- `docmgr ticket list --query lint` failed with `Error: unknown flag: --query`; the command's own help lists `--ticket` and `--status`, not `--query`. This did not affect creation of the explicitly named ticket.

### What I learned
- `vault.New` does not automatically apply `.publish/config.yaml`; `vault.WithConfig` must be supplied. The server resolves the configured root and loads this config before constructing a vault.
- `Note.WikiLinks` has no source locations, omits self-heading links, deduplicates references, and excludes wiki-image references; heading indexes are private to vault notes. Thus a useful line-precise validator needs a small parser/vault API addition, not a pretend call to existing public methods.
- The existing ticket inventory script counts output features and some broken markers; it is not a comprehensive lint tool.

### What was tricky to build
- Publish policy combines two exclusion matchers plus frontmatter, and the existing loader logs and skips some errors. A validator should use the same publication rules while avoiding a false claim that an incomplete load is clean.

### What warrants a second pair of eyes
- The design must clearly mark proposed APIs versus current APIs, specify config and snapshot behavior, avoid exposing excluded note contents, and avoid promising location data already discarded by the parser.

### What should be done in the future
- Complete the intern guide and its readability review, validate ticket metadata, commit the scoped documentation, and deliver the guide to reMarkable. Implement the tool separately.

### Code review instructions
- Start at `cmd/retro-obsidian-publish/commands/root.go`, `pkg/server/runtime.go`, `pkg/vault/vault.go`, and `internal/parser/document.go`; then read the proposed design.

### Technical details
- `vault.New` runs `LoadAll` and renders all retained notes. `ResolveAssetURL` constructs a URL but does not check that the target file exists. `ResolveAssetEmbed` does report existence for wiki embeds. A successful load is therefore not itself a complete validation result.
