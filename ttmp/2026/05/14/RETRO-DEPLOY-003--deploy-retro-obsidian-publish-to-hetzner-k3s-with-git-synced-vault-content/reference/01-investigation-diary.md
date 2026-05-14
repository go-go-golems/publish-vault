---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/sources/01-kubernetes-git-sync.md
      Note: Downloaded git-sync reference
    - Path: ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/sources/02-argocd-webhooks.md
      Note: Downloaded Argo CD webhook reference
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track the research, design, documentation, validation, and reMarkable delivery work for deploying `retro-obsidian-publish` to the Hetzner k3s cluster with Git-backed vault content synchronization.

## Step 1: Ticket creation, source download, and evidence gathering

Created a new docmgr ticket for the deployment design and gathered both external references and local repository evidence before writing recommendations. The goal was to answer whether a Git-backed vault should be synchronized into the running web server and how that should fit the existing Argo CD/k3s setup.

I downloaded primary source material with Defuddle into the ticket `sources/` directory, then captured line-numbered evidence from the application repository and the k3s GitOps repository. This gives the final design doc concrete anchors instead of relying on generic Kubernetes advice.

### Prompt Context

**User prompt (verbatim):** "ok, so let's move towards deploying this in our ~/code/wesen/2026-03-27--hetzner-k3s/ k3s cluster. Before we start, after analyzing things a bit, how do you figure i should push my updated vault so that it gets updated in the web server? since i have it backed by git, i figure one nice solution would be to have a git sync that then does whatever indexing and such is necesary. What do you think? are there solutions out ther ealready for this kind of content sync by git for argocd/gitops/k3s deployments?"

**User prompt (verbatim):** "Create a new docmgr ticket and write up a full design and implementation guide. DOwnload relevant resources with defuddle and store in the ticket's sources directory.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a diary as you work"

**Assistant interpretation:** Create a full ticket-based design package for deploying the app to k3s and synchronizing a Git-backed Obsidian vault into the running server, including external source downloads and reMarkable upload.

**Inferred user intent:** Choose a robust production architecture before writing Kubernetes manifests, and leave an intern-friendly implementation guide for the eventual deployment work.

**Commit (code):** N/A — documentation/research phase.

### What I did

- Read the `ticket-research-docmgr-remarkable`, `diary`, and `defuddle` skills.
- Created ticket `RETRO-DEPLOY-003`.
- Created design doc `design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md`.
- Created diary doc `reference/01-investigation-diary.md`.
- Downloaded external references with Defuddle:
  - `sources/01-kubernetes-git-sync.md`
  - `sources/02-argocd-webhooks.md`
  - `sources/03-flux-gitrepositories.md`
  - `sources/04-stakater-reloader.md`
- Captured local evidence:
  - `sources/05-retro-source-evidence.txt`
  - `sources/06-k3s-source-evidence.txt`
- Updated `tasks.md` with research/design and future implementation tasks.

### Why

The deployment design needs to handle two separate sync loops: application deployment sync (Argo CD/GitOps) and content sync (vault commits). Treating the vault as content rather than Kubernetes manifests changes the right tool: Argo CD should deploy the workload, while a runtime content sync mechanism should keep the vault files fresh.

### What worked

- Defuddle successfully extracted clean Markdown for git-sync, Argo CD webhooks, Flux GitRepository, and Stakater Reloader references.
- The local k3s repo already has clear Argo CD Application and kustomize patterns to copy.
- The app already has an embedded single-binary Dockerfile and an fsnotify watcher, so the deployment can start simple and evolve toward explicit reload hooks.

### What didn't work

N/A so far.

### What I learned

- `kubernetes/git-sync` is the most directly relevant off-the-shelf solution: it is explicitly designed as a Kubernetes sidecar that syncs Git content into a shared volume and publishes updates through an atomic symlink.
- Argo CD webhooks help refresh manifest reconciliation, but they do not directly synchronize arbitrary content into a running Pod.
- Flux `GitRepository` produces in-cluster artifacts and is useful in Flux-native deployments, but this cluster currently uses Argo CD application patterns.

### What was tricky to build

The subtle point is the difference between GitOps for manifests and Git sync for runtime content. Argo CD owns Kubernetes object convergence; it should not be asked to mount a vault's Markdown files into a Pod unless the vault is packaged as manifests/configmaps, which would be a poor fit for a large, frequently changing Obsidian vault.

### What warrants a second pair of eyes

- The exact private Git authentication approach for the vault repo: SSH deploy key vs HTTPS token vs GitHub App.
- Whether the vault repo root is the vault itself or contains the vault in a subdirectory, because git-sync's symlink contract affects the safest mount path.

### What should be done in the future

- Write the full design guide.
- Run docmgr validation.
- Upload the document bundle to reMarkable.

### Code review instructions

Review the sources in the ticket workspace first:

```bash
ls -lh ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/sources
```

### Technical details

Commands run:

```bash
docmgr ticket create-ticket --ticket RETRO-DEPLOY-003 --title "Deploy retro-obsidian-publish to Hetzner k3s with git-synced vault content" --topics retro-obsidian-publish,k3s,gitops,argocd,git-sync,obsidian-vault
defuddle parse https://github.com/kubernetes/git-sync --md -o sources/01-kubernetes-git-sync.md
defuddle parse https://argo-cd.readthedocs.io/en/latest/operator-manual/webhook/ --md -o sources/02-argocd-webhooks.md
defuddle parse https://fluxcd.io/flux/components/source/gitrepositories/ --md -o sources/03-flux-gitrepositories.md
defuddle parse https://github.com/stakater/Reloader --md -o sources/04-stakater-reloader.md
```

## Step 2: Design guide, validation, and reMarkable delivery

Wrote the primary design guide as an intern-oriented technical document. It explains the current application architecture, the existing k3s Argo CD patterns, the external git-sync/Argo CD/Flux/Reloader options, and the recommended design: Argo CD for Kubernetes manifests plus a `git-sync` sidecar for runtime vault content.

I then validated the docmgr ticket and uploaded the design/diary bundle to reMarkable. The first upload attempt included the full external source markdown bundle and failed because Pandoc attempted to fetch or convert remote images/SVGs from those sources. The successful upload includes the design guide and diary, while the downloaded sources remain stored in the ticket workspace.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the requested written deliverable, validate docmgr metadata, and upload to reMarkable.

**Inferred user intent:** Receive a polished handoff package that can be read on reMarkable and used by a new intern to implement the deployment.

**Commit (code):** N/A — documentation/research phase.

### What I did

- Wrote `design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md`.
- Updated `tasks.md` to reflect completed research/design work.
- Related key source files to the design and diary docs with `docmgr doc relate`.
- Updated the ticket changelog.
- Ran `docmgr doctor --ticket RETRO-DEPLOY-003 --stale-after 30`.
- Fixed doctor findings:
  - Added missing vocabulary topics: `argocd`, `git-sync`, `gitops`, `k3s`, `obsidian-vault`, `retro-obsidian-publish`.
  - Added YAML frontmatter to the downloaded source Markdown files.
- Re-ran `docmgr doctor`; it passed.
- Uploaded the design guide + diary bundle to reMarkable at `/ai/2026/05/14/RETRO-DEPLOY-003`.

### Why

The design guide needed to be validated and portable. The reMarkable bundle provides a readable handoff document, while the ticket workspace keeps the raw downloaded sources and local evidence for deeper follow-up.

### What worked

- `docmgr doctor` caught missing source frontmatter and missing vocabulary entries.
- The second reMarkable upload succeeded with the focused design+diary bundle.

### What didn't work

The first reMarkable upload attempted to include all external source Markdown files and failed during Pandoc PDF generation. Exact failure summary:

```text
Error: pandoc failed: [WARNING] Could not fetch resource https://argo-cd.readthedocs.io/en/latest/assets/webhook-config.png ... ConnectionTimeout
[WARNING] Could not convert image ... .svgz: check that rsvg-convert is in path.
Error producing PDF.
! LaTeX Error: Cannot determine size of graphic ... .svgz (no BoundingBox).
```

Resolution: upload only the authored design guide and diary. Keep the full external sources in `sources/` for repository reference.

### What I learned

Defuddled external docs can still include remote image references. Those are useful in source files, but they can break Pandoc-based PDF upload when images time out or require `rsvg-convert`.

### What was tricky to build

The `docmgr doctor` validation scans downloaded Markdown sources too. Because Defuddle outputs plain Markdown without docmgr frontmatter, those files needed source metadata frontmatter before the ticket could pass validation.

### What warrants a second pair of eyes

- The design recommends a reload endpoint and token authentication, but the exact git-sync hook mechanism should be verified against the chosen git-sync image version before implementation.
- The vault repo root is now confirmed to be the vault. The app must resolve `/git/root/current` as a symlink before walking it.

### What should be done in the future

- Implement the app-side reload endpoint and health endpoint.
- Add k3s manifests and test against a non-production vault first.

### Code review instructions

Review:

- `design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md`
- `reference/01-investigation-diary.md`
- `sources/01-kubernetes-git-sync.md`
- `sources/02-argocd-webhooks.md`
- `sources/03-flux-gitrepositories.md`
- `sources/04-stakater-reloader.md`

Validate:

```bash
docmgr doctor --ticket RETRO-DEPLOY-003 --stale-after 30
```

### Technical details

Successful upload command:

```bash
remarquee upload bundle \
  design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md \
  reference/01-investigation-diary.md \
  --name "RETRO DEPLOY 003 K3s Git Sync Design" \
  --remote-dir "/ai/2026/05/14/RETRO-DEPLOY-003" \
  --toc-depth 2 \
  --non-interactive
```

Successful upload output:

```text
OK: uploaded RETRO_DEPLOY_003_K3s_Git_Sync_Design.pdf -> /ai/2026/05/14/RETRO-DEPLOY-003
```

## Step 3: Resolve vault repo layout and Vault-backed Git credential plan

The vault repository root is confirmed to be the vault itself. That changes the git-sync mount path from the earlier conservative `/git/root/current/vault` recommendation to `/git/root/current`, and it makes explicit symlink resolution in the app mandatory before production.

I also checked the k3s repo's Vault patterns. Existing applications use Vault Secrets Operator (`VaultConnection`, `VaultAuth`, `VaultStaticSecret`) for runtime secrets and image-pull secrets. Argo CD's own GitOps repository credential is documented separately as an `argocd` namespace repository Secret currently sourced from `.envrc` (`ARGOCD_TOKEN`), so the vault content sync should use a new dedicated Vault-backed Git credential rather than borrowing Argo CD's credential.

### Prompt Context

**User prompt (verbatim):** "the git repo root is the vault. Also, can we get the github credentials from the 'vault' (as in, the vault installation we run to manage our credentials?) . I think that's where argocd gets its github credentials from?"

**Assistant interpretation:** Update the deployment design with the confirmed vault repository layout and clarify whether Git credentials for git-sync should be sourced from HashiCorp Vault.

**Inferred user intent:** Avoid hardcoding GitHub credentials in manifests and align the vault sync deployment with the cluster's existing secret-management model.

**Commit (code):** N/A — design update.

### What I did

- Inspected k3s VSO patterns in existing apps such as `discord-ui-showcase` and `go-go-host`.
- Read `docs/argocd-private-gitops-repo-secret.md` to verify how Argo CD's private GitOps repo credential is currently documented.
- Updated the design guide:
  - vault root path is now `/git/root/current`.
  - app must call `filepath.EvalSymlinks` before startup/reload walks.
  - git-sync credentials should be materialized from Vault via VSO.
  - do not reuse the Argo CD GitOps repo credential.
  - proposed Vault path: `kv/apps/retro-obsidian-publish/prod/vault-git`.
  - added policy/role/VaultStaticSecret sketches.

### Why

The repo-root-is-vault detail affects correctness: `git-sync` publishes `current` as a symlink, and Go's `filepath.Walk` does not safely traverse a symlink root. The server must resolve the symlink on every full reload so it follows the newest worktree after git-sync flips the symlink.

Credential separation also matters. Argo CD's credential reads the GitOps repo; the runtime sidecar needs a credential for the vault repo. Those should be separate least-privilege credentials.

### What worked

- The k3s repo has a clear VSO convention: service account + `VaultConnection` + `VaultAuth` + `VaultStaticSecret`.
- Existing policies/roles under `vault/policies/kubernetes` and `vault/roles/kubernetes` provide direct templates for the new app.

### What didn't work

N/A.

### What I learned

Argo CD's current repository credential is not represented as a VSO-managed manifest in this repo. The docs say it is a Kubernetes Secret in `argocd`, created from an operator workstation token (`ARGOCD_TOKEN`). Separately, CI GitOps PR tokens can be Vault-backed through GitHub Actions OIDC. For runtime git-sync, the clean pattern is VSO in the app namespace.

### What was tricky to build

The tricky distinction is that there are three GitHub credential use cases:

1. Argo CD reads the GitOps repo.
2. CI opens PRs to the GitOps repo.
3. git-sync reads the vault repo at runtime.

They should not share one credential because their scopes and lifetimes differ.

### What warrants a second pair of eyes

- Confirm the exact vault repo URL and whether a deploy key or fine-grained PAT is preferred.
- Confirm whether git-sync should use SSH (`ssh`, `known_hosts`) or HTTPS (`username`, `password`) in this cluster.

### What should be done in the future

- Add app-side symlink resolution and reload endpoint before writing the final Deployment.
- Seed the Vault path and add the VSO manifests in the k3s repo.

### Code review instructions

Review:

- `design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md`, sections "Vault repo layout and symlink details" and "Phase 3: Add Git credentials from Vault".
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/argocd-private-gitops-repo-secret.md` for Argo CD credential distinction.
- Existing VSO examples under `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/discord-ui-showcase/`.

### Technical details

Recommended Vault path:

```text
kv/apps/retro-obsidian-publish/prod/vault-git
```

Recommended app vault path:

```text
/git/root/current
```

Required app behavior:

```go
resolvedRoot, err := filepath.EvalSymlinks(configuredRoot)
if err != nil {
    return fmt.Errorf("vault checkout is not ready: %w", err)
}
v, err := vault.New(resolvedRoot)
```
