---
title: Draft Note (Not Published)
tags: [draft, example]
publish: false
---

# Draft Note (Not Published)

This note demonstrates the per-note `publish: false` frontmatter flag. It is
parsed by the vault loader but **not** stored in the published note index, so
it never appears in:

- the `/api/notes` list,
- the `/api/tree` file tree,
- the full-text search index,
- any other note's backlinks,
- or the raw-source endpoint (`/api/notes/{slug}/raw`).

## How it works

Publishing is **opt-out only**. Setting `publish: false` hides this note.
Omitting the key (or `publish: true`) leaves the note eligible, subject to the
`.vault-ignore` file and the `.publish/config.yaml` blacklist.

A `publish: true` does **not** resurrect a note excluded by an ignore rule or
the config blacklist — exclusion always wins, to keep the security boundary
clear. If you need to publish a file inside an ignored folder, remove the
exclusion from the relevant file instead.

## When this takes effect

In `--watch` mode (the default for local use), toggling `publish: false` on a
note takes effect on the next debounced file reload — no restart needed. The
note disappears from the API and search within a moment of saving the file.

Changes to the `.publish/config.yaml` blacklist take effect on the next full
reload (restart the server, or call `POST /api/admin/reload`), the same as
`.vault-ignore`.

## See also

- [[Zettelkasten Method]]
- [[Stoicism]]
