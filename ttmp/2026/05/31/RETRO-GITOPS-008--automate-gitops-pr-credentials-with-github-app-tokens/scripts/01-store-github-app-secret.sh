#!/usr/bin/env bash
set -euo pipefail

# Store the GitHub App credentials used to mint short-lived GitOps PR tokens.
# This script does not print the private key.
#
# Usage:
#   GITOPS_APP_ID=3926776 \
#   GITOPS_APP_PRIVATE_KEY_FILE=$HOME/Downloads/wesen-gitops-pr-bot.2026-05-31.private-key.pem \
#     ./01-store-github-app-secret.sh

: "${GITOPS_APP_ID:=3926776}"
: "${GITOPS_APP_PRIVATE_KEY_FILE:=$HOME/Downloads/wesen-gitops-pr-bot.2026-05-31.private-key.pem}"
: "${GITOPS_APP_SECRET_PATH:=kv/ci/github/retro-obsidian-publish/gitops-pr-app}"

if [[ ! -f "$GITOPS_APP_PRIVATE_KEY_FILE" ]]; then
  echo "missing private key file: $GITOPS_APP_PRIVATE_KEY_FILE" >&2
  exit 1
fi

first_line=$(head -1 "$GITOPS_APP_PRIVATE_KEY_FILE")
last_line=$(tail -1 "$GITOPS_APP_PRIVATE_KEY_FILE")
if [[ "$first_line" != "-----BEGIN RSA PRIVATE KEY-----" && "$first_line" != "-----BEGIN PRIVATE KEY-----" ]]; then
  echo "private key does not look like a PEM key: unexpected first line" >&2
  exit 1
fi
if [[ "$last_line" != "-----END RSA PRIVATE KEY-----" && "$last_line" != "-----END PRIVATE KEY-----" ]]; then
  echo "private key does not look like a PEM key: unexpected last line" >&2
  exit 1
fi

vault kv put "$GITOPS_APP_SECRET_PATH" \
  app_id="$GITOPS_APP_ID" \
  private_key="$(cat "$GITOPS_APP_PRIVATE_KEY_FILE")"

echo "Stored GitHub App credentials in Vault path: $GITOPS_APP_SECRET_PATH"
vault kv get -format=json "$GITOPS_APP_SECRET_PATH" \
  | jq '{keys:(.data.data|keys), app_id:.data.data.app_id, private_key_length:(.data.data.private_key|length)}'
