# Tasks

## TODO

- [x] Store GitHub App credentials in Vault
- [x] Verify GitHub App installation token can access GitOps repo
- [x] Install GitHub App on wesen/2026-03-27--hetzner-k3s
- [x] Extend Vault policy so CI role can read GitHub App credentials
- [x] Patch infra-tooling reusable workflow to support gitops_pr_token_source=github_app
- [x] Patch publish-vault workflow to use GitHub App token source
- [x] Verify GitHub App installation token can push and clean up a temporary GitOps branch
- [x] Publish image through GitOps PR and verify Argo rollout
