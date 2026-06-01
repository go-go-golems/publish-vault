#!/usr/bin/env bash
set -euo pipefail

# Verify the GitHub App installation token can write to the GitOps repository
# without leaving a lasting branch behind. This script mints an installation
# token from the Vault-stored GitHub App credentials, creates a temporary branch
# with an empty commit, pushes it, then deletes the remote branch.
#
# The installation token is never printed.

: "${GITOPS_APP_SECRET_PATH:=kv/ci/github/retro-obsidian-publish/gitops-pr-app}"
: "${GITOPS_OWNER:=wesen}"
: "${GITOPS_REPO:=2026-03-27--hetzner-k3s}"

tmpdir=$(mktemp -d)
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

secret_json="$tmpdir/secret.json"
key_file="$tmpdir/app-private-key.pem"
vault kv get -format=json "$GITOPS_APP_SECRET_PATH" > "$secret_json"
app_id=$(jq -r '.data.data.app_id' < "$secret_json")
jq -r '.data.data.private_key' < "$secret_json" > "$key_file"
chmod 0600 "$key_file"

base64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }
now=$(date +%s)
iat=$((now - 60))
exp=$((now + 540))
header='{"alg":"RS256","typ":"JWT"}'
payload=$(jq -nc --arg iss "$app_id" --argjson iat "$iat" --argjson exp "$exp" '{iat:$iat, exp:$exp, iss:$iss}')
unsigned="$(printf '%s' "$header" | base64url).$(printf '%s' "$payload" | base64url)"
signature=$(printf '%s' "$unsigned" | openssl dgst -sha256 -sign "$key_file" | base64url)
jwt="$unsigned.$signature"

installation_json="$tmpdir/installation.json"
curl -fsS \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $jwt" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$GITOPS_OWNER/$GITOPS_REPO/installation" \
  > "$installation_json"
installation_id=$(jq -r '.id' < "$installation_json")

token_json="$tmpdir/token.json"
curl -fsS -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $jwt" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/app/installations/$installation_id/access_tokens" \
  -d "$(jq -nc --arg repo "$GITOPS_REPO" '{repositories:[$repo], permissions:{contents:"write", pull_requests:"write"}}')" \
  > "$token_json"
installation_token=$(jq -r '.token' < "$token_json")
expires_at=$(jq -r '.expires_at' < "$token_json")
if [[ -z "$installation_token" || "$installation_token" == "null" ]]; then
  echo "failed to mint installation token" >&2
  exit 1
fi

echo "Installation token minted OK: expires_at=$expires_at"

clone_dir="$tmpdir/repo"
remote="https://x-access-token:${installation_token}@github.com/${GITOPS_OWNER}/${GITOPS_REPO}.git"
git clone --depth 1 "$remote" "$clone_dir" >/dev/null 2>&1
cd "$clone_dir"

git config user.name "github-app-write-verify"
git config user.email "github-app-write-verify@example.invalid"
branch="verify/github-app-token-$(date -u +%Y%m%dT%H%M%SZ)-$$"
git checkout -b "$branch" >/dev/null 2>&1
git commit --allow-empty -m "verify github app gitops write access" >/dev/null

git push origin "$branch" >/dev/null 2>&1
echo "Remote branch push OK: $branch"

git push origin ":$branch" >/dev/null 2>&1
echo "Remote branch cleanup OK: $branch"
