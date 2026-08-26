---
Title: Phase F validation evidence
Ticket: PV-SEARCH-028
Status: complete
Topics:
    - search
    - regression
    - deployment
DocType: reference
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://pkg/search/search.go
      Note: Advanced search implementation
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-26T02:50:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Phase F validation evidence

## Repository gates

- `make ci-check`: PASS (exit 0)
- `go test -race ./... -count=1`: PASS (all packages)
- `pnpm --dir web vitest run`: 82/82 PASS
- `pnpm --dir web build`: PASS
- `pnpm --dir web build-storybook`: PASS
- `golangci-lint`: 0 issues
- `gosec`: 0 issues
- `git diff --check`: clean

## Docker / Compose smoke

- `docker build`: PASS (image `publish-vault:pv-search-028`)
- `docker compose up -d --build`: PASS (app + ssr Up)
- `/api/healthz`: 200, 6 notes, heapAlloc ~6.4 MiB
- `/api/search/advanced?tag=zettelkasten&sort=newest`: 200 envelope, 1 result with
  `path` and `score`; `date` omitted (truthful — demo note has no authored dates)
- `/api/search/advanced?date_from=2024-02-01&date_to=2024-01-01`: 400 with stable
  `invalid_search_request` envelope and `date_to/before_date_from` field error
- `/api/search?q=zettel`: 200 bare array (legacy adapter)
- app memory: ~14.7 MiB for the 6-note demo vault (well within PV-MEM-002 budgets)

## Index-size / memory note

The new noteDoc fields (tags_kw, path, path_kw, created_at, updated_at,
display_at, date_kind, date_precision) add to per-document index size.
`searchDocumentBytes` counts them so the PV-MEM-002 1 MiB batch bound stays
honest. Persistent indexing still uses bounded batches; the demo-vault smoke
and the race suite confirm no regression. A full-vault measurement against the
private vault should be run before the GitOps rollout to confirm the bounded
peak stays within the 1 GiB request / 2 GiB limit, but the bounded-batch
mechanism is unchanged.

## Rollout (pending review/merge)

- PR to be created against `main`.
- After merge: build and publish the optimized image, bump the GitOps image tag,
  and roll out to the Hetzner deployment with `maxSurge: 0, maxUnavailable: 1`.
- Rollback: revert the image tag; persistent indexes rebuild automatically.
