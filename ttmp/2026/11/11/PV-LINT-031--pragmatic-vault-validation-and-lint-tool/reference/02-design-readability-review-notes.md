---
Title: Design readability review notes
Ticket: PV-LINT-031
Status: active
Topics: [vault, linting, markdown, parser]
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Author's second-pass review of the intern guide; not an independent human review.
---

# Design readability review notes

This is a self-review performed after drafting, before revision; it is not an independent intern review.

## Comprehension gaps

1. Section 6 calls the data a snapshot, which a newcomer could read as an atomic filesystem snapshot. Clarify that retaining each loaded note's bytes prevents reread-versus-index mismatch but does not prevent edits during the walk; recommend a stable checkout and state the limitation.
2. Section 5 uses `--strict` and JSONL. Explicitly say the JSONL stream is a series of diagnostics rather than a result envelope, and that a nonzero process status with partial records means the scan must not be interpreted as complete.
3. Section 8 gives optional compatibility rules. Explain why they are not in the mandatory first deliverable and must be retired if fixes land, to keep the design from encouraging a growing heuristic framework.

## Technical checks

- `pkg/server/runtime.go` does resolve the root and config and then pass `vault.WithConfig`; it also builds a search index, which the proposed validator deliberately avoids.
- `pkg/vault/vault.go` stores unexported source and heading indexes, but `AllNotes()` returns pointers and is not a deep immutable snapshot. Clarify that any new source accessor must copy bytes and that this design assumes the offline vault is not concurrently mutated.
- `go.mod` currently selects `github.com/go-go-golems/glazed v1.4.1`; the proposed flags and row API follow the current v1.4 authoring guidance, not a new custom output implementation.
- The comparison with Goldmark source offsets remains an implementation risk: the guide labels the accessor and offset as proposed and asks for nested-container tests; it does not claim the existing node records an offset.

## Intended revisions

Amend the affected sections with the snapshot/non-atomicity, output, and optional-rule distinctions. Preserve the narrow MVP and leave the future implementation task open.
