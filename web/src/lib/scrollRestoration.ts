/**
 * Scroll restoration for the note view.
 *
 * The browser's `history.scrollRestoration` only restores *window* scroll, and
 * this app scrolls a nested `ScrollArea` (the app shell is `h-screen
 * overflow-hidden`, so `window.scrollY` is always 0 — see PV-SCROLL-023). We
 * therefore own restoration: save the note's scroll offset keyed by
 * `location.key` and re-apply it on back/forward.
 *
 * The hook lives in the persistent `/note/*` route element (`NotePage`), not
 * in `NoteHtml`: `NotePage`'s `isLoading` branch unmounts the `ScrollArea`
 * during a note fetch, so the scroller is not a stable DOM node. A passive
 * scroll listener captures the offset continuously into a `useRef` that
 * survives the unmount; the save reads that ref; the restore re-binds to the
 * scroller once it re-mounts (`ready = !!note`) and applies the offset via a
 * rAF poll that waits for the content height.
 *
 * `pickScrollAction` is pure and unit-tested (the web test runner is
 * `environment: "node"` — no DOM — see `vitest.config.ts`); the DOM-dependent
 * hook is exercised manually on the real vault.
 */
import { useEffect, useRef, type RefObject } from "react";
import { useLocation } from "react-router-dom";

/** What the hook should do with the scroll container for a given location. */
export type ScrollAction = "hash" | "restore" | "top" | "none";

/** Minimal location shape the policy reads (decoupled from react-router for tests). */
export interface ScrollLocation {
  key: string;
  pathname: string;
  hash: string;
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
  savedOffsets: Map<string, number>
): ScrollAction {
  if (!location.pathname.startsWith("/note/")) return "none";
  if (location.hash) return "hash";
  if (savedOffsets.has(location.key)) return "restore";
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
export function findVisibleScroller(root: HTMLElement | null): HTMLElement | null {
  if (!root) return null;
  const candidates = Array.from(root.querySelectorAll<HTMLElement>(".note-scroll"));
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
  const saved = useRef<Map<string, number>>(new Map());
  const lastOffset = useRef<number>(0);
  const lastKey = useRef<string | null>(null);

  // Declare manual restoration once: the browser's "auto" is a no-op here
  // (the window never scrolls), so we fully own it.
  useEffect(() => {
    if (typeof history !== "undefined" && "scrollRestoration" in history) {
      history.scrollRestoration = "manual";
    }
  }, []);

  // Save the offset of the location we are leaving. Reads the scroll
  // listener's last value, which survives the scroller unmounting for a
  // loading state. Runs before the bind/restore effect in the same commit
  // (effects fire in definition order), so the leaving offset is persisted
  // before the arriving one is applied.
  useEffect(() => {
    if (lastKey.current != null) {
      saved.current.set(lastKey.current, lastOffset.current);
    }
    lastKey.current = location.key;
    lastOffset.current = 0;
  }, [location.key]);

  // Bind to the (re)mounted scroller and restore. Re-runs when the location
  // changes or when `ready` flips (note loaded / loading), so it re-binds after
  // the scroller reappears following a fetch.
  useEffect(() => {
    if (!ready) return;
    const scroller = findVisibleScroller(containerRef.current);
    if (!scroller) return;

    const onScroll = () => {
      lastOffset.current = scroller.scrollTop;
    };
    scroller.addEventListener("scroll", onScroll, { passive: true });

    const action = pickScrollAction(location, saved.current);
    let raf = 0;
    if (action === "restore" || action === "top") {
      const offset = action === "restore" ? saved.current.get(location.key) ?? 0 : 0;
      let tries = 0;
      const restore = () => {
        // Wait for the content to be tall enough that the offset is
        // meaningful; bail after ~1s (60 rAFs) so a short note still settles.
        if (scroller.scrollHeight >= offset || tries++ > 60) {
          scroller.scrollTop = offset;
          return;
        }
        raf = requestAnimationFrame(restore);
      };
      raf = requestAnimationFrame(restore);
    }
    // `hash` and `none` attach the listener only (no restore), leaving
    // NoteHtml's #heading effect to handle hashed URLs.

    return () => {
      cancelAnimationFrame(raf);
      scroller.removeEventListener("scroll", onScroll);
    };
  }, [location.key, location.pathname, location.hash, ready]); // eslint-disable-line react-hooks/exhaustive-deps
  // containerRef is a stable ref object and intentionally omitted.
}
