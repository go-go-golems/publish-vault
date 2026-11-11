---
Title: Vault validation design and intern implementation guide
Ticket: PV-LINT-031
Status: active
Topics:
    - vault
    - linting
    - markdown
    - parser
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/retro-obsidian-publish/commands/serve/serve.go
      Note: Glazed CLI conventions
    - Path: repo://cmd/retro-obsidian-publish/commands/validate/validate.go
      Note: Implemented validate command
    - Path: repo://internal/ignore/ignore.go
      Note: Vault ignore semantics
    - Path: repo://internal/parser/document.go
      Note: AST parsing and heading indexes
    - Path: repo://internal/parser/wikilink_ast.go
      Note: Wiki node parsing and references
    - Path: repo://internal/validate/validate.go
      Note: Implemented published-view lint rules
    - Path: repo://pkg/server/runtime.go
      Note: Root and config snapshot setup
    - Path: repo://pkg/vault/vault.go
      Note: Publication loading and link and asset resolution
ExternalSources: []
Summary: Practical design for a read-only validator that uses publishing semantics and reports actionable note diagnostics.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Vault validation: design and implementation guide

## Implementation status

The MVP described here is now implemented as `retro-obsidian-publish validate`: it checks published-body wiki targets, headings and image embeds. The design below is retained as the original intern guide, so proposed names and future-tense statements describe the plan at the time of writing; consult the current code and README for the implemented contract. Optional compatibility heuristics and Markdown-image existence checks remain out of scope.

## 1. What you are building

Publish Vault turns selected Obsidian Markdown notes into a browsable website. Authors can write wiki links such as `[[Projects/Map#Plan|the plan]]`, embed notes and images, and include headings and tables. A note may look acceptable in the editor while the published site cannot resolve its target, a heading fragment is missing, or an attachment does not exist. The proposed **validator** runs locally before publication and reports such problems with enough context for an author to investigate them.

This document records the original design and implementation guide. The examples under `validate`, the Go types in Section 6, and the rules in Section 8 were proposed at design time; the MVP command now exists, but the pseudocode remains illustrative. The existing code was inspected at commit `469a4bf02707d299103de1d1a41df694d75a2e72` on the Markdown AST migration branch; verify the implementation and PR status again before starting work because it may have changed.

The first version should be deliberately modest:

- Read the same published-note set and resolve links using the same rules as the server.
- Report broken wiki-note links, missing wiki-heading fragments, and missing wiki-image embeds. Make locations accurate where available; report an unknown line honestly when not.
- Optionally flag the two known, narrow rendering-risk patterns discussed below. These are warnings about this renderer, **not claims that the Markdown is invalid**.
- Produce human-readable results and structured rows, never modify the vault, and have a useful strict mode for CI.
- Do not implement a new Markdown parser, an auto-fixer, an exhaustive compatibility verifier, a rule-plugin ecosystem, or a live HTTP service.

A validator can find known problems in a given snapshot. A clean result does not prove all Markdown is rendered correctly. The main goal is to tell the operator where to look, not to adjudicate every possible piece of Markdown.

## 2. A concrete example

Suppose a vault has two files:

```text
vault/
  Guide.md       # [[Roadmap#Release plan|release plan]]
  Roadmap.md     # contains heading "## Scope", but not "Release plan"
```

The note `Roadmap.md` exists, so resolving `Roadmap` succeeds. But the requested heading does not exist. The correct diagnostic is **missing heading**, not missing note:

```text
Guide.md:3: warning WIKI_HEADING_MISSING  Roadmap has no heading "Release plan"
```

The path identifies the **source** note to edit. The line identifies the wiki link in that note, if the parser can provide it. The target's current canonical slug and the requested heading may be shown for context. Do not include the body of either note. If `Roadmap.md` has `publish: false`, the site has no published target; report the link as unresolved without printing the path or contents of an excluded note.

The operator may decide that `Release plan` should be restored, that the link should point to `Scope`, or that the link is intentional and should be ignored. The validator does not perform that edit. Its output is evidence for a human decision.

## 3. Current system: from files to rendered notes

### 3.1 CLI and publication boundary

`cmd/retro-obsidian-publish/commands/root.go` registers the current `serve` and `build` commands. `commands/serve/serve.go` defines a Glazed command and reads `--vault`, `VAULT_DIR`, and `--config`. The server's `pkg/server/runtime.go` resolves the configured vault root, loads the vault config, and constructs the runtime snapshot. The proposed validator is a separate short-lived CLI operation: it should not start the server, watcher, React frontend, or search index.

```text
                           CURRENT SERVING PATH
  serve --vault ROOT
       |
       v
  server runtime: resolve root + load publish config
       |
       v
  vault.New(root, vault.WithConfig(cfg))
       |
       +--> .vault-ignore / config / publish:false eligibility
       +--> parse notes; derive slugs and heading indexes
       +--> build note and asset resolution indexes
       +--> build backlinks; render HTML
       |
       v
  API + frontend serve a snapshot

                           PROPOSED LINT PATH
  validate --vault ROOT [--config PATH] [--strict]
       |
       +--> same resolved root, config, and eligible published snapshot
       +--> collect source occurrences and line locations
       +--> execute small ordered set of checks against snapshot
       +--> diagnostics on stdout; operational errors on stderr
```

The validator must not silently inspect a larger private vault as though it were the published site. `.vault-ignore` and `.publish/config.yaml` are different matchers, and a path excluded by either remains excluded. A note with `publish: false` is also removed from published notes. Hidden directories are skipped by the loader. See `internal/ignore/ignore.go`, `pkg/vaultconfig/config.go`, `pkg/vault/vault.go` (`LoadAll`, `IsExcluded`, `ShouldPruneDir`, `publishFlag`), and README sections on publication and exclusions. Negation affects directory pruning; do not reimplement it using a naive recursive walk.

**Important API detail:** `vault.New(root)` loads `.vault-ignore`, but **does not automatically load** `.publish/config.yaml`. The server loads the config separately and passes `vault.WithConfig(cfg)`. An initial validator must do the same. With no explicit `--config`, load `<resolved-root>/.publish/config.yaml`; with an explicit path, follow the existing server convention. The server logs and ignores malformed config. For validation, report a clear operational error instead of claiming a clean result under a different publication policy; explain this intentional strictness in command help.

### 3.2 Parse, index, resolve, render

`internal/parser/document.go` creates a Goldmark document with GFM, frontmatter, callout, math, wiki-link, and table extensions. `ParseDocument` returns a `ParsedNote` and a `Document`. The document owns its source bytes and AST, and derives a `HeadingIndex`. A wiki link is parsed as a custom node; `wikiReferences` extracts references used to build vault backlinks. Rendering is a later operation that receives `parser.Resolver` callbacks.

`pkg/vault/vault.go` stores published notes by slug, builds a wiki-link index and asset index, then reparses and renders one note at a time. A **slug** is the URL-oriented identity assigned to a note, not necessarily its filename: collisions can be disambiguated. A wiki target is the string the author wrote; `ResolveWikiLink(target)` maps it to a canonical slug or reports failure. For cross-note fragments, rendering resolves the target's heading against its private heading index. A self-heading link such as `[[#Scope]]` refers to the source note itself.

```text
 source: [[Roadmap#Release plan|release plan]]
                     |
                     v
 parser wiki node {target: Roadmap, heading: Release plan, alias: release plan}
                     |
                     v
 vault.ResolveWikiLink("Roadmap") --> canonical slug "roadmap"
                     |
                     v
 target heading index.Lookup("Release plan") --> ID or not found
                     |
                     v
 renderer writes link; validator proposes a diagnostic when lookup fails
```

Use those resolution semantics rather than guessing by constructing a path or slug yourself. Names can be aliases, the `.md` suffix can be stripped, and collisions are handled in the index. See `buildWikiLinkIndex`, `ResolveWikiLink`, and `rebuildHTML` in `pkg/vault/vault.go`.

### 3.3 Assets are not notes

A wiki image such as `![[diagram.png]]` is resolved through `Vault.ResolveAssetEmbed`, which checks the indexed assets and returns `(path, found)`. Markdown images such as `![diagram](images/diagram.png)` instead pass through `Vault.ResolveAssetURL`, which converts a destination into a URL. **It does not check file existence.** Treating a nonempty return value from `ResolveAssetURL` as proof that an image exists would be a bug.

The MVP should check wiki-image embeds using the existing asset lookup. Defer Markdown image existence checks until there is one shared, tested filesystem/path-policy helper that agrees with the static asset server, including exclusions, URL decoding, traversal and symlink behavior. Do not invent a second security policy in the validator. External URLs are not fetched by the MVP.

### 3.4 What already exists and what is missing

The CLI has no general-purpose vault validation verb in the inspected tree. The README's validation checklist runs build and smoke commands. The ticket `PV-MARKDOWN-030` contains `scripts/03-vault-inventory/main.go`, which loads a vault and reports per-note counts of links, backlinks, math, images, and some rendered broken markers. This is useful for an aggregate comparison, but it does not provide complete source locations or detect every missing fragment. In particular, a cross-note link can lose its fragment without showing the self-link `broken` CSS class.

The following facts constrain the new design:

- `Note.WikiLinks` is a derived, deduplicated list. It does not carry positions, does not retain self-heading references, and excludes wiki-image references. It is not a complete list of source occurrences.
- `Note.headings` and `Note.source` are private. `HeadingIndex.Lookup` exists, but there is no public `Vault` method for a caller to query a target note's headings.
- `vault.New` calls `LoadAll`, which parses and renders the vault. A CLI can reuse that for correctness in v1, accepting its startup and memory cost. Do not promise a faster analysis-only mode without implementing one.
- The loader currently logs and skips some unreadable entries and parse failures. A successful `New` is **not proof that every candidate was analyzed**.
- Frontmatter wiki references use a separate metadata convention and currently lack source positions. The MVP must not silently pretend they are AST body occurrences.

## 4. Scope and decisions

### Decision A: validate the published view, not all files

Default input is the same eligible view as the server. This makes a result relevant to actual site output and avoids accidental disclosure of private excluded content. There is no `--include-excluded` in v1. A user who wants authoring lint for private drafts can request it as a separate, explicitly privacy-reviewed feature. Config load errors are fatal for this command because otherwise it cannot know what it is validating.

### Decision B: report, do not repair

Use a short set of built-in rules, each implemented as an ordinary Go function. A slice of rule functions is enough; there is no need for dynamic plugins, a rule DSL, or an asynchronous job framework. The command never writes files. A warning is not authorization to rewrite vault content, especially not a link to an excluded note.

### Decision C: use actual parser and vault semantics

Add narrow read-only APIs where needed rather than inspecting HTML strings or writing regular expressions for wiki-link resolution. Parser-derived occurrences should include self links and embeds and should be distinct from the deduplicated backlink API. Reuse `ResolveWikiLink` and `ResolveAssetEmbed`. Add one read-only heading lookup to `Vault` that delegates to the same `HeadingIndex.Lookup` as rendering. Keep `Note.headings` private.

### Decision D: useful output and simple exit behavior

By default, findings are warnings and do not fail a local run; operational errors fail the run. `--strict` fails when any warning is present, so CI can adopt it deliberately after reviewing current output. Emit structured results through the repository's Glazed command pattern. Do not write JSON logs or progress to stdout where they would corrupt structured output.

The current `cmd/retro-obsidian-publish/main.go` maps any returned command error to exit 1. To avoid inventing infrastructure, v1 uses **exit 0** for completed runs without strict findings and **exit 1** for operational failure or strict findings. The stderr message must distinguish those two cases; consumers needing distinct exit codes can use structured diagnostics or a later root-level typed-error change. Do not call `os.Exit` inside a reusable rule or command implementation.

## 5. Proposed operator interface

These commands illustrate the intended interface; they do not work until implemented:

```sh
# Human-readable diagnostics, and warnings do not make this command fail.
retro-obsidian-publish validate --vault ./my-vault

# Machine-readable diagnostics, one object per line.
retro-obsidian-publish validate --vault ./my-vault --format jsonl

# Intended for CI after existing warnings are reviewed.
retro-obsidian-publish validate --vault ./my-vault --strict --format jsonl

# Reproduce a server using a non-default publish config.
retro-obsidian-publish validate --vault ./my-vault --config /etc/site/config.yaml
```

Proposed flags: `--vault` with the same `VAULT_DIR` fallback as `serve`; `--config` with the same default and explicit-path meaning; and `--strict`. Use Glazed's current standard structured-output flags rather than inventing `--json`. The default `--format table` is acceptable for v1; a custom compiler-style text formatter is not a prerequisite.

Each result is one diagnostic row:

```json
{"rule":"WIKI_HEADING_MISSING","severity":"warning","path":"Guide.md","line":3,"column":1,"message":"heading not found in published target","target":"Roadmap#Release plan"}
```

`path` is vault-relative, with forward slashes for stable output. `line` and `column` are 1-based; `0` means unknown and must never be represented as a real position. Define column as a **byte column**, not a Unicode display column, until the implementation explicitly supports something else. Keep rows sorted by path, line, column, and rule for repeatable runs. `target` is author-written content and can be sensitive; filenames and diagnostics are private operator data and must not be published as telemetry.

A clean validation emits zero diagnostic rows. JSONL is a sequence of diagnostic objects, not a result envelope; always inspect the process exit status as well. If a summary is added, send it to stderr or expose it as a separate explicit output mode; do not mix a summary object with JSONL diagnostics. An incomplete validation must return an operational error even if no findings were produced. Once rows have been emitted, a later operational error still means the overall result is incomplete; partial records cannot be treated as a clean scan.

## 6. Minimal internal API and control flow

The following is **pseudocode**, not an existing API. Names may change during implementation; the important boundaries are occurrences, the published snapshot, and diagnostics.

```go
// internal/parser: every relevant source occurrence, not deduplicated.
type WikiOccurrence struct {
    Target, Heading, Alias string
    Embed, Image           bool
    Offset                 int // source byte offset; -1 when unavailable
}

// Proposed: inspect body AST, including self-heading links and image embeds.
func (d *Document) WikiOccurrences() []WikiOccurrence

// pkg/vault: proposed read-only wrapper over the existing private heading index.
func (v *Vault) ResolveHeading(slug, heading string) (id string, ok bool)

type Diagnostic struct {
    Rule, Severity, Path, Message, Target string
    Line, Column int
}
```

`Image` must use the same classification as rendering, not a separate hard-coded extension list in the CLI. Put a small shared classification function or accessor near the wiki node if necessary. For a link whose target is empty but heading is not, use the **current source note's slug**, not `ResolveWikiLink("")`. For a note embed, resolve it as a note; for an image embed, use `ResolveAssetEmbed`. Preserve each body occurrence even when two links have the same target. The existing deduplicated `WikiLinks` stays unchanged for backlinks and API compatibility.

To provide a position, the parser should store the opening wiki marker's offset as it recognizes the node. `wikiInlineParser.Parse` currently clones its raw syntax, but does not save a source offset; its Goldmark `text.Reader` exposes position/segments. Record an absolute source byte offset from the reader at the parse boundary and test it, including lists, blockquotes, CRLF, and non-ASCII text. Do not assume `strings.Index(source, raw)` is correct: the same link can occur repeatedly or inside code. Convert the offset to line and byte column by counting newlines in the same source snapshot. When a reliable offset cannot be obtained, use `-1` internally and `0` in output, not a fabricated location.

**Body versus metadata:** `wikiReferences` separately finds frontmatter references by scanning metadata bytes. MVP rules operate on body AST occurrences only; say this in `--help` and documentation. A future metadata rule may report a file-level location, but must not merge it into body occurrences silently. Similarly, do not scan literal code blocks or code spans as if they contained live wiki links.

A straightforward engine can process one note at a time after loading the vault:

```go
// Sketch: policy and operational-error plumbing omitted; do not copy verbatim.
v := loadPublishedVault(root, config) // same config + exclusions as serve
for _, note := range sortByPath(v.AllNotes()) {
    source := readPublishedSourceOrFail(v, note.Path)
    _, doc := parser.ParseDocument(source)
    for _, ref := range doc.WikiOccurrences() {
        if ref.Embed && ref.Image {
            if _, ok := v.ResolveAssetEmbed(ref.Target); !ok { emit(missingImage(ref)) }
            continue
        }
        slug := note.Slug
        if ref.Target != "" {
            var ok bool
            slug, ok = v.ResolveWikiLink(ref.Target)
            if !ok { emit(missingNote(ref)); continue }
        }
        if ref.Heading != "" {
            if _, ok := v.ResolveHeading(slug, ref.Heading); !ok { emit(missingHeading(ref)) }
        }
    }
    runOptionalCompatibilityChecks(source, doc)
}
sortDiagnostics()
writeRows()
if strict && findingsExist { return strictFindingsError }
```

Do not call `ResolveHeading` for an unresolved target or emit two diagnostics for the same missing target and its fragment. References to private/ignored notes count as unavailable to the published site, but diagnostics should not resolve or reveal an excluded note's path.

### Snapshot and incomplete-load behavior

This is an offline command, not a long-running concurrent service. Resolve the root once at startup and use that path throughout; do not follow a moving deployment symlink on each file. Warn operators not to edit the vault during a run. Existing `Vault.ReadRaw` rereads disk, whereas the loaded note's original source is private, so rereading can race with changes and produce a different reference than the indexed snapshot. The preferred small change is a read-only `Vault` accessor that returns a **copy** of the stored source for a published slug, under the vault lock; never expose the mutable internal slice. Then parsing locations and heading lookup use the same loaded bytes for each note. This does not make the entire directory an atomic filesystem snapshot: files can change while the loader walks them. Prefer a stable checkout, and do not run the validator concurrently with edits or reloads. `AllNotes` returns note pointers, not deep immutable copies. If implementing the source-copy accessor is inconvenient, explicitly detect changed files and fail the run rather than quietly presenting a mixed snapshot. Do not build a snapshot manager for v1.

`LoadAll` currently ignores some walk errors and logs per-note load errors. A validator that uses it unchanged could falsely say the scan completed. Add a small optional load-diagnostic collection hook or equivalent error return at the actual failure sites, disabled for normal serving. It must report unreadable candidates and parse failures without exposing excluded file contents; validation returns an operational error if any eligible content could not be inspected. Keep expected policy exclusions separate from failures. Test this at the vault boundary. Do not change normal server availability behavior as a side effect of adding lint.

## 7. Rule execution and code layout

Suggested files, all **proposed**:

```text
cmd/retro-obsidian-publish/commands/root.go  # register validate
cmd/retro-obsidian-publish/commands/validate/
    validate.go                              # settings, Glazed output, exit handling
    validate_test.go
internal/validate/
    validate.go                              # loop, diagnostics, stable ordering
    rules.go                                 # small built-in check functions
    validate_test.go                         # temporary test vaults
internal/parser/document.go                  # occurrence accessor, or adjacent file
internal/parser/wikilink_ast.go              # source offset on parsed node
pkg/vault/vault.go                           # read-only heading/source accessors
```

`internal/validate` can import both the internal parser and public vault packages because it is inside this module. Keep domain diagnostics independent of Cobra and Glazed so tests can run the validator directly. Avoid a generic `Rule` interface unless there is a concrete need: a small function slice or an explicit sequence is readable and sufficient. `context.Context` should be checked between files and emission; this is a short-lived command, not a concurrent scheduler.

The command authoring reference is the installed `glazed-command-authoring` skill and the project's `commands/serve/serve.go` for local conventions. In the current Glazed v1.4 convention, a `cmds.GlazeCommand` implements `RunIntoGlazeProcessor`, decodes settings using `vals.DecodeSectionInto(schema.DefaultSlug, &settings)`, and emits each row with `processor.AddRow(ctx, types.NewRow(types.MRP(...)))`. Construct the Cobra command with `cli.BuildCobraCommandFromCommand` and register it in `root.NewRootCommand`. The builder supplies standard `--format` options for structured output; do not copy obsolete `--with-glaze-output` examples. Inspect the module version and current local APIs before coding.

**Strict-mode detail:** complete row emission before returning a typed or clearly worded strict-findings error. Verify with the actual Glazed/Cobra builder that formatter/processor output is flushed even when the command returns that error. If it is not, perform the strict decision at a wrapper point after processor close rather than losing JSON output. `main.go` already handles command errors as exit 1. Test the actual binary behavior, not only the rule function.

## 8. Initial rule list

| Rule | Evidence and condition | Suggested output |
|---|---|---|
| `WIKI_TARGET_MISSING` | A body wiki note link or note embed has a nonempty target; `ResolveWikiLink` returns false. | Warning at source occurrence. |
| `WIKI_HEADING_MISSING` | Target resolves (or the reference is a self-link), but the published target heading index does not resolve the requested fragment. | Warning at source occurrence; do not also report missing target. |
| `WIKI_IMAGE_MISSING` | A body wiki image embed is classified as an image by the same parser logic, and `ResolveAssetEmbed` returns false. | Warning at source occurrence. |
| `COMPAT_CODE_ENTITY_HEADING` | A heading code span contains a literal entity-like sequence that the inspected parser may normalize incorrectly. | Optional compatibility warning, not an error; enable only while the relevant renderer limitation exists. |
| `COMPAT_TABLE_ESCAPED_FINAL_PIPE` | A table row handled by the affected expanded-table path ends in an escaped pipe whose content may be lost. | Optional compatibility warning; prefer a conservative heuristic and explicitly identify it as such. |

The two compatibility findings came from PR 30 review. At the inspected revision both were reproduced in small fixtures; a quick text search of 2,411 local Markdown files found no matching heading or pipe-leading table-row lines. Those searches are only proxies: they were not a published-note-aware scan and can miss alternative syntax. The validator should help replace that uncertainty with actionable evidence, but check current code before adding a compatibility rule: if the parser bug is fixed, the rule should be removed or disabled, not warn forever about valid Markdown.

Compatibility heuristics should be off by default in the first basic implementation, or explicitly enabled with one `--compat` flag once implemented. They must avoid code fences and clearly state uncertainty. Do not add auto-fixes: changing valid Markdown to work around a renderer can alter the author's meaning. The three wiki rules are the MVP; compatibility checks are a small follow-up if detecting them accurately needs more parser internals. They are not a permanent registry of every hypothetical edge case: remove a check when its underlying renderer bug is fixed.

Out of scope for MVP: arbitrary raw HTML, Markdown image existence, remote HTTP links, full CommonMark conformance, alias-collision policy changes, custom rule configuration, a baseline database, and old-versus-new renderer diffs. A future differential command could compare normalized links, fragments, table cells and assets from two revisions, but a one-pass linter does not require it.

## 9. Implementation plan for the intern

1. **Orient and run the baseline.** Read `README.md` publishing/exclusion sections, `commands/root.go`, `commands/serve/serve.go`, `pkg/server/runtime.go` (`loadSnapshot`, root/config helpers), `pkg/vault/vault.go` (`New`, `LoadAll`, `loadNote`, `rebuildHTML`), and `internal/parser/document.go` plus `wikilink_ast.go`. Run `go test ./internal/parser ./pkg/vault` before edits. Confirm the current Glazed module/API and `main.go` error behavior.
2. **Add parser occurrences.** Extend wiki nodes with source offsets; expose an immutable body-occurrence list from `Document`. Keep old backlink extraction intact. Unit test repeated links, self links, image and note embeds, links inside code (not occurrences), and positions under blockquotes, lists, CRLF and UTF-8.
3. **Expose narrow vault queries.** Add a heading lookup and snapshot-source-copy accessor. Use locks consistently; avoid calling a lock-taking method while already holding the vault write lock. Add tests for found/missing headings and for the returned source being independent of internal storage. Add bounded load-issue reporting for incomplete eligible scans without changing default server behavior.
4. **Implement the engine.** Load the config and vault as described; iterate only `AllNotes`; handle the three MVP rules in explicit order. Keep source read, occurrence parsing, and vault lookup errors distinct from ordinary warnings. Sort results. Test a miniature vault with published, excluded, and `publish: false` notes; verify excluded content is never emitted.
5. **Implement the CLI.** Add a Glazed command, flags, row emission, and root registration. Exercise human/default and JSONL output, `VAULT_DIR`, explicit config, strict/non-strict status, and operational failure. Keep logs on stderr. Test the strict-mode flush path with the executable or Cobra root, not a mock processor alone.
6. **Validate and document.** Run the focused package tests, `go test ./...`, and repository-required checks (`make ci-check` at the integration boundary). Run on a small disposable fixture, then on a local vault with operator permission. Report counts and errors without copying private content into a public issue or CI artifact. Document CLI help and one example in the README. Do not mutate the corpus automatically.

Recommended test fixture:

```text
fixture/
  .vault-ignore           # Private/
  .publish/config.yaml    # ignore: [Drafts/]
  Source.md               # valid, missing target, missing heading, image embed
  Target.md               # ## Existing
  Hidden.md               # publish: false
  Private/Secret.md       # excluded
  Drafts/Idea.md          # config-excluded
  diagram.png             # present wiki image
```

Tests should assert both positive and negative cases: valid links produce no diagnostic; a missing target with a fragment produces only one target warning; a self heading looks in `Source`; present and absent embeds differ; ignored and unpublished notes are not scanned; repeated references retain distinct locations; and a file-level issue uses line 0 when its position is unknown. Include a deliberately incomplete load case and assert a nonzero exit even with no warnings. Use `t.TempDir()` and test data, not a real personal vault.

## 10. API reference and limitations

| API or file | Current behavior to use or inspect | Proposed change |
|---|---|---|
| `commands/root.go:NewRootCommand` | Mounts `serve` and `build`. | Register `validate`. |
| `commands/serve/serve.go` | Defines Glazed flags and decodes settings; reads `VAULT_DIR`. | Mirror domain flags; use structured rows for diagnostics. |
| `pkg/server/runtime.go` | Resolves snapshot root and loads config before `vault.New`. Helpers are currently internal to the server package. | Share or carefully extract minimal root/config logic; do not import the whole running server as the validator. |
| `vaultconfig.Load`, `vault.WithConfig`, `vault.New` | Load config separately, attach matcher, then load/render published notes. | Fail validation on config errors; collect load failures. |
| `Vault.AllNotes`, `Vault.ResolveWikiLink`, `Vault.ResolveAssetEmbed` | Published notes and real target/image lookups. | Reuse; do not infer aliases from filenames. |
| `Vault.ResolveAssetURL` | Builds an image URL, not an existence check. | Do not use as proof that a file exists. |
| `parser.ParseDocument`, `Document.Headings`, `HeadingIndex.Lookup` | Parse source and obtain heading index; the vault's index is private. | Add body occurrence accessor and vault heading wrapper. |
| `Vault.ReadRaw` | Rereads a path from disk and applies path checks. | Prefer copied loaded source for consistent offline analysis. |
| `scripts/03-vault-inventory/main.go` in PV-MARKDOWN-030 | Historical aggregate rendered-feature counts. | Use as background, not as the validator contract. |

The underlying parser intentionally allows raw HTML (`htmlrenderer.WithUnsafe()`); linting is not HTML sanitization. Published-vault diagnostics can include author-written target names and paths, so treat reports as private unless reviewed. The proposed validator does not contact external services or change deployment, and the current code has not been modified to implement it.

## 11. Review and acceptance

The design is ready for implementation when a reviewer can answer: **which files are included; what is a note versus a source occurrence; who resolves a target; which error means the scan was incomplete; and what does `--strict` change?** The implementation is ready when the fixture and CLI tests demonstrate those contracts, output is deterministic, the tool is read-only, and a real operator can locate and assess a warning. A clean result means only that the selected implemented checks found no issues in the successfully inspected published snapshot.

There are two ordinary implementation decisions to verify rather than block the design: the cleanest way to expose load errors without changing normal serving, and the exact Goldmark source-offset extraction for nested containers. Prototype each with a small test before building the command. If either requires a broad loader rewrite, stop and revisit the scope rather than growing the validator into a publishing-engine replacement.

## References

- Existing implementation: `cmd/retro-obsidian-publish/commands/root.go`, `commands/serve/serve.go`, `pkg/server/runtime.go`, `pkg/vault/vault.go`, `pkg/vaultconfig/config.go`, `internal/ignore/ignore.go`.
- Parser and rendering: `internal/parser/document.go`, `wikilink_ast.go`, `parser.go`, `table_ast.go`; tests in `internal/parser/pr30_review_test.go` and `pkg/vault/callout_pipeline_test.go`.
- Prior analysis and inventory: `ttmp/2026/11/09/PV-MARKDOWN-030--composable-markdown-ast-pipeline-and-callout-rendering/`.
- Review examples: https://github.com/go-go-golems/publish-vault/pull/30#discussion_r4541204043 and https://github.com/go-go-golems/publish-vault/pull/30#discussion_r4541204055.
- Diary and remaining implementation task: this ticket's `reference/01-implementation-diary.md` and `tasks.md`.
