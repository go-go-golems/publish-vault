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
      Note: |-
        Primary search implementation evidence inspected during ticket design
        Opt-in byte-and-document bounded Bleve batch experiment
    - Path: repo://pkg/search/search_test.go
      Note: Batch validation flush progress and search-equivalence tests
    - Path: repo://pkg/server/runtime.go
      Note: |-
        Primary runtime evidence inspected during ticket design
        Production persistent batch policy and full-snapshot wiring
    - Path: repo://pkg/server/runtime_test.go
      Note: Reviewed-bound pin plus persistent lifecycle coverage
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/attribution/01-attribution-report.md
      Note: Phase 1 retained heap allocation churn RSS and lifetime conclusions
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/attribution/summary.json
      Note: Machine-readable aligned attribution and selected hypothesis
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/baseline-current/privacy-audit.json
      Note: Structural content/privacy audit across 4693 canonical events
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/baseline-current/summary.json
      Note: Pinned clean-worktree Phase 0 baseline summary
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/batch-matrix/01-batch-matrix-report.md
      Note: Seven-variant matrix and selected 16-document candidate
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/candidate-current/01-candidate-proof.md
      Note: Three-run before/after and finite-cgroup acceptance report
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/candidate-current/comparison.json
      Note: Machine-readable 34 percent heap and 22 percent RSS reduction
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/finite-cgroup/result.json
      Note: Isolated 1 GiB cgroup completion and peak evidence
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/artifacts/implementation/real-vault-search-equivalence.json
      Note: Twenty-query 16725-result equivalence proof
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/01-run-persistent-baseline.sh
      Note: Phase 0 reproducible persistent-index runner with cleanup and artifact checks
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/02-summarize-persistent-baseline.py
      Note: Phase 0 workload-consistency and three-run distribution reducer
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/03-summarize-attribution.py
      Note: Content-free checkpoint and pprof aggregate reducer
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/05-compare-baseline-candidate.py
      Note: Workload-identity guarded median and phase comparison
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/benchmark-search-index/main.go
      Note: Fresh-process search-only matrix harness
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/compare-real-search/main.go
      Note: Content-free complete-result real-vault equivalence harness
    - Path: repo://ttmp/2026/08/25/PV-MEM-002--reduce-publish-vault-search-index-peak-memory/scripts/profile-search-index/main.go
      Note: Private forced-GC checkpoint harness using measure collectors and exact search progress
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

## Step 4: Phase 0 — print the plan, freeze a clean workload, and refresh the baseline

Phase 0 established a new reproducible baseline before any runtime code changed. The master-plan and Phase 0 start slips were printed first. I then built one exact publish-vault binary and one exact measure binary, ran the clean personal vault three times through persistent indexing at 100 ms, summarized run and phase distributions, recorded index size, and audited every canonical event for private content.

The refreshed workload differs materially from MEASURE-001. The old run used a dirty vault worktree with 3,396 candidates and 20,938,723 source bytes. The current clean commit has 2,036 candidates but 76,921,819 source bytes. The current median peak is consequently much larger: 826,146,848 bytes heap and 1,033,994,240 bytes RSS. This is not evidence that the instrumentation commit regressed memory; it is a new workload. PV-MEM-002 comparisons must use this pinned clean baseline.

### Prompt Context

**User prompt (verbatim):** "Implement PV-MEM-002, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done. [REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory."

**Assistant interpretation:** Execute all six PV-MEM-002 phases with coherent commits, strict chronological evidence, and real thermal plan/start/completion slips; begin by pinning and validating a clean three-run persistent baseline.

**Inferred user intent:** Complete the optimization empirically rather than selecting an implementation from intuition, while leaving enough durable operational and review evidence to reproduce every decision.

**Commit (code):** Pending at the time of this diary entry; Phase 0 contains scripts and evidence only, with no runtime behavior change.

### What I did

- Printed and archived:
  - `scripts/slips/00-master-plan.yaml` with Phases 0–5;
  - `scripts/slips/01-phase-0-start.yaml` before baseline work.
- Inspected the uncommitted MEASURE-001 runner and summarizer rather than repeating its design from memory.
- Confirmed the personal vault is clean at commit `5f9f18ca7791ba2ddeb8a2528e3c279e6ae5f75a`.
- Recorded `.vault-ignore` hash `d39336e4...3521`; `.publish/config.yaml` is absent, so the application uses an empty default config.
- Added ticket-local scripts:
  - `scripts/01-run-persistent-baseline.sh`;
  - `scripts/02-summarize-persistent-baseline.py`.
- Added cleanup traps, exact trace/receipt cardinality checks, index-size capture, source capture, workload consistency checks, and phase distributions.
- Validated scripts with `bash -n` and `python3 -m py_compile`.
- Built exact binaries with `GOWORK=off` and `-trimpath`:
  - publish-vault SHA-256 `3d2fbb53...3966`;
  - measure SHA-256 `9d8ae4a0...bdac`.
- Ran three persistent-index server startups at 100 ms and waited for `/api/healthz` before graceful shutdown.
- Produced canonical traces, receipts, measure summaries, metadata, a distribution summary, artifact hashes, privacy audit, and human-readable baseline report.

### Why

The existing MEASURE-001 baseline cannot be used as a strict before/after baseline because its private workload was dirty and is no longer reproducible. A clean Git commit plus ignore/config hashes, note/byte counts, exact binary hashes, and repeated runs make future candidate comparisons defensible.

### What worked

All three runs completed and reported the same workload:

```text
Markdown candidates: 2,036
published notes: 2,030
candidate source bytes: 76,921,819
sample interval: 100 ms
```

Distributions:

```text
total duration:      130.27 / 166.42 / 172.93 s (min/median/max)
peak heap:           805,533,208 / 826,146,848 / 857,792,960 B
peak RSS:            958,763,008 / 1,033,994,240 / 1,051,410,432 B
index size:          204,434,855 / 210,012,540 / 211,134,662 B
search duration:     87.92 / 107.41 / 115.67 s
search peak heap:    identical to run peak in every run
search peak RSS:     identical to run peak in every run
```

The privacy audit inspected 4,693 canonical events and passed. Only `processed_notes`, `processed_bytes`, and `total_bytes` annotations exist. No path, slug, title, body, Markdown, excerpt, tags, command, environment, repository marker, or absolute home path was found.

### What didn't work

- The binary validation command tried `retro-obsidian-publish --version`; the root does not define that flag and returned:

  ```text
  Error: unknown flag: --version
  ```

  I did not add an unrelated compatibility flag. I validated the built binary through `help`; measure's existing `--version` returned `measure version dev`.

- The first privacy-audit glob accidentally counted seven `run-*.jsonl` files because it included three derived summary JSONL files and `run-console.jsonl`. The structural scan still passed, but the reported `trace_files` count was misleading. I replaced the glob with the explicit canonical paths `run-1.jsonl` through `run-3.jsonl` and reran the audit; it now reports three traces and 4,693 events.

### What I learned

- Source bytes are more predictive than note count for this workload. The clean vault has 40% fewer candidates than the old baseline but 3.7 times as many candidate bytes and roughly twice the memory peak.
- The current host cgroup is unlimited and shared. Its 7.18 GB median current peak is truthful collector output but is not attributable to publish-vault. Process RSS is the useful baseline here; later finite-cgroup Docker runs are required for isolated cgroup evidence.
- Persistent index bytes varied by roughly 3%, so index-size comparison also needs repeated runs.
- `search_index` remains unambiguously dominant for both heap and RSS.

### What was tricky to build

The central difficulty was avoiding a false regression claim. Comparing the old 483 MB median RSS to the new 1.034 GB median without workload identity would suggest a code regression. The source-byte count and clean commit show that these are different corpora. The ticket now explicitly supersedes the older baseline for optimization comparisons.

The second subtlety was retaining only content-free artifacts. Server logs and temporary search directories are destroyed after each run. Canonical events, receipts, summaries, counters, and hashes are retained because their key/value domains were structurally audited.

### What warrants a second pair of eyes

- Review whether the clean vault commit should remain frozen for the entire ticket even as go-go-parc receives unrelated commits. Recommendation: use a detached worktree at `5f9f18c` for all candidate runs.
- Review the 100 ms sampling cost before interpreting duration differences smaller than run-to-run variation.
- Review why persistent index byte size varies across equivalent builds; query correctness is the acceptance gate, but deterministic index size is not currently assumed.
- Confirm the shared unlimited cgroup caveat is prominent enough in all comparisons.

### What should be done in the future

Phase 1 must create a detached vault worktree at the pinned commit, capture private heap profiles aligned to fixed `search_index` progress checkpoints, capture smaps and runtime/cgroup aggregates, and retain only reviewed content-free profile tables.

### Code review instructions

- Review the runner's cleanup/error handling before the baseline values.
- Verify `summary.json` workload identity and all three receipt phase totals.
- Review `privacy-audit.json` and `artifact-manifest.json`.
- Re-run the summarizer against the retained artifacts; it should produce the same distributions.
- Do not compare this baseline directly to MEASURE-001 without the workload caveat.

### Technical details

```text
publish-vault commit: 8648cfcd1690c086010fbc5a64d27fe0f5ad6a9c
measure commit: a3f4b045b5d204101e17e35458de9b8955d71772
vault commit: 5f9f18ca7791ba2ddeb8a2528e3c279e6ae5f75a
baseline artifacts: artifacts/baseline-current/
canonical traces: 3
canonical events: 4,693
privacy audit: PASS
runtime code changes: none
```

## Step 5: Phase 1 — attribute reachable heap, allocation churn, arenas, and RSS

Phase 1 used a private diagnostic harness against a detached clean vault worktree. Five checkpoints aligned exact search progress with measure events, forced-GC heap profiles, runtime counters, procfs RSS/PSS/anonymous/private-clean values, and cgroup availability. Raw profiles remained mode `0600` under a mode-`0700` `/tmp` directory, were reduced to aggregate pprof tables, passed a content audit, and were deleted.

The evidence changes the optimization hypothesis. Search does not retain an additional 600 MB Go object graph. After forced GC, `HeapAlloc` rises only 33.7 MB from 0% to 100%. The unperturbed 826 MB median peak comes from 51.65 GB of allocation traffic, 611.9 MB of heap-arena expansion, and persistent-index residency. Scorch performs one backend batch/update per document and repeatedly builds and merges segments. Bounded multi-document Bleve batches are therefore the first controlled experiment.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Complete the attribution phase before modifying production code, preserve only content-free evidence, and select one falsifiable implementation hypothesis from measured allocation and lifetime data.

**Inferred user intent:** Ensure the eventual optimization removes a proven source of memory pressure rather than hiding peaks with GC or capacity changes.

**Commit (code):** Pending at the time of this diary entry; Phase 1 adds only a private diagnostic harness and content-free evidence.

### What I did

- Printed and archived `scripts/slips/03-phase-1-start.yaml` before profiling.
- Created a detached, clean vault worktree at `5f9f18c`.
- Added `scripts/profile-search-index/main.go`, which:
  - loads the real vault;
  - starts a schema-v1 measure run and `search_index` phase;
  - captures exact 0/25/50/75/100% progress;
  - forces GC only in this explicitly perturbed diagnostic;
  - writes raw profiles privately;
  - reads runtime/procfs/smaps/cgroup state through `measure/pkg/collector`;
  - writes content-free checkpoints, trace, and receipt.
- Built and ran the harness against all 2,030 published notes.
- Generated in-use-space and alloc-space top tables with `go tool pprof` for all five checkpoints.
- Added `scripts/03-summarize-attribution.py` and machine-readable `summary.json`.
- Wrote `artifacts/attribution/01-attribution-report.md` with method, limitations, aligned table, retained/cumulative analysis, representation lifetime inventory, hypothesis decisions, and Phase 2 matrix.
- Audited aggregate artifacts for raw profiles, private paths, and content-bearing event fields.
- Deleted the raw profiles and temporary index after accepting the aggregates.

### Why

The phase-level baseline proves where the peak occurs but cannot distinguish reachable objects, garbage between collections, runtime arenas, and file-backed residency. Heap profiles after a controlled GC establish reachable ownership; cumulative allocation profiles identify churn; smaps and runtime values explain why RSS remains high after live heap falls.

### What worked

Checkpoint deltas from 0% to 100%:

```text
post-GC retained HeapAlloc:       +33,694,240 bytes
HeapSys:                          +611,876,864 bytes
RSS:                              +300,216,320 bytes
cumulative allocation:         +51,652,634,904 bytes
```

At 0%, before search:

```text
profile total: 209.4 MB
regexp.ReplaceAllStringFunc flat: 196.7 MB (93.9%)
```

The retained strings belong to vault parse/render paths and represent snapshot HTML. At 100%, the profile total is 227.4 MB and the same 196.7 MB remains. Active Bleve retained structures are comparatively small.

At 100%, cumulative allocation evidence includes:

```text
bytes.growSlice flat:                   26.47 GB
Scorch.Batch cumulative:                29.80 GB
Scorch planMergeAtSnapshot cumulative:  15.61 GB
zapx mergeToWriter cumulative:           15.34 GB
parser.stripMarkdown cumulative:          5.46 GB
```

Cumulative call paths overlap and are not added together. They show that one-document Scorch updates and repeated segment merges are the largest controllable path.

The attribution privacy audit passed: five checkpoint files, 18 diagnostic events, no raw `.pprof`, and no content-bearing event fields. Raw profiles were then deleted.

### What didn't work

No harness or profile command failed. One pprof source listing attributed the inlined `replaceUnresolvedNoteEmbeds` allocation to a nearby line after inlining rather than the exact source expression. I used cumulative call paths and function identity, not the displayed neighboring line, for the conclusion.

The diagnostic cgroup remained an unlimited shared user scope. Its values are retained as availability evidence but excluded from process attribution. A finite isolated container run remains required during candidate proof.

### What I learned

- The high natural `HeapAlloc` peak is transient allocation pressure, not a permanently reachable search graph.
- Forcing GC lowers live heap but leaves `HeapSys` around 1.014 GB and substantial RSS; forced GC is a diagnostic, not the selected fix.
- Rendered HTML contributes approximately 196.7 MB of required steady snapshot heap before search begins. It is not caused by Bleve and is not the first target of this ticket.
- Search adds only about 34 MB retained heap after GC but allocates roughly 51.65 GB over the phase.
- `Index.Index` maps one document and calls Scorch `Update`, which implements a one-document backend batch. Repeating that 2,030 times creates avoidable segment/merge traffic.
- Plain-text extraction is also allocation-heavy, but its 5.46 GB cumulative path is smaller than Scorch batch/merge work and is deferred until batching is measured.

### What was tricky to build

Heap and RSS numbers describe different lifetimes. At 100%, `HeapAlloc` is 245 MB after GC, `HeapSys` is 1.014 GB, RSS is 712 MB, anonymous RSS is 516 MB, and private-clean RSS is 195 MB. A statement such as “the process retains 712 MB of Go objects” would be false. The attribution report keeps these categories separate.

The second challenge was security. Heap profiles can include note text even when the summary does not. The workflow therefore permits raw profiles only in a private temporary directory and commits only aggregate function names and byte counts after scanning.

### What warrants a second pair of eyes

- Review whether `regexp.ReplaceAllStringFunc` output is indeed required final HTML in both parse and rebuild paths; it is large steady state but outside the selected search optimization.
- Review the interpretation of Scorch cumulative paths against Bleve v2.6.0 source.
- Review batch limits by both document count and estimated bytes; document-only limits are not content bounds.
- Review whether search-result scores/order can vary with segment layout and define the equivalence contract before accepting batches.

### What should be done in the future

Phase 2 should implement an experimental bounded batch path with explicit document and estimated-byte ceilings, compare several limits against the current one-document path, and select the best trade-off using generated and real workloads before changing the default.

### Code review instructions

- Begin with `artifacts/attribution/01-attribution-report.md` and `summary.json`.
- Inspect `scripts/profile-search-index/main.go` for private-path modes, forced-GC disclosure, exact thresholds, and measure collector reuse.
- Confirm no `.pprof` exists anywhere under the ticket.
- Compare 0% and 100% in-use tables; do not use alloc-space as retained heap.
- Cross-check `Index.Index -> Scorch.Update -> Scorch.Batch` against Bleve v2.6.0.

### Technical details

```text
diagnostic vault: detached clean 5f9f18c
checkpoints: 0, 25, 50, 75, 100 percent
forced collections: 5 (diagnostic only)
raw profile retention: none
aggregate artifacts: artifacts/attribution/
selected Phase 2 hypothesis: bounded Bleve batches
production behavior changes: none
```

## Step 6: Phase 2 — test bounded Bleve batches and select the 16-document candidate

Phase 2 added an opt-in experimental batch path while leaving the zero-value production behavior unchanged. Seven fresh-process variants measured the same detached 2,030-note workload. Every explicit batch reduced cumulative allocation and index size. A 16-document, 1 MiB estimated-field bound produced the best observed heap/RSS combination and also reduced duration, allocations, GC cycles, and index size.

The experiment confirms the Phase 1 hypothesis. Larger batches reduce total allocation further but retain more mapped/analyzed documents at each commit; the fastest 128-document variant remained close to the current memory peak. Batch 16 is selected for production wiring and three-run proof, not because one run is sufficient evidence, but because it is the strongest candidate from the controlled matrix.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Test the evidence-backed batching hypothesis across explicit document and byte bounds, preserve current behavior by default, and select a candidate only after memory, allocation, time, index-size, and correctness evidence agree.

**Inferred user intent:** Avoid replacing one unbounded behavior with an arbitrary tuning knob; establish an explicit bounded algorithm and a reviewed default.

**Commit (code):** Pending at the time of this diary entry.

### What I did

- Printed and archived:
  - `scripts/slips/04-phase-1-done.yaml` after attribution commit `ceddf95`;
  - `scripts/slips/05-phase-2-start.yaml` before batch implementation.
- Extended `search.Options` with opt-in `BatchDocuments` and `BatchBytes` bounds.
- Required both bounds to be zero or both positive; validation happens before a persistent target directory is removed.
- Kept the existing one-document path for zero values.
- Added `indexVaultBatched`, which:
  - maps documents into a Bleve batch;
  - flushes before a next document would cross the estimated byte bound;
  - flushes at the document bound;
  - commits an indivisible oversized document alone;
  - reports progress only after successful batch commit;
  - flushes the final partial batch.
- Replaced manual tag concatenation with `strings.Join`, preserving the stored field value.
- Added focused tests for validation, target preservation, progress, partial/oversized flush, and baseline/batched search equivalence.
- Added `scripts/benchmark-search-index/main.go` to measure only search construction in a fresh process.
- Ran current, 4, 8, 16, 32, 64, and 128-document variants with proportionate byte bounds.
- Added the batch matrix reducer, report, summary JSON, traces, metadata, receipts, and privacy audit.

### Why

Bleve v2.6.0's `Index` maps one document and calls backend `Update`; Scorch implements that update through a one-document backend batch. Repeating this for 2,030 documents creates many small segments and repeated merges. Explicit batches let Scorch construct fewer segments while the document and byte ceilings bound application-held batch content.

### What worked

The selected batch 16 / 1 MiB result versus the current exploratory run:

```text
peak heap:       869,712,904 -> 521,084,376 B  (-40.09%)
peak RSS:      1,068,277,760 -> 766,619,648 B  (-28.24%)
duration:             89.73 -> 61.78 s         (-31.15%)
allocation:      50,352,073,512 -> 19,273,219,336 B (-61.72%)
GC cycles:             192 -> 83
index size:      233,101,871 -> 199,028,353 B  (-14.62%)
```

All seven variants processed exactly 2,030 documents and 72,070,169 indexed field bytes. The matrix privacy audit inspected 4,784 events across seven variants and found no content-bearing fields.

Focused unit, race, vet, and gosec checks passed after fixes.

### What didn't work

- The initial baseline/batched equivalence test compared result array positions. The query `memory` produced two equal-score results in opposite order:

  ```text
  Search("memory")[0] got slug "one", want "two"
  Search("memory")[1] got slug "two", want "one"
  ```

  The existing API specifies score ranking but no secondary key for exact ties. I changed equivalence to compare result sets by slug, stored fields, and score tolerance. Non-tied score semantics remain checked.

- The first matrix shell completed the current run but its print helper expected lowercase `phases`; Go's direct `report.Summary` JSON uses `Phases`. It failed with:

  ```text
  KeyError: 'phases'
  ```

  The measurement artifacts were valid. I inspected `metadata.json`, corrected the helper to the actual schema, cleaned the temporary index, and continued the remaining variants.

- Gosec rejected two validated `int -> uint64` conversions for document limits as potential G115 overflow. Rather than suppressing warnings, I changed `BatchDocuments` and the harness flag to `uint64`, removing an impossible negative state and the conversions. Gosec then reported zero issues.

### What I learned

- Batch size creates a real throughput/memory curve, not a monotonic improvement. Batch 128 was fastest and allocated least overall but peaked at 801 MB heap and 980 MB RSS.
- Batch 16 gave the lowest observed heap and RSS. Batch 32 was faster but used 64 MB more heap and 77 MB more RSS.
- Batch 64's single-run duration exceeded current despite lower allocations, showing merge scheduling and system variance. Final proof needs repeated runs.
- Explicit batches reduce on-disk index size by approximately 15%, likely because they produce fewer initial segments and a different merge shape.
- Progress must mean committed documents. Reporting staged documents would overstate successful work on batch failure.

### What was tricky to build

The byte limit is based on the search document's slug, title, body, excerpt, and tag bytes before Bleve mapping. Bleve's analyzed representation can be larger. The limit is therefore an explicit input-size bound, not a promise about backend allocation. Pairing it with a document limit prevents many tiny documents from accumulating without bound.

Search equivalence also needed a precise tie rule. Segment layout can change tie ordering without changing scores or result content. The ticket now treats equal-score ordering as unspecified while preserving all result identities, stored fields, and score values.

### What warrants a second pair of eyes

- Review whether 16 documents / 1 MiB should remain internal constants rather than operator flags. Recommendation: internal defaults until a second workload needs tuning.
- Review whether estimated fields should include tag separator bytes; the omission is at most `len(tags)-1` bytes per document and does not alter the safety model materially, but could be included for exact accounting.
- Review lock scope: batched construction holds the wrapper mutex for the build, but the index is unpublished and single-owner during construction.
- Review whether progress callbacks should expose batch commit count; current numeric progress is sufficient and bounded.

### What should be done in the future

Phase 3 should wire 16 documents and 1 MiB into persistent full-snapshot construction as named internal constants, preserve in-memory behavior, add real-vault query equivalence, and validate every persistent publication/reload/failure path before repeated candidate measurement.

### Code review instructions

- Review `Options.validate`, `indexVaultBatched`, and flush ordering in `pkg/search/search.go`.
- Review tests for invalid target preservation, oversized documents, final partial flush, and tie-aware equivalence.
- Review `artifacts/batch-matrix/01-batch-matrix-report.md` and `summary.json`.
- Treat all matrix values as exploratory single runs; use Phase 4 for acceptance medians.
- Re-run focused tests, race, vet, and gosec.

### Technical details

```text
variants: current, batch 4, 8, 16, 32, 64, 128
selected: 16 documents / 1 MiB estimated fields
current default behavior: unchanged (zero/zero)
selected candidate heap reduction: 40.09%
selected candidate RSS reduction: 28.24%
selected candidate allocation reduction: 61.72%
content audit: PASS
```

## Step 7: Phase 3 — wire reviewed persistent defaults and prove runtime correctness

Phase 3 applied the reviewed 16-document and 1 MiB bounds to persistent full-snapshot construction only. In-memory indexes and incremental updates remain unchanged. The limits are named internal server constants with an evidence comment and a pinning test; they are not exposed as unsupported operator tuning flags.

Correctness was tested at three levels: focused batch semantics, existing persistent snapshot/reload/failure tests, and a real-vault current-versus-candidate query corpus. The real-vault harness compared 20 queries and 16,725 results without retaining result content. After correcting a tie-at-limit issue in the proof method, all identities, stored fields, and scores matched.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Turn the selected experimental bounds into the production persistent build path, document the behavior, and pass search, snapshot, rollback, cleanup, race, lint, security, frontend, and cross-build gates before measuring the candidate repeatedly.

**Inferred user intent:** Ensure the memory improvement does not trade away search results, persistent index freshness, or atomic reload safety.

**Commit (code):** Pending at the time of this diary entry.

### What I did

- Printed and archived:
  - `scripts/slips/06-phase-2-done.yaml` after experiment commit `c8450fa`;
  - `scripts/slips/07-phase-3-start.yaml` before production wiring.
- Added server-owned constants:

  ```text
  persistentSearchBatchDocuments = 16
  persistentSearchBatchBytes = 1 << 20
  ```

- Added `persistentSearchOptions` and wired it only into `buildSearchIndex`'s persistent constructor.
- Added a server test pinning reviewed values and runtime wiring.
- Added a batched source-read failure test proving the failed constructor closes its index and releases the target for removal.
- Documented batching, progress semantics, oversized documents, and unchanged in-memory/incremental behavior in README.
- Added `scripts/compare-real-search/main.go`, which emits only query hashes, counts, score deltas, and pass state.
- Built current and candidate indexes from the detached vault and compared exact, multi-word, fuzzy, prefix, `#tag`, and `tag:` queries.
- Wrote `artifacts/implementation/01-implementation-validation.md` and the content-free equivalence JSON.
- Ran generation, local CI, full race tests, lint, security, frontend typecheck/build, Darwin cross-build, and diff checks.

### Why

The search package owns generic batching mechanics; the server owns the workload-derived production policy. Keeping constants in `pkg/server/runtime.go` prevents a measurement from one application workload from becoming an undocumented universal Bleve default.

### What worked

Final real-vault equivalence:

```text
queries: 20
total results compared: 16,725
identities and stored fields: equal
maximum score difference: 3.469446951953614e-18
passed: true
```

Fresh validation:

```text
go generate ./...                         PASS
make ci-check                             PASS
go test -race ./... -count=1              PASS
golangci-lint and Glazed lint             PASS
gosec                                     0 issues
govulncheck                               0 called vulnerabilities
pnpm --dir web check                      PASS
pnpm --dir web build                      PASS
Darwin arm64 cross-build                  PASS
git diff --check                          PASS
```

Existing tests continue to cover fresh persistent rebuild, deleted-note removal, final-path reopen, failed-load rollback, reload serialization, unchanged-symlink skips, and delayed old-index cleanup.

### What didn't work

The first real-vault equivalence run failed four broad tag queries. Each requested only 50 results; all matching tag results had equal scores, so different segment layouts selected different top-50 tie subsets. The content-free report showed zero score difference but unequal identities for:

```text
#project
#article
tag:project
tag:article
```

This was a proof-harness limit problem, not evidence of different complete match sets. I raised the comparison limit to 10,000, above the entire 2,030-note corpus, rebuilt both indexes, and compared complete result sets. All 20 queries passed. The API still does not promise a secondary key for equal-score truncation; adding one would be a separate behavior change.

The frontend build emitted its existing chunk-size warning for several Mermaid/MathJax assets. No frontend files or chunk behavior changed in this phase.

### What I learned

- Batch segment layout can change which equal-score documents fall at an arbitrary limit without changing matches or scores. Equivalence tests must distinguish rank semantics from unspecified tie ordering.
- Production policy belongs at the consumer integration boundary; generic `search.Options` supports experiments without imposing one workload's constants on all callers.
- Progress now advances in committed batches, which is both less noisy and more truthful on errors.
- The selected implementation reduces index segments and size without changing analyzers, mapping, stored fields, query construction, or result hydration.

### What was tricky to build

The main review boundary is ownership. Holding `search.Index.mu` through batch construction is safe because the index is unpublished and single-owner during construction. The callback must not call back into the same index. Once opened at its final path and placed in a snapshot, query locking behavior remains unchanged.

The equivalence artifact had to prove real behavior without publishing private slugs or titles. It stores only SHA-256 query identifiers, counts, maximum score differences, bounds, and pass/fail state.

### What warrants a second pair of eyes

- Review the complete-result equivalence rule and whether product behavior should eventually define deterministic tie-breaking.
- Review the server/search ownership split and lock scope.
- Review whether the runtime test's pinned literal values is appropriately strict; changing them should require new PV-MEM evidence.
- Review generated-fixture budgets after repeated candidate runs; do not tighten them from the one-run matrix.

### What should be done in the future

Phase 4 must build the committed candidate binary and run three complete server startups against the pinned vault, then compare median/range heap, RSS, duration, and index bytes with Phase 0. It must also run under an isolated finite cgroup so cgroup current/limit are attributable.

### Code review instructions

- Start with `pkg/server/runtime.go` constants and `persistentSearchOptions`.
- Review `pkg/search/search.go` batching and `pkg/search/search_test.go` error/equivalence coverage.
- Review the content-free `real-vault-search-equivalence.json` and implementation validation report.
- Run `make ci-check` and full race tests.
- Confirm in-memory `search.NewWithOptions` calls still pass zero batching options in production.

### Technical details

```text
persistent production bound: 16 documents / 1 MiB estimated fields
real-vault equivalence queries: 20
real-vault results compared: 16,725
max score delta: 3.469446951953614e-18
raw result content retained: none
full local CI: PASS
```

## Step 8: Phase 4 — prove repeated improvement and finite-cgroup completion

Phase 4 rebuilt the committed candidate once, ran three complete server startups against the exact detached vault commit, and compared medians/ranges to the three Phase 0 runs. The candidate reduced median heap by 34.27%, median RSS by 21.52%, and complete-load duration by 48.13%. Candidate ranges are separated from baseline ranges, and `search_index` remains the dominant phase rather than shifting pressure into publication or snapshot swap.

The same binary then completed inside a network-isolated 1 GiB Docker cgroup. The app detected the finite limit, set its 85%-headroom `GOMEMLIMIT`, built all 2,030 documents, and emitted finite cgroup measurements. Peak cgroup current reached 988 MB, proving both successful constrained execution and that 1 GiB leaves little spare headroom for this private workload.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Prove the accepted implementation with repeated full-lifecycle runs, content/privacy audits, phase-shape checks, and an isolated finite-cgroup execution before changing budgets or declaring success.

**Inferred user intent:** Replace exploratory one-run claims with repeatable operational evidence that includes container accounting, not only Go heap measurements.

**Commit (code):** `34ec4b9` — `perf(PV-MEM-002): enable bounded persistent indexing`

### What I did

- Printed and archived:
  - `scripts/slips/08-phase-3-done.yaml` after implementation commit `34ec4b9`;
  - `scripts/slips/09-phase-4-start.yaml` before repeated measurement.
- Built one exact committed candidate binary with `GOWORK=off`, `-trimpath`, SHA-256 `212c6724...1b1e`.
- Reused the exact measure binary from Phase 0, SHA-256 `9d8ae4a...bdac`.
- Ran three complete persistent server startups at 100 ms against detached clean vault `5f9f18c`.
- Summarized candidate distributions and compared them to Phase 0 with `scripts/05-compare-baseline-candidate.py`.
- Added workload identity checks before comparison.
- Verified `search_index` remains the largest candidate heap/RSS phase.
- Ran the candidate in `debian:bookworm-slim` with `--memory=1g --memory-swap=1g --network=none` and a read-only vault mount.
- Captured finite cgroup limit/current, derived Go soft limit, heap, RSS, duration, progress, and index bytes.
- Audited three candidate traces and the finite-cgroup trace for private/content-bearing fields.
- Removed server logs and temporary indexes.
- Wrote the repeated candidate report and machine-readable comparison.

### Why

The Phase 2 matrix measured only search construction and one run per variant. Full server runs include vault parsing, rendering, persistent publication, and snapshot swap. Three runs establish whether the reduction survives normal variation and whether another phase becomes dominant. Docker provides an isolated finite cgroup, unlike the shared unlimited host user scope.

### What worked

Median before/after:

```text
peak heap: 826,146,848 -> 543,066,520 B  (-34.27%)
peak RSS:  1,033,994,240 -> 811,429,888 B (-21.52%)
duration:  166.42 -> 86.32 s                (-48.13%)
throughput: 12.20 -> 23.52 notes/s           (+92.80%)
index size: 210,012,540 -> 205,077,678 B      (-2.35%)
```

Candidate range:

```text
heap: 534,493,424 .. 551,111,344 B
RSS:  792,576,000 .. 829,358,080 B
duration: 75.69 .. 95.12 s
```

Finite-cgroup run:

```text
hard limit: 1,073,741,824 B
Go soft limit: 912,680,550 B
peak heap: 623,229,576 B
peak RSS: 803,254,272 B
peak cgroup current: 988,028,928 B
duration: 81.92 s
search documents: 2030 / 2030
result: succeeded
```

Candidate privacy audit inspected 2,687 events; finite-cgroup audit inspected 841. Both passed.

### What didn't work

The first Docker run completed successfully, but the container created `/out/traces` as root with mode `0700`. Host-side `find` failed:

```text
find: .../artifacts/finite-cgroup/traces: Permission denied
```

The container had already exited cleanly and artifacts were intact. I ran a one-shot Alpine container against the same output mount to change ownership to UID/GID 1000, then decoded and audited the trace. Future scripts should set output ownership before container exit. No `sudo`, mode broadening, or artifact loss was required.

The Debian image was not local and was pulled by immutable digest resolution before the run. The artifact report records behavior, not an assumption that the image was already cached.

### What I learned

- The complete-lifecycle improvement is smaller than the search-only matrix because required vault/render state contributes to the process peak, but the reduction remains large and non-overlapping.
- The selected batch nearly doubles end-to-end throughput, so memory reduction did not trade away reload speed.
- Cgroup current peaks roughly 185 MB above process RSS in the finite container, demonstrating why RSS alone is insufficient for resource-limit guidance.
- A 1 GiB limit succeeds but reaches 92% sampled utilization. It is a proof boundary, not a recommended comfortable production limit.
- Sub-400 MB RSS is not achievable by batch tuning alone; Phase 1 found about 197 MB retained rendered HTML before indexing plus runtime/file-backed residency. That would require a separate lazy-render/cache design.

### What was tricky to build

The baseline host cgroup values were shared and could not be compared to candidate host cgroup values. The comparison script deliberately compares heap, process RSS, duration, throughput, and index size while the finite-cgroup result is reported separately. Mixing the shared host cgroup's 7–9 GB current usage into application deltas would be false attribution.

The candidate report also distinguishes measured range separation from statistical inference. Three runs are enough for a robust engineering median/range comparison here, but not a general performance distribution model.

### What warrants a second pair of eyes

- Review finite-cgroup headroom and production resource recommendations; do not lower below 1 GiB from this result.
- Review whether the 1 GiB proof should be repeated after Docker image build rather than mounted binary; Phase 5 container smoke covers packaging.
- Review the conclusion that sub-400 MB requires a new rendering/lifetime ticket rather than further batch tuning.
- Review baseline/candidate workload identity checks and binary hashes.

### What should be done in the future

Phase 5 should update generated-fixture budgets from fresh repeated fixture runs, run final Docker/Compose smoke and all repository gates, push the branch/PR, obtain CI/review evidence, complete ticket bookkeeping, and perform the requirement-by-requirement completion audit.

### Code review instructions

- Read `artifacts/candidate-current/01-candidate-proof.md` and `comparison.json`.
- Compare all three baseline and candidate receipts, not only medians.
- Inspect finite-cgroup `result.json`, receipt, and privacy audit.
- Verify binary hashes in candidate manifest.
- Confirm no server log, search directory, raw profile, or note content is retained.

### Technical details

```text
baseline runs: 3
candidate runs: 3
sample interval: 100 ms
candidate commit: 34ec4b9
median heap reduction: 283,080,328 B
median RSS reduction: 222,564,352 B
finite cgroup: 1 GiB, succeeded
candidate content audit: PASS
```
