#!/usr/bin/env bash
set -euo pipefail

# Verify the GitHub App credentials in Vault can mint an installation token
# for the GitOps repository. The installation token is never printed.
#
# Requirements: vault, jq, openssl, curl, gh

: "${GITOPS_APP_SECRET_PATH:=kv/ci/github/retro-obsidian-publish/gitops-pr-app}"
: "${GITOPS_OWNER:=wesen}"
: "${GITOPS_REPO:=2026-03-27--hetzner-k3s}"

tmpdir=$(mktemp -d)
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

secret_json="$tmpdir/secret.json"
key_file="$tmpdir/app-private-key.pem"
vault kv get -format=json "$GITOPS_APP_SECRET_PATH" > "$secret_json"
app_id=$(jq -r '.data.data.app_id' < "$secret_json")
jq -r '.data.data.private_key' < "$secret_json" > "$key_file"
chmod 0600 "$key_file"

if [[ -z "$app_id" || "$app_id" == "null" ]]; then
  echo "missing app_id in $GITOPS_APP_SECRET_PATH" >&2
  exit 1
fi
if [[ ! -s "$key_file" ]]; then
  echo "missing private_key in $GITOPS_APP_SECRET_PATH" >&2
  exit 1
fi

echo "Vault secret OK: app_id=$app_id private_key_bytes=$(wc -c < "$key_file")"

base64url() {
  openssl base64 -A | tr '+/' '-_' | tr -d '='
}

now=$(date +%s)
iat=$((now - 60))
exp=$((now + 540))
header='{"alg":"RS256","typ":"JWT"}'
payload=$(jq -nc --arg iss "$app_id" --argjson iat "$iat" --argjson exp "$exp" '{iat:$iat, exp:$exp, iss:$iss}')
unsigned="$(printf '%s' "$header" | base64url).$(printf '%s' "$payload" | base64url)"
signature=$(printf '%s' "$unsigned" | openssl dgst -sha256 -sign "$key_file" | base64url)
jwt="$unsigned.$signature"

app_json="$tmpdir/app.json"
curl -fsS \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $jwt" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/app" \
  > "$app_json"
echo "App auth OK: $(jq -r '.slug + " (id=" + (.id|tostring) + ", owner=" + .owner.login + ")"' < "$app_json")"

installations_json="$tmpdir/installations.json"
curl -fsS \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $jwt" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/app/installations" \
  > "$installations_json"
installation_count=$(jq 'length' < "$installations_json")
if [[ "$installation_count" -eq 0 ]]; then
  echo "GitHub App is authenticated but has no installations." >&2
  echo "Install it on $GITOPS_OWNER/$GITOPS_REPO, then rerun this script." >&2
  echo "Install URL: https://github.com/apps/$(jq -r '.slug' < "$app_json")/installations/new" >&2
  exit 2
fi

installation_json="$tmpdir/installation.json"
if ! curl -fsS \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $jwt" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$GITOPS_OWNER/$GITOPS_REPO/installation" \
  > "$installation_json"; then
  echo "GitHub App is installed somewhere, but not on $GITOPS_OWNER/$GITOPS_REPO (or lacks repository access)." >&2
  jq '[.[] | {id, account:.account.login, target_type, repository_selection}]' < "$installations_json" >&2
  exit 3
fi
installation_id=$(jq -r '.id' < "$installation_json")
account=$(jq -r '.account.login' < "$installation_json")
echo "Installation OK: id=$installation_id account=$account repo=$GITOPS_OWNER/$GITOPS_REPO"

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

export GH_TOKEN="$installation_token"
gh api "repos/$GITOPS_OWNER/$GITOPS_REPO" \
  -q '{full_name:.full_name, private:.private, permissions:.permissions}'

git ls-remote "https://x-access-token:${installation_token}@github.com/${GITOPS_OWNER}/${GITOPS_REPO}.git" HEAD >/dev/null
echo "git clone credentials OK: ls-remote HEAD succeeded"
