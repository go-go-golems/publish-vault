.PHONY: all backend frontend storybook dev clean

# Build everything
all: backend frontend

# Build Go backend
backend:
	cd backend && go build -o bin/server ./cmd/server/

# Build frontend
frontend:
	pnpm build

# Start backend with example vault (dev)
backend-dev:
	cd backend && go run ./cmd/server/ --vault ./vault-example --port 8080

# Start frontend dev server
frontend-dev:
	VITE_API_URL=http://localhost:8080 pnpm dev

# Start Storybook
storybook:
	pnpm storybook

# Run both backend and frontend in parallel (requires GNU make)
dev:
	$(MAKE) -j2 backend-dev frontend-dev

# Clean build artifacts
clean:
	rm -rf dist backend/bin
	cd backend && go clean
