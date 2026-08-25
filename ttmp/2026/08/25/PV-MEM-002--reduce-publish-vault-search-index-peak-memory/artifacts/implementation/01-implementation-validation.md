---
Title: PV-MEM-002 production batching implementation validation
Ticket: PV-MEM-002
Status: complete
Topics:
    - memory
    - search
    - bleve
    - performance
    - reload
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Production wiring and correctness evidence for persistent 16-document 1 MiB Bleve batches, including 20-query real-vault equivalence across 16725 results and full repository gates.
LastUpdated: 2026-08-25T18:50:00-04:00
WhatFor: Reviewing the accepted implementation before repeated candidate measurements.
WhenToUse: Read before Phase 4 proof, code review, or changing persistent search batch bounds.
---

# PV-MEM-002 production batching implementation validation

## Implementation

Persistent full-snapshot construction now calls `search.NewPersistentWithOptions` with internal server-owned limits:

```text
persistentSearchBatchDocuments = 16
persistentSearchBatchBytes     = 1 MiB
```

In-memory index construction and incremental `Index` calls retain their existing one-document behavior. The limits are internal constants, not public CLI tuning flags. Both count and estimated field bytes are required; a document larger than 1 MiB is committed alone.

Progress advances only after a successful Bleve batch commit. Failed staged documents are not reported as processed. Constructor validation occurs before deleting a persistent target path.

## Real-vault search equivalence

The content-free harness built a current one-document persistent index and a batch-16 persistent index from the same detached vault commit. It ran 20 exact, multi-word, fuzzy, prefix, `#tag`, and `tag:` queries with a limit larger than the complete 2,030-note corpus.

```text
queries: 20
total results compared: 16,725
result identities: equal
stored title/excerpt/tags: equal
maximum score difference: 3.469446951953614e-18
passed: true
```

The retained JSON contains only query SHA-256 values, result counts, score deltas, bounds, and pass/fail state. It contains no query result slugs, titles, excerpts, tags, or note content.

The first equivalence attempt used a limit of 50. Four broad tag queries returned different equal-score top-50 subsets across segment layouts. That did not show a different match set; it showed that the existing API has no secondary order for exact score ties at a truncation boundary. The final harness requests the complete matching set and compares it by slug, fields, and score. This preserves the actual semantic contract while avoiding an arbitrary tie subset.

## Failure and lifecycle coverage

Tests cover:

- batch bounds must both be zero or both positive;
- invalid options preserve an existing target path;
- count-bound and byte-bound flush;
- indivisible oversized document handling;
- final partial-batch flush;
- monotonic committed progress;
- constructor close after source-read failure;
- current/batched fixture search equivalence;
- persistent fresh rebuild drops deleted documents;
- close and reopen at final snapshot path;
- runtime batch constants remain pinned to reviewed values;
- persistent reload drops deleted notes;
- failed build retains the prior snapshot;
- reload serialization and unchanged-revision skip;
- delayed old-index close and directory cleanup;
- measurement phases, receipts, private metrics, and generated fixture budgets.

## Validation gates

Fresh gates completed after production wiring:

```text
go generate ./...                         PASS
make ci-check                             PASS
golangci-lint + Glazed lint               PASS
logcopter generation check                PASS
go test ./...                             PASS
go test -race ./... -count=1              PASS
gosec                                     0 issues
govulncheck                               0 called vulnerabilities
pnpm --dir web check                      PASS
pnpm --dir web build                      PASS
GOOS=darwin GOARCH=arm64 go build ./...   PASS
git diff --check                          PASS
```

The frontend build retained its pre-existing large-chunk warning. This backend change does not alter frontend code or chunk composition.

## Remaining acceptance work

This phase establishes code and correctness, not the final memory claim. Phase 4 must rebuild the exact candidate binary, run the complete server lifecycle three times against the pinned workload, compare medians and ranges to Phase 0, and validate an isolated finite-cgroup container run.
