# GitHub Actions role for the retro-obsidian-publish release workflow.
#
# This policy intentionally reads only the GitOps PR credential used by
# scripts/open_gitops_pr.py. It does not grant app runtime secret access,
# auth-management access, or broad KV traversal.
path "kv/data/ci/github/retro-obsidian-publish/gitops-pr-token" {
  capabilities = ["read"]
}

path "auth/token/lookup-self" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "auth/token/revoke-self" {
  capabilities = ["update"]
}
