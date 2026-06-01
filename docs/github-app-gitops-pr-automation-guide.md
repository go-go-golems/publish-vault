---
title: "GitHub App Tokens for Automated GitOps PRs"
date: 2026-06-01
status: draft
repos:
  source: go-go-golems/publish-vault
  gitops: wesen/2026-03-27--hetzner-k3s
  tooling: go-go-golems/infra-tooling
---

# GitHub App Tokens for Automated GitOps PRs

## Executive summary

The `publish-vault` deployment pipeline currently publishes the Docker image successfully, then fails when the GitOps PR action tries to clone the GitOps repository:

```text
remote: Invalid username or token. Password authentication is not supported for Git operations.
fatal: Authentication failed for 'https://github.com/wesen/2026-03-27--hetzner-k3s.git/'
```

The immediate cause is the Vault-stored `GITOPS_PR_TOKEN` at:

```text
kv/ci/github/retro-obsidian-publish/gitops-pr-token
```

That token is a personal access token and can expire or be revoked. The workflow already proved that the image build and publish step works; the broken part is only the cross-repository GitOps PR credential.

The durable fix is to stop storing a long-lived personal access token for GitOps PR creation. Instead, create a GitHub App with narrow permissions on the GitOps repository, store the App credentials in Vault, and mint a short-lived installation token during each workflow run.

## Current deployment path

The current flow is:

1. Source repo: `go-go-golems/publish-vault`.
2. Workflow: `.github/workflows/publish-image.yaml`.
3. Reusable workflow: `go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@main`.
4. Published image: `ghcr.io/go-go-golems/publish-vault:sha-<commit>`.
5. GitOps target config: `deploy/gitops-targets.json`.
6. GitOps repo: `wesen/2026-03-27--hetzner-k3s`.
7. Manifest updated by PR: `gitops/kustomize/retro-obsidian-publish/deployment.yaml`.
8. Argo CD syncs `retro-obsidian-publish` to the cluster.

The workflow currently says:

```yaml
gitops_pr_token_source: vault
vault_role: retro-obsidian-publish-gitops-pr
vault_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-token
```

The Vault role exists and is bound to trusted `main` pushes from `go-go-golems/publish-vault`, but the secret value is an expired/revoked PAT.

## Target design

Replace this:

```text
GitHub Actions -> Vault -> long-lived PAT -> clone/push GitOps repo
```

with this:

```text
GitHub Actions
  -> Vault OIDC login
  -> read GitHub App app_id + private_key
  -> mint short-lived GitHub App installation token
  -> clone/push GitOps repo
  -> open PR
```

Benefits:

- No personal account dependency.
- No PAT expiry surprise.
- Token is minted per workflow run and expires automatically.
- Permissions are scoped to one repository: `wesen/2026-03-27--hetzner-k3s`.
- Existing `open-gitops-pr` code can keep using `GITOPS_PR_TOKEN`; only the token source changes.

## Step 1: Create the GitHub App

Create a GitHub App under the account or organization that should own deployment automation. A suggested name:

```text
wesen-gitops-pr-bot
```

Recommended settings:

- Homepage URL: any internal project URL, for example the GitOps repo URL.
- Webhook: disabled unless you need it for something else.
- Repository permissions:
  - Contents: Read and write
  - Pull requests: Read and write
  - Metadata: Read-only (automatic)
- Organization permissions: none.
- Account permissions: none.

Install the App only on:

```text
wesen/2026-03-27--hetzner-k3s
```

Generate a private key and download the `.pem` file.

Record the App ID from the App settings page. Do not commit the private key.

## Step 2: Store GitHub App credentials in Vault

Use a new Vault path so the old PAT path can be left alone during migration:

```text
kv/ci/github/retro-obsidian-publish/gitops-pr-app
```

Store the App ID and private key:

```bash
export GITOPS_APP_ID="123456"
export GITOPS_APP_PRIVATE_KEY_FILE="$HOME/Downloads/wesen-gitops-pr-bot.private-key.pem"

vault kv put \
  kv/ci/github/retro-obsidian-publish/gitops-pr-app \
  app_id="$GITOPS_APP_ID" \
  private_key=@"$GITOPS_APP_PRIVATE_KEY_FILE"
```

If your Vault CLI version does not accept `@file` syntax, use:

```bash
vault kv put \
  kv/ci/github/retro-obsidian-publish/gitops-pr-app \
  app_id="$GITOPS_APP_ID" \
  private_key="$(cat "$GITOPS_APP_PRIVATE_KEY_FILE")"
```

Verify the secret shape without printing the private key:

```bash
vault kv get -format=json \
  kv/ci/github/retro-obsidian-publish/gitops-pr-app \
  | jq '{keys:(.data.data|keys), app_id:.data.data.app_id, private_key_length:(.data.data.private_key|length)}'
```

Expected output should include `app_id` and `private_key`, with a non-zero `private_key_length`.

## Step 3: Extend the Vault policy for the workflow role

The existing Vault role is:

```text
auth/github-actions/role/retro-obsidian-publish-gitops-pr
```

It currently reads only:

```text
kv/data/ci/github/retro-obsidian-publish/gitops-pr-token
```

Add read access to:

```text
kv/data/ci/github/retro-obsidian-publish/gitops-pr-app
```

If maintaining this through Terraform in:

```text
/home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s/main.tf
```

extend the `retro-obsidian-publish-gitops-pr` role data model so its generated policy includes both paths, or add a dedicated policy stanza for this role.

The final policy should include:

```hcl
path "kv/data/ci/github/retro-obsidian-publish/gitops-pr-app" {
  capabilities = ["read"]
}
```

Keep the existing token path temporarily during migration so old workflows still fail in an obvious, reversible way rather than losing access unexpectedly.

Apply the Vault/Terraform change, then verify:

```bash
vault policy read gha-retro-obsidian-publish-gitops-pr \
  | grep -A3 'gitops-pr-app'
```

## Step 4: Add `github_app` support to the reusable workflow

Repository:

```text
go-go-golems/infra-tooling
```

Workflow:

```text
.github/workflows/publish-ghcr-image.yml
```

Add a new token source:

```yaml
gitops_pr_token_source: github_app
```

Add new inputs:

```yaml
gitops_app_secret_path:
  description: Vault KV v2 data path containing app_id and private_key
  required: false
  type: string
gitops_app_id_field:
  description: Field containing the GitHub App ID
  required: false
  default: app_id
  type: string
gitops_app_private_key_field:
  description: Field containing the GitHub App private key
  required: false
  default: private_key
  type: string
gitops_app_owner:
  description: Owner/account where the GitHub App is installed
  required: false
  type: string
gitops_app_repositories:
  description: Comma or newline separated repositories for the installation token
  required: false
  type: string
```

Update credential validation:

```bash
case "${GITOPS_PR_TOKEN_SOURCE}" in
  vault)
    # existing PAT-from-Vault path
    ;;
  secret)
    # existing legacy secret path
    ;;
  github_app)
    if [ -z "${VAULT_ROLE}" ] || [ -z "${GITOPS_APP_SECRET_PATH}" ]; then
      echo "gitops_pr_token_source=github_app requires vault_role and gitops_app_secret_path."
      exit 1
    fi
    ;;
  *)
    echo "unsupported gitops_pr_token_source: ${GITOPS_PR_TOKEN_SOURCE}"
    exit 1
    ;;
esac
```

Read the App credentials from Vault:

```yaml
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
```

Mint the installation token:

```yaml
- name: Mint GitHub App token for GitOps repository
  if: inputs.gitops_pr_token_source == 'github_app'
  id: gitops-app-token
  uses: actions/create-github-app-token@v2
  with:
    app-id: ${{ env.GITOPS_APP_ID }}
    private-key: ${{ env.GITOPS_APP_PRIVATE_KEY }}
    owner: ${{ inputs.gitops_app_owner }}
    repositories: ${{ inputs.gitops_app_repositories }}
```

Export it using the same environment variable consumed by the existing `open-gitops-pr` action:

```yaml
- name: Export GitHub App token as GitOps PR token
  if: inputs.gitops_pr_token_source == 'github_app'
  shell: bash
  run: |
    echo "GITOPS_PR_TOKEN=${{ steps.gitops-app-token.outputs.token }}" >> "$GITHUB_ENV"
    echo "::add-mask::${{ steps.gitops-app-token.outputs.token }}"
```

Leave the `Open GitOps pull requests for published image` step unchanged. It already passes through to the Docker action, whose entrypoint exports `GH_TOKEN` from `GITOPS_PR_TOKEN`.

## Step 5: Update `publish-vault` workflow to use the GitHub App

Repository:

```text
go-go-golems/publish-vault
```

File:

```text
.github/workflows/publish-image.yaml
```

Replace the current PAT-from-Vault settings:

```yaml
gitops_pr_token_source: vault
vault_role: retro-obsidian-publish-gitops-pr
vault_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-token
```

with:

```yaml
gitops_pr_token_source: github_app
vault_role: retro-obsidian-publish-gitops-pr
gitops_app_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-app
gitops_app_owner: wesen
gitops_app_repositories: 2026-03-27--hetzner-k3s
```

Keep:

```yaml
permissions:
  contents: read
  packages: write
  pull-requests: write
  id-token: write
```

The important permission is `id-token: write`; it lets GitHub Actions authenticate to Vault using OIDC. The GitHub App token then handles cross-repo GitOps write access.

## Step 6: Test safely

### Pull request test

Open a PR in `go-go-golems/publish-vault` that changes only workflow/tooling references if needed.

For PRs, the existing workflow should still avoid pushing images and opening GitOps PRs because it uses:

```yaml
push_image: ${{ github.event_name != 'pull_request' }}
open_gitops_pr: ${{ github.event_name != 'pull_request' && github.ref == 'refs/heads/main' }}
```

So the first full proof happens on a trusted `main` push.

### Main push test

After merging to `main`, watch:

```bash
gh run list --repo go-go-golems/publish-vault --workflow publish-image --limit 5
```

The expected successful path is:

1. Build and push `ghcr.io/go-go-golems/publish-vault:sha-<commit>`.
2. OIDC login to Vault succeeds.
3. GitHub App credentials are read from Vault.
4. `actions/create-github-app-token` mints a token.
5. `open-gitops-pr` clones `wesen/2026-03-27--hetzner-k3s`.
6. A PR is opened that updates only:

```text
gitops/kustomize/retro-obsidian-publish/deployment.yaml
```

### Validate the PR

The PR should change only the app container image line, for example:

```diff
- image: ghcr.io/go-go-golems/publish-vault:sha-4d34bcf
+ image: ghcr.io/go-go-golems/publish-vault:sha-0a63a32
```

Merge it, then Argo CD should reconcile the `retro-obsidian-publish` application.

## Step 7: Verify deployment

Use the k3s kubeconfig from the GitOps repo if needed:

```bash
cd /home/manuel/code/wesen/2026-03-27--hetzner-k3s
export KUBECONFIG=$PWD/kubeconfig-91.98.46.169.yaml

kubectl -n argocd get application retro-obsidian-publish
kubectl -n retro-obsidian-publish rollout status deploy/retro-obsidian-publish
kubectl -n retro-obsidian-publish get pods
kubectl -n retro-obsidian-publish get deploy retro-obsidian-publish \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="app")].image}{"\n"}'
```

Then smoke test the public endpoint:

```bash
curl -I https://parc.yolo.scapegoat.dev/
curl -s https://parc.yolo.scapegoat.dev/api/healthz | jq .
```

## Rollback plan

If GitHub App token minting fails:

1. Do not merge any GitOps PR.
2. Re-run the workflow after fixing the GitHub App installation or Vault secret.
3. If production is already updated and needs rollback, revert the GitOps PR or manually set the previous image tag in:

```text
/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/retro-obsidian-publish/deployment.yaml
```

Then commit and push to the GitOps repo; Argo CD will roll back.

## Clean-up after migration

After one or two successful releases using the GitHub App flow:

1. Delete or archive the old PAT in GitHub settings.
2. Remove the old Vault secret if no workflows use it:

```bash
vault kv metadata delete kv/ci/github/retro-obsidian-publish/gitops-pr-token
```

3. Remove the old policy stanza for:

```text
kv/data/ci/github/retro-obsidian-publish/gitops-pr-token
```

4. Keep the GitHub App private key rotation procedure documented. Rotating the App key is simple: generate a new private key, update Vault, verify one release, then delete the old private key in GitHub.

## Troubleshooting

### Vault login fails

Check the role binding:

```bash
vault read -format=json auth/github-actions/role/retro-obsidian-publish-gitops-pr \
  | jq '.data.bound_claims'
```

It should match trusted main pushes from:

```text
go-go-golems/publish-vault
```

### Vault secret read fails

Check policy includes the app secret path:

```bash
vault policy read gha-retro-obsidian-publish-gitops-pr
```

Look for:

```hcl
path "kv/data/ci/github/retro-obsidian-publish/gitops-pr-app" {
  capabilities = ["read"]
}
```

### GitHub App token minting fails

Common causes:

- App ID is wrong.
- Private key was pasted incorrectly or line endings were damaged.
- App is not installed on `wesen/2026-03-27--hetzner-k3s`.
- `owner` is wrong; it should be `wesen` for this GitOps repo.
- `repositories` should be `2026-03-27--hetzner-k3s`, not the full `wesen/...` string.

### Clone succeeds but PR creation fails

The GitHub App likely has Contents write but not Pull requests write. Update GitHub App permissions and reinstall/accept the permission change.

### Image PR points at wrong repository

Check `deploy/gitops-targets.json` in `publish-vault`:

```json
{
  "gitops_repo": "wesen/2026-03-27--hetzner-k3s",
  "manifest_path": "gitops/kustomize/retro-obsidian-publish/deployment.yaml",
  "container_name": "app"
}
```

## Minimal checklist

- [ ] Create GitHub App with Contents RW and Pull requests RW.
- [ ] Install App only on `wesen/2026-03-27--hetzner-k3s`.
- [ ] Store `app_id` and `private_key` in Vault at `kv/ci/github/retro-obsidian-publish/gitops-pr-app`.
- [ ] Extend Vault policy `gha-retro-obsidian-publish-gitops-pr` to read the app secret path.
- [ ] Add `gitops_pr_token_source: github_app` support to `go-go-golems/infra-tooling` reusable workflow.
- [ ] Update `go-go-golems/publish-vault/.github/workflows/publish-image.yaml` to use `github_app`.
- [ ] Merge to main and verify a GitOps PR is created.
- [ ] Merge the GitOps PR and verify Argo CD rollout.
- [ ] Remove the expired PAT path after successful migration.
