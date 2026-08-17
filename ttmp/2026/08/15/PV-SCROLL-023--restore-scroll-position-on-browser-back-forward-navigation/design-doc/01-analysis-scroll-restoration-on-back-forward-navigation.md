---
Title: 'Analysis: scroll restoration on back/forward navigation'
Ticket: PV-SCROLL-023
Status: active
Topics:
    - frontend
    - react
    - ux
    - hydration
    - ssr
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://web/src/components/atoms/ScrollArea/ScrollArea.tsx
      Note: nested ScrollArea used by NotePage (mobile/inline layouts)
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: |-
        existing scroll-to-hash effect (the only scroll logic today); the new save/restore must coexist with it
        existing #heading scroll effect; must coexist with restore
    - Path: repo://web/src/components/pages/NotePage/NotePage.tsx
      Note: NotePage renders inside the scroll container; owns the note view + navigation
    - Path: repo://web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: |-
        the scrollable <main className="overflow-y-auto retro-scroll"> — the real scroll container, not the window
        owns the nested <main> scroll container — where save/restore belongs
    - Path: repo://web/src/entry-client.tsx
      Note: |-
        <BrowserRouter> mount — client-side History API navigation, no full page loads
        BrowserRouter mount — client-side History API navigation
    - Path: repo://web/src/styles/chrome.css
      Note: '.retro-scroll { overflow-y: auto } — the CSS that makes <main> the scroller'
ExternalSources:
    - https://developer.mozilla.org/en-US/docs/Web/API/History/scrollRestoration
    - https://reactrouter.com/en/main/components/scroll-restoration
Summary: 'Browser back/forward does not restore scroll because the app scrolls a nested <main>, not the window, and BrowserRouter does client-side History-API navigation. Fix: save the scroll container offset on location change and restore it on POP (back/forward).'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Design for the PV-SCROLL-023 scroll-restoration fix
WhenToUse: Read before changing the scroll container, the router, or any scroll handling in the note view
---





# Analysis: scroll restoration on back/forward navigation

> **Audience.** A new engineer. This doc answers the user's question — "isn't
> scroll position something the browser should remember?" — and then gives the
> concrete reason it fails here and a focused implementation plan. It is paired
> with `PV-WIKILINK-022` (the wiki-link fix), because both live in the same
> note-view render chain, but this is an independent ticket.

---

## 1. Executive summary

When you scroll down a long note, click a link to another note, then press the
browser **Back** button, you expect to land where you left off. Today you land
at the **top** of the previous note instead.

This is not a browser bug and not a bug in the wiki-link work. It is a direct
consequence of two architectural choices:

1. **The app is a single-page app (SPA).** Navigation between notes is
   client-side, driven by React Router's `BrowserRouter` and the History API
   (`pushState`/`replaceState`). There is no full page load on a note click, so
   the browser's normal "snapshot the page and restore it on back" machinery
   does not fire the way it does for real navigations.
2. **The page does not scroll the window.** The whole layout is
   `h-screen overflow-hidden`; the thing that actually scrolls is a nested
   `<main className="overflow-y-auto retro-scroll">`. The browser only knows
> how to restore **window/document** scroll. When the scroller is a nested
> div, the window's scroll position is always `0`, so the browser has nothing
> to restore.

The fix is to save and restore the scroll offset of the **real scroll
container** ourselves, keyed by location, and to apply it on `popstate` (back/
forward). Browser back/forward **should** "just work" for traditional sites;
here it cannot, so we owe the user a small, explicit equivalent.

---

## 2. The user's question, answered: how does browser scroll restoration work?

> "isn't that something the browser should remember, or how does it work?"

Browsers expose `history.scrollRestoration`, which has two values:

- `"auto"` (the default) — the browser tries to restore the **document scroll
  position** it recorded when you left the page.
- `"manual"` — the browser does nothing; the page is responsible.

How `"auto"` actually works:

- On a **real navigation** (a full page load, or a same-document history
  traversal that the browser treats like one), the browser snapshots the
  `document.documentElement.scrollTop` / `window.scrollY` as part of the session
  history entry.
- On **back/forward** (`popstate`), it reapplies that snapshot by scrolling the
  **document/window** back to the recorded offset.

The critical limitation: **the browser only snapshots and restores the window
scroll.** It has no knowledge of nested scroll containers (`<div
style="overflow:auto">`). If your content scrolls inside a div and the window
itself never moves, `scrollRestoration: "auto"` restores `window.scrollY = 0`
— which is exactly where it already is. From the user's point of view, nothing
happens.

SPAs compound this because their "navigation" is `history.pushState`, which
**does not create the kind of history entry the browser snapshots a scroll
position for in the same way**. React Router offers `<ScrollRestoration>` —
but only for the **data router** (`createBrowserRouter`), not for the
`<BrowserRouter>` this app uses. So nothing in the stack is handling it.

In short: **yes, the browser remembers scroll position — but only for the
window, and only for navigations it treats as real page loads. This app
satisfies neither condition, so the browser has nothing to restore.**

---

## 3. Problem statement and scope

**Symptom.** Open a long note (e.g. the CoinVault index). Scroll to the middle.
Click a cross-note link (e.g. `[[README]]`). Press Back. The previous note loads
at the top, not at the scroll position you left.

**Scope (in).**

- Restore the **vertical** scroll offset of the note content container on
  browser back/forward (`popstate`) for `/note/*` routes.
- Coexist with the existing "scroll to `#heading` on hash navigation" behavior.

**Scope (out).**

- Horizontal scroll, the sidebar tree scroll, the right-panel (backlinks)
  scroll. These are separate containers with separate lifecycles; out of scope
  unless they prove disruptive.
- The static demo vault (`VITE_STATIC_VAULT`) — same component tree, so a fix
  here likely helps it too, but it is not a test target for this ticket.
- Switching to the React Router data router (`createBrowserRouter`) just to get
  `<ScrollRestoration>`. Too large a change for this fix; noted as an
  alternative.

---

## 4. Current-state analysis (evidence)

### 4.1 The router is `BrowserRouter` (client-side, History API)

`web/src/entry-client.tsx:64` mounts `<BrowserRouter>`. Note clicks go through
`useNavigate()` → `navigate("/note/<slug>")` (see
`web/src/components/pages/NotePage/NotePage.tsx`, `handleNavigate`), which
calls `history.pushState`. There is **no full page reload**, so the browser's
scroll-snapshot-on-navigation does not engage for note-to-note moves.

### 4.2 The window does not scroll; a nested `ScrollArea` does

`web/src/components/pages/VaultLayout/VaultLayout.tsx`:

- The root layout is `<div className="flex flex-col h-screen overflow-hidden …">`
  (line 121) — `overflow-hidden` means the **window never scrolls**.
- Inside it, the content region is `<main className="… overflow-y-auto retro-scroll">`,
  rendered in three branches (desktop split, desktop no-sidebar, mobile) at
  lines 248, 257, and 263. `retro-scroll` is defined in `web/src/styles/chrome.css:227`
  as `overflow-y: auto; overflow-x: hidden;`.
- **But `<main>` is *not* the actual scroller.** `NotePage` renders a nested
  `<ScrollArea className="h-full p-6">` (a `<div class="retro-scroll h-full p-6">`,
  `web/src/components/pages/NotePage/NotePage.tsx:140`) *inside* `<main>`. Both
  have `h-full`, so `<main>`'s only child is exactly `clientHeight` tall and
  `<main>` never overflows — the `ScrollArea` is the innermost scrollable and
  holds the real 16495 px note content.

**Live evidence (the published page, verified in DevTools):**

| Element | clientHeight | scrollHeight | scrolls? |
|---|---|---|---|
| `window`/document | 1745 | 1745 | No — `window.scrollTo(0,500)` stays `0` |
| `<main class="h-full overflow-y-auto retro-scroll">` | 1717 | 1717 | No (content == client) |
| `<div class="retro-scroll h-full p-6">` (NotePage's `ScrollArea`) | 1717 | **16495** | **Yes** (`scrollTop=2000` accepted) |
| `history.scrollRestoration` | — | — | `"auto"` (default; nothing here overrides it) |

So `window.scrollY` is always `0`; `scrollRestoration: "auto"` restores `0`;
the user sees no movement. **The scroller to save/restore is the `ScrollArea`
div owned by `NotePage`, not `VaultLayout`'s `<main>`.** This corrects an earlier
revision of this doc that assumed `<main>` was the scroller — reading the code
alone was insufficient; measuring the live DOM showed `<main>` does not scroll.

### 4.3 The only scroll logic today is "scroll to `#heading`"

`web/src/components/organisms/NoteHtml/NoteHtml.tsx:199-214`:

```ts
// Scroll to heading on hash navigation
useEffect(() => {
  const hash = window.location.hash.slice(1);
  if (!hash || !contentRef.current) return;
  const timer = setTimeout(() => {
    const target = contentRef.current?.querySelector(`#${CSS.escape(hash)}`);
    if (target) {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, 200);
  return () => clearTimeout(timer);
}, [slug, resolvedHtml]);
```

This handles **in-page heading anchors** (`#heading`), not back/forward scroll
position. It also calls `scrollIntoView` on the heading element, which scrolls
the nearest scrollable ancestor (the `<main>`) — confirming again that `<main>`
is the scroller and `contentRef` is inside it.

### 4.4 There is no scroll-save/restore anywhere

A search for `scrollRestoration`, `scrollTo`, `window.scroll`, `saveScroll`,
`restoreScroll`, `sessionStorage`, and scroll-position logic across `web/src`
returns **nothing** relevant (only `widgets/actions.ts` using
`history.pushState/replaceState` for widget navigation). No code saves the
container's `scrollTop` on navigation or restores it on `popstate`.

### 4.5 Note about `NoteHtml` scroll-to-hash and `scrollIntoView`

`scrollIntoView({block:"start"})` scrolls the heading to the very top of the
scroll container. On a deep `#heading` link this can be desirable, but it means
a restored scroll offset and a hash-scroll can fight each other. The design
must define precedence (see §6.3).

---

## 5. Root cause (one paragraph)

The browser's `history.scrollRestoration: "auto"` only restores **window**
scroll, and only for navigations it treats as real page loads. This SPA (a)
navigates client-side via `BrowserRouter`/`pushState` (no page load) and (b)
scrolls a nested `<main className="overflow-y-auto">` rather than the window
(the root layout is `h-screen overflow-hidden`, so `window.scrollY` is always
`0`). The browser therefore snapshots and restores `0`, and the user lands at
the top on Back. There is no custom scroll-save/restore code to compensate.

---

## 6. Proposed solution

Save the real scroll container's `scrollTop` when leaving a location, and
restore it when returning via back/forward. Keep it keyed by location key so
distinct notes keep distinct positions.

### 6.1 Design overview (pseudocode)

```
on every location change (before the new view mounts / while leaving):
    container.scrollTop  ->  save to a Map<location.key, number>  (in-memory)

on popstate (back/forward), after the target view has rendered:
    if we have a saved offset for this location.key:
        container.scrollTop = saved offset   (restore, instant — not smooth)
    else if the URL has a #hash:
        existing scroll-to-heading behavior wins
    else:
        container.scrollTop = 0  (top of a new note)
```

The map is kept in a ref/module scope (not `sessionStorage` by default — see
decision record), survives across in-app navigations within the tab, and is
trivially bounded.

### 6.2 Where the logic lives

Two candidate homes:

- **`NotePage`** (`web/src/components/pages/NotePage/NotePage.tsx`) — **owns the
  actual scroller**: the `<ScrollArea className="h-full p-6">` div (desktop,
  line 140) and `h-full p-4` (mobile, line 176) proved to be the real scroll
  containers (§4.2). It also owns the `/note/*` lifecycle and `slug`.
- **`VaultLayout`** (`web/src/components/pages/VaultLayout/VaultLayout.tsx`) —
  wraps every route and owns `<main>`, but `<main>` is a non-scrolling
  pass-through here, so a ref on it would not give a usable `scrollTop`.

Chosen: **`NotePage`**, because it owns the element that actually scrolls. The
scroller handle is obtained by querying the nearest scrollable ancestor of
`NoteHtml`'s `contentRef` (the `scrollIntoView` path already relies on that
ancestor), or by a ref on the active `<ScrollArea>`. `NotePage` already branches
desktop/mobile, so the same single-ref-per-branch approach as before applies. The
`history.scrollRestoration = "manual"` declaration and the `useLocation()` +
save/restore effect live here. (An earlier revision chose `VaultLayout` on the
wrong premise that `<main>` was the scroller — corrected by the live probe.)

If you would rather avoid coupling to which element scrolls, the helper can find
the scroller at runtime: walk up from `contentRef` to the first ancestor whose
`scrollHeight > clientHeight`. That is robust to the `<main>`/`ScrollArea`
layout shifting, at the cost of a per-navigation walk.

### 6.3 Coexistence with the existing `#heading` scroll (precedence)

- A location **with a hash** (`/note/slug#heading`) means the user (or a link)
  explicitly asked for a heading. The hash-scroll effect in `NoteHtml` should
  win. Do **not** restore a saved offset on a hashed URL.
- A location **without a hash** that we have a saved offset for, reached via
  back/forward, restores the offset.
- A fresh forward navigation to a new note (no saved offset, no hash) scrolls
  to top.

This precedence is enforced by checking `location.hash` before applying a saved
offset.

### 6.4 Restore timing (the tricky part)

The DOM must exist and have its measured height before `scrollTop = offset`
does anything useful. The note HTML is injected after the RTK Query fetch
resolves and (for the wiki-link case) after `resolveWikiLinks` runs in an
effect. Restoring too early sets `scrollTop` on an empty container and is a
no-op. Options:

1. **Poll/`requestAnimationFrame` loop** until the content height exceeds the
   offset (or a timeout). Simple, robust, no new dependencies.
2. **`ResizeObserver`** on the content container, restore once when
   `scrollHeight >= offset`. Clean, but observer lifecycle must be managed.
3. **Hook into the existing `resolvedHtml` effect** in `NoteHtml` — restore
   after the same effect that finalizes HTML. Tight coupling across layers.

Chosen: **option 1** (a short rAF/poll loop with a timeout). It is the most
robust against async rendering and keeps the logic in one component. The
existing `NoteHtml` hash-scroll already uses a `setTimeout(…, 200)` heuristic,
so a small timed retry is consistent with existing style.

### 6.5 What to save/restore

- Save: `mainEl.scrollTop` (vertical only; horizontal is `overflow-hidden`).
- Key: `location.key` (React Router provides a stable per-entry key) — falls
  back to `location.pathname + location.hash` if `key` is unavailable.
- On a forward navigation to a brand-new note, there is no saved offset → top.

---

## 7. Decision records

### Decision: opt out of the browser default and restore the container ourselves

- **Context:** `history.scrollRestoration` defaults to `"auto"`, but it only
  restores window scroll, which is always `0` here.
- **Options considered:**
  1. Set `history.scrollRestoration = "manual"` and fully own save/restore.
  2. Leave `"auto"` and add our own restore on top (browser restores `0`, we
     restore the real offset). Harmless but redundant; can cause a flash.
- **Decision:** Option 1 — set `"manual"` and own it.
- **Rationale:** We are fully responsible anyway; declaring `manual` removes the
  redundant/no-op browser pass and makes the contract explicit.
- **Consequences:** We must restore in every case the browser would have
  (including the no-hash, no-saved-offset → top case). One more thing to get
  right, but the behavior is now fully defined by us.
- **Status:** proposed

### Decision: in-memory map, not `sessionStorage`

- **Context:** Saved offsets must persist across in-tab navigations. They need
  not survive a tab close/reopen (the user is starting fresh anyway).
- **Options considered:**
  1. `sessionStorage` — survives reloads; survives long sessions.
  2. In-memory `Map` — lost on reload; simpler; no serialization; no quota.
- **Decision:** Option 2 (in-memory `Map`), with `sessionStorage` noted as a
  follow-up if reload-restoration is requested.
- **Rationale:** The reported problem is back/forward within a session, which an
  in-memory map solves with zero serialization risk. A reload is a fresh start,
  which is the expected UX. Adding `sessionStorage` later is a one-line change.
- **Consequences:** A page reload (F5) will not restore the previous scroll.
  Acceptable; matches most SPAs.
- **Status:** proposed

### Decision: scope to `/note/*`, in `VaultLayout`

- **Context:** The scroll container serves all routes, but the problem is
  reported on note reading. Search and widget pages have different scroll
  expectations.
- **Options considered:**
  1. Global save/restore for every route.
  2. Scope to `/note/*` only, in `VaultLayout` (the scroller owner).
- **Decision:** Option 2.
- **Rationale:** Smallest correct surface; `/note/*` is where long, scrollable
  content lives and where the user expects "return to where I was." Search
  results resetting to top on Back is acceptable.
- **Consequences:** If search/widget scroll restoration is later wanted, extend
  the same hook; the design generalizes.
- **Status:** proposed

### Decision: instant restore, not smooth

- **Context:** Back/forward should land exactly where you were, instantly, like
  a native browser does.
- **Options considered:** `scrollTo({behavior:"smooth"})` vs instant
  `scrollTop = offset`.
- **Decision:** Instant.
- **Rationale:** Smooth-scroll on restore animates from the top, which feels
  slow and disorienting on Back. Native back/forward is instant.
- **Consequences:** None; matches user expectation.
- **Status:** proposed

---

## 8. Implementation plan (file-level)

1. **`web/src/components/pages/NotePage/NotePage.tsx`** (owns the real scroller —
   see §4.2 correction)
   - Add a ref to the active `<ScrollArea>` scroll container (desktop `h-full
     p-6` at line 140; mobile `h-full p-4` at line 176; one ref per branch, or a
     single ref since only one branch is visible at a time).
   - `useEffect` on mount: `history.scrollRestoration = "manual"`.
   - `const location = useLocation()`; keep a `useRef<Map<string, number>>` of
     saved offsets.
   - On `location` change (cleanup of the previous effect): save the current
     `scrollEl.scrollTop` under the previous `location.key`.
   - On `location` change (new effect), if `pathname.startsWith("/note/")`:
     - if `location.hash` is present → do nothing (let `NoteHtml` handle it);
     - else if a saved offset exists → run a short rAF/poll loop (with a
       ~1s timeout) that sets `scrollEl.scrollTop = offset` once
       `scrollEl.scrollHeight >= offset`; restore instantly.
     - else → `scrollEl.scrollTop = 0`.
2. **`web/src/components/organisms/NoteHtml/NoteHtml.tsx`**
   - No change required for the restore path. The existing `#heading` effect
     (lines 199-214) already wins for hashed URLs because the VaultLayout
     restore explicitly skips hashed locations.
3. **No backend changes.** This is entirely client-side.
4. **No router change.** Stay on `BrowserRouter`; do not migrate to the data
   router for this ticket.

### Sketch (VaultLayout, essence)

```ts
const scrollRef = useRef<HTMLElement | null>(null);
const saved = useRef<Map<string, number>>(new Map());
const location = useLocation();
const prevKey = useRef<string | null>(null);

useEffect(() => {
  history.scrollRestoration = "manual";
}, []);

useEffect(() => {
  const el = scrollRef.current;
  // Save the position we are leaving.
  if (el && prevKey.current != null) {
    saved.current.set(prevKey.current, el.scrollTop);
  }
  prevKey.current = location.key;

  if (!location.pathname.startsWith("/note/")) return;
  if (location.hash) return; // NoteHtml handles #heading

  const el2 = scrollRef.current;
  if (!el2) return;
  const offset = saved.current.get(location.key) ?? 0;

  // Restore once the content is tall enough; instant.
  let tries = 0;
  const restore = () => {
    if (el2.scrollHeight >= offset || tries++ > 60) {
      el2.scrollTop = offset; // 0 when no saved offset
      return;
    }
    requestAnimationFrame(restore);
  };
  requestAnimationFrame(restore);
}, [location.key, location.pathname, location.hash]);
```

(`scrollRef` is attached to the active `<main>`; because exactly one of the
three `<main>` branches renders at a time, a single ref suffices.)

---

## 9. Test strategy

**Manual (decisive).**

```bash
pnpm --dir web build
go run ./cmd/retro-obsidian-publish serve --vault <vault> --port 8080 --watch=false --serve-web
```

1. Open a long note (`/note/research/.../index-of-design-patterns`). Scroll to
   ~50%.
2. Click a cross-note link → new note loads.
3. Press **Back** → assert the previous note restores to ~50% (instantly).
4. Press **Forward** → assert the second note is at the top (fresh) or its saved
   offset if you scrolled it.
5. Click a `#heading` link (same note) → assert it scrolls to the heading.
6. From that hashed URL, click another note, then Back → assert you return to
   the **hashed** note at the heading (hash wins), not the pre-hash offset.

**Automated (vitest, node env — limited).** The scroll container + rAF loop is
DOM-dependent; without `happy-dom`/`jsdom` (not currently a direct dependency),
only pure helpers (e.g. a `pickScrollAction(location, saved)` returning
`"hash" | "restore" | "top"`) can be unit-tested. Extract that policy and test
it like the PV-WIKILINK-022 helper:

```ts
pickScrollAction({pathname:"/note/a", hash:"#h"}, saved) === "hash"
pickScrollAction({pathname:"/note/a", hash:""},   new Map([["k",300]])) === "restore"  // key matches
pickScrollAction({pathname:"/note/a", hash:""},   new Map()) === "top"
pickScrollAction({pathname:"/search", hash:""},   new Map()) === "none"
```

**Type/build gate:** `pnpm --dir web check` and `pnpm --dir web build`.

---

## 10. Risks, alternatives, open questions

**Risks.**

- **Restore-before-paint flash.** If the container briefly renders at top
  before the rAF restores, the user sees a flash. Mitigation: the rAF loop runs
  before the first painted frame in most cases; if a flash appears, gate the
  content visibility until restored (heavier; deferred).
- **Multiple `<main>` branches.** Only one renders at a time (desktop split vs
  desktop no-sidebar vs mobile), so a single ref works — but verify the ref
  attaches on resize-driven branch switches, or the restore may target a
  detached node.
- **Resizing the split pane** changes `scrollHeight`; a saved offset may then
  undershoot. Acceptable (Back still lands near the right place).

**Alternatives considered (not chosen).**

- **Migrate to `createBrowserRouter` + `<ScrollRestoration>`.** React Router's
  built-in restores window scroll for the data router; it still would not help
  with the nested container without a custom `<ScrollRestoration>` `getKey`,
  and the migration is a large, cross-cutting change. Not worth it for this fix.
- **Option B — scroll the window instead of a nested container ("let the
  browser do it for free").** The user's instinct that "the browser should
  remember it" is correct *for window-scrolling sites*; the only reason it
  fails here is that we deliberately scroll an inner div. Restructuring
  `VaultLayout` so the document scrolls (`min-h-screen` instead of
  `h-screen overflow-hidden`; `position: sticky` header and sidebar instead of
  flex columns with `shrink-0`; main content flowing the document) would make
  `scrollRestoration: "auto"` work unmodified, plus give native keyboard
  scroll (Space/PgDn/Cmd+End), better mobile, and SSR-friendlier behavior. For
  a *publishing* tool this is arguably the more principled long-term shape. It
  is **not chosen for this ticket** because it is a large rework of the layout
  (resizable panels, sticky header, sidebar height) and risks the existing
  app-like chrome. It is recorded as the genuine "use the browser" option the
  user asked about; the hand-rolled save/restore (Option A, the rest of this
  doc) is the smaller, lower-risk path that preserves the current layout.

**Open questions.**

1. Should the **sidebar tree** and **right panel (backlinks)** also restore
   scroll? Probably not for this ticket; confirm with a user.
2. Should offsets survive a page reload (`sessionStorage`)? Default says no;
   revisit if requested.
3. Does the SSR sidecar need any change? No — SSR renders initial HTML; scroll
   is a client concern. Confirm the hydrated client owns the `<main>` ref.

---

## 11. References (file:line)

| Concern | File | Key line |
|---|---|---|
| Router mount (BrowserRouter) | `web/src/entry-client.tsx` | `<BrowserRouter>` :64 |
| Layout: window does not scroll | `web/src/components/pages/VaultLayout/VaultLayout.tsx` | root `h-screen overflow-hidden` :121; `<main overflow-y-auto retro-scroll>` :248, :257, :263 |
| Scroll container CSS | `web/src/styles/chrome.css` | `.retro-scroll { overflow-y:auto }` :227 |
| Note navigation (pushState) | `web/src/components/pages/NotePage/NotePage.tsx` | `handleNavigate` → `navigate("/note/…")` |
| Existing `#heading` scroll | `web/src/components/organisms/NoteHtml/NoteHtml.tsx` | scroll-to-hash effect :199-214 |
| Widget pushState (not in scope) | `web/src/widgets/actions.ts` | `pushState/replaceState` :104-105 |
| Browser API reference | — | `history.scrollRestoration` (MDN) |
| React Router reference | — | `<ScrollRestoration>` (data router only) |

---

## 12. Glossary

- **`history.scrollRestoration`** — browser API (`"auto"` default / `"manual"`)
  controlling whether the browser restores **window** scroll on back/forward.
  Only affects the window/document, never nested containers.
- **`popstate`** — the event fired on back/forward (and on same-document
  history traversal). The hook where restore logic runs.
- **`BrowserRouter`** — React Router's History-API router; navigation is
  client-side `pushState`, no full page load.
- **Scroll container** — the element whose `overflow` makes it the scroller.
  Here it is `<main className="overflow-y-auto retro-scroll">`, not the window.
- **`location.key`** — React Router's stable per-history-entry identifier; used
  to key saved offsets.
