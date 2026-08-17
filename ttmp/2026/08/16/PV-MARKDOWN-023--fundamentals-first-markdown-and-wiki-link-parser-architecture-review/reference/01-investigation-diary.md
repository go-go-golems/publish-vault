---
Title: Investigation Diary
Ticket: PV-MARKDOWN-023
Status: active
Topics:
    - parser
    - wiki-link
    - frontmatter
    - html-rendering
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: Primary evidence source
    - Path: repo://pkg/vault/vault.go
      Note: Resolution and graph evidence source
    - Path: repo://ttmp/2026/08/16/PV-MARKDOWN-023--fundamentals-first-markdown-and-wiki-link-parser-architecture-review/scripts/01-parser-edge-probe/main.go
      Note: Chronological investigation probe
ExternalSources: []
Summary: Chronological evidence and decisions from the fundamentals-first Markdown and wiki-link parser architecture review.
LastUpdated: 2026-08-16T22:15:00Z
WhatFor: Preserve investigation commands, corrected assumptions, findings, design reasoning, and delivery evidence.
WhenToUse: Read before continuing PV-MARKDOWN-023 or implementing one of its proposed phases.
---


# Diary

## Goal

Map the complete Markdown/wiki-link system, identify the algorithms and invariants beneath recent incident fixes, find correctness risks at subsystem seams, and propose a robust reusable architecture that a new intern can understand and implement in phases.

## Step 1: Map the system, probe its seams, and design the typed replacement

The investigation began with the current pipeline rather than the recent diffs. The parser is a sequence of source transformations, a goldmark render, HTML string rewrites, vault-wide resolution, and a separate static implementation. The central finding is that one Markdown source is interpreted by several independent recognizers. Recent bugs are instances of those recognizers disagreeing about context or identity.

A ticket-local probe turned several code-review suspicions into observed behavior. It demonstrated mutation of valid four-dash frontmatter, loss of distinct link occurrences during deduplication, cross-line wiki-link recognition, raw alias HTML, and broad rewriting of unrelated HTML. The design therefore focuses on typed occurrences, one frontmatter splitter, goldmark AST extensions, ambiguity-aware resolution, and renderers over typed state.

### Prompt Context

**User prompt (verbatim):** "We are taking over for your little brother who has been working on the markdown parsing and on the project report. We want to do a full pass on the markdown handling and wiki link parsing in this project. Create anew docmgr ticket for doing this algorithmic focused + code review, focused on looking for the fundamentals underneath it all and seeing how to make this a robust building block that can be built upon for other similar projects too.

Create  a detailed analysis / architecture, design / code review  that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a fresh ticket, audit the entire Markdown/wiki-link subsystem from algorithmic fundamentals through vault consumers and the static renderer, document evidence-backed findings and a reusable target architecture for an intern, validate the ticket, and deliver it as a reMarkable bundle.

**Inferred user intent:** Move beyond incremental parser bug fixes and establish a coherent technical foundation that can support this project and future Markdown/Obsidian publishing systems.

### What I did

- Created ticket `PV-MARKDOWN-023` with a primary design document, investigation diary, API/test inventory, tasks, and ticket-local probe script.
- Enumerated the parser's public API, 49 parser/math tests, vault consumers, static TypeScript implementation, README contracts, and goldmark dependencies.
- Read the core paths:
  - `internal/parser/parser.go` (1009 lines);
  - `internal/parser/math.go` (480 lines);
  - `pkg/vault/vault.go` load/index/backlink/rebuild paths;
  - `web/src/vault/staticVault.ts` grammar, graph, and render paths;
  - parser, math, and vault regression tests.
- Verified goldmark v1.8.2 extension APIs from the installed module: `parser.InlineParser`, `renderer.NodeRenderer`, and the strikethrough extension pattern.
- Added and ran `scripts/01-parser-edge-probe/main.go`.
- Ran `go test ./internal/parser ./pkg/vault -count=1`; both packages passed.
- Searched for fuzz/property tests and static-vault wiki-link tests; none exist.
- Wrote the current-state inventory and the target architecture with API sketches, pseudocode, Mermaid diagrams, decision records, phases, test strategy, risks, and open questions.

### Why

The parser's correctness is not localized to `Parse`. Rendered HTML, structured links, backlinks, search text, and static builds are separate products. A robust review must trace the same source construct through every product and identify where a representation changes from typed source to bytes, HTML, or a single-value index.

The edge probe was necessary because several important findings are not visible from a normal rendered page. A body link dropped from `WikiLinks` can still render. A frontmatter scalar containing injected anchor HTML still parses as valid metadata. An unrelated HTML attribute rewritten by a global regex remains valid HTML. Concrete outputs turn those concerns into reviewable facts.

### What worked

- The existing code comments and incident tests explain many ordering constraints precisely, especially math restoration, heading IDs, code fences, and reversible reloads.
- The ticket probe reproduced all targeted seams without modifying production code.
- Goldmark provides a direct typed extension route; the proposal does not require replacing the Markdown engine.
- The existing static renderer already uses a `marked` inline extension for HTML rendering, which supports the recommendation. Its remaining inconsistency is graph extraction, not rendering.
- Existing tests remained green throughout the documentation-only investigation.

### What didn't work

The initial historical note that the static renderer "has no code-region scanner and preprocesses with a bare regex" is no longer fully accurate. `staticVault.ts` now registers a marked inline extension and no longer calls `preprocessWikiLinks` for rendering. Reading current code corrected the claim: rendering is parser-context-aware, but outgoing-link extraction still uses a global regex, so code samples can still create static backlinks. The design and inventory record the current split rather than repeating the outdated finding.

No production failure occurred during the investigation. The code-review probe intentionally exposes existing behavior; its surprising outputs are findings, not probe failures.

### What I learned

- The system's fundamental problem is representation fragmentation, not simply regex usage. Regex is safe for narrowly bounded strings; it becomes fragile here because typed parser state is encoded in HTML and recovered later.
- An occurrence and a graph edge require different equivalence relations. Early deduplication is an architectural decision even when written as a three-line optimization.
- Ambiguity is part of link resolution's return type. A `map[string]string` cannot represent it, and first-wins behavior is not deterministic when populated from a Go map.
- Frontmatter boundary parsing is part of syntax protection. A splitter that differs from the metadata parser can mutate metadata before it is parsed.
- Backend/static parity requires shared expected behavior, not necessarily one cross-compiled parser implementation.

### What was tricky to build

**Separating immediate defects from target architecture.** The four-dash frontmatter mutation and broad HTML rewrite should be fixed without waiting for an AST migration. The design therefore has a Phase 0 correctness layer and later structural phases. This avoids using a large refactor as a prerequisite for small high-severity fixes.

**Choosing the reusable boundary.** Vault resolution depends on note inventory, publication state, assets, and heading IDs; Markdown syntax does not. The proposed API keeps parsing in a reusable `markdown.Engine` and supplies a resolver/index separately. This allows another project to reuse syntax and renderers with a different storage/index policy.

**Preserving compatibility while removing HTML protocols.** Current heading IDs and note slugs are public URLs. The design resolves typed nodes before rendering but explicitly preserves `Slugify` and goldmark IDs. Structural cleanup is not permission to change URL identity.

### What warrants a second pair of eyes

- The severity and compatibility policy for ambiguous short links. Strict ambiguity is safer; existing vault content may rely on arbitrary first-wins resolution.
- The decision to keep frontmatter `[[...]]` as backlinks. The current vault has 123 notes using such values, but rendering omits them.
- Whether aliases are plain text, Markdown inline content, or trusted raw HTML.
- The memory cost of retaining goldmark ASTs for all notes versus copying a compact IR and reparsing for render.
- Goldmark engine reuse and concurrency guarantees should be measured before a singleton implementation.
- Browser consumers of current `data-*` attributes must be inventoried before removing internal transport fields.

### What should be done in the future

- Open focused implementation tickets for Phase 0 findings F1–F5.
- Add a deterministic ambiguity audit over the real vault before changing resolver policy.
- Prototype the goldmark wiki-link extension behind a feature flag and run old/new differential parsing over go-go-parc.
- Define the cross-language conformance fixture schema before adding more syntax to either backend.
- Keep math masking intact until a typed math prototype passes every current test and showcase note.

### Code review instructions

Start with:

1. `reference/02-parser-api-algorithm-and-test-inventory.md` for the map and finding table.
2. The primary design document for the proposed architecture and decisions.
3. `scripts/01-parser-edge-probe/main.go`, then run:

```bash
go run ./ttmp/2026/08/16/PV-MARKDOWN-023--*/scripts/01-parser-edge-probe
go test ./internal/parser ./pkg/vault -count=1
```

Verify finding F1 by comparing `splitFrontmatter` (`parser.go:361–378`) with `isFrontmatterDelimiter` (`parser.go:959–970`). Verify F2 at `vault.go:371–407`. Verify F3 with the final probe output and `ReplaceWikiLinksString` at `parser.go:556–590`.

### Technical details

Evidence counts at investigation time:

- `internal/parser`: 1489 production lines, 1351 test lines.
- 39 parser tests and 10 math tests, plus vault integration tests.
- No Go fuzz/property test functions.
- No static-vault wiki-link tests.
- Goldmark v1.8.2 and goldmark-meta v1.1.0.
- Core current tests: `go test ./internal/parser ./pkg/vault -count=1` passed.

## Step 2: Validate and deliver the review

The completed documents were validated as a docmgr ticket and as executable evidence. Frontmatter validation passed for all three focused documents, `docmgr doctor` reported all checks passed, the parser/vault tests remained green, and the ticket probe executed successfully.

The index, primary design, inventory, and diary were then bundled into one PDF with a depth-two table of contents. The dry-run showed the exact four inputs and destination; the real upload returned an explicit success response.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Complete the documentation lifecycle by validating the ticket and delivering the intern-oriented bundle to reMarkable.

**Inferred user intent:** Make the review immediately available as a durable reading package, not only as repository files.

### What I did

- Ran frontmatter validation on the design and reference documents.
- Ran `docmgr doctor --ticket PV-MARKDOWN-023 --stale-after 30`.
- Ran `go test ./internal/parser ./pkg/vault -count=1` and the edge probe.
- Ran a reMarkable bundle dry-run with four documents.
- Uploaded `PV-MARKDOWN-023 Markdown Parser Architecture Review.pdf` to `/ai/2026/08/16/PV-MARKDOWN-023`.

### Why

The deliverable combines architecture, evidence, and chronology. Bundling keeps the reading order and table of contents intact while the ticket retains independently searchable source documents.

### What worked

- All docmgr checks passed.
- Both Go package test suites passed.
- The upload command returned `OK: uploaded PV-MARKDOWN-023 Markdown Parser Architecture Review.pdf -> /ai/2026/08/16/PV-MARKDOWN-023`.

### What didn't work

N/A. The dry-run and real upload succeeded on the first attempt.

### What I learned

The four-document bundle is the useful review unit: the design explains direction, the inventory supports claim verification, the diary preserves reasoning, and the index gives an intern a reading order.

### What was tricky to build

The reMarkable bundle needed documents ordered by reader intent rather than filesystem order: index first, design second, inventory third, diary last. This preserves the conceptual progression from orientation to proposal to evidence to chronology.

### What warrants a second pair of eyes

The uploaded document is a design review, not an accepted implementation specification. The ambiguity policy, frontmatter-link semantics, alias trust mode, and AST retention strategy remain proposed decisions.

### What should be done in the future

Create focused implementation tickets for Phase 0 findings before beginning the AST migration.

### Code review instructions

- Run `docmgr doctor --ticket PV-MARKDOWN-023 --stale-after 30`.
- Run `go test ./internal/parser ./pkg/vault -count=1`.
- Run the ticket probe and compare output with the finding inventory.

### Technical details

- Remote directory: `/ai/2026/08/16/PV-MARKDOWN-023`
- PDF: `PV-MARKDOWN-023 Markdown Parser Architecture Review.pdf`
- Bundle ToC depth: 2

## Step 3: Extract and publish the reusable pattern family

The architecture review yielded one coherent Garden candidate rather than four disconnected refactoring slogans. The entry names four compositional laws: parser-owned structural context, occurrence preservation before projection, explicit ambiguity, and typed resolution before rendering.

The Garden source was committed and pushed before issue creation. The public PARC URL was initially unavailable, so the repository issue used the verified GitHub source URL as a temporary fallback; once PARC publication propagated, the issue was updated to the public reader-facing URL. The issue was added to the Architecture & Pattern Catalog with honest `Documented` maturity and explicit provenance.

### Prompt Context

**User prompt (verbatim):** "Let's use the occasion to extract design patterns and such /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/playbooks/creating-github-issues-and-software-design-garden-entries.md"

**Assistant interpretation:** Apply the referenced playbook to turn PV-MARKDOWN-023's reusable laws into a published Garden entry, repository issue, and Architecture Catalog item.

**Inferred user intent:** Preserve the parser review as cross-project architecture knowledge and make it discoverable to both publish-vault maintainers and the wider ecosystem.

### What I did

- Created Garden entry `publish-vault/05 - Parser-Owned Structure and Typed Reference Resolution.md`.
- Committed/pushed the entry (`5d55012`) and metadata follow-up (`b47db6c`).
- Verified the GitHub source and public PARC rendering.
- Created publish-vault issue #22 with the `documentation` label.
- Added issue #22 to project #3 as item `PVTI_lADOB23p8s4BgM1Yzg2wPV0`.
- Set status `Documented` and populated ticket/session/working-directory provenance.

### Why

The four laws protect one boundary: source meaning stays typed from parser context through resolution and consumer-specific projection. One integrated entry preserves their dependency; four isolated issues would make each look like a local refactor.

### What worked

- Staged commits contained only the new Garden entry.
- PARC eventually published the intended URL and rendered the expected title.
- The issue, label, catalog status, and provenance fields all verified independently.

### What didn't work

The first three PARC checks returned HTTP 404 because publication was asynchronous. The workflow used the verified GitHub source URL temporarily, then replaced it with the public PARC URL after publication succeeded.

### What I learned

A delayed public publisher is not a reason to guess or block issue creation indefinitely. The verified repository source is a valid fallback when the issue is updated promptly after public publication.

### What was tricky to build

The main judgment was granularity. The extracted laws are individually reusable, but in this source they form one dataflow contract. Treating them as one pattern family keeps the parser-context, occurrence, resolution, and rendering laws connected while the issue remains concise enough for maintainers.

### What warrants a second pair of eyes

The catalog status is `Documented`, not `Validated`. Validation requires a second independent implementation or deliberate adoption with conformance evidence.

### What should be done in the future

Compare the candidate against another parser/reference system, then decide whether individual laws deserve separate validated entries.

### Code review instructions

- Public note: https://parc.yolo.scapegoat.dev/note/research/software-architecture-garden/publish-vault/05-parser-owned-structure-and-typed-reference-resolution
- Issue: https://github.com/go-go-golems/publish-vault/issues/22
- Catalog: https://github.com/orgs/go-go-golems/projects/3

### Technical details

- Garden commits: `5d55012`, `b47db6c`
- Catalog item: `PVTI_lADOB23p8s4BgM1Yzg2wPV0`
- Catalog status: `Documented`

## Step 4: Build the publish-vault back-of-the-book index

The publish-vault Garden study now has a hybrid index/glossary, notation table, companion rationale, and backlinks from the project README. The index spans the original runtime/deployment/debt/guideline documents and the newer parser-owned reference pattern without flattening their distinct source snapshots.

### Prompt Context

**User prompt (verbatim):** "Now tackle /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/playbooks/creating-an-index.md"

**Assistant interpretation:** Apply the index playbook to the publish-vault Architecture Garden project and publish the index/rationale safely.

**Inferred user intent:** Make the completed architecture study retrievable by remembered concept, failure mode, operation, and implementation handle rather than only by document order.

### What I did

- Created `Index of Design Patterns.md` with 44 anchored entries/redirects, a notation table, and cross-Garden correspondence summary.
- Created `Index of Design Patterns - Rationale.md` with selection/exclusion principles, per-term rationale, and 24 reader situations.
- Added index/rationale backlinks to the project README.
- Ran the shipped link validator on the index/rationale and its negative control.
- Committed and pushed only the two new files and README edit as `a9cbe51`.

### Why

The project now contains five studies across runtime, deployment, debt, guidelines, and parser architecture. An alphabetic conceptual access layer prevents a reader from needing to remember which numbered document owns a concern.

### What worked

- Every link in the new index and rationale resolves.
- All 47 intra-index targets exist.
- The negative validator fixture correctly reported a missing anchor and missing file with exit code 1.
- The staged commit contained exactly three intended files.

### What didn't work

The project README validator still reports pre-existing schema and legacy basename-link errors. The new index/rationale backlinks are not among them. Initial PARC checks for the two new files returned 404 while asynchronous publication was pending; GitHub source copies were verified.

### What I learned

The most useful access paths are conflation corrections and reader tasks: “discovery order is not identity,” “graph edge is not occurrence,” “HTML is not an IR,” safe reload, and static/backend parity. These redirects do more retrieval work than indexing implementation nouns.

### What was tricky to build

The index spans two evidence snapshots. The original study remains pinned to `560e71d`; the parser pattern and index use `e02b73d`. The index states this provenance rather than rewriting the original study as if it had analyzed current code.

### What warrants a second pair of eyes

The project README's pre-existing validator debt should be addressed separately; this workflow intentionally did not rewrite unrelated historical links or retrofit the newer Garden schema.

### What should be done in the future

Add redirects based on real reader searches, and rerun the validator whenever study headings change.

### Code review instructions

Run:

```bash
python3 Research/playbooks/scripts/validate_index_links.py \
  "Research/Software Architecture Garden/publish-vault/Index of Design Patterns.md" \
  "Research/Software Architecture Garden/publish-vault/Index of Design Patterns - Rationale.md"
```

### Technical details

- Garden commit: `a9cbe51`
- Index: 279 lines / 2447 words
- Rationale: 259 lines / 2399 words
- Reader situations: 24
