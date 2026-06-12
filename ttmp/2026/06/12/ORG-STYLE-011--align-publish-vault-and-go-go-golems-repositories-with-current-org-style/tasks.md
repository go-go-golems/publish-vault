# Tasks

## TODO

- [x] Create ORG-STYLE-011 ticket, design doc, diary, and initial implementation plan
- [x] Relate key source files from publish-vault, go-template, and infra-tooling to the design doc
- [x] Upload the initial design package to reMarkable after docmgr validation and dry run
- [ ] Update infra-tooling reusable GHCR workflow to current checkout action style and validate YAML diff
- [ ] Update go-template baseline: Go version, generic lint config, fmt-check/ci-check, optional glazed-lint, and CI/local parity
- [ ] Update publish-vault baseline: Go version, local quality gates, CI parity, and validation commands
- [ ] Add an org audit helper in the ticket scripts directory that reports Go/golangci/tooling drift across repositories
- [ ] Use the audit report to identify safe follow-up batches for other repositories without modifying dirty working trees
- [ ] Maintain the investigation diary, changelog, file relations, and commits after each implementation step
