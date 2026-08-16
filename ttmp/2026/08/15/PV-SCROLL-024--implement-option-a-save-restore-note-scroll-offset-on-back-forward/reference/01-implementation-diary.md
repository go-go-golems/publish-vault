---
Title: Implementation diary
Ticket: PV-SCROLL-024
Status: active
Topics:
    - frontend
    - react
    - ux
    - hydration
    - ssr
    - wiki-link
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: |-
        existing heading effect; hash precedence must be preserved
        heading effect; hash precedence
    - Path: web/src/components/pages/NotePage/NotePage.tsx
      Note: |-
        hook wiring target; owns the ScrollArea; isLoading unmounts the scroller
        isLoading unmounts the ScrollArea — drives hook placement
    - Path: web/src/lib/scrollRestoration.test.ts
      Note: |-
        vitest regression test for pickScrollAction
        scrollKeyOf + pickScrollAction tests
    - Path: web/src/lib/scrollRestoration.ts
      Note: |-
        new module to implement (pickScrollAction + useScrollRestoration + findVisibleScroller)
        3 review fixes + scroller-discovery fix (commit 2e2ae67)
ExternalSources:
    - https://developer.mozilla.org/en-US/docs/Web/API/History/scrollRestoration
Summary: 'Diary for PV-SCROLL-024: implement Option A scroll restoration in persistent NotePage with a scroll-listener capture + rAF restore. Step 1 records ticket creation and the design, including the NotePage-loading-unmount correction.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Diary for the PV-SCROLL-024 implementation
WhenToUse: Read before resuming work on the scroll-restoration implementation
---




# Implementation diary

## Goal

Implement Option A from PV-SCROLL-023: hand-rolled save/restore of the note
scroll offset on browser back/forward, preserving the app-shell layout. Keep a
diary, commit at intervals, add a regression test, and verify end-to-end on the
real vault.

## Step 1: Create ticket and design; correct the scroller-lifecycle premise

The user chose Option A and asked for a new ticket, a design doc, and a
task-by-task implementation with commits and a diary. Before writing code I had
to nail down where the hook lives, and that overturned a second premise from
the PV-SCROLL-023 analysis.

### Prompt Context

**User prompt (verbatim):** "option A. Create a new docmgr ticket and a design doc and ten implement the tasks, one by one, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"

**Assistant interpretation:** Create PV-SCROLL-024, write a design doc for Option A, then implement the scroll save/restore task by task, committing at intervals and keeping a diary in the diary skill format.

**Inferred user intent:** Ship the back/forward scroll-restore behavior now via the low-risk Option A (preserve the app shell), with the same rigor as PV-WIKILINK-022 (ticket + design + test + diary + end-to-end proof).

### What I did

- Re-read the corrected PV-SCROLL-023 design (real scroller is the `ScrollArea`, fix in `NotePage`).
- Read `NoteBody.tsx` (contentRef attaches to the `.note-prose` div) and re-read `NoteHtml.tsx` (owns the `#heading` effect) and `NotePage.tsx`.
- Found the critical lifecycle fact the PV-SCROLL-023 design missed: `NotePage`'s `isLoading` branch (`NotePage.tsx:88-95`) returns a loading spinner, so `NoteView`/`NoteHtml` and the article `ScrollArea` **unmount** while a note fetches. The scroller is therefore NOT a stable DOM node across navigations to uncached notes.
- Realized this breaks a naive "read scrollTop on location change" save: on forward navigation to an uncached note, the source `ScrollArea` is already gone when the location effect runs, so the source offset is lost — and the user's core complaint (Back returns to where I was) depends on that save.
- Designed the fix: hook lives in **persistent `NotePage`** (the `/note/*` route element, mounted through loading); a passive **scroll listener** captures the offset continuously into a persistent `ref` before the scroller unmounts; `useLayoutEffect` save (reads the ref) runs before restore; restore re-binds to the scroller when `ready` flips true and applies the offset via a rAF poll.
- Created ticket `PV-SCROLL-024` with a design-doc and a diary. Wrote the design doc (pure policy `pickScrollAction`, the `useScrollRestoration(containerRef, ready)` hook, `findVisibleScroller`, wiring in `NotePage`, precedence, decision records, test plan).

### Why

Getting the lifecycle right before coding avoids the PV-WIKILINK-022 trap in
reverse: there, the server output looked correct and the bug was in the
measured DOM. Here, the code looked like the scroller was stable (`h-full`
chain), but the `isLoading` early return unmounts it — only reading
`NotePage.tsx` end-to-end revealed it. A hook in `NoteHtml` would have silently
lost the save on every uncached forward navigation.

### What worked

- Reading `NotePage.tsx`'s early-return order (`if (!slug)` → `if (isLoading)` → `if (isError)`) before deciding where the hook lives: the loading branch unmounts the scroller, so the hook must be above the early returns and in the persistent component.
- Reusing the PV-WIKILINK-022 "extract the pure policy" pattern: `pickScrollAction` is testable in the node env without a DOM, since the web test runner is `environment: "node"`.

### What didn't work

- Nothing yet (design phase).

### What I learned

- `NotePage` is the persistent `/note/*` route element; it stays mounted across slug changes and across the loading state, which makes it the correct owner for cross-navigation state (the saved-offsets map and the last-offset ref).
- The scroller is a child that comes and goes; the hook must re-bind to it each time it (re)mounts, keyed on `ready = !!note`.
- `history.scrollRestoration = "manual"` is the right global declaration: the browser's `"auto"` is a no-op here (window never scrolls), so we fully own restoration.

### What was tricky to build

The save/restore timing is the sharp edge. Save must read the offset **before**
the source scroller unmounts (on forward-uncached nav) — solved by a passive
scroll listener writing to a persistent ref. Restore must run **after** the
target scroller re-mounts with its content — solved by keying the bind effect
on `ready` and waiting via rAF until `scrollHeight >= offset`. Ordering save
before restore within one commit is enforced by making both `useLayoutEffect`
and placing save above restore.

### What warrants a second pair of eyes

- The `findVisibleScroller` visibility filter (`clientHeight > 0`): desktop and mobile layouts both render, one hidden via `md:` classes (clientHeight 0). Confirm the visible branch is selected on both viewports.
- The flash risk: one frame may paint at `scrollTop=0` before the rAF restore. Acceptable for this ticket; gate content visibility only if it's noticeable.

### What should be done in the future

- Implement `scrollRestoration.ts` + test, wire into `NotePage`, verify tsc/vitest/build, then end-to-end on the real vault (Steps 2-4).
- Optional follow-up: `sessionStorage` persistence for reload-restoration; smooth-scroll consistency with heading permalinks.

### Code review instructions

- Design: `design-doc/01-design-hand-rolled-scroll-restoration-option-a.md` §4 (pure policy), §5 (hook), §8 (wiring).
- Verify the lifecycle premise yourself: in `NotePage.tsx`, confirm the `isLoading` branch returns before the `desktopLayout`/`mobileLayout` that hold the `ScrollArea`.

### Technical details

The decisive fact: `NotePage.tsx:88-95` —
```tsx
if (isLoading) {
  return (<div …>Loading note…</div>);
}
```
This early return means the `ScrollArea` (rendered only in `desktopLayout`/`mobileLayout` below) is absent from the DOM during loading. Therefore the scroller is not stable; the hook must live in persistent `NotePage` and capture offsets via a scroll listener before unmount.

## Step 2: Library + tests (committed) and the innerHTML-rewrite bug (found, not yet fixed)

Implemented the library and wired it into NotePage. The build/tests pass, but
the end-to-end browser test failed for a real reason that the design did not
anticipate: NoteHtml's resolveWikiLinks rewrites NoteBody innerHTML AFTER my
restore, resetting scrollTop to 0.

### Prompt Context

**User prompt (verbatim):** (see Step 1) — continuing the task-by-task implementation.

**Assistant interpretation:** Implement the library + test, wire into NotePage, then verify end-to-end on the real vault.

**Inferred user intent:** Ship working back/forward scroll restoration, proven on the real note, not just passing unit tests.

### What I did

- Wrote `web/src/lib/scrollRestoration.ts`: `pickScrollAction` (pure), `useScrollRestoration(containerRef, ready)`, `findVisibleScroller`. tsc clean.
- Wrote `web/src/lib/scrollRestoration.test.ts` (8 cases, node env). All pass.
- Committed library (`11e842f`) — lefthook `web-check` passed.
- Wired into `NotePage.tsx`: `useScrollRestoration(layoutRef, !!note)` before the early returns; wrapped the desktop+mobile layout in `<div ref={layoutRef}>`; added `note-scroll` class to the three article `ScrollArea`s. tsc clean, vitest 34 pass, build OK. Committed (`9b84e01`).
- End-to-end test on the real vault (CoinVault index → README → Back):
  - Set scroller.scrollTop = 3500; scroll listener captured it (confirmed 3500 after 150ms).
  - Clicked cross-note link → README loaded at top (correct fresh-forward behavior).
  - Pressed Back → index reloaded but scrollTop = 0, NOT 3500. **Restore failed.**
  - Verified `history.scrollRestoration === "manual"` (hook mount effect ran) and `resolveWikiLinks` ran (952 self-links, 0 broken → NoteBody innerHTML was rewritten).

### Why it failed (root cause)

The save/restore timing is correct relative to the scroller's mount, but NOT
relative to the content's final settle. On Back, the sequence is:

1. NotePage mounts; scroller (`.note-scroll`) exists.
2. My bind/restore effect runs: rAF loop sets `scroller.scrollTop = 3500`.
3. NoteHtml's `resolveWikiLinks` effect runs (also keyed on `html`/`slugSet`):
   `setResolvedHtml(resolveWikiLinks(html, slugSet))` → NoteBody's
   `dangerouslySetInnerHTML` changes → React rewrites `innerHTML` → **the
   browser resets the scroller's `scrollTop` to 0** (a content swap resets
   scroll position).
4. My rAF loop had already exited (step 2 set it once and returned), so nothing
   re-applies 3500.

The design assumed the content was stable when the restore ran. It is not:
`resolveWikiLinks` is a *client-side* rewrite that happens after first paint,
and it destroys scroll position. This is exactly the PV-WIKILINK-022-era
"server output looks correct but the client clobbers it" pattern, one layer
up: here the client's own content-finalization clobbers the scroll restore.

### What didn't work

- Naive rAF-then-set restore: it runs before `resolveWikiLinks` finalizes, so the
  set is undone by the innerHTML rewrite.

### What I learned

- The restore must be keyed on the *final* content, not just the scroller's
  presence. `NoteHtml` is where `resolvedHtml` is known; `NotePage`'s hook does
  not know when content is finalized.
- Adding `useLocation` to `NoteHtml` would break its Storybook stories (they
  are not wrapped in a Router — confirmed in `NoteHtml.stories.tsx`).

### Path forward (not yet implemented)

Keep the save logic + scroll listener + saved-offsets map in the persistent
`NotePage` hook, but export a module-level store so `NoteHtml` can trigger the
restore once its content is finalized. Two viable approaches:

1. **Restore in NoteHtml, keyed on `resolvedHtml`**: `NoteHtml` reads the saved
   offset for the current `location.key` from a shared module-level store and
   applies it in an effect on `[resolvedHtml]` (after the innerHTML rewrite).
   NotePage's hook still owns save + the `lastOffset` ref + `manual` declaration.
   This keeps the pure policy testable and avoids a Router in NoteHtml stories
   (NoteHtml would receive the saved offset via a prop or a context, OR use a
   tiny module-level `getSavedOffset(key)` helper that NotePage populates).
2. **MutationObserver in the hook**: observe the scroller's subtree; on
   mutation that resets `scrollTop` below the saved offset (within the restore
   window), re-apply. More robust to any future content-finalization pass, but
   more complex and easier to fight user scroll.

Chosen direction: **Approach 1** (restore after `resolvedHtml` in `NoteHtml`),
because it keys on the actual content-stable signal rather than guessing with a
timer/observer. Needs a module-level store (e.g. `scrollStore` in
`scrollRestoration.ts` with `save(key,offset)`/`get(key)`/`use(key)`) shared by
the NotePage hook (writer) and the NoteHtml restore (reader).

### What warrants a second pair of eyes

- The shared-store approach introduces module-level mutable state; ensure it is
  keyed by `location.key` and cleared sensibly (or just grows boundedly — a Map
  of a handful of entries per session is fine).
- Confirm Approach 1 does not re-clobber when the user scrolls after restore:
  the restore effect should run once per `[resolvedHtml, location.key]` and then
  stop; user scroll updates `lastOffset` via the listener, so a later
  save captures the user's position, not the restored one.

### What should be done in the future (immediate next)

- Implement Approach 1: add a module-level store to `scrollRestoration.ts`;
  have the NotePage hook write saved offsets to it; add a restore effect in
  `NoteHtml` keyed on `[resolvedHtml]` that reads the store for the current key
  and applies the offset (respecting hash precedence). Re-run the end-to-end
  test (index→README→Back must restore ~3500).
- Then: test forward-to-cached-note restore, hash precedence, and reload→top.
- Update diary Step 3 with results; commit; final ticket bookkeeping.

### Code review instructions

- Current state: `git show 11e842f` (library) and `git show 9b84e01` (wiring).
- Reproduce the failure: `pnpm --dir web build && go run ./cmd/retro-obsidian-publish serve --vault <vault> --port 8080 --watch=false`; open the CoinVault index; in DevTools run `document.querySelectorAll('.note-scroll').forEach(e=>e.clientHeight>0&&(e.scrollTop=3500)); wait 200ms; click a /note/ link; press Back; observe scrollTop==0 (bug)`.

## Step 3: Address the 3 PR #21 review comments (committed) and the scroller-discovery trap (found and fixed)

Implemented the three GitHub review comments from the clean PR baseline,
guided by the PV-SCROLL-REVIEW-025 fundamental review. All three browser
scenarios pass: note-to-note Back, note-to-search Back, and same-note fragment
Back all restore the saved offset. A fourth issue — the scroller-discovery
trap that had regressed cross-note Back in an earlier attempt — was found
empirically and fixed by moving the scroll listener to the document capture
phase.

### Prompt Context

**User prompt (verbatim):** "Then implement the 3 fixes. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"

**Assistant interpretation:** Implement the three PR #21 review fixes from the clean baseline, verify them in the browser, commit, and record a detailed diary.

**Inferred user intent:** Ship working scroll-restoration fixes for the three review comments, proven on the real vault, without regressing the cross-note flow that an earlier attempt broke.

**Commit (code):** 2e2ae67 — "fix(PV-SCROLL-024): address 3 PR #21 review comments in scroll restoration"

### What I did

- Discarded the failed worktree experiments (`git checkout -- web/src/lib/scrollRestoration.ts web/src/lib/scrollRestoration.test.ts`) to restore the clean PR baseline (148 lines, the version with the 3 review comments).
- Applied the three fixes to `web/src/lib/scrollRestoration.ts`:
  1. **Persist outside NotePage:** replaced `useRef<Map>` with a module-level `const savedOffsets = new Map<string,number>()`; added an unmount cleanup that writes `lastOffset` to the store keyed by `lastKey`.
  2. **Save on hash-only changes:** introduced `scrollKeyOf(location) = location.key + "|" + hash`; added a `hashchange` listener that re-renders so a native heading-permalink navigation (which sets `window.location.hash` directly, bypassing React Router) updates `scrollKey`.
  3. **Wait until actually scrollable:** changed the restore predicate from `scroller.scrollHeight >= offset` to `scroller.scrollHeight - scroller.clientHeight >= offset` (the maximum valid `scrollTop`).
- Updated `web/src/lib/scrollRestoration.test.ts` to cover `scrollKeyOf` (3 tests) and re-key `pickScrollAction` tests through `scrollKeyOf` (12 tests total).
- Verified: `pnpm --dir web check` (tsc clean), `vitest run` (38 pass), `prettier --check` (clean), `pnpm --dir web build` (clean).
- Built the Go binary and served `vault-example` on :8080 via a launcher script (`/tmp/pv-launch.sh`; the binary had to be launched via a script file because direct `cmd &` backgrounding in the bash tool was reaped/killed — see What was tricky).
- Manually verified in Playwright (viewport 1200×400 to make the note scrollable, maxScroll 222): set scrollTop=150, then for each scenario navigated away and pressed Back:
  - **Scenario 1 (same-note fragment Back, comment 2):** click `#about-this-system` heading permalink → Back → restored to 150 (was 0 before the fix).
  - **Scenario 2 (note-to-search Back, comment 1):** SPA-navigate to `/search?q=welcome` (NotePage unmounts) → Back → restored to 150 (module store survived).
  - **Scenario 3 (note-to-note Back, regression check):** SPA-navigate to `zettelkasten-method` → Back → restored to 150 (no regression).

### Why

The three comments are valid symptoms of missing abstractions (storage lifetime,
entry identity, readiness). The PV-SCROLL-REVIEW-025 review diagnosed the
underlying traps so this implementation could avoid the scroller-discovery
failure that sank the earlier attempt — specifically, it predicted that
binding the listener to `containerRef` or to a not-yet-sized scroller would
fail, and that proved true (see What was tricky).

### What worked

- The module-level `savedOffsets` Map cleanly survives `NotePage` unmount; the unmount cleanup handles the no-trailing-scroll-event case.
- `scrollKeyOf` (key + hash) plus the `hashchange` re-render makes the save effect fire on native heading-permalink navigation.
- The `scrollHeight - clientHeight >= offset` predicate is the correct feasibility check (max valid `scrollTop`).
- Document-level capture-phase listener: catches `.note-scroll` scrolls regardless of which React tree renders the scroller, sidestepping both the SplitPane-sizing race and the containerRef-not-an-ancestor problem.

### What didn't work

- **`containerRef` capture listener (first attempt):** attached the scroll listener to `containerRef.current` in the capture phase. A debug probe showed `rootIsAncestor: false` — `containerRef.current` (a `div.h-full`) was NOT an ancestor of the visible `.note-scroll`. The listener never fired, `lastOffset` stayed 0, and Back returned 0. Root cause: in this app the ref points at a `div.h-full` that is not an ancestor of the visible scroller (duplicate/transitional trees under StrictMode + lazy/eager hydration in `entry-client.tsx`). Fixed by moving the listener to `document` capture phase, filtered by the `note-scroll` class.
- **Save-on-key-change racing the smooth scroll (second attempt):** with the document listener capturing `lastOffset` only, the save effect (keyed on `scrollKey` change) saved `lastOffset` AFTER the heading's smooth `scrollIntoView` had already advanced it to 222 (the bottom), overwriting the pre-jump 150. Debug showed `saved["default|"] = 222` instead of 150. Fixed by making the capture listener write to the store **continuously, keyed by the live `window.location.hash` read at event time** — so the jump's scroll events are attributed to the hashed identity and the unhashed identity keeps the reader's pre-click position.
- **`findVisibleScroller(containerRef.current)` for restore:** same ancestry problem as the listener; the scroller was not found. Fixed by querying `document.documentElement`.
- **Backgrounding the server binary:** `cmd &`, `nohup cmd &`, `( exec cmd & )`, and `setsid cmd &` all failed — the log file was never created and the process died. A plain `sleep 300 &` survived, so the reaping was specific to the server. Fixed by writing a launcher script (`/tmp/pv-launch.sh`) and running `bash /tmp/pv-launch.sh`, which detached correctly.

### What I learned

- The scroller-discovery trap the research predicted is real and reproducible: `containerRef.current` is not a reliable ancestor of the visible scroller in this app. A document-level capture listener filtered by class is the robust binding.
- Save-on-key-change alone is insufficient for smooth-scroll fragment navigation: the scroll events of the animation arrive after the key changes, so they clobber the pre-jump offset. Continuous save keyed by the *live* hash (read at event time) is required.
- The three review comments are interdependent: fixing identity (key+hash) without fixing the capture (continuous, live-hash-keyed) leaves the fragment-Back case broken, because the save races the smooth scroll.
- A launcher script is the reliable way to background a long-running server from this bash tool environment.

### What was tricky to build

The hardest part was the interaction between the `hashchange` re-render and the
heading permalink's smooth scroll. The permalink handler in
`noteEnhancements.ts:196-200` does `window.location.hash = id;
heading.scrollIntoView({ behavior: "smooth" })` — hash first, then an async
smooth scroll. The naive save model (save `lastOffset` on `scrollKey` change)
saves *after* the smooth scroll has updated `lastOffset`, so it captures the
post-jump position, not the pre-jump one. The fix is to save continuously in
the capture listener, keyed by the live `window.location.hash` at event time.
Because the hash is set *before* the scroll, the jump's scroll events carry
the hashed identity, and the unhashed identity retains the reader's pre-click
offset. This is the one invariant that makes fragment-Back work, and it is not
obvious from the code.

The second tricky part was the scroller ancestry. `containerRef` is on
NotePage's `<div className="h-full">`, which logically wraps the layouts, but a
debug probe (`root.contains(visibleScroller)`) returned `false`. The visible
scroller's actual ancestor chain goes through a SplitPane whose DOM is not a
descendant of the `containerRef` node the effect captured. Rather than chase
why (likely a duplicate/transitional tree from StrictMode + lazy/eager
hydration), the robust fix is to not depend on the ref's ancestry at all:
listen on `document` capture phase and query `document.documentElement` for
the restore target.

### What warrants a second pair of eyes

- The document-level capture listener filters by `.note-scroll`; confirm no other component uses that class for a non-article scroller (a grep shows only the three NotePage article `ScrollArea`s use it).
- The continuous save writes the module-level `Map` on every scroll event (no throttling). For a note reader this is fine (scroll events are not high-frequency during reading), but a throttled rAF write would be safer if scroll-bound work is ever added.
- The `scrollKey = key + "|" + hash` surrogate can still collide on repeated identical fragment visits (`A -> A#x -> A -> A#x`); the robust fix is router-minted per-entry keys (see PV-SCROLL-REVIEW-025 §16.6). This is a known limitation, documented in the file header, not addressed here.
- The "hash always beats restore" policy is still wrong for POP-to-a-fragment-after-scrolling (research T4); not addressed by these 3 fixes. Documented as a follow-up.
- Confirm the unmount cleanup runs before the module store is read on Back (it does in the note-to-search test, but worth a Strict-Mode double-invoke check).

### What should be done in the future

- Port TanStack's `data-scroll-restoration-id` registration to replace `findVisibleScroller`'s class query (PV-SCROLL-REVIEW-025 §16.6) — the lightweight de-risk for the scroller-discovery dimension.
- Route heading permalinks through React Router (`navigate({ hash })`) instead of `window.location.hash`, so every entry gets a router-minted unique key and the composite-key collision class disappears (PV-SCROLL-REVIEW-025 §8.4, §16.6).
- Add the `useNavigationType()`-aware reducer so POP-to-a-fragment-after-scroll restores the snapshot, not the heading (PV-SCROLL-REVIEW-025 §5, T4).
- Add Playwright browser tests for the three scenarios (the manual verification here is not regression-protected).

### Code review instructions

- Start at `web/src/lib/scrollRestoration.ts`: `savedOffsets` (module store), `scrollKeyOf`, the `hashchange` effect, the continuous-save capture listener, the `scrollHeight - clientHeight` restore predicate.
- Validate: `pnpm --dir web check && pnpm --dir web test && pnpm --dir web build`.
- Reproduce the browser scenarios: build, serve `vault-example`, open `/note/Index` at 1200×400, set `.note-scroll` scrollTop=150, and for each of (heading permalink → Back), (search → Back), (cross-note link → Back) assert scrollTop restores to 150.
- `git show 2e2ae67` for the full diff.

### Technical details

- Commit: 2e2ae67.
- Scroller maxScroll on the fixture (1200×400): 222; test offset 150.
- `containerRef.current` ancestry: `root.contains(visible) === false` (observed via debug probe).
- Continuous save key: `${location.key}|${window.location.hash}` read at event time.
- Restore predicate: `scroller.scrollHeight - scroller.clientHeight >= offset`.
