---
Title: Implemented Markdown document pipeline and operational guide
Ticket: PV-MARKDOWN-030
Status: active
Topics:
    - markdown
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/document.go
      Note: Document ownership, heading analysis and rendering
    - Path: repo://internal/parser/document_test.go
      Note: Structural pipeline regressions
    - Path: repo://internal/parser/math_ast.go
      Note: Math block and inline AST extensions
    - Path: repo://internal/parser/table_ast.go
      Note: Table cell preservation and syntax precedence
    - Path: repo://internal/parser/wikilink_ast.go
      Note: Wiki nodes and render-time resolution
    - Path: repo://pkg/vault/vault.go
      Note: Indexing, reload and structural rebuild
ExternalSources: []
Summary: The structural Markdown pipeline, its contracts, limits and validation
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# The structural Markdown document pipeline

This guide describes the implementation in the working branch, not a deployment. The [original design](01-markdown-pipeline-architecture-and-implementation-guide.md) records the earlier string-based renderer and the motivations for the change. Read the diary for command results and commit hashes: a design description is not evidence that a build or deployment passed.

## 1. The problem, in concrete terms

A vault is a directory of Markdown notes and assets. Publish Vault loads eligible notes, extracts their links, builds an index, and serves HTML to a browser. In the old renderer, one ordinary newline inside a Markdown paragraph generated a `<br>`; the browser then also wrapped the resulting lines at the viewport width. The callout post-processor relied on that same generated `<br>` to distinguish its header from its body. Removing the hard-wrap option without replacing the callout parser would therefore break a different feature.

The broader problem was the representation used between stages. Wiki links and math were protected by replacing source fragments before Markdown parsing. Later code searched generated HTML strings to recover and rewrite note links, image paths and heading references. Each stage depended on the exact bytes produced by the previous stage. The new pipeline retains a document tree through rendering and represents callouts, wiki links and math as nodes. Soft line breaks use normal Markdown semantics; explicit two-space or backslash hard breaks remain hard breaks.

## 2. The data flow and ownership boundary

An abstract syntax tree (AST) is a tree of parsed constructs. An inline node is content inside a block, such as a wiki link or inline formula; a block node is a paragraph, callout or display equation. Goldmark nodes refer to byte segments in their input, so their source bytes must stay alive and unmodified while the tree is in use.

```text
Markdown bytes
   | parser.ParseDocument: clone source, parse blocks + inline nodes
   v
Document { owned source, Goldmark tree, heading index }
   | analyze metadata and wiki nodes
   v
Vault note { original source, compact heading index, references }
   | all eligible notes are indexed; backlinks are derived
   v
rebuildHTML: reparse one source, render against vault Resolver
   | note / heading / wiki-image / Markdown-image callbacks
   v
HTML -> API -> React NoteHtml -> optional MathJax / Mermaid enhancement
```

`internal/parser/document.go` defines `Document`, `ParseDocument`, `ParseForRendering` and `Resolver`. `ParseDocument` returns metadata plus an immutable document whose source is a clone. `Document.Render` creates a renderer for the supplied resolver; it does not modify the tree. `ParseForRendering` skips repeated metadata and reference analysis and accepts the heading index from the same source and parser configuration. That precondition is important: the vault keeps the source and index together and replaces both when a note reloads.

The vault does **not** keep all ASTs resident. `pkg/vault/vault.go` retains each note's source bytes and compact heading index; `rebuildHTML` reparses one source at a time. It first accumulates new HTML and publishes it only after every render succeeds. This trades a second parse for bounded AST retention and avoids partially publishing an HTML rebuild. The vault lock is held during rebuild; resolver callbacks consult already-protected maps and must not call a method that reacquires that lock.

The internal API is concrete Go, not the speculative builder in the old design:

```go
note, doc, err := parser.ParseDocument(source)
if err != nil { return err }
html, err := doc.Render(&parser.Resolver{
    Note: func(target string) (slug, title string, found bool) { /* index */ },
    Heading: func(slug, heading string) (id string, found bool) { /* index */ },
    WikiImage: func(target string) (url string, found bool) { /* assets */ },
    Image: func(destination string) string { /* relative to note */ },
})
```

The callbacks are illustrative; see the actual declarations in `document.go`. Missing targets are normal results (`found == false`), not I/O errors. A nil resolver produces standalone parser output. During a vault rebuild a resolver sees the current locked index; it is not an independently persisted snapshot or a public network API.

## 3. The structural features

### Callouts and ordinary paragraphs

`internal/parser/callout.go` registers its passes together as one Goldmark extension. A paragraph transformer runs before inline parsing and extracts the first physical line of a callout header. Goldmark then parses title formatting normally. After block and inline parsing, an AST transformer collects callout blockquotes in postorder and replaces them with dedicated nodes. Collection precedes mutation because detaching a node while Goldmark is iterating siblings can skip the next sibling. A renderer writes the callout wrapper and then delegates its children to normal renderers.

This separation matters for an input such as `> [!note] A **title**` followed by a multiline body: the header boundary comes from the source line, while the body is a normal paragraph. It does not depend on a generated `<br>`. The frontend relies on `.callout-title`, `.callout-body`, and `.callout-collapsible` in `web/src/components/organisms/NoteHtml/NoteHtml.tsx` and `web/src/styles/prose.css`; the `-` marker starts collapsed, while `+` retains the existing open behavior.

### Wiki references and heading IDs

`internal/parser/wikilink_ast.go` recognizes `[[target#heading|alias]]` and `![[embed]]` in inline Markdown and stores parsed fields in a wiki node. The reference visitor and renderer consume the same node; code spans and fenced or indented code remain code rather than producing body backlinks. Frontmatter references are a separate, intentional metadata convention and are scanned independently because YAML is not rendered as Markdown. Image embeds are assets, not backlink references.

A note's `HeadingIndex` is derived from heading nodes and the IDs Goldmark assigned, not guessed from a second slug function or extracted from rendered HTML. It matches normalized heading text, chooses the first duplicate, and accepts an exact ID as a fallback. For cross-note links the vault callback resolves the canonical note first and then asks that note's heading index. An unresolved self heading is visibly marked broken; an unresolved cross-note link uses a same-page unresolved fragment; a missing embed displays a marker. The exact output is in the wiki renderer and its tests.

`Vault.buildWikiLinkIndex` reserves canonical slugs first and sorts candidates before adding suffix and title aliases. Thus a canonical path cannot be stolen by an alias, and ambiguous aliases have a deterministic lexicographic winner. This is a deliberate change from the former map-iteration-dependent result; it can change which note an ambiguous short link selects. Explicit full paths avoid ambiguity.

### Math and precedence

`internal/parser/math_ast.go` supplies inline and display math nodes, Goldmark parsers and renderers. `scanMathAt` in `math.go` supplies the delimiter rules shared with plain-text and table processing. Goldmark identifies code blocks and code spans structurally, so a formula-like string in code is not emitted as math. TeX is escaped as HTML text inside `.math` elements; browser MathJax enhancement replaces their contents later. The server does not execute TeX.

The block parser handles multiline display math; an AST pass lifts display nodes from ordinary paragraphs. When display syntax occurs inside a container such as a table cell or link, it remains a `span` with `math-display`, preserving a valid inline HTML element while the client uses the class to request display typesetting. Closing-delimiter lookahead checks enclosing quote markers; unmatched source is restored rather than intentionally using a later quote's closer. This area is subtle: nested containers, malformed delimiters and lazy continuation need regression coverage, and the lexical corpus audit in the diary is not proof of every possible input.

### Tables and pipes

Goldmark's standard GFM table transformer truncates row cells beyond the header's column count and identifies cell boundaries before inline extensions run. Real imported notes included both extra cells and math/wiki syntax containing `|`; a naive migration could therefore discard text or split an alias or formula. `internal/parser/table_ast.go` intercepts the affected tables before Goldmark's table transformer, makes the required additional columns, and preserves pipes inside recognized wiki links and math. A subsequent AST pass removes table-specific escaped pipes in code spans. Conforming unaffected tables continue through Goldmark's normal table extension.

This extension is deliberately limited to a concrete data-preservation issue; it is not a general Markdown parser framework. Its delimiter and cell-boundary rules require maintenance alongside Goldmark upgrades, so the table tests and real-vault inventory are important. Review changes to the extension against both valid GFM tables and malformed imported rows rather than assuming all `|` characters have the same role.

### Images and raw HTML

Ordinary Markdown images are AST image nodes. Their renderer invokes `Resolver.Image` with the destination and emits an escaped attribute. Wiki image nodes call `Resolver.WikiImage`; unresolved assets produce a marker. Raw HTML remains trusted under the existing `html.WithUnsafe()` policy. For raw `<img>` tags only, `document.go` tokenizes the raw HTML fragment and updates its `src` attribute, preserving other raw token bytes. This is structural token processing during node rendering, not a regex over the assembled document.

This is **not sanitization**. A vault author who can write raw HTML can still provide active markup. Resolver results and wiki labels are escaped in their HTML contexts; this does not turn untrusted vaults into safe input. Changing the trust policy requires a separate product/security decision and comprehensive tests.

## 4. Reloads, errors and operational consequences

When a note changes, `ReloadNote` reloads its source and compact heading index, updates the vault's indexes and backlinks, then rebuilds HTML under the vault lock. Other notes are rendered again against the new index, so a previously missing note, image or heading can become resolvable. Removal similarly rebuilds the remaining notes. Rendering errors are returned rather than silently publishing a partially rendered HTML map; the watcher still removes a deleted note from search when a render after removal fails.

The implementation changes `Vault.RemoveNote` to return `(slug, error)`, so its watcher caller handles that error explicitly. An index update has already occurred if the subsequent rebuild fails; callers must not interpret the error as a rollback of every vault mutation. The HTML publication step itself is all-or-nothing within a rebuild.

Source changes become visible on the live site only after building and deploying the backend and refreshing its vault snapshot. The local branch, a successful build, and the reMarkable design upload do not by themselves change production. No production deployment is implied by the design or by local smoke tests.

## 5. Verification and practical review

The current tests are the executable contract: `internal/parser/document_test.go`, `callout_ast_test.go`, `softbreak_test.go`, `math_test.go`, `wikilink_ast_test.go`, and `pkg/vault/callout_pipeline_test.go` cover source ownership, parser composition, aliases, headings, display/inline math, code, explicit breaks, table recovery, raw images and vault resolution. `pkg/vault/vault_test.go` contains reload, missing target, asset, heading and idempotence cases. Run `go test ./internal/parser ./pkg/vault ./pkg/watcher` for focused iteration and `make ci-check` at the delivery boundary. See the diary for actual results, including any failures and retries; do not infer a final pass from the commands listed here.

The ticket scripts include `02-vault-load` for a content-free real-vault load measurement, `03-vault-inventory` for comparing aggregate feature counts, and `04-math-container-audit.py` for an approximate source audit. The latter is lexical, does not apply publication filters, and must not be used as proof that all Markdown nesting cases are covered. A comparison of counts is likewise a discovery tool: equal counts cannot prove identical content, and differences require source inspection.

For a review, start at `ParseDocument` and trace a note through `Vault.loadNote`, index construction and `rebuildHTML`. Then follow one wiki link and one display equation through their node renderers. Check the exact scope of a proposed change before adding another parser pass: source syntax, AST analysis, vault-specific resolution and browser presentation are distinct responsibilities.
