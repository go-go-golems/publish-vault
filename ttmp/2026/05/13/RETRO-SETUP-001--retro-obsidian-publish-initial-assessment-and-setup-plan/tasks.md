# Tasks

## Phase 0: Ticket planning and baseline

- [x] Create docmgr ticket workspace for the setup assessment.
- [x] Add a primary design/implementation guide document.
- [x] Add an investigation diary document.
- [x] Inspect current backend CLI, API, vault, parser, search, watcher, frontend, Vite, Docker, and Makefile setup.
- [x] Run baseline validation commands and record outcomes.
- [ ] Keep the diary updated after each implementation phase.
- [ ] Commit documentation/ticket updates separately from code changes when practical.

## Phase 1: Move frontend into `web/`

- [ ] Move frontend source and tooling into `web/` with `git mv` where possible.
- [ ] Update `web/vite.config.ts` for `web/` root, `web/dist` output, and new aliases.
- [ ] Update `web/tsconfig*.json`, `web/components.json`, and `web/.storybook/*` paths.
- [ ] Update root `.gitignore` for `web/public/__manus__/version.json` and build outputs.
- [ ] Update root `Makefile` to use `pnpm --dir web ...`.
- [ ] Validate `pnpm --dir web install --frozen-lockfile`, `pnpm --dir web check`, and `pnpm --dir web build`.
- [ ] Commit the frontend layout migration.

## Phase 2: Make the Go app a single binary with embedded web assets

- [ ] Add `backend/internal/web` with embedded and development filesystem variants.
- [ ] Add a SPA handler that serves embedded `web/dist` assets and falls back to `index.html` for client routes.
- [ ] Extract backend startup into `backend/internal/server` with API routes plus optional SPA serving.
- [ ] Ensure the binary can be built with `-tags embed` after web assets are generated.
- [ ] Validate API and SPA serving from the Go process.
- [ ] Commit the single-binary server extraction and embed support.

## Phase 3: Replace backend flags with Glazed command tree

- [ ] Add Glazed/Cobra dependencies.
- [ ] Create `backend/cmd/retro-obsidian-publish/main.go`.
- [ ] Use directory-per-verb structure under `backend/cmd/retro-obsidian-publish/commands/`.
- [ ] Implement one file per verb: `serve/serve.go`, `build/web.go`, and any root wiring file.
- [ ] Implement `serve` as a Glazed command with schema-backed `--vault`, `--port`, and static serving flags.
- [ ] Preserve `VAULT_DIR` fallback behavior.
- [ ] Validate `help`, `serve --help`, and a live `serve` smoke test.
- [ ] Commit the Glazed CLI migration.

## Phase 4: Add Dagger-backed build verb inside the same binary

- [ ] Add Dagger dependency to the backend module.
- [ ] Implement `retro-obsidian-publish build web` in `commands/build/web.go`.
- [ ] Build `web/dist` with Node 22, corepack, pinned pnpm, and a Dagger pnpm cache volume.
- [ ] Add `BUILD_WEB_LOCAL=1` fallback for local pnpm builds.
- [ ] Wire `go generate` or a Makefile target so embedded builds can regenerate web assets.
- [ ] Validate Dagger and local fallback builds.
- [ ] Commit the build verb.

## Phase 5: Add devctl support

- [ ] Add `.devctl.yaml`.
- [ ] Add `plugins/retro-obsidian-publish.py`.
- [ ] Implement `config.mutate`, `validate.run`, and `launch.plan`.
- [ ] Launch the single Go backend binary and web/Vite service for development.
- [ ] Validate `devctl plugins list`, `devctl plan`, `devctl up --force`, smoke curls, and `devctl down`.
- [ ] Commit devctl support.

## Phase 6: Documentation and cleanup

- [ ] Update `README.md` for `web/`, single binary, Glazed CLI, build verb, and devctl workflow.
- [ ] Update or de-emphasize Docker/Compose docs.
- [ ] Remove stale root `server/index.ts` if it is no longer needed.
- [ ] Resolve Go version mismatch between `backend/go.mod` and `backend/Dockerfile`.
- [ ] Run final validation suite.
- [ ] Update diary, changelog, and docmgr doctor.
- [ ] Upload final docs to reMarkable if requested.
- [ ] Commit final docs/cleanup.

## Follow-up correctness tasks

- [ ] Fix watcher/search-index consistency so file changes update both the vault map and Bleve index.
- [ ] Add tests for parser edge cases, route smoke tests, and static file serving behavior.
