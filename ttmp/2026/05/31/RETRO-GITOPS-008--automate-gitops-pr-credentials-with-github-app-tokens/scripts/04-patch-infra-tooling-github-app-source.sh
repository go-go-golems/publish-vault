#!/usr/bin/env bash
set -euo pipefail

# Patch infra-tooling's reusable publish workflow to support
# gitops_pr_token_source=github_app.
#
# Usage:
#   INFRA_TOOLING_REPO=/home/manuel/code/wesen/go-go-golems/infra-tooling \
#     ./04-patch-infra-tooling-github-app-source.sh

: "${INFRA_TOOLING_REPO:=/home/manuel/code/wesen/go-go-golems/infra-tooling}"
workflow="$INFRA_TOOLING_REPO/.github/workflows/publish-ghcr-image.yml"

if [[ ! -f "$workflow" ]]; then
  echo "workflow not found: $workflow" >&2
  exit 1
fi

python3 - "$workflow" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
s = path.read_text()

repls = []
repls.append((
'''      vault_secret_field:
        required: false
        type: string
        default: token
''',
'''      vault_secret_field:
        required: false
        type: string
        default: token
      gitops_app_secret_path:
        required: false
        type: string
        default: ""
      gitops_app_id_field:
        required: false
        type: string
        default: app_id
      gitops_app_private_key_field:
        required: false
        type: string
        default: private_key
      gitops_app_owner:
        required: false
        type: string
        default: ""
      gitops_app_repositories:
        required: false
        type: string
        default: ""
'''))

repls.append((
'''          VAULT_SECRET_PATH: ${{ inputs.vault_secret_path }}
          LEGACY_GITOPS_PR_TOKEN: ${{ secrets.GITOPS_PR_TOKEN }}
''',
'''          VAULT_SECRET_PATH: ${{ inputs.vault_secret_path }}
          GITOPS_APP_SECRET_PATH: ${{ inputs.gitops_app_secret_path }}
          GITOPS_APP_OWNER: ${{ inputs.gitops_app_owner }}
          GITOPS_APP_REPOSITORIES: ${{ inputs.gitops_app_repositories }}
          LEGACY_GITOPS_PR_TOKEN: ${{ secrets.GITOPS_PR_TOKEN }}
'''))

repls.append((
'''            secret)
              if [ -z "${LEGACY_GITOPS_PR_TOKEN}" ]; then
                echo "gitops_pr_token_source=secret requires the legacy GITOPS_PR_TOKEN secret."
                exit 1
              fi
              ;;
            *)
''',
'''            secret)
              if [ -z "${LEGACY_GITOPS_PR_TOKEN}" ]; then
                echo "gitops_pr_token_source=secret requires the legacy GITOPS_PR_TOKEN secret."
                exit 1
              fi
              ;;
            github_app)
              if [ -z "${VAULT_ROLE}" ] || [ -z "${GITOPS_APP_SECRET_PATH}" ]; then
                echo "gitops_pr_token_source=github_app requires vault_role and gitops_app_secret_path."
                exit 1
              fi
              if [ -z "${GITOPS_APP_OWNER}" ] || [ -z "${GITOPS_APP_REPOSITORIES}" ]; then
                echo "gitops_pr_token_source=github_app requires gitops_app_owner and gitops_app_repositories."
                exit 1
              fi
              ;;
            *)
'''))

repls.append((
'''      - name: Export legacy GitOps PR token
        if: inputs.gitops_pr_token_source == 'secret'
        shell: bash
        env:
          LEGACY_GITOPS_PR_TOKEN: ${{ secrets.GITOPS_PR_TOKEN }}
        run: |
          set -euo pipefail
          echo "GITOPS_PR_TOKEN=${LEGACY_GITOPS_PR_TOKEN}" >> "$GITHUB_ENV"
          echo "::add-mask::${LEGACY_GITOPS_PR_TOKEN}"

      - name: Open GitOps pull requests for published image
''',
'''      - name: Export legacy GitOps PR token
        if: inputs.gitops_pr_token_source == 'secret'
        shell: bash
        env:
          LEGACY_GITOPS_PR_TOKEN: ${{ secrets.GITOPS_PR_TOKEN }}
        run: |
          set -euo pipefail
          echo "GITOPS_PR_TOKEN=${LEGACY_GITOPS_PR_TOKEN}" >> "$GITHUB_ENV"
          echo "::add-mask::${LEGACY_GITOPS_PR_TOKEN}"

      - name: Read GitHub App credentials from Vault
        if: inputs.gitops_pr_token_source == 'github_app'
        uses: hashicorp/vault-action@v3
        with:
          url: ${{ inputs.vault_addr }}
          method: jwt
          path: ${{ inputs.vault_auth_path }}
          role: ${{ inputs.vault_role }}
          jwtGithubAudience: ${{ inputs.vault_audience }}
          exportToken: true
          secrets: |
            ${{ inputs.gitops_app_secret_path }} ${{ inputs.gitops_app_id_field }} | GITOPS_APP_ID ;
            ${{ inputs.gitops_app_secret_path }} ${{ inputs.gitops_app_private_key_field }} | GITOPS_APP_PRIVATE_KEY

      - name: Mint GitHub App token for GitOps repository
        if: inputs.gitops_pr_token_source == 'github_app'
        id: gitops-app-token
        uses: actions/create-github-app-token@v2
        with:
          app-id: ${{ env.GITOPS_APP_ID }}
          private-key: ${{ env.GITOPS_APP_PRIVATE_KEY }}
          owner: ${{ inputs.gitops_app_owner }}
          repositories: ${{ inputs.gitops_app_repositories }}

      - name: Export GitHub App token as GitOps PR token
        if: inputs.gitops_pr_token_source == 'github_app'
        shell: bash
        run: |
          set -euo pipefail
          echo "GITOPS_PR_TOKEN=${{ steps.gitops-app-token.outputs.token }}" >> "$GITHUB_ENV"
          echo "::add-mask::${{ steps.gitops-app-token.outputs.token }}"

      - name: Open GitOps pull requests for published image
'''))

for old, new in repls:
    if old not in s:
        if new in s:
            continue
        raise SystemExit(f"expected block not found in {path}:\n{old}")
    s = s.replace(old, new, 1)

path.write_text(s)
PY

echo "Patched $workflow"
git -C "$INFRA_TOOLING_REPO" diff -- .github/workflows/publish-ghcr-image.yml
