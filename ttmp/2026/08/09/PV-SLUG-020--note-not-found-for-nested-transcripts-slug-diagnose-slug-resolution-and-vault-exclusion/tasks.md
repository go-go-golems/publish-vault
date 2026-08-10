# Tasks

## TODO

- [x] Phase 1 (F1): make web/server.mjs fetchAPI return a tagged result {kind: ok|not_found|unreachable|server_error|bad_body} instead of null, and branch at server.mjs:242 so only a genuine API 404 yields 404 'Note not found'; unreachable yields 503 and server_error/bad_body yield 502 <!-- t:1cxh -->
- [x] Phase 2 (F4): log every silently-dropped note in Vault.LoadAll (pkg/vault/vault.go:145) with its reason - vault-ignore, config blacklist, publish:false, parse error - and WARN on slug collision when overwriting v.notes[slug] <!-- t:r8pm -->
- [x] Phase 3 (F2): add a normalized fallback index (lowercase, strip trailing '/', collapse '//', Unicode NFC) beside v.notes and have pkg/api getNote 308-redirect an exact-miss to the canonical slug; assert Slugify idempotence to prevent redirect loops <!-- t:hltq -->
- [ ] Phase 4: refuse to store the empty slug (non-Latin filenames slugify to "") and replace last-write-wins slug collisions with a deterministic disambiguating suffix; re-run scripts/05-vault-slug-audit until the collision section is empty <!-- t:875m -->
- [ ] Phase 5 (F5+F6): add a gated /api/notes/_diagnose?path=... endpoint reporting computed slug, seen/not-seen and exclusion reason, and thread 'reason' into the 404 body with a public/private disclosure split <!-- t:stay -->
- [x] Add the regression tests from design doc section 12: parser slug algebra table (27 rows + idempotence), vault trailing-slash/collision/empty-slug/exclusion-reason tests, api route-shape table incl. the %2F-encoded row, and a web test pinning the four fetchAPI outcomes to 404/503/502/502 <!-- t:j9kg -->
- [ ] Phase 4 status: the empty-slug half shipped in 878e372 (degenerate slugs are refused, not stored under ""). The collision half is NOT done — last-write-wins is now logged as a warning but still drops a note; a disambiguating suffix would change live URLs and needs a decision. <!-- t:0t4x -->
