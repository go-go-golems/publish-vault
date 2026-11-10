# Changelog

## 2026-11-09

- Initial workspace created

## 2026-11-09

Created architectural guide and uploaded it to reMarkable; baseline parser and vault tests pass.

## 2026-11-09

Recorded targeted failing soft-break and malformed callout regression probe; structural implementation remains open.

## 2026-11-09

Implemented AST-backed callouts and ordinary Markdown soft breaks; migrated regression probe into parser tests.

## 2026-11-09

Packaged callout passes as a Goldmark extension and verified vault-level links and actual affected note rendering.

## 2026-11-09

Fixed sibling traversal mutation by collecting callout nodes in postorder before replacement; added sibling regression.

## 2026-11-09

Validated structural callout slice with full Go tests, backend lint, plugin test and vault integration; guide previously delivered to reMarkable.

## 2026-11-09

Migrated wiki-link body parsing and reference collection onto Goldmark AST nodes; removed duplicate code-region pre-scanner.

## 2026-11-09

Stored parsed wiki-link fields on AST nodes and escaped alias text instead of emitting raw markup.

## 2026-11-09

Recorded requested post-migration Obsidian deep-dive and verified existing vault changes must be preserved.

## 2026-11-10

Completed structural math, wiki and asset resolution with document-owned AST, deterministic vault indexing and integrated validation.

## 2026-11-10

Committed structural renderer as b4384c3; wrote, visually reviewed and pushed the go-go-parc deep-dive report as 11a6fa1.

## 2026-11-10

Added bounded structural parser fuzz coverage; 43,718 executions passed.

## 2026-11-10

Removed duplicate Goldmark GFM extension registrations; targeted tests and same-corpus inventory matched the pre-cleanup implementation.

## 2026-11-10

Completed requirement-to-evidence audit for the structural pipeline, validation, reMarkable design and pushed Obsidian report.

## 2026-11-10

Unified math element markup and escaping for structural nodes and wiki labels.
