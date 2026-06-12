---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-template/.github/workflows/push.yml
      Note: Template CI flow for logcopter
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-template/.golangci.yml
      Note: Baseline golangci-lint v2 configuration
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-template/Makefile
      Note: Baseline Go repository quality and release targets
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml
      Note: Reusable GHCR image publishing and GitOps PR workflow
    - Path: .github/workflows/ci.yml
      Note: publish-vault application CI for Go
    - Path: Makefile
      Note: publish-vault local build
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Org Style Alignment Analysis and Implementation Guide

## Executive summary

This ticket aligns `publish-vault`, `go-template`, and the shared `infra-tooling` workflows with the current Go Go Golems repository style. The practical goal is consistency: new repositories generated from `go-template`, deployable applications such as `publish-vault`, and reusable workflows in `infra-tooling` should use the same tool versions, linting conventions, CI shape, release publishing conventions, and documentation publishing patterns.

The most important observation is that the organization already has a modern style, but it is distributed across several places:

- `go-template` contains the baseline Go CLI/library repository skeleton.
- `infra-tooling` contains reusable GitHub Actions workflows and rollout playbooks.
- `publish-vault` is a real application repository with Go, web, Docker, SSR, GitOps image publishing, and docmgr ticket history.
- Many active repositories have already moved to Go `1.26.x` and `golangci-lint` `v2.12.2`, while older repositories still lag behind.

This guide explains the system for a new intern, then defines a phased implementation plan. The first implementation wave should update the source-of-truth repositories (`go-template` and `infra-tooling`) and `publish-vault`. A later rollout can apply the same changes to the rest of the organization in batches.

## Problem statement and scope

The user asked for a new docmgr ticket that explains how to bring `publish-vault` and related Go Go Golems repositories up to the latest org style, then implements the work sequentially with commits and a detailed diary.

The specific consistency problems are:

- Tool versions drift across repositories.
- `go-template` does not fully encode the linting and publishing practices described by the rollout playbooks.
- `infra-tooling` has reusable workflows that are mostly modern, but still use older `actions/checkout` pins in the GHCR image publisher.
- `publish-vault` is modern in some areas, but should be checked against the current template and reusable workflow style.
- The organization lacks a small, repeatable audit/update recipe for applying the same style to many repositories.

### In scope

- Document the system components a new intern needs to understand.
- Define a consistent target state for Go versions, linting, logcopter, docs publishing, GHCR publishing, and CI layout.
- Update `publish-vault`, `go-template`, and `infra-tooling` where changes are safe and directly supported by evidence.
- Add tasks and diary entries to this ticket.
- Upload the design package to reMarkable.

### Out of scope for the first wave

- Blindly modifying every repository under `~/code/wesen/go-go-golems` without per-repo validation.
- Enabling docs publishing for repositories that do not have Glazed help export commands.
- Enabling logcopter in repositories that do not use zerolog or the logcopter package model.
- Reworking repository architecture beyond build, CI, linting, release, and publishing style.

## Current-state architecture

### Repository roles

The system has three categories of repositories.

```text
+----------------+        seeds new repos        +---------------------+
| go-template    | ----------------------------> | app/library repos   |
| baseline repo  |                               | publish-vault, ...  |
+----------------+                               +---------------------+
        |                                                   |
        | references shared workflows/playbooks              |
        v                                                   v
+----------------+        reusable CI/CD          +---------------------+
| infra-tooling  | ----------------------------> | GitHub Actions     |
| workflows      |                               | releases/images    |
+----------------+                               +---------------------+
```

- `go-template` is the source-of-truth for new Go Go Golems repositories. Evidence: `/home/manuel/code/wesen/go-go-golems/go-template/Makefile:1-73`, `.github/workflows/lint.yml:1-34`, `.github/workflows/push.yml:1-32`.
- `infra-tooling` is the source-of-truth for reusable publishing workflows. Evidence: `/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml:1-260`.
- `publish-vault` is an application repository that consumes org conventions and has extra frontend/Docker/GitOps concerns. Evidence: `/home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/Makefile:1-79`, `.github/workflows/ci.yml:1-59`, `.github/workflows/publish-image.yaml:1-64`.

### Toolchain version model

The current organization style is file-driven. The Go toolchain version comes from `go.mod`; GitHub Actions read it through `actions/setup-go` with `go-version-file: go.mod`. The golangci-lint version comes from `.golangci-lint-version`; CI reads it through `golangci/golangci-lint-action` with `version-file: .golangci-lint-version`.

Observed files:

- `go-template/go.mod:3` currently says `go 1.25.0`.
- `publish-vault/go.mod` currently says `go 1.25.0`.
- `go-template/.golangci-lint-version:1` is already `v2.12.2` in the working tree, but that change was present before this ticket started.
- `publish-vault/.golangci-lint-version` is `v2.12.2`.

A quick org scan showed many active repos already use Go `1.26.3` or `1.26.4`, while `go-template` and `publish-vault` are still at `1.25.0`. The target state should therefore be Go `1.26.4` for active repositories that can build with it.

### Linting model

`go-template` and `publish-vault` both use golangci-lint v2 config files. The shared shape is:

```yaml
version: "2"
linters:
  default: none
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - exhaustive
    - nonamedreturns
    - predeclared
formatters:
  enable:
    - gofmt
```

Evidence:

- `go-template/.golangci.yml:7-43`.
- `publish-vault/.golangci.yml:1-24`.

The config currently declares `gofmt` as a formatter, but standard `golangci-lint run` does not make format drift visible in the same way as a dedicated formatting check. A repository can have a formatter configured but still never run `golangci-lint fmt --diff` or an equivalent check in CI.

### Glazed lint model

The glazed linting playbook defines a separate lint layer for Glazed CLI conventions. It installs `github.com/go-go-golems/glazed/cmd/tools/glazed-lint` and can enforce policies such as:

- do not read raw environment variables directly with `os.Getenv` in Glazed commands;
- do not add raw Cobra/pflag flags directly where Glazed parameter layers should own flags;
- use Glazed processor conventions such as `RunIntoGlazeProcessor` where appropriate.

The important distinction is that `glazed-lint` is only relevant to repositories that depend on Glazed command patterns. It should be present in `go-template` as an optional target, but not forced onto repositories that do not use Glazed.

### Logcopter model

Logcopter standardizes logging by generating package-level logger declarations from a `go:generate` entry. `go-template` already includes the relevant pieces:

- `go-template/go.mod:5` requires `github.com/go-go-golems/logcopter v0.1.0`.
- `go-template/go.mod:18` declares `tool github.com/go-go-golems/logcopter/cmd/logcopter-gen`.
- `go-template/logcopter_generate.go:1-3` contains the generator directive.
- `go-template/Makefile:37-41` defines `logcopter-generate` and `logcopter-check`.
- `go-template/.github/workflows/push.yml:21-23` runs `make logcopter-check`.

This is a good pattern and should remain the baseline. The only template gap is that `ci-check` should include `logcopter-check` so local and CI checks match.

### Docs publishing model

The docs publishing playbook describes publishing generated Glazed help docs to the docs registry. The important pieces are:

- A repository exports help to SQLite, usually with a command like `go run ./cmd/<binary> help export --format sqlite --output-path .docsctl/help.sqlite`.
- A reusable workflow in `infra-tooling` publishes the generated SQLite package via `docsctl`.
- Vault OIDC provides short-lived credentials.
- The release workflow should publish docs only for version tags and only after release artifacts exist.

`go-template/.github/workflows/release.yaml` includes a disabled `publish-docs` job with instructions for enabling it. This is the correct default because not every generated repository will have Glazed help export immediately.

### GHCR image publishing model

`infra-tooling/.github/workflows/publish-ghcr-image.yml` is a reusable workflow for Docker image publication and optional GitOps pull-request creation.

The workflow accepts inputs for:

- Dockerfile and build context (`dockerfile`, `build_context`), evidence lines `6-13`.
- Test command and Go cache settings (`test_command`, `go_version_file`, `go_cache_dependency_path`), evidence lines `14-25`.
- Image coordinates and platforms (`image_name`, `platforms`), evidence lines `26-33`.
- GitOps target config and PR behavior, evidence lines `34-61`.
- Vault/GitHub App credentials for opening GitOps PRs, evidence lines `62-109`.

The reusable workflow still uses `actions/checkout@v5` in three places, while other current workflows use `actions/checkout@v6`. Evidence: `infra-tooling/.github/workflows/publish-ghcr-image.yml:129-130`, `205-209`. This is a safe modernization change.

## Gap analysis

### Gap 1: Source-of-truth template is behind org Go version

The template still says `go 1.25.0`, while many active repositories now use `1.26.3` or `1.26.4`. New repositories generated from the template will inherit an older baseline unless the template is updated.

### Gap 2: Glazed linting playbook is not encoded in go-template

The rollout playbook describes Makefile targets for `glazed-lint`, but `go-template/Makefile` has no such target. This causes drift because new Glazed CLI repositories have to rediscover and copy the target manually.

### Gap 3: Formatters are configured but not explicit in workflows

Both template and application config files include `formatters.enable: [gofmt]`, but workflows only run the normal lint action. Interns need an explicit `fmt-check` target and CI step so formatting failures are obvious and reproducible locally.

### Gap 4: infra-tooling checkout action versions are inconsistent

Reusable image publishing uses `actions/checkout@v5`, while current workflow style uses `@v6`. Since this workflow is shared, stale pins propagate to every consuming repository.

### Gap 5: Local and CI check surfaces differ

`go-template` CI runs `make logcopter-check`, but `Makefile` has no `ci-check` target that mirrors the full CI surface. Local developers need one command for the standard quality gate.

### Gap 6: Org-wide rollout needs batching, not blind mutation

There are many repositories under `~/code/wesen/go-go-golems`. Some are old or archived, some use older Go versions intentionally, and some have frontend or CGO constraints. The implementation should produce a repeatable audit/update method rather than editing every repo at once.

## Proposed target state

The target style has these properties:

1. **Tool versions**
   - Active repos use Go `1.26.4` where validation passes.
   - Active repos use `.golangci-lint-version` `v2.12.2`.
   - GitHub Actions use `actions/checkout@v6`, `actions/setup-go@v6`, `golangci/golangci-lint-action@v9`, and current Docker actions.

2. **Makefile targets**
   - `lint`: run `golangci-lint run` with `GOWORK=off`.
   - `fmt-check`: run a non-mutating formatter diff/check.
   - `test`: run unit tests.
   - `build`: run generators and build.
   - `ci-check`: aggregate the checks a developer should run before pushing.
   - `logcopter-check`: present when logcopter is configured.
   - `glazed-lint`: present as an optional target in the template; enabled in Glazed repos.

3. **CI workflows**
   - Go CI installs Go from `go.mod`.
   - Lint CI installs golangci-lint from `.golangci-lint-version`.
   - Generated files are checked with `git diff --exit-code` after generation.
   - Application repos include frontend checks where relevant.

4. **Publishing workflows**
   - GHCR image publishing uses the reusable `infra-tooling` workflow.
   - GitOps PR creation uses GitHub App credentials from Vault where possible.
   - Docs publishing remains disabled in templates, with clear instructions to enable per repository.

## Implementation architecture

### Rollout flow

```text
Phase 0: document and plan
  |
  v
Phase 1: update sources of truth
  - go-template
  - infra-tooling reusable workflows
  |
  v
Phase 2: update publish-vault
  - Go version
  - local checks
  - CI parity
  |
  v
Phase 3: audit org repositories
  - report drift
  - classify repos
  - batch safe updates
  |
  v
Phase 4: per-repo rollout
  - update version files
  - run checks
  - commit/push/PR
```

### Pseudocode: repository audit

```pseudo
for repo in ~/code/wesen/go-go-golems/*:
    if not exists(repo/.git) or not exists(repo/go.mod):
        continue

    result = {
        name: basename(repo),
        go_version: parse_go_directive(repo/go.mod),
        golangci_version: read_optional(repo/.golangci-lint-version),
        has_golangci_config: exists(repo/.golangci.yml),
        has_makefile: exists(repo/Makefile),
        has_logcopter: contains(repo/go.mod, "logcopter"),
        has_glazed: contains(repo/go.mod, "github.com/go-go-golems/glazed"),
        workflows: list(repo/.github/workflows/*.yml),
    }

    classify result:
        if go_version < 1.25: legacy/manual-review
        else if checks missing: needs-style-update
        else if versions old: safe-version-bump-candidate
        else: current

write markdown table and JSON report
```

### Pseudocode: safe per-repo update

```pseudo
for repo in selected_repos:
    git status --short must be empty

    update go.mod go directive to target version
    update .golangci-lint-version to target version
    if repo has go-template-style Makefile:
        ensure fmt-check and ci-check exist
    if repo has Glazed dependency:
        ensure glazed-lint target exists
    if repo has logcopter dependency:
        ensure logcopter-check is part of ci-check

    run:
        GOWORK=off go mod tidy
        make lint
        make test
        make ci-check if present

    if checks pass:
        git add changed files
        git commit -m "Align tooling with org style"
    else:
        record exact failure in diary and leave repo uncommitted
```

## API and command references

### docmgr commands

```bash
docmgr ticket create-ticket --ticket ORG-STYLE-011 \
  --title "Align publish-vault and go-go-golems repositories with current org style" \
  --topics tooling,ci,release,linting,documentation

docmgr doc add --ticket ORG-STYLE-011 --doc-type design-doc \
  --title "Org style alignment analysis and implementation guide"

docmgr doc add --ticket ORG-STYLE-011 --doc-type reference \
  --title "Investigation diary"

docmgr task add --ticket ORG-STYLE-011 --text "..."
docmgr task check --ticket ORG-STYLE-011 --id 1
docmgr changelog update --ticket ORG-STYLE-011 --entry "..."
docmgr doc relate --doc <doc> --file-note "/abs/path:reason"
```

### Go toolchain commands

```bash
# Read current Go directive
grep '^go ' go.mod

# Update module metadata after changing the directive
GOWORK=off go mod tidy

# Standard local checks
make lint
make test
make ci-check
```

### golangci-lint commands

```bash
# Install the current pinned version locally if needed
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

# Run linters
GOWORK=off golangci-lint run -v

# Formatter check candidate
GOWORK=off golangci-lint fmt --diff
```

### Glazed lint target shape

```make
GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_FLAGS ?=

.PHONY: glazed-lint-build glazed-lint

glazed-lint-build:
	@if [ -n "$(GLAZED_VERSION)" ] && [ "$(GLAZED_VERSION)" != "(devel)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION); \
	else \
		echo "Installing $(GLAZED_LINT_PKG) from workspace/module"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) go install $(GLAZED_LINT_PKG); \
	fi

glazed-lint: glazed-lint-build
	$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) ./...
```

## Decision records

### Decision: Update source-of-truth repositories before mass rollout

- **Context:** `go-template` and `infra-tooling` propagate conventions to many repositories.
- **Options considered:** Update every repo first; update only `publish-vault`; update source-of-truth repos first.
- **Decision:** Update source-of-truth repos first, then `publish-vault`, then run a separate org-wide audit.
- **Rationale:** Template/workflow drift creates repeated future work. Fixing the template first prevents new repositories from inheriting outdated patterns.
- **Consequences:** The first wave does not touch every repo, but creates a safer rollout path.
- **Status:** accepted

### Decision: Treat glazed-lint as optional template capability

- **Context:** Not every Go Go Golems repository uses Glazed commands.
- **Options considered:** Force `glazed-lint` everywhere; omit it from the template; include optional target and enable per repo.
- **Decision:** Add the target to the template but only wire it into required checks for repositories with Glazed dependencies.
- **Rationale:** This keeps the target discoverable without breaking non-Glazed libraries.
- **Consequences:** Interns must still decide whether a repository is a Glazed CLI repo before enabling the required check.
- **Status:** accepted

### Decision: Keep docs publishing disabled in the template

- **Context:** Docs publishing requires an export command, Vault role, and registry package setup.
- **Options considered:** Enable by default; remove from template; keep disabled with instructions.
- **Decision:** Keep the disabled job with instructions.
- **Rationale:** The template should teach the pattern without assuming every repo can export Glazed help.
- **Consequences:** Repositories must explicitly opt in after their help export command works.
- **Status:** accepted

### Decision: Use audit reports for org-wide rollout

- **Context:** The org contains active, legacy, frontend, CGO, and possibly archived repositories.
- **Options considered:** Mass edit all repositories; manually inspect each repository; generate an audit report and batch safe candidates.
- **Decision:** Generate an audit report and batch safe candidates.
- **Rationale:** This avoids contaminating unrelated working trees and gives reviewers a clear checklist.
- **Consequences:** Full org convergence takes more than one pass, but failures are isolated and documented.
- **Status:** accepted

## Implementation plan

### Phase 1: Ticket setup and documentation

- Create `ORG-STYLE-011`.
- Add design doc and diary.
- Add detailed tasks.
- Relate key files from `publish-vault`, `go-template`, and `infra-tooling`.
- Upload the initial design package to reMarkable.

### Phase 2: Update infra-tooling reusable workflow pins

- Change all `actions/checkout@v5` occurrences in `infra-tooling/.github/workflows/publish-ghcr-image.yml` to `actions/checkout@v6`.
- Validate by checking workflow YAML syntax enough to catch obvious mistakes.
- Commit in `infra-tooling`.

### Phase 3: Update go-template baseline

- Change `go-template/go.mod` from `go 1.25.0` to `go 1.26.4`.
- Preserve the pre-existing `.golangci-lint-version` update to `v2.12.2` if validation passes.
- Remove the Glazed-specific `SA1019: cli.CreateProcessorLegacy` exclusion from the generic template.
- Add `fmt-check`, `ci-check`, and optional `glazed-lint` targets.
- Add `fmt-check` and `logcopter-check` to CI or local aggregate checks.
- Reconsider `lefthook` pre-push `make goreleaser`; if changed, document the rationale.
- Run `GOWORK=off go mod tidy`, `make lint`, `make test`, and `make logcopter-check`.
- Commit in `go-template`.

### Phase 4: Update publish-vault application baseline

- Change `publish-vault/go.mod` from `go 1.25.0` to `go 1.26.4` if validation passes.
- Add a local formatter check if supported by installed golangci-lint.
- Ensure `ci-check` reflects the important local quality gate.
- Run `GOWORK=off go mod tidy`, `make lint`, `make test`, `make web-check`, and `make web`.
- Commit in `publish-vault`.

### Phase 5: Add org audit helper

- Add a small script under this ticket workspace, not directly to product code, that scans repository versions and emits a Markdown report.
- Use the script to classify repositories into current, safe bump candidate, and manual review.
- Attach the report to the ticket.

### Phase 6: Batch rollout to other repositories

- Select a small batch of safe candidates.
- Ensure each repository has a clean working tree before editing.
- Apply version bumps and minimal style updates.
- Run checks per repository.
- Commit per repository, not as a cross-repo mixed commit.
- Record failures in the diary.

## Testing and validation strategy

### Documentation validation

- Run `docmgr doctor --ticket ORG-STYLE-011 --stale-after 30`.
- Run frontmatter validation on the design doc if doctor reports metadata issues.
- Upload to reMarkable with a dry run first.

### infra-tooling validation

- Use `git diff --check` after editing YAML.
- Inspect workflow snippets around each changed checkout action.
- If available, run an action lint tool such as `actionlint`.

### go-template validation

- `GOWORK=off go mod tidy`.
- `make lint`.
- `make test`.
- `make logcopter-check`.
- `make ci-check` after adding it.

### publish-vault validation

- `GOWORK=off go mod tidy`.
- `make lint`.
- `make test`.
- `make web-check`.
- `make web`.
- `make ci-check` if time permits; note that it includes security scanners and may take longer.

## Risks and open questions

- Go `1.26.4` may require the Go toolchain to download a newer patch release when the local machine has `go1.26.0`; this is acceptable if `GOTOOLCHAIN=auto` works, but failures should be recorded.
- Some older repositories may intentionally remain on old Go versions due to dependencies or archived status.
- `golangci-lint fmt --diff` support depends on the installed v2 CLI; if local tooling is older, update local tooling before enforcing the target.
- `glazed-lint` should not be run in repositories that do not depend on Glazed.
- Publishing docs requires Vault roles and registry setup; template instructions are not enough to enable it safely.

## References

- `/home/manuel/code/wesen/go-go-golems/go-template/Makefile` — baseline Makefile targets.
- `/home/manuel/code/wesen/go-go-golems/go-template/.golangci.yml` — baseline lint config.
- `/home/manuel/code/wesen/go-go-golems/go-template/.github/workflows/lint.yml` — baseline lint workflow.
- `/home/manuel/code/wesen/go-go-golems/go-template/.github/workflows/push.yml` — baseline test/generated-file workflow.
- `/home/manuel/code/wesen/go-go-golems/go-template/.github/workflows/release.yaml` — release and disabled docs publishing template.
- `/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml` — reusable GHCR/GitOps workflow.
- `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/glazed-linting-rollout-playbook.md` — Glazed lint rollout guide.
- `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/playbooks/docsctl-docs-publishing-rollout-playbook.md` — docsctl publishing guide.
- `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/playbooks/logcopter-package-rollout-playbook.md` — logcopter rollout guide.
- `/home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/Makefile` — publish-vault local quality gates.
- `/home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/.github/workflows/ci.yml` — publish-vault application CI.
- `/home/manuel/workspaces/2026-06-12/publish-vault-style/publish-vault/.github/workflows/publish-image.yaml` — publish-vault image publishing workflow.
