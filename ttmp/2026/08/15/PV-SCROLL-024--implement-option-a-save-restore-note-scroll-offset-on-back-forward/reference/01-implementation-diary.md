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
      Note: vitest regression test for pickScrollAction
    - Path: web/src/lib/scrollRestoration.ts
      Note: new module to implement (pickScrollAction + useScrollRestoration + findVisibleScroller)
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
