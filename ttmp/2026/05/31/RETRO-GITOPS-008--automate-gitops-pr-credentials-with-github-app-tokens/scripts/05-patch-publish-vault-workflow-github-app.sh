#!/usr/bin/env bash
set -euo pipefail

# Patch publish-vault's publish-image workflow to request GitOps PR credentials
# from a GitHub App token source instead of the legacy Vault-stored PAT.

: "${PUBLISH_VAULT_REPO:=/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish}"
workflow="$PUBLISH_VAULT_REPO/.github/workflows/publish-image.yaml"

if [[ ! -f "$workflow" ]]; then
  echo "workflow not found: $workflow" >&2
  exit 1
fi

python3 - "$workflow" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
s = path.read_text()
old = '''      gitops_pr_token_source: vault
      vault_role: retro-obsidian-publish-gitops-pr
      vault_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-token
      tooling_repository: go-go-golems/infra-tooling
      tooling_ref: main
'''
new = '''      gitops_pr_token_source: github_app
      vault_role: retro-obsidian-publish-gitops-pr
      gitops_app_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-app
      gitops_app_owner: wesen
      gitops_app_repositories: 2026-03-27--hetzner-k3s
      tooling_repository: go-go-golems/infra-tooling
      tooling_ref: main
'''
if old not in s:
    if new in s:
        print('workflow already patched')
        raise SystemExit(0)
    raise SystemExit(f'expected block not found in {path}')
path.write_text(s.replace(old, new, 1))
PY

echo "Patched $workflow"
git -C "$PUBLISH_VAULT_REPO" diff -- .github/workflows/publish-image.yaml
