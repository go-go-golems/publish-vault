// Package vault manages loading and indexing an Obsidian vault from the filesystem.
package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/publish-vault/internal/ignore"
	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vaultconfig"
)

// ErrIgnored is returned by ReloadNote when the target path is excluded by a
// .vault-ignore file. Callers (the file watcher) treat it as a no-op rather
// than an error.
var ErrIgnored = errors.New("vault: path is excluded by .vault-ignore")

// ErrUnpublished is returned by ReloadNote when the reloaded note carries
// publish: false. It wraps ErrIgnored so callers that only care about "this
// path is not part of the published vault" keep working, while callers that
// must clean up secondary indexes (the file watcher deleting the note from
// search) can distinguish a note that just became hidden from a path that was
// never indexed.
var ErrUnpublished = fmt.Errorf("%w: note frontmatter sets publish: false", ErrIgnored)

// Note represents a single Obsidian note.
type Note struct {
	Slug        string                 `json:"slug"`
	Title       string                 `json:"title"`
	Path        string                 `json:"path"` // relative path inside vault
	Frontmatter map[string]interface{} `json:"frontmatter"`
	Tags        []string               `json:"tags"`
	Excerpt     string                 `json:"excerpt"`
	HTML        string                 `json:"html"`
	WikiLinks   []WikiLinkRef          `json:"wikiLinks"`
	Backlinks   []string               `json:"backlinks"` // slugs that link to this note
	ModTime     time.Time              `json:"modTime"`
	Publish     bool                   `json:"-"` // false => excluded from publication

	// sourceHTML is the parser output before vault-level link, asset, and embed
	// resolution. rebuildHTML always renders HTML from it rather than from the
	// previously rendered HTML, so a resolution that depended on vault state
	// (e.g. an embed target hidden by publish: false) is re-evaluated instead of
	// being baked into the note forever.
	sourceHTML string
}

// WikiLinkRef is the JSON-serialisable form of parser.WikiLink.
type WikiLinkRef struct {
	Target  string `json:"target"`
	Alias   string `json:"alias,omitempty"`
	IsEmbed bool   `json:"isEmbed,omitempty"`
	Heading string `json:"heading,omitempty"`
}

// ExclusionReason names why a Markdown file in the vault did not become a
// published note.
//
// Four mechanisms could previously drop a note without leaving any trace, which
// made "why is this URL a 404?" unanswerable without attaching a debugger — the
// question that opened PV-SLUG-020. Recording the reason turns it into a log
// line and a map lookup.
type ExclusionReason string

const (
	ExcludedByIgnore    ExclusionReason = "vault-ignore"
	ExcludedByConfig    ExclusionReason = "config-blacklist"
	ExcludedByPublish   ExclusionReason = "publish-false"
	ExcludedByParse     ExclusionReason = "parse-error"
	ExcludedByEmptySlug ExclusionReason = "degenerate-slug"
	ExcludedByCollision ExclusionReason = "slug-collision"
)

// FileNode represents a node in the vault file tree.
type FileNode struct {
	Name     string      `json:"name"`
	Slug     string      `json:"slug,omitempty"`
	Path     string      `json:"path"`
	IsFolder bool        `json:"isFolder"`
	Children []*FileNode `json:"children,omitempty"`
}

// SearchDocument is the plain-text representation used by the full-text index.
// It is built from Markdown source on demand instead of from rendered HTML.
type SearchDocument struct {
	Slug    string
	Title   string
	Body    string
	Tags    []string
	Excerpt string
}

// LoadStage is a finite, content-free vault build phase suitable for traces and metrics.
type LoadStage string

const (
	LoadStageWalkParse  LoadStage = "vault_walk_parse"
	LoadStageNormalize  LoadStage = "vault_normalize"
	LoadStageWikiLinks  LoadStage = "wiki_link_index"
	LoadStageBacklinks  LoadStage = "backlinks"
	LoadStageRenderHTML LoadStage = "render_html"
)

// LoadProgress reports completed candidate notes and source bytes. TotalNotes
// and TotalBytes count publish-eligible Markdown candidates before parsing;
// parse failures and publish:false notes still count as processed work.
type LoadProgress struct {
	Stage          LoadStage
	ProcessedNotes uint64
	TotalNotes     uint64
	ProcessedBytes uint64
	TotalBytes     uint64
}

// LoadObserver receives bounded progress while LoadAll holds the vault write
// lock. It must not call back into the Vault.
type LoadObserver func(LoadProgress)

// Vault holds all notes and provides lookup methods.
type Vault struct {
	mu            sync.RWMutex
	notes         map[string]*Note // keyed by slug
	excluded      map[string]ExclusionReason
	normalizedIdx map[string]string    // normalizeSlug(slug) -> canonical slug
	wikiLinkIndex map[string]string    // short slug -> full vault slug (e.g., "tribal/foo" -> "research/kb/tribal/foo")
	assetIndex    map[string]string    // lowercased basename and vault-relative path -> vault-relative path (![[pic.png]] resolution)
	root          string               // absolute path to vault directory
	ignore        *ignore.Ignore       // compiled .vault-ignore; nil/empty means exclude nothing
	configMatcher *vaultconfig.Matcher // compiled .publish/config.yaml blacklist; nil/empty means exclude nothing
	loadObserver  LoadObserver         // optional bounded lifecycle callback
}

// Option configures a Vault.
type Option func(*Vault)

// WithLoadObserver reports bounded load-stage progress.
func WithLoadObserver(observer LoadObserver) Option {
	return func(v *Vault) { v.loadObserver = observer }
}

// WithConfig attaches a vault config (blacklist matcher) to the vault. When
// set, the loader excludes paths matched by the config blacklist in addition
// to .vault-ignore. The matcher is treated as immutable after construction
// (same lifecycle as ignore), so it is safe to read concurrently without a
// lock. A nil or compile-failed config is logged and ignored.
func WithConfig(cfg *vaultconfig.Config) Option {
	return func(v *Vault) {
		m, err := vaultconfig.NewMatcher(cfg)
		if err != nil {
			log.Printf("vault: warning compiling config blacklist: %v; ignoring no paths", err)
			return
		}
		v.configMatcher = m
	}
}

// New creates a Vault and loads all notes from rootDir. If <rootDir>/.vault-ignore
// exists it is read and used to exclude directories and files from the index, the
// file tree, search, backlinks, and the raw-source endpoint. A missing ignore
// file is harmless; a malformed one is logged and treated as "ignore nothing" so
// publishing is not blocked by a bad ignore file. Options may attach a vault
// config (WithConfig) whose blacklist is applied in addition to .vault-ignore.
func New(rootDir string, opts ...Option) (*Vault, error) {
	ig, err := ignore.Load(rootDir)
	if err != nil {
		log.Printf("vault: warning reading %s: %v; ignoring no paths", ignore.IgnoreFile, err)
		ig = &ignore.Ignore{}
	}
	v := &Vault{
		notes:         make(map[string]*Note),
		wikiLinkIndex: make(map[string]string),
		assetIndex:    make(map[string]string),
		root:          rootDir,
		ignore:        ig,
	}
	for _, opt := range opts {
		opt(v)
	}
	if err := v.LoadAll(); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Vault) observeLoad(progress LoadProgress) {
	if v.loadObserver != nil {
		v.loadObserver(progress)
	}
}

func (v *Vault) loadCandidateTotals() (uint64, uint64, error) {
	var notes, bytes uint64
	err := filepath.Walk(v.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path != v.root && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if path != v.root && v.ShouldPruneDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") || v.IsExcluded(path, false) {
			return nil
		}
		notes++
		if info.Size() > 0 {
			// #nosec G115 -- a positive int64 is exactly representable by uint64.
			bytes += uint64(info.Size())
		}
		return nil
	})
	return notes, bytes, err
}

// LoadAll scans the vault directory and parses every .md file.
func (v *Vault) LoadAll() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.notes = make(map[string]*Note)
	v.assetIndex = make(map[string]string)
	v.excluded = make(map[string]ExclusionReason)

	// Counts, not one line per file: a vault with broad ignore rules drops
	// thousands of paths and per-path logging would bury the two reasons that
	// always matter (parse errors and slug collisions), which are logged
	// individually below.
	counts := map[ExclusionReason]int{}
	collisions := 0
	totalNotes, totalBytes, err := v.loadCandidateTotals()
	if err != nil {
		return err
	}
	var processedNotes, processedBytes uint64
	v.observeLoad(LoadProgress{Stage: LoadStageWalkParse, TotalNotes: totalNotes, TotalBytes: totalBytes})
	drop := func(absPath string, reason ExclusionReason) {
		v.excluded[v.relPath(absPath)] = reason
		counts[reason]++
	}

	err = filepath.Walk(v.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			// Skip hidden dirs
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			// Prune excluded directories only when no negation patterns exist;
			// otherwise descend so a "!" can re-include a file beneath them.
			if v.ShouldPruneDir(path) {
				// Record the directory, not its contents: the walk never visits
				// them, and "drafts/ is excluded" is the answer an operator
				// asking about drafts/Foo.md actually needs.
				// ExclusionReasonFor walks up to find it.
				drop(path, v.exclusionMechanism(path, true))
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			if !v.IsExcluded(path, false) {
				v.indexAsset(path)
			}
			return nil
		}
		if v.IsExcluded(path, false) {
			drop(path, v.exclusionMechanism(path, false))
			return nil
		}
		processedNotes++
		if info.Size() > 0 {
			// #nosec G115 -- a positive int64 is exactly representable by uint64.
			processedBytes += uint64(info.Size())
		}
		defer v.observeLoad(LoadProgress{
			Stage: LoadStageWalkParse, ProcessedNotes: processedNotes, TotalNotes: totalNotes,
			ProcessedBytes: processedBytes, TotalBytes: totalBytes,
		})
		note, err := v.loadNote(path, info)
		if err != nil {
			// Always individual and always visible: an unparseable note is a
			// content bug the author needs to know about, and it is the one
			// exclusion nobody asked for.
			drop(path, ExcludedByParse)
			log.Printf("warning: note excluded path=%q reason=%s err=%v", v.relPath(path), ExcludedByParse, err)
			return nil
		}
		// A note carrying publish: false is parsed but not stored, so it is
		// absent from every consumer that reads v.notes (API, file tree,
		// search, backlinks, raw endpoint).
		if !note.Publish {
			drop(path, ExcludedByPublish)
			return nil
		}
		if note.Slug == "" {
			// slugify strips everything outside [a-z0-9-_/], so a note whose
			// filename is entirely non-Latin ("Привет.md", "日本語.md") slugs to
			// "". Storing it would put every such note on the same "" key.
			drop(path, ExcludedByEmptySlug)
			log.Printf("warning: note excluded path=%q reason=%s (filename has no URL-safe characters)", v.relPath(path), ExcludedByEmptySlug)
			return nil
		}
		// Both notes get published. filepath.Walk is lexical, so the first path
		// to claim a slug keeps it across restarts and only the later one is
		// renamed. Previously the second note silently replaced the first.
		if assigned, existing, renamed := v.assignSlug(note); renamed {
			collisions++
			log.Printf("warning: slug collision slug=%q kept=%q renamed=%q to=%q",
				note.Slug, existing, v.relPath(path), assigned)
			note.Slug = assigned
		}
		v.notes[note.Slug] = note
		return nil
	})
	if err != nil {
		return err
	}

	if len(counts) > 0 || collisions > 0 {
		log.Printf("vault load: %d notes published, excluded %s, renamed %d colliding slug(s)",
			len(v.notes), formatExclusionCounts(counts), collisions)
	}

	v.observeLoad(LoadProgress{Stage: LoadStageNormalize, TotalNotes: 1})
	v.buildNormalizedIndex()
	v.observeLoad(LoadProgress{Stage: LoadStageNormalize, ProcessedNotes: 1, TotalNotes: 1})
	v.observeLoad(LoadProgress{Stage: LoadStageWikiLinks, TotalNotes: 1})
	v.buildWikiLinkIndex()
	v.observeLoad(LoadProgress{Stage: LoadStageWikiLinks, ProcessedNotes: 1, TotalNotes: 1})
	v.observeLoad(LoadProgress{Stage: LoadStageBacklinks, TotalNotes: 1})
	v.buildBacklinks()
	v.observeLoad(LoadProgress{Stage: LoadStageBacklinks, ProcessedNotes: 1, TotalNotes: 1})
	v.observeLoad(LoadProgress{Stage: LoadStageRenderHTML, TotalNotes: uint64(len(v.notes))})
	v.rebuildHTML()
	v.observeLoad(LoadProgress{Stage: LoadStageRenderHTML, ProcessedNotes: uint64(len(v.notes)), TotalNotes: uint64(len(v.notes))})
	return nil
}

// loadNote parses a single .md file into a Note (caller must hold lock or be in init).
func (v *Vault) loadNote(absPath string, info os.FileInfo) (*Note, error) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	parsed, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(v.root, absPath)
	slug := pathToSlug(relPath)

	title := parsed.Title
	if title == "" {
		// Fall back to filename without extension
		title = parser.StripNoteExtension(info.Name())
	}

	frontmatter := parsed.Frontmatter
	if frontmatter == nil {
		frontmatter = map[string]interface{}{}
	}
	tags := parsed.Tags
	if tags == nil {
		tags = []string{}
	}

	wikiRefs := []WikiLinkRef{}
	for _, wl := range parsed.WikiLinks {
		wikiRefs = append(wikiRefs, WikiLinkRef{
			Target:  wl.Target,
			Alias:   wl.Alias,
			IsEmbed: wl.IsEmbed,
			Heading: wl.Heading,
		})
	}

	return &Note{
		Slug:        slug,
		Title:       title,
		Path:        relPath,
		Frontmatter: frontmatter,
		Tags:        tags,
		Excerpt:     parsed.Excerpt,
		HTML:        parsed.HTML,
		sourceHTML:  parsed.HTML,
		WikiLinks:   wikiRefs,
		ModTime:     info.ModTime(),
		Publish:     publishFlag(frontmatter),
	}, nil
}

// publishFlag reads the "publish" frontmatter key case-insensitively and returns
// the publication eligibility of a note. The default is true (eligible, subject
// to ignore/config exclusion); publish is opt-out only: an absent key means
// eligible, and publish: true never overrides an ignore or config exclusion.
func publishFlag(fm map[string]interface{}) bool {
	v, _ := frontmatterBool(fm, "publish", true)
	return v
}

// frontmatterBool looks up a boolean frontmatter key case-insensitively. It
// accepts YAML booleans and the strings "true"/"false" (goldmark-meta may
// surface scalars as strings depending on YAML quoting). It returns
// (value, true) when the key is present and (defaultValue, false) when absent.
func frontmatterBool(fm map[string]interface{}, key string, defaultValue bool) (bool, bool) {
	if fm == nil {
		return defaultValue, false
	}
	lowerKey := strings.ToLower(key)
	for k, v := range fm {
		if strings.ToLower(k) != lowerKey {
			continue
		}
		switch val := v.(type) {
		case bool:
			return val, true
		case string:
			s := strings.TrimSpace(strings.ToLower(val))
			switch s {
			case "true", "yes":
				return true, true
			case "false", "no":
				return false, true
			}
			return defaultValue, true
		default:
			return defaultValue, true
		}
	}
	return defaultValue, false
}

// buildWikiLinkIndex creates a lookup from short slugified wiki targets to full
// vault slugs. Obsidian wiki links like [[Tribal/foo]] reference notes by short
// path, but the vault stores notes by their full relative path (e.g.,
// Research/KB/Tribal/foo.md → slug "research/kb/tribal/foo").
// The index maps every suffix of each note's path to the note's full slug,
// so "tribal/foo" resolves to "research/kb/tribal/foo".
func (v *Vault) buildWikiLinkIndex() {
	v.wikiLinkIndex = make(map[string]string)
	for _, note := range v.notes {
		// Register the full slug
		v.wikiLinkIndex[note.Slug] = note.Slug

		// Register suffix-based short paths
		// e.g., path "Research/KB/Tribal/App.md" → register:
		//   "tribal/app", "kb/tribal/app"
		parts := strings.Split(filepath.ToSlash(note.Path), "/")
		filename := parser.StripNoteExtension(parts[len(parts)-1])
		suffixes := []string{parser.Slugify(filename)}

		// Build progressive suffixes from the end of the path
		for i := len(parts) - 2; i >= 0; i-- {
			shortPath := strings.Join(parts[i:], "/")
			shortPath = parser.StripNoteExtension(shortPath)
			suffixes = append(suffixes, parser.Slugify(shortPath))
		}

		for _, suffix := range suffixes {
			if _, exists := v.wikiLinkIndex[suffix]; !exists {
				v.wikiLinkIndex[suffix] = note.Slug
			}
		}

		// Also register by title slug
		titleSlug := parser.Slugify(note.Title)
		if titleSlug != "" {
			if _, exists := v.wikiLinkIndex[titleSlug]; !exists {
				v.wikiLinkIndex[titleSlug] = note.Slug
			}
		}
	}
}

// indexAsset registers a non-markdown vault file for ![[image.png]] embed
// resolution. Every path suffix is registered ("pic.png", "project-a/pic.png",
// "Attachments/project-a/pic.png") because Obsidian's shortest-path link
// format produces exactly the suffix that disambiguates duplicate basenames —
// mirroring what buildWikiLinkIndex does for notes. Lookups are
// case-insensitive; on collisions the lexicographically first path wins,
// keeping resolution deterministic across reloads.
func (v *Vault) indexAsset(absPath string) {
	indexAssetInto(v.assetIndex, v.root, absPath)
}

func indexAssetInto(index map[string]string, root, absPath string) {
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return
	}
	relSlash := filepath.ToSlash(rel)
	parts := strings.Split(relSlash, "/")
	for i := range parts {
		key := strings.ToLower(strings.Join(parts[i:], "/"))
		if existing, ok := index[key]; !ok || relSlash < existing {
			index[key] = relSlash
		}
	}
}

// RefreshAssetIndex rebuilds the asset index from the current filesystem
// state. The file watcher calls this when non-markdown files change so that
// ![[image.png]] embeds in subsequently (re)loaded notes resolve against
// current attachments without a full vault reload. The walk runs without the
// vault lock (ignore rules are immutable after load); the fresh index is
// swapped in atomically.
func (v *Vault) RefreshAssetIndex() {
	fresh := make(map[string]string)
	_ = filepath.Walk(v.root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if v.ShouldPruneDir(p) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		if v.IsExcluded(p, false) {
			return nil
		}
		indexAssetInto(fresh, v.root, p)
		return nil
	})
	v.mu.Lock()
	v.assetIndex = fresh
	v.mu.Unlock()
}

// ResolveAssetEmbed maps a ![[...]] embed target (as written in the note,
// e.g. "pic.png" or "Attachments/pic.png") to the vault-relative path of the
// matching file. Returns ("", false) when no asset matches.
func (v *Vault) ResolveAssetEmbed(target string) (string, bool) {
	key := strings.ToLower(filepath.ToSlash(strings.TrimSpace(target)))
	if key == "" {
		return "", false
	}
	if p, ok := v.assetIndex[key]; ok {
		return p, true
	}
	return "", false
}

// ResolveWikiLink maps a wiki link target (as written in the note) to the
// actual vault slug. Returns ("", false) if no match is found.
//
// A trailing ".md" is stripped first because the index is keyed on
// extension-less paths (see buildWikiLinkIndex). The parser already strips it
// from targets it stores, but this is public API taking a target "as written",
// so callers passing the raw [[Note.md]] form get the same answer as [[Note]].
func (v *Vault) ResolveWikiLink(target string) (string, bool) {
	slug := parser.Slugify(parser.StripNoteExtension(strings.TrimSpace(target)))
	if resolved, ok := v.wikiLinkIndex[slug]; ok {
		return resolved, true
	}
	return "", false
}

// rebuildHTML re-renders all note HTML with wiki links resolved to correct slugs
// and display text replaced with target note titles.
// This must be called after buildWikiLinkIndex so links point to actual notes.
//
// Rendering always starts from the note's parser output (sourceHTML), never
// from the previously rendered HTML: every resolution below depends on vault
// state that changes between reloads, so re-running them over already-rendered
// HTML would make the first outcome permanent. In particular an embed whose
// target was hidden or missing would keep its "Note not published" marker after
// the target became publishable, because the placeholder it was rendered from
// would be gone.
func (v *Vault) rebuildHTML() {
	// Heading indexes are built on demand and cached for the length of the
	// pass: only notes that are the target of a [[Note#Heading]] link need one,
	// which is a small minority of the vault, but a popular target is linked
	// many times. A nil entry caches "this slug is not a published note".
	headingIndexes := map[string]*parser.HeadingIndex{}
	headingIndexFor := func(slug string) *parser.HeadingIndex {
		if idx, seen := headingIndexes[slug]; seen {
			return idx
		}
		var idx *parser.HeadingIndex
		if target, ok := v.notes[slug]; ok {
			idx = parser.BuildHeadingIndex(target.sourceOrRenderedHTML())
		}
		headingIndexes[slug] = idx
		return idx
	}

	for _, note := range v.notes {
		note.HTML = parser.ReplaceWikiLinksString(note.sourceOrRenderedHTML(), func(target string) string {
			if resolved, ok := v.wikiLinkIndex[target]; ok {
				return resolved
			}
			return ""
		})
		// Must follow ReplaceWikiLinksString: the fragment is resolved against
		// the note the link ends up at, which is only known once the slug has
		// been. Consults v.notes directly because rebuildHTML runs under v.mu.
		note.HTML = parser.ResolveWikiLinkHeadings(note.HTML, func(slug, heading string) (string, bool) {
			return headingIndexFor(slug).Lookup(heading)
		})
		note.HTML = parser.ReplaceWikiLinkDisplay(note.HTML, func(slug string) string {
			if n, ok := v.notes[slug]; ok {
				return n.Title
			}
			return ""
		})
		note.HTML = parser.RewriteImageSources(note.HTML, func(src string) string {
			return v.ResolveAssetURL(note.Path, src)
		})
		note.HTML = parser.ReplaceWikiEmbedImages(note.HTML, func(target string) string {
			if p, ok := v.ResolveAssetEmbed(target); ok {
				return "/vault-assets/" + escapeAssetPath(p)
			}
			return ""
		})
		// Render a visible marker for note embeds (![[Note]]) whose target is
		// not in the index — this covers notes hidden by publish: false as well
		// as genuinely missing targets, so a hidden note does not render as an
		// empty embed. Consult v.notes directly (lock-free) because rebuildHTML
		// runs under v.mu; calling the locking GetNote here would deadlock.
		note.HTML = replaceUnresolvedNoteEmbeds(note.HTML, func(slug string) bool {
			_, ok := v.notes[slug]
			return ok
		})
	}
}

// sourceOrRenderedHTML returns the parser output a rebuild should render from,
// falling back to the current HTML for notes constructed without going through
// loadNote (test fixtures), so a rebuild never blanks such a note.
func (n *Note) sourceOrRenderedHTML() string {
	if n.sourceHTML != "" {
		return n.sourceHTML
	}
	return n.HTML
}

// unresolvedNoteEmbedRe matches the placeholder div emitted by the parser for
// a note embed (![[Note]]): <div class="wiki-embed" data-target="..." ...>.
var unresolvedNoteEmbedRe = regexp.MustCompile(`<div class="wiki-embed" data-target="([^"]*)"([^>]*)></div>`)

// replaceUnresolvedNoteEmbeds replaces note-embed placeholders whose target
// slug is not a published note with a visible broken-embed marker. The isPublished
// callback reports whether a slug is in the index; callers that already hold
// v.mu must pass a lock-free lookup to avoid deadlock.
func replaceUnresolvedNoteEmbeds(html string, isPublished func(slug string) bool) string {
	return unresolvedNoteEmbedRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := unresolvedNoteEmbedRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		target := sub[1]
		if target == "" {
			return match
		}
		if isPublished(target) {
			return match
		}
		return `<span class="wiki-embed wiki-embed-broken">⚠ Note not published: ` + target + `</span>`
	})
}

// ResolveAssetURL maps a Markdown image src to a public /vault-assets URL. Relative
// image paths are resolved against the note's directory; root-relative image
// paths are treated as vault-root-relative paths. External and already-routed
// application URLs are left unchanged.
func (v *Vault) ResolveAssetURL(notePath, src string) string {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" || shouldLeaveAssetURL(trimmed) {
		return src
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return src
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return src
	}

	assetPath := parsed.Path
	if assetPath == "" {
		return src
	}
	if strings.HasPrefix(assetPath, "/") {
		assetPath = strings.TrimPrefix(assetPath, "/")
	} else {
		base := path.Dir(filepath.ToSlash(notePath))
		if base == "." {
			base = ""
		}
		assetPath = path.Join(base, assetPath)
	}

	cleaned := cleanVaultRelativePath(assetPath)
	if cleaned == "" {
		return src
	}
	parsed.Path = "/vault-assets/" + escapeAssetPath(cleaned)
	parsed.RawPath = ""
	return parsed.String()
}

func shouldLeaveAssetURL(src string) bool {
	lower := strings.ToLower(src)
	return strings.HasPrefix(src, "#") ||
		strings.HasPrefix(src, "//") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(src, "/assets/") ||
		strings.HasPrefix(src, "/vault-assets/") ||
		strings.HasPrefix(src, "/api/") ||
		strings.HasPrefix(src, "/note/")
}

func cleanVaultRelativePath(p string) string {
	p = strings.TrimSpace(filepath.ToSlash(p))
	if p == "" || strings.HasPrefix(p, "/") {
		return ""
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func escapeAssetPath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// buildBacklinks populates the Backlinks field for every note.
func (v *Vault) buildBacklinks() {
	// Reset to an empty slice, not nil, so JSON clients always receive [] instead
	// of null and can safely treat backlinks as an array.
	for _, n := range v.notes {
		n.Backlinks = []string{}
	}
	for slug, note := range v.notes {
		for _, wl := range note.WikiLinks {
			resolved, ok := v.ResolveWikiLink(wl.Target)
			if !ok {
				continue
			}
			if target, ok := v.notes[resolved]; ok {
				target.Backlinks = appendUnique(target.Backlinks, slug)
			}
		}
	}
}

// ReloadNote re-parses a single file, updates the vault index, and returns the
// updated note so callers can refresh secondary indexes. If absPath is excluded
// by .vault-ignore or the config blacklist, ReloadNote returns ErrIgnored and
// leaves the index untouched; callers (the file watcher) treat this as a no-op.
// A note carrying publish: false is likewise not stored: if it was previously
// published it is removed from the index, and ReloadNote returns ErrUnpublished
// (which wraps ErrIgnored) so the watcher drops it from the search index too.
func (v *Vault) ReloadNote(absPath string) (*Note, error) {
	if v.IsExcluded(absPath, false) {
		return nil, ErrIgnored
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	note, err := v.loadNote(absPath, info)
	if err != nil {
		return nil, err
	}
	if !note.Publish {
		// Was published, now hidden by frontmatter: drop it from the index and
		// signal the watcher to remove it from secondary indexes.
		v.RemoveNote(absPath)
		return nil, ErrUnpublished
	}
	v.mu.Lock()
	// Drop whatever slug this path currently holds before reinserting. A note
	// whose slug was disambiguated does not live at its natural slug, so
	// inserting under a freshly computed one would overwrite the note that owns
	// it and strand the old suffixed entry.
	v.forgetPath(note.Path)
	if assigned, existing, renamed := v.assignSlug(note); renamed {
		log.Printf("warning: slug collision slug=%q kept=%q renamed=%q to=%q",
			note.Slug, existing, note.Path, assigned)
		note.Slug = assigned
	}
	v.notes[note.Slug] = note
	v.buildNormalizedIndex()
	v.buildWikiLinkIndex()
	v.buildBacklinks()
	v.rebuildHTML()
	v.mu.Unlock()
	return note, nil
}

// SlugForPath returns the vault slug an absolute path inside the vault root
// maps to, whether or not a note with that slug is currently indexed. The file
// watcher uses it to address secondary indexes (search) for a note that
// ReloadNote has already dropped from the vault index.
func (v *Vault) SlugForPath(absPath string) string {
	relPath, err := filepath.Rel(v.root, absPath)
	if err != nil {
		return ""
	}
	// Consult the index first: a note renamed to resolve a slug collision does
	// not live at its natural slug, and returning that would make the watcher
	// delete the wrong note (or nothing at all) when the file is removed.
	rel := filepath.ToSlash(relPath)
	v.mu.RLock()
	for slug, n := range v.notes {
		if n.Path == rel {
			v.mu.RUnlock()
			return slug
		}
	}
	v.mu.RUnlock()
	return pathToSlug(relPath)
}

// assignSlug decides which slug a note should occupy, given what is already in
// the index. It returns the slug to use, the path of the note that owns the
// natural slug when one is being displaced, and whether a rename happened.
//
// A note is never considered to collide with itself: reloading a file must
// return the slug it already holds, whether that is the natural one or a
// previously assigned suffix. Caller must hold v.mu.
func (v *Vault) assignSlug(note *Note) (string, string, bool) {
	existing, clash := v.notes[note.Slug]
	if !clash || existing.Path == note.Path {
		return note.Slug, "", false
	}
	assigned := disambiguateSlug(note.Slug, note.Path, func(candidate string) bool {
		other, taken := v.notes[candidate]
		return taken && other.Path != note.Path
	})
	return assigned, existing.Path, true
}

// forgetPath removes every index entry pointing at a note's path, returning the
// slug it held. A note whose slug was disambiguated does not live at its
// natural slug, so reinserting it under a freshly computed natural slug would
// both overwrite whichever note owns that slug and strand the old suffixed
// entry. Caller must hold v.mu.
func (v *Vault) forgetPath(relPath string) string {
	previous := ""
	for slug, n := range v.notes {
		if n.Path == relPath {
			previous = slug
			delete(v.notes, slug)
		}
	}
	return previous
}

// disambiguateSlug returns a deterministic alternative for a slug already taken
// by another note. The suffix is derived from the note's own vault-relative
// path, so it does not shift when unrelated notes are added or removed — a
// positional suffix like "-2" would renumber and break links.
func disambiguateSlug(natural, relPath string, taken func(string) bool) string {
	sum := sha256.Sum256([]byte(relPath))
	digest := hex.EncodeToString(sum[:])
	for n := 6; n < len(digest); n += 2 {
		if candidate := natural + "-" + digest[:n]; !taken(candidate) {
			return candidate
		}
	}
	return natural + "-" + digest
}

// RemoveNote removes a note from the vault index and returns the removed slug so
// callers can refresh secondary indexes. Remaining notes are re-rendered so an
// embed of the removed note (![[Gone]]) falls back to the broken-embed marker
// instead of keeping a placeholder that no longer resolves to anything.
func (v *Vault) RemoveNote(absPath string) string {
	slug := v.SlugForPath(absPath)
	v.mu.Lock()
	delete(v.notes, slug)
	v.buildNormalizedIndex()
	v.buildWikiLinkIndex()
	v.buildBacklinks()
	v.rebuildHTML()
	v.mu.Unlock()
	return slug
}

// IsIgnored reports whether absPath (an absolute, OS-native path inside the
// vault root) is excluded by the current .vault-ignore. isDir indicates whether
// the path is a directory, which affects directory-only patterns. v.ignore and
// v.root are set once at construction and never mutated, so IsIgnored is safe to
// call concurrently without a lock (mirroring Root).
//
// Prefer IsExcluded, which also accounts for the config blacklist. IsIgnored is
// retained because the file watcher consults it directly and is updated in a
// separate phase; it delegates to the unified IsExcluded so both ignore sources
// remain in agreement.
func (v *Vault) IsIgnored(absPath string, isDir bool) bool {
	return v.IsExcluded(absPath, isDir)
}

// IsExcluded reports whether absPath is excluded by EITHER .vault-ignore OR the
// config blacklist. It is the unified exclusion decision consulted by the loader,
// the file watcher, and the raw/asset endpoints. isDir indicates whether the
// path is a directory, which affects directory-only patterns in both matchers.
// Excluded-if-either semantics means a negation in one file cannot override
// exclusion in the other (an operator must remove the exclusion from the other
// file to re-include a path). v.ignore, v.configMatcher, and v.root are set once
// at construction and never mutated, so IsExcluded is safe to call concurrently
// without a lock.
func (v *Vault) IsExcluded(absPath string, isDir bool) bool {
	if v.isIgnored(absPath, isDir) {
		return true
	}
	if v.configMatcher == nil || v.configMatcher.Empty() {
		return false
	}
	rel, err := filepath.Rel(v.root, absPath)
	if err != nil {
		return false
	}
	return v.configMatcher.Match(filepath.ToSlash(rel), isDir)
}

// ShouldPruneDir reports whether a filesystem walk should skip absPath entirely.
// It returns true only when the directory is excluded AND no negation patterns
// exist in EITHER matcher. When negations exist, a later "!" could re-include a
// file beneath an otherwise-ignored directory, so pruning the directory would
// silently drop that file; in that case the walk must descend and match each
// file individually. This keeps the loader and the matcher consistent.
func (v *Vault) ShouldPruneDir(absPath string) bool {
	if (v.ignore == nil || v.ignore.Empty()) && (v.configMatcher == nil || v.configMatcher.Empty()) {
		return false
	}
	if v.ignore != nil && !v.ignore.Empty() && v.ignore.HasNegations() {
		return false
	}
	if v.configMatcher.HasNegations() {
		return false
	}
	return v.IsExcluded(absPath, true)
}

// isIgnored is the lock-free internal matcher used from contexts that may
// already hold v.mu (e.g. LoadAll). It is nil-safe.
func (v *Vault) isIgnored(absPath string, isDir bool) bool {
	if v.ignore == nil || v.ignore.Empty() {
		return false
	}
	return v.ignore.MatchAbs(v.root, absPath, isDir)
}

// GetNote returns a note by slug.
func (v *Vault) GetNote(slug string) (*Note, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	n, ok := v.notes[slug]
	return n, ok
}

// CanonicalSlug resolves a slug that did not match exactly to the one real
// slug it denotes, reporting whether a different canonical form exists.
//
// GetNote is an exact map lookup, so a key differing by a single byte is a hard
// 404 with no suggestion — and `slugify` preserves both a trailing "/" and a
// doubled "//", so `/note/a/b/` (a plausible copy-paste) permanently 404s a
// note that is one normalization step away. Callers 308-redirect to the
// returned slug rather than serving it, keeping one canonical URL per note.
//
// Returns ok=false when the input already is canonical, so a caller can never
// redirect a slug to itself.
func (v *Vault) CanonicalSlug(slug string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if _, exact := v.notes[slug]; exact {
		return slug, false
	}
	canonical, ok := v.normalizedIdx[normalizeSlug(slug)]
	if !ok || canonical == slug {
		return "", false
	}
	return canonical, true
}

// ExclusionReasonFor reports why a vault-relative path did not become a
// published note, if it was seen and dropped during the last load.
func (v *Vault) ExclusionReasonFor(relPath string) (ExclusionReason, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if reason, ok := v.excluded[relPath]; ok {
		return reason, ok
	}
	// A pruned directory is recorded instead of the files beneath it, which the
	// walk never visited. Walk up so a question about drafts/Foo.md is answered
	// by the rule that excluded drafts/.
	for dir := path.Dir(relPath); dir != "." && dir != "/" && dir != ""; dir = path.Dir(dir) {
		if reason, ok := v.excluded[dir]; ok {
			return reason, true
		}
	}
	return "", false
}

// buildNormalizedIndex maps each note's normalized slug to its canonical slug.
// Caller must hold v.mu.
func (v *Vault) buildNormalizedIndex() {
	byKey := make(map[string][]string, len(v.notes))
	for slug := range v.notes {
		key := normalizeSlug(slug)
		byKey[key] = append(byKey[key], slug)
	}

	idx := make(map[string]string, len(byKey))
	for key, slugs := range byKey {
		if len(slugs) == 1 {
			idx[key] = slugs[0]
			continue
		}
		// Several real notes share this normalized key. If one of them *is* the
		// canonical form it owns the key; otherwise picking between them would
		// be a guess, and 404 beats silently serving the wrong note. Resolving
		// this explicitly rather than by last-write-wins also keeps the index
		// independent of Go's randomized map iteration order.
		for _, slug := range slugs {
			if slug == key {
				idx[key] = slug
				break
			}
		}
	}
	v.normalizedIdx = idx
}

// exclusionMechanism reports which matcher excluded absPath. IsExcluded ORs the
// two, so this re-tests .vault-ignore to attribute the drop.
func (v *Vault) exclusionMechanism(absPath string, isDir bool) ExclusionReason {
	if v.isIgnored(absPath, isDir) {
		return ExcludedByIgnore
	}
	return ExcludedByConfig
}

// normalizeSlug is the equivalence class used for fallback lookup: case,
// surrounding and duplicated slashes are the differences a user can introduce
// by hand that should still find the note.
//
// It must be idempotent — normalizeSlug(normalizeSlug(x)) == normalizeSlug(x) —
// or CanonicalSlug could hand back a slug that normalizes differently and the
// redirect would loop. TestNormalizeSlugIsIdempotent pins this.
func normalizeSlug(slug string) string {
	s := strings.ToLower(strings.TrimSpace(slug))
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return strings.Trim(s, "/")
}

// relPath returns the vault-relative, slash-separated form of an absolute path.
func (v *Vault) relPath(absPath string) string {
	rel, err := filepath.Rel(v.root, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}

// formatExclusionCounts renders the per-reason tally in a stable order so the
// load line is greppable and diffable across restarts.
func formatExclusionCounts(counts map[ExclusionReason]int) string {
	order := []ExclusionReason{
		ExcludedByIgnore, ExcludedByConfig, ExcludedByPublish,
		ExcludedByParse, ExcludedByEmptySlug, ExcludedByCollision,
	}
	parts := make([]string, 0, len(order))
	for _, reason := range order {
		if n := counts[reason]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", reason, n))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}

// AllNotes returns a snapshot of all notes.
func (v *Vault) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.notes)
}

func (v *Vault) AllNotes() []*Note {
	v.mu.RLock()
	defer v.mu.RUnlock()
	notes := make([]*Note, 0, len(v.notes))
	for _, n := range v.notes {
		notes = append(notes, n)
	}
	return notes
}

func (v *Vault) SearchDocument(note *Note) (SearchDocument, error) {
	raw, err := v.ReadRaw(note.Path)
	if err != nil {
		return SearchDocument{}, err
	}
	return SearchDocument{
		Slug:    note.Slug,
		Title:   note.Title,
		Body:    parser.PlainText(raw),
		Tags:    append([]string(nil), note.Tags...),
		Excerpt: note.Excerpt,
	}, nil
}

// ForEachSearchDocument streams Markdown-derived search documents to fn without
// materializing a full-vault plaintext slice.
func (v *Vault) ForEachSearchDocument(fn func(SearchDocument) error) error {
	for _, note := range v.AllNotes() {
		doc, err := v.SearchDocument(note)
		if err != nil {
			return err
		}
		if err := fn(doc); err != nil {
			return err
		}
	}
	return nil
}

// SearchDocuments returns all Markdown-derived search documents. Prefer
// ForEachSearchDocument for indexing large vaults.
func (v *Vault) SearchDocuments() ([]SearchDocument, error) {
	docs := make([]SearchDocument, 0, v.Count())
	if err := v.ForEachSearchDocument(func(doc SearchDocument) error {
		docs = append(docs, doc)
		return nil
	}); err != nil {
		return nil, err
	}
	return docs, nil
}

// FileTree returns the hierarchical file tree of the vault.
func (v *Vault) FileTree() *FileNode {
	v.mu.RLock()
	defer v.mu.RUnlock()

	root := &FileNode{Name: "root", Path: "", IsFolder: true}
	nodeMap := map[string]*FileNode{"": root}

	// Collect all paths
	for _, note := range v.notes {
		parts := strings.Split(filepath.ToSlash(note.Path), "/")
		current := ""
		for i, part := range parts {
			parent := current
			if current == "" {
				current = part
			} else {
				current = current + "/" + part
			}
			if _, exists := nodeMap[current]; exists {
				continue
			}
			isLast := i == len(parts)-1
			node := &FileNode{
				Name:     parser.StripNoteExtension(part),
				Path:     current,
				IsFolder: !isLast,
			}
			if isLast {
				node.Slug = note.Slug
			}
			nodeMap[current] = node
			parentNode := nodeMap[parent]
			parentNode.Children = append(parentNode.Children, node)
		}
	}

	sortTree(root)

	return root
}

// sortTree recursively sorts tree nodes: folders first, then alphabetically.
func sortTree(node *FileNode) {
	if node == nil {
		return
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsFolder != b.IsFolder {
			return a.IsFolder
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for _, child := range node.Children {
		sortTree(child)
	}
}

// Root returns the vault root directory.
func (v *Vault) Root() string {
	return v.root
}

// ReadRaw reads a Markdown note source from disk on demand. The relPath must be
// a clean vault-relative Markdown path, normally taken from a Note.Path field.
// Paths excluded by .vault-ignore OR the config blacklist return os.ErrNotExist
// so the raw-source endpoint cannot be used to bypass an exclusion. (A note
// hidden by publish: false is never reachable here because it is absent from
// v.notes, so GetNote returns not-found before ReadRaw is called.)
func (v *Vault) ReadRaw(relPath string) ([]byte, error) {
	cleaned := cleanVaultRelativePath(relPath)
	if cleaned == "" || !strings.EqualFold(filepath.Ext(cleaned), ".md") {
		return nil, os.ErrNotExist
	}
	if v.IsExcluded(filepath.Join(v.root, filepath.FromSlash(cleaned)), false) {
		return nil, os.ErrNotExist
	}

	root, err := os.OpenRoot(v.root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	file, err := root.Open(filepath.FromSlash(cleaned))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() || !strings.EqualFold(filepath.Ext(info.Name()), ".md") {
		return nil, os.ErrNotExist
	}
	return io.ReadAll(file)
}

// pathToSlug converts a relative file path to a URL slug.
func pathToSlug(relPath string) string {
	s := filepath.ToSlash(relPath)
	s = parser.StripNoteExtension(s)
	return parser.Slugify(s)
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
