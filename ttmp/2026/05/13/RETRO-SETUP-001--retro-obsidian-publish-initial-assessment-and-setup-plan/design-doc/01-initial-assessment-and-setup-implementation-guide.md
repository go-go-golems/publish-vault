---
Title: Initial assessment and setup implementation guide
Ticket: RETRO-SETUP-001
Status: active
Topics:
    - glazed
    - frontend
    - dagger
    - devctl
    - pnpm
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .devctl.yaml
      Note: Implementation now realizes the setup design
    - Path: Dockerfile.frontend
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: Makefile
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: README.md
      Note: |-
        Evidence and migration target referenced by the initial setup implementation guide
        Implementation now realizes the setup design
    - Path: backend/Dockerfile
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/cmd/retro-obsidian-publish/commands/build/web.go
      Note: Implementation now realizes the setup design
    - Path: backend/cmd/retro-obsidian-publish/commands/root.go
      Note: Implementation now realizes the setup design
    - Path: backend/cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Implementation now realizes the setup design
    - Path: backend/cmd/retro-obsidian-publish/main.go
      Note: Implementation now realizes the setup design
    - Path: backend/cmd/server/main.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/api/api.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/parser/parser.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/search/search.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/server/server.go
      Note: Implementation now realizes the setup design
    - Path: backend/internal/vault/vault.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/watcher/watcher.go
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: backend/internal/web/embed.go
      Note: Implementation now realizes the setup design
    - Path: backend/internal/web/embed_none.go
      Note: Implementation now realizes the setup design
    - Path: backend/internal/web/static.go
      Note: Implementation now realizes the setup design
    - Path: client/src/App.tsx
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: client/src/main.tsx
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: client/src/store/vaultApi.ts
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: docker-compose.yml
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: package.json
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: plugins/retro-obsidian-publish.py
      Note: Implementation now realizes the setup design
    - Path: vite.config.ts
      Note: Evidence and migration target referenced by the initial setup implementation guide
    - Path: web/package.json
      Note: Implementation now realizes the setup design
    - Path: web/vite.config.ts
      Note: Implementation now realizes the setup design
ExternalSources:
    - devctl help user-guide
    - devctl help scripting-guide
    - devctl help plugin-authoring
Summary: Assessment and implementation guide for migrating the project to Glazed CLI commands, a web/ pnpm layout, Dagger bundling, and devctl orchestration.
LastUpdated: 2026-05-13T13:45:00-04:00
WhatFor: Use this to onboard an intern and guide the first implementation pass for repository setup and toolchain migration.
WhenToUse: Before changing CLI wiring, moving frontend files, replacing Docker-only builds, or adding devctl support.
---






























# Initial assessment and setup implementation guide

## Executive summary

This ticket is an initial assessment and implementation guide for turning `retro-obsidian-publish` into a more standard Go plus web application with four setup goals:

1. Replace the backend's direct `flag` parsing with a Glazed-backed command setup using Glazed schemas and command sections.
2. Move the frontend application into a dedicated `web/` folder and keep pnpm as the package manager.
3. Add a Dagger-backed bundling/build path so the web build is reproducible and cacheable.
4. Add devctl support so a new developer can run the full stack with `devctl plan`, `devctl up`, `devctl status`, `devctl logs`, and `devctl down`.

The current repository already has a clear split between a Go backend under `backend/` and a Vite/React frontend under `client/`, but the JavaScript package metadata and Vite config live at the repository root. The backend currently starts from `backend/cmd/server/main.go` and reads only two command-line flags, `--vault` and `--port`, with an environment fallback for `VAULT_DIR` (`backend/cmd/server/main.go:24-36`). The frontend is built with pnpm scripts in root `package.json` (`package.json:6-14`), and Vite points its root at `client/` while outputting to `dist/public` (`vite.config.ts:217-221`). The project also has Docker Compose and Dockerfiles, but no Dagger or devctl files yet.

The recommended implementation is incremental:

- First stabilize repository layout by moving frontend package files and Vite config into `web/` while preserving a root-level Makefile or task aliases.
- Then refactor the Go backend entrypoint into a Glazed command tree with a `serve` command and typed settings struct.
- Then add a Dagger-powered `cmd/build-web` command with a local fallback and a pinned pnpm version.
- Finally add `plugins/retro-obsidian-publish.py` and `.devctl.yaml` so devctl can validate prerequisites and supervise the backend and frontend services.

The first validation finding is that `cd backend && go test ./...` passes, but `pnpm check` fails because `node_modules` is missing. The failure is expected in a fresh checkout and should become an actionable devctl validation warning or prepare step.

## Problem statement and scope

The project is currently runnable, but a new intern has to infer setup knowledge from several places:

- Backend flags are defined in Go code, not in a reusable command framework.
- Frontend files are partly under `client/` and partly at the repository root.
- Build behavior is split across `Makefile`, `package.json`, Dockerfiles, and Docker Compose.
- There is no devctl plugin that tells a contributor which tools are required, which ports are used, or how the backend and frontend should be supervised.

The requested project setup is therefore not a feature change to Obsidian parsing or React rendering. It is a developer-experience and build-system migration. The scope of this guide is:

- Explain the current system and file layout.
- Identify the repo changes needed for Glazed, pnpm under `web/`, Dagger bundling, and devctl.
- Provide implementation-level pseudocode and API references for each part.
- Provide a phased plan that an intern can execute and validate.

Out of scope for this first setup ticket:

- Rewriting API behavior or changing response schemas.
- Replacing `gorilla/mux` with another router.
- Designing a public deployment platform.
- Completing a full static-site publishing pipeline beyond reproducible bundling.
- Adding broad unit test coverage for parser/search/vault internals, except where small smoke tests are needed for setup validation.

## Current-state architecture

### Repository shape

Observed high-level repository layout:

```text
retro-obsidian-publish/
  backend/                  Go module and server implementation
  client/                   Vite frontend root used by current vite.config.ts
  server/index.ts           Express static-file wrapper bundled by root package build
  shared/                   Shared TypeScript constants
  .storybook/               Storybook configuration
  package.json              Root web package metadata and scripts
  pnpm-lock.yaml            Root pnpm lockfile
  vite.config.ts            Root Vite config; uses client/ as Vite root
  Makefile                  Thin backend/frontend/dev wrappers
  Dockerfile.frontend       Nginx frontend image build
  docker-compose.yml        Backend + frontend Docker Compose stack
```

The README describes the intended architecture as Go backend plus React/Vite frontend with atomic component folders and API integration. The actual file tree is consistent with that description, but the web package is not isolated into a `web/` directory yet.

### Backend entrypoint and runtime

The backend entrypoint is `backend/cmd/server/main.go`. It currently uses the Go standard library `flag` package:

- `--vault` is declared at `backend/cmd/server/main.go:25`.
- `--port` is declared at `backend/cmd/server/main.go:26`.
- `VAULT_DIR` is used as fallback when `--vault` is omitted at `backend/cmd/server/main.go:29-35`.
- The server constructs its listen address with `":" + *port` at `backend/cmd/server/main.go:76`.

The backend start sequence is:

```text
parse flags/env
  -> absolute vault path
  -> vault.New(absVault)
  -> search.New(vault)
  -> watcher.New(vault)
  -> mux.NewRouter()
  -> api.New(vault, search).Register(router)
  -> http.ListenAndServe(port, cors.Handler(router))
```

Evidence:

- Vault loading happens at `backend/cmd/server/main.go:43-48`.
- In-memory search index creation happens at `backend/cmd/server/main.go:50-54`.
- File watcher creation happens at `backend/cmd/server/main.go:56-62`.
- REST route registration happens at `backend/cmd/server/main.go:64-67`.
- CORS allows all origins and only `GET` plus `OPTIONS` at `backend/cmd/server/main.go:69-74`.

The backend module is nested under `backend/go.mod`. One important issue: `backend/go.mod` declares `go 1.25.0`, while `backend/Dockerfile` uses `golang:1.23-alpine` at `backend/Dockerfile:1`. That mismatch should be fixed during the setup pass, either by lowering the module `go` directive to an installed/supported version or by updating Docker to a matching Go image.

### Backend domain packages

The backend is split into small internal packages:

```text
backend/internal/parser   Markdown, frontmatter, wiki-link parsing
backend/internal/vault    Filesystem scan, note map, backlinks, tree
backend/internal/search   Bleve index and search result mapping
backend/internal/watcher  fsnotify watcher and vault reloads
backend/internal/api      HTTP JSON handlers
```

The `api` package exposes six routes in `Handler.Register`:

- `GET /api/notes` at `backend/internal/api/api.go:37`
- `GET /api/notes/{slug:.*}` at `backend/internal/api/api.go:38`
- `GET /api/tree` at `backend/internal/api/api.go:39`
- `GET /api/search` at `backend/internal/api/api.go:40`
- `GET /api/tags` at `backend/internal/api/api.go:41`
- `GET /api/graph` at `backend/internal/api/api.go:42`

The `vault` package stores parsed notes in an in-memory map keyed by slug (`backend/internal/vault/vault.go:45-50`). It scans all markdown files under the vault root using `filepath.Walk` (`backend/internal/vault/vault.go:64-98`), skips hidden directories (`backend/internal/vault/vault.go:75-80`), skips non-`.md` files (`backend/internal/vault/vault.go:82-84`), and builds backlinks after loading all notes (`backend/internal/vault/vault.go:96`).

The `parser` package uses goldmark plus extensions, metadata, and custom regex preprocessing for wiki links. The main parser flow is in `parser.Parse`:

- Extract wiki links before goldmark at `backend/internal/parser/parser.go:41-45`.
- Configure goldmark extensions and renderer options at `backend/internal/parser/parser.go:47-65`.
- Extract frontmatter from goldmark context at `backend/internal/parser/parser.go:73`.
- Extract title, tags, and excerpt at `backend/internal/parser/parser.go:76-83`.

The `search` package builds an in-memory Bleve index in `search.New` (`backend/internal/search/search.go:36-49`) and has a persistent index constructor that is currently not used by the main server (`backend/internal/search/search.go:51-73`).

The `watcher` package watches the vault root and subdirectories (`backend/internal/watcher/watcher.go:36-48`) and debounces events every 500 ms (`backend/internal/watcher/watcher.go:60-103`). It reloads or removes notes in the `Vault` but does not currently update the search index. That is a runtime correctness risk unrelated to the setup migration, but the intern should be aware of it when testing live reload.

### Frontend runtime

The frontend is a React app mounted from `client/src/main.tsx`, which renders `App` into `#root` (`client/src/main.tsx:1-5`). `App` sets up the Redux provider and Wouter routes:

- `/` renders the index note (`client/src/App.tsx:12` and `client/src/App.tsx:21-23`).
- `/note/*` renders note pages (`client/src/App.tsx:13` and `client/src/App.tsx:25-30`).
- `/search` renders search (`client/src/App.tsx:14` and `client/src/App.tsx:32-34`).

The API slice in `client/src/store/vaultApi.ts` supports two modes:

- Backend mode if `VITE_API_URL` is set (`client/src/store/vaultApi.ts:27-28`).
- Static/demo mode if `VITE_API_URL` is absent, lazily loading `../vault/staticVault` (`client/src/store/vaultApi.ts:30-34`).

This is important for devctl and Dagger because dev mode should usually set `VITE_API_URL=http://localhost:8080`, while static preview or standalone deployment may omit it.

### Frontend package and Vite config

Root `package.json` currently defines the web scripts:

- `dev`: `vite --host` (`package.json:7`)
- `build`: `vite build && esbuild server/index.ts ...` (`package.json:8`)
- `start`: `NODE_ENV=production node dist/index.js` (`package.json:9`)
- `check`: `tsc --noEmit` (`package.json:11`)
- `storybook`: `storybook dev --port 6006` (`package.json:13`)

The package pins pnpm with `packageManager` at `package.json:111` and keeps patched dependencies in `package.json:112-119`.

The root `vite.config.ts` sets:

- `envDir` to the repository root (`vite.config.ts:217`).
- Vite `root` to `client/` (`vite.config.ts:218`).
- Build output to `dist/public` (`vite.config.ts:219-221`).
- Dev server port to `3000` (`vite.config.ts:223-225`).

Because the requested target is a `web/` folder, the future Vite config should live at `web/vite.config.ts`, package metadata should live at `web/package.json`, and paths should become relative to `web/`. The current `client/` folder can either be renamed to `web/` directly or moved to `web/client/`. The recommended option is direct rename/migration: `client/src` becomes `web/src`, `client/index.html` becomes `web/index.html`, and `client/public` becomes `web/public`. This avoids preserving an unnecessary nested `client/` layer.

### Current build and orchestration

The current Makefile delegates to backend and frontend commands:

- `make backend` runs `cd backend && go build -o bin/server ./cmd/server/` (`Makefile:7-8`).
- `make frontend` runs `pnpm build` (`Makefile:10-12`).
- `make backend-dev` runs the backend with example vault and port 8080 (`Makefile:14-16`).
- `make frontend-dev` sets `VITE_API_URL=http://localhost:8080` and runs `pnpm dev` (`Makefile:18-20`).
- `make dev` runs both in parallel using `make -j2` (`Makefile:26-28`).

Docker Compose defines separate `backend` and `frontend` services (`docker-compose.yml:3-27`). The frontend Dockerfile installs pnpm globally, installs from root `package.json` plus `pnpm-lock.yaml`, builds the app, and serves `/app/dist/public` through nginx (`Dockerfile.frontend:1-15`). This should be replaced or supplemented by Dagger for reproducible local and CI builds.

## Gap analysis against requested outcomes

### Glazed CLI migration gaps

Current state:

- The backend command uses `flag.String` directly (`backend/cmd/server/main.go:24-27`).
- There is no command schema, no sections, no Glazed output flags, no Glazed command settings, and no Glazed help integration.
- The backend has only one implicit verb: start the server.

Target state:

- A root Cobra command initialized with Glazed logging/help conventions.
- A `serve` command implemented as a Glazed command with typed settings.
- Existing `--vault` and `--port` flags represented as Glazed fields.
- Optional future verbs such as `index`, `check-vault`, or `print-config` can be added by adding command structs rather than editing a large `main`.

The user explicitly said there is no need for structured data. Interpreted for Glazed, this means the migration does not have to turn every command into rich tabular output. The important requirement is still to use `cmds.CommandDescription`, fields, and sections so the CLI has consistent schema/help/settings behavior. For long-running `serve`, it is acceptable for the command to log normal server messages and not emit meaningful rows.

### pnpm plus web/ layout gaps

Current state:

- pnpm is already used and pinned, but package files live at repository root (`package.json:111`).
- Vite root points at `client/` from root config (`vite.config.ts:217-221`).
- Storybook config lives at `.storybook/` in the root.
- Patches live at root `patches/` and are referenced from root `package.json` (`package.json:112-115`).

Target state:

- `web/package.json`, `web/pnpm-lock.yaml`, `web/vite.config.ts`, `web/tsconfig*.json`, `web/components.json`, `web/.storybook/`, and `web/patches/` are colocated.
- Source moves from `client/src` to `web/src`, `client/public` to `web/public`, and `client/index.html` to `web/index.html`.
- Root Makefile and devctl commands run `pnpm --dir web ...` or use `cwd: web`.
- Dagger mounts `web/` and uses a cache volume for pnpm's store.

### Dagger bundling gaps

Current state:

- No Dagger module or build command exists.
- Dockerfile-based frontend build installs pnpm globally (`Dockerfile.frontend:3`) and builds from root paths.
- The backend Dockerfile's Go image does not match the module's Go version.

Target state:

- Add `backend/cmd/build-web` or root `cmd/build-web` depending on chosen Go module layout.
- The command pins Node and pnpm, uses corepack, mounts a pnpm store cache, runs `pnpm install --frozen-lockfile`, and runs `pnpm build` in `web/`.
- The command copies `web/dist/` into a stable Go-servable or distributable location if the project chooses to bundle frontend into the Go binary.
- A local fallback (`BUILD_WEB_LOCAL=1`) runs the same pnpm commands on the host for developers without Docker/Dagger.

### devctl gaps

Current state:

- No `.devctl.yaml` exists.
- No devctl plugin exists.
- `make dev` can run both services, but it does not validate prerequisites, capture state, expose logs consistently, or provide a machine-readable launch plan.

Target state:

- `.devctl.yaml` points at a plugin under `plugins/`.
- The plugin implements at least `config.mutate`, `validate.run`, and `launch.plan`.
- Optionally add `prepare.run` for `pnpm install` and `build.run` for backend/web builds after the first iteration.
- `devctl plan` tells an intern which ports, vault path, and URLs will be used.
- `devctl up` supervises a backend service and a Vite web service.

## Proposed target architecture

### Target repository layout

Recommended layout after migration:

```text
retro-obsidian-publish/
  backend/
    go.mod
    go.sum
    cmd/
      retro-obsidian-publish/
        main.go                    # root Cobra/Glazed command wiring
      build-web/
        main.go                    # Dagger-backed web bundle builder
    internal/
      api/
      parser/
      search/
      vault/
      watcher/
      server/                      # extracted server runtime, optional but recommended
    vault-example/
    Dockerfile

  web/
    package.json
    pnpm-lock.yaml
    vite.config.ts
    tsconfig.json
    tsconfig.node.json
    components.json
    index.html
    public/
    src/
    .storybook/
    patches/

  plugins/
    retro-obsidian-publish.py       # devctl NDJSON plugin

  .devctl.yaml
  Makefile
  README.md
  docker-compose.yml               # optional, either update or de-emphasize
```

The major path decisions are:

- Keep the Go module under `backend/` for the first pass. This is the least invasive option because all existing Go imports assume `retro-obsidian-publish/backend/...`.
- Move all frontend-specific tooling into `web/` to match the user's requirement and reduce root clutter.
- Keep root `Makefile` as a compatibility wrapper around new locations.
- Put devctl plugin code under root `plugins/` because devctl config belongs at repository root.

### Architecture diagram

```text
Developer machine

  devctl up
      |
      v
  .devctl.yaml
      |
      v
  plugins/retro-obsidian-publish.py
      | config.mutate: ports, paths, URLs
      | validate.run: go/node/pnpm/web package/vault path checks
      | launch.plan: backend + web service specs
      v
  devctl supervisor
      |---------------------------|
      v                           v
  backend service             web service
  cwd=backend                 cwd=web
  go run ./cmd/... serve      pnpm dev --host 127.0.0.1 --port 3000
  --vault vault-example       VITE_API_URL=http://127.0.0.1:8080
  --port 8080
      |                           |
      | REST JSON                 | browser UI
      | /api/*                    |
      |<--------------------------|
```

Dagger build path:

```text
web/ package + lockfile
      |
      v
cmd/build-web (Dagger client)
      |
      v
Dagger container node:22
  corepack enable pnpm@pinned
  pnpm install --frozen-lockfile --store-dir /pnpm/store
  pnpm build
      |
      v
web/dist/ or backend/internal/web/embed/public/
      |
      v
backend binary or static artifact bundle
```

### Data/API flow diagram

```text
Obsidian vault directory
  (*.md files)
      |
      v
backend/internal/vault.LoadAll
      |
      +--> backend/internal/parser.Parse
      |       - frontmatter
      |       - wiki links
      |       - HTML
      |       - tags/title/excerpt
      |
      +--> backend/internal/vault.buildBacklinks
      |
      +--> backend/internal/search.New (Bleve mem index)
      |
      v
backend/internal/api.Handler.Register
      |
      +--> GET /api/notes
      +--> GET /api/notes/{slug}
      +--> GET /api/tree
      +--> GET /api/search?q=...
      +--> GET /api/tags
      +--> GET /api/graph
      |
      v
web/src/store/vaultApi.ts
      |
      v
React pages and components
```

## Glazed CLI design

### Intern mental model

Glazed is a command framework layered on Cobra. Cobra handles the shell-level command tree and argument dispatch. Glazed adds schema-backed fields, sections, settings decoding, output settings, command introspection, and help integration.

For this project, the goal is not to produce tables of server events. The goal is to make the backend command self-describing and extensible. Every flag should be declared as a Glazed field and decoded into a typed settings struct. That allows later tools to print schemas, inspect defaults, add environment/config sources, and reuse command sections.

### Proposed commands

Start with one explicit verb:

```text
retro-obsidian-publish serve --vault ./backend/vault-example --port 8080
```

Optional follow-up verbs:

```text
retro-obsidian-publish check-vault --vault ./backend/vault-example
retro-obsidian-publish print-config --vault ./backend/vault-example --port 8080 --output yaml
retro-obsidian-publish version
```

Only `serve` is required for the initial migration.

### Proposed backend package split

Extract server startup logic out of `main.go` before adding Glazed. This keeps the Glazed command small and testable.

```text
backend/internal/server/
  server.go

backend/cmd/retro-obsidian-publish/
  main.go
  commands/
    root.go
    serve.go
```

`internal/server/server.go` API sketch:

```go
package server

type Config struct {
    VaultDir string
    Port     string
}

func Run(ctx context.Context, cfg Config) error {
    if cfg.VaultDir == "" {
        return fmt.Errorf("vault dir is required")
    }

    absVault, err := filepath.Abs(cfg.VaultDir)
    if err != nil { return err }

    v, err := vault.New(absVault)
    if err != nil { return err }

    si, err := search.New(v)
    if err != nil { return err }

    fw, err := watcher.New(v)
    if err == nil { defer fw.Close() }

    r := mux.NewRouter()
    h := api.New(v, si)
    h.Register(r)

    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: cors.New(cors.Options{...}).Handler(r),
    }

    // Optional: support ctx cancellation for devctl shutdown friendliness.
    go func() {
        <-ctx.Done()
        _ = srv.Shutdown(context.Background())
    }()

    return srv.ListenAndServe()
}
```

### Glazed `serve` command skeleton

API references from the Glazed authoring guide:

- Use `cmds.NewCommandDescription` for command metadata.
- Use `fields.New` for flags and arguments.
- Use `settings.NewGlazedSchema()` for output settings.
- Use `cli.NewCommandSettingsSection()` for schema/debug flags.
- Decode with `vals.DecodeSectionInto(schema.DefaultSlug, settings)`.
- Build with `cli.BuildCobraCommandFromCommand`.

Pseudocode:

```go
package commands

import (
    "context"
    "os"

    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/fields"
    "github.com/go-go-golems/glazed/pkg/cmds/schema"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
    "github.com/go-go-golems/glazed/pkg/middlewares"
    "github.com/go-go-golems/glazed/pkg/settings"

    appserver "retro-obsidian-publish/backend/internal/server"
)

type ServeCommand struct {
    *cmds.CommandDescription
}

type ServeSettings struct {
    Vault string `glazed:"vault"`
    Port  string `glazed:"port"`
}

func NewServeCommand() (*ServeCommand, error) {
    glazedSection, err := settings.NewGlazedSchema()
    if err != nil { return nil, err }

    commandSettingsSection, err := cli.NewCommandSettingsSection()
    if err != nil { return nil, err }

    desc := cmds.NewCommandDescription(
        "serve",
        cmds.WithShort("Serve an Obsidian vault as a retro web app"),
        cmds.WithLong(`Serve scans a vault, builds a search index, watches files, and exposes /api routes.

Examples:
  retro-obsidian-publish serve --vault ./backend/vault-example --port 8080
  VAULT_DIR=/path/to/vault retro-obsidian-publish serve --port 8080
`),
        cmds.WithFlags(
            fields.New("vault", fields.TypeString,
                fields.WithHelp("Path to an Obsidian vault directory. Defaults to VAULT_DIR when omitted."),
            ),
            fields.New("port", fields.TypeString,
                fields.WithDefault("8080"),
                fields.WithHelp("HTTP port for the backend API server."),
            ),
        ),
        cmds.WithSections(glazedSection, commandSettingsSection),
    )

    return &ServeCommand{CommandDescription: desc}, nil
}

func (c *ServeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    s := &ServeSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
        return err
    }

    if s.Vault == "" {
        s.Vault = os.Getenv("VAULT_DIR")
    }
    if s.Vault == "" {
        return fmt.Errorf("--vault or VAULT_DIR is required")
    }

    return appserver.Run(ctx, appserver.Config{VaultDir: s.Vault, Port: s.Port})
}
```

Important implementation notes:

- Keep the existing `VAULT_DIR` behavior because the README documents it and Docker Compose uses it (`docker-compose.yml:12-13`).
- Prefer `port` as a string to preserve current behavior, but validate it before passing to `http.Server`.
- If using Glazed's default output flags, long-running `serve` should not emit rows unless a `--print-config` or dry-run mode is added. It can simply run until interrupted.
- Add a root command with Glazed logging and help wiring. This is important because Glazed applications expect logging/help sections at the root, not ad-hoc per command setup.

### Root command skeleton

```go
func NewRootCommand() (*cobra.Command, error) {
    rootCmd := &cobra.Command{
        Use:   "retro-obsidian-publish",
        Short: "Publish an Obsidian vault with a retro Mac-inspired web UI",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            return logging.InitLoggerFromCobra(cmd)
        },
    }

    if err := logging.AddLoggingSectionToRootCommand(rootCmd, "retro-obsidian-publish"); err != nil {
        return nil, err
    }

    helpSystem := help.NewHelpSystem()
    // Optional later: doc.AddDocToHelpSystem(helpSystem)
    help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

    serveCmd, err := NewServeCommand()
    if err != nil { return nil, err }

    serveCobra, err := cli.BuildCobraCommandFromCommand(serveCmd,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
        }),
    )
    if err != nil { return nil, err }

    rootCmd.AddCommand(serveCobra)
    return rootCmd, nil
}
```

### Glazed validation checklist

After implementing:

```bash
cd backend
go mod tidy
go test ./...
go run ./cmd/retro-obsidian-publish help
go run ./cmd/retro-obsidian-publish serve --help
go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080
curl http://localhost:8080/api/notes
```

## web/ plus pnpm migration design

### Recommended move plan

Move files in a way that preserves history where possible:

```bash
mkdir -p web
git mv client/index.html web/index.html
git mv client/public web/public
git mv client/src web/src
git mv package.json web/package.json
git mv pnpm-lock.yaml web/pnpm-lock.yaml
git mv vite.config.ts web/vite.config.ts
git mv tsconfig.json web/tsconfig.json
git mv tsconfig.node.json web/tsconfig.node.json
git mv components.json web/components.json
git mv .storybook web/.storybook
git mv patches web/patches
```

Then inspect import aliases and relative paths.

### Vite config target

The current `vite.config.ts` uses `client/` as root because it lives at repository root. Once it moves into `web/`, simplify paths:

```ts
const WEB_ROOT = import.meta.dirname;

export default defineConfig({
  plugins,
  resolve: {
    alias: {
      "@": path.resolve(WEB_ROOT, "src"),
      "@shared": path.resolve(WEB_ROOT, "../shared"),
      "@assets": path.resolve(WEB_ROOT, "assets"), // only if this folder exists
    },
  },
  envDir: WEB_ROOT,
  root: WEB_ROOT,
  build: {
    outDir: path.resolve(WEB_ROOT, "dist"),
    emptyOutDir: true,
  },
  server: {
    port: 3000,
    host: "127.0.0.1",
  },
});
```

The biggest decisions:

- Use `web/dist` rather than root `dist/public`. This is the standard Vite convention inside a standalone `web/` package.
- Keep `@shared` pointing back to root `shared/` only if those files remain used.
- Evaluate whether Manus-specific plugins should stay. They add dev-server logging and storage proxy behavior (`vite.config.ts:71-204`). If this project is no longer meant to run inside Manus, consider making them opt-in or removing them in a separate cleanup.

### package.json target

The `web/package.json` scripts should be direct and local:

```json
{
  "name": "retro-obsidian-publish-web",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite --host 127.0.0.1 --port 3000",
    "build": "vite build",
    "preview": "vite preview --host 127.0.0.1 --port 3000",
    "check": "tsc --noEmit",
    "format": "prettier --write .",
    "storybook": "storybook dev --port 6006",
    "build-storybook": "storybook build"
  },
  "packageManager": "pnpm@10.4.1+sha512..."
}
```

Remove `server/index.ts` and the Express static wrapper from the normal build path unless there is a current deployment requiring it. The requested direction is Dagger bundling, and the Go backend can eventually serve static files directly if a single binary is desired.

### Root Makefile compatibility wrapper

Update root Makefile so common commands still work:

```makefile
.PHONY: backend web web-install web-check web-dev storybook dev clean build-web

backend:
	cd backend && go build -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

web-install:
	pnpm --dir web install --frozen-lockfile

web-check:
	pnpm --dir web check

web:
	pnpm --dir web build

backend-dev:
	cd backend && go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080

web-dev:
	VITE_API_URL=http://127.0.0.1:8080 pnpm --dir web dev

build-web:
	cd backend && go run ./cmd/build-web

dev:
	$(MAKE) -j2 backend-dev web-dev
```

### pnpm validation checklist

```bash
pnpm --dir web install --frozen-lockfile
pnpm --dir web check
pnpm --dir web build
pnpm --dir web storybook -- --smoke-test || true  # if Storybook supports desired smoke mode
```

If `pnpm --dir web check` fails after moving files, inspect:

- `tsconfig.json` include paths.
- import aliases in `web/vite.config.ts`.
- Storybook config path references.
- `components.json` aliases.
- references to `client/`, `dist/public`, or root `vite.config.ts` in docs and scripts.

## Dagger bundling design

### Intern mental model

Dagger lets us describe a build as code. Instead of relying on whichever Node/pnpm version is installed on the developer's machine, the build command can spin up a controlled container, mount the source directory, mount a persistent cache for pnpm's package store, run deterministic commands, and export the resulting `web/dist` directory.

The Dagger command should be boring and reproducible:

```text
read web/package.json packageManager
  -> choose pnpm version
  -> create node container
  -> enable corepack
  -> prepare pnpm
  -> mount web source
  -> mount pnpm store cache
  -> pnpm install --frozen-lockfile
  -> pnpm build
  -> export dist
```

### Proposed command location

Because the current Go module is under `backend/`, put the Dagger command at:

```text
backend/cmd/build-web/main.go
```

It can refer to the repository root as `..` from `backend/` or detect it by walking upward until it finds both `backend/go.mod` and `web/package.json`.

### Dagger API sketch

```go
package main

const defaultBuilderImage = "node:22-bookworm-slim"
const defaultPNPMVersion = "10.4.1"

func main() {
    ctx := context.Background()
    repoRoot := findRepoRoot()

    if os.Getenv("BUILD_WEB_LOCAL") == "1" {
        runLocal(ctx, repoRoot)
        return
    }

    if err := runDagger(ctx, repoRoot); err != nil {
        log.Printf("dagger build failed: %v", err)
        log.Printf("falling back to local build; set BUILD_WEB_LOCAL=1 to force local")
        runLocal(ctx, repoRoot)
    }
}

func runDagger(ctx context.Context, repoRoot string) error {
    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
    if err != nil { return err }
    defer client.Close()

    webDir := client.Host().Directory(filepath.Join(repoRoot, "web"), dagger.HostDirectoryOpts{
        Exclude: []string{"node_modules", "dist", ".turbo", ".vite"},
    })

    store := client.CacheVolume("retro-obsidian-publish-pnpm-store")

    ctr := client.Container().
        From(defaultBuilderImage).
        WithDirectory("/src/web", webDir).
        WithWorkdir("/src/web").
        WithMountedCache("/pnpm/store", store).
        WithEnvVariable("PNPM_HOME", "/pnpm").
        WithEnvVariable("PATH", "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
        WithExec([]string{"corepack", "enable"}).
        WithExec([]string{"corepack", "prepare", "pnpm@" + defaultPNPMVersion, "--activate"}).
        WithExec([]string{"pnpm", "config", "set", "store-dir", "/pnpm/store"}).
        WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
        WithExec([]string{"pnpm", "build"})

    _, err = ctr.Directory("/src/web/dist").Export(ctx, filepath.Join(repoRoot, "web", "dist"))
    return err
}

func runLocal(ctx context.Context, repoRoot string) error {
    cmd := exec.CommandContext(ctx, "pnpm", "install", "--frozen-lockfile")
    cmd.Dir = filepath.Join(repoRoot, "web")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil { return err }

    cmd = exec.CommandContext(ctx, "pnpm", "build")
    cmd.Dir = filepath.Join(repoRoot, "web")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### Bundle output options

There are two valid next steps after Dagger produces `web/dist`:

1. **Static artifact only**: keep `web/dist` as the artifact. This is the smallest first pass and works with nginx, static hosting, or Docker.
2. **Go embedded frontend**: copy `web/dist` into `backend/internal/web/embed/public` and serve it from the Go backend. This enables a single binary, but it is more invasive.

Recommended for first pass: implement static artifact only. Add embedding in a follow-up ticket if the project needs a single binary.

### Dagger validation checklist

```bash
cd backend
go run ./cmd/build-web
BUILD_WEB_LOCAL=1 go run ./cmd/build-web
ls -la ../web/dist
```

If Dagger fails, verify:

- Docker or the Dagger engine is available.
- `web/package.json` has a `packageManager` field.
- `web/pnpm-lock.yaml` matches `web/package.json`.
- The Vite config no longer writes outside `web/dist` unless intentionally configured.

## devctl design

### Intern mental model

devctl is the local development orchestrator. It starts a plugin, the plugin emits a handshake, and then devctl asks it for configuration, validation, and launch plans. The plugin should not start long-running services itself. It should return service definitions, and devctl supervises those services.

The most important protocol rules:

- The first stdout line must be a JSON handshake.
- Every stdout line after that must be an NDJSON protocol frame.
- Human logs go to stderr, never stdout.
- Unknown operations should return `E_UNSUPPORTED`.

### Proposed `.devctl.yaml`

```yaml
plugins:
  - id: retro-obsidian-publish
    path: python3
    args:
      - ./plugins/retro-obsidian-publish.py
    priority: 10
```

Optional later:

```yaml
profile:
  active: development

profiles:
  development:
    display_name: Full stack development
    plugins: [retro-obsidian-publish]
  backend:
    display_name: Backend only
    plugins: [retro-obsidian-publish]
    env:
      RETRO_SERVICES: backend
```

### Plugin capabilities

Start with:

```json
{
  "ops": ["config.mutate", "validate.run", "launch.plan"],
  "commands": [
    { "name": "install-web", "help": "Install web dependencies with pnpm" },
    { "name": "build-web", "help": "Run the Dagger-backed web build" }
  ]
}
```

If dynamic commands are added, also implement `command.run`. For the first version, dynamic commands are optional. Keep the initial plugin as small as possible.

### Config keys

Use stable dotted keys:

```text
env.vault_dir = backend/vault-example
services.backend.port = 8080
services.backend.url = http://127.0.0.1:8080
services.web.port = 3000
services.web.url = http://127.0.0.1:3000
services.web.api_url = http://127.0.0.1:8080
paths.backend = backend
paths.web = web
```

These values should appear in `devctl plan` output through `config.mutate`.

### Plugin pseudocode

```python
#!/usr/bin/env python3
import json, os, shutil, sys
from pathlib import Path


def emit(obj):
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def log(msg):
    sys.stderr.write(msg + "\n")
    sys.stderr.flush()


emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "retro-obsidian-publish",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
})


def repo_root(ctx):
    return Path(ctx.get("repo_root") or os.getcwd()).resolve()


def handle_config(rid, ctx, inp):
    root = repo_root(ctx)
    vault = os.environ.get("VAULT_DIR", "backend/vault-example")
    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"config_patch": {"set": {
            "env.vault_dir": vault,
            "services.backend.port": 8080,
            "services.backend.url": "http://127.0.0.1:8080",
            "services.web.port": 3000,
            "services.web.url": "http://127.0.0.1:3000",
            "services.web.api_url": "http://127.0.0.1:8080",
            "paths.backend": "backend",
            "paths.web": "web",
        }, "unset": []}}
    })


def handle_validate(rid, ctx, inp):
    root = repo_root(ctx)
    errors = []
    warnings = []

    for exe in ["go", "node", "pnpm"]:
        if shutil.which(exe) is None:
            errors.append({"code": "E_MISSING_TOOL", "message": f"{exe} not found on PATH"})

    if not (root / "backend" / "go.mod").exists():
        errors.append({"code": "E_MISSING_BACKEND", "message": "backend/go.mod not found"})
    if not (root / "web" / "package.json").exists():
        errors.append({"code": "E_MISSING_WEB", "message": "web/package.json not found; run the web/ migration first"})
    if not (root / "web" / "node_modules").exists():
        warnings.append({"code": "W_WEB_DEPS", "message": "web/node_modules missing; run pnpm --dir web install --frozen-lockfile"})
    if not (root / "backend" / "vault-example").exists():
        warnings.append({"code": "W_NO_EXAMPLE_VAULT", "message": "backend/vault-example is missing"})

    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"valid": len(errors) == 0, "errors": errors, "warnings": warnings},
    })


def handle_launch(rid, ctx, inp):
    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"services": [
            {
                "name": "backend",
                "cwd": "backend",
                "command": ["go", "run", "./cmd/retro-obsidian-publish", "serve", "--vault", "./vault-example", "--port", "8080"],
                "env": {},
                "health": {"type": "http", "url": "http://127.0.0.1:8080/api/notes", "timeout_ms": 30000},
            },
            {
                "name": "web",
                "cwd": "web",
                "command": ["pnpm", "dev", "--host", "127.0.0.1", "--port", "3000"],
                "env": {"VITE_API_URL": "http://127.0.0.1:8080"},
                "health": {"type": "http", "url": "http://127.0.0.1:3000", "timeout_ms": 30000},
            },
        ]}}
    })


for line in sys.stdin:
    if not line.strip():
        continue
    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")
    ctx = req.get("ctx", {}) or {}
    inp = req.get("input", {}) or {}
    try:
        if op == "config.mutate":
            handle_config(rid, ctx, inp)
        elif op == "validate.run":
            handle_validate(rid, ctx, inp)
        elif op == "launch.plan":
            handle_launch(rid, ctx, inp)
        else:
            emit({"type": "response", "request_id": rid, "ok": False,
                  "error": {"code": "E_UNSUPPORTED", "message": f"unsupported op: {op}"}})
    except Exception as e:
        emit({"type": "response", "request_id": rid, "ok": False,
              "error": {"code": "E_PLUGIN", "message": str(e)}})
```

### devctl validation checklist

```bash
chmod +x plugins/retro-obsidian-publish.py
devctl plugins list
devctl plan
devctl up --force
devctl status --tail-lines 10
devctl logs --service backend --stderr
devctl logs --service web --stderr
curl http://127.0.0.1:8080/api/notes
curl http://127.0.0.1:3000
devctl down
```

## Phased implementation plan

### Phase 0: Baseline and branch hygiene

Goal: capture current behavior before changing file layout.

Steps:

1. Create a Git branch.
2. Run and record:
   - `cd backend && go test ./...`
   - `pnpm install --frozen-lockfile`
   - `pnpm check`
   - `pnpm build`
3. Start the current backend and frontend once if dependencies install cleanly.
4. Confirm API smoke request:
   - `curl http://localhost:8080/api/notes`

Current observed baseline:

- `cd backend && go test ./...` passes.
- `pnpm check` fails before install with `sh: 1: tsc: not found` and pnpm warning that `node_modules` is missing.

### Phase 1: Move frontend into web/

Goal: all frontend tooling and source are under `web/`.

Steps:

1. Use `git mv` commands listed in the web migration section.
2. Update `web/vite.config.ts` aliases and `root`/`outDir`.
3. Update Storybook config paths if needed.
4. Update `web/components.json` aliases if it references root/client paths.
5. Update root Makefile to call `pnpm --dir web`.
6. Update README setup commands.
7. Run:
   - `pnpm --dir web install --frozen-lockfile`
   - `pnpm --dir web check`
   - `pnpm --dir web build`

Review focus:

- No remaining script should assume root `package.json` unless intentionally kept as a wrapper.
- No Vite output should write to root `dist/public` unless intentionally retained.
- `VITE_API_URL` should still control backend mode in `web/src/store/vaultApi.ts`.

### Phase 2: Refactor backend startup into internal/server

Goal: make the runtime callable independently of CLI parsing.

Steps:

1. Create `backend/internal/server/server.go`.
2. Move all runtime setup from `backend/cmd/server/main.go` into `server.Run(ctx, Config)`.
3. Keep a temporary compatibility entrypoint if desired.
4. Add validation for empty vault and invalid port.
5. Run `cd backend && go test ./...`.

Review focus:

- Behavior of `VAULT_DIR` fallback should remain visible in CLI layer, not hidden in runtime package unless deliberately documented.
- The server should handle context cancellation if possible, because devctl sends process termination and graceful shutdown helps reduce port conflicts.

### Phase 3: Add Glazed CLI

Goal: replace direct `flag` usage with Glazed-backed commands.

Steps:

1. Add Glazed dependencies to `backend/go.mod`.
2. Create `backend/cmd/retro-obsidian-publish/main.go`.
3. Create command wiring for root and `serve`.
4. Define `ServeSettings` with `glazed` struct tags for `vault` and `port`.
5. Decode settings with `vals.DecodeSectionInto(schema.DefaultSlug, settings)`.
6. Run:
   - `go run ./cmd/retro-obsidian-publish help`
   - `go run ./cmd/retro-obsidian-publish serve --help`
   - `go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080`

Review focus:

- No remaining direct `flag.String` usage should exist in the active backend command.
- Help output should show Glazed command settings such as schema printing flags.
- Logging should initialize once at root.

### Phase 4: Add Dagger build-web command

Goal: reproducible web build with pnpm cache volume and local fallback.

Steps:

1. Add Dagger dependency to `backend/go.mod`.
2. Create `backend/cmd/build-web/main.go`.
3. Implement repo root detection.
4. Implement Dagger path with `node:22` image, corepack, pinned pnpm, cache volume, install, build, export.
5. Implement `BUILD_WEB_LOCAL=1` fallback.
6. Update Makefile `build-web` target.
7. Run:
   - `cd backend && go run ./cmd/build-web`
   - `cd backend && BUILD_WEB_LOCAL=1 go run ./cmd/build-web`

Review focus:

- The command must not require root `package.json`; it should use `web/package.json`.
- Cache volume name should be stable and project-specific.
- Build output should be deterministic and documented.

### Phase 5: Add devctl plugin and config

Goal: one-command full-stack local environment.

Steps:

1. Add `.devctl.yaml`.
2. Add `plugins/retro-obsidian-publish.py`.
3. Implement handshake, `config.mutate`, `validate.run`, and `launch.plan`.
4. Make the plugin executable.
5. Run:
   - `devctl plugins list`
   - `devctl plan`
   - `devctl up --force`
   - `devctl status --tail-lines 10`
   - `curl http://127.0.0.1:8080/api/notes`
   - `curl http://127.0.0.1:3000`
   - `devctl down`

Review focus:

- The plugin must never print logs to stdout.
- Missing tools or dependencies must produce actionable validation messages.
- devctl, not the plugin, must supervise backend and web processes.

### Phase 6: Documentation and cleanup

Goal: make the repository understandable after migration.

Steps:

1. Update README architecture tree and Getting Started.
2. Document devctl workflow.
3. Document Dagger build command and local fallback.
4. Update Dockerfiles/Compose or mark them secondary.
5. Remove stale files such as root `server/index.ts` if no deployment still uses them.
6. Re-run all checks.

## Testing and validation strategy

### Required commands after full implementation

```bash
git status --short

cd backend
go test ./...
go run ./cmd/retro-obsidian-publish serve --help
go run ./cmd/build-web
BUILD_WEB_LOCAL=1 go run ./cmd/build-web

cd ..
pnpm --dir web install --frozen-lockfile
pnpm --dir web check
pnpm --dir web build

devctl plugins list
devctl plan
devctl up --force
devctl status --tail-lines 10
curl http://127.0.0.1:8080/api/notes
curl http://127.0.0.1:3000
devctl down
```

### API smoke tests

```bash
curl -fsS http://127.0.0.1:8080/api/notes | jq 'length'
curl -fsS http://127.0.0.1:8080/api/tree | jq '.name'
curl -fsS 'http://127.0.0.1:8080/api/search?q=stoicism' | jq 'length'
curl -fsS http://127.0.0.1:8080/api/tags | jq 'length'
curl -fsS http://127.0.0.1:8080/api/graph | jq '{nodes: (.nodes|length), edges: (.edges|length)}'
```

### Manual browser smoke test

1. Open `http://127.0.0.1:3000`.
2. Confirm the Index note renders.
3. Click an internal wiki link.
4. Run a search.
5. Open the graph view if exposed in the UI.
6. Edit a note under `backend/vault-example` and check whether backend logs show reload. Remember: search index refresh may not currently update when the watcher reloads notes.

## Risks, alternatives, and open questions

### Risks

- **Go version mismatch**: `backend/go.mod` declares Go 1.25 while the Dockerfile uses Go 1.23. This can break container builds.
- **Watcher/search inconsistency**: `watcher` reloads the vault map but does not update the Bleve index. Live search may become stale after file changes.
- **Frontend move churn**: Moving root web files into `web/` can break Storybook, aliases, patches, and TypeScript includes if not done carefully.
- **Manus-specific Vite plugins**: `vite.config.ts` contains Manus debug/storage proxy plugins. These may be unwanted in a clean open-source/local developer setup.
- **Glazed long-running command behavior**: `serve` is not a typical row-producing command. Keep output expectations simple and document that Glazed is used for schema/help/settings, not structured server output.
- **Dagger availability**: Dagger builds may fail on machines without Docker/Dagger support. The local fallback is important.

### Alternatives considered

1. **Keep root package.json and only move `client/` to `web/`**: rejected because the user explicitly asked to move the web app into `web/`, and colocating package metadata reduces confusion.
2. **Move Go module to repository root immediately**: deferred because it would require changing import paths and increases first-pass risk.
3. **Use Docker Compose instead of Dagger**: rejected as the primary build path because the request specifically asks to bundle with Dagger. Docker Compose can remain as optional runtime documentation.
4. **Write the devctl plugin in Bash**: possible, but Python is safer for JSON construction and path validation.
5. **Make devctl run `make dev`**: rejected because devctl should supervise individual services and expose logs/status per service.

### Open questions

- Should the final production artifact be a single Go binary embedding `web/dist`, or separate backend and frontend artifacts?
- Should `server/index.ts` be removed completely, or is it used by an external hosting environment?
- Should the Manus-specific Vite plugins be removed, kept, or gated behind an environment variable?
- Should the backend router remain `gorilla/mux`, or should the project later move to Go 1.22 `http.ServeMux` patterns?
- Should file watcher events update the search index in this setup ticket or a follow-up correctness ticket?

## File references

- `backend/cmd/server/main.go`: current standard-library flag parsing and server startup.
- `backend/internal/api/api.go`: REST route definitions and JSON payload handlers.
- `backend/internal/vault/vault.go`: vault scanning, note indexing, backlinks, file tree.
- `backend/internal/parser/parser.go`: markdown/frontmatter/wiki-link parsing.
- `backend/internal/search/search.go`: Bleve in-memory and persistent index support.
- `backend/internal/watcher/watcher.go`: fsnotify watcher and note reload/remove logic.
- `client/src/store/vaultApi.ts`: frontend backend/static API mode switch and RTK Query endpoints.
- `client/src/App.tsx`: Wouter route tree.
- `vite.config.ts`: current root Vite config and client root/output paths.
- `package.json`: current pnpm scripts, dependencies, package manager, patches.
- `Makefile`: current backend/frontend/dev commands.
- `docker-compose.yml`: current backend/frontend Docker Compose services.
- `Dockerfile.frontend`: current nginx frontend image build.
- `backend/Dockerfile`: current backend image build and Go image mismatch.

## Quick intern checklist

If you are the intern starting this implementation, do this in order:

1. Read this whole document once.
2. Run the baseline commands and save failures.
3. Move frontend files into `web/` and make `pnpm --dir web check` pass.
4. Extract backend runtime into `internal/server`.
5. Add the Glazed `serve` command.
6. Add Dagger `cmd/build-web`.
7. Add devctl plugin and `.devctl.yaml`.
8. Update README and Makefile.
9. Run the full validation suite.
10. Ask for review focusing on file moves, CLI behavior, and devctl process supervision.
