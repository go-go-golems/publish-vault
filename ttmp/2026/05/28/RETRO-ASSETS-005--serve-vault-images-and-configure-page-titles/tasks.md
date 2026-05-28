# Tasks

## TODO

- [ ] Phase 1 — Ticket setup and planning
  - [x] Create `RETRO-ASSETS-005` with topics `assets,images,config,page-title,obsidian-vault`.
  - [x] Add design document: `design-doc/01-image-serving-and-page-title-implementation-guide.md`.
  - [x] Add diary document: `reference/01-diary.md`.
  - [x] Capture evidence-backed current-state analysis for server routes, parser/vault HTML pipeline, API config, and frontend title handling.
  - [x] Relate key source files to the design document.
  - [ ] Commit the planning artifacts.

- [ ] Phase 2 — Backend HTML image URL rewriting
  - [ ] Add parser tests for rewriting `<img src="...">` URLs through a resolver.
  - [ ] Implement `parser.RewriteImageSources(html, resolver)` for quoted image `src` attributes.
  - [ ] Add vault helper for resolving a note-relative image URL to `/assets/{clean-vault-relative-path}`.
  - [ ] Call image-source rewriting from `vault.rebuildHTML` after wiki-link rewrites.
  - [ ] Add vault tests for same-directory, nested-directory, and traversal image paths.
  - [ ] Run `cd backend && go test ./... -count=1`.
  - [ ] Commit the focused parser/vault image rewrite work.

- [ ] Phase 3 — Backend asset HTTP serving and page-title config
  - [ ] Add `PageTitle` to `serve.Settings` and `server.Config`.
  - [ ] Add a `--page-title` Glazed flag with help text and defaulting behavior.
  - [ ] Extend `api.SiteConfig` with `pageTitle`.
  - [ ] Change API handler construction so `/api/config` can return both `vaultName` and `pageTitle`.
  - [ ] Add a safe `/assets/{path}` handler that reads from `RuntimeState.ResolvedRoot()` at request time.
  - [ ] Register `/assets/` before the SPA `PathPrefix("/")` catch-all.
  - [ ] Reject empty paths, absolute paths, traversal paths, hidden path components, directories, and Markdown files.
  - [ ] Add server/API tests for asset serving, rejection cases, and config JSON.
  - [ ] Run `cd backend && go test ./... -count=1`.
  - [ ] Commit the focused server/API config and asset-serving work.

- [ ] Phase 4 — Frontend title consumption
  - [ ] Add `pageTitle` to `web/src/store/vaultApi.ts` `SiteConfig`.
  - [ ] Update the static vault config provider, if needed, to include `pageTitle`.
  - [ ] Add a `useEffect` in `web/src/App.tsx` that sets `document.title` from `config.pageTitle`, falling back to `config.vaultName` or `Retro Obsidian Publish`.
  - [ ] Run frontend validation (`pnpm --dir web typecheck` or the available equivalent).
  - [ ] Commit the focused frontend title work.

- [ ] Phase 5 — End-to-end validation and ticket closeout
  - [ ] Build or create a temp vault containing a Markdown note and an image attachment.
  - [ ] Verify `/api/notes/{slug}` emits `/assets/...` image URLs.
  - [ ] Verify `/assets/...` returns the image bytes.
  - [ ] Verify `/api/config` includes the configured `pageTitle`.
  - [ ] Optionally verify the SPA updates the browser title in Playwright.
  - [ ] Update the diary with all commands, failures, commits, and follow-ups.
  - [ ] Update changelog and task checkboxes.
  - [ ] Run `docmgr doctor --ticket RETRO-ASSETS-005 --stale-after 30` and resolve vocabulary/frontmatter warnings.
