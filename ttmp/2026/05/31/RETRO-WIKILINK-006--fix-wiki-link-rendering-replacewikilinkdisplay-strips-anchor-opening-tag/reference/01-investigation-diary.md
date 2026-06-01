---
doc_type: reference
title: Investigation diary
ticket: RETRO-WIKILINK-006
status: active
intent: long-term
topics: [parser, wiki-link, html-rendering, regression]
owners: []
created: 2026-05-31
---

# Diary

## Goal

Fix wiki-link rendering in the retro-obsidian-publish system where `ReplaceWikiLinkDisplay` strips the `<a` opening tag from anchor elements, causing broken rendering especially in table contexts.

## Step 1: Reproduce and root-cause the bug

I investigated the wiki-link rendering pipeline by tracing the code from Markdown source through all five stages of processing. I created temporary test files in the parser package to confirm the bug.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to fix the link rendering for wiki link objects, for example ~/code/wesen/go-go-golems/go-go-parc/Research/KB/Tribal/dmeta-design-system-compiler-pipeline.md does render like: href=\"/note/projects/2026/05/23/article-dmeta-design-system-factory-from-semantic-schemas-to-generated-react-widgets\" class=\"wiki-link\" data-target=\"projects/2026/05/23/article-dmeta-design-system-factory-from-semantic-schemas-to-generated-react-widgets\" data-raw=\"ARTICLE - DMETA Design System Factory - From Semantic Schemas to Generated React Widgets\" data-alias=\"\">DMETA Design System Factory: From Semantic Schemas to Generated React Widgets     2026-05-23     Canonical description: nine-layer model, archetypes, capabilities, widget templates, scaffold generation, promotion workflow"

**Assistant interpretation:** Fix the wiki-link rendering bug where the `<a>` tag opening is missing and table cell content bleeds into link text.

**Inferred user intent:** Make wiki links render correctly in all contexts (especially tables), write documentation, fix the issue, keep a diary, and upload docs to reMarkable.

### What I did

- Read all relevant source files: parser.go, vault.go, api.go, NoteRenderer.tsx, wikiLinks.ts, staticVault.ts, App.tsx, index.css
- Created temporary Go test files to reproduce the bug in isolation
- Confirmed that `ReplaceWikiLinkDisplay` strips the `<a` prefix from the anchor tag
- Verified that Stage 1 (wikiLinkHTML) and Stage 2 (goldmark) produce correct HTML — the bug only appears in Stage 3 (ReplaceWikiLinkDisplay)
- Traced the regex capturing groups and identified the missing `<a` prefix in the reconstruction
- Created RETRO-WIKILINK-006 ticket with design doc and diary

### Why

The user reported broken wiki-link rendering. I needed to find exactly where in the pipeline the HTML gets corrupted.

### What worked

- Creating targeted Go test files in the parser package was the fastest way to confirm the bug
- The `TestDebugReplaceWikiLinkDisplayRegex` test showed exactly which groups captured what
- The `TestReplaceWikiLinkDisplayManglesAnchorTag` test confirmed the `<a` prefix is missing

### What didn't work

- Initially I tried to reproduce the bug with a simple table, which passed. The bug is actually in `ReplaceWikiLinkDisplay`, not in the initial wiki-link-to-HTML conversion.

### What I learned

- The Go parser's `wikiLinkHTML` function produces correct HTML even in table contexts
- The bug is solely in `ReplaceWikiLinkDisplay` — a one-character fix (adding `<a` prefix)
- The "table cell content bleeding" is a downstream effect of the broken HTML, not a separate bug

### What was tricky to build

- Understanding why the table cell content appeared inside the link text was tricky until I realized that a missing `<a>` tag makes the `href=...` attributes appear as raw text content inside the `<td>`, and the browser renders all that text in one visual block.

### What warrants a second pair of eyes

- The regex-based HTML manipulation in `ReplaceWikiLinkDisplay` is inherently fragile. The fix (adding `<a`) is simple, but verify that the regex doesn't fail on edge cases like wiki links with complex attributes.
- Verify that `ReplaceWikiLinksString` doesn't have a similar reconstruction bug (it doesn't — it only modifies attribute values).

### What should be done in the future

- Refactor `ReplaceWikiLinkDisplay` to use `golang.org/x/net/html` for proper HTML parsing
- Add `data-raw` and `data-alias` attributes to the static vault's marked extension
- Add fuzz testing for the wiki-link replacement pipeline

## Step 3: Full vault smoke test — 650 notes, 1308 wiki-links, 0 broken anchors

Ran a comprehensive scan against the real go-go-parc vault (650 notes, 313 with wiki-links, 1308 total anchors). Every wiki-link anchor has a proper `<a>` tag — zero broken anchors found. Also audited the codebase for similar regex reconstruction bugs; `ReplaceWikiLinksString` and `RewriteImageSources` both handle their tag prefixes correctly.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue with verification — deploy to real vault and audit for similar bugs.

**Inferred user intent:** Confirm the fix works at scale and check for related issues.

### What I did

- Started the server against the real go-go-parc vault (650 notes)
- Verified the specific problematic note (dmeta-design-system-compiler-pipeline) renders all table wiki-links correctly
- Ran a comprehensive scan of ALL 650 notes checking every wiki-link anchor for the missing `<a>` tag bug
- Audited `ReplaceWikiLinksString` and `RewriteImageSources` for similar reconstruction issues

### Why

A single-note test isn't sufficient — the vault has diverse content and edge cases.

### What worked

- The comprehensive scan found 1308 wiki-link anchors across 313 notes, all properly formed
- The table in the dmeta note renders all 9 report links with resolved titles and no cell bleeding
- No similar regex reconstruction bugs exist in the codebase

### What didn't work

- Initial false-positive regex check flagged some notes as broken — the check was matching `href=` in attribute context, not just standalone. Fixed by checking backwards from `class="wiki-link"` for a preceding `<a `.

### What I learned

- The vault has a wide variety of wiki-link styles (plain targets, path targets, aliased links, heading links) — all render correctly after the fix.

### What was tricky to build

- Writing a correct regex to detect "broken" wiki-link anchors (missing `<a>` prefix) is tricky because `href=` appears both as a standalone broken text and as a valid attribute inside `<a href=...>`. The reliable approach is to look backwards from `class="wiki-link"` for a preceding `<a `.

### What warrants a second pair of eyes

- Nothing further — the fix is verified at scale.

### What should be done in the future

- N/A — this ticket is complete.

### Code review instructions

- Server was started with: `./retro-obsidian-publish serve --vault ~/code/wesen/go-go-golems/go-go-parc --port 8097`
- Verification: scanned all 650 notes, 1308 wiki-link anchors, 0 broken

### Technical details

- Full vault scan confirms: the one-character fix resolves all instances of the bug

### Code review instructions

- Start with `backend/internal/parser/parser.go`, function `ReplaceWikiLinkDisplay`, the `prefix :=` line
- Verify the fix by running: `cd backend && go test ./internal/parser/ -v -run TestReplaceWikiLinkDisplay`
- Smoke test: `./retro-obsidian-publish serve --vault-dir ~/code/wesen/go-go-golems/go-go-parc --port 8090`

### Technical details

- Bug location: `backend/internal/parser/parser.go`, `ReplaceWikiLinkDisplay`, line `prefix := sub[1] + ...`
- Fix: Change to `prefix := "<a" + sub[1] + ...`
- Regex: `<a([^>]*?)class="wiki-link"([^>]*?)data-raw="([^"]*?)"([^>]*?)>([^<]*?)</a>`

## Step 2: Implement the fix and add regression tests

Applied the one-character fix: added `<a` prefix to the `prefix` line in `ReplaceWikiLinkDisplay`. Added four regression tests covering anchor tag preservation, table context, explicit alias preservation, and end-to-end table rendering with display replacement.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the fix for the missing `<a` prefix bug.

**Inferred user intent:** Fix the rendering bug and verify it with tests.

**Commit (code):** 10533cf — "fix(parser): add missing <a prefix in ReplaceWikiLinkDisplay regex reconstruction"

### What I did

- Changed `prefix := sub[1] + ...` to `prefix := "<a" + sub[1] + ...` in `ReplaceWikiLinkDisplay`
- Added 4 regression tests in `parser_test.go`
- Ran the full test suite — all tests pass
- Smoke tested with a test vault containing wiki links in tables — HTML renders correctly

### Why

The fix is the minimal change needed to restore the `<a>` opening tag in the reconstructed anchor element.

### What worked

- The one-character fix immediately resolved both symptoms (missing `<a>` tag and table cell bleeding)
- The test vault confirmed that display text replacement now works correctly in table contexts
- Lefthook pre-commit hooks (golangci-lint + go test) pass cleanly

### What didn't work

- The real vault (go-go-parc) is too large to load quickly for smoke testing — the server process took too long. Used a smaller test vault instead.

### What I learned

- The `ReplaceWikiLinkDisplay` function was the only place with this reconstruction bug. `ReplaceWikiLinksString` only modifies attribute values and doesn't reconstruct tags.
- In a table context, the bug manifests as table cell content bleeding into the "link text" because without the `<a>` tag, `href=...` attributes become raw text in the `<td>`.

### What was tricky to build

- Smoke testing with the real vault was impractical due to its size (~7000+ files). Created a minimal test vault with wiki links in tables to verify the fix.

### What warrants a second pair of eyes

- Verify that the fix works on the full go-go-parc vault when deployed.
- The regex-based approach in `ReplaceWikiLinkDisplay` is still fragile — consider refactoring to use an HTML parser in the future.

### What should be done in the future

- Refactor `ReplaceWikiLinkDisplay` to use `golang.org/x/net/html`
- Add `data-raw` and `data-alias` attributes to the static vault's marked extension

### Code review instructions

- See the diff: `git show 10533cf`
- Run tests: `cd backend && go test ./internal/parser/ -v -run TestReplaceWikiLinkDisplay`
- Full suite: `cd backend && go test ./... -count=1`

### Technical details

- Fix: `backend/internal/parser/parser.go:294` — `prefix := "<a" + sub[1] + ...`
- Tests: `backend/internal/parser/parser_test.go` — 4 new test functions
