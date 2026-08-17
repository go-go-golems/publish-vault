---
Title: External sources — scroll restoration research
Ticket: PV-SCROLL-REVIEW-025
Status: reference
Topics:
    - frontend
    - react
    - routing
    - ssr
    - ux
    - wiki-link
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://tanstack.com/router/latest/docs/guide/scroll-restoration
    - https://github.com/remix-run/react-router/discussions/9495
    - https://www.davidtran.dev/blogs/scroll-restoration-in-spas
    - https://github.com/nanyang24/react-scroll-restoration
    - https://dev.to/ktg0215/browser-scroll-restoration-is-broken-on-spas-heres-how-a-chrome-extension-fixes-it-578p
    - https://dev.to/tene/scroll-restoration-in-react-router-4gnm
Summary: Curated external sources showing how other frameworks, libraries, and developers solve nested-container SPA scroll restoration. Used to validate the PV-SCROLL-REVIEW-025 architecture recommendation.
LastUpdated: 2026-08-16T22:50:00-04:00
WhatFor: Compare the proposed coordinator design against established solutions and mine reusable patterns (per-entry keys, explicit scrollable-area registration, before-paint restore, async content).
WhenToUse: Read before implementing the coordinator or evaluating whether to adopt TanStack Router / react-router ScrollRestoration / a third-party library instead of a custom hook.
---

# External sources — scroll restoration research

## Why these sources

The PV-SCROLL-REVIEW-025 review recommends a persistent coordinator that owns one
scroll snapshot per browser history entry, registers nested scrollable areas
explicitly, and converges when async content changes geometry. These sources
confirm each of those decisions is a known, solved problem in the wider
ecosystem — they are not novel inventions. They also expose the failure modes
the PR hit (window-only restore, `!!note`-as-readiness, non-unique keys).

## Source index

### 1. TanStack Router — Scroll Restoration (official docs) ★ most relevant

File: `tanstack-router-scroll-restoration.md`
URL: https://tanstack.com/router/latest/docs/guide/scroll-restoration

The most directly applicable reference. TanStack Router solves the exact
problem this app has: nested scrollable areas (CSS-grid panes, overflow divs)
where `window.scrollY` is always 0. Key validated patterns:

- **Nested scrollable areas via `scrollToTopSelectors`**: the router scrolls
  registered nested elements in addition to window. Confirms the review's
  "explicit scroller registration" recommendation over global class queries.
- **Per-history-entry cache keys**: default key is `location.state.__TSR_key`
  (a unique key generated for *each history entry*), not the pathname. This is
  exactly the review's "entry identity, not URL" invariant (I1). They
  explicitly deprecate the older `state.key` in favor of a per-entry key.
- **`useElementScrollRestoration` + `data-scroll-restoration-id`**: manual
  registration of a specific scrollable element by a unique ID. This is the
  review's "forwarded ref + scoped registration" pattern, implemented as a DOM
  attribute picked up by a scroll-restoration watcher.
- **Restore before DOM paint**: their list of mechanisms includes "Restoring
  scroll positions after successful navigations before DOM paint" — the same
  before-paint intent behind the review's layout-effect save.
- **`resetScroll={false}` on `<Link>`/`navigate`/`redirect`**: a per-navigation
  opt-out, the equivalent of distinguishing PUSH (reset) from POP (restore).
- **Virtualized lists as the manual case**: async/height-changing content is a
  first-class concern, addressed by `initialOffset: scrollEntry?.scrollY`.

Takeaway for this ticket: TanStack's design is strong evidence for the
coordinator + explicit registration + per-entry-key approach. If the project
ever migrates routers, TanStack would solve most of this out of the box.

### 2. react-router discussion #9495 — ScrollRestoration with non-window container ★ very relevant

File: `react-router-discussion-9495-non-window-scrollrestoration.md`
URL: https://github.com/remix-run/react-router/discussions/9495

A maintainer-acknowledged gap: `<ScrollRestoration>` only uses
`window.scrollY`/`window.scrollTo`, so CSS-grid layouts where one pane scrolls
break it. Relevant points:

- The maintainer (`@brophdawg11`) explicitly notes the edge case the review
  raises: "what if when you come back to that location the scrollElement is no
  longer present? falling back on window isn't right since the scroll position
  is relative to the no-longer-rendered element." This is the review's
  Finding 7 (DOM discovery by class/visibility is a weak contract) and the
  "scroller comes and goes" lifecycle, validated by the upstream maintainer.
- A community workaround proposal patches `useScrollRestoration` to target an
  element by id/ref instead of window.
- `elementRef.current is still empty when it's time to restore` — a community
  user hit the exact "ref not ready at restore time" failure the review's
  Finding 5 (`!!note` ≠ readiness) describes.

Takeaway: React Router's built-in component does not solve our problem; the
review's custom coordinator is necessary, and the maintainer's own caveat
matches the review's registration/ownership recommendation.

### 3. David Tran — Scroll restoration in SPAs (blog)

File: `davidtran-scroll-restoration-in-spas.md`
URL: https://www.davidtran.dev/blogs/scroll-restoration-in-spas

A clear conceptual overview of the three forces the review formalizes:

- **History entries carry scroll position** (native, for MPA).
- **`history.scrollRestoration`** lets the app opt into `manual` and own
  restoration — the exact mode PR 21 sets and the review keeps.
- The async-content problem: the browser doesn't know page height until after
  render. Maps to the review's convergence/readiness section.

Useful as the "why" explanation for an intern; not a ready-made implementation.

### 4. nanyang24/react-scroll-restoration (library)

File: `nanyang24-react-scroll-restoration.md`
URL: https://github.com/nanyang24/react-scroll-restoration

A small React Router v4/v5 `<ScrollRestoration>` component. Minimal but
confirms two points:

- React Router historically did **not** provide scroll restoration
  out-of-the-box (cites react-router issue #3950).
- `history.scrollRestoration` is primarily a way to *disable* the browser's
  (broken-for-SPA) automatic attempts and take over — exactly the review's
  "manual mode with cleanup" decision.

Limited (window-based, older Router); kept as a historical reference.

### 5. ktg0215 — Browser Scroll Restoration Is Broken on SPAs (dev.to)

File: `ktg0215-browser-scroll-restoration-broken-spas.md`
URL: https://dev.to/ktg0215/browser-scroll-restoration-is-broken-on-spas-heres-how-a-chrome-extension-fixes-it-578p

Crisp problem statement for the failure modes the review addresses:

- `history.scrollRestoration = 'auto'` "falls apart for SPAs where content is
  injected into the DOM after navigation" and "infinite scroll pages where the
  content at a given Y position changes." Maps to the review's
  async-content and content-revision concerns.
- Documents `sessionStorage` as the persistence layer and
  `requestAnimationFrame` polling for "wait for the page to reach its full
  height" — a simpler version of the review's convergence controller (the
  review upgrades rAF-polling to ResizeObserver/event-driven convergence).

Useful for the intern to see the rAF-poll approach the review argues against
as a *primary* mechanism (it's a timer, not a readiness signal).

### 6. tene — Scroll Restoration in React Router (dev.to)

File: `tene-scroll-restoration-react-router.md`
URL: https://dev.to/tene/scroll-restoration-in-react-router-4gnm

Practical catalog of scroll-restoration pitfalls in React Router:

- **Back/forward scroll loss**: a naive scroll-to-top component breaks native
  back/forward restore. The review's Finding 4 (ignoring POP/PUSH) is the
  generalized form.
- Recommends `history.state` for advanced handling — the same "entry identity"
  direction the review formalizes (TanStack later productized this as
  `__TSR_key`).
- Notes mobile Safari/iOS quirks (relevant to the review's responsive
  scroller-branch handling).

## Patterns confirmed across sources

| Review recommendation | Confirmed by |
|---|---|
| Per-history-entry cache key (not URL/slug) | TanStack `__TSR_key`; tene `history.state` |
| Explicit nested-scrollable registration | TanStack `scrollToTopSelectors` + `data-scroll-restoration-id`; RR #9495 maintainer caveat |
| `history.scrollRestoration = "manual"` + own it | David Tran; nanyang24; ktg0215 |
| Async content needs readiness, not `!!data` | TanStack virtualizer `initialOffset`; ktg0215 "injected after navigation" |
| Restore before paint | TanStack "before DOM paint" mechanism list |
| window-only restore is insufficient for grid layouts | RR #9495 (the whole discussion); TanStack nested-area section |
| Distinguish POP (restore) from PUSH (top) | TanStack `resetScroll`; tene back/forward loss |

## Divergences / what the review adds

- The sources mostly assume *one* scrollable area (window or a single pane).
  This app has desktop/mobile branches that replace each other and a
  right-panel toggle that swaps the desktop scroller — the review's explicit
  multi-instance registration and responsive-handling test matrix (T7, T8) go
  beyond the sources.
- The sources use rAF polling or the router's own before-paint hook for
  readiness. The review argues for `ResizeObserver` + content-revision +
  deadline because embeds fetch on a network timeline unrelated to frames.
- The sources do not formalize the POP-to-a-fragment-entry-after-user-scroll
  case (review T4 / reducer row 1). TanStack's `resetScroll` is the closest,
  but it is an opt-out on new navigation, not a POP-with-snapshot precedence
  rule.
