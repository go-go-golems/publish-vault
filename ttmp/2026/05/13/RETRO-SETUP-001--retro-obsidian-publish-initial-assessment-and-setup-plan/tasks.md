# Tasks

## Phase 0: Ticket planning and baseline

- [x] Create docmgr ticket workspace for the setup assessment.
- [x] Add a primary design/implementation guide document.
- [x] Add an investigation diary document.
- [x] Inspect current backend CLI, API, vault, parser, search, watcher, frontend, Vite, Docker, and Makefile setup.
- [x] Run baseline validation commands and record outcomes.
- [x] Keep the diary updated after each implementation phase.
- [x] Commit documentation/ticket updates separately from code changes when practical.

## Phase 1: Move frontend into `web/`

- [x] Move frontend source and tooling into `web/` with `git mv` where possible.
- [x] Update `web/vite.config.ts` for `web/` root, `web/dist` output, and new aliases.
- [x] Update `web/tsconfig*.json`, `web/components.json`, and `web/.storybook/*` paths.
- [x] Update root `.gitignore` for `web/public/__manus__/version.json` and build outputs.
- [x] Update root `Makefile` to use `pnpm --dir web ...`.
- [x] Validate `pnpm --dir web install --frozen-lockfile`, `pnpm --dir web check`, and `pnpm --dir web build`.
- [x] Commit the frontend layout migration.

## Phase 2: Make the Go app a single binary with embedded web assets

- [x] Add `backend/internal/web` with embedded and development filesystem variants.
- [x] Add a SPA handler that serves embedded `web/dist` assets and falls back to `index.html` for client routes.
- [x] Extract backend startup into `backend/internal/server` with API routes plus optional SPA serving.
- [x] Ensure the binary can be built with `-tags embed` after web assets are generated.
- [x] Validate API and SPA serving from the Go process.
- [x] Commit the single-binary server extraction and embed support.

## Phase 3: Replace backend flags with Glazed command tree

- [x] Add Glazed/Cobra dependencies.
- [x] Create `backend/cmd/retro-obsidian-publish/main.go`.
- [x] Use directory-per-verb structure under `backend/cmd/retro-obsidian-publish/commands/`.
- [x] Implement one file per verb: `serve/serve.go` and `build/web.go`.
- [x] Implement `serve` as a Glazed command with schema-backed `--vault`, `--port`, and static serving flags.
- [x] Preserve `VAULT_DIR` fallback behavior.
- [x] Validate `help`, `serve --help`, and a live `serve` smoke test.
- [x] Commit the Glazed CLI migration.

## Phase 4: Add Dagger-backed build verb inside the same binary

- [x] Add Dagger dependency to the backend module.
- [x] Implement `retro-obsidian-publish build web` in `commands/build/web.go`.
- [x] Build `web/dist` with Node 22, corepack, pinned pnpm, and a Dagger pnpm cache volume.
- [x] Add `BUILD_WEB_LOCAL=1` fallback for local pnpm builds.
- [x] Wire `go generate` and a Makefile target so embedded builds can regenerate web assets.
- [x] Validate local fallback builds; Dagger path attempted and fell back because the engine image pull timed out.
- [x] Commit the build verb.

## Phase 5: Add devctl support

- [x] Add `.devctl.yaml`.
- [x] Add `plugins/retro-obsidian-publish.py`.
- [x] Implement `config.mutate`, `validate.run`, and `launch.plan`.
- [x] Launch the single Go backend CLI and web/Vite service for development.
- [x] Validate `devctl plugins list`, `devctl plan`, `devctl up --force`, smoke curls, and `devctl down`.
- [x] Commit devctl support.

## Phase 6: Documentation and cleanup

- [x] Update `README.md` for `web/`, single binary, Glazed CLI, build verb, and devctl workflow.
- [x] Update Docker/Compose docs and config for the single app container.
- [x] Remove stale root `server/index.ts`, `Dockerfile.frontend`, and `nginx.conf`.
- [x] Resolve Go version mismatch by updating the backend Dockerfile to Go 1.25.
- [x] Run final validation suite.
- [x] Update diary, changelog, and docmgr doctor.
- [x] Upload final docs to reMarkable.
- [x] Commit final docs/cleanup.

## Follow-up Phase 7: Review, correctness, and test hardening

- [ ] Review implementation commit `df40b4e` and record review notes for the main migration files.
- [ ] Fix watcher/search-index consistency so file changes update both the vault map and Bleve index.
- [ ] Add parser edge-case tests.
- [ ] Add API route smoke tests.
- [ ] Add static SPA serving tests, including `/api/*` exclusion from SPA fallback.
- [ ] Add CLI smoke tests for help output and required vault handling.
- [ ] Add devctl plugin syntax/handshake smoke test.
- [ ] Remove or gate analytics placeholders in `web/index.html` to eliminate Vite warnings.
- [ ] Validate frontend, backend, single-binary build, and devctl after fixes.
- [ ] Commit Phase 7 implementation and diary updates.

## Follow-up Phase 8: Container and CI hardening

- [ ] Verify `docker build -f backend/Dockerfile -t retro-obsidian-publish .`.
- [ ] Verify container runtime with a mounted vault.
- [ ] Decide CI shape for web check/build, Go tests, build-web fallback, and embedded Go build.
- [ ] Add CI workflow if desired.
- [ ] Decide whether generated embedded assets should remain ignored or be committed.

## Follow-up Phase 9: Release polish

- [ ] Add a `version` command.
- [ ] Consider `--addr` in addition to `--port`.
- [ ] Consider `--no-watch` for deployments that do not need fsnotify.
- [ ] Consider a structured config dump command.
- [ ] Add GoReleaser config for single-binary releases if this becomes a distributed CLI.
