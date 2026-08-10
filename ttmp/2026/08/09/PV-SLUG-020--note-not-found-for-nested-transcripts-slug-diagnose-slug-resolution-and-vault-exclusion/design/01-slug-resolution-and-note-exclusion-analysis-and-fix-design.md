---
Title: 'Slug Resolution and Note Exclusion: Analysis and Fix Design'
Ticket: PV-SLUG-020
Status: active
Topics:
    - slug
    - routing
    - parser
    - vault
    - api
    - frontmatter
    - ignore
    - retro-obsidian-publish
    - obsidian-vault
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/go-go-parc/.vault-ignore
      Note: the real vault's only exclusion rule is ttmp/_*/ - proves the note was not ignored (excludes 22 files, none under Transcripts/)
    - Path: repo://publish-vault/internal/parser/parser.go
      Note: slugify (228) is the lossy transform defining every note URL; keeps [a-z0-9-_/] so underscores and slashes survive, trims '-' but NOT trailing '/', and maps Cyrillic/CJK to the empty string
    - Path: repo://publish-vault/pkg/api/api.go
      Note: registers /api/notes/{slug:.*} and getNote, whose 404 body carries no reason for why the note is absent
    - Path: repo://publish-vault/pkg/server/runtime.go
      Note: RuntimeState.Reload (100) keeps the stale snapshot when a reload fails, creating the window where a committed note is still reported missing
    - Path: repo://publish-vault/pkg/server/server.go
      Note: newSSRProxy (322) falls back to the SPA only on 5xx, so the sidecar's bare-text 404 reaches the browser verbatim
    - Path: repo://publish-vault/pkg/vault/vault.go
      Note: pathToSlug (893), LoadAll (145) with four silent note-drop paths, and GetNote (725) which is a bare exact-match map lookup with no normalization
    - Path: repo://publish-vault/web/server.mjs
      Note: ROOT CAUSE - fetchAPI (83-91) collapses genuine-404, unreachable, non-2xx and unparseable-body into null, and 242-245 turns null into the literal user-visible 404 'Note not found'
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-09T21:31:06.218992768-04:00
WhatFor: ""
WhenToUse: ""
---


# Slug Resolution and Note Exclusion: Analysis and Fix Design

> **Audience.** You have never seen this codebase. This document is meant to be
> read top to bottom. It explains how a URL in a browser becomes a note on the
> screen, where that chain can break, what actually broke, and what to change.
> Every symbol is cited as `file.go:line` against commit `9fbbd42` of
> `publish-vault`.

## 1. The symptom

A user reported that this URL fails:

```
https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields
```

with the message **"Note not found"**. The note plainly exists: it is a 209 KB
Markdown thesis sitting in the vault at

```
/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md
```

The obvious hypothesis is that the slugifier mangled the path — the URL segment
`the_algebra_of_intervention_fields` has underscores, the directory has spaces
and capitals, and the path is five levels deep. That hypothesis is wrong, and
proving it wrong is most of the value of this investigation. The slug is
generated exactly as the URL spells it. Something else produced that message.

The first thing to understand is that **the string "Note not found" is produced
in three different places by three different layers**, and they are not
interchangeable. Knowing which one you are looking at tells you most of what you
need:

| Where | Code | HTTP status | Content-Type | What it looks like |
|---|---|---|---|---|
| SSR sidecar (Node/Express) | `web/server.mjs:243` | **404** | `text/plain` | A bare, unstyled page with the two words `Note not found` |
| React app, note route | `web/src/components/pages/NotePage/NotePage.tsx:103` | 200 | `text/html` | Styled app shell, red alert icon, "Note not found" + "Slug: …" |
| React app, unmatched route | `web/src/App.tsx:217` | 200 | `text/html` | Styled app shell, `404 — Note not found` heading |
| Go API | `pkg/api/api.go` `getNote` | 404 | `text/plain` | `{"error":"note not found"}` |

The user saw the SSR flavour — a bare white page — which is what you get from
`web/server.mjs:243`.

### 1.1 What the server returns today

Measured against production on 2026-08-09 (times UTC):

```console
$ curl -s -i 'https://parc.yolo.scapegoat.dev/api/notes/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields' | head -5
HTTP/2 200
content-type: application/json
date: Mon, 10 Aug 2026 00:37:11 GMT
vary: Origin

{"slug":"transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields","title":"The Algebra of Intervention Fields","path":"Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md", ...
```

```console
$ curl -s -o note.html -D - -w 'http=%{http_code} size=%{size_download} time=%{time_total}\n' \
    'https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields'
HTTP/2 200
content-type: text/html; charset=utf-8
etag: W/"1cb299-3ccSHy4pA6pFXc0WKQzC5id8w0s"
link: <https://.../the_algebra_of_intervention_fields>; rel="canonical", <....md>; rel="alternate"; type="text/markdown"
x-powered-by: Express
content-length: 1880729
http=200 size=1880729 time=2.428034

$ grep -o '<title>[^<]*</title>' note.html
<title>The Algebra of Intervention Fields — PARC</title>
```

**The exact URL the user reported now returns HTTP 200 with the correct title.**
So the failure was not a permanent property of that slug. Two things follow: the
slug pipeline is innocent, and the failure mode is state-dependent. Both are
proven below.

### 1.2 The symptom reproduced deterministically, in production

Append one character — a trailing slash — and the exact user-visible symptom
comes back, reliably, on the live host:

```console
$ curl -s -o /dev/stdout -w '\nhttp=%{http_code} size=%{size_download}\n' \
    'https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields/'
Note not found
http=404 size=14

$ curl -s -o /dev/stdout -w '\nhttp=%{http_code} size=%{size_download}\n' \
    'https://parc.yolo.scapegoat.dev/api/notes/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields/'
{"error":"note not found"}
http=404 size=27
```

That is byte-for-byte the reported symptom, from a URL that differs from the
working one by a single `/`.

### 1.3 The other symptom, observed live

While running the variant matrix above, the production host stopped answering
entirely:

```console
$ for v in "/note/$S" "/note/$S/" "/note/$S%20" ... ; do curl ... ; done
/note/transcripts/.../the_algebra_of_intervention_fields    503 20 no available server
/note/transcripts/.../the_algebra_of_intervention_fields/   503 20 no available server
/note/transcripts/.../the_algebra_of_intervention_fields%20 503 20 no available server
/note/Transcripts/2026/08/09/Designing%20RAG%20Abstractions/... 503 20 no available server
/api/notes/transcripts/.../the_algebra_of_intervention_fields/  503 20 no available server
```

`no available server` is the ingress reporting that no pod endpoint is ready. It
recovered within about ten seconds:

```console
00:44:47 api/config=503 no available server
00:44:57 api/config=200 {"vaultName":"go-go-parc","pageTitle":"PARC","notes":1712}
```

This matters because, as section 6 shows, **an unavailable backend and a missing
note produce the same "Note not found" page**. The instability itself is being
investigated separately as PV-MEMORY-019; this ticket only records the
interaction.

## 2. How a URL becomes a note

Read this section once and the rest of the document is mechanical. There are
seven hops, and the two ends of the chain independently compute a string that
must match exactly.

```mermaid
flowchart TD
    A["Browser<br/>GET /note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields"] --> B["Ingress / Traefik"]
    B --> C["Go server :8080<br/>pkg/server/server.go"]
    C --> D{"agentPageHandler<br/>pkg/server/agent_markdown.go:19"}
    D -->|"path ends in .md, or Accept: text/markdown"| E["renderNoteMarkdown → GetNote"]
    D -->|"otherwise"| F["newSSRProxy<br/>pkg/server/server.go:322"]
    F --> G["SSR sidecar :8089 (Node/Express)<br/>web/server.mjs:202"]
    G --> H["parseRoute<br/>web/server.mjs:94<br/>slug = path.replace(/^\\/note\\//, '')"]
    H --> I["fetchAPI('/api/notes/' + encodeURIComponent(slug))<br/>web/server.mjs:83-91, 235"]
    I --> J["Go API :8080<br/>mux route /api/notes/{slug:.*}<br/>pkg/api/api.go Register"]
    J --> K["Handler.getNote<br/>pkg/api/api.go"]
    K --> L["Vault.GetNote(slug)<br/>pkg/vault/vault.go:725"]
    L --> M[("v.notes map[string]*Note<br/>EXACT-MATCH key lookup")]
    I -->|"null for ANY reason"| X["res.status(404).send('Note not found')<br/>web/server.mjs:242-245"]
    K -->|"!ok"| Y["404 {\"error\":\"note not found\"}"]

    N["filepath.Walk at load time<br/>pkg/vault/vault.go:145 LoadAll"] --> O["loadNote<br/>pkg/vault/vault.go:201"]
    O --> P["pathToSlug(relPath)<br/>pkg/vault/vault.go:893"]
    P --> Q["parser.Slugify<br/>internal/parser/parser.go:228"]
    Q --> M

    style M fill:#ffe8b3,stroke:#b8860b,stroke-width:2px
    style X fill:#ffd0d0,stroke:#c00,stroke-width:2px
    style Y fill:#ffd0d0,stroke:#c00,stroke-width:2px
```

The same picture as ASCII, for terminals:

```
  REQUEST SIDE                                    LOAD SIDE (startup / reload)
  ------------                                    ---------------------------
  browser URL                                     filepath.Walk(vault root)
      |  /note/<slug>                                   |  Transcripts/2026/08/09/
      v                                                 v  Designing RAG Abstractions/
  Traefik ingress                                  loadNote()   The_Algebra_...md
      |                                                 |
      v                                                 v
  Go :8080 agentPageHandler ----.                  pathToSlug()  = TrimSuffix ".md"
      |  (not *.md, not Accept:md)|                     |          + parser.Slugify
      v                           |                     v
  newSSRProxy --> SSR :8089       |            "transcripts/2026/08/09/
      |                           |             designing-rag-abstractions/
      |  parseRoute -> slug       |             the_algebra_of_intervention_fields"
      |  encodeURIComponent(slug) |                     |
      v                           |                     v
  GET /api/notes/<slug>  <--------'          +----------------------------+
      |                                      |  v.notes[slug] = *Note     |
      v                                      |  map — EXACT MATCH ONLY    |
  mux {slug:.*} -> getNote -----------------> +----------------------------+
      |                                              ^
      |  miss -> 404                                 |
      v                                    THE TWO SIDES MUST AGREE
  SSR: 404 "Note not found"                BYTE FOR BYTE. Nothing
                                           normalizes either side.
```

The single most important fact in this diagram: **`Vault.GetNote`
(`pkg/vault/vault.go:725`) is a bare Go map lookup.**

```go
// pkg/vault/vault.go:725
func (v *Vault) GetNote(slug string) (*Note, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	n, ok := v.notes[slug]
	return n, ok
}
```

There is no case folding, no trailing-slash trimming, no NFC/NFD Unicode
normalization, no re-slugification of the incoming key, and no fallback to the
wiki-link index. If the request string and the load-time string differ by one
byte, the answer is "not found", and the user is told the note does not exist.

## 3. The slug algebra

`slugify` is the function that decides what a note's URL is. It is 5 lines and
it is where every intern's intuition goes wrong, so here it is in full.

```go
// internal/parser/parser.go:228
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9\-_/]`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
```

Four steps, in order:

1. **Trim + lowercase.** `strings.ToLower` is ASCII-and-Unicode aware, so `Ä`
   becomes `ä` — but see step 2, which then destroys it.
2. **Replace every character outside `[a-z0-9-_/]` with a single `-`.** This is
   the lossy step. Note carefully what *survives*: lowercase ASCII letters,
   digits, hyphen, **underscore**, and **forward slash**. Underscores and slashes
   surviving is exactly why our note's slug is
   `.../designing-rag-abstractions/the_algebra_of_intervention_fields` — the
   underscores in the filename are preserved verbatim while the spaces in the
   folder name became hyphens.
3. **Collapse runs of hyphens** to one. `Cats & Dogs` → `cats---dogs` → `cats-dogs`.
4. **Trim leading/trailing hyphens.** Note this trims `-` only; it does **not**
   trim `/`.

`pathToSlug` is the only caller that matters for URLs:

```go
// pkg/vault/vault.go:893
func pathToSlug(relPath string) string {
	s := filepath.ToSlash(relPath)
	s = strings.TrimSuffix(s, ".md")
	return parser.Slugify(s)
}
```

### 3.1 Measured input → output table

These are not hand-derived; they are the recorded stdout of
`scripts/04-slug-algebra` run against `internal/parser`:

| Case | Input | `Slugify(input)` |
|---|---|---|
| the real note path | `Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields` | `transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields` |
| plain lowercase | `notes/hello` | `notes/hello` |
| uppercase | `Notes/Hello World` | `notes/hello-world` |
| single space | `hello world` | `hello-world` |
| repeated spaces | `hello    world` | `hello-world` |
| underscore kept | `the_algebra_of_fields` | `the_algebra_of_fields` |
| hyphen kept | `already-a-slug` | `already-a-slug` |
| slash kept | `a/b/c` | `a/b/c` |
| leading/trailing space | `␣␣padded␣␣` | `padded` |
| leading/trailing dash | `--edges--` | `edges` |
| dot | `v1.2.3 release` | `v1-2-3-release` |
| ampersand | `Cats & Dogs` | `cats-dogs` |
| apostrophe | `Manuel's Notes` | `manuel-s-notes` |
| colon | `Design: Part 1` | `design-part-1` |
| parens/brackets | `Branch (copy) [6a785ead]` | `branch-copy-6a785ead` |
| plus and hash | `C++ #tips` | `c-tips` |
| unicode accents | `Café Münster` | `caf-m-nster` |
| cyrillic | `Привет мир` | `""` (**empty**) |
| CJK | `日本語ノート` | `""` (**empty**) |
| emoji | `done ✅ shipped 🚀` | `done-shipped` |
| percent | `50% off %20 test` | `50-off-20-test` |
| **trailing slash** | `a/b/` | **`a/b/`** (slash survives!) |
| double slash | `a//b` | `a//b` |
| only punctuation | `!!!` | `""` |
| empty string | `""` | `""` |
| NBSP (U+00A0) | `hello world` | `hello-world` |
| tab | `hello\tworld` | `hello-world` |

Two rows deserve a red box.

**`a/b/` → `a/b/`.** Step 4 trims `-`, not `/`. So a trailing slash is a
*legal, load-bearing* character in slug space, but no note on disk ever produces
one (because `pathToSlug` starts from a file path). The consequence: the URL
`/note/<slug>/` asks for a key that can never exist. That is section 1.2's
deterministic reproduction, explained.

**Cyrillic and CJK → `""`.** A note named `Привет.md` gets the **empty slug**.
Every such note collides on the key `""`, only the last one wins, and it is
served at `/api/notes/` (empty tail matches `{slug:.*}`). Non-Latin note titles
are effectively unpublishable. The go-go-parc vault happens to have zero such
files today (measured: `files whose slug is the empty string (0)`), so this is
latent, not active — but it is one Russian filename away from being active.

### 3.2 Slugify is lossy, therefore collisions are real

The transformation is many-to-one. The audit script's collision probe on
synthetic input:

```
  COLLISION: "Designing RAG Abstractions"   and "Designing-RAG-Abstractions"   both -> "designing-rag-abstractions"
  COLLISION: "Cats & Dogs"                  and "Cats   Dogs"                  both -> "cats-dogs"
  COLLISION: "Cats & Dogs"                  and "Cats+Dogs"                    both -> "cats-dogs"
  COLLISION: "Cats & Dogs"                  and "Cats!Dogs"                    both -> "cats-dogs"
```

And on the **real** go-go-parc vault (`scripts/05-vault-slug-audit`), five slugs
are claimed by two files each:

```
== slug collisions: distinct files mapping to one slug (5 slugs) ==
  slug="transcripts/2026/07/28/chatgpt-transcript-zitadel-branding-setup" (2 files)
      Transcripts/2026/07/28/CHATGPT TRANSCRIPT - ZITADEL Branding Setup.md
      Transcripts/2026/07/28/CHATGPT TRANSCRIPT - Zitadel Branding Setup.md
  slug="transcripts/2026/08/06/branch-branch-branch-clim-ui-in-react/p06-typed-ports-binding-quotient-compiler" (2 files)
      .../P06-TYPED-PORTS-BINDING-QUOTIENT-COMPILER.md
      .../P06-typed-ports-binding-quotient-compiler.md
  slug="transcripts/2026/08/06/branch-branch-branch-clim-ui-in-react/presentation-based-ui-architectures-beyond-clim" (2 files)
  slug="transcripts/2026/08/06/branch-branch-clim-ui-in-react/presentation-based-ui-architectures-beyond-clim" (2 files)
  slug="transcripts/2026/08/06/branch-clim-ui-in-react/presentation-based-ui-architectures-beyond-clim" (2 files)
```

These are case-only differences on a case-sensitive filesystem. `LoadAll`
(`pkg/vault/vault.go:145`) writes `v.notes[note.Slug] = note` with no collision
check, so **whichever file `filepath.Walk` visits last silently wins** and the
other note is permanently unreachable. Walk order is lexical, so it is stable —
but it is stable by accident, not by design, and nothing logs the loss.

This is the general shape of the danger: `slugify` is a hash with no collision
handling, and the map write is last-write-wins and silent.

## 4. The exclusion paths

Independently of slugging, a note on disk can be deliberately kept out of
`v.notes`. There are exactly three mechanisms, all of which produce the same
user-visible 404 with no explanation. An intern debugging "why is my note not
published" must check all three.

### 4.1 `.vault-ignore`

A gitignore-subset file at the vault root, loaded by `ignore.Load`
(`internal/ignore/ignore.go`) in `vault.New` (`pkg/vault/vault.go:~262`).
Matching happens in `Vault.isIgnored` (`pkg/vault/vault.go:717`) via
`ignore.MatchAbs`, and is folded into `Vault.IsExcluded`
(`pkg/vault/vault.go:682`). `LoadAll` consults it twice: `ShouldPruneDir`
(`pkg/vault/vault.go:702`) to skip whole directories, and `IsExcluded` per file.

**What the real go-go-parc vault has** (`/home/manuel/code/wesen/go-go-golems/go-go-parc/.vault-ignore`, verbatim):

```gitignore
# .vault-ignore — paths excluded from the published vault everywhere
# (note index, file tree, search, backlinks, raw source, /vault-assets/, watcher).
# Syntax: documented gitignore subset. See publish-vault README.
# Takes effect on the next full reload (git-sync POSTs /api/admin/reload).

# docmgr authoring scaffolding — private, never publish
ttmp/_*/
```

One rule. It excludes `ttmp/_guidelines/` and `ttmp/_templates/` — and nothing
under `Transcripts/`. This was the leading suspect ("a note under `transcripts/`
is a plausible ignore hit") and it is **not** what happened.

Note the subtlety in `ShouldPruneDir`: pruning is skipped entirely when *either*
matcher contains a negation (`!`) pattern, because a `!` could re-include a file
beneath an ignored directory. With negations present the walk descends and pays
per-file matching cost. Correct, but it means adding one `!` rule changes load
performance globally.

### 4.2 `.publish/config.yaml` blacklist

Compiled by `vaultconfig.NewMatcher` and attached with `vault.WithConfig`
(`pkg/vault/vault.go:~103`); matched in `IsExcluded` (`pkg/vault/vault.go:682`)
against the vault-relative slash path. It is re-read on **every snapshot**, not
once at startup (`loadVaultConfig`, `pkg/server/runtime.go:~163`), so a reload
picks up edits and a git-sync symlink flip to a revision with different rules.

**What the real vault has:** *nothing.* The `.publish/` directory exists but
contains only widget page scripts:

```console
$ find .publish -type f
.publish/pages/reader.js
.publish/pages/tags.js
.publish/pages/recent.js
$ cat .publish/config.yaml
(no such file)
```

`vaultconfig.Load` on a missing file yields an empty config, and an empty
matcher excludes nothing. So this path is also **not** what happened.

Semantics worth memorising: exclusion is **"excluded if either"**. A `!`
negation in `.vault-ignore` cannot un-exclude something blacklisted in
`config.yaml`, and vice versa. To re-include a path you must edit whichever file
excluded it.

### 4.3 `publish: false` frontmatter

Read by `publishFlag` (`pkg/vault/vault.go:~250`) → `frontmatterBool`
(`pkg/vault/vault.go:~258`), stored on `Note.Publish`, and enforced in `LoadAll`:

```go
// pkg/vault/vault.go:~180
if !note.Publish {
	return nil          // parsed, then thrown away — no log line
}
v.notes[note.Slug] = note
```

`frontmatterBool` is case-insensitive on the key and accepts YAML booleans plus
the strings `true`/`false`/`yes`/`no`. Default is `true` (opt-out only). On the
live-reload path, `ReloadNote` (`pkg/vault/vault.go:605`) additionally *removes*
a previously-published note and returns `ErrUnpublished` so the watcher can drop
it from search.

**Our note's frontmatter** has `title`, `subtitle`, `author`, `date`, `lang`,
`rights`, `abstract` — and no `publish` key. So this path is also **not** what
happened.

### 4.4 The silent fourth exclusion: parse failure

This one is not documented as an exclusion mechanism but behaves as one:

```go
// pkg/vault/vault.go:~176
note, err := v.loadNote(path, info)
if err != nil {
	return nil // skip unparseable notes
}
```

If `parser.Parse` (`internal/parser/parser.go:56`) returns an error — malformed
YAML frontmatter, a goldmark conversion failure — the note vanishes with **no
log line at all**. For a 209 KB document with a multi-paragraph YAML block
scalar, this was a serious candidate. It is not what happened here (the note
loads, see section 5), but it is the hardest exclusion to diagnose in production
because it leaves no trace whatsoever.

### 4.5 How to tell which one fired

Today: you cannot, from outside. All four produce an identical 404. That is
itself a finding, and fix option F5 addresses it.

## 5. Proving the slug pipeline is innocent

`scripts/01-slug-probe` loads the real vault through the real `vault.New` +
`LoadAll` and asks the real `GetNote`. Recorded output:

```console
$ go run ./publish-vault/probe -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
    -slug 'transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields' \
    -grep intervention \
    -file '/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md'

== vault.New("/home/manuel/code/wesen/go-go-golems/go-go-parc") ==
loaded notes: 1712

== SlugForPath / IsExcluded for ".../The_Algebra_of_Intervention_Fields.md" ==
os.Stat err          : <nil>
SlugForPath          : "transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields"
IsExcluded(file)     : false
IsExcluded(parentDir): false
ShouldPruneDir(parent): false
  ancestor .../Transcripts/2026/08/09/Designing RAG Abstractions excluded=false prune=false
  ancestor .../Transcripts/2026/08/09                            excluded=false prune=false
  ancestor .../Transcripts/2026/08                               excluded=false prune=false
  ancestor .../Transcripts/2026                                  excluded=false prune=false
  ancestor .../Transcripts                                       excluded=false prune=false

== GetNote("transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields") ==
found: true
  title   : "The Algebra of Intervention Fields"
  path    : "Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md"
  htmlLen : 240761
ResolveWikiLink -> "transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields"

== slugs containing "intervention" ==
  transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields
      path=Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md
(1 matches)
```

Every hypothesis in the original brief is closed by that one run:

| Hypothesis | Verdict | Evidence |
|---|---|---|
| Slugifier mangles underscores | **No** | `_` is in the allowed class; slug has them verbatim |
| Multi-level path collapses | **No** | `/` is in the allowed class; all five levels survive |
| `.vault-ignore` hit | **No** | `IsExcluded=false` for the file and every ancestor; the file has one rule, `ttmp/_*/` |
| `.publish/config.yaml` blacklist hit | **No** | The file does not exist in the vault |
| `publish: false` frontmatter | **No** | No `publish` key; note is in `v.notes` |
| Unusual extension / case (`.MD`) | **No** | File is `.md`; loader lowercases before matching (`pkg/vault/vault.go:~168`) |
| Parse error swallowed | **No** | `htmlLen: 240761` — it parsed and rendered |
| Note simply isn't loaded | **No** | `found: true`, and production `/api/config` reports `notes: 1712`, identical to the local load |

And the HTTP-level matrix (`scripts/02-http-slug-matrix.sh`) against a locally
served copy of the same vault:

```console
base: http://127.0.0.1:18420
LABEL                              HTTP   BYTES      PATH
api raw slashes                    200    300360     /api/notes/transcripts/.../the_algebra_of_intervention_fields
api percent-encoded /              200    300360     /api/notes/transcripts%2F2026%2F08%2F09%2F...
api uppercase path                 000    0          /api/notes/Transcripts/2026/08/09/Designing RAG Abstractions/...
api trailing slash                 404    27         /api/notes/transcripts/.../the_algebra_of_intervention_fields/
api with .md suffix                404    27         /api/notes/transcripts/.../the_algebra_of_intervention_fields.md
api raw endpoint                   200    209223     /api/notes/transcripts/.../the_algebra_of_intervention_fields/raw
markdown mirror                    200    207731     /note/transcripts/.../the_algebra_of_intervention_fields.md
page route (SPA/SSR)               404    67         /note/transcripts/.../the_algebra_of_intervention_fields
api genuinely-missing              404    27         /api/notes/transcripts/2026/08/09/does-not-exist
config                             200    65         /api/config
```

Three rows need reading carefully:

- **`api percent-encoded /` returns 200.** The SSR sidecar builds its API URL
  with `encodeURIComponent(route.slug)` (`web/server.mjs:235`), which encodes the
  path separators as `%2F`. Go's `net/http` populates `r.URL.Path` with the
  *decoded* path and gorilla/mux matches on that, so `%2F` collapses back to `/`
  and `{slug:.*}` still matches. This looked like a bug and is not one — but it
  is load-bearing behaviour that no test covers.
- **`api uppercase path` shows `000`** — that is curl refusing to send a URL
  containing literal spaces, not a server response. It is a harness artifact; the
  correct conclusion is only that lookup is case-sensitive, which
  `GetNote` makes obvious.
- **`page route (SPA/SSR)` returns 404 with 67 bytes locally.** The body is
  `web bundle not found; run 'retro-obsidian-publish build web' first`. The local
  binary has no embedded SPA and no SSR sidecar, so the local page route cannot
  reproduce the SSR behaviour. That is why section 6 reproduces the SSR decision
  directly instead.

## 6. Root cause

**Stated plainly:**

> The note is correctly slugged, correctly parsed, and correctly indexed —
> slug generation and vault exclusion are *not* at fault. The user-visible
> "Note not found" is produced by `web/server.mjs:242-245`, which collapses
> **every** possible failure to obtain `/api/notes/<slug>` into a hard 404 whose
> body is the literal string `Note not found`. A genuine miss, an unreachable or
> restarting Go backend, a non-2xx response, and a body that fails to parse are
> indistinguishable to the user and to the logs. Because `Vault.GetNote` is an
> exact-match map lookup with no normalization, a lookup key that differs from
> the stored key by a single byte (a trailing slash is sufficient, and is
> reproducible in production today) lands in the same 404 as a backend that is
> simply down.

The offending code, verbatim:

```js
// web/server.mjs:83-91
async function fetchAPI(path) {
  try {
    const res = await fetch(`${API_BASE}${path}`);
    if (!res.ok) return null;      // 404 AND 500 AND 502 AND 503 -> null
    return await res.json();       // throws on truncated/non-JSON body
  } catch {
    return null;                   // connection refused, timeout, DNS, parse -> null
  }
}

// web/server.mjs:242-245
if (route.type === "note" && route.slug && !note) {
  res.status(404).type("text").send("Note not found");
  return;
}
```

`null` is a four-way overload: *absent*, *unreachable*, *erroring*,
*unparseable*. The 404 branch asserts the first.

### 6.1 The conflation, proven

`scripts/03-ssr-conflation-repro.mjs` copies `fetchAPI` verbatim from
`web/server.mjs` and drives it against a live local backend. Recorded output:

```console
$ node 03-ssr-conflation-repro.mjs http://127.0.0.1:18420
API_BASE = http://127.0.0.1:18420

A. note exists, API healthy        -> HTTP 200  "<title>The Algebra of Intervention Fields</title>"  (45ms)
B. note genuinely absent (API 404) -> HTTP 404  "Note not found"  (2ms)
C. API unreachable (fetch throws)  -> HTTP 404  "Note not found"  (1ms)
D. API 5xx (res.ok === false)      -> HTTP 200  "<title>undefined</title>"  (1ms)
E. non-JSON body (res.json throws) -> HTTP 404  "Note not found"  (1ms)
```

Rows **B, C and E are byte-identical**. A note that exists but whose backend is
unreachable (C) is reported to the user, to crawlers, and to the ETag cache as
*the note does not exist*. That is the bug.

### 6.2 Why this fired for the reported URL

The note was committed to the vault repo at `8da9f79` on **2026-08-09 18:04:31
-0400** and is present on `origin/main`
(`git ls-tree -r --name-only origin/main` lists it). Publication is git-sync
driven: a sidecar pulls the repo and `POST`s `/api/admin/reload`
(`pkg/server/server.go:119`), which builds a whole second snapshot and swaps it
(`RuntimeState.Reload`, `pkg/server/runtime.go:100`). Two windows exist between
"the author committed" and "the URL works", and both render the identical page:

1. **Not-yet-synced / not-yet-reloaded.** `Reload()` explicitly keeps the old
   snapshot on failure ("If loading or indexing fails, the previous state remains
   active"). Until a reload *succeeds*, `GetNote` misses → API 404 → SSR 404
   "Note not found". Genuinely correct behaviour, catastrophically bad message.
2. **Backend unavailable.** Observed live during this investigation: five
   consecutive requests returned `503 no available server` from the ingress, and
   the host recovered ten seconds later. In the window where the SSR sidecar is
   up but the Go API is not yet answering, `fetch` throws → row C → 404
   "Note not found" for a note that exists.

Window 2 is not hypothetical for this deployment. Loading this vault is
expensive — measured locally from `serve020.log`:

```
20:42:33 phase=load_start        heapAllocBytes=3460600
20:44:00 phase=load_vault_done   heapAllocBytes=168038392   notes=1712  duration=19.3s
20:44:00 phase=load_search_done  heapAllocBytes=1561762776  notes=1712  duration=1m2.5s
20:44:00 phase=load_done                                                duration=1m21.8s
```

**1.56 GB heap and 82 seconds** for one snapshot. A reload builds a *second*
one before swapping, so peak is roughly double. That is the subject of
PV-MEMORY-019 and is not re-investigated here; what matters for PV-SLUG-020 is
that the memory/restart behaviour is the *trigger* and the SSR 404 conflation is
the *reason the user was told the wrong thing*. Fixing the memory issue reduces
how often the window opens; only fixing the conflation makes the message true.

### 6.3 The second, independent defect

Separately and permanently: `/note/<slug>/` 404s (section 1.2), because
`slugify` preserves a trailing `/` and `GetNote` is exact-match. Any URL a user
can plausibly produce — a trailing slash from a copy-paste, a mixed-case path, a
`%20` — is a hard 404 with no suggestion, even though the correct note is one
normalization step away.

## 7. Pseudocode: current vs proposed

### 7.1 Current behaviour

```text
LOAD TIME (per markdown file):
    if excluded_by_vault_ignore_or_config: skip silently
    parsed = parse(file)
    if parse failed:                       skip silently        # no log
    if frontmatter.publish == false:       skip silently        # no log
    slug = slugify(trim_suffix(relpath, ".md"))
    notes[slug] = note                                          # last write wins,
                                                                # collisions silent

REQUEST TIME:
    slug = url_path.removeprefix("/note/")
    json = HTTP GET api_base + "/api/notes/" + urlencode(slug)
    if json is null_for_any_reason:                             # <-- ROOT CAUSE
        return 404, "Note not found"
    return 200, render(json)

API:
    note, ok = notes[slug]                                      # exact match only
    if not ok: return 404
```

### 7.2 Proposed behaviour

```text
LOAD TIME (per markdown file):
    reason = exclusion_reason(file)          # VAULT_IGNORE | CONFIG_BLACKLIST |
                                             # PUBLISH_FALSE | PARSE_ERROR | none
    if reason != none:
        excluded[relpath] = reason           # keep a diagnostic map
        log.Warn("note excluded", path, reason)
        continue
    slug = slugify(...)
    if slug == "":
        slug = fallback_slug(relpath)        # never allow the empty key
        log.Warn("note slug degenerate", path)
    if slug in notes:
        log.Warn("slug collision", slug, existing=notes[slug].path, new=relpath)
        slug = disambiguate(slug, relpath)   # deterministic suffix
    notes[slug] = note
    normalized_index[normalize(slug)] = slug # lowercase, strip trailing "/",
                                             # collapse "//", NFC

REQUEST TIME (SSR):
    slug = url_path.removeprefix("/note/")
    result = fetch_api("/api/notes/" + urlencode(slug))
    switch result.kind:
        case OK:              return 200, render(result.note)
        case NOT_FOUND:       return 404, not_found_page(slug, result.reason)
        case UNREACHABLE:     return 503, backend_unavailable_page()   # retryable
        case SERVER_ERROR:    return 502, backend_error_page()
        case BAD_BODY:        return 502, backend_error_page()

API:
    note, ok = notes[slug]
    if ok: return 200, note
    canonical, ok = normalized_index[normalize(slug)]              # fallback lookup
    if ok: return 308 redirect to "/api/notes/" + canonical
    reason = excluded[slug_to_relpath(slug)]                       # may be absent
    return 404, {"error": "note not found", "reason": reason or "no such slug"}
```

The single most important line in the proposal is the `switch` on
`result.kind`: a backend that is down must never be reported as a note that does
not exist.

## 8. API reference

Every symbol named in this document, with its citation.

| Symbol | Location | Role |
|---|---|---|
| `slugify` | `internal/parser/parser.go:228` | The lossy URL-safe transform; 4 steps, section 3 |
| `Slugify` | `internal/parser/parser.go` (exported wrapper) | Public entry point used by `pkg/vault` |
| `Parse` | `internal/parser/parser.go:56` | Markdown → `ParsedNote`; an error here silently drops the note |
| `wikiLinkHTML` | `internal/parser/parser.go:~205` | Builds `/note/<slugify(target)>` hrefs — the same algebra, applied to link text |
| `pathToSlug` | `pkg/vault/vault.go:893` | `relpath` → slug: `ToSlash`, strip `.md`, `Slugify` |
| `Vault.SlugForPath` | `pkg/vault/vault.go:636` | Absolute path → slug, without requiring the note to be indexed |
| `Vault.LoadAll` | `pkg/vault/vault.go:145` | The walk; holds the write lock for the whole scan; where notes are dropped |
| `Vault.loadNote` | `pkg/vault/vault.go:201` | Parses one file and computes its slug |
| `publishFlag` | `pkg/vault/vault.go:~250` | `publish` frontmatter → bool, default `true` |
| `frontmatterBool` | `pkg/vault/vault.go:~258` | Case-insensitive bool reader; accepts `true/false/yes/no` |
| `Vault.GetNote` | `pkg/vault/vault.go:725` | **Exact-match map lookup.** No normalization. The crux |
| `Vault.buildWikiLinkIndex` | `pkg/vault/vault.go:302` | Suffix + title index for `[[wiki links]]` — *not* consulted by `GetNote` |
| `Vault.ResolveWikiLink` | `pkg/vault/vault.go:415` | Short target → full slug; the fallback that already exists but is unused by the API |
| `Vault.IsExcluded` | `pkg/vault/vault.go:682` | Unified "`.vault-ignore` OR config blacklist" decision |
| `Vault.IsIgnored` | `pkg/vault/vault.go:669` | Watcher-facing alias, delegates to `IsExcluded` |
| `Vault.ShouldPruneDir` | `pkg/vault/vault.go:702` | Directory skip; disabled when either matcher has negations |
| `Vault.ReloadNote` | `pkg/vault/vault.go:605` | Single-file reload; returns `ErrIgnored` / `ErrUnpublished` |
| `Vault.ReadRaw` | `pkg/vault/vault.go:861` | Raw markdown; re-checks extension and exclusion |
| `ErrIgnored` / `ErrUnpublished` | `pkg/vault/vault.go:26` / `:34` | The only place exclusion is *named* — and only on the reload path |
| `Handler.Register` | `pkg/api/api.go` | Mounts `/api/notes/{slug:.*}` and `/api/notes/{slug:.*}/raw` |
| `Handler.getNote` | `pkg/api/api.go` | `GetNote` → 200 or `404 {"error":"note not found"}` |
| `RuntimeState.Reload` | `pkg/server/runtime.go:100` | Builds a full second snapshot, then swaps; old state survives failure |
| `loadSnapshot` | `pkg/server/runtime.go:~115` | `vault.New` + search index; emits the `memory phase=` log lines |
| `loadVaultConfig` | `pkg/server/runtime.go:~163` | Re-reads `.publish/config.yaml` per snapshot |
| `newSSRProxy` | `pkg/server/server.go:322` | Reverse proxy to the sidecar; **falls back to SPA only on 5xx** — a 404 passes through |
| `newAgentPageHandler` | `pkg/server/agent_markdown.go:19` | Serves `.md` mirrors and `Accept: text/markdown` before the SSR proxy |
| `fetchAPI` | `web/server.mjs:83` | **The conflation.** Returns `null` for four distinct failures |
| `parseRoute` | `web/server.mjs:94` | `/note/<x>` → `{type:"note", slug:x}`; does **not** trim a trailing slash |
| SSR 404 branch | `web/server.mjs:242-245` | Emits the literal `Note not found` |
| `extractNoteSlug` | `web/src/entry-server.tsx:36` | `decodeURIComponent` of the path tail for the SSR render |
| `NoteRoute` | `web/src/App.tsx:~199` | `decodeURIComponent(useParams()["*"])` |
| `NotFoundPage` | `web/src/App.tsx:217` | Client-side `404 — Note not found` |
| `NotePage` not-found branch | `web/src/components/pages/NotePage/NotePage.tsx:103` | Client-side styled "Note not found" |
| `getNote` query | `web/src/store/vaultApi.ts:89,101` | RTK Query; builds `/api/notes/${slug}` **without** encoding |

Note the asymmetry at the bottom of that table: the SSR sidecar encodes the slug
(`encodeURIComponent`, `web/server.mjs:235`) and the browser client does not
(`web/src/store/vaultApi.ts:101`). Both happen to work today. Neither is tested.

## 9. File reference table

| File | Why it matters |
|---|---|
| `/home/manuel/workspaces/2026-08-09/publish-vault-mathjax/publish-vault/web/server.mjs` | Contains the root cause: `fetchAPI` (83-91) collapses four failure modes to `null`, and 242-245 turns `null` into the literal user-visible `Note not found` 404 |
| `.../publish-vault/internal/parser/parser.go` | `slugify` (228) — the lossy, collision-prone, non-Unicode-safe transform that defines every note URL |
| `.../publish-vault/pkg/vault/vault.go` | `pathToSlug` (893), `LoadAll` (145), `GetNote` (725, exact-match), and all three exclusion mechanisms |
| `.../publish-vault/pkg/api/api.go` | The `{slug:.*}` route and `getNote`; the 404 that carries no reason |
| `.../publish-vault/pkg/server/runtime.go` | `Reload` (100) keeps the stale snapshot on failure — the eventual-consistency window |
| `.../publish-vault/pkg/server/server.go` | `newSSRProxy` (322) falls back to the SPA on 5xx but passes a sidecar 404 straight to the user |
| `/home/manuel/code/wesen/go-go-golems/go-go-parc/.vault-ignore` | The real vault's only exclusion rule (`ttmp/_*/`) — proves the note was not ignored |
| `/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md` | The 209 KB note under investigation; no `publish` key, parses cleanly |

## 10. Fix options and trade-offs

### F1 — Distinguish "not found" from "backend unavailable" in the SSR sidecar

Change `fetchAPI` to return a tagged result instead of `null`, and branch on it.

- **Pro:** Directly fixes the root cause. Small, local, ~30 lines. Makes the
  error message true, and makes 503s retryable by crawlers/CDNs instead of being
  cached as permanent 404s.
- **Con:** None material. Slightly more verbose call sites.
- **Risk:** Low. Behaviour change is visible in status codes only.

### F2 — Normalize the lookup key in `GetNote` (fallback lookup)

On an exact miss, retry against a normalized index (lowercased, trailing `/`
stripped, `//` collapsed, Unicode NFC), and 308-redirect to the canonical slug.

- **Pro:** Kills the trailing-slash class of bug permanently. Cheap: one extra
  map. Redirecting (rather than serving) keeps one canonical URL for SEO.
- **Con:** Must not mask genuine collisions; the normalized index needs its own
  collision policy.
- **Risk:** Medium — redirect loops if normalization is not idempotent. Measured:
  `Slugify` *is* idempotent for all tested inputs (section 3), so build the
  normalizer on top of it and assert idempotence in a test.

### F3 — Make slug generation lossless / reversible

Percent-encode instead of replacing, or store a `slug → path` sidecar mapping.

- **Pro:** Eliminates collisions and the empty-slug class outright. Non-Latin
  note titles become publishable.
- **Con:** **Breaks every existing URL**, every `[[wiki link]]` href already
  rendered, every external link, and the markdown mirrors. The whole point of the
  current slugs is that they are pretty.
- **Risk:** High. Not recommended as a first step. If ever done, it needs a
  permanent redirect table from old slugs.

### F4 — Log at WARN when a note is excluded or a slug collides

Add one log line per drop in `LoadAll`, naming the mechanism, plus a collision
warning on map overwrite.

- **Pro:** Turns the four silent exclusions into a greppable answer. Nearly free.
  Would have shortened this investigation from an hour to a minute.
- **Con:** Noise on vaults with large ignore rules — mitigate by logging a
  per-reason *count* at INFO and individual paths at DEBUG, except
  `PARSE_ERROR`, which should always be WARN.
- **Risk:** Low.

### F5 — A diagnostic endpoint: "why is this path not published?"

`GET /api/notes/_diagnose?path=<vault-relative-path>` returning the computed
slug, whether the file was seen, which exclusion fired, and any collision.

- **Pro:** Makes the system self-explaining; the single highest-leverage
  operator tool here.
- **Con:** New surface area; must not leak the content of excluded notes (return
  the *reason* only, never the body), and should be gated the same way the admin
  reload endpoint is.
- **Risk:** Low-medium (information disclosure if built carelessly).

### F6 — A 404 page that says which reason applied

Have the API's 404 body carry `reason`, and have the SSR/React 404 render it.

- **Pro:** Users self-diagnose (`publish: false` is the common author mistake).
- **Con:** Do not reveal that an excluded path *exists* to anonymous users —
  render a generic message publicly and the detailed reason only for
  loopback/authenticated callers.
- **Risk:** Medium (information disclosure), fully mitigated by the above.

### Recommendation

Do **F1 + F4 first**, in that order — they are small, safe, and between them
they turn every remaining instance of this bug into something a log line
answers. Then **F2**, which permanently removes the trailing-slash and
case-variant class. Then **F5**, then **F6** with the disclosure guard. Do
**not** do F3.

F1 is the actual fix for the reported symptom. F4 is what makes the next one
take five minutes.

## 11. Phased implementation plan

Phases map 1:1 to the docmgr tasks on this ticket.

### Phase 1 — Make the SSR sidecar tell the truth (F1)

Rework `fetchAPI` (`web/server.mjs:83`) to return
`{kind: "ok"|"not_found"|"unreachable"|"server_error"|"bad_body", data}` and
branch at `web/server.mjs:242` accordingly: 404 only for `not_found`, 503 for
`unreachable`, 502 for `server_error`/`bad_body`.

```bash
cd publish-vault/web && pnpm test
node ttmp/.../scripts/03-ssr-conflation-repro.mjs http://127.0.0.1:18420   # rows B/C/E must now differ
```

### Phase 2 — Log every silent drop (F4)

Add WARN/INFO logging in `LoadAll` (`pkg/vault/vault.go:145`) for each of:
ignore, config blacklist, `publish: false`, parse error, and slug collision on
map overwrite.

```bash
go test ./pkg/vault/... -count=1
go run ./cmd/retro-obsidian-publish serve --vault <vault> --port 18420 2>&1 | grep -E 'excluded|collision'
```

### Phase 3 — Normalized fallback lookup (F2)

Build `normalizedIndex` alongside `notes` in `LoadAll`; on an exact miss in
`getNote` (`pkg/api/api.go`), look up the normalized key and 308-redirect to the
canonical slug.

```bash
go test ./pkg/vault/... ./pkg/api/... -count=1
bash ttmp/.../scripts/02-http-slug-matrix.sh http://127.0.0.1:18420   # trailing-slash row must become 308/200
```

### Phase 4 — Guard the degenerate-slug and collision cases

Refuse to store the empty slug; give colliding slugs a deterministic
disambiguating suffix instead of last-write-wins.

```bash
go run ./ttmp/.../scripts/05-vault-slug-audit -vault <vault>   # collisions section must be empty
```

### Phase 5 — Diagnostic endpoint (F5) and reasoned 404 (F6)

Add `/api/notes/_diagnose`, gated like `/api/admin/reload`
(`pkg/server/server.go:119`), and thread `reason` into the 404 body with the
public/private disclosure split.

```bash
go test ./pkg/server/... -count=1
curl -s 'http://127.0.0.1:18420/api/notes/_diagnose?path=Transcripts/2026/08/09/...md' | jq
```

## 12. Regression tests to add

### 12.1 `internal/parser` — pin the slug algebra

Table test `TestSlugifyTable` in `internal/parser/parser_test.go`, one row per
line of section 3.1. The rows that must not silently change:

| name | in | want |
|---|---|---|
| `underscores_survive` | `The_Algebra_of_Intervention_Fields` | `the_algebra_of_intervention_fields` |
| `nested_path_survives` | `Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields` | `transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields` |
| `spaces_to_single_dash` | `hello    world` | `hello-world` |
| `dots_become_dashes` | `v1.2.3 release` | `v1-2-3-release` |
| `ampersand` | `Cats & Dogs` | `cats-dogs` |
| `apostrophe` | `Manuel's Notes` | `manuel-s-notes` |
| `accents_are_mangled` | `Café Münster` | `caf-m-nster` |
| `cyrillic_is_empty` | `Привет мир` | `""` |
| `cjk_is_empty` | `日本語ノート` | `""` |
| `emoji_dropped` | `done ✅ shipped 🚀` | `done-shipped` |
| `trailing_slash_survives` | `a/b/` | `a/b/` |
| `double_slash_survives` | `a//b` | `a//b` |
| `nbsp_is_a_dash` | `hello world` | `hello-world` |
| `idempotent` | `Slugify(Slugify(x))` | `== Slugify(x)` for every row above |

### 12.2 `pkg/vault` — lookup and exclusion

- `TestGetNoteTrailingSlashMisses` — index one note, assert
  `GetNote(slug+"/")` is `(nil,false)` **today**, and after Phase 3 assert the
  normalized index resolves it.
- `TestSlugCollisionIsReported` — two files differing only in case; assert both
  are reachable (Phase 4) rather than one silently winning.
- `TestEmptySlugIsRejected` — a file named `Привет.md`; assert it does not land
  on the `""` key.
- `TestExcludedNoteReportsReason` — one file per mechanism (`.vault-ignore`,
  config blacklist, `publish: false`, malformed YAML) asserting the recorded
  reason.

### 12.3 `pkg/api` — route shapes

Table test over paths, asserting status: raw slashes → 200; `%2F`-encoded → 200
(pins the SSR contract, currently untested); `.md` suffix → 404; genuinely
missing → 404 with `reason`; trailing slash → 308 after Phase 3.

### 12.4 `web` — the conflation

A vitest/node test over a stub server asserting the four `fetchAPI` outcomes map
to 404 / 503 / 502 / 502 respectively and that only the genuine miss produces
`Note not found`. `scripts/03-ssr-conflation-repro.mjs` is the executable
specification for this test.

## 13. Open questions and what I could not verify

- **Which of the two windows the user actually hit** — a stale snapshot (the
  note not yet reloaded) or an unavailable backend. Both produce the identical
  page, which is precisely the defect. Distinguishing them needs pod logs or
  ingress access logs for the moment of the report; I have neither. The `503 no
  available server` I observed proves window 2 is live on this deployment, and
  the commit timestamp (18:04) versus my first successful fetch (20:37) leaves
  ample room for window 1.
- **Kubernetes manifests were not readable from here.** `publish-vault/deploy/`
  contains only `gitops-targets.json`, which points at
  `wesen/2026-03-27--hetzner-k3s` at
  `gitops/kustomize/retro-obsidian-publish/deployment.yaml`. Memory limits,
  probe timings, replica count, and the git-sync reload wiring live in that other
  repo and were not inspected. Whether the reload sidecar retries a failed reload
  is therefore **unverified**.
- **Whether a failed reload is retried or leaves the vault stale indefinitely.**
  `RuntimeState.Reload` returns the error to its caller
  (`reloadHandler`, `pkg/server/server.go:211`); what the git-sync sidecar does
  with a non-2xx is out of this repo.
- **The `api uppercase path` matrix row is inconclusive** (`000`) because curl
  refuses literal spaces in a URL. Case-sensitivity is established from
  `GetNote`'s implementation instead, not from that row.
- **The local page route could not exercise SSR.** The locally built binary has
  no embedded web bundle (`web bundle not found; run 'retro-obsidian-publish
  build web' first`) and no Node sidecar was run, so section 6.1 reproduces the
  SSR *decision* with the verbatim `fetchAPI` source rather than through Express.
  Building the full SPA + SSR bundle would strengthen this but does not change
  the conclusion, since the branch is four lines long.
- **Not quantified:** whether any note in the vault is currently unreachable
  purely because of a slug collision *and* is one a user would look for. The
  audit found 5 colliding slug pairs (all case-only variants of the same
  content), 22 files excluded by the single `ttmp/_*/` rule, and 0 empty slugs;
  a deeper per-note reachability sweep would require another full 82-second
  vault load and was skipped for time.
- **Interaction with PV-MEMORY-019 is noted, not investigated.** The 1.56 GB /
  82 s snapshot cost and the double-snapshot reload are recorded here only
  because they explain how often the "Note not found" window opens. Ownership of
  that analysis stays with PV-MEMORY-019.

## 14. Reproduction scripts

All under this ticket's `scripts/`. They were run from the workspace root with a
pristine `git archive HEAD` export of `publish-vault` (the working tree carries
another agent's in-progress `internal/parser/math.go` change, which does not
compile: `publish-vault/internal/parser/parser.go:66:13: declared and not used:
mathSpans`).

| Script | What it proves |
|---|---|
| `01-slug-probe/main.go` | The note loads, its slug is exactly the URL, and no exclusion fired |
| `02-http-slug-matrix.sh` | Which URL spellings resolve over HTTP; isolates routing from lookup |
| `03-ssr-conflation-repro.mjs` | `fetchAPI`'s four failure modes collapse to one 404 — the root cause |
| `04-slug-algebra/main.go` | The measured slug table, idempotence, and collisions |
| `05-vault-slug-audit/main.go` | Real-vault counts: files on disk vs served, collisions, empty slugs |
