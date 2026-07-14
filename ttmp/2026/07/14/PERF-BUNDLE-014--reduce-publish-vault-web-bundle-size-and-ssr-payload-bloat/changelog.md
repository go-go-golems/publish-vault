# Changelog

## 2026-07-14

- Initial workspace created


## 2026-07-14

Created ticket: analysis of bundle size (2.03MB JS / 599KB gz, single chunk) and SSR HTML bloat (1.31MB, ~834KB inlined notes+tree). Wrote intern-grade design doc with 5 phased implementation steps including highlight.js per-language lazy loading via import.meta.glob.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Static hljs+mermaid imports are the heaviest contributors


## 2026-07-14

Validated with docmgr doctor (clean). Uploaded bundle (design doc + diary) to reMarkable at /ai/2026/07/14/PERF-BUNDLE-014.


## 2026-07-14

Step 1 complete (commit 88af5739d58c513b97d396985b53f39e067ed3b1): split NotePage from client entry while keeping SSR eager; main client chunk fell to 126.52 KB gzipped. TypeScript, SSR unit tests, and build:all passed; browser smoke awaits Playwright Chromium installation.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/entry-client.tsx — Client-only React.lazy NotePage import
- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/entry-server.tsx — SSR-eager NotePage injection


## 2026-07-14

Step 2 complete (commit f84d634f2b0a99925237dd5bbd032e485d11e99f): lazy-loaded Mermaid after diagram detection with cancellation. Mermaid moved to a 145.16 KB gzipped chunk; NotePage fell to 329.48 KB gzipped. TypeScript, SSR tests, and build passed.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Dynamic Mermaid import and stale-render cancellation


## 2026-07-14

Step 3 complete (commit 7d1a490633be241d8088ca41200fbc27a16b6895): replaced full highlight.js import with core plus import.meta.glob language chunks and aliases. NotePage is now 26.64 KB gzipped before optional languages/Mermaid; all checks passed.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Async highlighting and cancellation
- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/lib/highlightLanguages.ts — Core highlight.js loader, aliases, bounded auto-detection, and in-flight cache


## 2026-07-14

Steps 4-5 complete (commit 0b032b5610ac450a8f1f3be6d2ee87f7365c13bc): removed full note index from all SSR RTK cache payloads via explicit homeSlug; made route splitting hydration-safe; gated dev-only Vite plugins. 13 SSR tests, build:all, and browser smoke all pass. Fixture SSR root fell to 37,211 B.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/server.mjs — Home slug contract and no listNotes preload
- /home/manuel/code/wesen/go-go-golems/publish-vault/web/vite.config.ts — Dev-only plugin gating


## 2026-07-14

Final documentation bundle refreshed on reMarkable after implementation and validation (forced overwrite after dry-run): /ai/2026/07/14/PERF-BUNDLE-014/PERF-BUNDLE-014 — Bundle Size Reduction.pdf.


## 2026-07-14

Step 6 complete (commit 4542d8514a96ff7a49b39d4aee3ddf4846bd9154): changed the base paper/background and related surface tokens from aged off-white to #ffffff; retained retro panel/chrome gray contrast. Typecheck and production build passed.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/index.css — True-white base paper and background tokens


## 2026-07-14

Step 7 (commit 2d0df8090d207f968424b0969ccb7cd930294066): bumped go.mod and Docker builder from Go 1.26.4 to 1.26.5 to remediate CI-reported reachable standard-library CVEs GO-2026-5856 and GO-2026-4970. Go tests and govulncheck pass cleanly.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/Dockerfile — Production builder aligned to 1.26.5
- /home/manuel/code/wesen/go-go-golems/publish-vault/go.mod — CI go-version-file pin raised to 1.26.5


## 2026-07-14

Step 8 (commit 35d910b59c36bfde9917d1eceaf269821e0e03bc): addressed PR #10 P2 review feedback: synchronously reset rendered note HTML on SPA note changes and explicitly highlight with normalized canonical alias names. TypeScript, 13 SSR tests, and build:all pass.

### Related Files

- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/components/organisms/NoteRenderer/NoteRenderer.tsx — Pre-commit reset prevents stale note body flash
- /home/manuel/code/wesen/go-go-golems/publish-vault/web/src/lib/highlightLanguages.ts — Canonical alias highlighting

