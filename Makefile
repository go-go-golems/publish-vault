.PHONY: all backend web frontend storybook dev clean build-web lint lintmax fmt-check docker-lint gosec govulncheck golangci-lint-install glazed-lint-build glazed-lint logcopter-generate logcopter-check test web-check build ci-check devctl-example devctl-parc tag-major tag-minor tag-patch bump-go-go-golems

GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_VERSION ?= latest
GLAZED_LINT_FLAGS ?=

# Build everything
all: backend web

# Build Go CLI
backend:
	GOWORK=off go build -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

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
	GOWORK=off go run ./cmd/retro-obsidian-publish build web

# Run Go and plugin tests
test:
	GOWORK=off go test ./...
	python3 -m unittest plugins/test_retro_plugin.py

golangci-lint-install:
	@mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	@GOBIN=$(dir $(GOLANGCI_LINT_BIN)) GOWORK=off go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Run the pinned Go linter and glazed-lint from the root module using the root config.
lint: golangci-lint-install glazed-lint
	GOWORK=off $(GOLANGCI_LINT_BIN) run -c .golangci.yml -v

lintmax: golangci-lint-install glazed-lint
	GOWORK=off $(GOLANGCI_LINT_BIN) run -c .golangci.yml -v --max-same-issues=100

fmt-check:
	GOWORK=off golangci-lint fmt --diff

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -c .golangci.yml -v

gosec:
	GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	GOWORK=off gosec -exclude-generated -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history ./...

govulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	GOWORK=off govulncheck ./...

glazed-lint-build:
	@if [ -n "$(GLAZED_VERSION)" ] && [ "$(GLAZED_VERSION)" != "(devel)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION) || { \
			echo "Falling back to $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION)"; \
			GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION); \
		}; \
	else \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION); \
	fi

glazed-lint: glazed-lint-build
	GOWORK=off $(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) ./...

# logcopter package loggers. Invoked directly (not via go generate ./...)
# because internal/web/generate.go triggers the Dagger frontend build.
LOGCOPTER_FLAGS ?= -include-main -var zlog -area-prefix go-go-golems.publish-vault -strip-prefix retro-obsidian-publish
LOGCOPTER_PACKAGES ?= ./cmd/... ./internal/...

logcopter-generate:
	GOWORK=off go tool logcopter-gen $(LOGCOPTER_FLAGS) $(LOGCOPTER_PACKAGES)

logcopter-check:
	GOWORK=off go tool logcopter-gen $(LOGCOPTER_FLAGS) -check $(LOGCOPTER_PACKAGES)

build: backend web

ci-check: fmt-check lint logcopter-check test gosec govulncheck web-check web

# Start backend with example vault (dev)
backend-dev:
	GOWORK=off go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080

# Start frontend dev server
frontend-dev:
	VITE_API_URL=http://127.0.0.1:8080 pnpm --dir web dev

# Start Storybook
storybook:
	pnpm --dir web storybook

# Run both backend and frontend in parallel (requires GNU make)
dev:
	$(MAKE) -j2 backend-dev frontend-dev

# Start devctl with the example vault profile (default)
devctl-example:
	devctl up --profile example

# Start devctl with the go-go-parc vault profile
devctl-parc:
	devctl up --profile go-go-parc

# Validate the svu output before tagging: a bare `git tag` (empty argument)
# would just list tags and exit 0, silently skipping the release tag.
tag-major:
	@tag="$$(svu major)" && test -n "$$tag" && git tag "$$tag" && echo "Tagged $$tag"

tag-minor:
	@tag="$$(svu minor)" && test -n "$$tag" && git tag "$$tag" && echo "Tagged $$tag"

tag-patch:
	@tag="$$(svu patch)" && test -n "$$tag" && git tag "$$tag" && echo "Tagged $$tag"

bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest" || exit 1; done; \
	fi

# Clean build artifacts
clean:
	rm -rf web/dist bin internal/web/embed/public
	go clean
