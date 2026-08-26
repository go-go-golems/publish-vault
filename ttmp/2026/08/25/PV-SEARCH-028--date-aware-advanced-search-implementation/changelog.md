# Changelog

## 2026-08-25

- Initial workspace created

## 2026-08-25

Phase A: canonical authored-date domain in Go and TS, shared fixture, invalid-date counter, and static JSON_SCHEMA scalar preservation.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/vault/date.go — Date resolver
- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/web/src/vault/staticVault.ts — JSON_SCHEMA frontmatter parsing

## 2026-08-25

Phase B: typed SearchRequest/SearchResponse, Bleve date/tag/path keyword mappings, compound query builder, deterministic sorts, pagination, and date-aware result extraction.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/search/search.go — Mapping and SearchAdvanced

## 2026-08-25

Phase C: /api/search/advanced endpoint with stable 400 field-error envelope, unknown-parameter and repeated-singleton rejection, and legacy /api/search delegating to SearchAdvanced.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/pkg/api/search_request.go — Advanced handler

## 2026-08-25

Phase D: shared TS types, canonical URL codec, RTK Query searchAdvanced endpoint, and static-mode advanced search parity with the backend contract.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/web/src/vault/staticVault.ts — staticSearchAdvanced

## 2026-08-25

Phase E: URL-driven SearchPage with accessible filter panel, sort, applied chips, pagination, and NoteCard authored-date/path rendering.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/web/src/components/pages/SearchPage/SearchPage.tsx — URL-driven search

## 2026-08-25

Phase F: full validation (ci-check, race, web, storybook), Docker/Compose smoke, and validation evidence.

### Related Files

- /home/manuel/workspaces/2026-08-25/publish-vault-mem/publish-vault/ttmp/2026/08/25/PV-SEARCH-028--date-aware-advanced-search-implementation/artifacts/final/01-phase-f-validation.md — Phase F evidence
