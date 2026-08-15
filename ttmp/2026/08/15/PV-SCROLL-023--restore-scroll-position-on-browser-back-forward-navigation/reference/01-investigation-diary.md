---
Title: Investigation diary
Ticket: PV-SCROLL-023
Status: active
Topics:
    - frontend
    - react
    - ux
    - hydration
    - ssr
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: 'existing #heading scroll effect; must coexist with restore'
    - Path: repo://web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: |-
        owns the scroll container <main overflow-y-auto>; where save/restore belongs
        diagnosis evidence (h-screen overflow-hidden + nested scroller)
    - Path: repo://web/src/entry-client.tsx
      Note: BrowserRouter mount (client-side History API nav)
ExternalSources:
    - https://developer.mozilla.org/en-US/docs/Web/API/History/scrollRestoration
    - https://reactrouter.com/en/main/components/scroll-restoration
Summary: 'Diary for PV-SCROLL-023: diagnosed why back/forward does not restore scroll (nested <main> scroller + SPA pushState nav), wrote the design doc. Implementation deferred to a follow-up turn.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Diary for the PV-SCROLL-023 scroll-restoration investigation
WhenToUse: Read before implementing the scroll-restore fix
---


# Investigation diary

## Goal

Explain why browser back/forward does not restore the article's scroll
position in publish-vault, create a docmgr ticket with an intern-facing
analysis/design/implementation guide, and record the evidence. Implementation
of the fix is scoped for a follow-up turn; this diary captures the diagnosis.

## Step 1: Diagnose the scroll-restoration failure and write the design

The user asked whether scroll position on back/forward is "something the
browser should remember" and requested a separate ticket. The first job was to
answer the mechanism question correctly from the code, then prove why it fails
*here specifically* rather than in a generic SPA.

### Prompt Context

**User prompt (verbatim):** "Another thing worth fixing, make a separate ticket. When I use the back navigation / forward, the article oesn't scroll back to my previous scrolling position, isn't that something the brwoser should remember, or how does it work?"

**Assistant interpretation:** Create a new docmgr ticket for the missing scroll-position restoration on back/forward, explain how browser scroll restoration works and why it fails here, and write an analysis/design/implementation guide.

**Inferred user intent:** The user wants to understand the mechanism (is it a browser feature or our responsibility?) and have a tracked, designable fix — not just a hand-wave.

### What I did

- Read the router mount: `web/src/entry-client.tsx:64` uses `<BrowserRouter>` (History API, client-side nav, no full page load).
- Read the layout: `web/src/components/pages/VaultLayout/VaultLayout.tsx` — root is `h-screen overflow-hidden` (line 121), and the scroller is a nested `<main className="overflow-y-auto retro-scroll">` (lines 248, 257, 263). Confirmed the **window never scrolls**.
- Read the CSS: `web/src/styles/chrome.css:227` `.retro-scroll { overflow-y:auto; overflow-x:hidden; }` — the rule that makes `<main>` the scroller.
- Read the only existing scroll logic: `web/src/components/organisms/NoteHtml/NoteHtml.tsx:199-214` — a `#heading` scroll-to effect using `scrollIntoView`. It handles in-page heading anchors only, not back/forward position.
- Searched the whole `web/src` for `scrollRestoration`, `scrollTo`, `window.scroll`, `saveScroll`, `restoreScroll`, `sessionStorage`, scroll-position logic — **nothing** relevant (only `widgets/actions.ts` `pushState/replaceState` for widget nav).
- Created ticket `PV-SCROLL-023` with a design-doc and a diary.
- Wrote the design doc: explains `history.scrollRestoration` (the user's question), the two reasons it fails here (nested scroller + SPA pushState nav), current-state evidence with file:line, a save/restore design with precedence over the `#heading` effect, decision records, file-level implementation plan with a code sketch, and a test strategy.

### Why

The mechanism question matters because the fix is not "enable a browser flag" —
the browser default (`"auto"`) is already in effect and is useless here because
it only restores window scroll, which is always `0`. Naming both root causes
(nested container + client-side nav) keeps a future implementer from chasing
the wrong layer (e.g. trying to fix the router instead of the scroll container).

### What worked

- Reading `VaultLayout` immediately showed the `h-screen overflow-hidden` root + nested `overflow-y-auto` `<main>` — the single fact that explains why `window.scrollY` is always `0` and thus why `scrollRestoration:"auto"` restores nothing.
- The existing `NoteHtml` `#heading` effect already calls `scrollIntoView`, which scrolls the nearest scrollable ancestor — independent confirmation that `<main>` is the scroller and `contentRef` is inside it.
- A targeted grep across `web/src` proved there is zero existing save/restore code, so the fix is greenfield (no conflicting logic to untangle).

### What didn't work

- Nothing failed. The investigation was purely static reading + grep; no runtime needed for the diagnosis.

### What I learned

- `history.scrollRestoration:"auto"` snapshots/restores **only** `document`/`window` scroll, never nested containers. This is the crux of the answer to the user.
- React Router's `<ScrollRestoration>` exists but only for the **data router** (`createBrowserRouter`), not the `<BrowserRouter>` this app uses — so even "use the framework feature" is not a one-liner here.
- The `NoteHtml` `#heading` scroll and a new position-restore must define precedence: a URL with a `#hash` means the user asked for a heading, so the hash wins; only no-hash back/forward restores the saved offset.

### What was tricky to build

The diagnosis was straightforward, but the design has one sharp edge worth
flagging for the implementer: **restore timing**. The note HTML is injected
after an RTK Query fetch resolves and after `resolveWikiLinks` runs in an
effect, so restoring `scrollTop` too early sets it on an empty container (a
no-op). The design specifies a short `requestAnimationFrame`/poll loop with a
timeout that waits until `scrollHeight >= offset` before applying the restore.
Getting this wrong is the most likely way the fix silently does nothing.

### What warrants a second pair of eyes

- The single-ref-attaches-to-one-of-three-`<main>`-branches assumption. Only one branch renders at a time (desktop split / desktop no-sidebar / mobile), so one ref should work — but verify on resize-driven branch switches that the ref reattaches, or the restore targets a detached node.
- Setting `history.scrollRestoration = "manual"` makes us fully responsible for the top-of-new-note case too; confirm the "no saved offset, no hash → top" branch is implemented, not just the restore branch.

### What should be done in the future

- Implement the save/restore in `VaultLayout` per the design's §8 plan; add the `pickScrollAction` pure helper and its vitest test (node env, no DOM dep needed — same pattern as PV-WIKILINK-022).
- Decide whether sidebar/right-panel scroll should also restore (likely no for this ticket; confirm with the user).
- Optional: persist offsets in `sessionStorage` if reload-restoration is wanted.

### Code review instructions

- Start at the design doc §4 (current-state evidence) and §5 (root cause one-paragraph).
- Validate the mechanism claim yourself: in a browser DevTools console on a note page, run `window.scrollY` (expect `0` even after scrolling) and `document.querySelector('main.retro-scroll').scrollTop` (expect it to change) — that pair proves the window is not the scroller.

### Technical details

The decisive two facts:

1. `VaultLayout` root: `<div className="flex flex-col h-screen overflow-hidden …">` (line 121) → window does not scroll.
2. Scroller: `<main className="… overflow-y-auto retro-scroll">` (lines 248/257/263) + `.retro-scroll { overflow-y:auto }` (chrome.css:227) → a nested div is the scroller.

Browser default `history.scrollRestoration === "auto"` is in effect (no code sets it). It restores `window.scrollY`, which is always `0` here. Hence: nothing moves on Back.

Design essence (see design doc §8 for the full sketch):

```ts
history.scrollRestoration = "manual";   // we own it
// on leaving a location:  saved.set(prevKey, mainEl.scrollTop);
// on popstate, /note/*, no hash:
//   offset = saved.get(key) ?? 0
//   rAF loop until scrollHeight >= offset, then mainEl.scrollTop = offset (instant)
```
