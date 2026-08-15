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

## Step 2: Correct the scroller premise (live probe) and answer the user's mechanism question

The user asked whether the browser can remember inner-div scroll and whether
most sites scroll an inner element. Answering accurately required verifying
which element actually scrolls on the live page — which overturned the Step 1
premise that `<main>` is the scroller.

### Prompt Context

**User prompt (verbatim):** "is there no option to use the browser to scroll inner divs and remember  those? don't most websites scroll the inner element or so?"

**Assistant interpretation:** Explain whether any browser mechanism remembers nested-div scroll on back/forward, and whether the inner-scroll pattern is actually the norm.

**Inferred user intent:** The user suspects the fix should be "just use the browser" rather than hand-rolling, and wants to know if that's possible / normal.

### What I did

- On the live page, measured every `.retro-scroll` element with a real `scrollTop=2000` test plus a `window.scrollTo(0,500)` test.
- Found: `window.scrollTo` is a no-op (`window.scrollY` stays 0; body scrollHeight == innerHeight == 1745); `<main>` does NOT scroll (`clientHeight == scrollHeight == 1717`); the **real article scroller is the nested `<div class="retro-scroll h-full p-6">`** (NotePage's `ScrollArea`, scrollHeight 16495, `scrollTop` accepted). `history.scrollRestoration === "auto"`.
- Corrected the design doc: §4.2 (scroller is the ScrollArea, not `<main>`), §6.2 (fix lives in NotePage, not VaultLayout), §8 (implementation targets the ScrollArea), and elevated "scroll the window instead" to a real Option B.

### Why

Reading the code (h-screen overflow-hidden + main overflow-y-auto) suggested `<main>` was the scroller, but the nested `ScrollArea` with `h-full` makes `<main>` a non-scrolling pass-through. Only a live measurement found this. Shipping a fix that targeted `<main>` would have silently done nothing — the same failure mode the PV-WIKILINK-022 investigation warned about (server output looked correct; the bug was in the browser DOM).

### What worked

- The per-element `scrollTop` probe unambiguously identified the one element that actually accepts scroll, among 10 `.retro-scroll` candidates.
- It also directly answered the user: `window.scrollY` never moves, so `scrollRestoration:"auto"` (already in effect) restores 0.

### What didn't work

- Nothing; purely diagnostic.

### What I learned

- `history.scrollRestoration` is **window/document-only**; there is no browser API that remembers a div's scroll. Confirmed by spec and by the live probe (it's `"auto"` and does nothing useful here).
- Most *content* websites scroll the **window** (so the browser restores for free); the inner-div-scroll pattern is characteristic of *app-like* SPAs, which hand-roll restoration or use a framework feature. The user's intuition reflects app-SPAs, not the web norm.
- The genuine "use the browser" option is to **make the window scroll** (Option B) — a larger layout rework, recorded as a legitimate alternative.

### What was tricky to build

Diagnosis only. The trap was the nested `h-full` chain: both `<main>` and the inner `ScrollArea` have `overflow-y:auto` and `h-full`, so which one scrolls depends on whether the inner element's height is clamped to the viewport (it is, via `h-full`). The innermost clamped element with overflowing content wins. Getting this wrong means the fix targets a non-scrolling container.

### What warrants a second pair of eyes

- Confirm the desktop scroller is the `h-full p-6` `ScrollArea` (line 140) and mobile is `h-full p-4` (line 176); the fix must ref the active branch's ScrollArea, or walk up from `contentRef` to the nearest scrollable ancestor (robust to layout shifts).
- If Option B (window-scroll) is ever pursued, the resizable `ResizablePanelGroup` + sticky header rework is the risky part; budget it as its own ticket.

### What should be done in the future

- Decide Option A (hand-rolled save/restore in NotePage, this ticket) vs Option B (restructure to window-scroll, separate larger ticket). Option A is lower-risk and preserves the app layout; Option B is more "normal website" and gets browser restoration for free.

### Code review instructions

- Design doc §4.2 now carries the live evidence table; §6.2/§8/§10 corrected to the ScrollArea + Option B.
- Reproduce the probe: on a note page, DevTools console → `Array.from(document.querySelectorAll('.retro-scroll')).map(e=>({cls:e.className.slice(0,30),ch:e.clientHeight,sh:e.scrollHeight,moved:(e.scrollTop=2000,e.scrollTop)}))` and `window.scrollTo(0,500),window.scrollY`.

### Technical details

Live probe result (article scroller identified): `<div class="retro-scroll h-full p-6">` — `clientHeight 1717, scrollHeight 16495, scrollTop accepted 2000`. `window.scrollY` never changes; `history.scrollRestoration === "auto"`. Therefore no browser mechanism can restore the article's position; it must be saved/restored on the ScrollArea (Option A), or the layout must scroll the window (Option B).
