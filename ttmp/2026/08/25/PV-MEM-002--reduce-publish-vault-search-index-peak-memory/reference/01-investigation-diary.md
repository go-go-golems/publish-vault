---
Title: Investigation Diary
Ticket: PV-MEM-002
Status: active
Topics:
    - memory
    - profiling
    - search
    - bleve
    - performance
    - reload
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/08/25/ARTICLE - Measure - Phase-Aware Memory Measurement for Go Programs.md
      Note: Durable textbook report created before this ticket
    - Path: abs:///home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/08/25/ARTICLE - Publish Vault Memory Optimization - From OOM Incidents to Phase-Attributed Baselines.md
      Note: Durable optimization-process report and baseline narrative
    - Path: repo://pkg/search/search.go
      Note: Primary search implementation evidence inspected during ticket design
    - Path: repo://pkg/server/runtime.go
      Note: Primary runtime evidence inspected during ticket design
ExternalSources: []
Summary: Chronological research and implementation diary for attributing and reducing publish-vault search-index peak memory while preserving search behavior and atomic snapshot reloads.
LastUpdated: 2026-08-25T18:00:00-04:00
WhatFor: Preserving prompts, evidence, decisions, commands, failures, review guidance, and continuation state for PV-MEM-002.
WhenToUse: Read before resuming profiling, experiments, implementation, validation, or review for this ticket.
---


# Diary

## Goal

This diary records the evidence-first investigation and implementation of publish-vault search-index memory reduction. The baseline is the MEASURE-001 personal-vault run: persistent indexing has a 391,219,024-byte median peak heap and 482,586,624-byte median peak RSS, and `search_index` is the dominant phase. The work must preserve search semantics, snapshot atomicity, rollback, serialized reloads, privacy, and generated-fixture regression budgets.

## Step 1: Preserve the completed measurement work as two textbook-style project reports

Before opening a new optimization ticket, I converted the existing MEASURE-001 design, diary, commits, source code, baseline, and historical publish-vault report into two durable Obsidian articles. The first explains the standalone Go measure package from collector semantics through recorder lifecycle, JSONL, CLI, budgets, Prometheus, and Grafana. The second explains the complete publish-vault memory process from the original OOM shape through persistent snapshot indexes and the current phase-attributed baseline.

This ordering matters. The new ticket should not carry the burden of re-explaining all completed infrastructure, and the vault reports provide a durable technical narrative independent of a task checklist. Both reports use direct system definitions, code/API references, Mermaid diagrams, concrete measurements, failure history, and working rules without analogies.

### Prompt Context

**User prompt (verbatim):** "Ok, create the new ticket and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Before you do that, use design doc + diary + past git commits or whatever to  write a detailed project report for the obsidian vault as a deep dive technical analysis blog post using a textbook writing style (no analogies, see skill).
 Commit and push the bsidian vault when done (go-go-parc vault).

Actually two reports:

One about the golang measure package
One about th epublish vault optimization process."

**Assistant interpretation:** First write and push two evidence-backed textbook-style deep-dive articles to the go-go-parc Obsidian vault, one for measure and one for publish-vault memory optimization. Then create a new publish-vault search-index optimization ticket, write an intern-ready analysis/design/implementation guide and diary, validate the ticket, and upload it to reMarkable.

**Inferred user intent:** Preserve the completed project's engineering knowledge in durable blog-quality form, then separate the next optimization problem into a focused ticket that a new engineer can understand and execute without reconstructing prior incidents or weakening runtime invariants.

**Commit (code):** `5f9f18c` — `docs: add measure and publish-vault memory deep dives` (go-go-parc documentation repository)

### What I did

- Loaded the project-report, textbook-authoring, Obsidian-vault-writing, ticket-research, diary, docmgr, and reMarkable instructions.
- Read the go-go-parc project and article exemplars.
- Read the existing July publish-vault memory architecture article to avoid erasing or duplicating historical context.
- Read the MEASURE-001 architecture guide and complete diary through Phases 3 and 4, plus the Phase 5 baseline summary.
- Inspected measure package APIs, source inventory, and commits `e699821`, `0bdb79b`, and `a3f4b04`.
- Inspected publish-vault source, tests, and commits `4d597ac` and `83bb1f2`.
- Wrote:
  - `Projects/2026/08/25/ARTICLE - Measure - Phase-Aware Memory Measurement for Go Programs.md`;
  - `Projects/2026/08/25/ARTICLE - Publish Vault Memory Optimization - From OOM Incidents to Phase-Attributed Baselines.md`.
- Validated balanced code fences, YAML frontmatter presence, and `git diff --check`.
- Staged only the two intended notes, committed them, and pushed go-go-parc `main`.

### Why

The measure package and publish-vault optimization are related but distinct technical systems. One article explains reusable measurement semantics; the other explains application-specific snapshot, search, and memory behavior. Separate reports keep both topics navigable and allow the new ticket to reference stable background material.

### What worked

- go-go-parc was clean before writing.
- The measure article is 417 lines and approximately 2,871 words.
- The publish-vault article is 575 lines and approximately 3,408 words.
- `git diff --check` passed.
- Commit `5f9f18c` created exactly two files and pushed successfully:

  ```text
  To ssh://github.com/go-go-golems/go-go-parc
     810cc8b..5f9f18c  main -> main
  ```

### What didn't work

No report-writing or Git command failed in this step. The existing MEASURE-001 Phase 5 documentation is incomplete, so the reports distinguish technically validated implementation/baselines from formally closed ticket bookkeeping rather than claiming the old goal was completed.

### What I learned

The current work is the second generation of publish-vault memory engineering. The earlier 890-note incident established persistent per-snapshot indexes and reduced production memory sharply. The current 3,395-note workload establishes that persistent mode remains much better than in-memory mode but still concentrates 391 MB heap and 483 MB RSS in `search_index`. The next ticket must analyze that narrower phase rather than reopening the already-settled persistent-versus-memory decision.

### What was tricky to build

The difficult writing problem was keeping three scopes separate:

1. Generic measure contracts and platform semantics.
2. Historical publish-vault OOM and persistent-index architecture.
3. The current MEASURE-001 instrumentation and larger-vault baseline.

Combining them into one report would obscure which decisions are reusable and which belong to publish-vault. The two-report structure preserves the dependency between them without making the measurement package appear application-specific.

### What warrants a second pair of eyes

- Review whether the current vault article should link to the July 5 article through an explicit Obsidian wikilink in addition to the file reference.
- Review numerical conversions and terminology: source measurements are retained in bytes, while prose uses rounded MB only for readability.
- Confirm whether the historical 183 MiB production observation and current 483 MB process baseline need a dedicated note explaining workload and deployment differences; the new article already warns that they are not directly interchangeable.

### What should be done in the future

Add follow-up links from the two articles to the final PV-MEM-002 implementation report after optimization work is complete. Do not overwrite the historical July article.

### Code review instructions

- Review the measure article's sections on availability, child process semantics, canonical JSONL, and cardinality against the measure source.
- Review the publish-vault article's baseline table against `phase5-baseline/summary.json`.
- Verify go-go-parc commit `5f9f18c` contains only the two reports.

### Technical details

```text
Vault root: /home/manuel/code/wesen/go-go-golems/go-go-parc
Branch: main
Commit: 5f9f18c
Measure report: 417 lines, ~2,871 words
Publish-vault report: 575 lines, ~3,408 words
```

## Step 2: Create PV-MEM-002 and define the evidence-first optimization plan

I created `PV-MEM-002` in the publish-vault checkout and wrote the primary intern-oriented guide. The guide starts from runtime and data-model foundations, explains the exact snapshot and search pipeline, records the measured baseline, separates heap/RSS/cgroup meanings, defines a secure profiling protocol, and maps each possible optimization to evidence that would support or reject it.

The design intentionally does not choose batching, mapping changes, forced GC, or a backend replacement yet. `search_index` is broad enough that any of those could be correct or counterproductive. The first implementation phase is attribution with aligned measure traces, heap profiles, smaps, and cgroup checkpoints. Raw profiles remain private because they can contain note content.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Create a focused search-index memory ticket with enough technical context, APIs, pseudocode, diagrams, references, experiment design, and acceptance criteria for a new intern to execute safely.

**Inferred user intent:** Make the next optimization reproducible and reviewable, using the new measure package as stable infrastructure rather than mixing more generic instrumentation into application-specific changes.

**Commit (code):** N/A — ticket research and design only.

### What I did

- Confirmed the publish-vault docmgr store and vocabulary already include memory, profiling, search, Bleve, performance, and reload topics.
- Verified `PV-MEM-002` did not already exist.
- Created:

  ```text
  ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/
  ```

- Added the primary design doc and this diary.
- Added six tasks covering baseline refresh, attribution, lifetime inventory, implementation, correctness, and repeated comparison/rollout guidance.
- Read exact source ranges for:
  - `RuntimeState.Reload`, `loadSnapshot`, `buildSearchIndex`, and delayed release;
  - `runtimeMeasurement` and `measurementRun`;
  - vault load stages and progress;
  - `SearchDocument`, `ForEachSearchDocument`, and `ReadRaw`;
  - Bleve constructors, `indexVault`, `Index`, and `buildMapping`;
  - generated-fixture memory budgets.
- Wrote a long-form design guide containing:
  - reader orientation and definitions;
  - current snapshot, vault, search, publication, and measurement architecture;
  - baseline tables and interpretation;
  - explicit questions for heap/RSS/cgroup/lifetime analysis;
  - profile security and checkpoint protocol;
  - six falsifiable hypothesis designs;
  - ticket-local harness and run matrix;
  - API sketches and search-equivalence pseudocode;
  - six decision records;
  - six implementation phases;
  - unit, search, snapshot, measurement, memory, and repository validation strategy;
  - acceptance criteria, risks, alternatives, open questions, intern checklist, and absolute file references.

### Why

The existing evidence identifies a phase, not an allocation owner. A design that immediately prescribes batches or GC would encode an unsupported assumption. The ticket should teach a new engineer how to convert phase-level evidence into allocation attribution and then into one controlled implementation experiment.

### What worked

- `docmgr status --summary-only` found 33 existing tickets and a working vocabulary.
- Ticket creation generated the expected index, tasks, changelog, README, design, and reference structure.
- All six requested task records were added.
- Existing public APIs already provide strong instrumentation anchors; no speculative measure API is required to begin profiling.
- The generated-fixture baseline already enforces:

  ```text
  run peak heap <64MiB
  run peak RSS <192MiB
  search_index peak heap <64MiB
  search_index peak RSS <192MiB
  ```

### What didn't work

`docmgr ticket list --ticket PV-MEM-002` returned exit status `0` with `No tickets found.` rather than using a nonzero not-found status. This was not a blocker, but continuation scripts must inspect output rather than assume the command's exit code proves existence.

No source implementation or memory experiment was attempted in this step. The user requested analysis/design first, and raw profile handling requires an explicit private harness and reproducible workload state.

### What I learned

- `ForEachSearchDocument` is streaming only at the application boundary. It avoids a full plaintext slice, but Bleve may still retain segment state across calls.
- `AllNotes` creates a pointer slice, but its size is too small to explain a 268 MB heap increase by itself.
- The mapping stores title, tags, and excerpt but not body. Any storage reduction experiment must preserve result hydration or deliberately move hydration to the matching vault snapshot.
- The old baseline identifies a dirty private worktree. A refreshed clean pinned baseline should precede strict before/after claims.
- Profile capture can perturb duration and expose content. Diagnostic runs and benchmark runs must be separate.

### What was tricky to build

The hardest part was designing a useful guide without prematurely approving one optimization. The document resolves this by defining evidence thresholds for six hypotheses. For example, explicit Bleve batching is considered only if profiles show corpus-proportional Bleve retention; file-cache action is considered only if smaps/cgroup file accounting dominates after heap falls.

The second difficulty was explaining the difference between preserving a full atomic snapshot and preserving every current implementation detail. Snapshot publication is a correctness invariant. Per-document indexing, tag flattening, batch size, mapping storage, and backend options are implementation choices that can change after equivalence proof.

### What warrants a second pair of eyes

- Review the suggested sub-400 MB RSS target against actual Kubernetes request/limit and rollout headroom requirements.
- Review whether profile checkpoints should be a ticket-local harness, a test hook, or a narrow production-disabled interface.
- Review Bleve's pinned-version backend and public options before designing any batch or merge tuning.
- Review search equivalence requirements, especially whether exact score/order equality is contractual across segment layouts.
- Review whether result hydration from the matching `Vault` could permit fewer stored fields without introducing lock or consistency problems.

### What should be done in the future

Begin Phase 0 by choosing a reproducible clean vault state and refreshing three persistent baseline runs. Then build the private profile/checkpoint harness and produce aggregate attribution before selecting the first optimization.

### Code review instructions

- Start with the design guide §§3–4 to verify current architecture and measurements.
- Review §§7–9 for profile security, hypotheses, and experimental controls.
- Review §11 decision records and §12 implementation phases before approving code work.
- Cross-check file references against `pkg/server/runtime.go`, `pkg/server/measurement.go`, `pkg/vault/vault.go`, and `pkg/search/search.go`.
- Re-run `docmgr doctor --ticket PV-MEM-002 --stale-after 30` after relations and changelog are finalized.

### Technical details

```text
Ticket: PV-MEM-002
Baseline persistent heap median: 391,219,024 bytes
Baseline persistent RSS median: 482,586,624 bytes
Preferred initial target: median RSS <400 MB
Required comparisons: >=3 baseline and >=3 candidate runs
Canonical evidence: content-free measure schema-v1 JSONL + receipts
Private evidence: raw heap profiles, never committed or uploaded
```

## Step 3: Validate the ticket and deliver the implementation handoff to reMarkable

After authoring and relating the guide, I validated the complete ticket structure and rendered the index, design guide, and diary as one reMarkable PDF with a depth-two table of contents. The required dry run resolved the exact three inputs and destination before Pandoc or cloud mutation. The real upload then completed successfully.

The bundle intentionally excludes raw baseline traces and any future heap profiles. The guide contains aggregate measurements and content-free contracts; raw profiles can contain private note text and are explicitly prohibited from ticket or device upload.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the new ticket and publish its intern-oriented documentation as a readable bundled PDF to the requested reMarkable destination.

**Inferred user intent:** Make the research and implementation plan available as a durable handoff outside the source checkout, while preserving ticket structure and privacy constraints.

**Commit (code):** N/A — documentation validation and delivery.

### What I did

- Related seven primary implementation/evidence files to the design guide and four report/source files to the diary, using absolute paths.
- Updated the ticket index with the baseline, target, primary guide, diary, and scope.
- Updated the ticket changelog with the vault-report commit and design milestone.
- Checked every ticket Markdown file for balanced fenced code blocks.
- Ran `git diff --check`.
- Ran `docmgr doctor --ticket PV-MEM-002 --stale-after 30`.
- Ran the required reMarkable bundle dry run.
- Uploaded index, design guide, and diary as `PV-MEM-002 Search Index Memory Guide.pdf`.

### Why

The ticket is intended for a new intern. Broken frontmatter, missing relations, malformed Markdown, or an unrenderable PDF would undermine that handoff. The bundle preserves the intended reading order: overview, complete guide, then chronological diary.

### What worked

Validation results:

```text
docmgr doctor: ✅ All checks passed
git diff --check: PASS
code-fence balance: PASS for all ticket Markdown
primary design: 1,231 lines, ~7,015 words
diary before this step: 256 lines, ~2,214 words
```

The dry run resolved:

```text
index.md
design-doc/01-search-index-memory-analysis-design-and-implementation-guide.md
reference/01-investigation-diary.md
```

The real upload reported:

```text
OK: uploaded PV-MEM-002 Search Index Memory Guide.pdf -> /ai/2026/08/25/PV-MEM-002
```

### What didn't work

Nothing failed during validation or upload. No authentication retry or forced overwrite was required.

### What I learned

The guide is large enough that bundling with a table of contents is materially easier to navigate than three separate PDFs. The strict content boundary also works for delivery: aggregate baseline tables are safe to publish, while raw heap profiles remain explicitly outside both Git and reMarkable.

### What was tricky to build

The subtle issue was selecting bundle content. The diary and design require historical baseline evidence to be useful, but bundling raw traces would add volume and require a separate privacy audit. The final bundle references the canonical baseline by path and includes its aggregate values without embedding private-workload event files.

### What warrants a second pair of eyes

- Visually inspect Mermaid layout and wide baseline/risk tables on the reMarkable display.
- Confirm the depth-two table of contents provides enough navigation for the 1,200-line guide.
- Review whether a future implementation bundle should include a separate compact experiment playbook after profiling begins.

### What should be done in the future

After Phase 1 profiling, add a content-free attribution report and optionally upload a new versioned bundle. Do not overwrite annotated reMarkable documents without explicit approval.

### Code review instructions

- Run `docmgr doctor --ticket PV-MEM-002 --stale-after 30`.
- Review the ticket index, primary guide, and diary in that order.
- Confirm the remote destination is `/ai/2026/08/25/PV-MEM-002`.

### Technical details

```text
Bundle: PV-MEM-002 Search Index Memory Guide.pdf
Destination: /ai/2026/08/25/PV-MEM-002
ToC depth: 2
Upload result: OK
Raw profiles included: no
Private note content included: no
```
