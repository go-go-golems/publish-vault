# Changelog

## 2026-05-31

- Initial workspace created


## 2026-05-31

Stored GitHub App credentials in Vault, added retrace scripts, and found the App still needs installation on wesen/2026-03-27--hetzner-k3s

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/01-store-github-app-secret.sh — Vault storage retrace script
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/02-verify-github-app-secret-and-token.sh — Verification retrace script


## 2026-05-31

Extended Vault policy and patched infra-tooling/publish-vault workflows for GitHub App GitOps PR token source; end-to-end verification remains blocked until the App is installed on the GitOps repo

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/publish-image.yaml — publish-vault now requests github_app token source
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/reference/01-implementation-diary.md — Diary updated with Steps 1-2
- /home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml — Reusable workflow now supports github_app token source


## 2026-05-31

Verified GitHub App installation and write access: token minted, git ls-remote succeeded, temporary GitOps branch push/delete succeeded

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/02-verify-github-app-secret-and-token.sh — Read/token verification script
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/06-verify-github-app-gitops-write-access.sh — Write verification script


## 2026-05-31

Published sha-e61c800 through GitHub App GitOps PR automation; merged PR #97 and verified Argo rollout plus public endpoint health

### Related Files

- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/reference/01-implementation-diary.md — Diary updated with full publish and rollout proof
- /home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/07-check-published-deployment.sh — Deployment verification script


## 2026-05-31

Ticket closed

