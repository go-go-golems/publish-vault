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

## Step 2: Phase 1 — module rename

Renamed the module to `github.com/go-go-golems/publish-vault` and swept every
reference. The sweep was smaller than feared because Dockerfile/CI reference
filesystem paths, not the module path.

### Prompt Context

**User prompt (verbatim):** (see Step 1 — "Then, work on Step 1. Keep a detailed diary as you work, and commit at appropriate intervals.")

**Assistant interpretation:** Implement the design doc phases, committing per phase.

**Inferred user intent:** Land the framework-ification so golem-docs can import publish-vault.

**Commit (code):** 34b2e5d — "refactor: rename module to github.com/go-go-golems/publish-vault (PV-FRAMEWORK-017 phase 1)"

### What I did
- `go.mod` module line; sed over 24 Go files rewriting `"retro-obsidian-publish` imports.
- Makefile `LOGCOPTER_FLAGS -strip-prefix` + `logcopter_generate.go` directive; `make logcopter-generate`.
- README heading -> "Publish Vault"; `web/package.json` name -> `publish-vault-web`.
- Verified: build, full test suite, `make lint` (0 issues), `make logcopter-check`.

### Why
- Module path must equal the repo URL for `go get` resolution (design D1).

### What worked
- Generated logcopter files came out byte-identical in area terms because
  `-area-prefix go-go-golems.publish-vault` already used the target name.

### What didn't work
- N/A — mechanical phase, no failures.

### What I learned
- `bump-go-go-golems`'s awk only scans require lines, so the module's own new
  go-go-golems-prefixed name does not confuse it.

### What was tricky to build
- Nothing; the design doc's §3.5 reference inventory made this a checklist.

### What warrants a second pair of eyes
- grep for `"retro-obsidian-publish` in Go files must stay empty (verified 0).

### What should be done in the future
- N/A

### Code review instructions
- `git show 34b2e5d --stat`; spot-check an import in pkg and cmd.

### Technical details
- 29 files changed, pure rename mechanics.

## Step 3: Phase 2 — internal/ -> pkg/ promotion

Moved the nine framework packages to `pkg/` with `git mv` (history-preserving),
keeping `ignore` and `parser` internal per design D2. Updated every path
reference (Dockerfile embed COPY, .gitignore/.dockerignore, Makefile clean and
LOGCOPTER_PACKAGES, the build-web verb's output path) and regenerated logcopter
(areas changed `internal.X` -> `pkg.X`).

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** (phase 2 commit; `git log --oneline` on task/framework-ification)

### What I did
- `git mv internal/{server,vault,search,api,watcher,web,widgethost,vaultdata,vaultwidgets} pkg/`.
- Per-package sed rewriting `publish-vault/internal/<p>"` -> `publish-vault/pkg/<p>"`.
- Path updates listed above; `make logcopter-generate`.
- Verified: build; all tests; per-package `go doc ./pkg/<p> | grep internal/` empty
  (no exported signature names an internal type); `build web --local` then
  `go build -tags embed` produced a binary that served SPA (HTTP 200) and
  `/api/config` from vault-example; lint, gosec (0 issues), logcopter-check.

### Why
- Go forbids importing another module's internal/; pkg/ is the public surface.

### What worked
- A combined-alternation sed with `|` as both delimiter and alternation failed
  (`unknown option to 's'`); the per-package loop with `#` delimiters is the
  reliable form.
- The internal-leak check came back clean on the first pass, confirming the
  design-doc prediction that vault defines its own public types.

### What didn't work
- The first sed attempt (see above) — syntax error, no harm done.

### What I learned
- Only the tracked `.gitkeep` lives under embed/public in git; `git add -A`
  after the move stages exactly the rename, never build outputs (ignore rules
  moved with the directory).

### What was tricky to build
- Sequencing: regenerating logcopter BEFORE the import rewrite would have
  produced files with stale area names; the safe order is move -> rewrite
  imports -> update LOGCOPTER_PACKAGES -> regenerate -> build.

### What warrants a second pair of eyes
- Dockerfile embed COPY path (pkg/web/embed/public) — a typo here only fails in
  the Docker/CI image build, not locally. CI's test-build job covers it.

### What should be done in the future
- N/A

### Code review instructions
- Phase-2 commit; verify with `GOWORK=off go build ./...`, `make logcopter-check`,
  and the CI test-build job (Docker image build).

## Step 4: Phases 3-5 — downstream proof, WebFS override, release flow, README

Proved the framework story with an external scratch module, added the
`Config.WebFS` escape hatch (design D4's second delivery mode), wrote the
release-assets workflow, and documented library usage in the README.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** (phases 3-5 commit on task/framework-ification)

### What I did
- Scratch module `pvconsumer` (scratchpad) with
  `replace github.com/go-go-golems/publish-vault => <checkout>`: a 25-line main
  importing pkg/server. Built plain and with `-tags embed`; the embedded binary
  served the SPA (200) and `/api/config` reporting the caller's VaultName, and
  shut down cleanly on context cancel.
- Captured the missing-assets failure mode by emptying pkg/web/embed/public:
  `pattern embed/public: cannot embed directory embed/public: contains no
  embeddable files` — recorded verbatim in the README so downstream developers
  recognize it.
- `pkg/web.NewSPAHandlerFS` (the internal newSPAHandler was already
  fs-parametrized — 5-line export) + `server.Config.WebFS` wiring.
- `.github/workflows/release-assets.yml`: workflow_dispatch (bump choice),
  builds SPA, commits into pkg/web/embed/public, svu-tags that commit, pushes
  only the tag. Reuses the svu-output validation pattern from PR #12 review.
- README "Using publish-vault as a library" section.

### Why
- The proof catches API-surface problems no in-repo test can (a downstream
  module has no access to internal/).

### What worked
- The consumer built and ran on the first attempt after one signature fix.

### What didn't work
- First consumer build failed: `cannot use 8098 (untyped int constant) as
  string value` — `server.Config.Port` is a string. Recorded as an API wart
  (open question for a future breaking pass: int port + validation, or keep
  string for :port flexibility). Not changed now to avoid churning the CLI.

### What I learned
- goja/embed subtlety: `.gitkeep` does not satisfy go:embed (dot-files are
  excluded by embed patterns), so an "empty but tracked" embed dir still fails
  the tagged build — the release workflow MUST commit real assets.

### What was tricky to build
- The release flow's core trick: the assets commit is created on a detached
  position and only the tag ref is pushed, so `main` never contains built
  assets but `go get @tag` gets a self-contained module zip. This is the whole
  resolution of design decision D4.

### What warrants a second pair of eyes
- release-assets.yml has not been exercised yet (needs a manual dispatch);
  first real run should be watched. The `git push origin "$tag"` only pushes
  the tag ref — confirm no branch push occurs.

### What should be done in the future
- Dispatch release-assets to mint v0.1.0 once this branch merges; then flip
  golem-docs from `replace` to the tag.
- Consider `Port int` in a future v0.x breaking pass.

### Code review instructions
- Start: pkg/server/server.go (Config.WebFS + ServeWeb block),
  pkg/web/static.go (NewSPAHandlerFS), release-assets.yml.
- Validate: `GOWORK=off go build ./... && make test lint logcopter-check`; the
  scratch-consumer procedure is reproducible from design doc §6 Phase 3.

### Technical details
- Consumer module and binaries live in the session scratchpad (pv-consumer/).

## Step 5: PR #13 and the predicted Docker break

Opened PR #13. CI failed exactly the way design doc §6 Phase 2 warned about:
"compiles locally, breaks in Docker". The Dockerfile copies source directories
individually (`COPY cmd`, `COPY internal`) and the new `pkg/` was never copied,
so the image build could not resolve `publish-vault/pkg/*`. One-line fix
(`COPY pkg ./pkg`, commit 8d6d02f); all checks green after.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** 8d6d02f — "fix(docker): copy pkg/ into the builder stage"

### What I did
- `gh run view --log-failed` on test-build: `no required module provides package
  github.com/go-go-golems/publish-vault/pkg/server` at Dockerfile:23.
- Added `COPY pkg ./pkg`; both failing jobs (test-build, release/publish) build
  the same image and went green together.

### Why
- Local builds see the working tree; the Docker builder stage sees only what is
  COPYed. Directory-selective COPY is the one place the promotion was invisible
  to local verification.

### What worked
- The Phase-2 embed-path edits themselves were correct; only the source COPY was
  missing.

### What didn't work
- My local verification matrix (build/test/lint/gosec/embed binary) cannot catch
  Dockerfile COPY gaps; only CI's test-build does. Accepted residual risk in the
  design doc, now realized and fixed.

### What I learned
- When adding a top-level source directory to a repo with selective-COPY
  Dockerfiles, grep the Dockerfile for COPY lines as part of the move checklist.

### What warrants a second pair of eyes
- Nothing further; CI green across all 10 checks.

### What should be done in the future
- After merge: dispatch release-assets (patch) for v0.1.0; watch the first run
  (tag-only push assertion).

### Code review instructions
- PR https://github.com/go-go-golems/publish-vault/pull/13; commits 34b2e5d,
  ce9d3e3, 130e767, 8d6d02f.
