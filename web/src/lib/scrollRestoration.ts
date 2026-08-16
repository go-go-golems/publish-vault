/**
 * Scroll restoration for the note view.
 *
 * The browser's `history.scrollRestoration` only restores *window* scroll, and
 * this app scrolls a nested `ScrollArea` (the app shell is `h-screen
 * overflow-hidden`, so `window.scrollY` is always 0 — see PV-SCROLL-023). We
 * therefore own restoration: save the note's scroll offset and re-apply it on
 * back/forward.
 *
 * ## Persistence (review: "persist offsets outside the note route component")
 *
 * Offsets live in a module-level `Map` (`savedOffsets`), NOT a `useRef` held by
 * `NotePage`. A `useRef` map is destroyed when the route switch unmounts
 * `NotePage` (e.g. note -> `/search` -> Back), so the back navigation would
 * find an empty map and scroll to top. The module-level store survives that
 * unmount. It is client-only: it is mutated solely inside effects, so SSR
 * `renderToString` (which runs no effects) never touches it. The active offset
 * is also saved on `NotePage` unmount, so the final position is captured even
 * when no trailing scroll event fired.
 *
 * ## Identity (review: "save positions when only the fragment changes")
 *
 * `scrollKey = location.key + "|" + hash`. The save effect depends on
 * `scrollKey`, so a same-note fragment change (a heading permalink sets
 * `window.location.hash` directly in `noteEnhancements.enhanceHeadingAnchors`,
 * which advances the history stack WITHOUT changing React Router's
 * `location.key`) still fires the save and preserves the pre-jump offset. A
 * `hashchange` listener re-renders on native hash changes so `scrollKey` stays
 * current even when React Router's location does not update. (Note: this is a
 * composite surrogate and can still collide on repeated identical fragment
 * visits like `A -> A#x -> A -> A#x`; the robust fix is router-minted per-entry
 * keys — see PV-SCROLL-REVIEW-025 §16.6. This addresses the review comment
 * without that larger change.)
 *
 * ## Restore (review: "wait until the requested offset is actually scrollable")
 *
 * The hook lives in the persistent `/note/*` route element (`NotePage`), not
 * in `NoteHtml`: `NotePage`'s `isLoading` branch unmounts the `ScrollArea`
 * during a note fetch, so the scroller is not a stable DOM node. The restore
 * re-binds to the scroller once it re-mounts (`ready = !!note`) and applies the
 * offset via a rAF poll that waits until the offset is *actually* scrollable —
 * `scrollHeight - clientHeight >= offset` (the maximum valid `scrollTop`), not
 * `scrollHeight >= offset`, so an early exit no longer clamps to a smaller max
 * and the loop re-applies as content grows.
 *
 * ## Listener binding (PV-SCROLL-REVIEW-025 diagnosis)
 *
 * The scroll listener is attached in the **capture phase** to the `document`,
 * NOT to the scroller or to `containerRef`. Scroll events do not bubble, but
 * they traverse the capture phase, so a document-level capture listener
 * catches every `.note-scroll`'s scrolls without needing to find the scroller
 * first. This avoids two traps that sank earlier attempts: (1) binding to a
 * scroller not yet sized by the SplitPane, so the listener never attached and
 * cross-note Back regressed to 0; and (2) `containerRef.current` pointing at a
 * `div.h-full` that is not an ancestor of the visible scroller (duplicate /
 * transitional trees under StrictMode + lazy/eager hydration), so an ancestor
 * listener would miss it. The capture listener filters by the `note-scroll`
 * class to stay scoped to article scrollers.
 *
 * `pickScrollAction` and `scrollKeyOf` are pure and unit-tested (the web test
 * runner is `environment: "node"` — no DOM — see `vitest.config.ts`); the
 * DOM-dependent hook is exercised manually on the real vault.
 */
import { useEffect, useRef, useState, type RefObject } from "react";
import { useLocation } from "react-router-dom";

/**
 * Module-level scroll-offset store, keyed by `scrollKey`. Survives `NotePage`
 * unmount (note -> `/search` -> Back) so back/forward can restore. Client-only:
 * only ever mutated inside effects.
 */
const savedOffsets = new Map<string, number>();

/** What the hook should do with the scroll container for a given location. */
export type ScrollAction = "hash" | "restore" | "top" | "none";

/** Minimal location shape the policy reads (decoupled from react-router for tests). */
export interface ScrollLocation {
  key: string;
  pathname: string;
  hash: string;
}

/**
 * Stable scroll identity for a location: React Router's `location.key` plus the
 * URL hash. The key distinguishes history entries (different notes); the hash
 * distinguishes same-note fragment states (heading permalinks) that share a key
 * because they navigate via `window.location.hash` instead of React Router.
 */
export function scrollKeyOf(location: ScrollLocation): string {
  return `${location.key}|${location.hash}`;
}

/**
 * Decide how to treat the scroll container for `location`.
 *
 * Precedence: a non-`/note/*` route is left alone (`none`); a `#hash` location
 * defers to the existing scroll-to-heading behavior (`hash`); a note we have a
 * saved offset for restores it (`restore`); a fresh note scrolls to top
 * (`top`). `hash` beats `restore` so an explicit heading request always wins.
 */
export function pickScrollAction(
  location: ScrollLocation,
  saved: Map<string, number>
): ScrollAction {
  if (!location.pathname.startsWith("/note/")) return "none";
  if (location.hash) return "hash";
  if (saved.has(scrollKeyOf(location))) return "restore";
  return "top";
}

/**
 * The visible article scroller inside `root`, or `null` when there is none
 * (e.g. while a note is loading and the `ScrollArea` is unmounted).
 *
 * `NotePage` marks its article `ScrollArea`s with the `note-scroll` class; the
 * desktop and mobile branches both render but one is `display:none` (via
 * `md:` classes), so it has `clientHeight 0`. Picking the first with
 * `clientHeight > 0` selects the branch the user actually sees.
 */
export function findVisibleScroller(
  root: HTMLElement | null
): HTMLElement | null {
  if (!root) return null;
  const candidates = Array.from(
    root.querySelectorAll<HTMLElement>(".note-scroll")
  );
  for (const el of candidates) {
    if (el.clientHeight > 0) return el;
  }
  return null;
}

/**
 * Save the note scroll offset on leaving a `/note/*` location and restore it
 * on returning via back/forward.
 *
 * @param containerRef ref on a wrapper div in `NotePage` that contains the
 *   desktop + mobile layouts (and thus the article `ScrollArea`s).
 * @param ready `!!note` — true when a note is loaded and the scroller exists.
 *   The hook re-binds whenever this flips (e.g. after a loading period).
 */
export function useScrollRestoration(
  containerRef: RefObject<HTMLElement | null>,
  ready: boolean
): void {
  const location = useLocation();
  const lastOffset = useRef<number>(0);
  const lastKey = useRef<string | null>(null);
  // Re-render on native hash changes so `scrollKey` (which reads
  // window.location.hash) stays current even when React Router's location does
  // not update — heading permalinks set window.location.hash directly.
  const [, setHashTick] = useState(0);

  // Declare manual restoration once: the browser's "auto" is a no-op here
  // (the window never scrolls), so we fully own it.
  useEffect(() => {
    if (typeof history !== "undefined" && "scrollRestoration" in history) {
      history.scrollRestoration = "manual";
    }
  }, []);

  // Re-render when the native hash changes (heading permalinks bypass React
  // Router). Without this, `scrollKey` would stay stale across a hash-only
  // navigation and the save effect below would not fire.
  useEffect(() => {
    const onHash = () => setHashTick(t => t + 1);
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  // `window.location.hash` is the app's source of truth for the fragment
  // (heading permalinks set it directly, bypassing React Router). Reading it
  // here — after the hashchange listener has had a chance to re-render — keeps
  // `scrollKey` aligned with the live URL even when React Router's location has
  // not updated.
  const currentHash = typeof window !== "undefined" ? window.location.hash : "";
  const scrollKey = scrollKeyOf({ ...location, hash: currentHash });

  // Save the offset on `NotePage` unmount (the safety net for note -> /search
  // -> Back, where no scroll event fires after the leaving position). The
  // continuous capture below writes the store on every scroll keyed by the
  // *live* scroll identity, which is what preserves the pre-jump offset on a
  // heading permalink; this effect only handles the no-trailing-event case.
  useEffect(() => {
    lastKey.current = scrollKey;
    return () => {
      if (lastKey.current != null) {
        savedOffsets.set(lastKey.current, lastOffset.current);
      }
    };
  }, [scrollKey]);

  // Capture the live scroll offset via a capture-phase listener on the
  // document, and persist it to the store continuously keyed by the *live*
  // scroll identity (location.key + window.location.hash read at event time).
  // Continuous save is what preserves the pre-jump offset on a heading
  // permalink: the anchor sets `window.location.hash` *before* it scrolls, so
  // the jump's scroll events are attributed to the hashed identity and the
  // unhashed identity keeps the reader's pre-click position. (A save-on-
  // key-change effect alone races the smooth scroll: by the time the hashchange
  // re-render fires the save, the scroll has already updated `lastOffset`.)
  // Document-level (not `containerRef`) because in this app the ref can point
  // at a `div.h-full` that is not an ancestor of the visible scroller (duplicate
  // / transitional trees under StrictMode + lazy/eager hydration); filtering by
  // the `note-scroll` class keeps the listener scoped to article scrollers.
  // Re-binds when the note (key) changes or the scroller (re)mounts (`ready`).
  useEffect(() => {
    if (!ready) return;
    const onScroll = (e: Event) => {
      const el = e.target as HTMLElement | null;
      if (el && el.classList.contains("note-scroll")) {
        const y = el.scrollTop;
        lastOffset.current = y;
        const liveHash =
          typeof window !== "undefined" ? window.location.hash : "";
        savedOffsets.set(`${location.key}|${liveHash}`, y);
      }
    };
    document.addEventListener("scroll", onScroll, {
      capture: true,
      passive: true,
    });
    return () =>
      document.removeEventListener("scroll", onScroll, {
        capture: true,
      } as AddEventListenerOptions);
  }, [ready, location.key]); // eslint-disable-line react-hooks/exhaustive-deps

  // Bind to the (re)mounted scroller and restore. Re-runs when the location
  // changes or when `ready` flips (note loaded / loading), so it re-binds after
  // the scroller reappears following a fetch. The scroller is found globally
  // by the `note-scroll` class + `clientHeight > 0` (the visible branch); we do
  // not scope the query to `containerRef` because in this app the ref can
  // point at a `div.h-full` that is not an ancestor of the visible scroller
  // (see the capture-listener comment above).
  useEffect(() => {
    if (!ready) return;
    const scroller = findVisibleScroller(document.documentElement);
    if (!scroller) return;

    const action = pickScrollAction(
      { ...location, hash: currentHash },
      savedOffsets
    );
    let raf = 0;
    if (action === "restore" || action === "top") {
      const offset =
        action === "restore" ? (savedOffsets.get(scrollKey) ?? 0) : 0;
      let tries = 0;
      const restore = () => {
        // Wait until the offset is actually scrollable: the maximum valid
        // position is `scrollHeight - clientHeight`, so testing `scrollHeight`
        // alone can exit early while the offset is still unreachable (the set
        // would clamp to a smaller max and later content growth — e.g. async
        // embeds — would never reapply it). Bail after ~1s (60 rAFs) so a
        // short note still settles.
        const maxScroll = scroller.scrollHeight - scroller.clientHeight;
        if (maxScroll >= offset || tries++ > 60) {
          scroller.scrollTop = offset;
          return;
        }
        raf = requestAnimationFrame(restore);
      };
      raf = requestAnimationFrame(restore);
    }
    // `hash` and `none` do not restore, leaving NoteHtml's #heading effect to
    // handle hashed URLs.

    return () => cancelAnimationFrame(raf);
  }, [scrollKey, ready]); // eslint-disable-line react-hooks/exhaustive-deps
  // location/currentHash feed scrollKey; containerRef is a stable ref object.
}
