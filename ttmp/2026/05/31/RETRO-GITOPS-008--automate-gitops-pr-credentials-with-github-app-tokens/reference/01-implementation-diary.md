---
Title: Implementation diary
Ticket: RETRO-GITOPS-008
Status: active
Topics:
    - gitops
    - deployment
    - github-actions
    - vault
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml
      Note: Reusable workflow patched to mint GitHub App installation tokens
    - Path: .github/workflows/publish-image.yaml
      Note: Source workflow switched to gitops_pr_token_source=github_app
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/reference/01-implementation-diary.md
      Note: Chronological diary for the GitHub App GitOps PR migration
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/01-store-github-app-secret.sh
      Note: Retraceable script for storing GitHub App credentials in Vault
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/02-verify-github-app-secret-and-token.sh
      Note: Retraceable script for verifying Vault-stored GitHub App credentials and installation-token minting
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/03-extend-vault-policy-for-github-app.sh
      Note: Retraceable script that extended live Vault policy for the GitHub App secret path
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/04-patch-infra-tooling-github-app-source.sh
      Note: Retraceable script that patched infra-tooling reusable workflow
    - Path: ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/05-patch-publish-vault-workflow-github-app.sh
      Note: Retraceable script that patched publish-vault workflow
ExternalSources: []
Summary: Chronological implementation diary for replacing expiring GitOps PR PATs with GitHub App installation tokens.
LastUpdated: 2026-05-31T23:08:00-04:00
WhatFor: Track GitHub App GitOps PR automation implementation steps, failures, scripts, and validation commands.
WhenToUse: Use when resuming RETRO-GITOPS-008 or auditing how GitHub App credentials were stored and verified.
---



# Diary

## Goal

Track the migration from an expiring Vault-stored personal access token to GitHub App installation tokens for automated GitOps pull requests from `go-go-golems/publish-vault` into `wesen/2026-03-27--hetzner-k3s`.

## Step 1: Store GitHub App credentials and verify installation status

This step captured the new GitHub App identity and private key in Vault, then converted the one-off shell commands into reusable scripts under the ticket workspace. The storage step succeeded: Vault now contains `app_id` and `private_key` fields at the intended app credential path without printing the private key to logs.

The first verification pass proved that the private key matches the GitHub App (`wesen-gitops-pr-bot`, app id `3926776`), but also found the next blocker: the App currently has no installations. Until the App is installed on `wesen/2026-03-27--hetzner-k3s`, it cannot mint an installation token for the GitOps repository.

### Prompt Context

**User prompt (verbatim):** "app id: 3926776 key: ~/Downloads/wesen-gitops-pr-bot.2026-05-31.private-key.pem .

Keep a diary as you work"

**Assistant interpretation:** Store the supplied GitHub App private key in Vault, continue the GitHub App GitOps PR automation work, and start a chronological diary.

**Inferred user intent:** Replace the expired GitOps PR PAT with durable GitHub App automation while preserving enough execution history to audit or resume later.

**User prompt (verbatim):** "save all scripts you use in the scripts/ folder of the ticket, so that we can retrace your steps later. 

continue"

**Assistant interpretation:** Create a docmgr ticket workspace, save all operational scripts in the ticket `scripts/` directory, and continue verification from reproducible scripts instead of ad-hoc shell history.

**Inferred user intent:** Make the deployment-credential migration reproducible and reviewable by storing the exact operator scripts next to the ticket documentation.

**Commit (code):** N/A — no code commit yet for this step.

### What I did

- Created docmgr ticket `RETRO-GITOPS-008` with title `Automate GitOps PR credentials with GitHub App tokens`.
- Added diary document `ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/reference/01-implementation-diary.md`.
- Verified the local PEM file exists and looks like a private key:
  - Path: `~/Downloads/wesen-gitops-pr-bot.2026-05-31.private-key.pem`
  - Size: `1679` bytes
  - First line: `-----BEGIN RSA PRIVATE KEY-----`
  - Last line: `-----END RSA PRIVATE KEY-----`
- Stored the GitHub App credentials in Vault:
  - Vault path: `kv/ci/github/retro-obsidian-publish/gitops-pr-app`
  - `app_id`: `3926776`
  - `private_key`: contents of the PEM file
- Verified the Vault secret shape:
  - keys: `app_id`, `private_key`
  - `private_key_length`: `1678` via Vault JSON output
- Created executable retrace scripts:
  - `scripts/01-store-github-app-secret.sh`
  - `scripts/02-verify-github-app-secret-and-token.sh`
- Ran `scripts/02-verify-github-app-secret-and-token.sh`.
- Marked ticket task `Store GitHub App credentials in Vault` complete.

### Why

- The old `kv/ci/github/retro-obsidian-publish/gitops-pr-token` value was rejected by GitHub with `Bad credentials (HTTP 401)`.
- A GitHub App private key plus app id lets CI mint short-lived installation tokens instead of depending on a long-lived PAT.
- Ticket scripts make the sensitive process reproducible without storing secrets in Git.

### What worked

- Vault accepted the GitHub App credentials at `kv/ci/github/retro-obsidian-publish/gitops-pr-app`.
- The verification script successfully generated a GitHub App JWT from the Vault-stored private key.
- GitHub accepted the JWT and returned App metadata:

```text
App auth OK: wesen-gitops-pr-bot (id=3926776, owner=wesen)
```

### What didn't work

- The verification script could not mint a repository installation token because the App has no installations yet:

```text
Vault secret OK: app_id=3926776 private_key_bytes=1679
App auth OK: wesen-gitops-pr-bot (id=3926776, owner=wesen)
GitHub App is authenticated but has no installations.
Install it on wesen/2026-03-27--hetzner-k3s, then rerun this script.
Install URL: https://github.com/apps/wesen-gitops-pr-bot/installations/new
```

- Earlier, before improving the verification script, the direct repository installation endpoint failed with:

```text
curl: (22) The requested URL returned error: 404
```

That 404 was ambiguous, so I updated the script to first verify `/app`, then list `/app/installations`, and then produce a clear installation-needed message.

### What I learned

- The PEM key and app id are valid for the GitHub App.
- The missing piece is not Vault storage or key validity; it is GitHub App installation on the GitOps repository.
- GitHub returns an empty `/app/installations` list for this App right now, so no installation token can be minted yet.

### What was tricky to build

- The verification had to avoid printing secrets while still proving that the key works. The script writes the private key to a temporary `0600` file, generates a short-lived JWT with `openssl`, and only prints non-sensitive metadata.
- The first repository-installation check returned a generic `404`, which could mean wrong repo, missing installation, or insufficient installation scope. Querying `/app` and `/app/installations` first made the failure actionable.
- The local environment does not have Python `cryptography`, so the verification script uses standard command-line tools (`openssl`, `curl`, `jq`, `gh`) instead of a Python JWT library.

### What warrants a second pair of eyes

- Confirm the GitHub App should be installed only on `wesen/2026-03-27--hetzner-k3s` and not broadly across an organization.
- Confirm the App permissions are exactly `Contents: Read and write` and `Pull requests: Read and write`.
- Confirm whether the Vault role/policy should be updated via the Terraform Vault repo or via committed policy files in `2026-03-27--hetzner-k3s`; live Vault already has the `retro-obsidian-publish-gitops-pr` role, but source-of-truth alignment needs care.

### What should be done in the future

- Install `wesen-gitops-pr-bot` on `wesen/2026-03-27--hetzner-k3s` using:

```text
https://github.com/apps/wesen-gitops-pr-bot/installations/new
```

- Rerun:

```bash
ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/02-verify-github-app-secret-and-token.sh
```

- Extend the Vault policy to allow the CI role to read `kv/data/ci/github/retro-obsidian-publish/gitops-pr-app`.
- Update `infra-tooling` to support `gitops_pr_token_source: github_app`.
- Update `publish-vault/.github/workflows/publish-image.yaml` to use the GitHub App token source.

### Code review instructions

- Start with the scripts:
  - `ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/01-store-github-app-secret.sh`
  - `ttmp/2026/05/31/RETRO-GITOPS-008--automate-gitops-pr-credentials-with-github-app-tokens/scripts/02-verify-github-app-secret-and-token.sh`
- Validate that scripts do not print the private key or installation token.
- Rerun the verification script only after the GitHub App is installed on the GitOps repository.

### Technical details

Current Vault secret verification command:

```bash
vault kv get -format=json kv/ci/github/retro-obsidian-publish/gitops-pr-app \
  | jq '{keys:(.data.data|keys), app_id:.data.data.app_id, private_key_length:(.data.data.private_key|length)}'
```

Observed output:

```json
{
  "keys": [
    "app_id",
    "private_key"
  ],
  "app_id": "3926776",
  "private_key_length": 1678
}
```

GitHub App install URL:

```text
https://github.com/apps/wesen-gitops-pr-bot/installations/new
```

## Step 2: Wire GitHub App token source into workflows

This step moved from credential storage to workflow integration. I added a live Vault policy stanza so the existing CI Vault role can read the GitHub App credential path, then patched the reusable `infra-tooling` publish workflow to support a new `gitops_pr_token_source: github_app` mode.

I also patched the `publish-vault` workflow to request that new token source. The code path is ready for CI once the GitHub App is installed on `wesen/2026-03-27--hetzner-k3s` and the `infra-tooling` workflow change is pushed to the `main` ref used by `publish-vault`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing the GitHub App replacement for the expired GitOps PR PAT, keeping scripts and diary entries in the ticket.

**Inferred user intent:** Reach a working deployment path where GitHub Actions can open GitOps PRs using short-lived GitHub App installation tokens.

**Commit (code):** `d066320` in `/home/manuel/code/wesen/go-go-golems/infra-tooling` — `feat(gitops): support GitHub App token source for image publish PRs`

**Commit (code/docs):** `4dbdd96` in `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish` — `feat(ci): use GitHub App credentials for GitOps PR automation`

### What I did

- Added retrace script:
  - `scripts/03-extend-vault-policy-for-github-app.sh`
- Ran it to update the live Vault policy `gha-retro-obsidian-publish-gitops-pr`.
- The script wrote a backup of the previous policy to:
  - `sources/vault-policy-backups/gha-retro-obsidian-publish-gitops-pr-20260601T030831Z.hcl`
- Confirmed the live Vault policy now includes:

```hcl
path "kv/data/ci/github/retro-obsidian-publish/gitops-pr-app" {
  capabilities = ["read"]
}
```

- Added retrace script:
  - `scripts/04-patch-infra-tooling-github-app-source.sh`
- Ran it against:
  - `/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml`
- The reusable workflow now accepts:
  - `gitops_app_secret_path`
  - `gitops_app_id_field`
  - `gitops_app_private_key_field`
  - `gitops_app_owner`
  - `gitops_app_repositories`
- The reusable workflow now validates `gitops_pr_token_source=github_app`, reads GitHub App credentials from Vault, calls `actions/create-github-app-token@v2`, and exports the minted token as `GITOPS_PR_TOKEN` for the existing `open-gitops-pr` action.
- Added retrace script:
  - `scripts/05-patch-publish-vault-workflow-github-app.sh`
- Ran it against:
  - `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/publish-image.yaml`
- The `publish-vault` workflow now uses:

```yaml
gitops_pr_token_source: github_app
vault_role: retro-obsidian-publish-gitops-pr
gitops_app_secret_path: kv/data/ci/github/retro-obsidian-publish/gitops-pr-app
gitops_app_owner: wesen
gitops_app_repositories: 2026-03-27--hetzner-k3s
```

- Validated both workflow YAML files with PyYAML.
- Ran `git diff --check` in both changed repositories.
- Marked ticket tasks complete for:
  - extending the Vault policy,
  - patching `infra-tooling`,
  - patching `publish-vault`.

### Why

- `publish-vault` calls `go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml@main`, so the reusable workflow must know how to mint and export a GitHub App token before the source repo can use it.
- The existing `open-gitops-pr` action already consumes `GITOPS_PR_TOKEN`; reusing that environment variable keeps the lower-level action unchanged.
- Updating the live Vault policy now removes one future blocker once CI tries to read the GitHub App secret.

### What worked

- The Vault policy update succeeded:

```text
Success! Uploaded policy: gha-retro-obsidian-publish-gitops-pr
Updated Vault policy: gha-retro-obsidian-publish-gitops-pr
path "kv/data/ci/github/retro-obsidian-publish/gitops-pr-app" {
  capabilities = ["read"]
}
```

- The reusable workflow patch applied cleanly and `git diff --check` did not report whitespace errors.
- The source workflow patch applied cleanly and `git diff --check` did not report whitespace errors.
- Both YAML files parsed successfully with `yaml.safe_load`.

### What didn't work

- Full end-to-end token minting is still blocked by GitHub App installation status from Step 1:

```text
GitHub App is authenticated but has no installations.
Install it on wesen/2026-03-27--hetzner-k3s, then rerun this script.
Install URL: https://github.com/apps/wesen-gitops-pr-bot/installations/new
```

- The live Vault role/policy existed before this ticket, but the source-of-truth location for that exact role is not obvious in the checked-out repos. I updated the live policy via a recorded script and backup, but source-of-truth alignment still needs follow-up.

### What I learned

- The workflow migration can be additive: `vault` and `secret` token sources remain in `infra-tooling`, while `github_app` becomes a new third option.
- No changes are needed in `actions/open-gitops-pr`; it already accepts `GITOPS_PR_TOKEN`/`GH_TOKEN` and performs the clone/PR steps.
- The `publish-vault` workflow can switch to GitHub App mode with only `with:` input changes once `infra-tooling@main` supports the new inputs.

### What was tricky to build

- The reusable workflow has to support multiple credential modes without breaking existing callers. The validation step now has separate branches for `vault`, `secret`, and `github_app`.
- Vault App credentials have two fields (`app_id`, `private_key`), so the reusable workflow needs a new `vault-action` read step rather than reusing the single-field PAT read.
- The source workflow points to `tooling_ref: main`. This means the `infra-tooling` change must be merged/pushed before a `publish-vault` main workflow run can succeed with `github_app` mode.

### What warrants a second pair of eyes

- Review the `actions/create-github-app-token@v2` inputs and confirm `repositories: 2026-03-27--hetzner-k3s` is the intended format for a single repository installation token.
- Review whether the live Vault policy update should be imported into Terraform or another source-of-truth repo before relying on it long term.
- Confirm existing callers of `publish-ghcr-image.yml` are unaffected by the added inputs and validation branch.

### What should be done in the future

- Install `wesen-gitops-pr-bot` on `wesen/2026-03-27--hetzner-k3s`.
- Rerun `scripts/02-verify-github-app-secret-and-token.sh`.
- Commit and push `infra-tooling` workflow changes to `main`.
- Commit and push `publish-vault` workflow/ticket changes.
- Trigger the `publish-image` workflow and confirm it opens a GitOps PR.

### Code review instructions

- Review these changed workflow files:
  - `/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml`
  - `/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/publish-image.yaml`
- Review these retrace scripts:
  - `scripts/03-extend-vault-policy-for-github-app.sh`
  - `scripts/04-patch-infra-tooling-github-app-source.sh`
  - `scripts/05-patch-publish-vault-workflow-github-app.sh`
- Validate with:

```bash
python3 - <<'PY'
from pathlib import Path
import yaml
for p in [
'/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-ghcr-image.yml',
'/home/manuel/code/wesen/2026-05-13--retro-obsidian-publish/.github/workflows/publish-image.yaml',
]:
    yaml.safe_load(Path(p).read_text())
    print('yaml ok:', p)
PY
```

### Technical details

Current blocking installation URL:

```text
https://github.com/apps/wesen-gitops-pr-bot/installations/new
```

Current source workflow target:

```yaml
tooling_repository: go-go-golems/infra-tooling
tooling_ref: main
```

That means `infra-tooling` must be published before the `publish-vault` workflow can consume `gitops_pr_token_source: github_app` from `main`.
