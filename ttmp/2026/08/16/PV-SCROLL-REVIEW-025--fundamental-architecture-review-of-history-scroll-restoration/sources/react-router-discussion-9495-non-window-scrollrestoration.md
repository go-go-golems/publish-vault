I briefly 
[discussed](https://discord.com/channels/770287896669978684/1033046089034641488/1033046089034641488)
 this with [@brophdawg11](https://github.com/brophdawg11) on Discord and he suggested starting a 
discussion here. He raised some useful concerns:

> I'm sure the component could be altered to support a more advanced nested scroll use-case, but it 
introduces edge cases that would need to be thought through. Like what if when you come back to 
that location the scrollElement is no longer present for some reason? falling back on window isn't 
going to be right since the scroll position is relative to the no-longer-rendered element etc.

## The problem

If you use CSS grid to define a layout where only one pane scrolls rather than the entire page, 
`<ScrollRestoration>` does not work because it only uses `window.scrollY` and `window.scrollTo(_, 
_)`.

```
function useScrollRestoration({ 

  getKey, 

  storageKey, 

}: { 

  getKey?: GetScrollRestorationKeyFunction; 

  storageKey?: string; 

} = {}) { 

  let { router } = useDataRouterContext(DataRouterHook.UseScrollRestoration); 

  let { restoreScrollPosition, preventScrollReset } = useDataRouterState( 

    DataRouterStateHook.UseScrollRestoration 

  ); 

  let location = useLocation(); 

  let matches = useMatches(); 

  let navigation = useNavigation(); 

 

  // Trigger manual scroll restoration while we're active 

  React.useEffect(() => { 

    window.history.scrollRestoration = "manual"; 

    return () => { 

      window.history.scrollRestoration = "auto"; 

    }; 

  }, []); 

 

  // Save positions on unload 

  useBeforeUnload( 

    React.useCallback(() => { 

      if (navigation.state === "idle") { 

        let key = (getKey ? getKey(location, matches) : null) || location.key; 

        savedScrollPositions[key] = window.scrollY; 

      } 

      sessionStorage.setItem( 

        storageKey || SCROLL_RESTORATION_STORAGE_KEY, 

        JSON.stringify(savedScrollPositions) 

      ); 

      window.history.scrollRestoration = "auto"; 

    }, [storageKey, getKey, navigation.state, location, matches]) 

  ); 

 

  // Read in any saved scroll locations 

  React.useLayoutEffect(() => { 

    try { 

      let sessionPositions = sessionStorage.getItem( 

        storageKey || SCROLL_RESTORATION_STORAGE_KEY 

      ); 

      if (sessionPositions) { 

        savedScrollPositions = JSON.parse(sessionPositions); 

      } 

    } catch (e) { 

      // no-op, use default empty object 

    } 

  }, [storageKey]); 

 

  // Enable scroll restoration in the router 

  React.useLayoutEffect(() => { 

    let disableScrollRestoration = router?.enableScrollRestoration( 

      savedScrollPositions, 

      () => window.scrollY, 

      getKey 

    ); 

    return () => disableScrollRestoration && disableScrollRestoration(); 

  }, [router, getKey]); 

 

  // Restore scrolling when state.restoreScrollPosition changes 

  React.useLayoutEffect(() => { 

    // Explicit false means don't do anything (used for submissions) 

    if (restoreScrollPosition === false) { 

      return; 

    } 

 

    // been here before, scroll to it 

    if (typeof restoreScrollPosition === "number") { 

      window.scrollTo(0, restoreScrollPosition); 

      return; 

    } 

 

    // try to scroll to the hash 

    if (location.hash) { 

      let el = document.getElementById(location.hash.slice(1)); 

      if (el) { 

        el.scrollIntoView(); 

        return; 

      } 

    } 

 

    // Opt out of scroll reset if this link requested it 

    if (preventScrollReset === true) { 

      return; 

    } 

 

    // otherwise go to the top on new locations 

    window.scrollTo(0, 0); 

  }, [location, restoreScrollPosition, preventScrollReset]); 

}
```

## CodeSandbox demo

[![2022-10-24-scroll-restore-demo](https://user-images.githubusercontent.com/3612203/197569387-abc36
6ac-5865-447a-9663-92e76f8b7d38.gif)](https://user-images.githubusercontent.com/3612203/197569387-ab
c366ac-5865-447a-9663-92e76f8b7d38.gif)

## Very Bad patch fix

In my app at work, I'm working around this by using 
[patch-package](https://github.com/ds300/patch-package) to simply use 
`document.querySelector('#content-pane')` instead of `window`. This is a little scary, but it seems 
to work quite well. `#content-pane` is in layouts so we know it will more or less always be there.

```
diff --git a/node_modules/react-router-dom/dist/index.js 
b/node_modules/react-router-dom/dist/index.js
index aeec512..b617027 100644
--- a/node_modules/react-router-dom/dist/index.js
+++ b/node_modules/react-router-dom/dist/index.js
@@ -796,6 +796,19 @@ function useScrollRestoration(_temp3) {
   let matches = useMatches();
   let navigation = useNavigation(); // Trigger manual scroll restoration while we're active
 
+  // HACK: Scroll restoration doesn't work out of the box (issue #1155) because
+  // the container that scrolls is the content pane, one of the page-level grid
+  // cells. So instead of scrolling window, we pull that element directly by ID
+  // and use it instead. In the unlikely event it's not there, fall back to
+  // window (and likely do nothing).
+  const getScrollTarget = () => document.querySelector('#content-pane') || window;
+
+  // window has scrollY but normal elements have scrollTop
+  const getScrollY = () => {
+    const el = getScrollTarget();
+    return el ? el.scrollY || el.scrollTop : undefined;
+  }
+
   React.useEffect(() => {
     window.history.scrollRestoration = "manual";
     return () => {
@@ -806,7 +819,7 @@ function useScrollRestoration(_temp3) {
   useBeforeUnload(React.useCallback(() => {
     if (navigation.state === "idle") {
       let key = (getKey ? getKey(location, matches) : null) || location.key;
-      savedScrollPositions[key] = window.scrollY;
+      savedScrollPositions[key] = getScrollY();
     }
 
     sessionStorage.setItem(storageKey || SCROLL_RESTORATION_STORAGE_KEY, 
JSON.stringify(savedScrollPositions));
@@ -825,7 +838,7 @@ function useScrollRestoration(_temp3) {
   }, [storageKey]); // Enable scroll restoration in the router
 
   React.useLayoutEffect(() => {
-    let disableScrollRestoration = router == null ? void 0 : 
router.enableScrollRestoration(savedScrollPositions, () => window.scrollY, getKey);
+    let disableScrollRestoration = router == null ? void 0 : 
router.enableScrollRestoration(savedScrollPositions, () => getScrollY(), getKey);
     return () => disableScrollRestoration && disableScrollRestoration();
   }, [router, getKey]); // Restore scrolling when state.restoreScrollPosition changes
 
@@ -837,7 +850,7 @@ function useScrollRestoration(_temp3) {
 
 
     if (typeof restoreScrollPosition === "number") {
-      window.scrollTo(0, restoreScrollPosition);
+      getScrollTarget().scrollTo(0, restoreScrollPosition);
       return;
     } // try to scroll to the hash
 
@@ -856,8 +869,7 @@ function useScrollRestoration(_temp3) {
       return;
     } // otherwise go to the top on new locations
 
-
-    window.scrollTo(0, 0);
+    getScrollTarget().scrollTo(0, 0);
   }, [location, restoreScrollPosition, preventScrollReset]);
 }
```

## More general approach

Obviously hard coding an element doesn't make sense for the real library. Here are some ways I can 
picture it working:

```
<ScrollRestoration getScrollTarget={() => document.querySelector('#content-pane')} />
```

and in the library:

```
const getScrollTarget = () => props.getScrollTarget() || window;
```

Calling `document.querySelector` in React is usually not a thing we want to do, so maybe it should 
be a ref instead. I think this would require putting `ScrollRestoration` next to your scroll 
container so you can get the ref:

```
const ContentPane = () => {
  const ref = useRef(null)
  return (
    <>
      <ScrollRestoration scrollTargetRef={ref} /> 
      <div ref={ref}>...</div>
    </>
  )
}
```

In both cases I think falling back to `window` should be ok, because generally it will do nothing. 
Though maybe that's only true in my use case because we've made sure the whole page doesn't scroll. 
So maybe it's safer not to fall back, and instead just don't do anything if the ref is null or if 
the selector function comes back null.

I've built a hook to restore scroll of individual elements:

```
function SomeComp() {
  useElementScrollRestoration("content-pane");

  return (
    <div id="content-pane">
      {/* ... */}
    </div>
  )
}
```

And then the inside of that looks a whole lot like our code inside of `<ScrollRestoration>`. They 
ought to be able to share almost all of the same code.

I'd love to add this hook. If you or someone else wants to take this on I'd love to offer a few 
tips on how I think I'd implement it.

3 replies

Would be a better hook if it was called `useScrollRestorationRef` and then set the ref on the 
element:

```
function SomeComp() {
  const ref = useScrollRestorationRef(null);

  return (
    <div ref={ref}>
      ...
    </div>
  )
}
```

To add to the above: using this patch, you can write the following:

```
// root.tsx
export default (_: Route.ComponentProps) => (
  <html>
    <head>{/* ... */}</head>
    <body>
      {/* ... */}
      <ScrollRestoration
        getScrollContainer={() => document.querySelector("main.overflow-auto")}
      />
      <Scripts />
    </body>
  </html>
)
```

0 replies

**+1,** with a use case that hasn't been mentioned yet: native **WebViews** where the scroll root 
isn't `window`.

We ship a React Router app inside native iOS/Android WebViews. In several in-app browsers 
(Instagram iOS, some Android WebViews) the scroll root is `document.body` / 
`document.documentElement` / `document.scrollingElement`, not `window`. Since `ScrollRestoration` 
hardcodes `window.scrollY` / `window.scrollTo`, restoration is fully broken there:

- **POP (back/forward):** the saved value is `0` (the scroll lived on `body`, not `window`), so 
nothing restores.
- **PUSH:** the previous page's `body` offset is retained by the WebView and `window.scrollTo(0, 
0)` doesn't clear it, so the new page starts mid-scroll.

A configurable scroll target like [@lensbart's `getScrollContainer` 
patch](https://gist.github.com/lensbart/7310ad3066195a1f755289c1740d9bf3) looks like the right 
shape for this — in principle you'd point it at the real scroll root:

```
<ScrollRestoration getScrollContainer={() => document.scrollingElement} />
```

One caveat from our own experience: which element actually holds the scroll varies across these 
WebViews (`body` vs `documentElement` vs `scrollingElement`), so in practice we had to *detect* the 
active scroll root at runtime rather than assume `scrollingElement`. I haven't verified lensbart's 
patch against these specific WebViews, so I can't claim it works out of the box — but a single 
configurable target that can return the detected root would be enough to cover this case.

Flagging it as another class of problem worth SC interest: unlike the grid/flexbox cases above, 
this isn't an app design choice — it's the runtime environment. The same code that works in a 
normal browser silently breaks inside a WebView, with no supported fix short of reimplementing 
`ScrollRestoration` (which is what we currently do).

0 replies
