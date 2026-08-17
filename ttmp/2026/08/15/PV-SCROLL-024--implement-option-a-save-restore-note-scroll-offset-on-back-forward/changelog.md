# Changelog

## 2026-08-15

- Initial workspace created


## 2026-08-15

Step 1: created ticket and design; corrected scroller-lifecycle premise (NotePage isLoading unmounts ScrollArea) — hook lives in persistent NotePage with scroll-listener capture

### Related Files

- /home/manuel/workspaces/2026-08-15/better-index-links/publish-vault/web/src/components/pages/NotePage/NotePage.tsx — isLoading early return


## 2026-08-16

Step 3: addressed 3 PR #21 review comments (module store, key+hash identity, scrollHeight-clientHeight predicate) + fixed scroller-discovery trap via document capture listener; all 3 browser scenarios pass (commit 2e2ae67)

### Related Files

- /home/manuel/workspaces/2026-08-15/better-index-links/publish-vault/web/src/lib/scrollRestoration.ts — 3 review fixes implemented and verified

