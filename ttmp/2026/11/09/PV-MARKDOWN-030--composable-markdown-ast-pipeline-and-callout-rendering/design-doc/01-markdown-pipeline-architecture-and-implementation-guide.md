---
Title: Markdown pipeline architecture and implementation guide
Ticket: PV-MARKDOWN-030
Status: active
Topics:
    - markdown
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/parser/math.go
      Note: Math masking and restoration
    - Path: repo://internal/parser/parser.go
      Note: Parser pipeline and callout HTML post-pass
    - Path: repo://pkg/vault/vault.go
      Note: Vault indexing and HTML rebuild
    - Path: repo://web/src/components/organisms/NoteHtml/NoteHtml.tsx
      Note: Client enhancement and callout interaction
ExternalSources: []
Summary: Intern guide to structural Markdown parsing, vault resolution, and a staged migration
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# A composable Markdown rendering pipeline

**Historical design:** This document preserves the baseline `0c78a5e` investigation and proposed implementation plan, not the current code state. The broad migration is described in [the implementation and operational guide](02-implemented-markdown-document-pipeline-and-operational-guide.md); consult the diary for its validation and commit evidence. The copy previously uploaded to reMarkable is this original design, not the subsequent implementation guide.

## 1. What the system does and why this change exists

Publish Vault publishes an Obsidian-style vault: a directory of Markdown notes and attachments. A reader opens a URL such as `/note/research/kb/projects/javascript-operating-systems`; the server loads a corresponding note and supplies HTML to the React note view. Notes contain ordinary Markdown as well as vault-specific syntax such as `[[links]]`, `![[embeds]]`, math, and callouts (`> [!note]`). A backlink is a reverse reference: if note A links to B, B can display A among its linked mentions.

The observed defect is not failure to wrap at the viewport edge. Ordinary source line endings become explicit HTML breaks as well. A paragraph or list item that was manually wrapped for editing therefore breaks both at source line boundaries and at the browser edge. `internal/parser/parser.go` enables Goldmark's `html.WithHardWraps()`. Disabling that setting alone is unsafe: `renderCallouts` currently finds the end of a callout's title by searching rendered HTML for the first `<br />`.

This guide proposes representing meaning before presentation. A callout should be a structured node with a title and a body; a line break should carry its Markdown meaning; links should have target fields, not have to be rediscovered in generated HTML. The immediate deliverable is a safe vertical slice for callouts and soft breaks. Wiki links, math, and vault resolution are a subsequent migration; this distinction matters because the current implementation cannot honestly be described as a complete compiler pipeline yet.

## 2. Prerequisites and current code map

You need Go, basic Markdown, and familiarity with trees. An abstract syntax tree (AST) is a tree in which nodes describe parsed constructs, rather than strings of already rendered HTML. A parser reads source into nodes; a pass traverses or changes those nodes; a renderer turns them into output. A soft break is a newline inside a Markdown paragraph; an explicit hard break is written with two trailing spaces or a backslash. Browser layout chooses where soft-wrapped text actually breaks.

Read these entry points first (paths are repository-relative):

- `internal/parser/parser.go`: `Parse`, wiki link scanning, Goldmark construction, `renderCallouts`, heading and image HTML post-passes.
- `internal/parser/math.go`: math scanner, placeholder insertion and restoration; code regions and frontmatter are important boundaries.
- `internal/parser/parser_test.go` and `math_test.go`: current examples and regressions. In particular, verify that test strings contain actual newlines rather than literal `\\n` before drawing conclusions.
- `pkg/vault/vault.go`: `loadNote`, `buildWikiLinkIndex`, `buildBacklinks`, and `rebuildHTML`. `loadNote` stores parser output in `sourceHTML`; rebuilding starts from that source rather than repeatedly rewriting the last result.
- `web/src/components/organisms/NoteHtml/NoteHtml.tsx` and `web/src/styles/prose.css`: client enhancements and the CSS hooks expected for rendered callouts.
- Prior analysis: `ttmp/2026/08/16/PV-MARKDOWN-023--fundamentals-first-markdown-and-wiki-link-parser-architecture-review/design-doc/01-markdown-and-wiki-link-parsing-algorithms-architecture-and-robust-building-blocks.md`. Treat its proposals as research, not proof of current behavior.

At baseline `0c78a5e`, the code uses Goldmark v1.8.2 and goldmark-meta v1.1.0 (`go.mod`). Before implementing, run `go test ./internal/parser ./pkg/vault` to establish a baseline. The server and frontend have additional tests; a parser-only pass does not prove the live deployment has changed.

## 3. How a note became a page at the baseline

```text
Markdown bytes
  | mask math; scan and replace wiki links with HTML placeholders
  v
Goldmark + meta + GFM + hard-wrap option
  | rendered HTML
  v
callout HTML regex -> restore math -> self-heading HTML resolution
  | ParsedNote{HTML, WikiLinks, Frontmatter, Tags, Title, Excerpt}
  v
Vault.loadNote -> notes and sourceHTML
  | build normalized and wiki-link indexes; build backlinks
  v
Vault.rebuildHTML -> resolve cross-note headings, links and assets in HTML
  v
API/SSR -> React NoteHtml -> optional browser enhancements
```

The source scan occurs before Goldmark, so `parser.go` contains separate logic to exclude frontmatter, code and math from wiki-link recognition. Math has its own scanner and placeholders. `renderCallouts` then works on HTML, not the blockquote tree: it assumes an HTML break separates title and body. The vault must wait until notes are loaded to resolve `[[short/path]]` against its suffix index, and cross-note headings are looked up using target note heading indexes. These are real ordering requirements, not merely code organization.

The consequence is accidental coupling. `WithHardWraps()` helps the callout regex find the title boundary but makes all soft breaks hard breaks. A change to paragraph rendering should not change whether a callout has a title. HTML string rewriting also requires every producer and consumer to agree on serialization, escaping, and ordering.

## 4. The target: structure first, HTML last

```text
source -> parse -> document AST -> analyze -> resolve -> HTML renderer
                    |                |           |
                    |                |           + vault index / assets
                    |                + references, headings, metadata
                    + Callout, WikiLink, Math plus ordinary Markdown nodes
```

The AST is the intermediate representation (IR). It belongs to one source document and retains source bytes as long as nodes reference byte segments. A pass has a defined input, output, and stage; it should not assume another feature's HTML spelling. Goldmark already provides block/inline parsers, AST transformers, node renderers, and priority ordering. Prefer those existing mechanisms to a general-purpose plugin framework or an independently maintained second AST.

For example:

```markdown
> [!note] Read this
> A paragraph with a [[Map]]
> that flows naturally.
```

Conceptually:

```text
Callout(type=note, title="Read this", fold=none)
  Paragraph
    Text("A paragraph with a ")
    WikiLink(target="Map")
    SoftBreak
    Text("that flows naturally.")
```

The renderer writes the callout shell, then ordinary child rendering produces its contents. A soft break is whitespace; an explicit hard break remains `<br />`. The example is the target representation, not a claim that existing wiki links already use AST nodes.

### Callouts

A callout is a blockquote whose first line begins with an Obsidian marker `[!type]`, optionally followed by `+` or `-` and title text. The marker and header boundary must be read from the parsed source/AST, not inferred from `<br>`. A callout node holds type, folding state and title; its children hold normal block content. Render title text safely, escaping user-controlled data. Preserve nested paragraphs and lists as nodes rather than stripping paragraph tags from strings. Normal blockquotes and code blocks containing marker-like text are not callouts. Specify fold semantics explicitly and verify the existing browser click handler and CSS contract before changing generated markup.

Goldmark's `parser.ASTTransformer.Transform(*ast.Document, text.Reader, parser.Context)` runs after parsing, while `renderer.NodeRenderer` registers enter/exit callbacks. An alternative is a custom block parser that recognizes the header while retaining Goldmark's blockquote behavior. Select based on a prototype: a transformer must reliably identify the first physical line and preserve inline title markup; a block parser must correctly handle nesting and continuation. In either design, nested nodes and source segments must remain owned by the document. Do not split formatted HTML on the first newline.

### Wiki links and math

A future inline parser should create a link node containing target, optional heading and alias, and embed kind. An analysis visitor collects references from the exact same nodes the renderer uses. This prevents a scanner and renderer from disagreeing about links inside code. Math needs its own inline and block nodes and an explicit precedence contract against code, escapes, currency, wiki links and display delimiters. Current `math.go` handles several of those cases; preserve its regression fixtures while replacing it. Do not claim existing scanners can simply be removed until the new parser passes those tests.

### Vault resolution and lifetimes

Resolution maps a source target to a published note and possibly a heading. The parser must not call the filesystem or depend on note-load order. Instead, vault loading first parses/indexes all notes, then a resolver backed by that snapshot looks up paths, headings and assets. For an unresolved link, the renderer should intentionally produce a visibly unresolved link or diagnostic, not silently fabricate a successful target. The exact policy must be reconciled with current UI behavior during migration.

A Goldmark document may retain source byte slices. If ASTs are stored for every note, account for memory and ownership. An alternative is to parse again at rendering or store a compact IR; benchmark representative vaults before choosing. Existing `sourceHTML` is a string and is not an AST. Watcher reloads and index changes must cause dependent links to re-resolve; an immutable per-rebuild resolver snapshot is a simple way to make a render internally consistent. The current vault rebuild is the integration baseline, not evidence that this new snapshot API already exists.

## 5. API sketch and pass contracts

The following is **proposed pseudocode**, not an API already available in this repository:

```go
engine := markdown.New(markdown.Options{SoftBreaks: markdown.Space})
doc, err := engine.Parse(source)       // owns or retains source bytes
analysis := engine.Analyze(doc)        // headings, links, metadata
index := vault.BuildIndex(allAnalyses) // all notes before resolution
html, err := engine.Render(doc, markdown.RenderOptions{Resolver: index})
```

A minimal resolver exposes domain operations rather than HTML mutations:

```go
// Pseudocode: distinguish absent targets from operational errors.
type Resolver interface {
    ResolveNote(target string) (NoteTarget, bool)
    ResolveHeading(note NoteTarget, heading string) (string, bool)
    ResolveAsset(target string) (string, bool)
}
```

`Parse` determines syntax without vault access. `Analyze` reads nodes, ideally without mutation, and has no I/O. `Render` consumes a stable resolver and escapes dynamic text and attributes. APIs should return errors for actual failures, not for ordinary missing targets; missing-target presentation is an explicit policy. No extension may modify raw source offsets behind another node's back. Node traversal must use Goldmark's documented mutation rules; never remove siblings while relying on an iterator whose next node has become invalid.

Initially an internal `Callout` extension registered alongside the existing parser is enough. Avoid exposing AST internals publicly before lifetime and concurrency contracts are established. `Parse` currently returns `ParsedNote`; replacing that public shape is a later integration decision, not a prerequisite for correcting soft breaks.

## 6. Decisions, alternatives, and security

**Decision: use the existing Goldmark AST.** An HTML regex is smaller but couples semantics to serialization and loses source structure. A new independent AST can support multiple renderers but imposes conversion, ownership and testing costs now. A Goldmark extension is the narrowest composable choice; revisit if multiple outputs expose real limitations.

**Decision: default to normal Markdown soft breaks.** This is a deliberate change for notes that used a single source newline as a visual break. Authors can use two trailing spaces, a backslash, separate paragraphs, or an explicit HTML break where appropriate. Test real vault examples before rollout; the user does not require a compatibility mode.

**Decision: phase the migration.** A structural callout alone does not make wiki links and math structural. Keep the remaining pipeline accurately documented and migrate each feature with its own contract and regression suite. Avoid rewriting the whole renderer merely to fix wrapping.

**Security:** `html.WithUnsafe()` currently permits raw HTML, partly for generated placeholders. A structural migration can reduce that need but does not by itself sanitize vault content. Explicitly decide whether vault authors are trusted and where sanitization occurs before disabling or tightening unsafe HTML. Escape callout titles, types, attributes and resolver outputs according to their HTML context. Do not interpolate raw note text into tags. Test malicious title/type/URL inputs and existing raw-HTML expectations.

## 7. Implementation plan

1. Baseline `go test ./internal/parser ./pkg/vault`; capture failures rather than treating unrelated failures as fixed. Add failing tests with real physical lines: paragraph, list, explicit breaks, title and body, empty title, multiple blocks, nested callouts, normal blockquote and fenced code.
2. Prototype a Goldmark structural callout extension. Identify the marker from the first blockquote paragraph and first physical line, and retain inline nodes/body. Register a node renderer with stable `callout`, `callout-title`, `callout-body`, icon, and folding hooks expected by the frontend. Confirm whether `+` and `-` behavior matches product expectations.
3. Remove `WithHardWraps`, remove the old callout HTML pass only after new coverage demonstrates composition. Ensure callout body paragraphs and list items use the same soft-break policy as ordinary content.
4. Run parser and vault tests, then relevant broader Go tests; run a representative rendered-note integration test. Verify no dependency on stale HTML cache or indexes. Check rebuild behavior after links or target headings change.
5. Move wiki links, math, heading/asset resolution one feature at a time, if included in the implementation scope. Each move needs explicit precedence, safe rendering, reference consistency, and vault-level tests. Do not mark it done just because callouts are migrated.
6. Build and deploy separately from a local code commit. Confirm the live HTML lacks unwanted `<br>` and preserves explicit ones, and test callouts at multiple viewport widths. Deployment and reMarkable upload are distinct operations.

### Test matrix

| Input or operation | Expected property |
|---|---|
| `one\ntwo` in a paragraph or list item | One flowing paragraph, no generated hard break |
| `one  \ntwo` and `one\\\ntwo` | Explicit `<br>` remains |
| Callout header, title, body with multiple blocks | Separate title and structurally rendered body |
| Nested callout, normal quote, code fence with marker | Correct nested node; non-callouts unchanged |
| Wiki link and math inside/outside callouts | Existing link/math behavior and safe escaping retained |
| Missing target, moved target, heading edit | Visible unresolved state or updated target after rebuild |
| Unsafe-looking title and raw HTML | Escaped title; documented trust policy for raw HTML |

The test matrix specifies desired outcomes, not results. Include real Markdown line endings rather than backslash-n text. Assert HTML structure rather than whitespace alone; add browser-level checks for behavior that depends on CSS or client event handlers.

## 8. Operational handoff and open design questions

Parser changes affect newly rendered notes, not a running deployment until a new binary/image is deployed. Inspect the actual deployment's build and caching scheme before deciding whether persistent search snapshots, HTML caches or indexes need versioning; do not assume a frontend asset rebuild updates backend parsing. Track any deliberate output changes in release notes.

Open questions for the subsequent full migration: how to expose diagnostics; whether to retain ASTs or reparse under vault memory constraints; whether raw HTML is trusted; exact fold-open behavior; and whether resolution snapshots need dependency-level invalidation. None prevents a narrowly tested callout/soft-break slice, but they block claiming a completed universal renderer.

For the intern: start at `Parse`, follow `loadNote` to `rebuildHTML`, then add one test from the matrix and observe it fail. Implement the smallest structural change, run the narrow tests, and only then expand to vault integration. Read the ticket diary for actual implementation decisions, test output and commit hashes; this document separates intent from evidence.
