// Package vault manages loading and indexing an Obsidian vault from the filesystem.
package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"retro-obsidian-publish/backend/internal/parser"
)

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

// Vault holds all notes and provides lookup methods.
type Vault struct {
	mu    sync.RWMutex
	notes map[string]*Note // keyed by slug
	root  string           // absolute path to vault directory
}

// New creates a Vault and loads all notes from rootDir.
func New(rootDir string) (*Vault, error) {
	v := &Vault{
		notes: make(map[string]*Note),
		root:  rootDir,
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
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
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

	v.buildBacklinks()
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

// buildBacklinks populates the Backlinks field for every note.
func (v *Vault) buildBacklinks() {
	// Reset to an empty slice, not nil, so JSON clients always receive [] instead
	// of null and can safely treat backlinks as an array.
	for _, n := range v.notes {
		n.Backlinks = []string{}
	}
	for slug, note := range v.notes {
		for _, wl := range note.WikiLinks {
			target := wl.Target
			// Try exact slug match, then title-based match
			if _, ok := v.notes[target]; ok {
				v.notes[target].Backlinks = appendUnique(v.notes[target].Backlinks, slug)
				continue
			}
			// Try matching by title slug
			for _, candidate := range v.notes {
				if parser.Slugify(candidate.Title) == target || candidate.Slug == target {
					candidate.Backlinks = appendUnique(candidate.Backlinks, slug)
					break
				}
			}
		}
	}
}

// ReloadNote re-parses a single file, updates the vault index, and returns the
// updated note so callers can refresh secondary indexes.
func (v *Vault) ReloadNote(absPath string) (*Note, error) {
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
	v.buildBacklinks()
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
	v.buildBacklinks()
	v.mu.Unlock()
	return slug
}

// GetNote returns a note by slug.
func (v *Vault) GetNote(slug string) (*Note, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	n, ok := v.notes[slug]
	return n, ok
}

// AllNotes returns a snapshot of all notes.
func (v *Vault) AllNotes() []*Note {
	v.mu.RLock()
	defer v.mu.RUnlock()
	notes := make([]*Note, 0, len(v.notes))
	for _, n := range v.notes {
		notes = append(notes, n)
	}
	return notes
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
