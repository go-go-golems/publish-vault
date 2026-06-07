---
Title: React Router SSR hydration cleanup implementation guide
Ticket: RETRO-SEO-009
Status: active
Topics:
    - seo
    - ssr
    - html-layout
    - page-title
    - meta-tags
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: scripts/smoke-ssr-hydration.mjs
      Note: Live smoke validation documented as acceptance gate
    - Path: web/server.mjs
      Note: |-
        Express sidecar data prefetch and HTML/meta assembly; should keep prefetching but stop relying on divergent SSR markup.
        SSR sidecar data prefetch and head injection
    - Path: web/src/App.tsx
      Note: |-
        Current Wouter routes and global title effect; will become router-agnostic app shell/routes.
        Current Wouter route table and title effect; migration center
    - Path: web/src/components/pages/NotePage/NotePage.tsx
      Note: |-
        Uses Wouter navigation; should own note-specific document title after hydration.
        Current note route navigation and title target
    - Path: web/src/components/pages/SearchPage/SearchPage.tsx
      Note: Uses Wouter navigation; migrate to React Router navigation.
    - Path: web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: |-
        Uses Wouter navigation and browser window checks; must become SSR-safe.
        Current Wouter navigation and hydration-sensitive clock
    - Path: web/src/entry-client.tsx
      Note: |-
        Current createRoot-after-clear path; will switch to hydrateRoot with BrowserRouter.
        Current remount path; will switch to hydrateRoot
    - Path: web/src/entry-server.tsx
      Note: |-
        Current simplified SSR renderer; will render the real App through React Router server location.
        Current divergent SSR renderer; will render real app under StaticRouter
    - Path: web/src/store/store.ts
      Note: Store factory is already SSR-compatible and should stay per-request on the server.
    - Path: web/src/store/vaultApi.ts
      Note: RTK Query cache already supports preloading; hydration must preserve query keys.
    - Path: web/ssr.Dockerfile
      Note: Runtime packaging documented for externalized SSR dependencies
    - Path: web/vite.config.ts
      Note: Final externalized SSR dependency model documented with size measurements
ExternalSources: []
Summary: Second implementation guide for replacing Wouter with React Router, consolidating SSR and client rendering onto one component tree, and moving from SSR-preview-plus-remount to real hydrateRoot hydration.
LastUpdated: 2026-06-07T16:00:00Z
WhatFor: Use when implementing the React Router SSR hydration cleanup in RETRO-SEO-009.
WhenToUse: Before changing routing, SSR entry points, hydration, page title behavior, or SSR layout semantics.
---



# React Router SSR hydration cleanup implementation guide

## Executive summary

The current SSR implementation deliberately renders a simplified React tree on the server, injects it into the built `index.html`, then clears that server-rendered DOM in `entry-client.tsx` and mounts the full client SPA with `createRoot()`. That made the first SSR sidecar easy to ship, but it also creates long-term cleanup pressure: the server and client have duplicate route parsing, different page components, different layout behavior, and divergent title behavior.

This ticket now deliberately rips off that bandaid. We will replace Wouter with React Router, render the real application tree on the server through a server-side router, and hydrate that same tree in the browser with `hydrateRoot()`. The goal is not to make every interactive widget fully useful before JavaScript loads; the goal is that the DOM produced by SSR is the same DOM React expects during hydration.

The intended end state is:

```text
server.mjs prefetches data
  -> entry-server.renderApp(url, data)
      -> makeStore()
      -> seed RTK Query cache
      -> renderToString(
           <Provider store={store}>
             <StaticRouter location={url}>
               <App />
             </StaticRouter>
           </Provider>
         )
  -> server.mjs injects HTML + state + meta tags
  -> browser entry-client.tsx calls hydrateRoot(
       root,
       <Provider store={store}>
         <BrowserRouter>
           <App />
         </BrowserRouter>
       </Provider>
     )
```

The most important invariant: **the server and client must render the same component tree for the same URL and preloaded store state**.

## Problem statement

### Current pain

The current implementation has these intentional divergences:

- `entry-server.tsx` has custom `parseRoute()` logic.
- `server.mjs` has another custom `parseRoute()` implementation.
- `entry-server.tsx` renders hand-built `SSRNotePage`, `SSRHomePage`, and `SSRSearchPage` components.
- The browser renders `App -> VaultLayout -> NotePage/SearchPage`.
- `entry-client.tsx` clears SSR content before mounting the SPA.
- `App.tsx` overwrites server-generated note titles with site-level `config.pageTitle`.

That means SSR is useful for crawlers, but it is not hydration. It is a server-rendered preview followed by a client remount.

### Target cleanup

The target cleanup removes the split-brain model:

- One route table.
- One app shell.
- One note page implementation.
- One layout implementation.
- One data cache shape.
- One hydration path.

## Current technical constraints

### Wouter usage surface

Wouter is currently used in a small, migratable surface:

```text
web/src/App.tsx
web/src/components/pages/VaultLayout/VaultLayout.tsx
web/src/components/pages/NotePage/NotePage.tsx
web/src/components/pages/SearchPage/SearchPage.tsx
web/src/pages/NotFound.tsx
```

The migration is feasible because navigation is mostly imperative (`navigate('/note/...')`) and route declarations are centralized in `App.tsx`.

### Store/data surface

The Redux/RTK Query surface is already close to the desired shape:

- `makeStore(preloadedState?)` creates a new store for SSR and can hydrate a browser store.
- `entry-server.tsx` already seeds RTK Query cache via `vaultApi.util.upsertQueryData()`.
- `server.mjs` already serializes `preloadedState` into `window.__PRELOADED_STATE__` safely.

This means the routing/rendering mismatch is the main blocker, not data fetching.

### Browser-only behavior

Hydration requires the first client render to match the server render. Components must not branch on browser-only values during the initial render. Current danger points:

- `VaultLayout.tsx` uses `window.innerWidth` inside event handlers only; this is safe because event handlers do not execute during SSR.
- Any hook that reads `window` during render must be avoided or guarded.
- Time display (`new Date().toLocaleTimeString`) in `VaultLayout` is a hydration mismatch risk because server and client render different minutes/timezones.

## Proposed architecture

### Route architecture

Use React Router's declarative route tree. Keep `App` router-agnostic by letting entry points choose the router implementation.

```tsx
// App.tsx
export function AppRoutes() {
  return (
    <Routes>
      <Route element={<VaultLayoutShell />}>
        <Route index element={<HomeRedirect />} />
        <Route path="note/*" element={<NoteRoute />} />
        <Route path="search" element={<SearchPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
```

Entry points own routers:

```tsx
// entry-server.tsx
<StaticRouter location={url}>
  <AppRoutes />
</StaticRouter>

// entry-client.tsx
<BrowserRouter>
  <AppRoutes />
</BrowserRouter>
```

### Hydration architecture

Replace:

```tsx
root.textContent = "";
createRoot(root).render(<App />);
```

with:

```tsx
hydrateRoot(root, <AppWithProviders store={store} router="browser" />);
```

Do not clear the root. If hydration mismatches occur, treat them as bugs in render determinism.

### Layout architecture

`VaultLayout` should render the same structural DOM on server and client. Avoid time-dependent and viewport-dependent render differences on the first render.

Required changes:

- Replace Wouter `useLocation()` with React Router `useNavigate()`.
- Move the live clock behind a hydration-safe component or render a stable placeholder until mounted.
- Keep sidebar rendering deterministic from Redux state and API cache.

### Title architecture

Move title behavior out of the route shell and into page-level components:

- Home route: `document.title = pageTitle` after hydration.
- Search route: `Search — pageTitle`.
- Note route: `note.title — pageTitle`.

The SSR title still comes from `server.mjs`, because React does not manage `<head>` in this app. The client-side title effect should match that exact format after hydration.

### SSR metadata architecture

Keep `server.mjs` responsible for `<head>` injection:

- `<title>`
- `<meta name="description">`
- OpenGraph tags
- canonical link
- markdown alternate link
- JSON-LD WebPage and BreadcrumbList

Do not add React Helmet in this phase. Head management can be a later cleanup after the tree hydrates cleanly.

## Implementation plan

### Phase 0: Baseline and safety commit

Before touching code, commit the ticket docs already created so the working tree starts from a known point.

Validation:

```bash
git status --porcelain
docmgr doctor --ticket RETRO-SEO-009 --stale-after 30
```

Commit:

```bash
git add ttmp/vocabulary.yaml ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout
git commit -m "Docs: plan SEO SSR cleanup"
```

### Phase 1: Add React Router dependency

Add `react-router-dom` to the web package and update the lockfile.

Validation:

```bash
pnpm --dir web install
pnpm --dir web check
```

Commit:

```bash
git add web/package.json web/pnpm-lock.yaml
git commit -m "web: add React Router dependency"
```

### Phase 2: Migrate client routing from Wouter to React Router

Files:

- `web/src/App.tsx`
- `web/src/components/pages/VaultLayout/VaultLayout.tsx`
- `web/src/components/pages/NotePage/NotePage.tsx`
- `web/src/components/pages/SearchPage/SearchPage.tsx`
- `web/src/pages/NotFound.tsx`

Changes:

- Replace `Route`, `Switch`, `useLocation` imports from Wouter.
- Use `Routes`, `Route`, `useNavigate`, `useParams` from React Router.
- Keep current route URLs exactly the same:
  - `/`
  - `/note/*`
  - `/search`
- Keep current home-note selection behavior.

Validation:

```bash
pnpm --dir web check
pnpm --dir web build
```

Commit:

```bash
git add web/src/App.tsx web/src/components/pages/VaultLayout/VaultLayout.tsx web/src/components/pages/NotePage/NotePage.tsx web/src/components/pages/SearchPage/SearchPage.tsx web/src/pages/NotFound.tsx
git commit -m "web: migrate routing to React Router"
```

### Phase 3: Consolidate SSR onto the real app tree

Files:

- `web/src/entry-server.tsx`
- `web/src/entry-client.tsx`
- `web/src/App.tsx`

Changes:

- Export a router-free `AppRoutes` or `AppShell` from `App.tsx`.
- Server entry renders `<StaticRouter location={url}><AppRoutes /></StaticRouter>`.
- Client entry renders `<BrowserRouter><AppRoutes /></BrowserRouter>`.
- Remove `SSRNotePage`, `SSRHomePage`, `SSRSearchPage`, and `parseRoute()` from `entry-server.tsx` once server.mjs no longer depends on `parseRoute()` there.
- Keep `preloadCache()`.

Validation:

```bash
pnpm --dir web check
pnpm --dir web build:ssr
pnpm --dir web test -- entry-server.test.tsx
```

Commit:

```bash
git add web/src/App.tsx web/src/entry-server.tsx web/src/entry-client.tsx web/src/entry-server.test.tsx
git commit -m "web: render real app tree during SSR"
```

### Phase 4: Replace remount with hydrateRoot

Files:

- `web/src/entry-client.tsx`

Changes:

- Replace `createRoot` with `hydrateRoot`.
- Remove `root.textContent = ""`.
- Ensure preloaded state is read before hydration.

Validation:

```bash
pnpm --dir web check
pnpm --dir web build:all
```

Manual browser check:

- No React hydration mismatch warnings in console.
- Sidebar renders after hydration.
- Note content is not visually replaced by a blank state.

Commit:

```bash
git add web/src/entry-client.tsx
git commit -m "web: hydrate server-rendered app"
```

### Phase 5: Make render output hydration-safe

Files likely involved:

- `web/src/components/pages/VaultLayout/VaultLayout.tsx`
- `web/src/components/pages/NotePage/NotePage.tsx`
- any component that reads `window`, `document`, time, random values, or layout during render

Known issue:

- The live clock in `VaultLayout` should not render `new Date()` during SSR. It should render a stable placeholder until mounted, or be marked suppressHydrationWarning.

Preferred pattern:

```tsx
function HydrationSafeClock() {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  if (!mounted) return <span className="...">--:--</span>;
  return <span className="...">{new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>;
}
```

Validation:

```bash
pnpm --dir web check
pnpm --dir web build:all
```

Manual browser check for hydration warnings.

Commit:

```bash
git add web/src/components/pages/VaultLayout/VaultLayout.tsx ...
git commit -m "web: make SSR hydration output deterministic"
```

### Phase 6: Title and metadata cleanup

Files:

- `web/src/App.tsx`
- `web/src/components/pages/NotePage/NotePage.tsx`
- `web/src/components/pages/SearchPage/SearchPage.tsx`
- `web/server.mjs`

Changes:

- Keep `server.mjs` head injection.
- Client title effects should match SSR formatting:
  - note: `Note Title — Page Title`
  - search: `Search — Page Title`
  - home: `Page Title`
- Prefer `config.pageTitle || config.vaultName || "Retro Obsidian Publish"` as site label.

Validation:

```bash
curl -s http://localhost:8080/note/index | grep '<title>'
# Browser document.title should match after hydration.
```

Commit:

```bash
git add web/src/App.tsx web/src/components/pages/NotePage/NotePage.tsx web/src/components/pages/SearchPage/SearchPage.tsx web/server.mjs
git commit -m "web: align SSR and client page titles"
```

### Phase 7: Final validation and docs update

Run:

```bash
GOWORK=off go test ./...
pnpm --dir web check
pnpm --dir web build:all
make test
```

Update diary, changelog, and tasks.

Commit:

```bash
git add ttmp/2026/06/07/RETRO-SEO-009--fix-page-titles-ssr-sidebar-for-seo-and-proper-html-menu-content-layout
git commit -m "Docs: record React Router SSR hydration cleanup"
```

## Risk register

### Hydration mismatches

The most likely failure mode is a React hydration warning due to server and client rendering different markup. Common causes:

- `new Date()` during render.
- `Math.random()` during render.
- branching on `window.innerWidth` during render.
- route params decoded differently between server and client.
- RTK Query query keys not matching between preloaded cache and hook call.

### Data waterfalls

If server preloading misses a route query, the real app SSR tree may render a loading state instead of content. That still hydrates but hurts SEO. The prefetch set in `server.mjs` must continue to include:

- config
- notes list
- tree
- current note for `/note/*`

### Routing wildcard differences

Wouter currently exposes wildcard params as `params["*"]`. React Router exposes them through `useParams()["*"]`. This migration is small but easy to get wrong.

### Side effects during SSR

Any component-level effect is fine because `useEffect` does not run on the server. Any direct `document` or `window` access during render must be removed or guarded.

## Decision records

### Decision 1: Use React Router rather than custom Wouter SSR

- **Context:** Wouter has a small usage surface but no first-class server router in the current implementation.
- **Options:** Patch Wouter, write a custom static location hook, or use React Router.
- **Decision:** Use React Router.
- **Rationale:** React Router has well-known SSR primitives and reduces bespoke routing code.
- **Consequence:** Small migration now, less custom SSR surface later.
- **Status:** proposed.

### Decision 2: Keep `server.mjs` as head/metadata owner

- **Context:** The app currently injects head metadata outside React.
- **Options:** Keep server-side string injection or adopt React Helmet/head management.
- **Decision:** Keep string injection for this refactor.
- **Rationale:** Hydration cleanup is already broad; head management can be a later improvement.
- **Consequence:** React app hydrates body/root; `server.mjs` still owns `<head>`.
- **Status:** proposed.

### Decision 3: Do not make data fetching route loaders yet

- **Context:** React Router supports loaders in data-router APIs, but current app uses RTK Query.
- **Options:** Adopt React Router loaders or keep RTK Query cache preloading.
- **Decision:** Keep RTK Query preloading.
- **Rationale:** Store factory and preloaded cache already work; changing fetching architecture would expand scope.
- **Consequence:** `server.mjs` still knows which API calls to prefetch per route.
- **Status:** proposed.

## Intern checklist

Before touching code:

- Read `web/src/App.tsx`.
- Read `web/src/entry-server.tsx`.
- Read `web/src/entry-client.tsx`.
- Read `web/server.mjs`.
- Run `pnpm --dir web check` and record baseline.

During implementation:

- Keep each phase separately committed.
- Do not change URLs.
- Do not change API contracts unless required.
- Do not introduce React Router loaders yet.
- Treat every hydration warning as a bug.

After implementation:

- Verify raw SSR HTML includes the note body.
- Verify browser console has no hydration mismatch warnings.
- Verify clicking sidebar entries navigates correctly.
- Verify document title remains note-specific after hydration.
- Verify `/note/{slug}.md`, `/sitemap.xml`, `/llms.txt`, and `/AGENTS.md` still bypass SSR correctly.

## Follow-up result: clean SSR dependency model and bundle-size measurement

After the initial React Router hydration migration, live testing showed that `ssr.noExternal: true` was the safest immediate fix for duplicate React instances: it bundled the entire SSR dependency graph so React Router and `react-resizable-panels` could not import a second React singleton. That was correct as an emergency cutover, but it was not the clean runtime model.

The cleaner model is now in place:

```ts
resolve: {
  dedupe: ["react", "react-dom", "react-router", "react-router-dom"],
},
ssr: {
  external: [
    "react",
    "react-dom",
    "react-router",
    "react-router-dom",
    "react-resizable-panels",
  ],
},
```

This chooses the **Node SSR service** model rather than the **self-contained SSR bundle** model:

```text
Node SSR service model
  server.mjs
    imports dist/ssr/entry-server.js
      imports react / react-dom / react-router / layout libraries
        all resolved from web/node_modules
```

The invariant is not “small bundle at any cost.” The invariant is: **React and all hook-using React libraries must resolve through one coherent runtime dependency tree.** The sidecar now preserves that invariant by running from the `web` package and keeping production `node_modules` available in the SSR container.

### Runtime packaging result

`web/ssr.Dockerfile` now builds client + SSR output, then prunes dev dependencies:

```dockerfile
RUN pnpm build:all && pnpm prune --prod
```

This keeps production runtime libraries such as `react`, `react-dom`, `react-router`, `react-router-dom`, `react-resizable-panels`, and `express`, while removing build/test tools such as Vite, TypeScript, Vitest, Storybook, and Playwright.

The built image was validated with:

```bash
docker build -f web/ssr.Dockerfile -t retro-ssr-smoke:local .
docker run -d --rm \
  -e SSR_DEBUG_RESOLUTION=1 \
  -e SSR_PORT=8089 \
  -p 127.0.0.1:18091:8089 \
  retro-ssr-smoke:local
curl -fsS http://127.0.0.1:18091/health
```

Container startup diagnostics confirmed React-family packages resolve under `/app/web/node_modules/.pnpm/...` after pruning.

### Bundle-size measurement

Measured during the live smoke-test work:

| Model | `dist/ssr/entry-server.js` | SSR build time observed | Notes |
|---|---:|---:|---|
| Blunt correctness fix: `ssr.noExternal: true` | `4,979.92 kB` | `13.52s` | Bundled the full app dependency graph, including heavy Markdown/diagram libraries. |
| Clean Node SSR externalization | `72.39 kB` (`72,390` bytes on disk) | `409ms`–`619ms` | Imports production dependencies from `web/node_modules` at runtime. |

The current file-size check is:

```bash
wc -c web/dist/ssr/entry-server.js
# 72390 web/dist/ssr/entry-server.js

du -h web/dist/ssr/entry-server.js
# 72K web/dist/ssr/entry-server.js
```

That is approximately a **98.5% reduction** in the main SSR entry file while preserving live hydration correctness.

### Validation commands

Use these commands after changing SSR routing, Vite SSR externalization, Docker packaging, or React-family dependencies:

```bash
pnpm --dir web check
SSR_DEBUG_RESOLUTION=1 pnpm --dir web smoke:ssr

docker build -f web/ssr.Dockerfile -t retro-ssr-smoke:local .
docker run -d --rm \
  -e SSR_DEBUG_RESOLUTION=1 \
  -e SSR_PORT=8089 \
  -p 127.0.0.1:18091:8089 \
  retro-ssr-smoke:local
curl -fsS http://127.0.0.1:18091/health
```

The live smoke test is the important acceptance gate. The TypeScript build can pass even when the sidecar fails at runtime with duplicate React or SSR import errors. The smoke test exercises the path that matters:

```text
pnpm build:all
  -> node server.mjs
  -> go run ... --ssr-url <sidecar>
  -> GET / through Go proxy
  -> verify real SSR HTML, not SPA fallback
  -> hydrate in Chromium
  -> verify zero console warnings/errors
  -> click sidebar navigation
```

### Remaining tradeoffs

The externalized model makes the SSR entry small and conceptually clean, but it shifts responsibility to runtime packaging. If a future deployment copies only `server.mjs` and `dist/ssr/entry-server.js` without production `node_modules`, the sidecar will fail at import time. That is why `web/ssr.Dockerfile` now documents and preserves production dependencies, and why `SSR_DEBUG_RESOLUTION=1` exists.

A future optimization could convert the SSR Dockerfile to a multi-stage build that copies only `dist`, `server.mjs`, `package.json`, lockfile metadata, and production `node_modules` into a smaller runtime image. That would be an image-size cleanup, not a React correctness change.
