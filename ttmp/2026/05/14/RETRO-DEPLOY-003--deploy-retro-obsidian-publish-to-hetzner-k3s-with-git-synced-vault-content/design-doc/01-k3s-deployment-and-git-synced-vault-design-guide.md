---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../2026-03-27--hetzner-k3s/gitops/applications/goja-essay.yaml
      Note: Existing Argo CD Application pattern to copy
    - Path: ../../../../../../../2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/deployment.yaml
      Note: Existing kustomize Deployment pattern to copy
    - Path: ../../../../../../../2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/ingress.yaml
      Note: Existing Traefik/cert-manager ingress pattern to copy
    - Path: backend/Dockerfile
      Note: Production container build and single-binary packaging evidence
    - Path: backend/internal/search/search.go
      Note: Bleve search index rebuild behavior
    - Path: backend/internal/server/server.go
      Note: Runtime startup
    - Path: backend/internal/vault/vault.go
      Note: Vault parsing
    - Path: backend/internal/watcher/watcher.go
      Note: Current fsnotify watcher and production sync limitations
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# K3s deployment and git-synced vault design guide

## Executive summary

`retro-obsidian-publish` should be deployed to the Hetzner k3s cluster as a normal Argo CD-managed application, while the Obsidian vault content should be synchronized at runtime from its own Git repository using a `git-sync` sidecar. Argo CD should manage Kubernetes objects and image versions. `git-sync` should manage Markdown content updates.

The recommended production shape is:

1. Build and publish a container image for `retro-obsidian-publish`.
2. Add an Argo CD Application in `~/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/retro-obsidian-publish.yaml`.
3. Add a kustomize app under `gitops/kustomize/retro-obsidian-publish/`.
4. Run the app container and a `kubernetes/git-sync` sidecar in the same Pod.
5. Mount a shared `emptyDir` volume at `/git`.
6. Configure `git-sync` to clone the vault repo and publish the current revision through its atomic symlink contract.
7. Configure the app to read the vault from a path below the symlink, for example `/git/root/current/vault` if the repo contains a `vault/` subdirectory.
8. Add a small authenticated admin reload endpoint to the app and configure `git-sync` `--webhook-url` to call it after every successful sync.
9. On reload, the app should fully reload the vault and rebuild the in-memory search index.

This design avoids baking private vault content into the application image, avoids representing many Markdown files as ConfigMaps, and keeps the deployed service aligned with Git as the source of truth.

## Problem statement and scope

The current local application serves a vault from a filesystem path. The deployment target is a k3s cluster managed by Argo CD in `~/code/wesen/2026-03-27--hetzner-k3s`. The vault is backed by Git and changes independently from application code. We need a production design that answers:

- How should the running web server receive updated vault content?
- How should the server rebuild its indexes after new content arrives?
- What existing Kubernetes/GitOps tools solve this problem?
- What code changes are needed before manifests are written?
- What should a new intern understand before implementing the deployment?

This document is a design and implementation guide. It does not yet implement the Kubernetes manifests or application reload endpoint.

## Definitions

- **Application GitOps repo**: `~/code/wesen/2026-03-27--hetzner-k3s`, the repository Argo CD watches for Kubernetes manifests.
- **Application code repo**: `~/code/wesen/2026-05-13--retro-obsidian-publish`, the repository containing the Go server and React frontend.
- **Vault repo**: the Git-backed Obsidian content repository to publish. In local testing this has been `/home/manuel/code/wesen/go-go-golems/go-go-parc`.
- **Content sync**: pulling updated Markdown files into a running Pod.
- **Index reload**: parsing Markdown, rebuilding wiki-link resolution/backlinks, and rebuilding the Bleve search index.
- **GitOps reconciliation**: Argo CD applying Kubernetes manifests from Git.

## Current-state architecture

### Application build and runtime

The application is already designed as a single Go binary with embedded frontend assets. The Dockerfile builds the web frontend in a Node stage, builds the Go binary in a Go stage, then copies only the binary into an Alpine runtime image.

Evidence:

- `backend/Dockerfile` lines 1-12 build the Vite frontend with pnpm.
- `backend/Dockerfile` lines 14-21 build the Go binary with `-tags embed`.
- `backend/Dockerfile` lines 23-29 create the runtime image and run `serve --port 8080 --serve-web`.

Relevant excerpt from `sources/05-retro-source-evidence.txt`:

```text
backend/Dockerfile:20 COPY --from=web-builder /src/web/dist ./internal/web/embed/public
backend/Dockerfile:21 RUN CGO_ENABLED=1 go build -tags embed -o bin/retro-obsidian-publish ./cmd/retro-obsidian-publish
backend/Dockerfile:29 CMD ["serve", "--port", "8080", "--serve-web"]
```

The server requires a vault directory. It loads the vault once on startup, builds the search index, starts a filesystem watcher, registers API handlers, and serves the embedded SPA.

Evidence:

- `backend/internal/server/server.go` lines 23-28 define `VaultDir`, `Port`, and `ServeWeb`.
- `server.go` lines 48-58 load the vault and build the search index.
- `server.go` lines 60-65 start the watcher.
- `server.go` lines 67-72 register API handlers and the SPA handler.

Runtime flow today:

```text
process start
  ├─ validate --vault path
  ├─ vault.New(absVault)
  │    ├─ filepath.Walk(root)
  │    ├─ parse every .md file
  │    ├─ build wiki-link suffix index
  │    ├─ build backlinks
  │    └─ rewrite rendered HTML links
  ├─ search.New(vault)
  │    └─ build in-memory Bleve index
  ├─ watcher.New(vault, search)
  │    └─ fsnotify .md changes and reload individual files
  └─ HTTP server on :8080
       ├─ /api/notes
       ├─ /api/notes/{slug}
       ├─ /api/tree
       ├─ /api/search
       ├─ /api/tags
       └─ /
```

### Vault loading and indexing

`backend/internal/vault/vault.go` is the core content model.

Important behavior:

- `Vault.New(rootDir)` stores the root and calls `LoadAll()`.
- `LoadAll()` walks the vault directory, parses `.md` files, builds a wiki-link suffix index, builds backlinks, and rewrites rendered HTML links.
- `ReloadNote(path)` reparses one file, then rebuilds the wiki-link index, backlinks, and HTML for all notes.
- `RemoveNote(path)` removes one note and rebuilds the link structures.

This means the application already knows how to index content, but it currently has no explicit production reload API for a sidecar to call.

### Watcher behavior

`backend/internal/watcher/watcher.go` uses fsnotify and debounces `.md` file events.

Evidence:

- Lines 52-60 walk the vault root and add watches for directories.
- Lines 87-94 collect events only for files ending in `.md`.
- Lines 102-109 debounce and apply pending events.
- Lines 114-137 reload or delete individual notes and update search.

This watcher is useful for local development and direct file mutation. It is not sufficient as the only production update mechanism when `git-sync` publishes an entirely new checkout through an atomic symlink.

Why: `git-sync` deliberately avoids in-place checkout mutation. It creates a new worktree and flips a symlink. The current watcher watches directories discovered during startup; it will not automatically move its watches to a newly published worktree.

### Existing k3s GitOps patterns

The k3s repo uses Argo CD Application objects in `gitops/applications/` and app-specific kustomize directories under `gitops/kustomize/`.

Evidence from `sources/06-k3s-source-evidence.txt`:

```yaml
# gitops/applications/goja-essay.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: goja-essay
  namespace: argocd
spec:
  destination:
    server: https://kubernetes.default.svc
    namespace: goja-essay
  source:
    repoURL: https://github.com/wesen/2026-03-27--hetzner-k3s.git
    targetRevision: main
    path: gitops/kustomize/goja-essay
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

The `goja-essay` deployment shows the basic pattern for a single web application:

- `Deployment` with one container on port 8080.
- `Service` exposing port 80 to target port `http`.
- `Ingress` with Traefik and cert-manager annotations.
- Optional PVC volume mounted at `/data`.

`retro-obsidian-publish` should follow the same overall structure but add a git-sync sidecar and a shared content volume.

## External tools and references

### Kubernetes git-sync

`kubernetes/git-sync` is the best fit for runtime vault content synchronization.

Key facts from `sources/01-kubernetes-git-sync.md`:

- It is explicitly described as a sidecar that pulls a remote Git repository into a local directory so an application can consume it.
- It can pull once or periodically from a branch, tag, or specific hash.
- It publishes each sync through a worktree and named symlink, ensuring consumers do not see partially constructed checkouts.
- It supports HTTP(S), SSH, webhook notification, and exec hooks.
- The webhook is called after the symlink is updated.

Relevant source lines:

```text
01-kubernetes-git-sync.md:7 git-sync is a simple command that pulls a git repository into a local directory...
01-kubernetes-git-sync.md:9 It "publishes" each sync through a worktree and a named symlink. This ensures an atomic update...
01-kubernetes-git-sync.md:13 git-sync can also be configured to make a webhook call or exec a command upon successful git repo synchronization.
01-kubernetes-git-sync.md:23 One of the things in that directory is a symlink ... considered to be the "contract" between git-sync and consumers.
```

This directly matches the content problem: a running Pod needs fresh Markdown files from Git.

### Argo CD webhooks

Argo CD webhooks refresh Argo CD's view of Git/OCI/Helm sources. They are useful for making manifest changes apply faster, but they are not a content-sync mechanism for arbitrary files inside a running Pod.

Key facts from `sources/02-argocd-webhooks.md`:

- Argo CD polls repositories every three minutes by default.
- Webhooks eliminate that delay for application refresh.
- The endpoint is `/api/webhook` on the Argo CD server.

Use Argo CD webhooks for:

- New image tags in GitOps manifests.
- Kubernetes manifest changes.
- Application deployment changes.

Do not rely on Argo CD webhooks alone for:

- Pulling Markdown vault content into a running app.
- Rebuilding the app's in-memory search index.

### Flux GitRepository

Flux `GitRepository` is relevant but not recommended as the first implementation because the target cluster is already Argo CD-based.

Key facts from `sources/03-flux-gitrepositories.md`:

- Flux `GitRepository` produces an Artifact for a Git repository revision.
- `source-controller` fetches at a configured interval.
- The artifact revision is reported in `.status.artifact.revision`.

Flux would make sense if the cluster already used Flux source-controller and Kustomizations. Adding Flux solely for vault content introduces another GitOps controller and operational model.

### Stakater Reloader

Stakater Reloader watches ConfigMap/Secret changes and restarts workloads. It is useful for configuration updates but is not the right mechanism for a large vault repo.

Reasons:

- The vault should not be stored as ConfigMaps.
- Restarting the whole Pod for every content update is heavier than reloading indexes.
- It does not solve Git cloning.

## Recommendation

Use **Argo CD for deployment** and **git-sync sidecar for content**.

The recommended architecture is:

```text
GitHub / Git server
  ├─ app code repo
  │    └─ CI builds ghcr.io/.../retro-obsidian-publish:<sha>
  │
  ├─ k3s GitOps repo
  │    └─ Argo CD deploys manifests and image tags
  │
  └─ vault repo
       └─ git-sync sidecar pulls Markdown into Pod

Kubernetes Pod
  ├─ app container: retro-obsidian-publish
  │    ├─ serves / and /api/*
  │    ├─ reads vault from shared volume
  │    └─ exposes authenticated /api/admin/reload
  │
  ├─ git-sync sidecar
  │    ├─ pulls vault repo every N seconds
  │    ├─ updates atomic symlink after successful sync
  │    └─ POSTs app /api/admin/reload webhook
  │
  └─ emptyDir volume mounted at /git in both containers
```

### Why this is the right split

Argo CD is excellent at reconciling Kubernetes API objects. It is not meant to continuously copy arbitrary Markdown files into an already-running container. `git-sync` is specifically designed for that job.

The app already owns parsing, wiki-link resolution, backlinks, and search indexing. Therefore the content sync should end by telling the app: "a new Git revision is available; reload your content model now."

## Proposed Kubernetes design

### Namespace and resources

Create a kustomize directory:

```text
~/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/
├── namespace.yaml
├── deployment.yaml
├── service.yaml
├── ingress.yaml
├── vault-git-secret.yaml          # or VaultStaticSecret if using Vault Secrets Operator
├── reload-secret.yaml             # or VaultStaticSecret for reload token
└── kustomization.yaml
```

Create an Argo CD Application:

```text
~/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/retro-obsidian-publish.yaml
```

### Deployment sketch

This is a design sketch, not a final manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: retro-obsidian-publish
  labels:
    app.kubernetes.io/name: retro-obsidian-publish
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: retro-obsidian-publish
  template:
    metadata:
      labels:
        app.kubernetes.io/name: retro-obsidian-publish
    spec:
      enableServiceLinks: false
      volumes:
        - name: vault-git
          emptyDir: {}
        - name: git-ssh
          secret:
            secretName: retro-obsidian-vault-git-ssh
            defaultMode: 0400
      containers:
        - name: app
          image: ghcr.io/wesen/retro-obsidian-publish:<sha>
          args:
            - serve
            - --port
            - "8080"
            - --serve-web
            - --vault
            - /git/root/current/vault
          env:
            - name: RETRO_RELOAD_TOKEN
              valueFrom:
                secretKeyRef:
                  name: retro-obsidian-publish-reload
                  key: token
          ports:
            - name: http
              containerPort: 8080
          readinessProbe:
            httpGet:
              path: /api/healthz
              port: http
          livenessProbe:
            httpGet:
              path: /api/healthz
              port: http
          volumeMounts:
            - name: vault-git
              mountPath: /git

        - name: git-sync
          image: registry.k8s.io/git-sync/git-sync:v4.4.0
          args:
            - --repo=git@github.com:wesen/go-go-parc.git
            - --ref=main
            - --root=/git/root
            - --link=current
            - --period=60s
            - --depth=1
            - --ssh-key-file=/etc/git-secret/ssh
            - --ssh-known-hosts=true
            - --ssh-known-hosts-file=/etc/git-secret/known_hosts
            - --webhook-url=http://127.0.0.1:8080/api/admin/reload
            - --webhook-method=POST
            - --webhook-success-status=204
          env:
            - name: RETRO_RELOAD_TOKEN
              valueFrom:
                secretKeyRef:
                  name: retro-obsidian-publish-reload
                  key: token
          volumeMounts:
            - name: vault-git
              mountPath: /git
            - name: git-ssh
              mountPath: /etc/git-secret
              readOnly: true
```

Important caveat: `git-sync`'s webhook configuration may need a way to send an auth header. If git-sync cannot set the required header directly in the version used, use one of these patterns:

1. Put a tiny local reload-proxy sidecar on `127.0.0.1` that receives unauthenticated git-sync webhook calls and injects the secret to the app.
2. Use `--exechook-command` to run a `curl` command with the token, if the git-sync image contains the needed executable or a custom wrapper image is used.
3. Allow localhost-only unauthenticated reloads and rely on Pod network isolation. This is simpler but weaker; prefer token authentication if practical.

## Vault repo layout and symlink details

`git-sync` publishes updates by changing a symlink. This is good because consumers never see a half-updated checkout. It is also subtle for Go's `filepath.Walk` and fsnotify.

Recommended vault repo layout:

```text
vault-repo/
└── vault/
    ├── Index.md
    ├── Research/
    └── Projects/
```

Recommended app vault path:

```text
/git/root/current/vault
```

Why not `/git/root/current`?

- `current` is the git-sync symlink.
- If the application treats the symlink itself as the vault root, some filesystem walking/watching code can see the symlink rather than the directory.
- Using `/git/root/current/vault` makes the final path component a real directory inside the symlinked worktree.

If the vault repo root must be the vault itself, add an app-side symlink normalization/reload strategy before production rollout.

## Required application changes before production

The app can run in k3s today if it starts after content is present. However, for robust content updates it should gain explicit production reload APIs.

### Add `/api/healthz`

Purpose: Kubernetes readiness/liveness.

Response:

```http
GET /api/healthz
200 OK
Content-Type: application/json

{"ok":true,"notes":514}
```

### Add `/api/admin/reload`

Purpose: git-sync webhook target after successful sync.

Contract:

```http
POST /api/admin/reload
Authorization: Bearer <reload-token>

204 No Content
```

Behavior:

1. Authenticate request.
2. Re-evaluate the configured vault path if symlinks are involved.
3. Reload all notes from disk.
4. Rebuild wiki-link index, backlinks, rendered HTML links.
5. Rebuild the search index.
6. Swap the app's active index atomically.
7. Return 204.

Pseudocode:

```go
type RuntimeState struct {
    mu     sync.RWMutex
    vault  *vault.Vault
    search *search.Index
    root   string
}

func (s *RuntimeState) Reload(ctx context.Context) error {
    // Build new state before swapping so readers never see partial reloads.
    newVault, err := vault.New(s.root)
    if err != nil { return err }

    newSearch, err := search.New(newVault)
    if err != nil { return err }

    s.mu.Lock()
    oldSearch := s.search
    s.vault = newVault
    s.search = newSearch
    s.mu.Unlock()

    // Optional future cleanup if search.Index grows Close().
    _ = oldSearch
    return nil
}

func (h *AdminHandler) Reload(w http.ResponseWriter, r *http.Request) {
    if !validBearer(r.Header.Get("Authorization")) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    if err := h.state.Reload(r.Context()); err != nil {
        http.Error(w, "reload failed", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

### Replace fixed handler dependencies with runtime state

Current `api.Handler` stores a fixed `*vault.Vault` and `*search.Index`. A reloadable production server needs a layer that can atomically swap these references.

API sketch:

```go
type State struct {
    mu sync.RWMutex
    vault *vault.Vault
    search *search.Index
}

func (s *State) WithVault(fn func(*vault.Vault)) { ... }
func (s *State) Search(q string, limit int) ([]search.SearchResult, error) { ... }
func (s *State) Reload(root string) error { ... }
```

Simpler first implementation:

- Keep the existing app structure.
- Add `Vault.LoadAll()` call on reload.
- Add `search.Replace(vault)` or create a new `search.Index` and swap it behind a mutex.

The intern should prefer building a new index before swapping so failed reloads do not break the currently served site.

## Why filesystem watching is not enough

The existing watcher is still useful, but do not rely on it as the production sync trigger.

Reasons:

1. It watches directories discovered at startup.
2. `git-sync` publishes new content by flipping a symlink to a new worktree.
3. The watched directories may belong to the old worktree.
4. A full reload is simpler and safer than trying to map symlink flips into per-file fsnotify events.

Keep the watcher for local development. Add webhook reload for production.

## Alternatives considered

### Alternative A: Bake the vault into the application image

Flow:

```text
vault commit → CI builds new app image containing vault → Argo CD deploys new image
```

Pros:

- Immutable artifact.
- No runtime Git credentials in the cluster.
- Rollback is image rollback.

Cons:

- Every content edit requires an image build.
- Private content becomes part of the image artifact.
- Slower feedback loop.
- Mixes app release cadence and content release cadence.

Use this only if immutable publishing is more important than fast content updates.

### Alternative B: Argo CD renders vault files as ConfigMaps

Flow:

```text
vault repo → generated ConfigMaps → mounted files → app watches files
```

Pros:

- Pure GitOps.
- No runtime Git sidecar.

Cons:

- ConfigMaps are a bad fit for a large Obsidian vault.
- Object size limits and noisy diffs.
- Many file changes become Kubernetes API churn.
- Search/index reload still needs solving.

Not recommended.

### Alternative C: Flux GitRepository artifact

Flow:

```text
Flux source-controller fetches vault repo → artifact → custom consumer downloads artifact
```

Pros:

- Strong Git source abstraction.
- Good status reporting.

Cons:

- Adds Flux to an Argo CD cluster.
- Still needs a consumer that unpacks artifacts into a volume or app reloads from artifact.
- More moving parts for this use case.

Not recommended for first implementation.

### Alternative D: Pod restart on every vault change

Flow:

```text
vault commit → signal deployment rollout → Pod restarts → app loads fresh vault
```

Pros:

- No reload endpoint.
- App startup path already works.

Cons:

- More downtime/risk than a reload endpoint.
- Requires a mechanism to change a Deployment annotation on each vault commit.
- Does not fit git-sync's continuous sidecar model.

Acceptable as a fallback but not the best user experience.

## Implementation plan

### Phase 1: Prepare the application for production reloads

Files:

- `backend/internal/server/server.go`
- `backend/internal/api/api.go`
- `backend/internal/search/search.go`
- `backend/internal/vault/vault.go`
- `backend/internal/watcher/watcher.go`

Tasks:

1. Add `GET /api/healthz`.
2. Add `POST /api/admin/reload` behind a bearer token.
3. Add runtime state that can atomically swap vault/search index.
4. Ensure reload failure leaves old state active.
5. Add tests for reload endpoint auth and successful reload.
6. Decide whether production should disable fsnotify watcher (`--watch=false`) and rely only on webhook reload.

Suggested CLI flags:

```text
serve
  --vault /git/root/current/vault
  --port 8080
  --serve-web
  --reload-token-env RETRO_RELOAD_TOKEN
  --watch=false
```

### Phase 2: Build and publish image

Files:

- `backend/Dockerfile`
- `.github/workflows/ci.yml`
- possibly a new release workflow

Tasks:

1. Ensure Dockerfile builds after graph deletion and current embedded assets.
2. Publish image to GHCR with immutable SHA tag.
3. Optionally add `latest` or semver tag after release process exists.

Example image tag:

```text
ghcr.io/wesen/retro-obsidian-publish:sha-<gitsha>
```

### Phase 3: Add Git credentials

Files in k3s repo:

- `gitops/kustomize/retro-obsidian-publish/vault-git-secret.yaml`
- or Vault Secrets Operator resources if this cluster prefers Vault-backed secrets.

Recommended secret keys for SSH:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: retro-obsidian-vault-git-ssh
type: Opaque
stringData:
  ssh: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
  known_hosts: |
    github.com ssh-ed25519 ...
```

Use a read-only deploy key limited to the vault repo.

### Phase 4: Add kustomize app

Files:

```text
gitops/kustomize/retro-obsidian-publish/namespace.yaml
gitops/kustomize/retro-obsidian-publish/deployment.yaml
gitops/kustomize/retro-obsidian-publish/service.yaml
gitops/kustomize/retro-obsidian-publish/ingress.yaml
gitops/kustomize/retro-obsidian-publish/kustomization.yaml
```

Match existing cluster style:

- `namespace.yaml` with sync-wave `-1`.
- Deployment/service sync-wave `1`.
- Ingress sync-wave `2`.
- Traefik ingress class.
- cert-manager `letsencrypt-prod` cluster issuer.

### Phase 5: Add Argo CD Application

File:

```text
gitops/applications/retro-obsidian-publish.yaml
```

Sketch:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: retro-obsidian-publish
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  destination:
    server: https://kubernetes.default.svc
    namespace: retro-obsidian-publish
  source:
    repoURL: https://github.com/wesen/2026-03-27--hetzner-k3s.git
    targetRevision: main
    path: gitops/kustomize/retro-obsidian-publish
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

### Phase 6: End-to-end test

Test path:

1. Merge/push app image and k3s manifests.
2. Verify Argo CD syncs the app.
3. Verify `GET /api/healthz` returns 200.
4. Verify app loads notes from vault repo.
5. Commit a visible change to the vault repo.
6. Wait for git-sync period or force sync.
7. Confirm git-sync logs show new hash.
8. Confirm reload endpoint was called.
9. Confirm changed note is visible in `/api/notes/{slug}` and search results.
10. Confirm no Pod restart was required.

Useful commands:

```bash
kubectl -n retro-obsidian-publish get pods
kubectl -n retro-obsidian-publish logs deploy/retro-obsidian-publish -c git-sync --tail=100
kubectl -n retro-obsidian-publish logs deploy/retro-obsidian-publish -c app --tail=100
kubectl -n retro-obsidian-publish port-forward svc/retro-obsidian-publish 8080:80
curl -fsS http://127.0.0.1:8080/api/healthz
```

## Operational model

### Normal app release

```text
app code commit
  → CI builds image
  → update image tag in k3s repo
  → Argo CD syncs Deployment
  → Kubernetes rolls Pod
  → app starts and git-sync pulls vault
```

### Normal vault content release

```text
vault commit
  → git-sync detects remote change on next period
  → git-sync fetches new revision into new worktree
  → git-sync flips current symlink atomically
  → git-sync calls /api/admin/reload
  → app reloads vault and rebuilds search
  → users see updated content
```

### Rollback

Content rollback can be done in the vault Git repo:

```bash
git revert <bad-content-commit>
git push
```

or by changing git-sync `--ref` temporarily to a known-good tag/commit in the k3s manifests.

Application rollback is done through image tag rollback in the k3s GitOps repo.

## Security notes

- Use a read-only deploy key for the vault repo.
- Store private keys in Kubernetes Secret or Vault Secrets Operator, not in Git.
- Enable SSH known_hosts verification.
- Protect `/api/admin/reload` with a token if exposed beyond localhost.
- Bind reload calls to localhost from the sidecar if possible.
- Do not expose reload endpoint through Ingress.
- Consider NetworkPolicy later if the cluster has a CNI that enforces it.

## Reliability notes

- Use one replica initially. Multiple replicas are possible, but each Pod will run its own git-sync and index reload.
- Use `emptyDir` for the Git checkout. The vault can be re-cloned from Git after Pod reschedule.
- Set git-sync `--depth=1` unless history-dependent features are needed.
- Keep `--period` moderate, for example `30s` or `60s`.
- Ensure app startup handles the case where git-sync has not completed initial sync yet. Options:
  - Use an init container running git-sync `--one-time` first.
  - Or make the app wait/retry until the vault path exists.

Recommended robust pattern:

```text
initContainer: git-sync --one-time
main containers:
  - app
  - git-sync continuous sidecar
```

This avoids app startup failing because `/git/root/current/vault` does not exist yet.

## Intern implementation checklist

Before writing code:

- Read `backend/internal/server/server.go`.
- Read `backend/internal/vault/vault.go`.
- Read `backend/internal/search/search.go`.
- Read `backend/internal/watcher/watcher.go`.
- Read `backend/Dockerfile`.
- Read k3s examples:
  - `gitops/applications/goja-essay.yaml`
  - `gitops/kustomize/goja-essay/deployment.yaml`
  - `gitops/kustomize/goja-essay/service.yaml`
  - `gitops/kustomize/goja-essay/ingress.yaml`

Then implement in this order:

1. Add health endpoint.
2. Add reload endpoint and tests.
3. Add optional `--watch=false` if webhook reload is the production path.
4. Build and publish image.
5. Add kustomize manifests.
6. Add Argo CD Application.
7. Test with a private or test vault repo.
8. Switch to production vault repo.

## API references to add

### `GET /api/healthz`

```http
GET /api/healthz
200 OK
Content-Type: application/json

{
  "ok": true,
  "notes": 514,
  "vaultRoot": "/git/root/current/vault",
  "revision": "optional-git-sha"
}
```

### `POST /api/admin/reload`

```http
POST /api/admin/reload
Authorization: Bearer <token>
204 No Content
```

Error cases:

```http
401 Unauthorized
500 Internal Server Error
```

Logging:

```text
reload: requested by git-sync
reload: loaded 514 notes from /git/root/current/vault
reload: rebuilt search index in 1.2s
reload: active revision abc123
```

## Open questions

1. What is the exact vault repository URL and branch?
2. Is the vault repository root the vault, or does it contain a subdirectory?
3. Which domain should the public site use?
4. Should the published site require basic auth initially?
5. Is the cluster already using Vault Secrets Operator for app secrets, or should the first implementation use plain Kubernetes Secrets?
6. Should production disable fsnotify watcher and rely only on git-sync webhook reload?
7. Should image publication live in this repo's GitHub Actions or be done manually at first?

## References

### External sources downloaded with Defuddle

- `sources/01-kubernetes-git-sync.md`
- `sources/02-argocd-webhooks.md`
- `sources/03-flux-gitrepositories.md`
- `sources/04-stakater-reloader.md`

### Local evidence files

- `sources/05-retro-source-evidence.txt`
- `sources/06-k3s-source-evidence.txt`

### Application source files

- `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/Dockerfile`
- `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/server/server.go`
- `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/vault/vault.go`
- `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/search/search.go`
- `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/backend/internal/watcher/watcher.go`

### k3s GitOps source files

- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/goja-essay.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/deployment.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/service.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/ingress.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-essay/kustomization.yaml`
