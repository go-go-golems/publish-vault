# Changelog

## 2026-08-09

- Initial workspace created


## 2026-08-09

Investigated the reported 'Note not found' for /note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields. Proved with scripts/01-slug-probe against the real go-go-parc vault that the note loads, its slug is character-for-character the URL, and none of the four exclusion paths fired - the slugifier is NOT at fault.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — slugify keeps _ and / so the nested underscored path survives intact


## 2026-08-09

Root cause identified: web/server.mjs fetchAPI (83-91) returns null for four distinct failures (genuine 404, non-2xx, thrown fetch, unparseable body) and 242-245 renders all of them as HTTP 404 'Note not found'. Proven by scripts/03-ssr-conflation-repro.mjs: rows B (genuine 404), C (backend unreachable) and E (non-JSON body) are byte-identical.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/web/server.mjs — the four-line conflation that mislabels an unavailable backend as a missing note


## 2026-08-09

Second, independent defect reproduced deterministically ON PRODUCTION: appending a trailing slash to the note URL returns exactly HTTP 404 'Note not found' (14 bytes), because slugify's strings.Trim(s, "-") does not trim '/' and Vault.GetNote is an exact-match map lookup with no normalization.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/vault/vault.go — GetNote (725) performs no normalization of the incoming slug


## 2026-08-09

Recorded the trigger conditions: production served 503 'no available server' mid-investigation and recovered in ~10s, and a local load of the same vault costs 1.56 GB heap / 82s (19s vault + 62s search index) per snapshot, which RuntimeState.Reload doubles during a swap. Noted as the interaction with PV-MEMORY-019; not investigated further here.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/pkg/server/runtime.go — Reload keeps the previous snapshot on failure, which is the stale-content window


## 2026-08-09

Audited the real vault (scripts/05-vault-slug-audit): 1740 markdown files on disk vs 1713 indexed; 22 excluded by the single .vault-ignore rule ttmp/_*/; 5 slugs claimed by two files each (case-only variants, silently last-write-wins); 0 empty slugs today, though Cyrillic and CJK filenames slugify to the empty string.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/ttmp/2026/08/09/PV-SLUG-020--note-not-found-for-nested-transcripts-slug-diagnose-slug-resolution-and-vault-exclusion/scripts/05-vault-slug-audit/main.go — disk-vs-index diff, collision and empty-slug audit

