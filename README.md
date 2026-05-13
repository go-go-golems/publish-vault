# Retro Obsidian Publish

A retro macOS 1–inspired Obsidian vault publishing system. Publish your Obsidian vault as a website with cross-linking, full-text search, backlinks, and a link graph — all with a monochrome Chicago-aesthetic UI.

---

## Architecture

```
retro-obsidian-publish/
├── backend/                    # Go backend
│   ├── cmd/server/main.go      # Entry point (--vault, --port flags)
│   ├── internal/
│   │   ├── parser/parser.go    # Markdown + frontmatter + wiki-link parser
│   │   ├── vault/vault.go      # Vault index, backlink graph, file tree
│   │   ├── search/search.go    # Bleve full-text search index
│   │   ├── watcher/watcher.go  # fsnotify file watcher (hot reload)
│   │   └── api/api.go          # REST API handlers
│   └── vault-example/          # Sample vault for testing
│
├── client/src/
│   ├── components/
│   │   ├── atoms/              # Button, Icon, Tag, Badge, Divider, Input, Checkbox, ScrollArea
│   │   ├── molecules/          # SearchBar, NoteCard, FileTreeItem, BreadcrumbBar, BacklinkItem, FrontmatterPanel
│   │   ├── organisms/          # Sidebar, NoteRenderer, BacklinksPanel, GraphView
│   │   └── pages/              # VaultLayout, NotePage, SearchPage
│   ├── store/
│   │   ├── vaultApi.ts         # RTK Query API slice
│   │   ├── uiSlice.ts          # UI state (sidebar, search, graph)
│   │   └── store.ts            # Redux store
│   ├── hooks/redux.ts          # Typed useAppDispatch / useAppSelector
│   ├── lib/wikiLinks.ts        # Frontend wiki-link resolver
│   └── index.css               # Retro macOS 1 design tokens + utilities
│
├── .storybook/                 # Storybook configuration
└── ideas.md                    # Design philosophy documentation
```

---

## Stack

| Layer | Technology |
|-------|-----------|
| Backend language | Go 1.23 |
| Markdown parsing | goldmark + goldmark-meta |
| Full-text search | Bleve v2 |
| File watching | fsnotify |
| HTTP router | gorilla/mux |
| CORS | rs/cors |
| Frontend framework | React 19 + Vite 7 |
| State management | Redux Toolkit + RTK Query |
| Styling | Tailwind CSS 4 |
| Component docs | Storybook 10 |
| Component library | Radix UI + shadcn/ui |
| Icons | Lucide React |

---

## Getting Started

### 1. Build the Go backend

```bash
cd backend
go build -o bin/server ./cmd/server/
```

### 2. Start the backend

```bash
# Point at your Obsidian vault directory
./backend/bin/server --vault /path/to/your/vault --port 8080

# Or use the example vault
./backend/bin/server --vault ./backend/vault-example --port 8080

# Or via environment variable
VAULT_DIR=/path/to/vault ./backend/bin/server
```

### 3. Start the frontend dev server

```bash
# Set the API URL (defaults to http://localhost:8080)
VITE_API_URL=http://localhost:8080 pnpm dev
```

The app will be available at `http://localhost:3000`.

### 4. Run Storybook

```bash
pnpm storybook
# Opens at http://localhost:6006
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `http://localhost:8080` | Go backend URL |
| `VITE_VAULT_NAME` | `My Vault` | Display name in the UI |
| `VAULT_DIR` | — | Vault directory (env alternative to `--vault`) |

---

## API Reference

All endpoints return JSON. CORS is open (`*`) for development.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/notes` | List all notes (slug, title, tags, excerpt, modTime) |
| `GET` | `/api/notes/{slug}` | Full note (html, frontmatter, wikiLinks, backlinks) |
| `GET` | `/api/tree` | Hierarchical file tree |
| `GET` | `/api/search?q={query}` | Full-text search |
| `GET` | `/api/tags` | All tags with counts |
| `GET` | `/api/graph` | Graph nodes and edges |

---

## Markdown Features

The parser supports the full Obsidian Markdown dialect:

- **Frontmatter** — YAML between `---` delimiters; `title`, `tags`, and arbitrary fields
- **Wiki links** — `[[Note Title]]`, `[[Note Title|Alias]]`, `[[Note#Heading]]`
- **Embeds** — `![[Note]]` (rendered as embed placeholder)
- **GFM** — tables, task lists, strikethrough, footnotes
- **Code blocks** — syntax highlighting via highlight.js
- **Backlinks** — automatically computed from wiki-link graph

---

## Component Architecture

Components follow the **Atomic Design** methodology:

```
atoms/       ← Smallest indivisible UI elements (Button, Tag, Icon…)
molecules/   ← Composed atoms with single responsibility (SearchBar, NoteCard…)
organisms/   ← Complex UI sections (Sidebar, NoteRenderer, GraphView…)
pages/       ← Full page layouts wired to data (VaultLayout, NotePage…)
```

Each component lives in its own folder with a `ComponentName.tsx` and `ComponentName.stories.tsx`.

---

## Design Philosophy

**Retro System 1 (Macintosh 1984)**

- Monochrome foundation: near-black ink on warm aged paper
- Colour accents only for interactive/functional elements: links = `#0000cc`, tags = `#005500`, destructive = `#cc0000`
- Zero border-radius, hard 1px borders, inset box-shadows
- System-UI/Chicago font stack — no web fonts, pixel-crisp rendering
- Instant state changes — no smooth colour transitions

---

## Development

```bash
# Type-check
pnpm check

# Build for production
pnpm build

# Format code
pnpm format
```

---

## Deployment

**Frontend**: Deploy the `dist/public/` directory to any static host (Netlify, Vercel, Cloudflare Pages, etc.). Set `VITE_API_URL` at build time.

**Backend**: Build the Go binary and run it on any Linux server. The binary is self-contained with no runtime dependencies.

```bash
# Production build
cd backend && go build -o bin/server ./cmd/server/

# Run with systemd, Docker, or any process manager
VAULT_DIR=/path/to/vault ./bin/server --port 8080
```
