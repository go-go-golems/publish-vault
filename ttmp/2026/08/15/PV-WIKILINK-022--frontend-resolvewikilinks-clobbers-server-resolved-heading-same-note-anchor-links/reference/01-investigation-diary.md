---
Title: Investigation diary
Ticket: PV-WIKILINK-022
Status: active
Topics:
    - wiki-link
    - parser
    - frontend
    - react
    - html-rendering
    - regression
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/parser.go
      Note: wikiLinkHTML target-less branch, resolveSelfHeadingLinks, BuildHeadingIndex (server-side resolution)
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: handleClick + the post-hydration resolveWikiLinks effect (the live render path)
    - Path: repo://web/src/lib/wikiLinks.test.ts
      Note: regression test pinning the wiki-link-self skip policy
    - Path: repo://web/src/lib/wikiLinks.ts
      Note: |-
        resolveWikiLinks — the function that clobbered self-heading links; now guarded by wikiLinkTargetForValidation
        the clobber and the guard (commit f25d167)
ExternalSources:
    - https://parc.yolo.scapegoat.dev/note/research/software-architecture-garden/coinvault/index-of-design-patterns
Summary: 'Same-note [[#Heading]] links were clobbered by the frontend resolveWikiLinks: it treated their empty data-target as a missing note and overwrote the server-resolved href with "#". Fixed by skipping wiki-link-self anchors; verified 952/952 broken -> 0 on the real note.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Diary for the PV-WIKILINK-022 fix
WhenToUse: Read before changing wiki-link resolution, the wiki-link-self class, or the frontend slug-set validation
---


# Investigation diary

## Goal

Find why the local anchor links on the published CoinVault "Index of Design
Patterns" note are broken, fix the cause, and verify the fix end-to-end on the
real vault note. This diary records the investigation and the fix for review.

## Step 1: Locate the root cause — the frontend resolver clobbers server-resolved self anchors

The user reported that the "notation table" link (and, by their reading, "all
the doc local anchor links") on the live note was broken. The note is a
back-of-the-book index whose every `See` / `see also` is a `[[#Heading]]`
same-note link, so a single class of failure could break hundreds of links at
once. The first question was whether the links were broken in the **server
output** or only after the **React frontend** hydrated them.

### Prompt Context

**User prompt (verbatim):** "on https://parc.yolo.scapegoat.dev/note/research/software-architecture-garden/coinvault/index-of-design-patterns (path here locally: /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/Software Architecture Garden/coinvault/Index of Design Patterns.md)  , some of the relative links are broken (for example "notation table", I think it's all the doc local anchor links), Create a new docmgr ticket to address the issue, and Create  a detailed analysis / design / implementation guide that is for a new intern..."

**Assistant interpretation:** Investigate the broken same-page anchor links on the published note, create a docmgr ticket, write an intern-facing analysis/design/implementation guide, fix the issue, keep a diary, commit at intervals, and upload the guide to reMarkable.

**Inferred user intent:** The published index is unreadable because its index cross-references don't jump; the user wants the cause nailed down, a ticket opened, an intern guide that teaches the system, and the actual bug fixed.

### What I did

- Read the note source: `[[#Identity strings, schemas, and budgets|notation table]]` and ~950 more `[[#Heading]]` links.
- Confirmed the prior ticket `PV-WIKILINK-021` already made the **server** resolve `[[#Heading]]` via `resolveSelfHeadingLinks` in `internal/parser/parser.go`.
- Fetched the live SSR HTML and the `/api/notes/<slug>` JSON `html` field. Both already contained the **resolved** form:
  `<a href="#identity-strings-schemas-and-budgets" class="wiki-link wiki-link-self" ...>notation table</a>` — 238 `href="#…"` anchors, **0** `#unresolved-…`. So the server was correct.
- Loaded the live page in Playwright and inspected the DOM. The same anchor had become:
  `<a href="#" class="wiki-link wiki-link-self broken" ... title="Note not found: ">notation table</a>`. Counted **952/952** `wiki-link-self` links marked `broken` with `href="#"`; **0/596** cross-note links broken.
- Read the only place that adds the `broken` class and rewrites `href` to `"#"`: `web/src/lib/wikiLinks.ts` `resolveWikiLinks`. It selected `a.wiki-link` (which includes `wiki-link-self`), read `data-target ?? ""`, and — because self links carry **no** `data-target` — `slugSet.has("")` was false, so it marked them broken and overwrote the resolved href.

### Why

The server and the frontend disagree about what "a wiki link" is. The server
treats `[[#Heading]]` as a special `wiki-link-self` anchor with no target; the
frontend resolver treated every `a.wiki-link` as a cross-note link that must
carry a resolvable target. The empty target failed the check and the resolver
destroyed the href the server had carefully computed.

### What worked

- Fetching both the SSR HTML and the API JSON proved the server was correct, narrowing the bug to the client.
- The Playwright DOM inspection gave exact counts (952 broken self-links, 0 broken cross-note links) that scoped the bug to one class of link.
- Reading `resolveWikiLinks` (20 lines) immediately explained the symptom: `data-target ?? ""` → `slugSet.has("")` → clobber.

### What didn't work

- Nothing failed, but the first `pkill -f "go run ./cmd/retro-obsidian-publish …"` killed my own shell because the pattern appeared literally in the command line. Fixed with the `serv[e]` bracket trick so the regex matches the server but not the kill command itself.

### What I learned

- The `wiki-link-self` class is the discriminator the backend already provides; the frontend resolver just wasn't told to honor it. The fix is a one-line policy, not a rendering change.
- PV-WIKILINK-021 fixed the backend `[[#Heading]]` path but its "Open follow-ups" never wired the frontend resolver, so the fix shipped half-complete: correct HTML left the server and was then clobbered on the client.

### What was tricky to build

Nothing about the fix was tricky; the tricky part was **not being fooled by the
server output**. The SSR HTML and the API JSON both looked correct, so a
server-only inspection would have closed the ticket as "works for me." The bug
only appears after React hydration, in the browser DOM, which is why a
Playwright check was necessary rather than a `curl`.

### What warrants a second pair of eyes

- Confirm the `wiki-link-self` skip does not hide a genuinely broken self link. The server already marks unresolved self links `broken` with `href="#unresolved-…"`; the frontend skip preserves that verdict rather than re-litigating it — verify that is the desired behavior.
- The click path for self links falls through to the browser's native hash navigation (no `preventDefault`); confirm smooth-scroll within the `ScrollArea` container is acceptable vs. matching `enhanceHeadingAnchors`' smooth scroll.

### What should be done in the future

- Add a DOM-level test (needs `happy-dom`/`jsdom`, which the project doesn't depend on yet) asserting the resolved `href` survives `resolveWikiLinks`. The current node-env test pins the policy only.
- Wire `pnpm --dir web test` into CI (`ci-check` / a workflow) so vitest regressions actually run in CI; today only `pnpm --dir web check` and `build` run.

### Code review instructions

- Start at `web/src/lib/wikiLinks.ts` — `wikiLinkTargetForValidation` + `resolveWikiLinks`.
- Run `pnpm --dir web check && pnpm --dir web exec vitest run` (26 tests pass).
- For end-to-end proof, see Step 2.

### Technical details

Live DOM (before fix), the "notation table" link:
`<a href="#" class="wiki-link wiki-link-self broken" data-heading="Identity strings, schemas, and budgets" data-alias="notation table" title="Note not found: ">notation table</a>`

Server output (already correct), same link:
`<a href="#identity-strings-schemas-and-budgets" class="wiki-link wiki-link-self" data-heading="Identity strings, schemas, and budgets" data-alias="notation table">notation table</a>`

Counts on the live page: 1548 total `a.wiki-link`; 952 `wiki-link-self` (all broken); 596 cross-note (0 broken).

## Step 2: Implement the guard, test, and verify end-to-end on the real vault

With the cause pinned to `resolveWikiLinks`, the fix is to make the slug-set
check skip same-note heading links. I extracted the skip decision into a pure,
named, testable function (`wikiLinkTargetForValidation`) so the policy is
explicit and unit-testable in the node environment, then rebuilt the web bundle
and verified against the real `go-go-parc` vault in a browser.

### Prompt Context

**User prompt (verbatim):** (see Step 1) — same task, continuing into the fix and verification.

**Assistant interpretation:** Implement the minimal correct fix, add a regression test, and prove it works on the actual reported note by building and serving the vault locally.

**Inferred user intent:** The guide should be evidence-backed, so the fix must actually work on the real note, not just in theory.

### What I did

- In `web/src/lib/wikiLinks.ts`, added `wikiLinkTargetForValidation(el)`: returns `null` when `el.classList.contains("wiki-link-self")`, else `data-target ?? ""`. Rewrote `resolveWikiLinks` to skip when it returns `null`. Cross-note behavior is unchanged.
- Added `web/src/lib/wikiLinks.test.ts` (vitest, node env): 4 cases for the policy + 1 for the no-DOM guard. No new DOM dependency was needed because the policy is pure.
- Ran `pnpm --dir web check` (tsc clean) and `pnpm --dir web exec vitest run` (26 pass, 5 new).
- `pnpm --dir web build`, then `go run ./cmd/retro-obsidian-publish serve --vault /home/manuel/code/wesen/go-go-golems/go-go-parc --port 8080 --watch=false --serve-web` (serves the freshly built `web/dist`).
- Loaded `http://127.0.0.1:8080/note/research/software-architecture-garden/coinvault/index-of-design-patterns` in Playwright and re-ran the DOM audit, then clicked the "notation table" link.

### Why

Extracting the policy into a pure function is preferable to an inline `if`
because the project's test runner is node-only (no `DOMParser`), so a pure
helper is the only way to get a real regression test without adding a DOM
dependency. Naming the policy also makes the intern guide's "why" self-evident
in the code.

### What worked

- After the fix, the browser DOM audit returned: **952 self-links, 0 broken**; the "notation table" anchor is `<a href="#identity-strings-schemas-and-budgets" class="wiki-link wiki-link-self" …>` (no `broken`, no `title`), and `document.getElementById("identity-strings-schemas-and-budgets")` exists.
- Clicking "notation table": `window.location.hash` became `#identity-strings-schemas-and-budgets` and the heading scrolled from `top: 14488` to `top: 28` (in view). Screenshot saved in `sources/pv-wikilink-022-fixed-notation-table-jumps-to-heading.png`.
- Committed as `fix(PV-WIKILINK-022)` (commit `f25d167`); lefthook `web-check` (tsc) passed on commit.

### What didn't work

- The first server-start poll gave up before the HTTP listener was ready: `go run` compiles (~15s), then loads 1812 notes (~25s), then builds the in-memory Bleve index (~30s) before listening. I re-polltled for the listener specifically rather than the vault-load log line.

### What I learned

- End-to-end verification on the real vault is cheap insurance: the server output looked correct all along, so only a browser check could prove the client regression was gone.
- The Bleve index build, not the vault load, is the slow part of startup for a 1812-note vault; the "Server listening" log line is the true readiness signal.

### What was tricky to build

- None of the code. The only sharp edge was `pkill -f` self-matching (see Step 1).

### What warrants a second pair of eyes

- The pure-helper test pins the policy but not the DOM wiring; a future DOM test (happy-dom) would close that gap.
- Native hash-scroll vs. `enhanceHeadingAnchors` smooth-scroll consistency (noted in Step 1).

### What should be done in the future

- Deploy the rebuilt bundle to `parc.yolo.scapegoat.dev` so the live site picks up the fix (out of scope for this ticket; the bundle must be rebuilt and the image redeployed).
- Add a DOM-level test and wire vitest into CI (see Step 1).

### Code review instructions

- `git show f25d167` — the fix + test.
- Reproduce: `pnpm --dir web build && go run ./cmd/retro-obsidian-publish serve --vault <vault> --port 8080 --watch=false`, open the note, confirm `a.wiki-link-self` anchors are not `broken` and jump on click.

### Technical details

Fix (essence):

```ts
export function wikiLinkTargetForValidation(el: WikiLinkLike): string | null {
  if (el.classList.contains("wiki-link-self")) return null;
  return el.getAttribute("data-target") ?? "";
}
// in resolveWikiLinks:
const target = wikiLinkTargetForValidation(el);
if (target === null) return;          // same-note heading link: leave it alone
if (!slugSet.has(target)) { /* mark broken cross-note link */ }
```

Verification command sequence:

```bash
pnpm --dir web check
pnpm --dir web exec vitest run
pnpm --dir web build
go run ./cmd/retro-obsidian-publish serve --vault /home/manuel/code/wesen/go-go-golems/go-go-parc --port 8080 --watch=false --serve-web
# then in a browser: /note/research/software-architecture-garden/coinvault/index-of-design-patterns
```
