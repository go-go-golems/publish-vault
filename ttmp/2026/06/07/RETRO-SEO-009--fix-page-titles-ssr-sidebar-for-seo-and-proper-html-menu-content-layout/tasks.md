# Tasks

## Track A: Original SEO/title cleanup

- [ ] 1. Fix deployment `pageTitle` — set meaningful page title in devctl profile and k8s deployment config.
- [x] 2. Fix React client `document.title` — make it include the current note's title, not just config.vaultName.
- [x] 3. Add breadcrumb navigation to SSR output for SEO.
- [x] 4. Add minimal sidebar/navigation links to SSR for crawlers that don't follow JavaScript.
- [x] 5. Improve HTML semantic structure — add `<nav>`, `<article>`, `<main>` elements to the hydrated output.
- [ ] 6. Test and validate: verify page titles, meta tags, hydration, and accessibility.

## Track B: React Router + full hydration refactor

- [x] 7. Baseline docs/safety commit: commit existing RETRO-SEO-009 docs and vocabulary changes before code refactor.
- [x] 8. Add React Router dependency and update `web/package.json` / `web/pnpm-lock.yaml`.
- [x] 9. Replace Wouter route declarations in `web/src/App.tsx` with React Router routes while preserving `/`, `/note/*`, `/search`, and fallback URLs.
- [x] 10. Migrate `VaultLayout.tsx` from Wouter `useLocation` to React Router `useNavigate`.
- [x] 11. Migrate `NotePage.tsx` from Wouter navigation to React Router navigation.
- [x] 12. Migrate `SearchPage.tsx` and `NotFound.tsx` from Wouter navigation to React Router navigation.
- [x] 13. Export router-agnostic `AppRoutes`/`AppShell` so server and client can wrap the same tree with different routers.
- [x] 14. Rework `entry-server.tsx` to render `<StaticRouter location={url}><AppRoutes /></StaticRouter>` instead of simplified SSR-only page components.
- [x] 15. Keep and adapt RTK Query preloading (`preloadCache`) so the real app tree has config, notes, tree, and current note in cache before `renderToString()`.
- [x] 16. Rework `entry-client.tsx` to use `hydrateRoot()` and remove `root.textContent = ""`.
- [x] 17. Remove or quarantine duplicated route parsing from `entry-server.tsx`; keep `server.mjs` parsing only for sidecar data prefetch/meta decisions.
- [x] 18. Make first-render output deterministic: remove live `new Date()` rendering or gate it behind a hydration-safe component.
- [x] 19. Align SSR and client page-title behavior: note pages remain `Note Title — Site Title` after hydration.
- [x] 20. Update SSR tests to validate real-app rendering, preloaded state, and route coverage.
- [x] 21. Run `pnpm --dir web check`, `pnpm --dir web build:all`, and relevant SSR tests after each major phase.
- [ ] 22. Run final full validation: `GOWORK=off go test ./...`, `pnpm --dir web check`, `pnpm --dir web build:all`, and manual hydration warning check.
- [ ] 23. Update diary after each implementation phase with commands, failures, commits, and review notes.
- [ ] 24. Upload updated document bundle to reMarkable after the refactor guide and after final implementation notes.
