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
