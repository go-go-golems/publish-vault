.PHONY: all backend web frontend storybook dev clean build-web

# Build everything
all: backend web

# Build Go backend CLI
backend:
	cd backend && go build -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

# Backwards-compatible frontend alias
frontend: web

# Build frontend
web:
	pnpm --dir web build

# Build frontend via the Go/Dagger build verb
build-web:
	cd backend && go run ./cmd/retro-obsidian-publish build web

# Start backend with example vault (dev)
backend-dev:
	cd backend && go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080

# Start frontend dev server
frontend-dev:
	VITE_API_URL=http://127.0.0.1:8080 pnpm --dir web dev

# Start Storybook
storybook:
	pnpm --dir web storybook

# Run both backend and frontend in parallel (requires GNU make)
dev:
	$(MAKE) -j2 backend-dev frontend-dev

# Clean build artifacts
clean:
	rm -rf web/dist backend/bin backend/internal/web/embed/public
	cd backend && go clean
