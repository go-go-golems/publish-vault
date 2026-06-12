---
title: Org Style Audit Report
doc_type: reference
status: active
intent: short-term
topics:
  - tooling
  - ci
  - linting
---

# Org Style Audit Report

Target Go: `1.26.4`
Target golangci-lint: `v2.12.2`

| Repo | Class | Dirty | Go | golangci | Glazed | Logcopter | Notes |
|---|---|---:|---|---|---:|---:|---|
| ai-in-action-app | safe-bump-candidate | no | `1.25.0` | `-` | no | yes | go 1.25.0 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| almanach | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| barbar | legacy-manual-review | no | `1.20` | `-` | no | no | go 1.20 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml; missing Makefile |
| biberon | legacy-manual-review | no | `1.19` | `-` | yes | no | go 1.19 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| bobatea | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | no | yes | go 1.26.3 != 1.26.4 |
| bubble-table | legacy-manual-review | no | `1.18` | `-` | no | no | go 1.18 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| bucheron | legacy-manual-review | no | `1.19` | `-` | yes | no | go 1.19 != 1.26.4; missing .golangci-lint-version |
| clay | safe-bump-candidate | no | `1.25.0` | `v2.11.2` | yes | yes | go 1.25.0 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| cliopatra | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| codex-sessions | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| common-sense | legacy-manual-review | no | `1.20` | `-` | yes | no | go 1.20 != 1.26.4; missing .golangci-lint-version |
| cozodb-goja | safe-bump-candidate | no | `1.25.7` | `-` | yes | yes | go 1.25.7 != 1.26.4; missing .golangci-lint-version |
| css-visual-diff | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| devctl | safe-bump-candidate | no | `1.25.5` | `-` | yes | yes | go 1.25.5 != 1.26.4; missing .golangci-lint-version |
| discord-bot | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| dmeta | safe-bump-candidate | no | `1.26.1` | `-` | yes | yes | go 1.26.1 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| docmgr | safe-bump-candidate | no | `1.25.7` | `v2.12.2` | yes | yes | go 1.25.7 != 1.26.4 |
| ecrivain | legacy-manual-review | no | `1.19` | `-` | yes | no | go 1.19 != 1.26.4; missing .golangci-lint-version |
| escuse-me | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| esper | safe-bump-candidate | no | `1.25.5` | `-` | yes | yes | go 1.25.5 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| font-util | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| form-generator | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| geppetto | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| gitcommit | safe-bump-candidate | no | `1.25.0` | `-` | yes | yes | go 1.25.0 != 1.26.4; missing .golangci-lint-version |
| glazed | safe-bump-candidate | no | `1.25.0` | `v2.11.2` | yes | yes | go 1.25.0 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-emrichen | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| go-go-agent | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| go-go-agent-action | safe-bump-candidate | no | `1.25.0` | `-` | no | yes | go 1.25.0 != 1.26.4; missing .golangci-lint-version |
| go-go-app-arc-agi | safe-bump-candidate | no | `1.25.10` | `v2.11.2` | no | yes | go 1.25.10 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-go-app-inventory | current-or-nearly-current | no | `1.26.4` | `v2.12.2` | yes | yes | ok |
| go-go-app-sqlite | safe-bump-candidate | no | `1.25.0` | `v2.11.2` | no | yes | go 1.25.0 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-go-gepa | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-go-goja | safe-bump-candidate | no | `1.26.1` | `v2.11.2` | yes | yes | go 1.26.1 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-go-host | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| go-go-mcp | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| go-go-os-backend | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | no | yes | go 1.26.3 != 1.26.4 |
| go-go-os-chat | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| go-minitrace | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| go-sqlite-regexp | safe-bump-candidate | no | `1.25.0` | `-` | no | yes | go 1.25.0 != 1.26.4; missing .golangci-lint-version |
| go-template | current-or-nearly-current | no | `1.26.4` | `v2.12.2` | no | yes | ok |
| goja-bleve | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| goja-git | safe-bump-candidate | no | `1.26.3` | `-` | yes | yes | go 1.26.3 != 1.26.4; missing .golangci-lint-version |
| goja-github-actions | safe-bump-candidate | no | `1.26.1` | `v2.11.2` | yes | yes | go 1.26.1 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| goja-text | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| harkonnen | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| infra-tooling | safe-bump-candidate | no | `1.25.5` | `-` | yes | yes | go 1.25.5 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml; missing Makefile |
| jesus | current-or-nearly-current | no | `1.26.4` | `v2.12.2` | yes | yes | ok |
| js-analyzer | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4; missing .golangci.yml |
| llm-proxy | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| logcopter | safe-bump-candidate | no | `1.25.11` | `v2.11.2` | yes | yes | go 1.25.11 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| loupedeck | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| markdown-quizz | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| mastoid | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| oak | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| oak-git-db | safe-bump-candidate | no | `1.25.5` | `-` | no | yes | go 1.25.5 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| openai-app-server | safe-bump-candidate | no | `1.25.7` | `-` | yes | yes | go 1.25.7 != 1.26.4; missing .golangci-lint-version |
| openai-mock-server | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| parka | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| pi-launcher | safe-bump-candidate | no | `1.25.0` | `v2.11.2` | yes | yes | go 1.25.0 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| pinocchio | safe-bump-candidate | no | `1.26.3` | `v2.11.2` | yes | yes | go 1.26.3 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| plunger | legacy-manual-review | no | `1.19` | `-` | yes | no | go 1.19 != 1.26.4; missing .golangci-lint-version |
| plz-confirm | safe-bump-candidate | no | `1.26.1` | `-` | yes | yes | go 1.26.1 != 1.26.4; missing .golangci-lint-version |
| prescribe | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| prompto | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| publish-vault | safe-bump-candidate | no | `1.25.0` | `v2.12.2` | yes | no | go 1.25.0 != 1.26.4 |
| raza | legacy-manual-review | no | `1.17` | `-` | no | no | go 1.17 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml; missing Makefile |
| react-chat | safe-bump-candidate | no | `1.26.3` | `-` | yes | yes | go 1.26.3 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml; missing Makefile |
| refactorio | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| remarquee | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| salad | safe-bump-candidate | no | `1.25.3` | `-` | no | yes | go 1.25.3 != 1.26.4; missing .golangci-lint-version |
| sanitize | safe-bump-candidate | no | `1.25.7` | `v2.11.2` | yes | yes | go 1.25.7 != 1.26.4; golangci v2.11.2 != v2.12.2 |
| scraper | safe-bump-candidate | no | `1.26.4` | `v2.11.2` | yes | yes | golangci v2.11.2 != v2.12.2 |
| sessionstream | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| smailnail | current-or-nearly-current | no | `1.26.4` | `v2.12.2` | yes | yes | ok |
| sqleton | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| tactician | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| terraform-provider-stytch-b2b | legacy-manual-review | no | `1.24.7` | `-` | no | no | go 1.24.7 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml; missing Makefile |
| uhoh | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| vault-envrc-generator | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| vm-system | safe-bump-candidate | no | `1.26.1` | `-` | yes | yes | go 1.26.1 != 1.26.4; missing .golangci-lint-version; missing .golangci.yml |
| voyage | legacy-manual-review | no | `1.19` | `-` | no | no | go 1.19 != 1.26.4; missing .golangci-lint-version |
| web-agent-example | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
| workspace-manager | safe-bump-candidate | no | `1.26.4` | `-` | yes | yes | missing .golangci-lint-version |
| zine-layout | safe-bump-candidate | no | `1.26.3` | `v2.12.2` | yes | yes | go 1.26.3 != 1.26.4 |
