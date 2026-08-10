# Changelog

## 2026-08-09

- Initial workspace created


## 2026-08-09

Step 1: Investigated the note pipeline end to end and wrote the intern-facing design/analysis/implementation guide; created 14 phased tasks. Key finding: math must be protected in a pre-goldmark pass mirroring replaceWikiLinks, and typeset client-side mirroring enhanceMermaid.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/parser.go — Insertion point for the math pre-pass


## 2026-08-09

Step 2: Go-side math protection shipped (commit e9f2784). Scanner + sentinel round-trip + search stripping; 26-row table test. Design correction: inline HTML placeholders do NOT survive goldmark (it parses the text between raw-HTML tags as Markdown), so math travels as a U+E000 sentinel and is restored after renderCallouts.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/internal/parser/math.go — Scanner, sentinel substitution, and placeholder restoration


## 2026-08-09

Step 3: Browser typesetting shipped (commit 40886c3). Four bugs found by the showcase note and browser verification: $30 closing math inside a later code span; MathJax 4 dynamic font ranges needing an asyncLoad bridge plus handleRetriesFor; inline linebreaking on a detached node; Tailwind preflight svg{display:block}. 64/64 formulas typeset, SSR smoke clean, main stays 468 KB.

### Related Files

- /home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/web/src/lib/mathjax.ts — MathJax singleton, font-range loader map, retry handling

