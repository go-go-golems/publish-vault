.PHONY: all backend web frontend storybook dev clean build-web lint lintmax docker-lint gosec govulncheck test web-check build ci-check

# Build everything
all: backend web

# Build Go backend CLI
backend:
	cd backend && GOWORK=off go build -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

# Backwards-compatible frontend alias
frontend: web

# Build frontend
web:
	pnpm --dir web build

# Type-check frontend
web-check:
	pnpm --dir web check

# Build frontend via the Go/Dagger build verb
build-web:
	cd backend && GOWORK=off go run ./cmd/retro-obsidian-publish build web

# Run backend tests
test:
	cd backend && GOWORK=off go test ./...
	python3 -m unittest plugins/test_retro_plugin.py

# Run the standard Go linter from the backend module using the root config.
lint:
	cd backend && GOWORK=off golangci-lint run -c ../.golangci.yml -v

lintmax:
	cd backend && GOWORK=off golangci-lint run -c ../.golangci.yml -v --max-same-issues=100

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app/backend golangci/golangci-lint:latest golangci-lint run -c ../.golangci.yml -v

gosec:
	cd backend && GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	cd backend && GOWORK=off gosec -exclude-generated -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history ./...

govulncheck:
	cd backend && GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	cd backend && GOWORK=off govulncheck ./...

build: backend web

ci-check: lint test gosec govulncheck web-check web

# Start backend with example vault (dev)
backend-dev:
	cd backend && GOWORK=off go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080

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
