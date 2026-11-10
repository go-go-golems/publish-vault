---
Title: Completion audit and evidence map
Ticket: PV-MARKDOWN-030
Status: active
Topics:
    - markdown
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/document.go
      Note: Document ownership and renderer
    - Path: repo://internal/parser/math_ast.go
      Note: Structural math
    - Path: repo://internal/parser/wikilink_ast.go
      Note: Wiki node resolution
    - Path: repo://pkg/vault/vault.go
      Note: Index and rebuild lifecycle
ExternalSources: []
Summary: Requirement-to-evidence audit for the structural Markdown migration
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Completion audit and evidence map

Audit against the active goal after code commit `0185f51` and vault report commit `11a6fa1da29c8cea455c8520382ea56da26507aa`. This concerns the requested branch implementation and documentation; it does not assert production deployment.

| Requirement | Concrete evidence |
|---|---|
| Structural callouts, wiki links and math; ordinary soft breaks | `internal/parser/callout.go`, `wikilink_ast.go`, `math_ast.go`, `softbreak_test.go`; implementation commits `073797d`, `70c126b`, `9d47f60`, `0d184dc`, `b4384c3`. |
| Vault note, heading, embed and image resolution without whole-HTML rewriting | `internal/parser/document.go` and `wikilink_ast.go`; `pkg/vault/vault.go` builds indexes then renders documents via resolver callbacks; old `RestoreMath`, `ReplaceWikiLinksString`, `RewriteImageSources`, `ResolveWikiLinkHeadings`, and `sourceHTML` implementations are absent from production code. |
| Coherence, code hygiene and explicit behavior changes | The current guide `design-doc/02-implemented-markdown-document-pipeline-and-operational-guide.md` describes source/index ownership, deterministic alias collisions, malformed table handling, raw HTML trust, resolution failures and reparsing cost. Goldmark's duplicate GFM registrations were removed in `0185f51`. Search of changed production files found no TODO/FIXME/HACK or old sentinel/HTML-rewrite names. This is a review and test claim, not proof against all possible Markdown inputs. |
| Preserve unrelated work | Final `git status --short` contains only the same pre-existing untracked `.claude/`, `.playwright-mcp/`, `search-memory-results.png`, and `ttmp/vocabulary.yaml.orig`; tracked changes are committed. Vault working tree is clean. |
| Maintain detailed diary and commits | `reference/01-implementation-diary.md`, Steps 1–18, records requests, changes, test results, failures and recovery. `git log --oneline 0c78a5e..HEAD` contains separate design, callout, wiki, security, structural renderer, fuzz and cleanup commits. |
| Integrated tests, lint, type-check, build and security | `make ci-check` passed on the final production-code state before `0185f51`; command output is `/tmp/pv-markdown-030-ci-final-audit.log`. It covers fmt, lint, logcopter, Go/plugin tests, gosec, govulncheck, web check and build. Govulncheck reports zero called vulnerabilities, while noting uncalled dependency findings. Pre-commit backend test/lint passed when committing `0185f51`. |
| Concurrency, malformed input and executable build | `go test -race ./internal/parser ./pkg/vault ./pkg/watcher` passed after final production changes; `FuzzStructuralParse` completed 43,718 executions in the bounded run; `go build -tags embed` and CLI help passed. |
| Real-note and browser checks | The actual JavaScript OS note was locally rendered without the forced source-line break. The 2,372-note corpus inventory/load and a controlled 2,373-note pre/post-GFM-cleanup inventory are documented in diary Steps 14 and 18. `reference/assets/embedded-smoke.png` records a local embedded-server fixture; the DOM showed soft flow, resolved link, table cells, and MathJax `done`. Only fixture favicon 404s appeared. |
| Ticket/design/diary and requested original reMarkable delivery | `docmgr doctor --ticket PV-MARKDOWN-030 --stale-after 30` passed, all tasks are checked, original design and current implementation guide are present. Diary Step 2 records `OK: uploaded PV MARKDOWN 030 Markdown Pipeline Design.pdf -> /ai/2026/11/09/PV-MARKDOWN-030`. That upload predates implementation. |
| Textbook-style Obsidian report, commit and verified push | `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/11/10/PROJECT REPORT - Publish Vault - A Structural Markdown Rendering Pipeline.md` is committed as `11a6fa1da29c8cea455c8520382ea56da26507aa`; `git ls-remote origin refs/heads/main` returned the same hash. Only that note was staged. `reference/assets/report-diagram.png` records visual review after layout and contrast fixes. |

## Limits of the evidence

The browser check used a local synthetic vault, not the public site. The corpus scanner is lexical and inventory counts do not establish exhaustive semantic equivalence. Timing and live-heap figures are single local measurements, not statistical benchmarks or peak-memory measurements. Raw vault HTML remains intentionally trusted, not sanitized. The branch was not deployed or merged in this work; neither action was requested as part of this goal.
