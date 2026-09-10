---
Title: Investigation diary
Ticket: PV-WEBKIT-FONT-001
Status: complete
Topics:
    - frontend
    - styling
    - ux
    - regression
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://web/src/index.css
      Note: Frontend stylesheet entry point that imports base.css.
    - Path: repo://web/src/styles/base.css
      Note: Implementation and generated-CSS validation target.
ExternalSources: []
Summary: Chronological implementation diary for removing WebKit font pixelation from the published vault UI.
LastUpdated: 2026-09-10T16:20:00-04:00
WhatFor: Resuming or reviewing the WebKit font-rendering fix.
WhenToUse: Read before continuing work on PV-WEBKIT-FONT-001.
---




# Investigation diary

## Goal

Remove the unintended pixelated font rendering in WebKit browsers while preserving the application's intentional retro visual treatment. Confirm the smallest CSS-only change, validate the frontend build and type checks, and leave the ticket with an evidence-backed implementation record.

## Step 1: Create the ticket and locate the rendering control

The ticket workspace was created before changing source files so the scope, tasks, and investigation record have a stable home. I then traced the stylesheet entry point into the global base rules and searched the repository for font-rendering and pixelation declarations. The issue is localized: `web/src/index.css` imports `web/src/styles/base.css`, and `html` in `base.css` explicitly disables WebKit font smoothing with `-webkit-font-smoothing: none`.

The same rule also sets `font-smooth: never` and `image-rendering: pixelated`. The latter concerns raster/image rendering rather than glyph antialiasing, so it should not be changed as part of a narrowly scoped font fix. The likely one-line correction is to replace the WebKit value `none` with `antialiased`, allowing WebKit to smooth glyph edges while retaining the retro palette, fonts, sizes, and layout.

### Prompt Context
**User prompt (verbatim):** "Create a new docmgr ticket to remove the font pixelation on webkit. I think it's just a simple css tweak. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"
**Assistant interpretation:** Create a new docmgr ticket, investigate and implement the smallest CSS fix for WebKit font pixelation, validate it, commit coherent milestones, and maintain a detailed investigation diary in the ticket.
**Inferred user intent:** Make text readable in WebKit without undoing the site's intentional retro styling.
**Commit (code):** Not yet; investigation checkpoint only.

### What I did
- Ran `docmgr status --summary-only` to confirm the repository's existing docmgr workspace and vocabulary.
- Created ticket `PV-WEBKIT-FONT-001` with topics `frontend`, `styling`, `ux`, and `regression`.
- Added the ticket diary and three focused tasks.
- Read `web/src/index.css` and `web/src/styles/base.css`.
- Searched the source tree for `font`, `webkit`, `text-render`, `antialias`, `pixel`, and `image-render` declarations.
- Confirmed the stylesheet import order: `index.css` imports `tokens.css`, `bridge.css`, `base.css`, `chrome.css`, and `prose.css`.

### Why
- A global `-webkit-font-smoothing: none` declaration directly requests unsmoothed, pixelated glyph rendering in WebKit.
- Changing only that declaration minimizes visual and behavioral risk. It avoids conflating font smoothing with `image-rendering: pixelated`, which is a separate raster-image policy.
- `antialiased` is supported by WebKit and explicitly requests grayscale antialiasing, removing the reported glyph pixelation without introducing a font dependency.

### What worked
- The repository search found one global WebKit font-smoothing declaration in `web/src/styles/base.css`, making the proposed change easy to isolate.
- The relevant stylesheet is part of the normal Vite entry path, so a CSS edit will be covered by the existing frontend build.

### What didn't work
- No failures observed during ticket creation or investigation.

### What I learned
- The retro design notes intentionally mention pixel-crisp rendering, but the current implementation applies that choice globally to all text through the `html` rule. The requested fix should narrow that policy rather than remove the broader visual language.
- `font-smooth: never` is non-standard and does not provide the WebKit-specific control needed for this issue; `-webkit-font-smoothing` is the actionable declaration.

### What was tricky to build
- Separating glyph rasterization from image rasterization: both are described with “pixelated” language, but only `-webkit-font-smoothing: none` controls the WebKit font symptom. Touching `image-rendering` would broaden the change unnecessarily.

### What warrants a second pair of eyes
- Review whether grayscale `antialiased` is the desired WebKit appearance versus omitting the vendor declaration and accepting the browser default. The former is deterministic and directly reverses the current `none` policy; the latter would vary by browser/platform.
- Confirm that the global rule should remain global and that no component intentionally depends on unsmoothed text.

### What should be done in the future
- Validate the final appearance in a WebKit-capable browser on the affected deployment or a representative local page. Automated CSS/build checks cannot prove visual glyph quality.

### Code review instructions
- Review `web/src/styles/base.css`, the `html` rule near the top of the file.
- Confirm `web/src/index.css` still imports `base.css` in the expected order.
- Run `pnpm --dir web check` and `pnpm --dir web build`.

### Technical details
- Current declaration: `-webkit-font-smoothing: none`.
- Proposed declaration: `-webkit-font-smoothing: antialiased`.
- Deliberately unchanged: `font-smooth: never`, `image-rendering: pixelated`, font-family stacks, font sizes, weights, and all component styles.

## Step 2: Apply the one-line WebKit smoothing fix and run the first validation

The scoped implementation changed only the global WebKit smoothing value in `web/src/styles/base.css`: `none` is now `antialiased`. This directly removes the unsmoothed glyph mode that causes the reported pixelation while leaving image rendering and the rest of the retro design system untouched.

The first validation run exposed an unrelated local dependency-installation problem. Both the type check and production build reach the existing MathJax integration and fail because the checked-in dependencies are not installed under `web/node_modules/@mathjax`; the CSS edit itself is not implicated. I preserved the exact diagnostics below and will retry after a frozen-lockfile install before deciding whether any issue remains.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Implement the isolated CSS change and validate the frontend.
**Inferred user intent:** Fix WebKit font appearance without expanding scope into the unrelated MathJax dependency issue.
**Commit (code):** Not yet; implementation validation is in progress.

### What I did
- Changed `web/src/styles/base.css:11` from `-webkit-font-smoothing: none` to `-webkit-font-smoothing: antialiased`.
- Ran `pnpm --dir web check`.
- Ran `pnpm --dir web build`.
- Verified the source now contains one `-webkit-font-smoothing: antialiased` declaration and that `image-rendering: pixelated` remains unchanged.
- Checked the local dependency tree and confirmed `web/node_modules/@mathjax` is absent even though both packages are declared in `web/package.json` and `web/pnpm-lock.yaml`.

### Why
- `antialiased` is the smallest deterministic WebKit-specific correction for the current `none` setting.
- A frozen-lockfile install is the appropriate recovery for a missing local dependency when the manifest and lockfile already agree; changing application code to work around it would be unrelated scope.

### What worked
- The CSS edit applied cleanly and the source-level declaration check matches the intended state.

### What didn't work
- `pnpm --dir web check` failed with TypeScript `TS2307: Cannot find module` errors for `@mathjax/src/...` and `@mathjax/mathjax-newcm-font/...` imports in `web/src/lib/mathjax.ts`.
- `pnpm --dir web build` failed during Vite/Rollup resolution with: `[vite]: Rollup failed to resolve import "@mathjax/mathjax-newcm-font/js/svg/dynamic/accents.js" from "/home/manuel/code/wesen/go-go-golems/publish-vault/web/src/lib/mathjax.ts"`.
- These failures occur because `web/node_modules/@mathjax` is missing locally; they predate and are independent of the one-line CSS change.

### What I learned
- The local checkout has a stale or incomplete `web/node_modules` tree: the MathJax packages are present in the lockfile but not linked into `node_modules`.
- The existing repository diary for the MathJax ticket records the same class of local environment failure, reinforcing that dependency restoration is the correct next diagnostic step rather than a CSS change.

### What was tricky to build
- The requested change is visually motivated, but the first available automated validation path traverses unrelated MathJax imports. Separating the environment failure from the CSS change requires checking both the exact build diagnostic and the filesystem/package-lock state.

### What warrants a second pair of eyes
- Confirm the final rendered glyphs in a WebKit browser after dependencies are restored; automated compilation only proves that the CSS is accepted and bundled.
- Confirm that the intentional image pixelation rule remains appropriate for retro image assets; it is explicitly out of scope for this font fix.

### What should be done in the future
- If the frozen-lockfile install does not restore the missing packages, investigate the local pnpm store or report the environment issue separately; do not broaden this ticket into MathJax dependency maintenance.

### Code review instructions
- Review the one-line diff in `web/src/styles/base.css`.
- Re-run `pnpm --dir web install --frozen-lockfile`, then `pnpm --dir web check` and `pnpm --dir web build`.
- Use a WebKit-capable browser to compare the `html` computed `-webkit-font-smoothing` value and visual glyph edges.

### Technical details
- Source before: `-webkit-font-smoothing: none`.
- Source after: `-webkit-font-smoothing: antialiased`.
- Failed check command: `pnpm --dir web check`.
- Failed build command: `pnpm --dir web build`.
- Unrelated missing packages: `@mathjax/src@4.1.3` and `@mathjax/mathjax-newcm-font@4.1.3`.

## Step 3: Restore the local frontend dependencies and validate the change

The missing MathJax packages were restored with the repository's existing lockfile, without changing `package.json` or `pnpm-lock.yaml`. After that environment repair, the frontend type check, unit tests, and production build all passed. The generated CSS contains `-webkit-font-smoothing:antialiased`, confirming that Vite carries the source change into the production bundle.

The production build emits existing large-chunk warnings from the MathJax/Mermaid dependency graph; those warnings are unrelated to this one-line CSS change. A WebKit browser binary is not installed in the local Playwright cache, so visual glyph comparison in WebKit remains a review/deployment check rather than a completed automated check in this environment.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Complete and validate the CSS-only fix, preserving unrelated worktree files.
**Inferred user intent:** Have a small reviewable commit that removes WebKit font pixelation and is backed by normal frontend checks.
**Commit (code):** Pending at this checkpoint.

### What I did
- The initial `pnpm --dir web install --frozen-lockfile` invocation reached the confirmation prompt about removing and recreating `web/node_modules` but did not proceed in the non-interactive command wrapper.
- Retried with `CI=1 pnpm --dir web install --frozen-lockfile`; it recreated `web/node_modules` from the existing lockfile and installed 496 packages, including the two missing MathJax packages.
- Ran `pnpm --dir web check`; it passed.
- Ran `pnpm --dir web exec vitest run`; 9 test files and 93 tests passed.
- Ran `pnpm --dir web build`; it passed and emitted the production bundle.
- Checked the generated CSS with `rg`; the bundle contains `-webkit-font-smoothing:antialiased`.
- Ran `git diff --check`; it passed.
- Checked the Playwright browser cache; Chromium is installed, but no WebKit executable is available.

### Why
- Reinstalling from the frozen lockfile repairs the local validation environment without changing dependency intent.
- Type checking, unit tests, production bundling, and generated-CSS inspection cover the regression's source-to-artifact path.
- The browser-specific appearance still deserves a real WebKit review because the available automated browser is Chromium and no WebKit runtime is installed.

### What worked
- `pnpm --dir web check`: passed with no diagnostics.
- `pnpm --dir web exec vitest run`: 9 files / 93 tests passed.
- `pnpm --dir web build`: passed after 17.76 seconds; Vite transformed 4,106 modules and emitted the CSS bundle containing the new value.
- `git diff --check`: passed.

### What didn't work
- First install attempt: `pnpm --dir web install --frozen-lockfile` displayed `The modules directory at "/home/manuel/code/wesen/go-go-golems/publish-vault/web/node_modules" will be removed and reinstalled from scratch. Proceed? (Y/n)` and did not complete in the non-interactive wrapper. Retried with `CI=1`, which succeeded.
- No WebKit visual run was possible because the Playwright WebKit browser is not installed locally.
- The build retained existing warnings about chunks larger than 500 kB after minification; no new source warning was introduced by this change.

### What I learned
- The CSS change is accepted by the frontend toolchain and is present in the generated artifact, so the initial validation failure was only a stale dependency tree.
- The production build's large-chunk warning is part of the existing MathJax/Mermaid bundle shape and is not a reason to alter this focused ticket.

### What was tricky to build
- The main implementation was intentionally simple; the only operational wrinkle was that a stale `node_modules` directory masked the normal build until it was recreated with the frozen lockfile.

### What warrants a second pair of eyes
- Check the rendered text in Safari or another WebKit browser, especially at the 14px base size, to ensure grayscale antialiasing gives the intended balance between readability and the retro aesthetic.
- Review that retaining `image-rendering: pixelated` does not affect text indirectly through a component-specific transform; no such coupling was found in the source search.

### What should be done in the future
- If visual QA confirms the result, consider documenting the intentional distinction between text smoothing and pixel-art image rendering in the base stylesheet comments. This is optional and outside the fix itself.

### Code review instructions
- Review commit's one-line source change in `web/src/styles/base.css`.
- Run `pnpm --dir web check`.
- Run `pnpm --dir web exec vitest run`.
- Run `pnpm --dir web build` and inspect the emitted CSS for `-webkit-font-smoothing:antialiased`.
- Perform a manual Safari/WebKit visual check when available.

### Technical details
- Dependency recovery: `CI=1 pnpm --dir web install --frozen-lockfile`.
- Successful validation: `pnpm --dir web check`; `pnpm --dir web exec vitest run`; `pnpm --dir web build`; `git diff --check`.
- Test result: 93 frontend tests passed across 9 files.
- Artifact result: Vite emitted `dist/assets/main-BoV9Hi73.css` with `-webkit-font-smoothing:antialiased`.
- Known build note: existing chunks above 500 kB after minification.

## Step 4: Repair ticket relations and prepare the completion checkpoint

The implementation commit `60bd168` passed the repository pre-commit `web-check` hook and contains the CSS fix plus the diary and task/changelog updates. The first post-commit docmgr doctor run found two invalid related-file URIs because the manually authored frontmatter used `repo://publish-vault/...` while this workspace's repository-relative convention is `repo://web/...`. I corrected those entries, retained the valid implementation relation, added the correct stylesheet entry-point relation, and reran doctor successfully.

All three implementation tasks are now checked. The remaining completion work is documentation state: update the ticket's final summary/status, close it, rerun ticket hygiene, and commit that final bookkeeping separately from the implementation milestone.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Finish the ticket with clean docmgr metadata and a separate completion checkpoint after the implementation commit.
**Inferred user intent:** Leave a reviewable, searchable ticket rather than only a source-code change.
**Commit (code):** `60bd168` — "fix(web): smooth fonts in WebKit".

### What I did
- Reviewed the implementation commit and confirmed the pre-commit hook ran `pnpm --dir web check` successfully.
- Ran `docmgr doctor --ticket PV-WEBKIT-FONT-001 --stale-after 30`.
- Recorded and corrected the two invalid `repo://publish-vault/...` related-file entries.
- Added valid `repo://web/src/index.css` relation metadata for the stylesheet import entry point.
- Reran `docmgr doctor --ticket PV-WEBKIT-FONT-001 --stale-after 30`; all checks passed.
- Confirmed all tasks are checked in `tasks.md`.

### Why
- Related-file metadata must resolve to real repository paths so future agents can navigate from the diary to source evidence.
- Separating completion bookkeeping from the implementation commit keeps the CSS change independently reviewable while preserving a clean ticket history.

### What worked
- Docmgr hygiene now reports: `✅ All checks passed`.
- The implementation commit is focused on the requested CSS fix and its ticket evidence.

### What didn't work
- The first doctor run reported two warnings: `missing_related_file` for `repo://publish-vault/web/src/index.css` and `repo://publish-vault/web/src/styles/base.css`. Both were metadata path-prefix errors, not missing source files. Removing the invalid entries and re-adding repository-relative `repo://web/...` entries resolved them.

### What I learned
- In this repository, docmgr's `repo://` URI is rooted at the repository contents, so paths under `web/` must use `repo://web/...`, not the repository name as an extra path segment.

### What was tricky to build
- The source change was trivial; preserving high-quality ticket navigation required checking the generated relation paths against docmgr's actual URI resolver rather than assuming the visible repository name belonged in every URI.

### What warrants a second pair of eyes
- Review the final ticket status and summary after closing, and ensure the implementation commit remains the only source-code change for this issue.
- Perform manual WebKit visual QA when a Safari/WebKit runtime is available; this remains the only unexecuted validation item.

### What should be done in the future
- If this ticket is revisited for visual QA, append a new diary step with the browser/version, viewport, computed style, and before/after observation rather than rewriting this history.

### Code review instructions
- Start at `60bd168` and inspect `web/src/styles/base.css`.
- Read this diary's Steps 1–4 for scope, failure diagnostics, validation evidence, and metadata correction.
- Run `docmgr doctor --ticket PV-WEBKIT-FONT-001 --stale-after 30`.

### Technical details
- Implementation commit: `60bd168`.
- Valid related URIs: `repo://web/src/styles/base.css`, `repo://web/src/index.css`.
- Final pre-close doctor result: all checks passed.
- Outstanding non-automated check: manual WebKit visual review.

## Step 5: Close the ticket after the completion audit

The ticket is complete: the CSS fix is implemented in `60bd168`, all three tasks are checked, the frontend validation suite passed after restoring dependencies, and docmgr reports no findings. The ticket was closed with status `complete`, while the diary intentionally preserves the limitation that no local WebKit runtime was available for a visual screenshot check.

This is a documentation-only completion checkpoint after the focused implementation commit. Unrelated pre-existing untracked files (`.claude/`, `.playwright-mcp/`, `search-memory-results.png`, and `ttmp/vocabulary.yaml.orig`) remain untouched and are not part of this ticket.

### Prompt Context
**User prompt (verbatim):** see Step 1
**Assistant interpretation:** Finish the requested CSS fix and leave the new docmgr ticket closed with an evidence-backed detailed diary.
**Inferred user intent:** Have the WebKit readability issue addressed with a concise code change and an auditable work record.
**Commit (code):** `60bd168` — "fix(web): smooth fonts in WebKit".

### What I did
- Ran `docmgr ticket close --ticket PV-WEBKIT-FONT-001` with a completion-specific changelog entry.
- Updated the ticket index summary, overview, and status text to match the closed frontmatter.
- Updated this diary's document status to `complete`.
- Reran `docmgr doctor --ticket PV-WEBKIT-FONT-001 --stale-after 30`; it reported `✅ All checks passed`.
- Reviewed `git status` to ensure unrelated untracked files were not staged.

### Why
- Closing the ticket only after every explicit task is checked and the hygiene audit is clean provides a defensible completion boundary.
- Calling out the unavailable WebKit runtime prevents the validation record from implying visual evidence that was not collected.

### What worked
- Ticket close succeeded and set the index status from `active` to `complete`.
- Final docmgr doctor run passed with no findings.
- The final ticket has a source relation, a detailed diary, checked tasks, a changelog, and a concise implementation summary.

### What didn't work
- Manual WebKit visual validation remains unavailable locally because the Playwright cache has no WebKit browser executable. This is a known validation limitation, not a failed implementation check.

### What I learned
- The smallest safe change was indeed a single CSS declaration; the surrounding ticket work was mainly evidence capture, dependency restoration for validation, and metadata hygiene.

### What was tricky to build
- Maintaining an accurate completion claim required distinguishing “all automated and source-to-bundle checks passed” from “visual WebKit QA performed.” The ticket is complete for the requested CSS implementation, but the latter remains explicitly open as optional follow-up.

### What warrants a second pair of eyes
- Review `60bd168` and the final ticket metadata.
- If visual assurance is required for release, inspect the page in Safari/WebKit and append the browser-specific result to this diary.

### What should be done in the future
- Perform and record Safari/WebKit visual QA when a compatible runtime is available; no code change is currently indicated by the automated evidence.

### Code review instructions
- Review `web/src/styles/base.css` line 11 and commit `60bd168`.
- Read the ticket index and this diary for the complete scope and validation record.
- Re-run `pnpm --dir web check`, `pnpm --dir web exec vitest run`, `pnpm --dir web build`, and `docmgr doctor --ticket PV-WEBKIT-FONT-001 --stale-after 30`.

### Technical details
- Closed ticket: `PV-WEBKIT-FONT-001`.
- Implementation commit: `60bd168`.
- Completion audit: 3/3 tasks checked; 9 test files and 93 tests passed; type check, build, generated CSS check, diff check, and docmgr doctor passed.
- Remaining evidence gap: no local WebKit executable for visual comparison.
