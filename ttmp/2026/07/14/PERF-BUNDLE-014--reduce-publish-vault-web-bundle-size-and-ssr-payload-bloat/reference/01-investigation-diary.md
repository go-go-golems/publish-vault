---
Title: Investigation diary
Ticket: PERF-BUNDLE-014
Status: active
Topics:
    - bundle
    - frontend
    - obsidian-vault
    - performance
    - retro-obsidian-publish
    - ssr
    - vite
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/api/api.go
      Note: Go API route registration lines 60-66 — backend endpoints the sidecar prefetches
    - Path: repo://web/src/App.tsx
      Note: Injected NotePageComponent and shared Suspense boundary; commit 88af5739d58c513b97d396985b53f39e067ed3b1
    - Path: repo://web/src/components/pages/NotePage/NotePage.tsx
      Note: Only consumer of NoteRenderer; lazy-loading target for Phase 1
    - Path: repo://web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: Sidebar calls useGetTreeQuery line 68 — reason tree stays in SSR preload
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T17:05:15.025954318-04:00
WhatFor: ""
WhenToUse: ""
---











# Investigation Diary

## Goal

Record the investigation that produced the bundle-size analysis for ticket PERF-BUNDLE-014, so
the intern (or future me) can reproduce the diagnosis and continue from the same evidence.

## Step 1: Initial diagnosis of live site bundle weight

The user asked why the JS bundles and root page of `https://parc.yolo.scapegoat.dev/` are so
big. I navigated the live site, captured the network requests, and downloaded each asset to
measure raw and gzipped sizes. The site makes only three requests, so the bloat is concentrated:
a single 2.03 MB JS chunk (599 KB gzipped), a 126 KB CSS (20 KB gzipped — fine), and a 1.31 MB
root HTML document.

### Prompt Context

**User prompt (verbatim):** "Why are the JS bundles / the root page so big ? https://parc.yolo.scapegoat.dev/ (which is the current project)"

**Assistant interpretation:** Diagnose the bundle and HTML weight of the production
publish-vault site by measuring assets and reading the build/SSR configuration.

**Inferred user intent:** Understand the root cause(s) of the large payload before deciding what
to fix, and locate the exact code responsible.

### What I did
- `curl`-downloaded `main.js`, `main.css`, and the root HTML from the live site and measured
  bytes with `wc -c` and `gzip -c | wc -c`.
- Read `web/vite.config.ts` and confirmed there is no code splitting: a single `main` entry, no
  `manualChunks`, no `React.lazy` anywhere in `src/`.
- Read `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx` and found the two heavy static
  imports: `import hljs from "highlight.js"` (line 9) and `import mermaid from "mermaid"` (line 10).
- Read `web/server.mjs` and found the SSR sidecar unconditionally prefetches `/api/notes` (934
  notes) and `/api/tree` on every route, then inlines them as `window.__PRELOADED_STATE__`.
- Parsed the live inlined state with Python and confirmed the breakdown: `listNotes` ~550 KB,
  `getTree` ~284 KB, totaling ~834 KB of the 1.31 MB HTML.
- Confirmed `web/src/components/ui/chart.tsx` imports recharts but is itself imported by nothing,
  so it is tree-shaken out (negative result — do not chase recharts).

### Why
To give the user a precise, evidence-backed root-cause list rather than generic advice, and to
set up the follow-up design ticket.

### What worked
- Downloading the live assets and parsing the inlined JSON gave exact byte counts and a clean
  attribution of every kilobyte.
- The Vite `import.meta.glob` pattern for lazy-loading highlight.js languages is directly
  documented by Vite maintainers for this exact use case (Vite issue #1903), which made the
  highlight.js recommendation concrete and low-risk.

### What didn't work
- `node_modules` was not installed in `web/`, so I could not run `rollup-plugin-visualizer` for a
  per-module treemap. The grep-frequency heuristic on the built chunk (`d3` ×53, `mermaid` ×17)
  was used instead to confirm mermaid/d3's presence. A full visualizer pass is a good follow-up
  during Phase 1 implementation.

### What I learned
- The two problems are fully independent: the JS chunk size comes from static imports of heavy
  libs with no code splitting; the HTML size comes from the SSR sidecar inlining the full vault
  on every route. They can be fixed in separate phases.
- `highlight.js/lib/core` + per-language registration is the supported lightweight entry point;
  `import.meta.glob` over a curated `vendor/` directory is the clean way to load languages on
  demand in Vite without coupling to the package's internal paths.
- The SSR sidecar already has a `chooseHomeSlug` mirrored between `server.mjs` and `App.tsx`;
  this is the seam where route-conditional prefetch fits naturally.

### What was tricky to build
- Nothing runtime-tricky yet (this is the analysis step). The tricky part is reserved for Phase 1
  (the `React.lazy` + SSR interaction): `renderToString` does not fetch dynamic chunks, so a naive
  `React.lazy` wrapper would render a fallback during SSR and hydrate to the real component,
  causing a hydration mismatch. The mitigation is to keep the SSR entry importing `NotePage`
  eagerly and apply `React.lazy` only in `entry-client.tsx`. This must be validated with
  `pnpm smoke:ssr`.

### What warrants a second pair of eyes
- The route-conditional SSR preload change (Phase 4): verify no note-route component calls
  `useListNotesQuery()` synchronously at render time without a loading state, or it will now
  trigger a client fetch instead of reading preloaded state.
- The `manualChunks` function: avoid forcing mermaid/highlight.js into vendor chunks, since they
  are already isolated by dynamic `import()`.

### What should be done in the future
- Run `rollup-plugin-visualizer` during Phase 1 implementation to get an exact per-module size
  breakdown and confirm the predicted chunk sizes.
- Consider server-side Mermaid rendering (headless DOM) as a follow-up project.
- Consider lazy-loading the sidebar to drop the tree from note-page preload.

### Code review instructions
- Start with the live measurements in §3.8 of the design doc, then §3.2–3.6 for the evidence chain.
- To validate the diagnosis yourself:
  ```bash
  curl -s -o /tmp/p.js -w "%{size_download}\n" https://parc.yolo.scapegoat.dev/assets/main-CZnSi4zO.js
  gzip -c /tmp/p.js | wc -c
  curl -s https://parc.yolo.scapegoat.dev/ | python3 -c "import sys,re,json;m=re.search(r'__PRELOADED_STATE__=(\{.*?\});',sys.stdin.read());s=json.loads(m.group(1));print({k:(len(v.get('data',[])) if isinstance(v.get('data'),list) else 'dict') for k,v in s['vaultApi']['queries'].items()})"
  ```

### Technical details
- Live baseline (2026-07-14): `main.js` 2,030,018 B raw / 598,884 B gz; root HTML 1,305,031 B;
  inlined state ~834 KB (listNotes 934 items ~550 KB + getTree ~284 KB).

## Step 2: Split the note route from the initial client bundle

The first implementation phase moved the note route behind a client-side dynamic import while
keeping the SSR path eager. This preserves server-rendered note content and hydration structure,
but prevents the initial home/search bundle from statically importing the note page and its
renderer dependency graph. The implementation required making the note-page component an explicit
injection point in `AppRoutes`: the server passes the eager component, while the browser passes a
`React.lazy` component.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue implementing PERF-BUNDLE-014 according to the ticket design,
keeping a detailed diary and committing at appropriate checkpoints.

**Inferred user intent:** Turn the analysis into production-quality code changes, validate each
phase, and leave a reviewable history rather than making one unstructured batch edit.

**Commit (code):** 88af5739d58c513b97d396985b53f39e067ed3b1 — "perf: split note page from client entry"

### What I did
- Installed the locked frontend dependencies with `cd web && pnpm install --frozen-lockfile`.
- Added `AppRoutesProps` and injected `NotePageComponent` into `web/src/App.tsx`.
- Added a shared Suspense fallback so the route boundary exists in both server and client trees.
- Kept `web/src/entry-server.tsx` importing `NotePage` eagerly for SSR.
- Added a `React.lazy(() => import(...NotePage...))` loader in `web/src/entry-client.tsx`.
- Ran Prettier and TypeScript checking.
- Ran `pnpm exec vitest run src/entry-server.test.tsx` — all 11 tests passed.
- Ran `pnpm build:all`; the client build emitted a separate `NotePage-*.js` chunk.

### Why
The initial bundle contained the entire note route despite home/search not needing it. Passing the
component into the shared route tree avoids duplicating route logic and lets SSR and hydration use
different loading policies without changing route behavior.

### What worked
- The initial client chunk changed from the original ~599 KB gzipped to `main-B430904A.js` at
  126.52 KB gzipped.
- The new `NotePage-rj2S5xGt.js` route chunk is 474.63 KB gzipped and is loaded only when the note
  component is needed.
- `pnpm check` passed and the 11 SSR unit tests passed.
- `pnpm build:all` completed successfully, including the 78.74 KB SSR bundle.

### What didn't work
- `pnpm smoke:ssr` reached the SSR HTTP assertion successfully (`raw SSR HTML ok (432424 bytes, X-Powered-By=Express)`),
  then failed because Playwright's browser binary is absent:
  ```text
  browserType.launch: Executable doesn't exist at /home/manuel/.cache/ms-playwright/chromium_headless_shell-1223/chrome-headless-shell-linux64/chrome-headless-shell
  Looks like Playwright was just installed or updated. Please run: pnpm exec playwright install
  ```
- The local build emitted many Mermaid internal chunks and still warns about large chunks; that is
  expected because Phase 2 has not yet moved the static Mermaid import out of the route chunk.

### What I learned
- The route-level split is most safely implemented as dependency injection: `AppRoutes` does not
  statically import `NotePage`; the SSR entry supplies the eager component, and the client entry
  supplies the lazy component.
- A shared Suspense boundary is needed so server and browser trees have the same structural boundary.
- `NotePage` still calls `useListNotesQuery()` for wiki-link resolution/backlinks; the SSR preload
  optimization must therefore omit only the data on note routes and allow the client query to run.

### What was tricky to build
The main sharp edge was avoiding a hydration mismatch. A lazy component in the SSR entry would
render a fallback because `renderToString` cannot fetch a client chunk. The solution was to keep
`NotePage` eager in `entry-server.tsx`, inject it into `AppRoutes`, and use `React.lazy` only in
`entry-client.tsx`. The shared Suspense boundary keeps the tree shape consistent.

### What warrants a second pair of eyes
- Validate browser hydration after installing Chromium; specifically inspect the console for
  Suspense or hydration warnings on `/`, `/note/index`, and `/search`.
- Verify that `NotePage` is not accidentally pulled back into the initial chunk through a type or
  default import.

### What should be done in the future
- Install the Playwright browser and rerun `pnpm smoke:ssr` before considering the phase fully
  validated.
- Proceed to Phase 2: dynamic Mermaid import.

### Code review instructions
- Start at `web/src/entry-client.tsx` and `web/src/entry-server.tsx`, then follow the injected
  `NotePageComponent` through `web/src/App.tsx`.
- Validate with:
  ```bash
  cd web
  pnpm check
  pnpm exec vitest run src/entry-server.test.tsx
  pnpm build:all
  pnpm smoke:ssr
  ```

### Technical details
- Build output after Phase 1:
  - `main-B430904A.js`: 395,314 B raw / 126.52 KB gzipped
  - `NotePage-rj2S5xGt.js`: 1,633,432 B raw / 474.63 KB gzipped
  - `dist/index.html`: 367.48 KB raw / 105.47 KB gzipped (Vite shell, not SSR response)
