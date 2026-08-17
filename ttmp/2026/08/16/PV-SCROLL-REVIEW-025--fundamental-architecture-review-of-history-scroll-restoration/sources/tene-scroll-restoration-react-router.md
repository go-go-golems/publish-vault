When building Single Page Applications (SPAs) with **React Router**, one UX issue appears almost 
immediately:

> Navigation changes the URL, but the page **does not scroll to the top**.

This breaks user expectations and feels wrong-especially on content-heavy pages like blogs, docs, 
or dashboards.

In this article, we’ll build a **robust `ScrollToTop` component**, explain **why it’s needed**, 
and cover **every real-world edge case** you should be aware of.

## The Problem: Why Scroll Doesn't Reset in SPAs

In traditional multi-page websites:

- Every navigation triggers a **full page reload**
- Browsers automatically reset scroll position to `(0, 0)`

In SPAs:

- Navigation happens **client-side**
- The DOM does NOT reload
- Browser scroll position is **preserved by default**

So when you navigate from `/blog` → `/about`, you stay scrolled halfway down the page.

## The Solution: ScrollToTop Component

Here’s the simplest and **correct** implementation:

```
import { useLayoutEffect } from "react";
import { useLocation } from "react-router-dom";

export function ScrollToTop() {
  const { pathname } = useLocation();

  useLayoutEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);

  return null;
}
```

## How It Works (Line by Line)

### useLocation()

```
const { pathname } = useLocation();
```
- React Router updates `location` **on every navigation**
- `pathname` changes when the route changes
- It does **not** change for query params or hash changes unless explicitly used

### Why useLayoutEffect (NOT useEffect)?

```
useLayoutEffect(() => {
  window.scrollTo(0, 0);
}, [pathname]);
```

| Hook | Problem |
| --- | --- |
| `useEffect` | Scroll happens **after paint**, causing flicker |
| `useLayoutEffect` | Runs **before paint**, smooth UX |

**This prevents visible jump/glitch** during navigation.

📌 Use `useLayoutEffect` for layout-related side effects like scroll, focus, or DOM measurement.

### Why Depend on pathname?

```
[pathname];
```

This ensures scroll resets **only when route changes**, not on:

- Re-renders
- State updates
- Context updates

### Why return null?

This component:

- Renders no UI
- Exists **only for side effects**
- Should be mounted once at the app root

## How to Use It Correctly

Place it **inside your router**, near the root:

```
<BrowserRouter>
  <ScrollToTop />
  <App />
</BrowserRouter>
```
- Do NOT place it inside individual pages
- Do NOT wrap routes individually

## Every Important Edge Case Explained

### 1\. Query Params (?page=2) Don’t Trigger Scroll

```
/pathname stays the same
/search changes
```

❌ Scroll won't reset when only query changes.

✅ Fix (if desired):

```
const { pathname, search } = useLocation();

useLayoutEffect(() => {
  window.scrollTo(0, 0);
}, [pathname, search]);
```

### 2\. Hash Navigation (#section) Breaks Anchors

If a URL is:

```
/docs#getting-started
```

Your component **overrides browser anchor behavior**.

✅ Fix:

```
const { pathname, hash } = useLocation();

useLayoutEffect(() => {
  if (hash) return;
  window.scrollTo(0, 0);
}, [pathname, hash]);
```

### 3\. Back/Forward Navigation Scroll Loss

By default:

- Browsers remember scroll on back/forward
- This component **breaks that behavior**

✅ Best practice:

- Only use this for **forward navigation**
- Or disable for certain routes

Advanced handling requires `history.state`, which is outside basic SPA scope.

### 4\. Mobile Safari & iOS Quirk

```
window.scrollTo(0, 0);
```

Sometimes fails on iOS due to momentum scroll.

✅ Safer alternative:

```
document.documentElement.scrollTop = 0;
document.body.scrollTop = 0;
```

(Use only if you see issues)

### 5\. Scroll Containers (Not Window)

If your app scrolls inside a container:

```
<div id="scroll-root"></div>
```

You must target it directly:

```
document.getElementById("scroll-root")?.scrollTo(0, 0);
```

### 6\. React Strict Mode Double Execution (Dev Only)

In React:

- `useLayoutEffect` runs **twice** in development
- This does **not** affect production
- Safe to ignore

### 7\. Performance Concerns?

Negligible.

- Runs only on navigation
- One synchronous operation
- Zero re-renders

## When You Shouldn't Use ScrollToTop

- Infinite scrolling pages
- Chat applications
- Forms with autosave
- Map-based UIs

In these cases, preserving scroll is actually better UX.

## Production-Grade Version (Best Balance)

```
import { useLayoutEffect } from "react";
import { useLocation } from "react-router-dom";

export function ScrollToTop() {
  const { pathname, hash } = useLocation();

  useLayoutEffect(() => {
    if (hash) return;
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
  }, [pathname, hash]);

  return null;
}
```

## Final Thoughts

Scroll restoration is **not optional UX polish** it’s a core navigation expectation.

This small component:

- Fixes a fundamental SPA flaw
- Improves perceived performance
- Makes your app feel “native”

If you’re using React Router and **don’t** have this, your users notice-even if they don’t 
tell you.
