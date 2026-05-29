# Tasks

## TODO

- [x] Phase 1 — Ticket setup and planning
  - [x] Create `RETRO-ASSETS-005` with topics `assets,images,config,page-title,obsidian-vault`.
  - [x] Add design document: `design-doc/01-image-serving-and-page-title-implementation-guide.md`.
  - [x] Add diary document: `reference/01-diary.md`.
  - [x] Capture evidence-backed current-state analysis for server routes, parser/vault HTML pipeline, API config, and frontend title handling.
  - [x] Relate key source files to the design document.
- [x] Commit the planning artifacts.

- [x] Phase 2 — Backend HTML image URL rewriting
- [x] Add parser tests for rewriting `<img src="...">` URLs through a resolver.
- [x] Implement `parser.RewriteImageSources(html, resolver)` for quoted image `src` attributes.
- [x] Add vault helper for resolving a note-relative image URL to `/assets/{clean-vault-relative-path}`.
- [x] Call image-source rewriting from `vault.rebuildHTML` after wiki-link rewrites.
- [x] Add vault tests for same-directory, nested-directory, and traversal image paths.
- [x] Run `cd backend && go test ./... -count=1`.
- [x] Commit the focused parser/vault image rewrite work.

- [x] Phase 3 — Backend asset HTTP serving and page-title config
- [x] Add `PageTitle` to `serve.Settings` and `server.Config`.
- [x] Add a `--page-title` Glazed flag with help text and defaulting behavior.
- [x] Extend `api.SiteConfig` with `pageTitle`.
- [x] Change API handler construction so `/api/config` can return both `vaultName` and `pageTitle`.
- [x] Add a safe `/assets/{path}` handler that reads from `RuntimeState.ResolvedRoot()` at request time.
- [x] Register `/assets/` before the SPA `PathPrefix("/")` catch-all.
- [x] Reject empty paths, absolute paths, traversal paths, hidden path components, directories, and Markdown files.
- [x] Add server/API tests for asset serving, rejection cases, and config JSON.
- [x] Run `cd backend && go test ./... -count=1`.
- [x] Commit the focused server/API config and asset-serving work.

- [x] Phase 4 — Frontend title consumption
- [x] Add `pageTitle` to `web/src/store/vaultApi.ts` `SiteConfig`.
- [x] Update the static vault config provider, if needed, to include `pageTitle`.
- [x] Add a `useEffect` in `web/src/App.tsx` that sets `document.title` from `config.pageTitle`, falling back to `config.vaultName` or `Retro Obsidian Publish`.
- [x] Run frontend validation (`pnpm --dir web typecheck` or the available equivalent).
- [x] Commit the focused frontend title work.

- [x] Phase 5 — End-to-end validation and ticket closeout
- [x] Build or create a temp vault containing a Markdown note and an image attachment.
- [x] Verify `/api/notes/{slug}` emits `/assets/...` image URLs.
- [x] Verify `/assets/...` returns the image bytes.
- [x] Verify `/api/config` includes the configured `pageTitle`.
  - [ ] Optionally verify the SPA updates the browser title in Playwright.
- [x] Update the diary with all commands, failures, commits, and follow-ups.
- [x] Update changelog and task checkboxes.
- [x] Run `docmgr doctor --ticket RETRO-ASSETS-005 --stale-after 30` and resolve vocabulary/frontmatter warnings.
