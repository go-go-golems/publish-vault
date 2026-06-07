---
Title: "a14y configuration"
Ticket: "RETRO-SSR-009"
Status: "active"
Topics:
  - a14y
  - ssr
  - markdown
DocType: "reference"
Intent: "Record local a14y audit command, approved scope, and baseline/final score history."
Owners: []
RelatedFiles:
  - Path: backend/internal/server/agent_markdown.go
    Note: Agent discovery and Markdown mirror endpoint implementation
  - Path: web/server.mjs
    Note: SSR HTML alternate-link and metadata implementation
ExternalSources: []
Summary: "Local a14y scorecard configuration and audit history for RETRO-SSR-009."
LastUpdated: 2026-06-07T01:38:12Z
WhatFor: "Reproduce a14y audits against the local devctl example profile."
WhenToUse: "Use when rerunning a14y or comparing future score changes for the SSR publish-vault site."
---

# a14y configuration

- Target URL: http://localhost:8081/
- Scorecard: 0.2.0
- Mode: site
- Max pages: 200
- Last runs:
  - 2026-06-07 — 99 (scorecard 0.2.0)
  - 2026-06-07 — 62 (scorecard 0.2.0)

## Baseline run

Command:

```bash
npx -y a14y check http://localhost:8081/ --mode site --output agent-prompt --max-pages 200
```

Summary:

- Baseline score: 62/100
- Final score after markdown mirror implementation: 99/100
- Scorecard: v0.2.0
- Baseline failures: 63 instances, 13 unique checks
- Final failures: 1 instance, 1 unique check (`html.headings` on `/`, intentionally skipped in the approved plan)
- Pages crawled: baseline 7, final 16

Approved scope for the next pass:

- Add markdown endpoints.
- Do all markdown mirror support.
- Store this configuration in the ticket scripts folder instead of AGENTS.md/a14y.md.
