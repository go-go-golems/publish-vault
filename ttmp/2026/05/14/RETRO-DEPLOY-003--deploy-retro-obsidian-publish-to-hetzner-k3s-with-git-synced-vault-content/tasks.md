# Tasks

## Research and design

- [x] Create ticket workspace for k3s deployment and git-synced vault content.
- [x] Download external references with Defuddle into `sources/`.
- [x] Gather repo and cluster evidence into `sources/`.
- [x] Write full design and implementation guide for intern handoff.
- [x] Write investigation diary.
- [x] Relate key source files and update changelog.
- [x] Run `docmgr doctor` and resolve validation issues.
- [x] Upload bundle to reMarkable and verify listing.

## Future implementation phases proposed by the design

- [x] Add production reload API and health endpoints to `retro-obsidian-publish`.
- [x] Add Docker image publication workflow or document manual GHCR publishing.
- [x] Add kustomize deployment under `~/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish`.
- [x] Add Argo CD Application under `gitops/applications/retro-obsidian-publish.yaml`.
- [x] Add git-sync sidecar, SSH credentials from Vault, shared `emptyDir`, service, ingress, and probes.
- [x] Test end-to-end vault update: commit vault change → git-sync pulls → reload webhook → server updates API/search/backlinks.
- [x] Deploy and verify baseline production health at `https://parc.yolo.scapegoat.dev`.
