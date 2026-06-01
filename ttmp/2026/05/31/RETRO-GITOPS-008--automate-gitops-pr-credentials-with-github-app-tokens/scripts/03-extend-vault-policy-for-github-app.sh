#!/usr/bin/env bash
set -euo pipefail

# Extend the live Vault policy used by the publish-vault GitOps PR workflow so
# it can read GitHub App credentials in addition to the legacy PAT secret.
# This script is intentionally idempotent and stores a timestamped policy
# backup in the ticket's sources/ directory.

: "${VAULT_POLICY_NAME:=gha-retro-obsidian-publish-gitops-pr}"
: "${GITOPS_APP_DATA_PATH:=kv/data/ci/github/retro-obsidian-publish/gitops-pr-app}"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ticket_dir=$(cd -- "$script_dir/.." && pwd)
backup_dir="$ticket_dir/sources/vault-policy-backups"
mkdir -p "$backup_dir"
backup_file="$backup_dir/${VAULT_POLICY_NAME}-$(date -u +%Y%m%dT%H%M%SZ).hcl"
current_file=$(mktemp)
updated_file=$(mktemp)
cleanup() { rm -f "$current_file" "$updated_file"; }
trap cleanup EXIT

vault policy read "$VAULT_POLICY_NAME" > "$current_file"
cp "$current_file" "$backup_file"

if grep -Fq "path \"$GITOPS_APP_DATA_PATH\"" "$current_file"; then
  echo "Policy already includes $GITOPS_APP_DATA_PATH"
  echo "Backup written to $backup_file"
  exit 0
fi

cat "$current_file" > "$updated_file"
cat >> "$updated_file" <<EOF

# GitHub App credentials for minting short-lived GitOps PR installation tokens.
path "$GITOPS_APP_DATA_PATH" {
  capabilities = ["read"]
}
EOF

vault policy write "$VAULT_POLICY_NAME" "$updated_file"
echo "Updated Vault policy: $VAULT_POLICY_NAME"
echo "Backup written to $backup_file"
vault policy read "$VAULT_POLICY_NAME" | grep -A3 -F "path \"$GITOPS_APP_DATA_PATH\""
