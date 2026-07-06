// Package vault manages loading and indexing an Obsidian vault from the filesystem.
package vault

import (
	"errors"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"retro-obsidian-publish/internal/ignore"
	"retro-obsidian-publish/internal/parser"
)

// ErrIgnored is returned by ReloadNote when the target path is excluded by a
// .vault-ignore file. Callers (the file watcher) treat it as a no-op rather
// than an error.
var ErrIgnored = errors.New("vault: path is excluded by .vault-ignore")

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
}

// WikiLinkRef is the JSON-serialisable form of parser.WikiLink.
type WikiLinkRef struct {
	Target  string `json:"target"`
	Alias   string `json:"alias,omitempty"`
	IsEmbed bool   `json:"isEmbed,omitempty"`
	Heading string `json:"heading,omitempty"`
}

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

// Vault holds all notes and provides lookup methods.
type Vault struct {
	mu            sync.RWMutex
	notes         map[string]*Note  // keyed by slug
	wikiLinkIndex map[string]string // short slug -> full vault slug (e.g., "tribal/foo" -> "research/kb/tribal/foo")
	root          string            // absolute path to vault directory
	ignore        *ignore.Ignore    // compiled .vault-ignore; nil/empty means exclude nothing
}

// New creates a Vault and loads all notes from rootDir. If <rootDir>/.vault-ignore
// exists it is read and used to exclude directories and files from the index, the
// file tree, search, backlinks, and the raw-source endpoint. A missing ignore
// file is harmless; a malformed one is logged and treated as "ignore nothing" so
// publishing is not blocked by a bad ignore file.
func New(rootDir string) (*Vault, error) {
	ig, err := ignore.Load(rootDir)
	if err != nil {
		log.Printf("vault: warning reading %s: %v; ignoring no paths", ignore.IgnoreFile, err)
		ig = &ignore.Ignore{}
	}
	v := &Vault{
		notes:         make(map[string]*Note),
		wikiLinkIndex: make(map[string]string),
		root:          rootDir,
		ignore:        ig,
	}
	if err := v.LoadAll(); err != nil {
		return nil, err
	}
	return v, nil
}

// LoadAll scans the vault directory and parses every .md file.
func (v *Vault) LoadAll() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.notes = make(map[string]*Note)

	err := filepath.Walk(v.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			// Skip hidden dirs
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			// Prune ignored directories only when no negation patterns exist;
			// otherwise descend so a "!" can re-include a file beneath them.
			if v.ShouldPruneDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		if v.isIgnored(path, false) {
			return nil
		}
		note, err := v.loadNote(path, info)
		if err != nil {
			return nil // skip unparseable notes
		}
		v.notes[note.Slug] = note
		return nil
	})
	if err != nil {
		return err
	}

	v.buildWikiLinkIndex()
	v.buildBacklinks()
	v.rebuildHTML()
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
		title = strings.TrimSuffix(info.Name(), ".md")
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
		WikiLinks:   wikiRefs,
		ModTime:     info.ModTime(),
	}, nil
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
		filename := strings.TrimSuffix(parts[len(parts)-1], ".md")
		suffixes := []string{parser.Slugify(filename)}

		// Build progressive suffixes from the end of the path
		for i := len(parts) - 2; i >= 0; i-- {
			shortPath := strings.Join(parts[i:], "/")
			shortPath = strings.TrimSuffix(shortPath, ".md")
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

// ResolveWikiLink maps a wiki link target (as written in the note) to the
// actual vault slug. Returns ("", false) if no match is found.
func (v *Vault) ResolveWikiLink(target string) (string, bool) {
	slug := parser.Slugify(target)
	if resolved, ok := v.wikiLinkIndex[slug]; ok {
		return resolved, true
	}
	return "", false
}

// rebuildHTML re-renders all note HTML with wiki links resolved to correct slugs
// and display text replaced with target note titles.
// This must be called after buildWikiLinkIndex so links point to actual notes.
func (v *Vault) rebuildHTML() {
	for _, note := range v.notes {
		note.HTML = parser.ReplaceWikiLinksString(note.HTML, func(target string) string {
			if resolved, ok := v.wikiLinkIndex[target]; ok {
				return resolved
			}
			return ""
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
	}
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
// by .vault-ignore, ReloadNote returns ErrIgnored and leaves the index untouched;
// callers (the file watcher) treat this as a no-op.
func (v *Vault) ReloadNote(absPath string) (*Note, error) {
	if v.isIgnored(absPath, false) {
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
	v.mu.Lock()
	v.notes[note.Slug] = note
	v.buildWikiLinkIndex()
	v.buildBacklinks()
	v.rebuildHTML()
	v.mu.Unlock()
	return note, nil
}

// RemoveNote removes a note from the vault index and returns the removed slug so
// callers can refresh secondary indexes.
func (v *Vault) RemoveNote(absPath string) string {
	relPath, _ := filepath.Rel(v.root, absPath)
	slug := pathToSlug(relPath)
	v.mu.Lock()
	delete(v.notes, slug)
	v.buildWikiLinkIndex()
	v.buildBacklinks()
	v.mu.Unlock()
	return slug
}

// IsIgnored reports whether absPath (an absolute, OS-native path inside the
// vault root) is excluded by the current .vault-ignore. isDir indicates whether
// the path is a directory, which affects directory-only patterns. v.ignore and
// v.root are set once at construction and never mutated, so IsIgnored is safe to
// call concurrently without a lock (mirroring Root).
func (v *Vault) IsIgnored(absPath string, isDir bool) bool {
	return v.isIgnored(absPath, isDir)
}

// ShouldPruneDir reports whether a filesystem walk should skip absPath entirely.
// It returns true only when the directory is ignored AND the ignore file has no
// negation patterns. When negations exist, a later "!" could re-include a file
// beneath an otherwise-ignored directory, so pruning the directory would
// silently drop that file; in that case the walk must descend and match each
// file individually. This keeps the loader and the matcher consistent.
func (v *Vault) ShouldPruneDir(absPath string) bool {
	if v.ignore == nil || v.ignore.Empty() {
		return false
	}
	if v.ignore.HasNegations() {
		return false
	}
	return v.isIgnored(absPath, true)
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
				Name:     strings.TrimSuffix(part, ".md"),
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
// Paths excluded by .vault-ignore return os.ErrNotExist so the raw-source
// endpoint cannot be used to bypass an ignore.
func (v *Vault) ReadRaw(relPath string) ([]byte, error) {
	cleaned := cleanVaultRelativePath(relPath)
	if cleaned == "" || !strings.EqualFold(filepath.Ext(cleaned), ".md") {
		return nil, os.ErrNotExist
	}
	if v.isIgnored(filepath.Join(v.root, filepath.FromSlash(cleaned)), false) {
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
	s = strings.TrimSuffix(s, ".md")
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
