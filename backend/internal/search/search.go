// Package search provides full-text search over vault notes using bleve.
package search

import (
	"os"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	bq "github.com/blevesearch/bleve/v2/search/query"

	"retro-obsidian-publish/backend/internal/vault"
)

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
	for _, note := range v.AllNotes() {
		if err := si.Index(note); err != nil {
			return nil, err
		}
	}
	return si, nil
}

// NewPersistent creates a persistent bleve index at indexPath.
func NewPersistent(v *vault.Vault, indexPath string) (*Index, error) {
	var idx bleve.Index
	var err error

	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		idx, err = bleve.New(indexPath, buildMapping())
	} else {
		idx, err = bleve.Open(indexPath)
	}
	if err != nil {
		return nil, err
	}

	si := &Index{idx: idx}
	// Re-index all notes
	for _, note := range v.AllNotes() {
		if err := si.Index(note); err != nil {
			return nil, err
		}
	}
	return si, nil
}

// Index adds or updates a note in the search index.
func (si *Index) Index(note *vault.Note) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Flatten tags to space-separated string for indexing
	tags := ""
	for i, t := range note.Tags {
		if i > 0 {
			tags += " "
		}
		tags += t
	}
	doc := noteDoc{
		Title:   note.Title,
		Body:    stripHTML(note.HTML),
		Tags:    tags,
		Excerpt: note.Excerpt,
	}
	return si.idx.Index(note.Slug, doc)
}

// Delete removes a note from the search index.
func (si *Index) Delete(slug string) error {
	si.mu.Lock()
	defer si.mu.Unlock()
	return si.idx.Delete(slug)
}

// Search performs a full-text query and returns ranked results.
// Uses fuzzy matching for partial words and prefix matching for short queries.
func (si *Index) Search(query string, limit int) ([]SearchResult, error) {
	si.mu.Lock()
	defer si.mu.Unlock()

	if limit <= 0 {
		limit = 20
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

// stripHTML removes HTML tags for plain-text indexing.
func stripHTML(s string) string {
	inTag := false
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			inTag = true
			continue
		}
		if s[i] == '>' {
			inTag = false
			out = append(out, ' ')
			continue
		}
		if !inTag {
			out = append(out, s[i])
		}
	}
	return string(out)
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
