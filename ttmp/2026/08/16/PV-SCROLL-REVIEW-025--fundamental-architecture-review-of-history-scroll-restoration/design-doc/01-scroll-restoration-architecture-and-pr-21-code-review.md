---
Title: Scroll restoration architecture and PR 21 code review
Ticket: PV-SCROLL-REVIEW-025
Status: active
Topics:
    - frontend
    - react
    - routing
    - ssr
    - ux
    - wiki-link
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/08/16/PV-SCROLL-REVIEW-025--fundamental-architecture-review-of-history-scroll-restoration/sources/00-sources-index.md
      Note: Curated index mapping external patterns to review recommendations
    - Path: repo://ttmp/2026/08/16/PV-SCROLL-REVIEW-025--fundamental-architecture-review-of-history-scroll-restoration/sources/react-router-discussion-9495-non-window-scrollrestoration.md
      Note: 'Maintainer-acknowledged gap: window-only ScrollRestoration breaks grid layouts'
    - Path: repo://ttmp/2026/08/16/PV-SCROLL-REVIEW-025--fundamental-architecture-review-of-history-scroll-restoration/sources/tanstack-router-scroll-restoration.md
      Note: Confirms per-entry key pattern
    - Path: repo://web/src/App.tsx
      Note: Route boundaries that determine NotePage lifetime
    - Path: repo://web/src/components/atoms/ScrollArea/ScrollArea.tsx
      Note: Physical nested scroller and proposed forwarded-ref API
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: Two-phase HTML rendering and competing fragment scroll behavior
    - Path: repo://web/src/components/organisms/NoteView/noteEnhancements.ts
      Note: Native fragment mutation and asynchronous embed growth
    - Path: repo://web/src/components/pages/NotePage/NotePage.tsx
      Note: Route-local hook placement, loading branches, and responsive scroller ownership
    - Path: repo://web/src/entry-client.tsx
      Note: Hydration, Strict Mode, BrowserRouter, and provider placement
    - Path: repo://web/src/lib/scrollRestoration.ts
      Note: PR baseline implementation and uncommitted experiments under review
ExternalSources:
    - https://github.com/go-go-golems/publish-vault/pull/21
    - https://reactrouter.com/api/components/ScrollRestoration
    - https://reactrouter.com/api/hooks/useNavigationType
    - https://developer.mozilla.org/en-US/docs/Web/API/History/scrollRestoration
    - https://developer.mozilla.org/en-US/docs/Web/API/Element/scrollHeight
    - https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserver
    - https://developer.mozilla.org/en-US/docs/Web/API/Window/sessionStorage
    - https://html.spec.whatwg.org/multipage/browsing-the-web.html
Summary: Fundamental review of PR 21 and its failed follow-up experiments. Defines session-history entry identity, scroll invariants, lifecycle ownership, navigation policy, readiness/convergence, explicit scroller registration, and a phased implementation and browser-test plan.
LastUpdated: 2026-08-16T22:34:25-04:00
WhatFor: Use this document to replace the effect-driven PR 21 scroll restoration with a history-entry-centered design and to onboard an engineer implementing or reviewing it.
WhenToUse: Read before changing note routing, fragment navigation, nested scrollers, NotePage loading behavior, or back/forward restoration.
---












# Scroll restoration architecture and PR 21 code review

## 1. Executive summary

PR 21 correctly identifies the visible problem: the application scrolls a nested note container rather than `window`, so native window scroll restoration cannot restore the reader’s position. It also contains several good discoveries: the note `ScrollArea` is the physical scroller, the scroller disappears during loading, and explicit fragments must take priority over a generic “start at top” rule.

The implementation is nevertheless not safe to merge in its current form. The three GitHub review comments are valid, but they are not independent line-level mistakes. They are consequences of four missing abstractions:

1. **History-entry identity.** Scroll state belongs to one session-history entry, not to a note slug, pathname, fragment, or component instance.
2. **Persistent ownership.** The owner of scroll snapshots must outlive every route whose position it remembers.
3. **Navigation intent.** Arrival policy depends on whether navigation is `POP`, `PUSH`, or `REPLACE`; the existing pure policy does not know this.
4. **Restoration readiness.** Restoration is a convergence process over changing DOM geometry, not a single assignment after `!!note` becomes true.

The recommended design is a **persistent nested-scroll coordinator** mounted below `BrowserRouter` and above `<Routes>`. It owns a per-entry snapshot store, receives explicit registration of the active note scroller and content root, continuously captures scroll state with requestAnimationFrame throttling, and restores according to a reducer driven by `useNavigationType()`, `location.key`, and the fragment. All fragment navigation must go through React Router so every pushed history entry receives a unique router key. The coordinator retries restoration when content extent changes, using `ResizeObserver` rather than a fixed 60-frame guess.

The key policy is:

- Returning to an existing entry with `POP` and a snapshot restores that snapshot—even if the URL has a fragment, because the reader may have scrolled after initially landing on the fragment.
- A new `PUSH`/`REPLACE` with a fragment scrolls to the fragment.
- A new `PUSH`/`REPLACE` without a fragment scrolls to top.
- A `POP` without a snapshot falls back to fragment or top.

This is closer to native browser semantics than the PR’s unconditional `hash > restore > top` rule.

### Recommendation

Do not patch the current hook incrementally. Preserve the useful pure-policy tests, but replace the hook with the coordinator pattern described here. Before implementation, add a deterministic browser harness that proves the six essential flows: note-to-note Back, note-to-search Back, same-note fragment Back, Forward to a fragment entry, delayed content growth, and responsive scroller replacement.

---

## 2. Scope and evidence model

This review covers:

- PR 21 at commit `8e2774063e8432f0c9644db382b2bdfe2ab41412`.
- The three inline GitHub review findings.
- Existing ticket documents `PV-SCROLL-023` and `PV-SCROLL-024`.
- Current routing, SSR/hydration, note rendering, fragment handling, and responsive scroller code.
- The uncommitted follow-up experiments in `web/src/lib/scrollRestoration.ts` and `.test.ts`.
- Browser observations recorded during the attempted review fixes.

Three evidence layers must remain separate:

1. **Committed baseline:** reproducible with `git show HEAD:<path>` and safe to cite as PR behavior.
2. **Uncommitted experiment:** useful for learning, but not an accepted implementation and not a stable source of line numbers.
3. **Observed browser probe:** evidence of one run under one viewport/build; useful only when the harness and DOM target are unambiguous.

A recurring mistake in the prior debugging was promotion of layer 2 or 3 into an architectural fact. For example, seeing four `.note-scroll` nodes and `root.contains(visibleScroller) === false` suggested more than one rendered tree, but it did not identify each node’s React owner or prove duplicate `NotePage` instances. `entry-client.tsx:61-73` does show `React.StrictMode`, SSR hydration, and route-component preloading, all of which warrant controlled investigation. None, by itself, proves duplicate live UI trees. The correct response is to add instance identifiers and a deterministic test—not to redesign ownership around an unproven explanation.

---

## 3. System orientation for a new intern

### 3.1 Route tree and component lifetime

`AppRoutes` keeps the application shell (`VaultLayout`) mounted around route content (`web/src/App.tsx:44-101`). The relevant routes are:

```text
/          -> HomeRedirect -> NotePageComponent(homeSlug)
/note/*    -> NoteRoute    -> NotePageComponent(slug)
/search    -> SearchPage
/w/:pageId -> WidgetPage
```

Evidence: `web/src/App.tsx:67-85`, `:104-136`, and `:194-211`.

This produces two different lifetime cases:

- **Note A -> Note B:** both URLs match `/note/*`; React may reuse the same `NotePage` component type while its `slug` changes. The article subtree can still disappear because `NotePage` returns a loading branch.
- **Note -> Search:** the matching route element changes from `NoteRoute` to `SearchRoute`; `NotePage` unmounts. Every `useRef` and `useState` owned by it is destroyed.

The PR correctly reasons about the first case but incorrectly generalizes “persistent `/note/*` route element” to the second. A component can be persistent within one route branch and still be ephemeral relative to the application’s session history.

### 3.2 The physical scroller

`ScrollArea` is only a thin `<div>` wrapper (`web/src/components/atoms/ScrollArea/ScrollArea.tsx:15-30`). `NotePage` applies `note-scroll` to:

- the desktop split-pane article (`NotePage.tsx:146-169`),
- the desktop no-right-panel article (`:170-181`), and
- the mobile article (`:183-194`).

Desktop and mobile branches are both rendered under the wrapper at `NotePage.tsx:196-202`; CSS selects the visible branch. When the right panel toggles, the desktop scroller changes structural branch. Therefore the active scroll DOM node is not a stable singleton.

The app shell itself is fixed-height and overflow-hidden (`VaultLayout.tsx:120-121`, `:199-200`). That is why restoring `window.scrollY` is insufficient. PR 21 is correct on this point.

### 3.3 Loading removes the scroller

The hook is called before early returns (`NotePage.tsx:60-66`), but the wrapper and `ScrollArea` nodes are rendered only after loading/error checks. `isLoading` returns a spinner at `NotePage.tsx:99-106`; `isError || !note` returns an error at `:108-116`.

Consequently:

```text
NotePage hook instance: may survive Note A -> Note B
article ScrollArea:     may disappear during fetch
NotePage hook instance: does not survive Note -> Search
```

This distinction drives ownership. The active element ref is ephemeral. The snapshot store cannot be.

### 3.4 Content is not geometrically stable when `note` exists

`ready = !!note` means “the API returned a note,” not “scroll geometry is final.” After note data exists:

1. `NoteHtml` initially renders raw server HTML (`NoteHtml.tsx:65-72`).
2. A passive effect resolves wiki links and replaces `resolvedHtml` (`:74-76`).
3. Math, Mermaid, code highlighting, embeds, and anchors run in separate effects (`:157-197`).
4. Embeds fetch and append HTML asynchronously (`noteEnhancements.ts:216-242`).
5. Images and fonts can alter height outside these explicit effects.

Therefore `!!note` is a data-readiness predicate, not a restoration-readiness predicate.

### 3.5 Fragment navigation has two owners

Cross-note wiki links flow through React Router:

```text
NoteHtml.handleClick
  -> onWikiLinkNavigate(target + hash)
  -> NotePage.handleNavigate
  -> navigate(`/note/${targetSlug}`)
```

Evidence: `NoteHtml.tsx:125-145` and `NotePage.tsx:39-46`.

Generated heading permalinks bypass React Router:

```ts
window.location.hash = id;
heading.scrollIntoView({ behavior: "smooth", block: "start" });
```

Evidence: `noteEnhancements.ts:185-200`.

A third fragment behavior exists in `NoteHtml`: after `slug` or `resolvedHtml` changes, a 200 ms timer reads `window.location.hash` and calls `scrollIntoView` (`NoteHtml.tsx:199-210`). Its dependencies do not include the hash itself.

The result is split ownership:

- Router navigation owns some fragments.
- Native `window.location.hash` owns heading permalinks.
- `NoteHtml` owns a delayed fragment scroll.
- `useScrollRestoration` owns generic top/restore behavior.

These mechanisms can race because they have no common state machine.

### 3.6 SSR and hydration constraints

The server renders the real route tree under `StaticRouter` and preloads RTK Query state (`entry-server.tsx:95-117`). The client preloads the initial note module to preserve hydration identity, wraps the app in `React.StrictMode`, and chooses `hydrateRoot` versus `createRoot` based on existing SSR markup (`entry-client.tsx:49-85`).

Implications:

- A module-level mutable map is not automatically wrong on the browser, but it has poor test isolation and HMR behavior.
- A server-shared module map would be unsafe if mutated during SSR. Effects currently do not run on the server, but future refactors could violate that assumption.
- An application provider created inside each browser root naturally has the right lifetime and isolates SSR requests/tests.
- Strict Mode requires effect setup and cleanup to be idempotent.

---

## 4. Formal model

### 4.1 Session history

Let a browser tab’s session history be an ordered sequence:

```text
H = <e_0, e_1, ..., e_n>
```

Each `e_i` is a distinct history entry, even when two entries have the same URL. Define:

```text
e_i = (id_i, url_i, navigation_action_i)
url_i = (pathname_i, search_i, hash_i)
```

The required identity property is injectivity:

```text
i != j  =>  id_i != id_j
```

A pathname, slug, hash, or `(location.key, hash)` pair is not inherently injective. The user can visit the same URL repeatedly. React Router’s `location.key` is the appropriate entry identity only when all entry-creating navigation goes through the router. Native fragment mutation weakens that contract.

### 4.2 Scroll geometry

At time `t`, a scroll container has:

```text
C_t = (y_t, h_t, v_t)
```

where:

- `y_t = scrollTop`,
- `h_t = scrollHeight`,
- `v_t = clientHeight`.

The feasible scroll interval is:

```text
0 <= y_t <= M_t
M_t = max(0, h_t - v_t)
```

For a saved target `y*`, exact restoration is feasible iff:

```text
y* <= M_t
```

This proves GitHub review comment 3792846820. The PR checks `h_t >= y*`; that is weaker and can be true while `M_t < y*`. Assignment then clamps to `M_t`, and the one-shot loop stops too early.

### 4.3 Snapshot record

The minimal record is:

```ts
interface ScrollSnapshot {
  entryKey: string;
  scope: "note";
  y: number;
  capturedAt: number;
}
```

A stronger future record adds semantic anchoring:

```ts
interface SemanticScrollSnapshot extends ScrollSnapshot {
  anchorId?: string;          // first stable visible heading/block
  anchorViewportOffset?: number;
  contentRevision?: string;
}
```

Absolute pixels are sufficient for an unchanged document. An anchor plus viewport offset is more robust when embeds or images above the viewport change height.

### 4.4 Core invariants

A correct implementation should make these invariants executable in tests.

#### Invariant I1: entry isolation

```text
snapshot(e_i) never overwrites snapshot(e_j) when i != j
```

Repeated visits to the same URL remain separate.

#### Invariant I2: ownership lifetime

```text
lifetime(snapshotStore) >= lifetime(session navigation coordinator)
```

Specifically, it must outlive `/note/* -> /search -> Back`.

#### Invariant I3: source capture precedes destruction

Before the active scroller is removed or repurposed:

```text
store[id_current] = latest observed scroll state for id_current
```

Continuous capture plus cleanup provides this guarantee without intercepting every navigation source.

#### Invariant I4: one owner applies arrival scrolling

For each arrival, exactly one of these actions owns the viewport:

```text
RESTORE(snapshot) | FRAGMENT(hash) | TOP | NONE
```

`NoteHtml` and a restoration hook must not independently schedule competing scrolls.

#### Invariant I5: navigation type participates in policy

A `POP` revisits an existing history entry. A `PUSH` creates a new one. They cannot share one unconditional hash-first policy.

#### Invariant I6: eventual convergence

If target `y*` becomes feasible before deadline `T`, then:

```text
exists t <= T: abs(scrollTop_t - y*) <= epsilon
```

If it never becomes feasible, final state is bounded:

```text
scrollTop_T = clamp(y*, 0, M_T)
```

#### Invariant I7: cleanup is idempotent

Strict Mode setup/cleanup/setup cannot duplicate listeners, observers, or writes.

---

## 5. Arrival policy as a reducer

The PR’s policy is:

```text
non-note -> none
hash     -> fragment
saved    -> restore
else     -> top
```

That rule misses navigation type. Consider a history entry `/note/a#section`: the reader lands on the section, then scrolls farther down, navigates away, and presses Back. Native-like behavior restores the reader’s later position. “Hash always wins” sends them back to the original section and discards history state.

Use this reducer instead:

```ts
type NavigationType = "POP" | "PUSH" | "REPLACE";
type ArrivalAction =
  | { kind: "restore"; snapshot: ScrollSnapshot }
  | { kind: "fragment"; hash: string }
  | { kind: "top" }
  | { kind: "none" };

function decideArrival(input: {
  scopeMatches: boolean;
  navigationType: NavigationType;
  hash: string;
  snapshot?: ScrollSnapshot;
}): ArrivalAction {
  if (!input.scopeMatches) return { kind: "none" };

  if (input.navigationType === "POP" && input.snapshot) {
    return { kind: "restore", snapshot: input.snapshot };
  }

  if (input.hash) {
    return { kind: "fragment", hash: input.hash };
  }

  return { kind: "top" };
}
```

Truth table:

| Navigation | Snapshot | Hash | Action | Rationale |
|---|---:|---:|---|---|
| `POP` | yes | either | restore | Return to the exact prior entry state. |
| `POP` | no | yes | fragment | No state exists; honor the URL. |
| `POP` | no | no | top | Defined fallback. |
| `PUSH` | irrelevant/new | yes | fragment | New explicit destination. |
| `PUSH` | irrelevant/new | no | top | New page starts at top. |
| `REPLACE` | irrelevant/new | yes | fragment | Explicit replacement destination. |
| `REPLACE` | irrelevant/new | no | top | Replacement without fragment starts at top. |

React Router exposes this distinction through `useNavigationType()`.

---

## 6. PR 21 code review findings

### Finding 1 — component-local storage cannot survive route exit

- **Severity:** blocking.
- **Evidence:** baseline `scrollRestoration.ts:85` creates `saved` with `useRef`; `/search` is a sibling route (`App.tsx:78-83`).
- **Failure:** Note -> Search unmounts `NotePage`; its map, `lastOffset`, and `lastKey` disappear. Back creates a fresh empty map. Meanwhile `history.scrollRestoration` remains `manual` (`scrollRestoration.ts:89-95`), disabling browser fallback.
- **Review comment:** [3792846817](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846817).
- **Required correction:** persistent coordinator above `<Routes>`; save continuously and during element cleanup.

### Finding 2 — `location.key` does not model native fragment entries

- **Severity:** blocking.
- **Evidence:** save effect depends only on `location.key` (`scrollRestoration.ts:102-108`); heading anchors directly set `window.location.hash` (`noteEnhancements.ts:196-200`).
- **Failure:** same-note fragment traversal can create a history step without changing the key observed by the save effect. Back to the unhashed state has no pre-jump snapshot.
- **Review comment:** [3792846818](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846818).
- **Required correction:** stop creating history entries outside React Router. Use router-generated entry keys rather than inventing `(key, hash)` as a surrogate.

Why not simply use `key + hash`? Because repeated visits to the same fragment can still collide:

```text
A -> A#x -> A -> A#x
```

If native hash navigation reuses the router key, both `A#x` entries have the same composite identity. The fundamental repair is unique entry creation, not a larger non-unique string.

### Finding 3 — wrong feasibility predicate

- **Severity:** blocking.
- **Evidence:** baseline `scrollRestoration.ts:128-137` stops when `scrollHeight >= offset`.
- **Failure:** maximum valid `scrollTop` is `scrollHeight - clientHeight`. The assignment clamps early and the loop never reapplies after growth.
- **Review comment:** [3792846820](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846820).
- **Required correction:** observe content extent and converge using `maxScrollTop >= target`, then verify the applied value.

### Finding 4 — arrival policy ignores `POP`/`PUSH`/`REPLACE`

- **Severity:** blocking semantic gap.
- **Evidence:** `pickScrollAction` accepts only pathname, hash, key, and saved map (`scrollRestoration.ts:43-50`).
- **Failure:** a fragment in an existing entry always wins over a later user scroll; a fresh push and a history traversal are conflated.
- **Correction:** use `useNavigationType()` and the reducer in section 5.

### Finding 5 — `!!note` is not restoration readiness

- **Severity:** blocking for dynamic notes.
- **Evidence:** `ready` is passed as `!!note` (`NotePage.tsx:65-66`), while HTML replacement and enhancement continue after note data exists (`NoteHtml.tsx:65-76`, `:157-210`; `noteEnhancements.ts:216-242`).
- **Failure:** restore can run before final geometry, then be clamped or displaced.
- **Correction:** explicit content root registration plus `ResizeObserver`/content revision; restoration remains active until convergence or deadline.

### Finding 6 — competing scroll owners

- **Severity:** blocking race.
- **Evidence:** the restoration hook schedules rAF; `NoteHtml` schedules a 200 ms fragment scroll; heading anchors scroll immediately and smoothly.
- **Failure:** result depends on timing rather than policy.
- **Correction:** one coordinator owns `RESTORE`, `FRAGMENT`, and `TOP`. Rendering code reports readiness; it does not independently scroll.

### Finding 7 — DOM discovery by global class/visibility is a weak contract

- **Severity:** high.
- **Evidence:** `findVisibleScroller` queries `.note-scroll` and selects `clientHeight > 0` (`scrollRestoration.ts:53-69`). Responsive branches and right-panel toggles replace candidate nodes (`NotePage.tsx:145-194`).
- **Failure:** effect timing, responsive replacement, and duplicate/transitional DOM can produce a null or wrong element. Exploratory debugging observed candidate-count ambiguity but did not establish ownership.
- **Correction:** `ScrollArea` must forward a ref; the active note scroller registers explicitly with a scope and instance identifier.

### Finding 8 — `history.scrollRestoration = "manual"` has global consequences

- **Severity:** high.
- **Evidence:** baseline sets manual mode once inside `NotePage` and never restores the prior value (`scrollRestoration.ts:89-95`).
- **Failure:** after `NotePage` unmounts, the page remains in manual mode even though the owner that could restore note state is gone.
- **Correction:** the persistent coordinator sets manual mode for its full application lifetime and restores the previous mode on provider cleanup.

### Finding 9 — rAF count is a timer, not readiness

- **Severity:** high.
- **Evidence:** fixed 60-frame bailout (`scrollRestoration.ts:127-137`).
- **Failure:** 60 frames is refresh-rate dependent, unrelated to network-loaded embeds, and can expire before content settles. Conversely it can repeatedly write while a user starts interacting.
- **Correction:** event-driven retry (`ResizeObserver`, content revision, image load), plus a real-time deadline and cancellation on user input/navigation.

### Finding 10 — tests prove only a small pure function

- **Severity:** blocking process gap.
- **Evidence:** baseline tests cover only `pickScrollAction`; Vitest uses `environment: "node"` (`vitest.config.ts:18-22`). The file itself says the DOM hook is manual-only.
- **Failure:** all three GitHub findings involve lifecycle, history, or geometry and are invisible to current tests.
- **Correction:** reducer unit tests plus browser-level Playwright tests. jsdom cannot supply trustworthy layout metrics.

### Finding 11 — documentation and code had already diverged

- **Severity:** high process concern.
- **Evidence:** the PV-SCROLL-024 design says “all `useLayoutEffect`,” but the committed file imports and uses only `useEffect`. The diary’s Step 2 records an end-to-end failure and states that the selected fix was “not yet implemented,” yet the PR was reviewed at that state.
- **Correction:** a known failed decisive test must block PR readiness. Update design status from accepted to superseded when evidence invalidates it.

### Finding 12 — storage policy is underspecified

- **Severity:** medium.
- **Evidence:** no eviction, reload, tab, HMR, or test-isolation contract.
- **Correction:** provider-local memory map for the current tab/root; optional `sessionStorage` adapter only if reload restoration is explicitly required. Bound retained entries or remove snapshots no longer reachable where practical.

---

## 7. Review of the prior reasoning and experiments

### What was done well

1. **The physical scroller was measured rather than assumed.** The prior work corrected an earlier belief that `<main>` scrolled.
2. **The scroller lifetime was investigated.** Discovering that loading removes the `ScrollArea` is important.
3. **Policy was extracted into a pure function.** This is a useful pattern, even though the policy inputs were incomplete.
4. **Browser behavior was tested.** The same-note fragment failure (`150 -> fragment -> Back -> 0`) directly validated review comment 3792846818.
5. **Failures were recorded.** PV-SCROLL-024’s diary clearly states that the implementation failed the real-vault Back test.

### Where the reasoning drifted

1. **Component lifetime was mistaken for application lifetime.** “Persistent `/note/*` element” solved loading within a note route but not route exit to Search.
2. **Symptoms were patched before defining entry identity.** `key + hash` was proposed without proving uniqueness across repeated identical fragment entries.
3. **Effect ordering became the design language.** Statements such as “save effect runs before restore effect” are implementation details, not ownership guarantees. React commit, browser history, DOM mutation, scroll events, and async enhancement form separate timelines.
4. **Readiness was guessed from rendering phase.** Switching between `useEffect`, `useLayoutEffect`, and rAF retries attempted to find a lucky schedule instead of declaring a readiness contract.
5. **Exploratory DOM evidence was overinterpreted.** Four `.note-scroll` nodes and a failed `contains()` check suggested duplicate trees, but no per-instance markers or React-owner proof existed. The correct conclusion was “the selector/ref assumptions are unproven.”
6. **The experiment accumulated mechanisms.** Module globals, hash ticks, layout effects, capture listeners, rAF discovery, and debug globals increased state-space while the basic acceptance flow regressed.
7. **Pure tests provided false comfort.** Twelve passing tests still allowed note-to-note Back to regress because no test exercised a real scroller or route unmount.

### Code-review verdict on the uncommitted worktree

The worktree must not be committed. Useful ideas are present—the module-lifetime issue, composite identity concern, and corrected max-scroll predicate—but the implementation:

- introduces debug globals,
- relies on an isomorphic layout-effect alias without a proven timing contract,
- splits listener/save/restore into interacting effects,
- adds rAF discovery timers,
- still lacks navigation type,
- still permits two fragment owners,
- and regressed a previously working note-to-note flow during probes.

Treat it as a laboratory notebook, not a candidate patch.

---

## 8. Proposed architecture

### 8.1 Component diagram

```text
BrowserRouter
└── ScrollRestorationProvider          persistent across all routes
    ├── snapshot store Map<EntryKey, Snapshot>
    ├── navigation policy reducer
    ├── registered scope: note
    │   ├── active scroller element
    │   └── active content element / revision
    └── AppRoutes
        └── VaultLayout
            ├── NotePage               ephemeral on route exit
            │   └── NoteView
            │       └── NoteHtml        reports content root/readiness
            ├── SearchPage
            └── WidgetPage
```

### 8.2 Provider ownership

Mount the provider inside `BrowserRouter` (so it can use router hooks) and outside `AppRoutes` (so route switching cannot destroy it):

```tsx
<BrowserRouter>
  <ScrollRestorationProvider>
    <AppRoutes ... />
  </ScrollRestorationProvider>
</BrowserRouter>
```

Create one provider per client root. Do not use a process-global module singleton as the primary design.

### 8.3 Explicit registration

Change `ScrollArea` to forward its DOM ref:

```tsx
export const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(
  ({ children, className, ...props }, ref) => (
    <div ref={ref} className={clsx("retro-scroll", className)} {...props}>
      {children}
    </div>
  )
);
```

Register candidate note scrollers explicitly:

```ts
const desktopRef = useRestorableScroller("note", "desktop");
const mobileRef = useRestorableScroller("note", "mobile");
```

The coordinator selects the active registered element by an explicit predicate (`getClientRects().length > 0`, media state, or registration metadata). It never searches the document by class name.

### 8.4 Router-owned fragment navigation

Replace direct `window.location.hash` mutation with injected navigation:

```ts
interface HeadingAnchorOptions {
  onNavigateFragment(id: string): void;
}

function enhanceHeadingAnchors(root: HTMLElement, options: HeadingAnchorOptions) {
  // anchor click -> options.onNavigateFragment(id)
}
```

`NotePage` or the provider calls Router navigation:

```ts
navigate(
  { pathname: location.pathname, search: location.search, hash: `#${id}` },
  { replace: false }
);
```

This gives the fragment entry a router-owned `location.key`. Remove direct scrolling from the injected anchor and remove the independent 200 ms fragment scroll from `NoteHtml`; the coordinator owns fragment arrival.

This injection also preserves Storybook independence: `NoteHtml` need not call `useLocation()` when rendered outside a Router.

### 8.5 Snapshot store API

```ts
type EntryKey = string;
type ScrollScope = "note";

interface ScrollSnapshot {
  entryKey: EntryKey;
  scope: ScrollScope;
  y: number;
  capturedAt: number;
  anchorId?: string;
  anchorViewportOffset?: number;
}

interface ScrollSnapshotStore {
  read(entryKey: EntryKey, scope: ScrollScope): ScrollSnapshot | undefined;
  write(snapshot: ScrollSnapshot): void;
  delete(entryKey: EntryKey, scope: ScrollScope): void;
  clear(): void;
}
```

The initial adapter is an in-provider `Map`. An optional session-storage adapter can be added later behind the same interface.

### 8.6 Capture algorithm

Capture is continuous but throttled to one write per animation frame:

```text
on active scroller scroll:
    latestY = scroller.scrollTop
    if no capture frame scheduled:
        schedule rAF:
            store.write(entryKey, latestY)
            clear scheduled flag

on scroller ref cleanup or provider route transition:
    cancel capture frame
    store.write(entryKey, scroller.scrollTop) synchronously if element exists
```

Because the provider persists, the stored value survives Note -> Search. Because the element callback cleanup reads the actual element before forgetting it, capture does not depend on a later effect finding the old DOM.

### 8.7 Restoration controller

Restoration should be a cancelable controller, not an unstructured effect timer.

```ts
interface RestoreRequest {
  token: number;
  action: ArrivalAction;
  deadlineMs: number;
  epsilonPx: number;
}
```

Pseudocode:

```text
on arrival(location, navigationType):
    cancel previous request
    action = decideArrival(...)

    if action == TOP:
        activeScroller.scrollTop = 0
        complete

    if action == FRAGMENT:
        wait until target element exists
        align target with scroller viewport
        complete

    if action == RESTORE(snapshot):
        desired = snapshot.y
        observe content root and scroller with ResizeObserver
        attempt immediately and whenever size/content revision changes:
            maxY = max(0, scrollHeight - clientHeight)
            if semantic anchor exists:
                candidate = anchorTop - anchorViewportOffset
            else:
                candidate = desired

            if desired <= maxY + epsilon:
                scrollTop = clamp(candidate, 0, maxY)
                verify on next frame
                if within epsilon: complete
            else if deadline expired:
                scrollTop = maxY
                complete with degraded result
```

Cancellation triggers:

- new location/action,
- active scroller replacement,
- provider unmount,
- explicit user input after restoration begins (wheel/touch/pointer/keyboard), depending on UX policy.

### 8.8 Semantic anchoring as Phase 2

Absolute `scrollTop` preserves pixels, not reading position, when content above changes. A robust advanced strategy captures the first visible heading or block:

```text
anchor = stable element at viewport top
snapshot.anchorId = anchor.id
snapshot.anchorViewportOffset = anchor.top - scroller.top
```

On restore:

```text
targetY = currentAnchorOffsetTop - snapshot.anchorViewportOffset
```

Use absolute `y` as fallback. This is especially valuable for asynchronously resolved embeds.

---

## 9. Decision records

### Decision: provider above routes owns restoration

- **Context:** snapshots must survive Note -> Search and Strict Mode effect cycles.
- **Options considered:** component refs in `NotePage`; module-level singleton; provider above routes.
- **Decision:** provider below Router and above Routes.
- **Rationale:** correct browser-root lifetime, access to router hooks, test isolation, no SSR process-global state.
- **Consequences:** introduces a context/API, but removes hidden cross-route coupling.
- **Status:** proposed.

### Decision: Router creates every fragment history entry

- **Context:** native `window.location.hash` can bypass router entry identity.
- **Options considered:** composite `key + hash`; custom IDs in `history.state`; route all fragment pushes through Router.
- **Decision:** route fragment navigation through Router.
- **Rationale:** one history owner and unique `location.key`; avoids maintaining parallel identity logic.
- **Consequences:** heading enhancement receives an injected callback; direct `scrollIntoView` is removed.
- **Status:** proposed.

### Decision: navigation type precedes fragment in POP policy

- **Context:** an existing fragment entry may have a later user scroll snapshot.
- **Options considered:** unconditional hash precedence; snapshot precedence on POP only.
- **Decision:** `POP + snapshot -> restore`; new entries with hash -> fragment.
- **Rationale:** matches history-entry semantics and preserves user action after initial fragment landing.
- **Consequences:** pure-policy tests need navigation type.
- **Status:** proposed.

### Decision: explicit refs replace class-query discovery

- **Context:** desktop/mobile and right-panel branches produce multiple or replaced scrollers.
- **Options considered:** global/class query; wrapper query; forwarded refs and registration.
- **Decision:** forwarded refs plus scoped registration.
- **Rationale:** establishes element ownership and makes tests deterministic.
- **Consequences:** touches `ScrollArea` and `NotePage`; small API expansion.
- **Status:** proposed.

### Decision: event-driven convergence replaces fixed rAF count

- **Context:** embeds and rendering effects alter geometry on non-frame-related timelines.
- **Options considered:** 60-frame poll; MutationObserver; ResizeObserver plus explicit content revision.
- **Decision:** ResizeObserver/content revision with real-time deadline.
- **Rationale:** reacts to the property that matters—extent/anchor geometry—rather than elapsed frames.
- **Consequences:** controller needs cancellation and cleanup; browser tests must cover growth.
- **Status:** proposed.

### Decision: provider-local memory is the initial persistence adapter

- **Context:** requirement is Back/Forward within one page session; reload behavior is unspecified.
- **Options considered:** module Map; Redux; provider Map; sessionStorage.
- **Decision:** provider-local Map; optional sessionStorage adapter later.
- **Rationale:** minimal correct lifetime without serialization or global test leakage.
- **Consequences:** hard reload does not restore unless a later product decision enables the adapter.
- **Status:** proposed.

---

## 10. Phased implementation plan

### Phase 0 — restore a clean baseline

1. Preserve the uncommitted experiment as review evidence if desired, then discard it from the implementation branch.
2. Confirm `git diff` contains only intentional ticket docs before coding.
3. Keep PR 21 open but mark the scroll implementation as not ready.

### Phase 1 — deterministic test harness

1. Add pure reducer tests for `decideArrival` with `POP/PUSH/REPLACE`.
2. Add Playwright browser tests using a fixture vault with:
   - two long notes,
   - a same-note heading link,
   - a tag/search navigation,
   - a delayed embed/content-growth fixture.
3. Add `data-testid`/instance identifiers to the note scrollers and route root.
4. Prove the existing failures before implementation.

Do not proceed without red tests for note-to-search Back and fragment Back.

### Phase 2 — provider and store

1. Add `ScrollRestorationProvider` above `AppRoutes` in both client and server trees.
2. Add provider-local `Map` store and test-only inspection adapter.
3. Set `history.scrollRestoration = "manual"` in the provider and restore its prior value on cleanup.
4. Use `useNavigationType()` and `location.key`.

### Phase 3 — explicit scroller registration

1. Convert `ScrollArea` to `forwardRef`.
2. Register desktop/mobile note scrollers with scope/instance metadata.
3. Register a content root suitable for `ResizeObserver`.
4. Remove `findVisibleScroller` and `.note-scroll` as behavioral contracts; CSS classes may remain for styling/debugging.

### Phase 4 — unify fragment navigation

1. Inject `onNavigateFragment` into `enhanceHeadingAnchors`.
2. Route fragments through `navigate()`.
3. Move fragment alignment into the coordinator.
4. Remove `NoteHtml`’s independent 200 ms scroll timer.

### Phase 5 — convergence controller

1. Implement reducer-selected actions.
2. Restore absolute `y` with max-scroll feasibility.
3. Observe content/scroller geometry with `ResizeObserver`.
4. Add deadline, cancellation, verification, and degraded clamp behavior.

### Phase 6 — hardening

1. Test responsive breakpoint changes and right-panel toggles.
2. Test Strict Mode setup/cleanup idempotence.
3. Test SSR hydration without warnings or duplicate listeners.
4. Optionally add semantic anchor snapshots.
5. Decide and document reload/sessionStorage policy.

---

## 11. Test matrix

| ID | Initial state | Action | Required result |
|---|---|---|---|
| T1 | Note A at `y=500` | PUSH Note B, then Back | `POP` A restores 500. |
| T2 | Note A at `y=500` | PUSH Search, then Back | Provider survives route exit; A restores 500. |
| T3 | Note A at `y=500` | PUSH `#heading`, then Back | Unhashed entry restores 500. |
| T4 | On `#heading`, user scrolls to `y=900` | PUSH away, then Back | `POP + snapshot` restores 900, not heading. |
| T5 | Back to target `y=700`, initial max 200 | Content grows max to 900 | Controller eventually reaches 700. |
| T6 | Back to target `y=700`, final max 400 | Deadline expires | Final y is 400; request completes degraded. |
| T7 | Desktop y saved | viewport switches to mobile | Active registered scroller receives clamped/anchor restore. |
| T8 | Right panel toggles | scroller DOM is replaced | Snapshot persists; new active element registers. |
| T9 | Strict Mode setup/cleanup/setup | scroll and navigate | Exactly one logical listener/observer; no duplicate writes. |
| T10 | SSR initial `/note/a#h` | hydrate | No server mutation; initial fragment handled once. |
| T11 | Forward after Back | POP existing target entry | Stored target entry position restored. |
| T12 | Same URL visited twice | different positions | Unique entry keys preserve distinct snapshots. |

Browser assertions should inspect the explicitly registered scroller, never “the first `.note-scroll` globally.”

---

## 12. Intern implementation checklist

Start here:

1. Read `App.tsx` routes and draw component lifetimes.
2. Read `NotePage.tsx` early returns and all three scroller branches.
3. Read `NoteHtml.tsx` effects and `noteEnhancements.ts` fragment mutation.
4. Run the red Playwright matrix before writing coordinator code.
5. Implement the pure reducer and store independently from DOM effects.
6. Add provider wiring in both client and SSR trees.
7. Add explicit refs and registration.
8. Unify fragment ownership.
9. Add convergence/ResizeObserver last.

Review questions:

- Can two distinct history entries ever share the same store key?
- Does the snapshot store survive every route represented in browser Back?
- Can any component besides the coordinator call `scrollIntoView` or assign `scrollTop` during arrival?
- What event proves restoration can succeed?
- What cancels a stale restoration request?
- Is every listener/observer removed under Strict Mode cleanup?
- Which browser test fails if any invariant regresses?

---

## 13. Alternatives considered

### Use React Router `<ScrollRestoration>` directly

React Router’s component is valuable reference material and supplies `getKey` patterns, but it restores document/window scrolling. This application’s reading position lives in a nested responsive scroller. Direct adoption does not solve element registration or dynamic content readiness.

### Change the app so `window` scrolls

This would recover native browser behavior and simplify the feature substantially. It is architecturally attractive, but it changes the fixed shell, resizable panels, and mobile layout. Treat it as a separate product/layout decision, not a patch inside PR 21.

### Module-level singleton map

This is a smaller patch and fixes Note -> Search lifetime in the browser. It remains weaker than a provider because it leaks across tests/HMR, hides ownership, and can become unsafe if mutated in server code. It is acceptable only as an explicitly temporary implementation.

### Redux store

Redux would provide persistence across routes and observability, but scroll positions are ephemeral UI mechanics with high-frequency writes. A dedicated provider avoids global application-state noise. Redux is reasonable if the application already standardizes all transient navigation state there.

### `sessionStorage`

This adds reload persistence but does not solve entry identity, scroller registration, or readiness. It is a storage adapter, not an architecture.

### Fixed rAF polling

Simple but timing-based. It cannot express content readiness and is refresh-rate/network dependent. Use observers and explicit signals.

---

## 14. Open questions

1. Should hard reload restore the current entry or start at top? Product decision required before enabling session storage.
2. Should a responsive breakpoint change preserve pixels, scroll ratio, or semantic anchor? Semantic anchor is best for reading continuity.
3. What content element can reliably be observed for embed/image height changes?
4. Should user input cancel an in-progress restoration immediately? Recommended: yes after the first applied attempt.
5. How long should the degraded deadline be, and should content remain visible while restoration is pending?
6. Is window scrolling a viable future simplification of the app shell?

---

## 16. Comparison to external packages (web research)

This section compares the proposed architecture (sections 8-9) against the
external solutions surveyed in `sources/` (TanStack Router, the react-router
#9495 discussion, and the SPA scroll-restoration articles). Its purpose is to
answer two questions a reviewer will ask: how much does our design diverge from
established packages, and what can we port from them? The comparison is grounded
in the saved source documents, not in recall of their APIs.

### 16.1 The reference points

**TanStack Router** is the closest solved system. Its scroll-restoration model
(`sources/tanstack-router-scroll-restoration.md`) provides:

- `scrollToTopSelectors` — nested scrollable areas (divs, shadow DOM) the router
  scrolls to top in addition to `window`.
- A per-history-entry unique key, `location.state.__TSR_key`, with the older
  `state.key` explicitly **deprecated** as insufficiently unique.
- `data-scroll-restoration-id` + `useElementScrollRestoration` — explicit
  registration of a scrollable element by a stable id, picked up by a watcher.
- `resetScroll={false}` on `<Link>` / `navigate` / `redirect` — a per-navigation
  opt-out of restore-or-top.
- Restore "after successful navigations before DOM paint."

**react-router discussion #9495**
(`sources/react-router-discussion-9495-non-window-scrollrestoration.md`) is the
maintainer-acknowledged gap. The maintainer (`@brophdawg11`) raises our exact
failure mode verbatim: *"what if when you come back to that location the
scrollElement is no longer present for some reason? falling back on window
isn't going to be right since the scroll position is relative to the
no-longer-rendered element."* The discussion establishes that `window`-only
`<ScrollRestoration>` breaks CSS-grid layouts "where only one pane scrolls," and
that a community user hit `elementRef.current is still empty when it's time to
restore` — the same ref-not-ready failure this app's prior experiments hit.

### 16.2 Where they agree (the foundation both get right)

Both the proposed design and TanStack converge on three load-bearing decisions,
which is strong evidence those are correct:

1. **Per-history-entry unique key**, not pathname or slug. TanStack ships
   `__TSR_key` and deprecated `state.key`; the proposed design says the router
   must mint every entry, including fragments. Same conclusion reached
   independently.
2. **Explicit nested-element registration**, not `window`. TanStack uses
   `data-scroll-restoration-id`; the proposed design uses forwarded ref +
   scoped registration. Same conclusion, and it is the exact gap #9495 names as
   unsolved for grid layouts.
3. **Restore before paint.** TanStack lists it as a mechanism; the proposed
   design uses a layout effect for the save. Same timing intent.

### 16.3 Where the proposed design is stronger (for this app)

Two dimensions where the packages are weaker, because this application hits edge
cases they do not model:

1. **Navigation-type policy (the "back from a heading" case).** This is the
   design's strongest original contribution and the one place it is ahead of
   both packages. TanStack's `resetScroll={false}` is a per-navigation opt-out
   the *caller* must remember; it does not encode "POP to an existing entry with
   a snapshot restores the snapshot even if the URL has a fragment." The
   proposed reducer (section 5) makes `POP + snapshot -> restore` take precedence
   over `fragment` by default. This is the correct native-browser semantics for
   the flow: land on `#heading`, scroll down to `y=900`, navigate away, press
   Back -> land at `y=900`, not back at the heading. TanStack would restore the
   heading unless every link opted out. For a note reader where
   fragment-then-scroll is common, the default matters.
2. **Fragment ownership / async-content convergence.** TanStack's
   `scrollRestoration: true` is router-timed ("after successful navigations
   before DOM paint"). This application's content is *not* stable at that point:
   `NoteHtml`'s `resolveWikiLinks` rewrites `innerHTML` after first paint, and
   embeds fetch on a network timeline. A before-paint restore would set the
   offset, then the innerHTML rewrite would reset it to 0 — the exact
   PV-SCROLL-024 Step 2 failure. The proposed design's `ResizeObserver` +
   content-revision + deadline converges *after* geometry stabilizes. TanStack
   assumes content is ready when the router says navigation succeeded; the
   proposed design does not. For a note viewer with async embeds, the proposed
   model is more correct.

### 16.4 Where TanStack is stronger (and what to port)

1. **It exists, is tested, and is maintained.** The proposed design is a design
   doc. TanStack's is shipping code, and its deprecation cycle
   (`state.key -> __TSR_key`) is evidence it hit and fixed the same uniqueness
   bug the proposed design theorizes about. That deprecation note is reason to
   trust the per-entry-key conclusion and to distrust composite surrogates.
2. **The element-registration mechanism is cleaner.**
   `data-scroll-restoration-id` as a DOM attribute picked up by a watcher is
   less wiring than forwarded-ref + context registration, and it is the pattern
   the #9495 community is converging toward. The proposed forwarded-ref approach
   is correct but more invasive (touches `ScrollArea` and `NotePage`).
3. **Multiple independent scroll areas.** TanStack models this explicitly ("a
   scrollable sidebar and a scrollable chat area, restore both independently").
   The proposed design is single-scope (`"note"`). If the backlinks panel or
   sidebar ever need independent restoration, TanStack has it; the proposed
   design would need extension.

### 16.5 Verdict

For correctness on this application's specific flows, the proposed design is
ahead on 2 of 7 dimensions (navigation-type policy, async convergence) and even
on 3 (storage lifetime, entry-identity intent, before-paint timing). For
engineering maturity and multi-area generality, TanStack is ahead. The two are
not in conflict: the proposed design's *decisions* (POP+snapshot precedence,
event-driven convergence) are policy; TanStack's *mechanisms* (`__TSR_key`,
`data-scroll-restoration-id`, the watcher) are infrastructure. The ideal
implementation is the proposed policy on TanStack's infrastructure.

### 16.6 What to port (concrete)

Three TanStack mechanisms map cleanly onto the proposed design and replace its
weakest parts:

1. **`__TSR_key`-style per-entry key minting** instead of the
   `key + "|" + hash` surrogate. The research flagged the composite key as still
   able to collide on `A -> A#x -> A -> A#x`. TanStack solved this by having the
   router mint a unique key per entry. This application cannot use TanStack's
   exact field, but the pattern — the router mints the key, every navigation
   including `window.location.hash` goes through the router — is portable and
   removes the collision class entirely. This is the single biggest thing to
   learn from TanStack.
2. **`data-scroll-restoration-id` element registration** instead of
   `findVisibleScroller`'s `.note-scroll` class query. This is the exact
   mechanism that would have prevented the prior agent's regression (listener
   bound to the wrong or null scroller). Port: add
   `data-scroll-restoration-id="note"` to the `ScrollArea`, have the hook's
   watcher register by id instead of querying by class. Roughly 15 lines;
   removes the highest-risk part of the design.
3. **`resetScroll`-style per-navigation opt-out** as a complement to (not
   replacement for) the `POP + snapshot` reducer. The reducer handles the
   default correctly; `resetScroll` would let a specific link opt out. Low
   priority, but it is the one TanStack idea that is additive rather than
   substitutive.

### 16.7 What not to port

- **TanStack's before-paint restore timing.** It assumes
  content-ready-at-navigation-success, which is false for this application's
  async-embed notes. Keep the `ResizeObserver` convergence.
- **TanStack's router-timed "after successful navigation" trigger.** Same
  reason.

### 16.8 Relationship to the three surgical review fixes

The three review fixes being shipped now (module store + unmount save;
`key + hash` identity with hash capture; `scrollHeight - clientHeight`
predicate) are necessary and correct but partial. They close three of seven
dimensions: storage lifetime (fully), hash-only saves (as a surrogate that can
still collide), and the feasibility predicate (fully for the predicate; the
rAF-count timer remains). They do not touch the two dimensions where the
highest risk lives — scroller discovery (dimension 4, the prior agent's trap)
and navigation-type policy (dimension 3, the "back from a heading" bug). The
TanStack ports above (per-entry key minting, `data-scroll-restoration-id`
registration) are the lightweight ways to de-risk those two dimensions in a
follow-up without the full coordinator rewrite.

---

## 17. References

### Repository evidence

| Concern | File / baseline lines |
|---|---|
| PR implementation | `web/src/lib/scrollRestoration.ts` at `HEAD`, lines 1-148 |
| Pure policy tests only | `web/src/lib/scrollRestoration.test.ts` at `HEAD`, lines 1-62 |
| Hook placement and loading branch | `web/src/components/pages/NotePage/NotePage.tsx:53-116` |
| Desktop/mobile scrollers | `NotePage.tsx:145-202` |
| Route lifetime | `web/src/App.tsx:67-101`, `:194-211` |
| SSR/hydration/Strict Mode | `web/src/entry-client.tsx:49-85`; `entry-server.tsx:95-117` |
| Two-phase HTML and enhancements | `web/src/components/organisms/NoteHtml/NoteHtml.tsx:65-210` |
| Native fragment mutation | `web/src/components/organisms/NoteView/noteEnhancements.ts:181-203` |
| Async embeds | `noteEnhancements.ts:205-243` |
| Physical ScrollArea | `web/src/components/atoms/ScrollArea/ScrollArea.tsx:15-30` |
| SplitPane wrapper | `web/src/components/layout/SplitPane/SplitPane.tsx:27-50` |
| Prior design | `ttmp/2026/08/15/PV-SCROLL-024--.../design-doc/01-design-hand-rolled-scroll-restoration-option-a.md` |
| Prior failed E2E record | `ttmp/2026/08/15/PV-SCROLL-024--.../reference/01-implementation-diary.md`, Step 2 |

### GitHub review

- [Persist offsets outside the note route component](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846817)
- [Save positions when only the fragment changes](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846818)
- [Wait until the requested offset is actually scrollable](https://github.com/go-go-golems/publish-vault/pull/21#discussion_r3792846820)

### External APIs

- [React Router ScrollRestoration](https://reactrouter.com/api/components/ScrollRestoration)
- [React Router useNavigationType](https://reactrouter.com/api/hooks/useNavigationType)
- [MDN History.scrollRestoration](https://developer.mozilla.org/en-US/docs/Web/API/History/scrollRestoration)
- [MDN Element.scrollHeight](https://developer.mozilla.org/en-US/docs/Web/API/Element/scrollHeight)
- [MDN ResizeObserver](https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserver)
- [MDN sessionStorage](https://developer.mozilla.org/en-US/docs/Web/API/Window/sessionStorage)
- [WHATWG navigation and session history](https://html.spec.whatwg.org/multipage/browsing-the-web.html)
