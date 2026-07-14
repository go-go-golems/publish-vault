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
    - Path: repo://Dockerfile
      Note: Step 7 builder image aligned to Go 1.26.5
    - Path: repo://go.mod
      Note: Step 7 Go 1.26.5 security bump; commit 2d0df8090d207f968424b0969ccb7cd930294066
    - Path: repo://internal/api/api.go
      Note: Go API route registration lines 60-66 — backend endpoints the sidecar prefetches
    - Path: repo://web/src/App.tsx
      Note: Injected NotePageComponent and shared Suspense boundary; commit 88af5739d58c513b97d396985b53f39e067ed3b1
    - Path: repo://web/src/components/organisms/NoteRenderer/NoteRenderer.tsx
      Note: Phase 2 dynamic Mermaid import; commit f84d634f2b0a99925237dd5bbd032e485d11e99f
    - Path: repo://web/src/components/pages/NotePage/NotePage.tsx
      Note: Only consumer of NoteRenderer; lazy-loading target for Phase 1
    - Path: repo://web/src/components/pages/VaultLayout/VaultLayout.tsx
      Note: Sidebar calls useGetTreeQuery line 68 — reason tree stays in SSR preload
    - Path: repo://web/src/entry-client.tsx
      Note: Hydration-safe initial route component and home slug consumption; commit 0b032b5610ac450a8f1f3be6d2ee87f7365c13bc
    - Path: repo://web/src/entry-server.test.tsx
      Note: Regression tests proving home/note SSR cache omits listNotes; commit 0b032b5610ac450a8f1f3be6d2ee87f7365c13bc
    - Path: repo://web/src/index.css
      Note: Step 6 true-white CSS token update; commit 4542d8514a96ff7a49b39d4aee3ddf4846bd9154
    - Path: repo://web/src/lib/highlightLanguages.ts
      Note: |-
        Phase 3 per-language import.meta.glob loader; commit 7d1a490633be241d8088ca41200fbc27a16b6895
        Step 8 canonical highlight alias fix; commit 35d910b59c36bfde9917d1eceaf269821e0e03bc
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

## Step 3: Lazy-load Mermaid only for Mermaid notes

The second phase removed the top-level Mermaid import from `NoteRenderer` and moved it inside the
existing effect that already detects `code.language-mermaid` blocks. The effect now imports
Mermaid only when at least one diagram is present and cancels pending work when the note changes.
This preserves the raw `<pre>` fallback for failed rendering and prevents an old asynchronous
render from replacing DOM belonging to a newer note.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the next implementation phase after route-level splitting,
with the same validation and diary/commit discipline.

**Inferred user intent:** Remove Mermaid and its d3-heavy dependency graph from the note route's
immediate payload unless the current note actually contains a diagram.

**Commit (code):** f84d634f2b0a99925237dd5bbd032e485d11e99f — "perf: lazy load mermaid rendering"

### What I did
- Removed the static `import mermaid from "mermaid"` from `NoteRenderer.tsx`.
- Added `await import("mermaid")` after the existing zero-block early return.
- Kept the module-level initialization guard and existing Mermaid theme configuration.
- Added an effect cancellation flag and checked `pre.isConnected` before replacing raw code.
- Used `Promise.all` so multiple diagrams in one note render concurrently.
- Ran Prettier, `pnpm check`, `pnpm exec vitest run src/entry-server.test.tsx`, and `pnpm build`.

### Why
The previous phase moved Mermaid into the route chunk, but every note still downloaded it before
knowing whether it contained a diagram. Dynamic importing at the point where blocks are detected
makes the existing runtime early return effective at the network level too.

### What worked
- Build emitted `mermaid.core-B0R8ZnMT.js` at 607,354 B raw / 145.16 KB gzipped.
- `NotePage` fell from 474.63 KB to 329.48 KB gzipped.
- The initial `main` chunk remained approximately 126.51 KB gzipped.
- TypeScript and all 11 SSR unit tests passed.

### What didn't work
- The Vite build still reports large Mermaid-internal chunks. This is expected: those chunks are
  Mermaid's diagram-specific dynamic graph and are only fetched by Mermaid as needed.
- Browser-level network verification is still blocked until Playwright Chromium is installed.

### What I learned
- Mermaid 11 already dynamically imports many diagram definitions internally; moving the package
  boundary out of `NoteRenderer` compounds that behavior and avoids the core download entirely on
  non-diagram notes.
- Async DOM effects need cancellation even when the imported library is cached, because a render
  promise can resolve after a route transition.

### What was tricky to build
The effect has two asynchronous boundaries: loading Mermaid and rendering each diagram. A single
`cancelled` flag covers both. The DOM connection check prevents replacing a `<pre>` that React has
already removed during a note transition; errors deliberately leave the raw code visible.

### What warrants a second pair of eyes
- Check that Mermaid initialization remains safe if two `NoteRenderer` instances mount at the same
  time (the module-level guard is retained from the original code).
- Verify a note with several diagrams does not generate duplicate IDs or leave stale SVGs after
  navigation.

### What should be done in the future
- Add a browser smoke fixture containing one Mermaid note and one plain-code note, then assert the
  Mermaid chunk is absent/present respectively.
- Proceed to Phase 3: per-language highlight.js loading.

### Code review instructions
- Review the Mermaid effect in `web/src/components/organisms/NoteRenderer/NoteRenderer.tsx`.
- Validate with:
  ```bash
  cd web
  pnpm check
  pnpm exec vitest run src/entry-server.test.tsx
  pnpm build
  ```

### Technical details
- Phase 2 build output:
  - `main`: 395,300 B raw / 126.51 KB gzipped
  - `NotePage`: 1,025,652 B raw / 329.48 KB gzipped
  - `mermaid.core`: 607,354 B raw / 145.16 KB gzipped

## Step 4: Load individual highlight.js languages on demand

The third implementation phase replaced the full highlight.js registry with the small core
engine and a Vite-generated lazy module map. Each curated language is represented by a tiny
adapter under `web/src/vendor/highlight-languages/`; `import.meta.glob` turns those adapters into
independent dynamic chunks. `NoteRenderer` now awaits highlighting and only adds copy buttons
after the language definitions have loaded, with cancellation on note changes.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Implement the per-language highlight.js lazy-loading design and
continue validating and committing the work in phases.

**Inferred user intent:** Avoid paying for highlight.js's all-language registry and make a note
with a few code languages download only those language definitions.

**Commit (code):** 7d1a490633be241d8088ca41200fbc27a16b6895 — "perf: lazy load highlight languages on demand"

### What I did
- Added `web/src/lib/highlightLanguages.ts` with `highlight.js/lib/core`.
- Added a Vite `import.meta.glob` map over curated language adapter files.
- Added aliases for common markdown class names (`js`, `ts`, `tsx`, `py`, `sh`, `yml`, `html`,
  and related variants).
- Added a bounded auto-detection set for unlabelled blocks instead of restoring the full registry.
- Added an in-flight promise map so concurrent renderers do not register one language twice.
- Added language adapters for bash, CSS, Go, JavaScript, JSON, Markdown, Python, SQL, TypeScript,
  XML, and YAML.
- Updated `NoteRenderer` to await `highlightCodeBlocks` and cancel stale copy-button work.
- Ran Prettier, TypeScript checking, SSR unit tests, and a production build.

### Why
The bare `highlight.js` import registered approximately 190 languages and was part of the large
note route chunk. The core-plus-glob design creates one independently cacheable chunk per language,
so the browser downloads only the definitions represented by the current note.

### What worked
- Vite emitted individual language chunks:
  - `json`: 0.32 KB gzipped
  - `go`: 0.70 KB gzipped
  - `yaml`: 0.83 KB gzipped
  - `xml`: 0.78 KB gzipped
  - `python`: 1.49 KB gzipped
  - `bash`: 1.57 KB gzipped
  - `javascript`: 2.62 KB gzipped
  - `typescript`: 3.08 KB gzipped
  - `css`: 4.30 KB gzipped
- The note route chunk fell to 26.64 KB gzipped before any language, Mermaid, or diagram chunks.
- The initial `main` chunk remained 126.51 KB gzipped.
- `pnpm check`, all 11 SSR unit tests, and the production build passed.

### What didn't work
- The first implementation used `[...requestedLanguages]` and failed TypeScript because this
  project has no ES2015 target/downlevel iteration enabled:
  ```text
  src/lib/highlightLanguages.ts(129,25): error TS2802: Type 'Set<string>' can only be iterated through when using the '--downlevelIteration' flag or with a '--target' of 'es2015' or higher.
  ```
  I changed it to `Array.from(requestedLanguages).map(loadLanguage)`, then reran checking and tests successfully.
- Browser-level network assertions remain pending on the Playwright browser installation.

### What I learned
- Vite's `import.meta.glob` keys must be matched using the exact relative adapter paths; explicit
  `languageFiles` mapping makes this visible and stable while the build still emits one chunk per
  adapter.
- `highlight.js/lib/core` is small enough to remain in the note route, while language definitions
  are independently cacheable.
- An in-flight promise cache matters because React Strict Mode and simultaneous renderer mounts can
  otherwise call `registerLanguage` more than once.

### What was tricky to build
The old code highlighted synchronously and added copy buttons in the same pass. Dynamic language
loading makes that pass asynchronous. The new helper discovers all needed languages, loads them in
parallel, highlights only registered/known languages, and the component adds copy buttons only if
its effect is still current. Unknown language classes remain readable plain code instead of causing
an import failure.

### What warrants a second pair of eyes
- Confirm the curated language list covers the actual production vault after deployment; add a
  small audit script or expand adapters when new `language-*` classes appear.
- Validate that unlabelled blocks' bounded auto-detection is acceptable compared with the old full
  registry.
- Verify no language adapter is accidentally included in the initial route chunk; build output
  currently shows each as a separate file.

### What should be done in the future
- Add a browser fixture with mixed TypeScript/JSON/Bash blocks and assert only those language
  chunks are fetched.
- Proceed to Phase 4: omit the full `listNotes` data from note-route SSR while preserving the
  client-side backlinks/wiki-link query.

### Code review instructions
- Start at `web/src/lib/highlightLanguages.ts`, then inspect the adapter directory and the
  `NoteRenderer` highlighting effect.
- Validate with:
  ```bash
  cd web
  pnpm check
  pnpm exec vitest run src/entry-server.test.tsx
  pnpm build
  find dist/assets -type f | grep -E '(bash|json|typescript|yaml)-'
  ```

### Technical details
- Phase 3 build output:
  - `main`: 395,300 B raw / 126.51 KB gzipped
  - `NotePage`: 80,050 B raw / 26.64 KB gzipped
  - `mermaid.core`: remains a separate 145.16 KB gzipped chunk

## Step 5: Trim all SSR cache payloads and complete hydration-safe delivery

The final phases removed the full note index from the serialized RTK Query cache on both note and
home routes. The sidecar still fetches it on `/` to choose the home note and create the noscript
list, but sends only the selected slug in `window.__HOME_SLUG__`; `HomeRedirect` uses that slug
during SSR and first hydration. The browser fetches the complete index only when a later feature
(backlinks/wiki-link verification) actually needs it.

This phase also uncovered and fixed hydration constraints introduced by route splitting. The
initial client implementation used a Suspense boundary around every route, which changed the
React hydration shape. The corrected design eagerly resolves the already-SSR-rendered note chunk
on `/` and `/note/*`, uses no Suspense boundary for those first renders, and retains the lazy
boundary for non-note initial routes. A baseline worktree at `08bab91` confirmed that the original
application passed the browser smoke test, isolating the issue to this implementation rather than
masking it as pre-existing.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Finish the remaining SSR-payload and production-build phases,
resolve any regressions discovered by end-to-end validation, and keep commits/diary evidence.

**Inferred user intent:** Ship a real performance improvement without regressing SSR, hydration,
search, or navigation.

**Commit (code):** 0b032b5610ac450a8f1f3be6d2ee87f7365c13bc — "perf: reduce SSR payload and production bundle"

### What I did
- Changed `web/server.mjs` to fetch `/api/notes` only on `/`, only for home selection/noscript
  content, and never insert it into `renderApp`'s preloaded RTK cache.
- Added `homeSlug` to `SSRData`, passed it to `AppRoutes`, serialized it as `window.__HOME_SLUG__`,
  and consumed/deleted it in the client entry before hydration.
- Made `HomeRedirect` skip `useListNotesQuery` when a server-selected slug exists.
- Added a unit test proving `renderApp("/", { homeSlug: "index", ... })` renders the home note
  without a `listNotes` cache entry; SSR test count is now 13.
- Reworked client route hydration so home/note initial renders use the resolved `NotePage` component
  without a Suspense boundary; other initial routes retain the lazy fallback.
- Made `NoteRenderer` start from raw note HTML and resolve wiki links after hydration, avoiding
  server/client `DOMParser` serialization differences; deferred heading-anchor DOM enhancement.
- Ran the pre-change `08bab91` worktree smoke test to establish that the hydration failure was new.
- Gated JSX-location, Manus runtime/debug collector, and storage proxy plugins to `vite serve`
  in `vite.config.ts`; production builds no longer contain `data-loc` markers.
- Installed Playwright Chromium locally because the original smoke failure was caused by a missing
  browser executable, then ran the full browser hydration smoke test successfully.

### Why
The prior note-route change cut note HTML, but the root route still shipped the entire 934-note
index to let the client select the same home note. A tiny explicit home-slug contract preserves
identical SSR/client rendering while removing that payload. Production-only plugin gating also
removes development instrumentation from delivered JS and SSR output.

### What worked
- `pnpm check` passed.
- `pnpm exec vitest run src/entry-server.test.tsx` passed: **13 tests**.
- `pnpm build:all` passed; final main chunk is **388.78 KB raw / 125.83 KB gzipped**, down from
  the live baseline's **2.03 MB raw / 598.88 KB gzipped**.
- `pnpm smoke:ssr` passed end to end: production client build, SSR build, Go backend, Node sidecar,
  mobile sidebar interaction, desktop navigation, and zero browser console warnings/errors.
- Fixture root SSR HTML fell from the prior 432,424 B payload to **37,211 B** after list-index
  omission and production instrumentation removal.
- The final production `index.html` shell is 0.64 KB raw / 0.39 KB gzipped; no `data-loc` markers
  appear in built JavaScript.

### What didn't work
- `pnpm smoke:ssr` initially failed because Playwright Chromium was not installed. Exact message:
  ```text
  browserType.launch: Executable doesn't exist at .../chromium_headless_shell-1223/...
  Please run: pnpm exec playwright install
  ```
  Running `pnpm exec playwright install chromium` resolved the environment failure.
- The first `import.meta.glob` implementation of language adapters broke `pnpm build:ssr` with:
  ```text
  assets/yaml-!~{001}~.js:16:0: ERROR: Expected ":" but found "}"
  ```
  The loader now uses explicit dynamic imports plus an SSR alias to a no-op module; Vite still
  emits individual language chunks and the SSR graph stays browser-free.
- The first route-wide Suspense design produced React hydration error #418. A baseline smoke test
  passed, so it was not accepted as pre-existing. The final route-specific boundary fixed it.

### What I learned
- SSR payload decisions must account for the hydration render contract, not merely what the
  server fetched. Passing a small deterministic `homeSlug` is sufficient; serializing all note
  metadata is not.
- Browser-only HTML normalization and DOM enhancement must happen after hydration when the server
  rendered raw HTML.
- Vite client dynamic imports need an SSR-safe module boundary when their imported graph is not
  meaningful on the server.

### What was tricky to build
The central sharp edge was retaining SSR content *and* route splitting. A universal lazy/Suspense
boundary changed React's component/useId path and caused hydration failure. The solution is route
aware: on initial home/note URLs, wait for the note chunk and hydrate the same eager component
shape that SSR used; on routes without SSR note content, keep the lazy component and Suspense.

### What warrants a second pair of eyes
- Review the home-slug contract in `server.mjs`, `entry-server.tsx`, `entry-client.tsx`, and
  `App.tsx` together; server/client property names and deletion timing must remain aligned.
- The large Mermaid diagram chunks remain intentionally on-demand. Confirm CDN/cache behavior on
  a real Mermaid-heavy note after deployment.
- The first client visit to a note still fetches the note index after hydration for backlinks and
  wiki-link verification; this is deliberate and should be measured separately if that interaction
  becomes a bottleneck.

### What should be done in the future
- Deploy to a preview environment, measure the live 934-note root/note HTML and HTTP transfer
  sizes, and compare them to the documented production baseline.
- Add browser network assertions for exact language/Mermaid chunk requests.
- Consider lazy-loading the tree/sidebar in a future ticket if its 284 KB payload remains material.

### Code review instructions
- Review commits `88af573`, `f84d634`, `7d1a490`, `208f105`, and `0b032b5` in order.
- Validate with:
  ```bash
  cd web
  pnpm check
  pnpm exec vitest run src/entry-server.test.tsx
  pnpm build:all
  pnpm smoke:ssr
  ```

### Technical details
- Final final validation baseline:
  - initial client `main`: 388.78 KB raw / 125.83 KB gzipped
  - `NotePage`: 72.77 KB raw / 25.90 KB gzipped
  - Mermaid core: 145.16 KB gzipped, fetched only for Mermaid blocks
  - individual highlight language chunks: 0.32–4.30 KB gzipped for the curated set
  - fixture SSR root response: 37,211 B; browser hydration smoke: PASS

## Step 6: Make the base paper surface true white

The user requested a true white background instead of the aged-paper off-white. I changed the
base paper/background token and related surface foreground tokens from `#f0ede8` to `#ffffff`.
The deliberately separate panel and chrome gray tokens remain unchanged, preserving the retro
window hierarchy while making the page canvas white.

### Prompt Context

**User prompt (verbatim):** "make the background real white and not off white in the css"

**Assistant interpretation:** Change the global background/paper design token to pure white,
without flattening intentional UI panel/chrome contrast.

**Inferred user intent:** Remove the aged-paper tint from the primary page background.

**Commit (code):** 4542d8514a96ff7a49b39d4aee3ddf4846bd9154 — "style: use white base background"

### What I did
- Updated `web/src/index.css` `--color-paper`, `--background`, popover/input, and inverse
  foreground tokens to `#ffffff`.
- Updated the design-token comments to describe a true-white foundation.
- Kept `--color-panel`, `--color-panel-dark`, and `--color-chrome` unchanged.
- Ran `pnpm check` and `pnpm build` successfully.

### Why
The page canvas uses `--color-paper`/`--background`; changing only an individual component would
leave other paper surfaces visibly off-white.

### What worked
- TypeScript and production build passed.
- The main client artifact remains effectively unchanged at 125.82 KB gzipped.

### What didn't work
- N/A

### What I learned
- The visual paper surface and retro panel/chrome shades are independently tokenized, so the
  requested white background does not require removing UI contrast.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm visually after deployment that panel gray remains desirable against pure white.

### What should be done in the future
- N/A

### Code review instructions
- Review `web/src/index.css` design tokens near `--color-paper` and `:root`.
- Validate: `cd web && pnpm check && pnpm build`.

### Technical details
- Base white value: `#ffffff`; retained panel value: `#e8e4de`.

## Step 7: Bump Go to the patched standard-library release for PR CI

PR #10's Dependency Scanning workflow failed in the Go Vulnerability Check job because CI selected
Go 1.26.4. `govulncheck` reported reachable standard-library vulnerabilities GO-2026-5856
(crypto/tls Encrypted Client Hello privacy leak) and GO-2026-4970 (trailing-slash symlink root
escape in os); both are fixed in Go 1.26.5. The repository's `go.mod` is the CI version source and
the Docker builder must match it, so both pins were updated together.

### Prompt Context

**User prompt (verbatim):** "Address the bump needed for https://github.com/go-go-golems/publish-vault/actions/runs/29373295744/job/87221273797?pr=10"

**Assistant interpretation:** Inspect the failing PR workflow, identify the necessary version bump,
apply it consistently, validate the same vulnerability scan locally, and push the result to PR #10.

**Inferred user intent:** Restore green CI using the smallest secure runtime upgrade required by
`govulncheck`.

**Commit (code):** 2d0df8090d207f968424b0969ccb7cd930294066 — "build: bump Go to 1.26.5"

### What I did
- Read the failed GitHub Actions job with `gh run view 29373295744 --log-failed`.
- Confirmed CI's two reachable standard-library findings and their fixed version: Go 1.26.5.
- Changed `go.mod` from `go 1.26.4` to `go 1.26.5`.
- Changed the builder stage in `Dockerfile` from `golang:1.26.4-alpine` to `golang:1.26.5-alpine`.
- Ran `GOWORK=off go test ./... -count=1`.
- Ran `GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.

### Why
GitHub Actions uses `go-version-file: go.mod`, while Docker would otherwise keep compiling with
the vulnerable 1.26.4 standard library. Pinning both surfaces prevents CI/runtime drift.

### What worked
- Go auto-selected `go1.26.5 linux/amd64` locally.
- All Go package tests passed.
- `govulncheck` now reports: `No vulnerabilities found.` and `Your code is affected by 0 vulnerabilities.`

### What didn't work
- The original PR workflow failed with exit code 3 because Go 1.26.4 exposed the two reachable
  standard-library vulnerabilities. No application-code change was needed.

### What I learned
- The vulnerability workflow is correctly following `go.mod`; upgrading a module dependency would
  not fix a standard-library finding. The language/runtime pin is the required remediation.

### What was tricky to build
- The important invariant is aligning the CI toolchain source (`go.mod`) and container runtime
  source (`Dockerfile`). Updating only the former would make CI green but leave production images
  vulnerable.

### What warrants a second pair of eyes
- Confirm the CI run uses Go 1.26.5 after the branch push and that the dependency scanning job
  becomes green.

### What should be done in the future
- N/A

### Code review instructions
- Review the version-only changes in `go.mod` and `Dockerfile`.
- Validate with:
  ```bash
  GOWORK=off go test ./... -count=1
  GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  ```

### Technical details
- CI findings fixed by Go 1.26.5: GO-2026-5856 (`crypto/tls`) and GO-2026-4970 (`os`).

## Step 8: Address PR #10 inline review feedback

The automated review identified two P2 correctness issues in the newly lazy-loaded rendering path.
The first allowed a previous note body to remain visible for one committed frame when React reused
`NoteRenderer` during SPA navigation. The second loaded canonical highlight.js languages for aliases
but asked highlight.js to interpret the original alias class. Both fixes preserve the bundle design
while making the state transition and alias behavior explicit.

### Prompt Context

**User prompt (verbatim):** "Address the code review comments on : https://github.com/go-go-golems/publish-vault/pull/10"

**Assistant interpretation:** Read PR #10's inline review threads, fix every actionable issue,
validate the changes, document them, and push the result.

**Inferred user intent:** Resolve reviewer-identified correctness regressions before merge.

**Commit (code):** 35d910b59c36bfde9917d1eceaf269821e0e03bc — "fix: address note rendering review feedback"

### What I did
- Retrieved PR review comments through `gh api repos/go-go-golems/publish-vault/pulls/10/comments`.
- In `NoteRenderer`, tracked the currently rendered slug/HTML in a ref and conditionally reset
  `resolvedHtml` during render when either changes, so the raw new note body commits immediately.
- In `highlightLanguages`, replaced alias-sensitive `highlightElement` use with
  `hljs.highlight(source, { language: normalizedLanguage })`, assigned the resulting HTML, and
  added the `hljs` class. Unlabelled or unsupported blocks retain bounded auto-detection.
- Ran Prettier, `pnpm check`, 13 SSR unit tests, and `pnpm build:all`.

### Why
A post-paint effect is too late when a component instance receives a different note: it can show
new metadata with old body content. Likewise, `language-shell` normalizes to `bash`, but
`highlightElement` re-reads `language-shell`; explicit canonical highlighting ensures the loaded
language is actually applied.

### What worked
- TypeScript passed.
- All 13 SSR tests passed.
- Client and SSR production builds passed.

### What didn't work
- N/A

### What I learned
- Lazy post-processing must not be the only mechanism that updates the visible source HTML across
  SPA note changes; the raw source needs a synchronous reset first.
- A normalized language name must be passed explicitly when the DOM class is an application alias.

### What was tricky to build
The reset had to preserve hydration safety from Step 5. The conditional state update runs only when
note identity/content differs, so the initial server/client render still begins with identical raw
HTML, while later navigation never commits stale content.

### What warrants a second pair of eyes
- Test a real `language-shell` block after deployment and confirm both highlighting and copy text.
- Test rapid A→B→A note navigation with slow network throttling to confirm no stale body flash.

### What should be done in the future
- Add DOM/browser-level tests for alias highlighting and same-component note transitions when a
  DOM test environment is introduced.

### Code review instructions
- Review `NoteRenderer.tsx` near `resolvedHtml` and `highlightLanguages.ts` near the final
  highlighting loop.
- Validate: `cd web && pnpm check && pnpm exec vitest run src/entry-server.test.tsx && pnpm build:all`.

### Technical details
- Addressed threads: `discussion_r3583097063` (note change reset) and
  `discussion_r3583097070` (canonical highlight alias).
