# ssr.Dockerfile — Node.js SSR sidecar for Retro Obsidian Publish.
#
# Builds the web frontend and the SSR bundle, then runs server.mjs
# as a lightweight Express server that renders React on the server.
#
# This container runs as a sidecar alongside the Go server in the same pod.
# The Go server proxies page requests to this sidecar's port (8089).

FROM node:22-alpine

WORKDIR /app/web

# Enable pnpm
RUN corepack enable

# Install dependencies first (layer caching)
COPY web/package.json web/pnpm-lock.yaml ./
COPY web/patches ./patches
RUN pnpm install --frozen-lockfile

# Copy source and build both client + SSR bundles.
# The SSR bundle externalizes React/runtime dependencies, so this image must
# keep production node_modules available for server.mjs at runtime. Prune only
# build/dev dependencies after dist/ has been produced.
COPY web ./
RUN pnpm build:all && pnpm prune --prod

# Environment (overridable at runtime)
ENV SSR_PORT=8089
ENV API_BASE=http://localhost:8080
ENV BASE_URL=http://localhost:8080

EXPOSE 8089

# Run the SSR sidecar
CMD ["node", "server.mjs"]
