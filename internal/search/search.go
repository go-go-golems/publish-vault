// Package search provides full-text search over vault notes using bleve.
package search

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	bq "github.com/blevesearch/bleve/v2/search/query"

	"retro-obsidian-publish/internal/vault"
)

// ErrClosed is returned when callers use an index after Close.
var ErrClosed = errors.New("search index is closed")

// SearchResult represents a single search hit.
type SearchResult struct {
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Tags    []string `json:"tags"`
	Score   float64  `json:"score"`
}

// Index wraps a bleve index for vault notes.
type Index struct {
	mu  sync.Mutex
	idx bleve.Index
}

// noteDoc is the document shape stored in bleve.
type noteDoc struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Tags    string `json:"tags"`
	Excerpt string `json:"excerpt"`
}

// New creates an in-memory bleve index and indexes all vault notes.
func New(v *vault.Vault) (*Index, error) {
	idx, err := bleve.NewMemOnly(buildMapping())
	if err != nil {
		return nil, err
	}
	si := &Index{idx: idx}
	if err := v.ForEachSearchDocument(func(doc vault.SearchDocument) error {
		return si.Index(doc)
	}); err != nil {
		_ = si.Close()
		return nil, err
	}
	return si, nil
}

// NewPersistent creates a fresh persistent bleve index at indexPath and indexes
// all current vault notes. Any existing directory at indexPath is removed first
// so full reloads cannot retain stale documents for deleted notes.
func NewPersistent(v *vault.Vault, indexPath string) (*Index, error) {
	if err := os.RemoveAll(indexPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, err
	}
	idx, err := bleve.New(indexPath, buildMapping())
	if err != nil {
		return nil, err
	}

	si := &Index{idx: idx}
	if err := v.ForEachSearchDocument(func(doc vault.SearchDocument) error {
		return si.Index(doc)
	}); err != nil {
		_ = si.Close()
		return nil, err
	}
	return si, nil
}

// OpenPersistent opens an existing persistent bleve index at indexPath.
func OpenPersistent(indexPath string) (*Index, error) {
	idx, err := bleve.Open(indexPath)
	if err != nil {
		return nil, err
	}
	return &Index{idx: idx}, nil
}

// Index adds or updates a note document in the search index.
func (si *Index) Index(doc vault.SearchDocument) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	if si.idx == nil {
		return ErrClosed
	}

	// Flatten tags to space-separated string for indexing
	tags := ""
	for i, t := range doc.Tags {
		if i > 0 {
			tags += " "
		}
		tags += t
	}
	bleveDoc := noteDoc{
		Title:   doc.Title,
		Body:    doc.Body,
		Tags:    tags,
		Excerpt: doc.Excerpt,
	}
	return si.idx.Index(doc.Slug, bleveDoc)
}

// Delete removes a note from the search index.
func (si *Index) Delete(slug string) error {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.idx == nil {
		return ErrClosed
	}
	return si.idx.Delete(slug)
}

// Close releases resources held by the underlying bleve index. Persistent
// indexes must be closed so file descriptors and locks are not leaked across
// reloads.
func (si *Index) Close() error {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.idx == nil {
		return nil
	}
	err := si.idx.Close()
	si.idx = nil
	return err
}

// Search performs a full-text query and returns ranked results.
// Uses fuzzy matching for partial words and prefix matching for short queries.
//
// Tag-specific search:
//   - Queries starting with "#" perform a field-scoped search on the tags field only.
//     Example: "#philosophy" matches notes tagged with philosophy.
//   - Queries starting with "tag:" are treated as an alias for "#".
//     Example: "tag:philosophy" is equivalent to "#philosophy".
func (si *Index) Search(query string, limit int) ([]SearchResult, error) {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.idx == nil {
		return nil, ErrClosed
	}

	if limit <= 0 {
		limit = 20
	}

	// Check for tag-specific search prefixes (# or tag:)
	if tagQuery, ok := extractTagQuery(query); ok {
		return si.searchByTag(tagQuery, limit)
	}

	// Tokenize the query into words
	words := tokenizeQuery(query)
	if len(words) == 0 {
		return []SearchResult{}, nil
	}

	var bleveQuery bq.Query

	if len(words) == 1 && len(words[0]) <= 3 {
		// Short single word: use prefix wildcard (e.g., "goj" -> "goj*")
		bleveQuery = bleve.NewPrefixQuery(words[0])
	} else {
		// Multi-word or longer single word: use fuzzy match queries
		// MatchQuery with Fuzziness handles partial words automatically
		var disjuncts []bq.Query
		for _, w := range words {
			mq := bleve.NewMatchQuery(w)
			mq.SetFuzziness(1)
			disjuncts = append(disjuncts, mq)
		}
		if len(disjuncts) == 1 {
			bleveQuery = disjuncts[0]
		} else {
			// All words must match (AND)
			bleveQuery = bleve.NewConjunctionQuery(disjuncts...)
		}
	}

	req := bleve.NewSearchRequestOptions(bleveQuery, limit, 0, false)
	req.Fields = []string{"title", "excerpt", "tags"}
	req.Highlight = bleve.NewHighlight()

	result, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}

	var hits []SearchResult
	for _, hit := range result.Hits {
		sr := SearchResult{
			Slug:  hit.ID,
			Score: hit.Score,
		}
		if t, ok := hit.Fields["title"]; ok {
			sr.Title = asString(t)
		}
		if e, ok := hit.Fields["excerpt"]; ok {
			sr.Excerpt = asString(e)
		}
		if tg, ok := hit.Fields["tags"]; ok {
			sr.Tags = splitTags(asString(tg))
		}
		hits = append(hits, sr)
	}
	return hits, nil
}

// extractTagQuery checks if the query starts with a tag prefix (# or tag:)
// and returns the tag name without the prefix. Returns ("", false) if no prefix.
func extractTagQuery(query string) (string, bool) {
	q := strings.TrimSpace(query)
	if strings.HasPrefix(q, "#") {
		tag := strings.TrimSpace(strings.TrimPrefix(q, "#"))
		if tag != "" {
			return strings.ToLower(tag), true
		}
	}
	if strings.HasPrefix(strings.ToLower(q), "tag:") {
		tag := strings.TrimSpace(q[4:])
		if tag != "" {
			return strings.ToLower(tag), true
		}
	}
	return "", false
}

// searchByTag performs a field-scoped search on the tags field only.
func (si *Index) searchByTag(tagQuery string, limit int) ([]SearchResult, error) {
	// Use prefix query for short tag names, match query for longer ones
	var bleveQuery bq.Query

	if len(tagQuery) <= 3 {
		// Short tag: prefix match (e.g., "phi" matches "philosophy")
		pq := bleve.NewPrefixQuery(tagQuery)
		pq.SetField("tags")
		bleveQuery = pq
	} else {
		// Longer tag: fuzzy match on tags field
		mq := bleve.NewMatchQuery(tagQuery)
		mq.SetField("tags")
		mq.SetFuzziness(1)
		bleveQuery = mq
	}

	req := bleve.NewSearchRequestOptions(bleveQuery, limit, 0, false)
	req.Fields = []string{"title", "excerpt", "tags"}
	req.Highlight = bleve.NewHighlight()

	result, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}

	var hits []SearchResult
	for _, hit := range result.Hits {
		sr := SearchResult{
			Slug:  hit.ID,
			Score: hit.Score,
		}
		if t, ok := hit.Fields["title"]; ok {
			sr.Title = asString(t)
		}
		if e, ok := hit.Fields["excerpt"]; ok {
			sr.Excerpt = asString(e)
		}
		if tg, ok := hit.Fields["tags"]; ok {
			sr.Tags = splitTags(asString(tg))
		}
		hits = append(hits, sr)
	}
	if hits == nil {
		hits = []SearchResult{}
	}
	return hits, nil
}

// tokenizeQuery splits a search query into lowercase words.
func tokenizeQuery(q string) []string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return nil
	}
	var tokens []string
	for _, w := range strings.Fields(q) {
		if w != "" {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

// buildMapping creates the bleve index mapping with English analyzer.
func buildMapping() mapping.IndexMapping {
	im := bleve.NewIndexMapping()

	dm := bleve.NewDocumentMapping()

	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = standard.Name
	titleField.Store = true
	dm.AddFieldMappingsAt("title", titleField)

	bodyField := bleve.NewTextFieldMapping()
	bodyField.Analyzer = standard.Name
	bodyField.Store = false
	dm.AddFieldMappingsAt("body", bodyField)

	tagsField := bleve.NewTextFieldMapping()
	tagsField.Analyzer = standard.Name
	tagsField.Store = true
	dm.AddFieldMappingsAt("tags", tagsField)

	excerptField := bleve.NewTextFieldMapping()
	excerptField.Store = true
	dm.AddFieldMappingsAt("excerpt", excerptField)

	im.AddDocumentMapping("note", dm)
	im.DefaultMapping = dm
	return im
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	var tags []string
	for _, t := range splitBySpace(s) {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func splitBySpace(s string) []string {
	var parts []string
	start := -1
	for i, c := range s {
		if c == ' ' || c == '\t' {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}
