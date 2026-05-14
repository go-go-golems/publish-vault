# Retro Obsidian Publish

Retro Obsidian Publish turns an Obsidian vault into a small self-hosted website. It reads Markdown files from a vault directory, builds an in-memory note index, resolves wiki links, computes backlinks, builds a search index, and serves both a JSON API and a retro monochrome React frontend from one Go process.

It is designed for people who want to publish a personal knowledge base without changing how they write notes. Your source of truth remains a normal folder of Markdown files. The application treats that folder as read-only content and derives the website from it.

---

## What you get

- **Single-binary publishing**: build one Go binary that serves the API and the web app.
- **Obsidian-style links**: supports `[[Note]]`, `[[Folder/Note]]`, `[[Note|Alias]]`, `[[Note#Heading]]`, and `![[Embeds]]`.
- **Backlinks**: every note gets a computed list of notes that link to it.
- **Full-text search**: notes are indexed with Bleve and queried through `/api/search`.
- **File tree navigation**: the sidebar mirrors the vault folder hierarchy and opens to the active note.
- **Markdown rendering**: supports frontmatter, tables, task lists, footnotes, code blocks, callouts, and heading anchors.
- **Client-side enhancements**: syntax highlighting, Mermaid diagrams, copy buttons on code blocks, collapsible callouts, and inline embeds.
- **Live local development**: run the Go backend and Vite frontend separately while editing UI code.
- **Content update hook**: optional reload endpoint for setups where a Git checkout is updated by another process.

---

## Repository layout

```text
retro-obsidian-publish/
├── backend/                         # Go module and server binary
│   ├── cmd/retro-obsidian-publish/   # CLI entrypoint and commands
│   ├── internal/api/                 # JSON API handlers
│   ├── internal/parser/              # Markdown, frontmatter, wiki-link parsing
│   ├── internal/search/              # Bleve search index
│   ├── internal/server/              # HTTP server, health, reload runtime
│   ├── internal/vault/               # Vault loader, slugs, tree, backlinks
│   ├── internal/watcher/             # Local filesystem watcher
│   ├── internal/web/                 # SPA static-file handler and embed support
│   └── vault-example/                # Tiny example vault
├── web/                              # React/Vite frontend
│   ├── src/components/               # UI components
│   ├── src/store/                    # RTK Query API layer and UI state
│   ├── src/vault/                    # Static demo vault support
│   └── src/index.css                 # Retro design system and prose styles
├── plugins/retro-obsidian-publish.py # Optional devctl plugin
├── .devctl.yaml                      # Optional devctl local orchestration
├── ideas.md                          # Design philosophy and product notes
└── README.md
```

Useful starting points:

- [`ideas.md`](./ideas.md) — background, design philosophy, and product ideas.
- [`backend/vault-example/`](./backend/vault-example/) — a tiny vault you can serve immediately.
- [`web/src/components/`](./web/src/components/) — React UI implementation.
- [`backend/internal/parser/parser.go`](./backend/internal/parser/parser.go) — Markdown and Obsidian syntax handling.
- [`backend/internal/vault/vault.go`](./backend/internal/vault/vault.go) — note loading, slugs, backlinks, and file tree construction.
- [`backend/internal/server/server.go`](./backend/internal/server/server.go) — HTTP server, health endpoint, and reload endpoint.

---

## Quick start: serve the example vault

You need:

- Go 1.25 or newer;
- Node.js 22 or newer;
- pnpm through Corepack;
- optional: Dagger, if you want containerized web builds instead of local pnpm builds.

From the repository root:

```bash
corepack enable
pnpm --dir web install --frozen-lockfile

cd backend
go run ./cmd/retro-obsidian-publish serve \
  --vault ./vault-example \
  --port 8080
```

Open:

```text
http://127.0.0.1:8080
```

The development build serves the frontend from `web/dist`. If you have not built the web app yet and the page reports that the web bundle is missing, run:

```bash
cd backend
go run ./cmd/retro-obsidian-publish build web --local
```

Then start the server again.

---

## Serve your own vault

Point `--vault` at any Obsidian vault directory:

```bash
cd backend
go run ./cmd/retro-obsidian-publish serve \
  --vault /path/to/your/obsidian-vault \
  --port 8080
```

You can also use `VAULT_DIR`:

```bash
cd backend
VAULT_DIR=/path/to/your/obsidian-vault \
  go run ./cmd/retro-obsidian-publish serve --port 8080
```

The server scans every Markdown file below the vault root, skipping hidden directories. It does not write to your vault. Local file watching is enabled by default, so edits to Markdown files are picked up while the server is running.

---

## Build a single production binary

The production path builds the React app, copies its static assets into the Go embed directory, and then compiles a Go binary with the `embed` build tag.

```bash
# 1. Build the web app and stage it for Go embedding.
cd backend
go run ./cmd/retro-obsidian-publish build web --local

# 2. Build the single binary.
go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

# 3. Run it against your vault.
./bin/retro-obsidian-publish serve \
  --vault /path/to/your/obsidian-vault \
  --port 8080
```

Open:

```text
http://127.0.0.1:8080
```

The generated frontend assets are intentionally not meant to be edited by hand. Rebuild them from `web/` whenever the frontend changes.

---

## Build and run with Docker

The repository includes a multi-stage Dockerfile at [`backend/Dockerfile`](./backend/Dockerfile). Build it from the repository root so Docker can see both `backend/` and `web/`:

```bash
docker build \
  -f backend/Dockerfile \
  -t retro-obsidian-publish:local \
  .
```

Run it with your vault mounted read-only:

```bash
docker run --rm \
  -p 8080:8080 \
  -v /path/to/your/obsidian-vault:/vault:ro \
  retro-obsidian-publish:local \
  serve --vault /vault --port 8080 --serve-web
```

Open:

```text
http://127.0.0.1:8080
```

For a small server or VPS, this Docker mode is the simplest deployment model: build the image, copy it to the host or push it to a registry, mount the vault directory, and run the container behind your preferred reverse proxy.

---

## Development mode

For frontend work, run the backend API and Vite separately.

Terminal 1:

```bash
cd backend
go run ./cmd/retro-obsidian-publish serve \
  --vault ./vault-example \
  --port 8080 \
  --serve-web=false
```

Terminal 2:

```bash
VITE_API_URL=http://127.0.0.1:8080 \
  pnpm --dir web dev
```

Open:

```text
http://127.0.0.1:3000
```

The Vite server gives fast frontend reloads while the Go backend serves real vault data from `/api/*`.

### Optional: devctl

If you use `devctl`, this repository includes `.devctl.yaml` and a plugin under `plugins/`:

```bash
devctl plugins list
devctl plan
devctl up --force
devctl status
devctl logs --service backend
devctl logs --service web
devctl down
```

This is optional. The plain Go and pnpm commands above are the canonical workflow.

---

## How the publishing pipeline works

The application has two phases: load time and request time.

At load time, the server builds a complete in-memory representation of the vault:

```text
Markdown files
  -> parser.Parse
  -> Note objects
  -> wiki-link suffix index
  -> backlinks
  -> rendered HTML with resolved links
  -> Bleve search index
```

At request time, handlers read from that prepared state:

```text
Browser
  -> React app
  -> /api/notes or /api/notes/{slug}
  -> current vault snapshot
  -> JSON response
  -> rendered note page
```

This keeps normal page loads simple. The expensive parsing and indexing work happens when the vault is loaded or reloaded, not every time a note is viewed.

---

## API reference

All API endpoints are served from the same process as the web app.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/healthz` | Health information, note count, configured vault root, resolved vault root. |
| `GET` | `/api/notes` | Lightweight list of all notes. |
| `GET` | `/api/notes/{slug}` | Full note with HTML, frontmatter, tags, wiki links, backlinks, and modification time. |
| `GET` | `/api/tree` | Folder tree for sidebar navigation. |
| `GET` | `/api/search?q={query}` | Full-text search results. |
| `GET` | `/api/tags` | Tag counts. |
| `POST` | `/api/admin/reload` | Optional administrative reload endpoint. Disabled unless configured. |

Example:

```bash
curl -fsS http://127.0.0.1:8080/api/healthz | jq
curl -fsS http://127.0.0.1:8080/api/notes | jq '.[0]'
curl -fsS 'http://127.0.0.1:8080/api/search?q=zettelkasten' | jq
```

---

## Supported Markdown and Obsidian features

The parser supports:

- YAML frontmatter between `---` delimiters;
- headings with generated IDs;
- GitHub-flavored Markdown tables;
- task lists;
- strikethrough;
- footnotes;
- fenced code blocks;
- Obsidian wiki links:
  - `[[Note]]`
  - `[[Folder/Note]]`
  - `[[Note|Alias]]`
  - `[[Note#Heading]]`
- Obsidian embeds:
  - `![[Note]]`
- callouts:
  - `> [!note]`
  - `> [!warning]`
  - `> [!summary]`
  - `> [!type]-` for collapsed callouts;
- computed backlinks;
- Mermaid diagrams in fenced `mermaid` code blocks;
- syntax highlighting for code blocks;
- copy buttons on code blocks;
- heading permalink anchors.

Some Obsidian-specific behavior is intentionally approximated. The goal is not to reimplement the full Obsidian application. The goal is to publish a readable, linkable, searchable website from the same Markdown source files.

---

## Wiki-link resolution

Obsidian links often use short paths. A note can contain:

```markdown
See [[Tribal/App-Auth]].
```

while the actual file may live at:

```text
Research/KB/Tribal/App-Auth.md
```

Retro Obsidian Publish builds a suffix-based index so short links can resolve to full vault slugs. For that file, the resolver can register forms such as:

```text
research/kb/tribal/app-auth
kb/tribal/app-auth
tribal/app-auth
app-auth
```

If two notes share the same short suffix, the first registered note wins. Use more specific paths in your wiki links when your vault has ambiguous names.

---

## Frontmatter

Frontmatter is included in the full note API and shown in the frontend metadata panel. Nested YAML structures are normalized so they can be served as JSON. For example:

```yaml
---
title: Example Note
tags:
  - publishing
  - obsidian
RelatedFiles:
  - Path: docs/example.md
    Note: Source document
---
```

The frontend receives `frontmatter` as a JSON object. Tags are also extracted into the top-level `tags` field used by search and tag navigation.

---

## Keeping a published vault up to date

For local use, leave file watching enabled. It is the default:

```bash
retro-obsidian-publish serve --vault /path/to/vault --port 8080
```

For server deployments where another process updates the vault directory, use explicit reloads instead:

```bash
RETRO_RELOAD_TOKEN=change-me \
  retro-obsidian-publish serve \
  --vault /srv/vault/current \
  --watch=false \
  --reload-token-env RETRO_RELOAD_TOKEN
```

Then, after updating the vault checkout, call:

```bash
curl -X POST \
  -H "Authorization: Bearer change-me" \
  http://127.0.0.1:8080/api/admin/reload
```

The reload endpoint builds a new vault and search index first. If parsing or indexing fails, the old state remains active.

### Optional Git workflow

A simple Git-based publishing workflow looks like this:

```text
1. Write notes locally in Obsidian.
2. Commit and push the vault repository.
3. On the server, pull the latest commit into the published checkout.
4. Call POST /api/admin/reload.
5. The site serves the new vault snapshot.
```

You can implement step 3 with a cron job, a webhook receiver, a small systemd timer, `git-sync`, or any other Git automation you prefer. The application does not require a particular deployment platform.

---

## Configuration

### Server flags

```bash
retro-obsidian-publish serve --help
```

Important flags:

| Flag | Default | Description |
|---|---:|---|
| `--vault` | from `VAULT_DIR` | Path to the vault directory. Required if `VAULT_DIR` is unset. |
| `--port` | `8080` | HTTP port. |
| `--serve-web` | `true` | Serve the bundled web app from the Go process. |
| `--watch` | `true` | Watch Markdown files and update local state as files change. |
| `--reload-token-env` | `RETRO_RELOAD_TOKEN` | Environment variable containing the reload bearer token. |
| `--reload-allow-loopback` | `false` | Allow unauthenticated reloads from loopback clients. Useful for same-host automation. |

### Environment variables

| Variable | Description |
|---|---|
| `VAULT_DIR` | Default vault path when `--vault` is omitted. |
| `RETRO_RELOAD_TOKEN` | Bearer token for `POST /api/admin/reload`, if reload token auth is enabled. |
| `BUILD_WEB_LOCAL=1` | Force `build web` to use local pnpm instead of Dagger. |
| `WEB_BUILDER_IMAGE` | Optional container image override for web builds. |
| `VITE_API_URL` | API URL for Vite development mode. Leave unset for same-origin production builds. |
| `VITE_VAULT_NAME` | Display name used by the frontend. |
| `VITE_STATIC_VAULT=true` | Build the frontend in static demo mode instead of using the live API. |

---

## Validation checklist

Run this before publishing a new build:

```bash
pnpm --dir web check
pnpm --dir web build

cd backend
go test ./...
go run ./cmd/retro-obsidian-publish build web --local
go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish
./bin/retro-obsidian-publish serve --vault ./vault-example --port 8080
```

In another shell:

```bash
curl -fsS http://127.0.0.1:8080/api/healthz | jq
curl -fsS http://127.0.0.1:8080/api/notes | jq 'length'
curl -fsS http://127.0.0.1:8080/ | head
```

---

## Troubleshooting

### `web bundle not found`

Build the frontend bundle:

```bash
cd backend
go run ./cmd/retro-obsidian-publish build web --local
```

### `--vault or VAULT_DIR is required`

Pass a vault path:

```bash
retro-obsidian-publish serve --vault /path/to/vault
```

or set:

```bash
export VAULT_DIR=/path/to/vault
```

### A note appears in the list but fails to render

Run the backend tests and check the server logs. Nested YAML frontmatter should be normalized before JSON encoding. If you find a case that still fails, reduce it to a small Markdown file and add it to the parser tests.

### Links point to the wrong note

Use a more specific wiki-link path. Short suffix links are convenient, but ambiguous note names can resolve to the first matching suffix.

### Search does not show a recent edit

If you run with `--watch=true`, check whether the file watcher logged an error. If you run with `--watch=false`, call the reload endpoint after updating the vault.

---

## Project status

Retro Obsidian Publish is usable, but it is still a young project. The current implementation favors a straightforward architecture that is easy to inspect and modify:

- one Go server process;
- one in-memory vault snapshot;
- one in-memory search index;
- one embedded React frontend;
- optional reload endpoint for content automation.

Good next improvements include configurable home-note selection, reload metrics, explicit ambiguity reports for wiki-link resolution, smaller frontend bundles through dynamic imports, and packaged release binaries.

---

## License

Add the license that matches how you intend to distribute the project.
