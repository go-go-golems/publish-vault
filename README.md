# Retro Obsidian Publish

A retro macOS 1–inspired Obsidian vault publishing system. It serves an Obsidian vault as a web app with cross-linking, full-text search, backlinks, and a link graph — all with a monochrome Chicago-aesthetic UI.

---

## Architecture

```text
retro-obsidian-publish/
├── backend/                                   # Go module and single-binary app
│   ├── cmd/retro-obsidian-publish/            # Glazed/Cobra CLI
│   │   ├── main.go
│   │   └── commands/
│   │       ├── root.go
│   │       ├── serve/serve.go                 # serve verb
│   │       └── build/web.go                   # build web verb
│   ├── internal/
│   │   ├── api/api.go                         # REST API handlers
│   │   ├── parser/parser.go                   # Markdown/frontmatter/wiki-link parser
│   │   ├── search/search.go                   # Bleve full-text search index
│   │   ├── server/server.go                   # API + SPA HTTP runtime
│   │   ├── vault/vault.go                     # Vault index, backlinks, tree
│   │   ├── watcher/watcher.go                 # fsnotify hot reload
│   │   └── web/                               # embedded web assets + SPA handler
│   └── vault-example/                         # Sample vault for testing
│
├── web/                                       # React/Vite frontend package
│   ├── package.json
│   ├── pnpm-lock.yaml
│   ├── vite.config.ts
│   ├── index.html
│   ├── public/
│   └── src/
│       ├── components/                        # Atomic Design UI components
│       ├── store/vaultApi.ts                  # RTK Query API/static mode slice
│       ├── lib/wikiLinks.ts
│       └── index.css                          # Retro macOS design tokens
│
├── plugins/retro-obsidian-publish.py          # devctl plugin
├── .devctl.yaml                               # devctl wiring
├── Makefile                                   # convenience wrappers
└── ideas.md                                   # design philosophy
```

The production target is a **single Go binary**. The `retro-obsidian-publish build web` verb builds `web/` and copies `web/dist` into `backend/internal/web/embed/public`; `go build -tags embed` then embeds those assets into the binary.

---

## Stack

| Layer | Technology |
|-------|------------|
| Backend language | Go |
| CLI framework | Glazed + Cobra |
| Markdown parsing | goldmark + goldmark-meta |
| Full-text search | Bleve v2 |
| File watching | fsnotify |
| HTTP router | gorilla/mux |
| Frontend framework | React 19 + Vite 7 |
| State management | Redux Toolkit + RTK Query |
| Styling | Tailwind CSS 4 |
| Package manager | pnpm |
| Build bundling | Dagger with local pnpm fallback |
| Local orchestration | devctl |

---

## Getting started

### 1. Install web dependencies

```bash
pnpm --dir web install --frozen-lockfile
```

### 2. Build the web bundle for embedding

Dagger-first, with automatic local fallback if the Dagger engine is unavailable:

```bash
cd backend
go run ./cmd/retro-obsidian-publish build web
```

Force local pnpm mode:

```bash
cd backend
BUILD_WEB_LOCAL=1 go run ./cmd/retro-obsidian-publish build web --local
```

### 3. Build the single binary

```bash
cd backend
go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish
```

### 4. Run the app

```bash
cd backend
./bin/retro-obsidian-publish serve --vault ./vault-example --port 8080
```

Open:

- Web app: <http://127.0.0.1:8080>
- API: <http://127.0.0.1:8080/api/notes>

You can also use `VAULT_DIR`:

```bash
VAULT_DIR=/path/to/your/vault ./bin/retro-obsidian-publish serve --port 8080
```

---

## Development with devctl

The repo includes a devctl plugin that validates tools and launches both the backend API and Vite dev server.

```bash
devctl plugins list
devctl plan
devctl up --force
devctl status --tail-lines 10
devctl logs --service backend --stderr
devctl logs --service web --stderr
devctl down
```

Development URLs:

- Backend API: <http://127.0.0.1:8080/api/notes>
- Vite web app: <http://127.0.0.1:3000>

The devctl backend service runs the same single-binary CLI with `go run ./cmd/retro-obsidian-publish serve`, while Vite is used for fast frontend iteration.

---

## CLI reference

```bash
cd backend
go run ./cmd/retro-obsidian-publish help
go run ./cmd/retro-obsidian-publish serve --help
go run ./cmd/retro-obsidian-publish build web --help
```

Primary verbs:

- `serve` — scans the vault, builds an in-memory search index, exposes `/api/*`, watches markdown files, and serves the SPA when enabled.
- `build web` — builds `web/` and stages the result for Go embedding.

Examples:

```bash
cd backend
go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080
go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080 --serve-web=false
go run ./cmd/retro-obsidian-publish build web --local
```

---

## Web development commands

```bash
pnpm --dir web check
pnpm --dir web build
VITE_API_URL=http://127.0.0.1:8080 pnpm --dir web dev
pnpm --dir web storybook
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_DIR` | — | Vault directory, used when `--vault` is omitted |
| `VITE_API_URL` | unset | Frontend backend URL; when unset, the frontend uses static/demo data |
| `VITE_VAULT_NAME` | `My Vault` | Display name in the UI |
| `BUILD_WEB_LOCAL` | unset | Set to `1` to force local pnpm web builds |
| `WEB_BUILDER_IMAGE` | `node:22` | Optional Dagger builder image override |

---

## API reference

All endpoints return JSON. CORS is open (`*`) for development.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/notes` | List all notes |
| `GET` | `/api/notes/{slug}` | Full note |
| `GET` | `/api/tree` | Hierarchical file tree |
| `GET` | `/api/search?q={query}` | Full-text search |
| `GET` | `/api/tags` | All tags with counts |
| `GET` | `/api/graph` | Graph nodes and edges |

---

## Markdown features

The parser supports the core Obsidian Markdown dialect:

- frontmatter between `---` delimiters;
- wiki links such as `[[Note Title]]`, `[[Note Title|Alias]]`, and `[[Note#Heading]]`;
- embeds such as `![[Note]]` as embed placeholders;
- GFM tables, task lists, strikethrough, and footnotes;
- code blocks with highlight.js-compatible markup;
- automatically computed backlinks.

---

## Validation checklist

```bash
pnpm --dir web check
pnpm --dir web build
cd backend
go test ./...
BUILD_WEB_LOCAL=1 go run ./cmd/retro-obsidian-publish build web --local
go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish
./bin/retro-obsidian-publish serve --vault ./vault-example --port 8080
```

Then in another shell:

```bash
curl -fsS http://127.0.0.1:8080/api/notes
curl -fsS http://127.0.0.1:8080/
```

---

## Generated assets policy

`web/dist/` and `backend/internal/web/embed/public/*` are generated build outputs and are intentionally ignored. A production embedded binary should be built by running:

```bash
cd backend
BUILD_WEB_LOCAL=1 go run ./cmd/retro-obsidian-publish build web --local
go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish
```

This keeps frontend build artifacts out of code review while preserving reproducible single-binary builds.

## Known follow-up

The watcher now updates the Bleve search index for file reload/remove events. Remaining correctness work is mostly test expansion and release hardening.
