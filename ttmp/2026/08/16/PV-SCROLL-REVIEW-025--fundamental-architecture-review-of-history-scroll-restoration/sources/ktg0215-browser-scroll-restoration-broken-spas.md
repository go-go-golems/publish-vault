Chrome has had scroll restoration support since 2015. You can even control it: 
`history.scrollRestoration = 'manual'`. But if you've ever tried to reliably restore a user's 
position on a React or Next.js app, you know it doesn't work the way you'd expect.

Here's what breaks, why it breaks, and how a browser extension can sidestep the entire problem.

---

## What the Browser Actually Does

The default behavior is `history.scrollRestoration = 'auto'`. When you navigate back to a page, the 
browser tries to scroll to where you were.

This works fine for static pages. It falls apart for:

- **SPAs** where content is injected into the DOM after navigation
- **Infinite scroll** pages where the content at a given Y position changes depending on what was 
previously loaded
- **Lazy-loaded images** that push content down after the scroll restore fires

The fundamental problem: the browser fires scroll restoration when the page HTML is parsed, not 
when the page *content* is fully rendered. A React app that loads a skeleton → fetches data → 
renders actual content will restore scroll into a partially-rendered DOM.

## The history.scrollRestoration = 'manual' Trap

If you set `manual`, you own scroll restoration completely. Most Next.js apps do this. The typical 
approach:

```
// Save position before navigation
router.beforeEach((to, from) => {
  savedPositions[from.path] = window.scrollY;
});

// Restore after navigation
router.afterEach((to) => {
  const position = savedPositions[to.path];
  if (position !== undefined) {
    nextTick(() => window.scrollTo(0, position));
  }
});
```

The `nextTick` is the problem. It fires after the Vue/React render cycle, but before async data 
fetching completes. The page renders empty containers, scroll restores to Y=800, then data loads 
and pushes everything down. User ends up at Y=800 in a now-different page position.

The correct fix is to wait until the content that was at Y=800 actually exists. There's no clean 
hook for this — you'd need to observe the DOM until the expected content is present.

## How Extensions Can Solve What the Framework Can't

A Chrome extension operates outside the app's JavaScript context. It has access to:

- `window.scrollY` at any point
- `MutationObserver` to detect when DOM content stabilizes
- `chrome.storage.local` to persist positions across sessions without touching the app's state

The approach 
[ReadMark](https://chromewebstore.google.com/detail/readmark/inejhohffndeacbihghjcobndpoejdfn) uses:

**1\. Save position before navigation:**

```
// content script
let lastUrl = location.href;
let saveTimeout;

function savePosition() {
  clearTimeout(saveTimeout);
  saveTimeout = setTimeout(() => {
    const data = {
      url: location.href,
      scrollY: window.scrollY,
      timestamp: Date.now(),
    };
    chrome.storage.local.set({ [location.href]: data });
  }, 500); // debounce: don't save on every pixel
}

window.addEventListener('scroll', savePosition, { passive: true });
```

The 500ms debounce is important — you don't want to fire a storage write on every scroll event.

**2\. Detect SPA navigation:**

SPAs don't fire `load` events on navigation. The `popstate` event fires for back/forward, but not 
for `pushState` navigations (clicking a link within an SPA). You need to intercept 
`history.pushState`:

```
const originalPushState = history.pushState.bind(history);
history.pushState = function (...args) {
  originalPushState(...args);
  onUrlChange();
};

window.addEventListener('popstate', onUrlChange);

function onUrlChange() {
  if (location.href !== lastUrl) {
    lastUrl = location.href;
    scheduleRestore();
  }
}
```

Note: this technique is MV3-compatible as long as it's in the content script, not the service 
worker. You cannot inject into `window` from a service worker.

**3\. Wait for content stability before restoring:**

```
function scheduleRestore() {
  // Wait for initial DOM mutations to settle
  const observer = new MutationObserver(() => {
    clearTimeout(restoreTimeout);
    restoreTimeout = setTimeout(() => {
      observer.disconnect();
      restorePosition();
    }, 150); // 150ms quiescence = content is probably done loading
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  // Fallback if no mutations occur
  setTimeout(() => {
    observer.disconnect();
    restorePosition();
  }, 2000);
}

async function restorePosition() {
  const result = await chrome.storage.local.get(location.href);
  const saved = result[location.href];
  if (saved) {
    window.scrollTo({ top: saved.scrollY, behavior: 'instant' });
  }
}
```

The `MutationObserver` approach waits for 150ms of DOM quiescence — no new mutations for 150ms 
— before restoring. This handles most async data loading patterns without needing app-level hooks.

## Why Not Use sessionStorage?

Session storage is per-tab and cleared when the tab closes. If you read an article halfway through, 
close Chrome, and reopen it the next day, the position is gone.

`chrome.storage.local` persists across sessions. It's also shared across tabs for the same 
extension (useful if you have the same URL open in multiple tabs and want consistent position).

The downside: storage limits. `chrome.storage.local` has a 10MB limit by default (but 
`unlimitedStorage` permission removes this). For scroll positions (just a URL + Y coordinate + 
timestamp), 10MB fits tens of thousands of saved positions.

## The Multi-Domain Problem

One thing that doesn't generalize: URL key collisions. If you're saving scroll position keyed by 
`location.href`, query parameters and hash fragments become part of the key. This is usually 
correct — `example.com/article?page=2` should have a different position than 
`example.com/article?page=1`.

But hash-only SPAs where the actual content is the same regardless of hash will accumulate 
duplicate position entries. Worth handling explicitly if you're building this yourself:

```
// Normalize URL for storage key
function getKey(url) {
  const u = new URL(url);
  u.hash = ''; // ignore hash if your SPA uses hash routing
  return u.toString();
}
```

---

## What Actually Works

For apps you control, the cleanest solution is framework-level scroll restoration with a proper 
content-ready hook. But for third-party sites — documentation, long-form content, GitHub issues, 
news articles — there's no access to the app's internals.

That's the real use case for extension-based scroll restoration: it works on any site, including 
ones you don't own and can't modify.

[ReadMark](https://chromewebstore.google.com/detail/readmark/inejhohffndeacbihghjcobndpoejdfn) is 
the extension I built for this. Free for up to 5 saved positions, unlimited with Pro.

What approach are you using for scroll restoration on your SPAs? The MutationObserver quiescence 
pattern has some obvious failure cases (sites that continuously mutate the DOM for animation) — 
curious what others have run into.
