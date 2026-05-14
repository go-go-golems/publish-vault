# Changelog

## 2026-05-14

- Initial workspace created


## 2026-05-14

Created deployment design package, downloaded external references, gathered source evidence, and drafted git-sync sidecar architecture.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md — Primary intern-ready deployment guide
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-05-14

Validated ticket with docmgr doctor and uploaded design/diary bundle to reMarkable at /ai/2026/05/14/RETRO-DEPLOY-003.

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/design-doc/01-k3s-deployment-and-git-synced-vault-design-guide.md — Uploaded design guide
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/14/RETRO-DEPLOY-003--deploy-retro-obsidian-publish-to-hetzner-k3s-with-git-synced-vault-content/reference/01-investigation-diary.md — Uploaded diary and recorded PDF failure/success



## 2026-05-14 - Deployed production app to k3s

- Implemented app-side health/reload/symlink-aware runtime support and pushed GHCR image `sha-b3f93bb`.
- Added Argo CD/kustomize deployment for `retro-obsidian-publish` with git-sync vault content, VSO-managed git credentials, VSO-managed GHCR image pull secret, service, ingress, and TLS.
- Fixed rollout failures: git-sync secret permissions, private GHCR image pull auth, and app startup/reload probe timing.
- Verified live site at `https://parc.yolo.scapegoat.dev` and health endpoint reporting 513 notes from `/git/root/current`.


## 2026-05-14 - Fixed homepage/frontmatter bug and verified git-sync update

- Fixed nested YAML frontmatter serialization by recursively normalizing YAML maps before JSON responses.
- Improved homepage note selection so the deployed vault prefers `projects/00-project-index-repos-and-concepts` instead of nested `sources/index` notes.
- Deployed image `ghcr.io/go-go-golems/retro-obsidian-publish:sha-6c22a66` through Argo CD.
- Committed and pushed new go-go-parc article `e468e03`; git-sync pulled it and the app reloaded from 513 to 516 notes.
