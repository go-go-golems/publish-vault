## Scroll restoration in SPAs

> TL;DR: Scroll restoration make UX great but it's hard man.

## Scroll and scroll

When you scroll down a page, let's say Google, you click on a search result, you browse in that 
page, and then you hit the back button, bump! you are at the right position like before you clicked 
the search result. Now hit the forward button, bump! you are at the last position you were before 
you hit the back button.

<video controls="" alt="Scroll restoration" 
src="https://www.davidtran.dev/007/scroll-01.mp4"></video>

X (Twitter) did a great job on this thing.

But at the end, seem like normal behavior for all websites, right?

And then when you access another pages, like a lot of pages!!! Then back and forth just always at 
the top of the page, it's annoying and driving me crazy.

That's is **scroll restoration**. It's help:

- **Preserve Context:** When navigating back and forth, users expect to see the same content 
position where they left off.
- **Improve Perceived Performance:** A smooth transition that restores scroll position can make an 
app feel faster and more responsive.
- **Bridge SPA Limitations:** In SPAs, routes change without a full page reload, which means the 
browser's default scroll restoration may not work as expected—it's up to the developer to manage 
it.

In short, Scroll restoration is the process of restoring the scroll position of a page when the 
user navigates back to it.

## Browser's default scroll restoration

Browsers have a built-in mechanism to track scroll positions for each page in the session history. 
Here's how it works:

- **History Entries:** Every time you navigate, the browser creates a history entry that includes 
the current scroll position (usually stored as vertical and horizontal offsets).
- **Native Restoration:** For traditional multi-page applications, the browser automatically 
restores the scroll position when you use the back/forward buttons.
- **The `scrollRestoration` API:** Modern browsers offer the `window.history.scrollRestoration` 
property, which allows you to toggle the default behavior:

|  | auto | manual |
| --- | --- | --- |
| Behavior | The browser restores the scroll position with tracked scroll position | The browser 
does not restore the scroll position |
| When to use | When you want the browser default behavior | When you want to implement your own 
scroll restoration logic |

In normal websites (static HTML), you don't need to care about this thing, it just works.

But in SPAs, it's another story.

## SPA and scroll restoration

In SPAs, the auto scroll restoration is mostly not working as expected. It's because:

- SPAs typically use the history.pushState API for navigation, so the browser doesn't know to 
restore the scroll position natively
- SPAs sometimes render content asynchronously, so the browser doesn't know the height of the page 
until after it's rendered
- SPAs can sometimes use nested scrollable containers to force specific layouts and features.

TanStack router did [a great 
job](https://tanstack.com/router/v1/docs/framework/react/guide/scroll-restoration#scroll-restoration
) on explaining this.

So, you (or the library you're using) need to implement your own scroll restoration logic.

If you're using `react-router-dom` (or `react-router` now, yeah, it's confusing as 
[hell](https://remix.run/blog/merging-remix-and-react-router)), it already support scroll 
restoration with `<ScrollRestoration />` component

The implementation of this component is quite interesting, you can check it out 
[here](https://github.com/remix-run/react-router/blob/a3e4b8ed875611637357647fcf862c2bc61f4e11/packa
ges/react-router/lib/dom/lib.tsx#L1160).

But it's implement is not easy to understand, because so much edge cases to cover.

So, here is a simple implementation of it:

```javascript
import { useEffect, useRef } from 'react';
import { useLocation } from 'react-router-dom';
 
function useScrollRestoration() {
  // Get the current location object from react-router-dom.
  // This object changes whenever the route in the application changes.
  const location = useLocation();
 
  // useRef is used to create a mutable object that persists across renders.
  // Here, it's used to store scroll positions for different paths.
  // \`scrollPositions.current\` will be an object where keys are pathnames
  // and values are objects containing x and y scroll positions.
  const scrollPositions = useRef({});
 
  // This effect runs on every route change.
  useEffect(() => {
    // -------------------- RESTORATION LOGIC (Runs on route change) --------------------
    // When the route changes (component renders for a new route), this effect runs.
    // We check if there's a saved scroll position for the *current* pathname.
    if (scrollPositions.current[location.pathname]) {
      // If a saved scroll position exists for this path, restore it.
      // \`window.scrollTo()\` is a browser API to scroll to a specific position.
      window.scrollTo(scrollPositions.current[location.pathname]);
    } else {
      // If no saved scroll position is found for this path (e.g., first visit to this route),
      // optionally scroll to the top of the page. This is common behavior for new routes.
      window.scrollTo(0, 0);
    }
 
    // -------------------- SAVING LOGIC (Cleanup function runs before next effect) 
--------------------
    // The return value from useEffect is a cleanup function.
    // This function runs BEFORE the next effect runs (i.e., before the component re-renders 
because of a route change).
    // This is CRUCIAL for saving the scroll position of the page *before* we navigate away from it.
    return () => {
      // In the cleanup function, we save the *current* scroll position.
      // We use \`window.scrollX\` and \`window.scrollY\` to get the current horizontal and 
vertical scroll positions.
      scrollPositions.current[location.pathname] = {
        x: window.scrollX,
        y: window.scrollY,
      };
    };
  }, [location]);
 
  return null;
}
 
// Then in your root component
import { useScrollRestoration } from './useScrollRestoration';
 
const App = () => {
  useScrollRestoration();
  // Other logic
}
```

The main idea is:

- Save the scroll position before navigating away from a route
- Restore the scroll position when navigating back to that route

But how about server-side rendering (SSR) app? This custom hook depends on the `location` object, 
and the `location` object is not available on the server.

Also, in SSR app, when you navigate on client-side, then hard reload, their is no way to know the 
scroll position to restore.

## Scroll restoration in NextJS

In case you don't know, NextJS do support scroll restoration, but it behind a experimental flag and 
they even don't document it!!

You have to search through the github discussions and the source code to find 
[it](https://github.com/vercel/next.js/blob/canary/packages/next/src/server/config-shared.ts#L1212).

Yeah I know it experimental, mean it not production ready yet, but hey, better than nothing right? 
I enable this "experimental" feature in all of my NextJS app, and it works like a charm.

Let's see how Next team implement it.

From the `config-shared.ts` file, you can follow the reference to the `router.ts` file that 
contains the main logic to handle scroll restoration. To easy navigate, you can search for 
`process.env.__NEXT_SCROLL_RESTORATION` in the codebase.

First, look at the `manualScrollRestoration` variable. We see `sessionStorage` is used to save the 
scroll position.

They use a IIFE to check if the `sessionStorage` is available and if the `scrollRestoration` 
property is supported by the browser. Quite clever.

```typescript
const manualScrollRestoration =
  process.env.__NEXT_SCROLL_RESTORATION &&
  typeof window !== 'undefined' &&
  'scrollRestoration' in window.history &&
  !!(function () {
    try {
      let v = '__next'
      // eslint-disable-next-line no-sequences
      return sessionStorage.setItem(v, v), sessionStorage.removeItem(v), true
    } catch (n) {}
  })()
```

And then, they set `history.scrollRestoration` mode:

```typescript
if (process.env.__NEXT_SCROLL_RESTORATION) {
  if (manualScrollRestoration) {
    window.history.scrollRestoration = 'manual'
  }
}
```

Now to the most interesting part, on the `onPopState` and `onPush` event, they perform the scroll 
restoration logic:

```typescript
let forcedScroll: { x: number; y: number } | undefined
const { url, as, options, key } = state
if (process.env.__NEXT_SCROLL_RESTORATION) {
  if (manualScrollRestoration) {
    if (this._key !== key) {
      // Snapshot current scroll position:
      try {
        sessionStorage.setItem(
          '__next_scroll_' + this._key,
          JSON.stringify({ x: self.pageXOffset, y: self.pageYOffset })
        )
      } catch {}
 
      // Restore old scroll position:
      try {
        const v = sessionStorage.getItem('__next_scroll_' + key)
        forcedScroll = JSON.parse(v!)
      } catch {
        forcedScroll = { x: 0, y: 0 }
      }
    }
  }
}
 
// push
push(url: Url, as?: Url, options: TransitionOptions = {}) {
  if (process.env.__NEXT_SCROLL_RESTORATION) {
    // TODO: remove in the future when we update history before route change
    // is complete, as the popstate event should handle this capture.
    if (manualScrollRestoration) {
      try {
        // Snapshot scroll position right before navigating to a new page:
        sessionStorage.setItem(
          '__next_scroll_' + this._key,
          JSON.stringify({ x: self.pageXOffset, y: self.pageYOffset })
        )
      } catch {}
    }
  }
  ;({ url, as } = prepareUrlAs(this, url, as))
  return this.change('pushState', url, as, options)
}
```

`key` just a random string generated by `Math.random().toString(36).slice(2, 10)`

And you can see they save the scroll position before navigating in `sessionStorage`.

NextJS check a lot of edge cases to decide whether to scroll or not, and if it should, where to 
scroll. This is following priority to decide:

- Scroll to scroll position in `forcedScroll`
- Scroll to hash in URL
- Scroll to top of the page

You can check the `subscription` function of `Router` class to deep dive more how they handle these 
edge cases.

After read all these logics, I understand why they keep this feature behind a experimental flag. 
For SSR app, it's not that simple to implement.

Happy coding and make scroll restoration great again 💪
