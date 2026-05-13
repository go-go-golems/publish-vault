FROM node:22-alpine AS web-builder
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
COPY web/patches ./patches
RUN pnpm install --frozen-lockfile
COPY web ./
ARG VITE_API_URL=
ARG VITE_VAULT_NAME=My Vault
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_VAULT_NAME=$VITE_VAULT_NAME
RUN pnpm build

FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache build-base
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend ./
COPY --from=web-builder /src/web/dist ./internal/web/embed/public
RUN CGO_ENABLED=1 go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=go-builder /src/backend/bin/retro-obsidian-publish ./retro-obsidian-publish
EXPOSE 8080
ENTRYPOINT ["./retro-obsidian-publish"]
CMD ["serve", "--port", "8080", "--serve-web"]
