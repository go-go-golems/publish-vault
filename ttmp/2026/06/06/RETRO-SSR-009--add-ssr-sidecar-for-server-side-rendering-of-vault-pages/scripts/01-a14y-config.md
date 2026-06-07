# a14y configuration

- Target URL: http://localhost:8081/
- Scorecard: 0.2.0
- Mode: site
- Max pages: 200
- Last runs:
  - 2026-06-07 — 62 (scorecard 0.2.0)

## Baseline run

Command:

```bash
npx -y a14y check http://localhost:8081/ --mode site --output agent-prompt --max-pages 200
```

Summary:

- Score: 62/100
- Scorecard: v0.2.0
- Failed: 63 instances, 13 unique checks
- Pages crawled: 7

Approved scope for the next pass:

- Add markdown endpoints.
- Do all markdown mirror support.
- Store this configuration in the ticket scripts folder instead of AGENTS.md/a14y.md.
