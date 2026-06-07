# Tasks

## Phase 1: Store refactor + entry points (no sidecar yet)

- [x] 1.1 Refactor `store/store.ts` to export `makeStore(preloadedState?)` factory + `store` singleton
- [x] 1.2 Update `hooks/redux.ts` types to use `makeStore` return types
- [x] 1.3 Create `entry-client.tsx` with `hydrateRoot` + `window.__PRELOADED_STATE__` support
- [x] 1.4 Update `vite.config.ts` rollupOptions.input to point at `entry-client.tsx`
- [x] 1.5 Add `ssr.noExternal` config to `vite.config.ts` for SSR dependencies
- [x] 1.6 Verify `pnpm build` succeeds and app works identically in browser

## Phase 2: SSR entry point

- [x] 2.1 Create `entry-server.tsx` with `renderApp(url, data)` function
- [x] 2.2 Implement SSR-compatible route matching (bypass Wouter, render page components directly)
- [x] 2.3 Add `build:ssr` script to `package.json`: `vite build --ssr src/entry-server.tsx --outDir dist/ssr`
- [x] 2.4 Add `build:all` script: `pnpm build && pnpm build:ssr`
- [x] 2.5 Verify `pnpm build:ssr` produces `dist/ssr/entry-server.js`
- [x] 2.6 Write unit test for `entry-server.tsx` (`renderApp` with mock data)

## Phase 3: Node.js sidecar

- [x] 3.1 Create `server.mjs` with Express app + health check endpoint
- [x] 3.2 Implement URL parsing for `/`, `/note/{slug}`, `/search` routes
- [x] 3.3 Implement data pre-fetching from Go API (`config`, `notes`, `tree`, `note`)
- [x] 3.4 Implement HTML assembly: inject SSR content into `index.html` template
- [x] 3.5 Inject `window.__PRELOADED_STATE__` + `<meta>` tags + `<title>` + JSON-LD
- [x] 3.6 Add `<noscript>` fallback content for non-JS agents
- [x] 3.7 Add `ssr` script to `package.json`: `node server.mjs`
- [x] 3.8 Test manually: Go server + sidecar, verify page source has real content

## Phase 4: Go server SSR proxy

- [x] 4.1 Add `SSRURL` field to `server.Config` struct
- [x] 4.2 Add `--ssr-url` flag to `serve.go` command definition
- [x] 4.3 Implement `newSSRProxy()` in `server.go` with fallback to SPA handler
- [x] 4.4 Wire SSR proxy into the router when `SSRURL` is set
- [x] 4.5 Write tests: SSR proxy routes pages, skips API/assets, falls back on failure

## Phase 5: Docker + deployment

- [x] 5.1 Create `ssr.Dockerfile` for Node.js sidecar
- [x] 5.2 Update `docker-compose.yml` with sidecar service
- [x] 5.3 Verify full deployment works end-to-end

## Phase 6: SEO and a14y verification

- [x] 6.1 Verify `curl` against pages shows real HTML content
- [x] 6.2 Verify hydration correctness (no React warnings in console)
- [x] 6.3 Verify SPA fallback works when sidecar is down
