---
Title: retro-obsidian-publish prod OOM, memory architecture, and git-push reload — intern guide
Ticket: RETRO-MEMORY-012
Status: active
Topics:
    - retro-obsidian-publish
    - vault
    - search
    - deployment
    - git-sync
    - k3s
    - obsidian-vault
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/2026-03-27--hetzner-k3s/gitops/applications/retro-obsidian-publish.yaml
      Note: ArgoCD Application with automated prune+selfHeal
    - Path: ../../../../../../../../../../code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml
      Note: prod pod spec
    - Path: cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: serve flags incl --reload-allow-loopback
    - Path: internal/api/api.go
      Note: route table; getNoteRaw is the only consumer of RawMarkdown
    - Path: internal/search/search.go
      Note: New() uses bleve.NewMemOnly (in-RAM); NewPersistent() on-disk path exists but is unused
    - Path: internal/server/runtime.go
      Note: RuntimeState.Reload builds a full second state before swapping (transient 2x memory = OOM trigger)
    - Path: internal/server/server.go
      Note: Run registers /api/admin/reload; reloadHandler; assetHandler shows safe OpenRoot pattern to reuse for lazy raw reads
    - Path: internal/vault/vault.go
      Note: Note struct holds RawMarkdown + HTML (3x memory redundancy); LoadAll
ExternalSources: []
Summary: 'Intern-facing analysis/design/implementation guide for the retro-obsidian-publish prod OOM: diagnosis, three-fold in-memory redundancy, the git-sync reload carry-over, and a phased fix.'
LastUpdated: 2026-07-05T00:00:00Z
WhatFor: Onboarding a new engineer to the retro-obsidian-publish system and the prod OOM fix
WhenToUse: Read before changing vault loading, the search index, the reload path, or the deployment memory limits
---


# retro-obsidian-publish: Prod OOM, Memory Architecture, and Git-Push Reload — Intern Guide

**Ticket:** RETRO-MEMORY-012
**Audience:** A new engineer joining the team who needs to understand the whole system before changing it.
**Goal of this document:** Explain, end to end, (1) what `retro-obsidian-publish` is, (2) why the production deployment is currently down, (3) how memory is consumed inside the Go process, (4) how a git push reaches the running pod without a restart, and (5) exactly how to fix the outage and harden the system. Every claim is anchored to a file and (where useful) a line number.

---

## 0. How to read this document

- **Bold inline tokens** like `app container` refer to Kubernetes concepts.
- `monospace` tokens like `RawMarkdown` refer to code identifiers you can grep.
- `path/to/file.go:NN` is a line reference. Resolve these against the workspace root: `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault/`.
- Diagrams are ASCII so they render everywhere (including the reMarkable upload).
- **"Evidence" callouts** quote observed production state captured on 2026-07-05.

There are two repositories involved. Keep them distinct:

| Repo | Path on this machine | Role |
|------|----------------------|------|
| `publish-vault` | `/home/manuel/workspaces/2026-07-05/memory-publish-vault/publish-vault` (this workspace; a worktree on branch `task/memory-publish-vault`) | The Go application source. This is where code changes happen. |
| `2026-03-27--hetzner-k3s` | `/home/manuel/code/wesen/2026-03-27--hetzner-k3s` | The GitOps repo: Kubernetes manifests (ArgoCD + Kustomize) that deploy the app to the cluster. This is where the deployment YAML lives. |

---

## 1. Executive summary

- **Prod is down.** The `retro-obsidian-publish` pod in namespace `retro-obsidian-publish` is in `CrashLoopBackOff`. The `app` container is repeatedly **OOMKilled** (exit code `137`), with 236 restarts observed over ~22 days. The pod reports `2/3` ready because only the `app` container dies; the `ssr` and `git-sync` containers are healthy.
- **Root cause is memory pressure inside the Go process, not cluster starvation.** The single node has ~6.8 GiB free RAM and 7% CPU usage. The `app` container simply outgrows its own `1536Mi` memory limit.
- **Why it grows:** each of the 887 vault notes is held in memory in **three** redundant forms — raw markdown, rendered HTML, and a bleve full-text index — and the index is built entirely in RAM (`bleve.NewMemOnly`). A reload transiently holds **two complete copies** of all three before swapping, which is the most likely trigger that tips the process over the limit every ~60 seconds when `git-sync` pulls.
- **Immediate unblock (ops-only, no code):** raise the `app` container memory limit to ~2560–3072 Mi. The node has the headroom.
- **Proper fix (code):** move the search index to disk via the **already-existing-but-unused** `search.NewPersistent()` path, and stop caching `RawMarkdown` (and optionally `HTML`) in the in-memory `Note` struct. A full SQL database is **not** required.

---

## 2. Problem statement and scope

### 2.1 What is broken right now

- The public site `https://parc.yolo.scapegoat.dev` is served by this deployment.
- The backing pod cannot stay up long enough to serve traffic: it starts, loads the vault (~21 s), begins listening, and is killed ~19 s later.

> **Evidence (captured 2026-07-05):**
> ```
> kubectl get pod -n retro-obsidian-publish
> NAME                                      READY   STATUS             RESTARTS          AGE
> retro-obsidian-publish-5b46b488f5-gx69l   2/3     CrashLoopBackOff   439 (2m20s ago)   22d
> ```
> Container statuses:
> - `app`      → `ready=false`, `restartCount=236`, last state `terminated.exitCode=137`, reason `OOMKilled`, ran ~40 s.
> - `ssr`      → `ready=true`,  `restartCount=0`.
> - `git-sync` → `ready=true`,  `restartCount=203` (old; last error 2026-06-25, currently healthy).

Exit code `137` = `128 + 9` (SIGKILL). Inside a container with a memory limit, this is the kernel/cgroup OOM killer. Kubernetes surfaces this as `reason: OOMKilled`.

### 2.2 What is in scope for this ticket

1. Diagnose and confirm the OOM root cause (done — see §4).
2. Unblock production with a minimal, safe change (see §7, Phase 0).
3. Reduce the application's steady-state memory so it has headroom to grow (see §7, Phases 1–3).
4. Preserve and harden the "git push updates the live site" flow (see §5 and §7, Phase 2).

### 2.3 What is explicitly out of scope

- Migrating to a SQL/Postgres backing store (considered and rejected; see Decision D3).
- Changing the React frontend, SSR hydration, or SEO behaviour.
- Re-architecting the cluster, node size, or other namespaces.

---

## 3. Current-state architecture (evidence-based)

### 3.1 System context — what `retro-obsidian-publish` actually is

`retro-obsidian-publish` (the binary) is a small Go web server that publishes an Obsidian Markdown vault as a browsable, searchable, retro-styled website. It is **not** Obsidian; it reads `.md` files from a directory tree and serves them over HTTP.

At runtime it does four things:

1. **Loads** every `.md` file in a vault directory into memory.
2. **Indexes** them into a full-text search index (bleve).
3. **Serves** a JSON HTTP API (`/api/...`) for listing, reading, searching, and browsing notes.
4. **Proxies** page requests to an optional **SSR sidecar** (a Node process) for server-side rendering, falling back to a bundled React SPA.

The vault content itself is **not** baked into the image. It is fetched at runtime by a `git-sync` sidecar that clones `git@github.com:go-go-golems/go-go-parc.git` and keeps it up to date.

### 3.2 Deployment topology

The deployment runs **one pod** with three long-running containers plus one init container. All containers share an `emptyDir` volume named `vault-git` mounted at `/git`.

```
                        Pod: retro-obsidian-publish-5b46b488f5-*
                        (namespace: retro-obsidian-publish)
+------------------------------------------------------------------------+
|  shared emptyDir "vault-git" -> /git  (git checkout + worktrees)       |
|------------------------------------------------------------------------|
|                                                                        |
|  [init] git-sync-init   --one-time, clones go-go-parc main into /git   |
|                                                                        |
|  [container] app        publish-vault serve  (Go API + SPA, :8080)     |
|     - reads /git/root/current (symlink to a git worktree)              |
|     - holds ALL notes + search index in RAM                           |
|                                                                        |
|  [container] ssr        publish-vault-ssr   (Node, :8089)              |
|     - calls back into app at http://127.0.0.1:8080 for page data       |
|                                                                        |
|  [container] git-sync   polls git every 60s; on new commit:            |
|     - updates /git/root/current worktree                               |
|     - POSTs http://127.0.0.1:8080/api/admin/reload  (the carry-over)   |
+------------------------------------------------------------------------+
            |                          |                        |
       Service :80            (loopback)                 Ingress (traefik)
            |                                                     |
            +--------------------> parc.yolo.scapegoat.dev <------+
```

- The init container and the `git-sync` sidecar both use the image `registry.k8s.io/git-sync/git-sync:v4.4.0`.
- The `app` and `ssr` containers use image tag `sha-575803c` from `ghcr.io/go-go-golems/publish-vault` and `publish-vault-ssr` respectively.
- Secrets (GitHub SSH key for git, GHCR image-pull credentials) are materialized from HashiCorp Vault by the Vault Secrets Operator (`VaultStaticSecret` resources).

### 3.3 GitOps wiring

The deployment is declared in the **hetzner-k3s repo**, not here:

- Manifests: `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml`
- ArgoCD Application: `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/retro-obsidian-publish.yaml`

ArgoCD continuously reconciles the `main` branch of the hetzner-k3s repo against the cluster, with `automated.prune` and `selfHeal` enabled. **This matters:** any change you merge to that repo's `main` branch is applied to prod automatically and quickly. To change a memory limit, you edit the manifest there and open a PR (see §7 Phase 0).

### 3.4 The HTTP surface (what the app actually serves)

Routes are registered in `internal/api/api.go` (`Handler.Register`, around `api.go:54-64`) and `internal/server/server.go` (`Run`, `server.go:60-110`).

| Method | Path | Handler | What it returns |
|--------|------|---------|-----------------|
| GET | `/api/config` | `getConfig` | `{ vaultName, pageTitle, notes }` |
| GET | `/api/notes` | `listNotes` | All notes (lightweight list items) |
| GET | `/api/notes/{slug}` | `getNote` | One full note incl. `html`, `rawMarkdown`, `frontmatter`, `wikiLinks`, `backlinks` |
| GET | `/api/notes/{slug}/raw` | `getNoteRaw` | Raw markdown source of one note |
| GET | `/api/tree` | `getTree` | Hierarchical file tree of the vault |
| GET | `/api/search?q=` | `searchNotes` | Full-text search results |
| GET | `/api/tags` | `listTags` | All tags with counts |
| GET | `/api/healthz` | `healthHandler` | Liveness/readiness probe target |
| POST | `/api/admin/reload` | `reloadHandler` | **Full reindex without restart** (§5) |
| GET | `/vault-assets/*` | `assetHandler` | Vault images/attachments streamed from disk |
| GET | `/*` | SPA handler / SSR proxy | The website itself |

Note the asymmetry that matters for memory: `getNote` (`api.go:122-135`) returns `HTML`, and `getNoteRaw` (`api.go:137-153`) returns `RawMarkdown`. Both fields are stored on every `Note` in memory at all times (§4.2).

---

## 4. Gap analysis: why the process runs out of memory

### 4.1 It is NOT cluster starvation

> **Evidence (captured 2026-07-05):**
> ```
> kubectl top nodes
> NAME         CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
> k3s-demo-1   631m         7%       9152Mi          58%
> ```
> Node allocatable: `15982912Ki` (~15.6 GiB). Measured total RSS across all pods: ~5.6 GiB. Sum of pod memory **requests**: ~9.1 GiB (fits). Sum of **limits**: ~24 GiB (overcommitted, but normal and not the cause here).

The heavyweights are platform services (monitoring 1320 MiB, mysql 726, argocd 679, keycloak 623). `retro-obsidian-publish` measures low (~230 MiB) only because it gets killed before it settles. There is ~6.8 GiB free on the node. Nothing is starving the app; the app outgrows its **own** limit.

The offending limit, from `deployment.yaml`:

```yaml
# app container resources
resources:
  requests: { cpu: 100m, memory: 256Mi }
  limits:   { memory: 1536Mi }     # <-- the ceiling it keeps hitting
```

### 4.2 The in-memory data model holds three copies of every note

The core data structure is the `Note` struct (`internal/vault/vault.go:18-29`):

```go
type Note struct {
    Slug        string                 // URL slug
    Title       string
    Path        string                 // relative path inside vault
    Frontmatter map[string]interface{}
    Tags        []string
    Excerpt     string
    HTML        string                 // copy #1: rendered HTML
    RawMarkdown string                 // copy #2: original .md source
    WikiLinks   []WikiLinkRef
    Backlinks   []string
    ModTime     time.Time
}
```

`LoadAll()` (`vault.go:71`) walks the vault directory and calls `loadNote()` for every `.md` file. `loadNote()` (`vault.go:131-162`) reads the file with `os.ReadFile` and stores **both** the rendered HTML and the full raw source:

```go
return &Note{
    ...
    HTML:        parsed.HTML,
    RawMarkdown: string(src),   // vault.go:156  — entire file held as a Go string
    ...
}, nil
```

On top of these two text copies, the search package builds a **third**, tokenized representation entirely in RAM (`internal/search/search.go:41-53`):

```go
func New(v *vault.Vault) (*Index, error) {
    idx, err := bleve.NewMemOnly(buildMapping())   // search.go:42 — in-memory only
    if err != nil {
        return nil, err
    }
    si := &Index{idx: idx}
    for _, note := range v.AllNotes() {
        if err := si.Index(note); err != nil {     // indexes title + body + tags + excerpt
            return nil, err
        }
    }
    return si, nil
}
```

`Index()` (`search.go:80-103`) pushes the full body text (HTML stripped) into bleve's analyzer, which builds term dictionaries and postings lists — all in RAM because of `NewMemOnly`.

### 4.3 Quantifying the footprint

> **Evidence:** the prod checkout contains 887 `.md` files (confirmed via the `git-sync` sidecar). The same vault locally is 934 `.md` files totaling **~22.8 MiB of raw markdown**.

Steady-state, the process therefore holds, very roughly:

```
~23  MiB  RawMarkdown       (copy #2, one Go string per file)
~25-45 MiB  HTML             (copy #1, often larger than the source due to markup)
~30-60 MiB  bleve in-RAM     (copy #3, term dictionaries + postings + analyzers)
+ per-Note struct/map/slice overhead, frontmatter maps, wiki-link slices
+ Go runtime, goroutines, HTTP buffers, GC overhead
```

For 887 notes this lands in the low-to-mid hundreds of MiB at idle. That fits under 1536 MiB — which is exactly why the pod *sometimes* starts cleanly. The killer is the **transient spike** described next.

### 4.4 The reload transiently doubles memory (the likely OOM trigger)

`POST /api/admin/reload` calls `RuntimeState.Reload()` (`internal/server/runtime.go:62-76`):

```go
func (s *RuntimeState) Reload() error {
    configured := s.ConfiguredRoot()
    v, si, resolved, err := loadVaultAndSearch(configured)  // build ENTIRE new state
    if err != nil {
        return err
    }
    s.mu.Lock()
    s.vault = v          // swap
    s.search = si        // swap
    s.resolvedRoot = resolved
    s.mu.Unlock()
    return nil
}
```

`loadVaultAndSearch()` (`runtime.go:77-96`) constructs a brand-new `vault.Vault` **and** a brand-new bleve index. The old `Vault` and `Index` are only eligible for garbage collection **after** the pointer swap. So during a reload the process briefly holds **two** full vaults + **two** full indexes concurrently:

```
steady state:    [ Vault_A + Index_A ]                          ~ a few hundred MiB
during reload:   [ Vault_A + Index_A ] + [ Vault_B + Index_B ]  ~ 2x  ←  crosses 1536Mi → OOMKilled
after swap+GC:   [ Vault_B + Index_B ]                          ~ a few hundred MiB
```

This is consistent with the observed cadence: the app loads (~21 s), serves briefly (~19 s), and dies — right around the time `git-sync` (polling every 60 s) would fire its reload webhook. The pod restarts, and the cycle repeats. (Note: the `app` logs themselves are clean — no panic, no error — which is exactly what an OOM kill looks like: the kernel simply removes the process.)

### 4.5 Why `RawMarkdown` is wasteful to cache

`RawMarkdown` is consumed by exactly **one** endpoint: `GET /api/notes/{slug}/raw` (`api.go:137-153`), used for "view source" / raw download. The file already lives on disk at `/git/root/current/<path>.md`. Caching it in RAM for every note to serve an infrequently used endpoint is a poor memory trade. The same logic applies, less urgently, to `HTML` (served by `GET /api/notes/{slug}`).

---

## 5. The git-push carry-over flow (how a push reaches the live pod)

This is the "nicer carry-over on git push" the team wants to preserve and harden. It is **already implemented and wired in prod** — no new mechanism is required.

### 5.1 The end-to-end path

```
 you push to go-go-parc main
        |
        v
 GitHub (git@github.com:go-go-golems/go-go-parc.git)
        |
        | (polled every 60s)
        v
 git-sync sidecar in the pod  (--period=60s, --depth=1)
        |
        | 1. fetches new commit into a NEW worktree under /git/root/.worktrees/<sha>
        | 2. atomically re-points /git/root/current -> new worktree
        | 3. POSTs the webhook
        v
 POST http://127.0.0.1:8080/api/admin/reload
   --webhook-method=POST
   --webhook-success-status=204
   --webhook-timeout=30s
        |
        v
 app container: reloadHandler() -> RuntimeState.Reload()
        |
        | builds new Vault + new bleve Index from /git/root/current
        | swaps them under a write lock
        v
 readers now see the new notes; old state GC'd
```

The relevant `git-sync` args are in `deployment.yaml`:

```yaml
- --repo=git@github.com:go-go-golems/go-go-parc.git
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
- --webhook-timeout=30s
```

### 5.2 Authentication of the reload endpoint

The reload endpoint is only registered when a token or loopback mode is configured (`server.go:88-94`):

```go
if cfg.ReloadToken != "" || cfg.ReloadAllowLoopback {
    r.HandleFunc("/api/admin/reload", reloadHandler(state, cfg.ReloadToken, cfg.ReloadAllowLoopback)).Methods("POST")
}
```

Prod runs with `--reload-allow-loopback` (set in `deployment.yaml` args) and **no** token. Authorization is enforced by `validReloadRequest()` (`server.go`): a request is accepted if it carries the correct bearer token **or** originates from a loopback IP. Because `git-sync` calls `127.0.0.1`, it authenticates purely by being local.

> **Implication for the intern:** to trigger a reload manually from outside the pod you have two options:
> 1. `kubectl port-forward` + a bearer token (requires setting `RETRO_RELOAD_TOKEN`), or
> 2. exec into the pod and curl loopback:
>    ```bash
>    kubectl exec -n retro-obsidian-publish <pod> -c app -- \
>      curl -sS -X POST http://127.0.0.1:8080/api/admin/reload -w '%{http_code}\n'
>    ```
>    Expect `204` on success.

### 5.3 The carry-over works — but it is also what kills the pod

The reload path is correct and elegant (atomic snapshot swap, readers never block). The problem is purely the **memory cost of building a second full state** (§4.4). So "harden the carry-over" and "fix the OOM" are the same work: make a reload cheap enough that the transient 2× footprint stays under the limit.

---

## 6. Proposed architecture and APIs

The fix is staged so that production can be unblocked in minutes (Phase 0) before any code change, and each subsequent phase is independently shippable and reversible.

### 6.1 Design principles

1. **Do not change the public HTTP API.** `/api/notes/{slug}` and `/api/notes/{slug}/raw` keep the same JSON shape.
2. **Keep the reload contract.** `POST /api/admin/reload` stays the carry-over trigger; the atomic-swap semantics stay.
3. **Prefer the smallest change that removes the failure mode.** A SQL migration is a bigger surface area for no incremental benefit at this vault size.
4. **Make memory usage measurable.** Add a `GOMEMLIMIT` and expose note/index counts and a memory figure on `/api/healthz`.

### 6.2 Component-by-component changes

- **`search.Index`** — switch from `bleve.NewMemOnly` to the existing `bleve.New`/`bleve.Open` on-disk path (`search.NewPersistent`, `search.go:56-75`). Store the index on an `emptyDir` (or a small PVC for persistence across restarts). This removes the single largest RAM consumer and makes a reload an incremental update rather than a from-scratch rebuild.
- **`vault.Note`** — drop `RawMarkdown` from the in-memory struct. Serve `GET /api/notes/{slug}/raw` by reading the file from `ResolvedRoot()` on demand (the `assetHandler` already demonstrates safe `os.OpenRoot` usage at `server.go:189-218`).
- **(Optional) `vault.Note.HTML`** — drop `HTML` from the hot struct too, and regenerate a single note's HTML on demand in `getNote`. This is a larger change because `rebuildHTML` (`vault.go:214-235`) currently rewrites every note's HTML to resolve wiki links against the index; on-demand rendering must resolve links per request. Defer unless the vault grows much larger.
- **`RuntimeState.Reload`** — add a `--search-index-path` flag plumbed through `serve.Settings` → `appserver.Config` → `loadVaultAndSearch`. When set, use the persistent index; when empty, fall back to today's in-memory behaviour (safe default).

### 6.3 API reference (unchanged externally, internal behaviour changes)

```
GET  /api/notes/{slug}/raw
  Behaviour change: reads <ResolvedRoot()>/<note.Path> from disk instead of
                    returning a cached string. Same response body & headers.
  New failure mode: if the file was removed between index time and request time,
                    return 404 (today this would return stale content; the new
                    behaviour is arguably more correct).

POST /api/admin/reload
  Behaviour change: rebuilds the on-disk bleve index incrementally where possible,
                    rather than constructing a full in-RAM index. Same 204 success.
```

No other route changes.

---

## 7. Implementation plan (phased)

### Phase 0 — Unblock prod (ops only, no code, ~10 minutes)

**Goal:** stop the CrashLoop so the site comes back while code work proceeds.

1. In the **hetzner-k3s repo**, edit `gitops/kustomize/retro-obsidian-publish/deployment.yaml`, `app` container:
   ```yaml
   resources:
     requests: { cpu: 100m, memory: 256Mi }
     limits:   { memory: 3072Mi }   # was 1536Mi
   ```
2. Open a PR titled `retro-obsidian-publish: raise app memory limit to 3072Mi (OOM hotfix)`. Merge to `main`.
3. ArgoCD auto-applies it; the pod's `Deployment` revision changes and it rolls.
4. Verify:
   ```bash
   export KUBECONFIG=$PWD/.cache/kubeconfig-tailnet.yaml
   kubectl get pod -n retro-obsidian-publish -w      # watch READY go to 3/3
   kubectl top pod -n retro-obsidian-publish          # observe steady RSS
   curl -s https://parc.yolo.scapegoat.dev/api/healthz
   ```
5. Acceptance: `READY 3/3`, restarts stop climbing, `/api/healthz` returns `{"ok":true,...}`.

> **Why 3072Mi and not higher:** measured steady state is a few hundred MiB; the reload transient roughly doubles it. 3072Mi gives ~2× headroom over the worst observed transient on a node that has ~6.8 GiB free. Revisit after Phase 1/2 reduce the steady state.

### Phase 1 — Persist the search index on disk (biggest single win)

**Goal:** remove the in-RAM bleve index, the largest memory consumer, and make reloads cheaper.

1. Add a flag `--search-index-path` (default empty) in `serve.Settings` (`cmd/retro-obsidian-publish/commands/serve/serve.go`) and thread it into `appserver.Config`.
2. In `internal/server/runtime.go` `loadVaultAndSearch`, branch on the configured path:
   ```go
   // pseudocode — do not copy verbatim; adapt to the real signatures
   func loadVaultAndSearch(root, indexPath string) (*vault.Vault, *search.Index, string, error) {
       v, err := vault.New(resolvedRoot)        // unchanged
       if err != nil { return nil, nil, "", err }

       var si *search.Index
       if indexPath == "" {
           si, err = search.New(v)              // today's in-memory path (fallback)
       } else {
           si, err = search.NewPersistent(v, indexPath)  // already exists at search.go:56
       }
       if err != nil { return nil, nil, "", err }
       return v, si, resolvedRoot, nil
   }
   ```
3. In `deployment.yaml`, mount an `emptyDir` at `/data/search` and set `--search-index-path=/data/search/index`.
4. Optional hardening: change `search.NewPersistent` to **incrementally** update rather than re-index everything on each call (today it re-indexes all notes on every open — `search.go:66-72`). For a first cut, full re-index on an on-disk index is fine and still far cheaper than in-RAM.
5. Tests:
   - Extend `internal/search/search_test.go` with a case that opens, closes, and reopens a persistent index and asserts search results are identical.
   - Add a `serve` integration test asserting `/api/search` works with `--search-index-path` set.
6. Acceptance: steady-state RSS drops materially; reload no longer holds two full in-RAM indexes.

> **Trick to watch:** `search.NewPersistent` currently re-indexes **all** notes on every `Open` (`search.go:66-72`). That keeps correctness simple but means a reload still does a full reindex — just on disk, which is far cheaper memory-wise. Document this; a later phase can make it incremental by diffing note `ModTime`/slugs.

### Phase 2 — Stop caching `RawMarkdown` (medium win, low risk)

**Goal:** remove copy #2 from the hot struct.

1. In `internal/vault/vault.go`:
   - Remove the `RawMarkdown` field from `Note` (or keep it for JSON compat but stop populating it — prefer removal plus an on-demand reader).
   - Stop assigning it in `loadNote` (`vault.go:156`).
2. Add a method to read raw source on demand:
   ```go
   // pseudocode
   func (v *Vault) ReadRaw(relPath string) ([]byte, error) {
       root, err := os.OpenRoot(v.root)
       if err != nil { return nil, err }
       defer root.Close()
       f, err := root.Open(filepath.FromSlash(relPath))
       if err != nil { return nil, err }
       defer f.Close()
       return io.ReadAll(f)
   }
   ```
3. In `internal/api/api.go` `getNoteRaw` (`api.go:137-153`), replace `note.RawMarkdown` with a `Vault.ReadRaw(note.Path)` call. Mirror the safe-path checks already used by `assetHandler`.
4. Tests: update `internal/api/api_test.go` to assert `/raw` still returns the file content and 404s for missing files.
5. Acceptance: `/api/notes/{slug}/raw` unchanged behaviour; `Note` struct measurably smaller; no test regressions.

### Phase 3 — (Optional) Lazy `HTML` and `GOMEMLIMIT`

Only if the vault keeps growing after Phases 1–2:

1. Move `HTML` rendering to per-request in `getNote`, resolving wiki links against the current `wikiLinkIndex` at request time.
2. Set `GOMEMLIMIT=2304MiB` and `GOGC=50` in the container env to cap the Go GC and bound the heap softly under the limit.
3. Expose `runtime/metrics` Go memory stats on `/api/healthz` for observability.

### Sequencing summary

```
Phase 0 (ops)   raise limit 3072Mi        -> site back up            [~10 min]
Phase 1 (code)  persist bleve index       -> biggest RAM drop        [~1 day]
Phase 2 (code)  lazy RawMarkdown          -> removes copy #2         [~0.5 day]
Phase 3 (code)  lazy HTML + GOMEMLIMIT    -> future-proofing         [optional]
```

---

## 8. Decision records

### Decision D1: How to unblock prod

- **Context:** The site is down now. Code fixes take days; the cluster has spare RAM.
- **Options considered:** (a) raise the memory limit, (b) wait for a code fix, (c) vertically scale the node.
- **Decision:** Raise the `app` container memory limit to 3072Mi via the hetzner-k3s GitOps repo.
- **Rationale:** Lowest-risk, fastest, reversible, and the node has the headroom. It does not mask the root cause; Phases 1–2 still land.
- **Consequences:** Prod returns to service immediately. The memory budget is consumed less efficiently until code fixes land.
- **Status:** proposed

### Decision D2: Where the search index lives

- **Context:** `bleve.NewMemOnly` (`search.go:42`) makes the index the largest in-RAM consumer and doubles on reload.
- **Options considered:** (a) keep in-RAM, (b) on-disk bleve via the existing `NewPersistent`, (c) external SQL/Postgres.
- **Decision:** On-disk bleve (`NewPersistent`) on an `emptyDir`/PVC.
- **Rationale:** The code path already exists (`search.go:56-75`); bleve is already the dependency; the vault is small (~23 MiB) so disk I/O is trivial; no new operational surface (no DB to run/backup).
- **Consequences:** Removes the biggest RAM consumer. Reloads become cheaper. Adds a writable volume mount. Future incremental indexing is possible but not required now.
- **Status:** proposed

### Decision D3: SQL/Postgres backend

- **Context:** Asked whether notes should be indexed into a database.
- **Options considered:** (a) SQLite, (b) Postgres (already running in-cluster for other apps), (c) on-disk bleve.
- **Decision:** Do **not** introduce a SQL backend at this time.
- **Rationale:** The problem is *redundant in-memory copies*, not a lack of a query engine. bleve already does full-text search well. A SQL migration adds schema management, a new dependency, backup burden, and a new failure mode — for no incremental benefit at 887 notes. Revisit only if the vault exceeds ~5–10k notes or if multi-replica horizontal scale is needed.
- **Consequences:** Smaller blast radius. The team keeps a single-process, file-backed mental model.
- **Status:** accepted

### Decision D4: Reload semantics

- **Context:** Reload builds a full second state before swapping (`runtime.go:62-76`).
- **Options considered:** (a) keep full rebuild + swap, (b) incremental in-place update.
- **Decision:** Keep full rebuild + atomic swap; make each rebuild cheap via Phase 1/2.
- **Rationale:** Atomic swap gives readers a consistent snapshot with no locks on the read path — a valuable property. Incremental in-place updates would require fine-grained locking and invalidation. Cheaper rebuilds preserve the simple model.
- **Consequences:** Reload transient still briefly holds two states, but each state is much smaller after Phases 1–2, so the transient stays well under the limit.
- **Status:** accepted

---

## 9. Test strategy

1. **Unit tests** in each touched package (`vault`, `search`, `api`) — extend the existing `*_test.go` files. All packages already have tests; follow their style.
2. **Regression test for the OOM symptom:** a Go test or small benchmark that loads the `vault-example` (or a generated large vault), triggers a `Reload`, and asserts peak heap (via `runtime.MemStats`) stays below a budget. This is the test that would have caught the prod regression.
3. **Integration test** for the carry-over: simulate `git-sync` by swapping the symlink target of a vault dir, call `state.Reload()`, and assert `/api/notes` reflects the new content and `/api/notes/{slug}/raw` still works.
4. **Local smoke (tmux):** run `go run ./cmd/retro-obsidian-publish serve --vault ./vault-example --port 8080` in tmux, hit `/api/healthz`, `/api/search?q=...`, `/api/notes`, and a `/raw` endpoint, then kill with `lsof-who -p 8080 -k` (per the workspace `AGENT.md`).
5. **Prod verification** (post-deploy): the commands in Phase 0 step 4, plus a manual `POST /api/admin/reload` via exec and a check that the pod survives the next few `git-sync` cycles.

---

## 10. Risks, alternatives, and open questions

### Risks

- **R1 — `NewPersistent` re-indexes on every open.** Not a memory risk, but a CPU/latency risk on each reload. Mitigation: acceptable for now; make incremental later (note `ModTime` diffing).
- **R2 — Concurrent reload + request.** The atomic swap handles this correctly today; verify Phase 2's on-disk raw read does not race a worktree rotation by `git-sync`. `git-sync` flips the `current` symlink atomically and only removes old worktrees after a grace period, so reading through `EvalSymlinks`-resolved root is safe, but add a test.
- **R3 — `/raw` 404 on deleted notes.** Slight behaviour change; document it in the changelog.

### Alternatives considered and rejected

- SQL backend (D3).
- Increasing node RAM / moving to a bigger instance (treats the symptom, costs money, doesn't fix the 3× redundancy).
- Disabling the reload webhook (would break the carry-over; rejected).

### Open questions

- Q1: Should the persistent index live on an `emptyDir` (lost on pod restart, rebuilt once) or a small PVC (survives restarts)? Recommend `emptyDir` first; revisit if restart latency matters.
- Q2: Do we want a bearer token for `/api/admin/reload` in addition to loopback, for external triggers? Not required today.

---

## 11. References (key files)

**Application source (this workspace):**
- `internal/vault/vault.go` — `Note` struct (`:18-29`), `LoadAll` (`:71`), `loadNote` (`:131-162`), `rebuildHTML` (`:214-235`), `buildBacklinks` (`:312`).
- `internal/search/search.go` — `New` (in-mem, `:41-53`), `NewPersistent` (on-disk, `:56-75`), `Index` (`:80-103`).
- `internal/server/runtime.go` — `RuntimeState`, `Reload` (`:62-76`), `loadVaultAndSearch` (`:77-96`).
- `internal/server/server.go` — `Run`, route registration (`:60-110`), `reloadHandler` (`:166`), `assetHandler` (`:189-218`), reload auth (`validReloadRequest`).
- `internal/api/api.go` — route table (`:54-64`), `getNoteRaw` (`:137-153`), `searchNotes`, `listNotes`.
- `cmd/retro-obsidian-publish/commands/serve/serve.go` — CLI flags incl. `--reload-allow-loopback`, `--ssr-url`.

**GitOps / deployment (hetzner-k3s repo):**
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml` — pod spec, container args, resource limits, git-sync webhook wiring.
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/retro-obsidian-publish.yaml` — ArgoCD Application (auto-prune, self-heal).
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/projects/prod-apps.yaml` — AppProject.

**Production observation commands:**
- `kubectl get pod -n retro-obsidian-publish -o wide`
- `kubectl top nodes` / `kubectl top pods -A --sort-by=memory`
- `kubectl get pod -n retro-obsidian-publish -o jsonpath='{...containerStatuses...}'`
- `kubectl exec -n retro-obsidian-publish <pod> -c git-sync -- ...` (the `app` container is unreachable via exec while in CrashLoop; use `git-sync` to inspect the shared `/git` volume).

---

## 12. Glossary

- **OOMKilled** — Kubernetes reason recorded when a container is killed by the cgroup memory OOM killer (exit 137).
- **git-sync** — A Kubernetes sidecar image (`registry.k8s.io/git-sync`) that clones/pulls a repo into a shared volume and can fire a webhook on update.
- **bleve** — A Go full-text search library. `NewMemOnly` builds an index fully in RAM; `New`/`Open` use a directory on disk.
- **Carry-over** — In this ticket: the mechanism by which a new git push reaches the already-running pod without a container restart (git-sync → webhook → `/api/admin/reload`).
- **emptyDir** — A Kubernetes volume backed by the node's local storage, scoped to the pod's lifetime.
- **Snapshot swap** — The `RuntimeState` pattern of building a fully new state, then atomically replacing the pointer under a lock so readers never observe a half-built state.
