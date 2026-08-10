---
Title: Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://publish-vault/internal/parser/parser.go
      Note: read in Step 2; slugify's final strings.Trim(s, "-") not trimming '/' is the trailing-slash reproduction
    - Path: repo://publish-vault/pkg/vault/vault.go
      Note: read in Step 2 and exercised via vault.New/LoadAll/GetNote in scripts/01-slug-probe
    - Path: repo://publish-vault/scripts/smoke-ssr-upstream-failures.mjs
      Note: Regression test proving the four failure modes are now distinguishable
    - Path: repo://publish-vault/web/server.mjs
      Note: |-
        the four lines the investigation converged on; copied verbatim into scripts/03-ssr-conflation-repro.mjs
        The root-cause fix implemented in Step 10 (commit 878e372)
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-09T21:31:06.345231951-04:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Capture the investigation of a production "Note not found" report for
`https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields`:
what I suspected, what I measured, which hypotheses died and how, and where the
defect actually lives. No application source was changed on this ticket — the
deliverable is analysis, five reproduction scripts, and a fix design.

## Step 1: Orient in the workspace and locate the note on disk

I started by mapping the three moving parts: the Go workspace at
`/home/manuel/workspaces/2026-08-09/publish-vault-mathjax` (a `go.work` over
`./glazed` and `./publish-vault`), the `publish-vault` repo itself on branch
`task/publish-vault-mathjax`, and the actual Obsidian vault. The brief said the
vault was at one of two paths; only
`/home/manuel/code/wesen/go-go-golems/go-go-parc` exists.

The first thing I noticed was promising and turned out to be a red herring: the
vault directory is `Transcripts/` with a capital T, while the URL says
`transcripts/`. A case mismatch is exactly the kind of thing that breaks an
exact-match lookup, so I chased it. `find -iname "*intervention*"` located the
real file immediately at
`Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md`
— 209 KB, with spaces and capitals in the folder name and underscores in the
filename. Every character class the brief flagged as suspicious, in one path.

### Prompt Context

**User prompt (verbatim):** "Launch another subagent to figure out why this https://parc.yolo.scapegoat.dev/note/transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields fails with Note not found. Same thing: ticket, design doc, diary, commit"

**Assistant interpretation:** Find the concrete reason that one specific note URL
returns "Note not found" on the production PARC site, prove it with a local
reproduction rather than a plausible story, and package the result as a docmgr
ticket containing an intern-readable design document, an implementation diary,
tasks, and focused commits.

**Inferred user intent:** The user has a live site with a note that should be
reachable and is not. They want the actual mechanism identified — not a guess —
so a fix can be scheduled, and they want the reasoning preserved in the ticket
system alongside the two sibling investigations (PV-MATHJAX-018, PV-MEMORY-019)
rather than lost in a chat log.

### What I did
- Read `.ttmp.yaml`, `go.work`, and `git status`/`git log` in `publish-vault`.
- Confirmed the vault path: `/home/manuel/code/wesen/go-go-golems/go-go-parc`.
- `find . -iname "*intervention*"` → the `.md` plus `.pdf`, `.docx`, and a
  `_Source.zip` sibling.
- Read the note's first 2 KB: a YAML frontmatter block with `title`, `subtitle`,
  `author`, `date`, `lang`, `rights`, and a multi-paragraph `abstract: |` scalar.
- Checked `git log` for the file and `git ls-tree -r origin/main` for the folder.

### Why
Before theorising about the slugifier I needed to know that the file exists, what
its exact path is, whether it is committed and pushed (publication is git-sync
driven, so an unpushed file is trivially "not found"), and whether its
frontmatter carries anything that would exclude it.

### What worked
The file is real, committed at `8da9f79` ("`:art: Clean up transcripts`",
2026-08-09 18:04:31 -0400), and present on `origin/main` — `git ls-tree` lists
it. Its frontmatter has no `publish` key. So "the author never pushed it" and
"`publish: false`" were both dead on arrival.

### What didn't work
Nothing failed here, but the `Transcripts` vs `transcripts` case difference sent
me down a wrong path for a few minutes. `slugify` lowercases as its very first
operation, so the capital T is a non-issue; I only established that by reading
the function rather than by reasoning about the directory listing.

### What I learned
The sibling files in that folder (`.pdf`, `.docx`, `_Source.zip`, dozens of
`user-…jpg-<hash>` attachments) are untracked in git while the `.md` is
committed. That is fine for publication — the loader only indexes `.md` as notes
— but it means `git status` on that directory is extremely noisy, and I nearly
misread "lots of `??` lines" as "the note is untracked".

### What was tricky to build
N/A — orientation step, no code.

### What warrants a second pair of eyes
Nothing yet.

### What should be done in the future
N/A.

### Code review instructions
Start at `/home/manuel/code/wesen/go-go-golems/go-go-parc/.vault-ignore` and the
note file itself; both are quoted verbatim in the design doc §4.1 and §5.

### Technical details
```console
$ find . -iname "*intervention*" -not -path "./.git/*"
./Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md
./Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.pdf
./Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.docx
./Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields_Source.zip

$ git log --oneline -3 -- "Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields.md"
8da9f79 :art: Clean up transcripts
```

## Step 2: Read the slug pipeline and the exclusion mechanisms

With the file located I read the two functions the brief pointed at —
`slugify` (`internal/parser/parser.go:228`) and `pathToSlug`
(`pkg/vault/vault.go:893`) — plus `LoadAll`, `loadNote`, `GetNote`,
`IsExcluded`, `ShouldPruneDil`, and the three exclusion mechanisms. The
character class `[^a-z0-9\-_/]` keeps `_` and `/`, so by hand the note's path
should slugify to exactly the string in the URL. That made the slugifier
hypothesis look wrong before I ran anything.

I then read the two exclusion configs in the real vault. `.vault-ignore` has a
single rule, `ttmp/_*/`, which cannot touch `Transcripts/`. `.publish/config.yaml`
does not exist at all — the `.publish/` directory contains only three widget page
scripts. Both leading suspects from the brief were therefore already implausible,
which is exactly when it becomes important to stop reasoning and start measuring.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Verify each named hypothesis against the code
rather than assuming, and identify every place a note can be dropped.

**Inferred user intent:** Avoid a plausible-sounding but wrong diagnosis.

### What I did
- Read `internal/parser/parser.go` lines 1–260 (`Parse`, `slugify`,
  `wikiLinkHTML`, `splitFrontmatter`).
- Read `pkg/vault/vault.go` lines 1–345 and 600–906 (`LoadAll`, `loadNote`,
  `publishFlag`, `frontmatterBool`, `buildWikiLinkIndex`, `GetNote`,
  `IsExcluded`, `ShouldPruneDir`, `ReadRaw`, `pathToSlug`).
- Read `pkg/api/api.go` in full.
- `cat` of the vault's `.vault-ignore`; `find .publish -type f`.

### Why
The design doc has to cite `file.go:line` for every symbol, and I cannot write
"the note was not excluded" without having read the exclusion code and the
operator config that drives it.

### What worked
Three findings that shaped everything after:

1. `GetNote` is a bare map lookup — `n, ok := v.notes[slug]` — with no
   normalization of any kind. Whatever key the request produces must match the
   load-time key byte for byte.
2. `LoadAll` drops notes in **four** ways, and three of them are completely
   silent: `IsExcluded`, `publish: false`, and — the one not documented as an
   exclusion — `if err != nil { return nil // skip unparseable notes }`. A
   209 KB file with a large YAML block scalar failing to parse would vanish
   without a single log line.
3. `Vault.ResolveWikiLink` and `buildWikiLinkIndex` already implement a
   forgiving suffix/title-based lookup — and `getNote` does not use it.

### What didn't work
My reading of the slugifier predicted the slug would match the URL, which meant
my leading hypothesis was dead and I had no replacement. I had to widen the
search to the whole request path rather than the slug pipeline.

### What I learned
`ShouldPruneDir` disables directory pruning entirely when *either* matcher has a
negation pattern, because a `!` rule could re-include a file beneath an ignored
directory. That is correct but it means adding one `!` to `.vault-ignore`
silently changes load cost for the whole vault.

Also: `IsIgnored` is a pure delegation to `IsExcluded`, kept only because the
file watcher calls it directly. Worth knowing so you do not read it as a second,
different matcher.

### What was tricky to build
N/A — reading step.

### What warrants a second pair of eyes
The silent `return nil` on parse error at `pkg/vault/vault.go:~176`. It is the
single hardest failure to diagnose in this system, and it is one line.

### What should be done in the future
Log every drop in `LoadAll` with its reason (design doc fix F4).

### Code review instructions
`pkg/vault/vault.go:145` (`LoadAll`) and `:725` (`GetNote`) are the two functions
that define the contract. Read them together.

### Technical details
```go
// internal/parser/parser.go:228
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9\-_/]`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
```
Note `strings.Trim(s, "-")` trims hyphens only — not slashes. That detail became
the deterministic reproduction in Step 5.

## Step 3: Build a repro that loads the real vault (`01-slug-probe`)

I wrote a small Go command under the ticket's `scripts/` that calls the real
`vault.New` + `LoadAll` against the real vault and then asks the real `GetNote`,
plus reports `SlugForPath` and `IsExcluded` for the note file and for each of its
ancestor directories. The point was to close every hypothesis in one run instead
of arguing about regexes.

The first attempt did not compile, because the working tree carries another
agent's in-progress MathJax change. I worked around it by exporting a pristine
`git archive HEAD` of `publish-vault` into a scratch directory with its own
`go.work`, which also guaranteed I was measuring committed behaviour rather than
someone else's half-finished edit.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Do not ship a theory that has not been executed.

**Inferred user intent:** A conclusion backed by recorded output.

### What I did
- Wrote `scripts/01-slug-probe/main.go` (flags `-vault`, `-slug`, `-grep`,
  `-file`).
- Ran it from the workspace root; hit a compile error from an unrelated file.
- Exported `git archive HEAD | tar -x` into a scratch workspace, wrote a
  `go.work` pointing at the real `./glazed` and the exported `./publish-vault`,
  and re-ran there.

### Why
`GetNote` is the exact function the API calls. Anything less than calling it with
the real vault loaded leaves room for doubt.

### What worked
The probe closed every hypothesis at once:

```
loaded notes: 1712
SlugForPath          : "transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields"
IsExcluded(file)     : false
IsExcluded(parentDir): false
  ancestor .../Transcripts   excluded=false prune=false
== GetNote("transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields") ==
found: true
  title   : "The Algebra of Intervention Fields"
  htmlLen : 240761
== slugs containing "intervention" ==
  (1 matches)
```

The slug is character-for-character the URL. The note parsed (240 KB of HTML).
Nothing excluded it. Underscores survived, all five path levels survived.

### What didn't work
The first run, verbatim:

```
# github.com/go-go-golems/publish-vault/internal/parser
publish-vault/internal/parser/parser.go:66:13: declared and not used: mathSpans
```

That is PV-MATHJAX-018's work in progress, not mine, and I must not touch it.

### What I learned
When several agents share a checkout, "run the tests" is not well defined. The
`git archive HEAD` export is a clean way to measure committed behaviour without
stashing, branching, or otherwise disturbing anyone — and it costs about two
seconds.

### What was tricky to build
The compile failure was the trick, and the underlying cause is structural: the
brief explicitly warns not to modify `internal/` or `pkg/`, while another agent
is actively editing exactly those files. Symptom: an unrelated compile error in a
package I never touched. Approach: rather than `git stash` (which would yank the
other agent's work out from under them) I exported the committed tree:

```bash
mkdir -p $SP/ws/publish-vault
git -C .../publish-vault archive HEAD | tar -x -C $SP/ws/publish-vault
printf 'go 1.26.5\n\nuse (\n\t/…/glazed\n\t./publish-vault\n)\n' > $SP/ws/go.work
cp .../go.work.sum $SP/ws/
```

Every subsequent Go run in this diary happened in that scratch workspace.

### What warrants a second pair of eyes
Nothing in the probe itself; it only reads.

### What should be done in the future
The probe is genuinely useful as an operator tool. Design doc fix F5 proposes
promoting its logic to a `/api/notes/_diagnose` endpoint.

### Code review instructions
`scripts/01-slug-probe/main.go`. It uses only exported API (`vault.New`,
`Count`, `SlugForPath`, `IsExcluded`, `ShouldPruneDir`, `GetNote`,
`ResolveWikiLink`, `AllNotes`), so it cannot drift from production behaviour.

### Technical details
Loading this vault is expensive: `vault.New` alone took ~28 s wall in the probe.
That number returns in Step 6.

## Step 4: Check production — and find it already works

With the slug pipeline exonerated I went to the live host. The API returned
**HTTP 200** with the full note JSON, and the SSR page route returned **HTTP 200**
with `<title>The Algebra of Intervention Fields — PARC</title>` and 1.88 MB of
HTML. `/api/config` reported `notes: 1712` — byte-identical to my local load.

That reframed the whole investigation. The reported URL is not permanently
broken; it is fine right now. So the question changed from "what is wrong with
this slug" to "what state can this system be in such that a correctly indexed
note reports as missing", which is a much better question.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Confirm the symptom against the real host before
designing a fix for it.

**Inferred user intent:** Fix the thing that actually broke.

### What I did
- `curl -i` the production `/api/notes/<slug>` and `/note/<slug>`.
- `curl` two sibling notes in the same folder and `/note/index` as controls.
- `curl /api/config` to compare the served note count against my local load.

### Why
A symptom that has already healed has a different root cause than a symptom that
reproduces. Establishing which one I had was the highest-value single check
available.

### What worked
```
api  : HTTP/2 200, content-type application/json, correct slug/title/path
page : http=200 size=1880729 time=2.428034
       <title>The Algebra of Intervention Fields — PARC</title>
siblings: the_semantics_and_dynamics_of_rag 200 (4.07 MB),
          compositional_retrieval_systems_thesis 200 (2.08 MB),
          index 200
/api/config: {"vaultName":"go-go-parc","pageTitle":"PARC","notes":1712}
```

`notes: 1712` matching my local count exactly proves production is fully synced
with the vault I audited — the deployed content is not stale *now*.

### What didn't work
My first combined `curl -i` of both endpoints produced 1.1 MB of output and was
spilled to a file by the harness, which cost me a round trip. Subsequent probes
used `-o /dev/null -w '%{http_code} %{size_download}'`.

### What I learned
These notes are enormous. One note is 1.1 MB of API JSON and 1.88 MB of rendered
HTML; a sibling is 4 MB. Any per-request memory or timeout budget on this
deployment is being tested by ordinary page views, which turned out to matter.

### What was tricky to build
N/A.

### What warrants a second pair of eyes
The conclusion that "it works now" does not mean "there is no bug" — it means the
bug is state-dependent. Reviewers should check I did not stop there.

### What should be done in the future
N/A for this step.

### Code review instructions
Design doc §1.1 quotes these responses verbatim.

### Technical details
`etag: W/"1cb299-…"` and `x-powered-by: Express` on the page route confirm the
HTML is served by the Node SSR sidecar, not by the Go binary — which told me
where to look next.

## Step 5: Find the deterministic reproduction — one trailing slash

`x-powered-by: Express` pointed at `web/server.mjs`. Reading it, two things
jumped out. `parseRoute` (`web/server.mjs:94`) strips `/note/` and keeps
everything after it *including a trailing slash*, and `slugify`'s final
`strings.Trim(s, "-")` trims hyphens but not slashes. So `/note/<slug>/` asks for
a key ending in `/`, which `pathToSlug` can never produce from a file path.

I tested it against production. It reproduces the reported symptom exactly:
HTTP 404, body `Note not found`, 14 bytes. One character's difference from a URL
that returns 200.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Find a way to make the exact symptom happen on
demand.

**Inferred user intent:** A reproduction, not a narrative.

### What I did
- Read `web/server.mjs` end to end (409 lines).
- Probed production with five URL variants: exact, trailing slash, trailing
  `%20`, original-case with `%20` for spaces, and the API with a trailing slash.
- Wrote `scripts/02-http-slug-matrix.sh` to run the same matrix against any host.

### Why
A deterministic reproduction converts an argument into a fact, and it gives the
regression test its first row.

### What worked
```console
$ curl '…/note/…/the_algebra_of_intervention_fields/'
Note not found
http=404 size=14

$ curl '…/api/notes/…/the_algebra_of_intervention_fields/'
{"error":"note not found"}
http=404 size=27
```

Byte-for-byte the reported symptom, on the live host, from a URL a user could
easily produce by copy-pasting with a trailing slash.

### What didn't work
Midway through the variant loop, **every** request started returning
`503 no available server` — the ingress reporting no ready backend. That was not
my doing in any deliberate sense, but I had just pulled several megabyte-scale
pages in quick succession. It recovered in about ten seconds:

```
00:44:47 api/config=503 no available server
00:44:57 api/config=200 {"vaultName":"go-go-parc","pageTitle":"PARC","notes":1712}
```

I stopped hammering and switched to single requests with 8-second pauses.

### What I learned
Two independent facts, and the second is the more important one:

1. The trailing slash is a real, permanent, reproducible defect.
2. The production host will drop to 503 under a handful of large-note requests.
   And — critically — a 503 from the Go backend does **not** reach the user as a
   503 on the page route. It reaches them as "Note not found", for reasons Step 6
   makes precise.

### What was tricky to build
Reproducing without further destabilising a live site. Symptom: my probe loop
knocked the host over. Approach: after the 503 I stopped batching, added `sleep 8`
between requests, and moved everything else I could to the local server built in
Step 6. The two production probes I did keep were the minimum needed to establish
the trailing-slash reproduction.

### What warrants a second pair of eyes
Whether `newSSRProxy` (`pkg/server/server.go:322`) should pass a sidecar 404
through untouched. It falls back to the SPA on 5xx but not on 4xx, which is why
the bare-text `Note not found` reaches the browser instead of the styled app.

### What should be done in the future
Normalize the lookup key (design doc fix F2): strip trailing `/`, collapse `//`,
case-fold, NFC, then 308-redirect to the canonical slug.

### Code review instructions
`web/server.mjs:94` (`parseRoute`) and `internal/parser/parser.go:228`
(`slugify`, final line) are the two halves of this bug. Neither is wrong on its
own.

### Technical details
The slug algebra confirms it directly — `scripts/04-slug-algebra` prints
`"a/b/" -> "a/b/"`. A trailing slash is a legal slug character that no note ever
has.

## Step 6: Stand up a local server and run the full matrix

To finish the matrix without touching production I built the binary from the
pristine export and served the real vault locally. Startup is slow and
memory-hungry, and the log it emits turned out to be the most useful artifact of
the whole investigation.

The local run also produced a clean negative result I had to be careful not to
over-read: the local page route 404s with `web bundle not found`, because the
binary has no embedded SPA and I ran no Node sidecar. That is a harness
limitation, not the bug.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Serve locally and curl the API path, as the brief
suggested.

**Inferred user intent:** A safe environment to probe exhaustively.

### What I did
- `go build -o $SP/rop ./publish-vault/cmd/retro-obsidian-publish` from the
  scratch workspace.
- Started it under tmux against the real vault, `--watch=false --serve-web`.
- Ran `scripts/02-http-slug-matrix.sh` against it.
- Read the `memory phase=` log lines it emits.

### Why
Ten HTTP probes against a 1712-note vault should not be aimed at a live site that
had just returned 503.

### What worked
The matrix, in full:

```
api raw slashes                    200    300360
api percent-encoded /              200    300360
api uppercase path                 000    0
api trailing slash                 404    27
api with .md suffix                404    27
api raw endpoint                   200    209223
markdown mirror                    200    207731
page route (SPA/SSR)               404    67
api genuinely-missing              404    27
config                             200    65
```

The `percent-encoded /` row is the one I most wanted. The SSR sidecar builds its
API URL with `encodeURIComponent(route.slug)` (`web/server.mjs:235`), which turns
the path separators into `%2F`. I expected that to break `{slug:.*}` matching.
It does not: Go decodes `%2F` back to `/` in `r.URL.Path` before gorilla/mux
matches. Load-bearing, correct, and covered by no test.

### What didn't work
Three things, all instructive:

1. My first tmux server bound port 18080 and failed: `Error: listen tcp :18080:
   bind: address already in use`. Another agent's server (PV-MEMORY-019) owns
   that port. I moved to 18420.
2. A readiness poll of mine timed out and was killed: `Exit code 143 / Command
   timed out after 2m 0s`. The vault takes ~82 s to load and I had been polling
   `/api/config` for a `notes` key while the process was still indexing.
3. `api uppercase path` returned `000` with 0 bytes — that is curl refusing a URL
   containing literal spaces, not a server response. The row is inconclusive and
   I labelled it as such in the design doc rather than pretending it proved
   case-sensitivity.

### What I learned
The startup log is the single best piece of evidence in this ticket:

```
20:42:33 phase=load_start        heapAllocBytes=3460600
20:44:00 phase=load_vault_done   heapAllocBytes=168038392   notes=1712  duration=19.3s
20:44:00 phase=load_search_done  heapAllocBytes=1561762776  notes=1712  duration=1m2.5s
20:44:00 phase=load_done                                                duration=1m21.8s
```

**1.56 GB of heap and 82 seconds** for one snapshot, of which 62 s is the search
index. And `RuntimeState.Reload` (`pkg/server/runtime.go:100`) builds a complete
*second* snapshot before swapping — so a reload peaks at roughly double that.
That explains both the 503 I saw and the existence of a long window during which
a freshly committed note is not yet visible.

### What was tricky to build
Waiting correctly for readiness. Symptom: the server binds its port and answers
immediately, but `/api/config` returns `404 page not found` for over a minute
because routes are registered only after the vault and search index finish
building — so a naive "is the port open" check succeeds far too early, and a
naive poll loop blows the tool timeout. Approach: poll for the *content*
(`case "$r" in *notes*)`) rather than for a status code, with an explicit
per-attempt `--max-time 3`, and give the wrapping call a 115 s budget.

### What warrants a second pair of eyes
Whether `LoadAll` holding `v.mu` for the entire 19-second walk is acceptable.
Readers block rather than see a half-built map, which is the safe choice, but it
means a `LoadAll` on the serving path would stall every request for ~20 s.
(`RuntimeState.Reload` avoids this by building off to the side — worth confirming
no other caller does it in-place.)

### What should be done in the future
Belongs to PV-MEMORY-019, deliberately not pursued here.

### Code review instructions
`pkg/server/runtime.go:100` (`Reload`) and `:115` (`loadSnapshot`). Note the
comment "If loading or indexing fails, the previous state remains active" — that
is the stale-snapshot window in one sentence.

### Technical details
The local page route's 404 body is
`web bundle not found; run 'retro-obsidian-publish build web' first` (67 bytes) —
recorded in the design doc so nobody mistakes it for the production symptom.

## Step 7: Isolate the root cause in the SSR sidecar

Reading `web/server.mjs` closely, the defect is four lines. `fetchAPI`
(`web/server.mjs:83-91`) returns `null` for *four different* conditions — a
genuine 404, a non-2xx of any kind, a thrown `fetch` (connection refused,
timeout, DNS), and a body that fails `res.json()`. Then `web/server.mjs:242-245`
interprets `null` as one specific thing and emits
`res.status(404).type("text").send("Note not found")`.

I proved the conflation by copying `fetchAPI` verbatim into a standalone repro
and driving it through all five scenarios against the live local backend. Three
of them are indistinguishable.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Name the root cause and prove it.

**Inferred user intent:** A defect a developer can go fix.

### What I did
- Wrote `scripts/03-ssr-conflation-repro.mjs`, copying `fetchAPI` verbatim from
  `web/server.mjs` (with a comment saying so) and reducing lines 234–245 to a
  single `ssrOutcome` decision.
- Ran it against the local server on 18420.

### Why
Paraphrasing the function would have made the repro prove something about my
paraphrase. Copying it verbatim makes it prove something about the shipped code.

### What worked
```
A. note exists, API healthy        -> HTTP 200  "<title>The Algebra of Intervention Fields</title>"  (45ms)
B. note genuinely absent (API 404) -> HTTP 404  "Note not found"  (2ms)
C. API unreachable (fetch throws)  -> HTTP 404  "Note not found"  (1ms)
D. API 5xx (res.ok === false)      -> HTTP 200  "<title>undefined</title>"  (1ms)
E. non-JSON body (res.json throws) -> HTTP 404  "Note not found"  (1ms)
```

**B, C and E are byte-identical.** A note that exists but whose backend is down
is reported to the user — and to crawlers, and to any caching layer — as a note
that does not exist. Combined with the live 503 from Step 5 and the 82 s / 1.56 GB
reload cost from Step 6, that is the mechanism behind the report.

### What didn't work
Scenario D is mislabelled in my script: I aimed it at `/api/search` expecting a
5xx and got a 200 with a JSON array, so it printed
`HTTP 200 "<title>undefined</title>"` instead of exercising the `!res.ok` branch.
The branch is still proven — B goes through `!res.ok` on a real 404 — but D as
written does not test what its label claims. I left it in with this note rather
than quietly rewriting it, because the honest record is more useful than a tidy
one.

### What I learned
The `null` return is a four-way overload, and the call site picks one meaning out
of four. This is the generic shape of the bug: an error channel that discards the
distinction between "absent" and "unavailable" will eventually tell a user that
something they are looking at does not exist.

Two further asymmetries fell out of the reading. The SSR sidecar encodes its slug
(`encodeURIComponent`, `web/server.mjs:235`) and the browser client does not
(`web/src/store/vaultApi.ts:101`) — both work, neither is tested. And
`newSSRProxy` falls back to the SPA on 5xx but passes a sidecar 404 straight
through, which is why the user gets bare text rather than the styled app shell.

### What was tricky to build
Proving an SSR behaviour without an SSR bundle. Symptom: the local binary has no
embedded web assets and building the full Vite SPA + SSR bundle via pnpm would
have been a long detour. Approach: since the decision under test is four lines,
I lifted those four lines verbatim (marked as such in the file header) and drove
them against a real backend. That gives a faithful test of the branch while
being explicit in the design doc's Open Questions that the Express layer itself
was not exercised.

### What warrants a second pair of eyes
The proposed fix changes status codes: `unreachable` becomes 503 and
`server_error`/`bad_body` become 502. Anything that caches by status — the CDN,
crawlers, the `etag` on the page route — will behave differently, and that is the
point, but it should be a deliberate decision.

### What should be done in the future
Design doc fixes F1 (tagged result) and F4 (log every silent drop), in that
order. F1 is the fix for this symptom; F4 is what makes the next one take five
minutes instead of an hour.

### Code review instructions
Read `web/server.mjs:83-91` and `:242-245` together — they are the whole bug.
Then `scripts/03-ssr-conflation-repro.mjs`, which is the executable spec for the
regression test.

### Technical details
```js
async function fetchAPI(path) {
  try {
    const res = await fetch(`${API_BASE}${path}`);
    if (!res.ok) return null;   // 404 AND 500 AND 502 AND 503
    return await res.json();    // throws on truncated / non-JSON body
  } catch {
    return null;                // refused, timeout, DNS, parse
  }
}
```

## Step 8: Quantify the slug algebra and audit the real vault

Two more scripts, both aimed at the design doc's intern audience. `04-slug-algebra`
prints `Slugify` over 27 representative inputs plus idempotence and collision
probes; `05-vault-slug-audit` walks the real vault and diffs "markdown on disk"
against "notes reachable at `/note/<slug>`".

Both turned up latent defects that are not the reported bug but are the same
family, and both gave the design doc measured numbers instead of assertions.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Give the design doc concrete, measured examples.

**Inferred user intent:** A document a new engineer can learn the system from.

### What I did
- `scripts/04-slug-algebra/main.go` — 27 inputs, idempotence check, collision
  probe.
- `scripts/05-vault-slug-audit/main.go` — disk-vs-index diff, collision report,
  empty-slug report, against the real vault.

### Why
"Underscores survive" is a claim. A table of 27 measured rows is a reference.

### What worked
The algebra table (design doc §3.1) plus two findings I did not expect:

- **Cyrillic and CJK slugify to the empty string.** `Привет мир` → `""`,
  `日本語ノート` → `""`. Every non-Latin-titled note would collide on the key
  `""` and be served at `/api/notes/`. The vault has zero such files today
  (measured), so it is latent — one Russian filename from being live.
- **`Slugify` is idempotent** for all 27 inputs, which is exactly the property
  fix F2's normalize-and-redirect needs to avoid redirect loops. Good news, and
  now assertable.

The vault audit:

```
markdown files on disk (excluding dot-dirs): 1740
notes indexed by vault.LoadAll            : 1713
difference (unreachable at /note/<slug>)  : 27

== on disk but NOT reachable (22) ==   all ttmp/_guidelines/ and ttmp/_templates/, excluded=true
== slug collisions (5 slugs) ==        all case-only filename variants
== files whose slug is the empty string (0) ==
```

The 22 unreachable files are the `.vault-ignore` rule `ttmp/_*/` doing precisely
its job — a clean confirmation that the exclusion machinery works and simply did
not fire for our note. The 5 collisions are real: pairs like
`CHATGPT TRANSCRIPT - ZITADEL Branding Setup.md` and
`CHATGPT TRANSCRIPT - Zitadel Branding Setup.md` map to one key, and `LoadAll`'s
`v.notes[note.Slug] = note` is last-write-wins with no collision check and no
log. One of each pair is permanently unreachable.

### What didn't work
The audit needs a full vault load (~30 s) and I was asked to timebox, so I did
not go on to check whether any *specific* colliding note is one a user would
actually look for. That limitation is recorded in the design doc's Open Questions
rather than papered over.

### What I learned
`slugify` is a hash function with no collision handling, feeding a map write with
no collision check. That is the general statement of which the trailing-slash bug
and the empty-slug bug are both instances.

### What was tricky to build
Counting "markdown on disk" the same way `LoadAll` does without reusing its
filters — the audit must skip dot-directories (or it counts `.git` and `.trash`,
inflating the difference by thousands) while deliberately *not* applying
`IsExcluded`, since exclusions are what it is trying to measure. I mirrored
`LoadAll`'s dot-directory skip and its `.md` suffix test exactly, and left every
other filter out.

### What warrants a second pair of eyes
The collision policy in fix F4/Phase 4. A deterministic disambiguating suffix
changes URLs for whichever note currently loses, so it needs a redirect story.

### What should be done in the future
Refuse to store the empty slug; warn on collision; add the §12.1 table test so
the algebra cannot drift silently.

### Code review instructions
Run `scripts/04-slug-algebra` first — it is the fastest way to build intuition
for what `slugify` does. Then `05-vault-slug-audit -limit 20` against any vault.

### Technical details
```
COLLISION: "Designing RAG Abstractions" and "Designing-RAG-Abstractions" both -> "designing-rag-abstractions"
COLLISION: "Cats & Dogs" and "Cats   Dogs" both -> "cats-dogs"
COLLISION: "Cats & Dogs" and "Cats+Dogs"   both -> "cats-dogs"
COLLISION: "Cats & Dogs" and "Cats!Dogs"   both -> "cats-dogs"
```

## Step 9: Write up the ticket and commit

I created ticket PV-SLUG-020 with `docmgr`, adding three vocabulary topics that
did not exist (`slug`, `routing`, `frontmatter`) alongside the existing `parser`,
`vault`, `api`, `ignore`, `retro-obsidian-publish`, and `obsidian-vault`. The
design document is written for someone who has never seen this codebase: a
mermaid and an ASCII trace of URL → note, the measured slug table, all four
exclusion mechanisms with the real vault's configuration, the root cause with its
proof, pseudocode for current and proposed behaviour, a cited API reference, six
fix options with trade-offs, a five-phase plan, concrete regression-test rows,
and an explicit list of what I could not verify.

No application source was modified. The brief called for an analysis ticket, and
two other agents are editing `internal/` and `pkg/` concurrently.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Package everything as a docmgr ticket with a design
doc, this diary, tasks, related files, changelog entries, and focused commits.

**Inferred user intent:** A durable, reviewable artifact rather than a chat
transcript.

### What I did
- `docmgr vocab add` × 3; `docmgr ticket create-ticket --ticket PV-SLUG-020`.
- `docmgr doc add` for the design doc and this diary.
- Filled in `index.md` Overview and Summary; added five tasks in implementation
  order; related the files that matter; added changelog entries.
- Staged only `ttmp/2026/08/09/PV-SLUG-020--…` and `ttmp/vocabulary.yaml`.

### Why
Two sibling tickets (PV-MATHJAX-018, PV-MEMORY-019) are being written
concurrently in adjacent directories, and `ttmp/vocabulary.yaml` is shared. A
`git add -A` would have swept up their work.

### What worked
Scoped staging. `git status --short` in `publish-vault` shows another agent's
`internal/parser/math.go`, `math_test.go`, modified `parser.go`, and the whole
`PV-MEMORY-019--…` tree — none of which is in my commits.

### What didn't work
Recorded in the commit section of this ticket's changelog if the pre-commit hook
needed bypassing for other agents' in-flight files.

### What I learned
The `git archive HEAD` scratch-workspace pattern from Step 3 is worth keeping as
a habit for any multi-agent checkout: it costs seconds and it guarantees you are
measuring committed behaviour.

### What was tricky to build
Keeping the design doc honest about its own limits. Three separate results were
weaker than they first appeared — the `000` curl row (a harness artifact, not a
server response), the local page-route 404 (`web bundle not found`, not the bug),
and scenario D in the conflation repro (mislabelled). Each is flagged in place
rather than dropped, because a reader who cannot tell which rows are load-bearing
cannot trust any of them.

### What warrants a second pair of eyes
The root-cause statement in design doc §6. It asserts that slug generation is
*not* at fault, which contradicts the brief's prime suspect. The evidence is
Step 3's probe output and the production 200s; if a reviewer disagrees, that is
the paragraph to argue with.

### What should be done in the future
Implement Phases 1–5 from design doc §11, starting with F1 (SSR tagged result)
and F4 (log every drop).

### Code review instructions
Read design doc §6 (root cause) first, then §2 (the trace diagram) for context,
then §10–§12 for the plan. The scripts are ordered `01`…`05` and each one's
header comment states what it proves and how to run it.

### Technical details
Ticket: `publish-vault/ttmp/2026/08/09/PV-SLUG-020--note-not-found-for-nested-transcripts-slug-diagnose-slug-resolution-and-vault-exclusion`.
Scripts run from the scratch workspace described in Step 3; Go 1.26.5, Node
v22.22.1.

---

## Step 10: Implement the fix (F1 + F4 + F2)

Implemented the three fixes the design doc recommended, in its recommended
order. F1 is the actual fix for the reported symptom: the SSR sidecar now
distinguishes "this note does not exist" from "I could not ask", so a restarting
backend can no longer be reported — and cached — as a missing note. F4 makes the
next instance of this take a `grep` instead of a debugger. F2 removes the
permanent trailing-slash class of 404 that had nothing to do with the incident
but was found on the way.

The work confirmed the investigation's central claim in the most direct way
available: a regression test that drives the *real* `web/server.mjs` against a
stub backend rigged to fail in each distinct way. Before the change all four
failure modes produced `404 Note not found`; after it they produce 404, 503, 502
and 502 respectively.

### Prompt Context

**User prompt (verbatim):** "ok, implement slug fix"

**Assistant interpretation:** Implement the fix designed in this ticket, rather
than continuing to analyse it.

**Inferred user intent:** The investigation is finished and trusted; turn its
recommendation into working code.

**Commit (code):** `878e372` — "fix(PV-SLUG-020): stop reporting an unreachable backend as a missing note"

### What I did

- **F1** — replaced `fetchAPI`'s `null` with `fetchAPIResult` returning
  `{kind: ok|not_found|unreachable|server_error|bad_body}`, and branched the
  note and widget-page paths on it. Added `sendUpstreamFailure` (503 with
  `Retry-After` for unreachable, 502 otherwise) and `Cache-Control: no-store` on
  every error response. Also guarded `/api/config`, whose 5xx used to render a
  200 page titled `undefined` (row D of the original repro).
- **F4** — added `ExclusionReason` and a `v.excluded` map recorded during
  `LoadAll`, with a per-reason tally logged once and parse errors, slug
  collisions and degenerate slugs logged individually.
  `ExclusionReasonFor(relPath)` walks up parent directories so a question about
  `drafts/Foo.md` is answered by the rule that excluded `drafts/`.
- **F2** — added `normalizedIdx` (lowercase, trimmed and collapsed slashes)
  built beside `notes`, `CanonicalSlug`, and a 308 redirect in `api.getNote` on
  an exact miss. Wired `buildNormalizedIndex` into `ReloadNote`/`RemoveNote` so
  the watcher path stays consistent.
- Refused the empty slug (the half of Phase 4 that does not change live URLs).
- Wrote `scripts/smoke-ssr-upstream-failures.mjs` and the section 12 regression
  tests.

### Why

The doc's recommendation was explicit — F1 + F4 first, then F2, and never F3.
F1 is the root cause; F4 is what makes the class of bug cheap next time; F2
closes a second, permanent defect found during the investigation.

### What worked

- The stub-backend smoke test. Making the four failure modes distinguishable is
  the entire fix, so a test that produces all four against the real server is
  the right shape, and it matches the repo's existing `smoke-ssr-hydration.mjs`
  idiom rather than inventing a new one.
- `socket.destroy()` on one endpoint to simulate an unreachable backend while
  `/api/config` stays healthy. That isolates the note path; killing the whole
  stub tests a different (also real) scenario, so the suite does both.
- The ticket's own `02-http-slug-matrix.sh` as an acceptance check: the
  trailing-slash row moved from 404 to `308 -> 200` with no edits to the script.
- The slug-algebra table test reproduced all 12 measured rows from section 3.1
  on the first run, including `Привет мир -> ""`, which is reassuring about the
  investigation's numbers.

### What didn't work

**Pruned directories recorded nothing.** First run of the exclusion test:

```
--- FAIL: TestExcludedNotesRecordTheirReason
    vault_test.go:1115: ExclusionReasonFor("drafts/Hid.md") recorded nothing, want "vault-ignore"
```

`ShouldPruneDir` makes the walk `SkipDir` an excluded directory, so files
beneath it are never visited and could never be recorded. Fixed by recording the
*directory* at the prune point and having `ExclusionReasonFor` walk up ancestors
— which is also the better answer for an operator, since "drafts/ is excluded"
explains the rule rather than the instance.

**The doubled-slash row expected the wrong status.**

```
api_test.go:233: GET /api/notes/transcripts/2026//designing-rag/the_algebra = 301, want 308
```

gorilla/mux cleans `//` and issues its own 301 before the handler runs, so that
shape never reaches `CanonicalSlug`. The test now pins 301 with a comment, so a
future router change that stops cleaning is caught instead of silently 404ing.

**The first `buildNormalizedIndex` was map-order dependent.** When two slugs
shared a normalized key, which one won depended on Go's randomized map
iteration. Rewrote it to bucket slugs by key first and then apply an explicit
rule: a canonical slug owns its key; otherwise the key is dropped.

### What I learned

- `null` as a failure value is not a small wart. Four distinct upstream states
  collapsed into one, and the *loudest* of them (a hard 404 that CDNs cache)
  became the reported answer for all four. The fix is mostly about restoring
  information the code already had and threw away.
- `Cache-Control: no-store` matters as much as the status code here. A 404
  emitted during a restart outlives the restart if anything caches it.
- Recording the exclusion *reason* costs one map and makes an entire class of
  support question self-service. It should have been there from the start.
- Ambiguity via case is actually unreachable from real vault paths, because
  `slugify` already lowercases — so two files differing only in case collide on
  the *slug*, not on the normalized key. The ambiguity guard in
  `buildNormalizedIndex` is therefore defensive rather than load-bearing, which
  is worth knowing before someone "simplifies" it away.

### What was tricky to build

Making the redirect provably loop-free. A 308 to a slug that itself normalizes
differently would bounce forever, and that failure only shows up under a client
that follows redirects. Three things together rule it out: `normalizeSlug` is
idempotent (`TestNormalizeSlugIsIdempotent`), `CanonicalSlug` returns
`ok=false` when the input is already canonical so a slug can never redirect to
itself, and the API route-shape test *follows* every `Location` it receives and
asserts the follow-up is 200 rather than another redirect.

The other subtlety was keeping `normalizedIdx` consistent on the watcher path.
`ReloadNote` and `RemoveNote` mutate `v.notes` directly under their own lock, so
a stale normalized index would outlive an edit and redirect to a note that no
longer exists. Both now rebuild it alongside the wiki-link and backlink indexes.

### What warrants a second pair of eyes

- The `/api/config` guard is a behaviour change beyond the literal scope of F1:
  a page that used to render (badly) now returns 502/503. I think that is right
  — a page with no vault name and no title is not a successful response — but it
  is the change most likely to surprise.
- `buildNormalizedIndex` runs on every `ReloadNote`, i.e. once per file change,
  and is O(notes). For the 1712-note production vault that is cheap next to the
  `rebuildHTML` already happening there, but it is another full scan on a path
  that PV-MEMORY-019 shows is already expensive.
- 308 versus 301. 308 preserves method and body; nothing here is non-GET today,
  so 301 would also work and is better cached by old clients. 308 is the
  stricter choice.

### What should be done in the future

- Phase 4's remaining half: colliding slugs still resolve last-write-wins, now
  with a warning. A deterministic suffix would fix it but changes live URLs, so
  it needs a decision rather than an implementation.
- Phase 5: `/api/notes/_diagnose` and the reasoned 404 body, with the
  public/private disclosure split.
- The `excluded` map is now populated but only read by tests; the diagnose
  endpoint is what makes it useful in production.

### Code review instructions

- Start at `web/server.mjs` — `fetchAPIResult`, `sendUpstreamFailure`, and the
  note branch. That is the actual fix; everything else is supporting work.
- Then `pkg/vault/vault.go`: `LoadAll`'s `drop` closure,
  `buildNormalizedIndex`, `normalizeSlug`, `CanonicalSlug`.
- Then `pkg/api/api.go` `getNote`.
- Validate:

  ```bash
  go test ./publish-vault/... -count=1
  golangci-lint run ./pkg/vault/... ./pkg/api/... ./internal/parser/...
  cd publish-vault && node scripts/smoke-ssr-upstream-failures.mjs --skip-build
  ```

  and, against a running server with the fixture note:

  ```bash
  bash ttmp/.../scripts/02-http-slug-matrix.sh http://127.0.0.1:18420
  ```

### Technical details

Measured before to after, same fixture and same scripts:

| Case | Before | After |
|---|---|---|
| note exists, API healthy | 200 | 200 |
| note genuinely absent | 404 `Note not found` | 404 `Note not found` |
| backend hangs up | 404 `Note not found` | **503** `Backend unavailable` |
| backend 5xx | 200 title `undefined` | **502** `Backend error` |
| backend returns bad JSON | 404 `Note not found` | **502** `Backend error` |
| backend fully down | 404 `Note not found` | **503** `Backend unavailable` |
| `/api/notes/<slug>/` | 404 | **308** then 200 |

Load-time diagnostics now emitted for a vault with one ignored dir, one
`publish: false` note and one non-Latin filename:

```
warning: note excluded path="Привет.md" reason=degenerate-slug (filename has no URL-safe characters)
vault load: 2 notes published, excluded vault-ignore=1 publish-false=1 degenerate-slug=1
```
