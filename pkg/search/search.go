// Package search provides full-text search over vault notes using bleve.
package search

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	bq "github.com/blevesearch/bleve/v2/search/query"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

// ErrClosed is returned when callers use an index after Close.
var ErrClosed = errors.New("search index is closed")

// SearchResult represents a single search hit.
type SearchResult struct {
	Slug    string            `json:"slug"`
	Title   string            `json:"title"`
	Excerpt string            `json:"excerpt"`
	Tags    []string          `json:"tags"`
	Path    string            `json:"path"`
	Score   float64           `json:"score"`
	Date    *SearchResultDate `json:"date,omitempty"`
}

// Index wraps a bleve index for vault notes.
type Index struct {
	mu  sync.Mutex
	idx bleve.Index
}

// IndexProgress reports completed documents and the content-free byte count of
// fields submitted to Bleve. TotalDocuments is fixed from the source snapshot.
type IndexProgress struct {
	ProcessedDocuments uint64
	TotalDocuments     uint64
	IndexedBytes       uint64
}

// Options configures search-index construction and bounded progress
// observation. BatchDocuments and BatchBytes must either both be zero (the
// legacy one-document update path) or both be positive. A document larger than
// BatchBytes is committed alone because documents are indivisible.
type Options struct {
	ObserveIndexed func(IndexProgress)
	BatchDocuments uint64
	BatchBytes     uint64
}

func (o Options) validate() error {
	if (o.BatchDocuments == 0) != (o.BatchBytes == 0) {
		return errors.New("search batch documents and bytes must both be zero or both be positive")
	}
	return nil
}

// noteDoc is the document shape stored in bleve.
type noteDoc struct {
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	Tags          string     `json:"tags"`
	TagsKw        []string   `json:"tags_kw"`
	Excerpt       string     `json:"excerpt"`
	Path          string     `json:"path"`
	PathKw        string     `json:"path_kw"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	DisplayAt     *time.Time `json:"display_at,omitempty"`
	DateKind      string     `json:"date_kind"`
	DatePrecision string     `json:"date_precision"`
}

// New creates an in-memory bleve index and indexes all vault notes.
func New(v *vault.Vault) (*Index, error) {
	return NewWithOptions(v, Options{})
}

// NewWithOptions creates an in-memory index with bounded progress callbacks.
func NewWithOptions(v *vault.Vault, options Options) (*Index, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}
	idx, err := bleve.NewMemOnly(buildMapping())
	if err != nil {
		return nil, err
	}
	si := &Index{idx: idx}
	if err := indexVault(si, v, options); err != nil {
		_ = si.Close()
		return nil, err
	}
	return si, nil
}

// NewPersistent creates a fresh persistent bleve index at indexPath and indexes
// all current vault notes. Any existing directory at indexPath is removed first
// so full reloads cannot retain stale documents for deleted notes.
func NewPersistent(v *vault.Vault, indexPath string) (*Index, error) {
	return NewPersistentWithOptions(v, indexPath, Options{})
}

// NewPersistentWithOptions creates a persistent index with bounded progress callbacks.
func NewPersistentWithOptions(v *vault.Vault, indexPath string, options Options) (*Index, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}
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
	if err := indexVault(si, v, options); err != nil {
		_ = si.Close()
		return nil, err
	}
	return si, nil
}

func indexVault(index *Index, v *vault.Vault, options Options) error {
	// #nosec G115 -- Count is map len: nonnegative and int fits uint64 on supported architectures.
	progress := IndexProgress{TotalDocuments: uint64(v.Count())}
	observeIndexProgress(options.ObserveIndexed, progress)
	if options.BatchDocuments > 0 {
		return indexVaultBatched(index, v, options, progress)
	}
	return v.ForEachSearchDocument(func(doc vault.SearchDocument) error {
		if err := index.Index(doc); err != nil {
			return err
		}
		progress.ProcessedDocuments++
		progress.IndexedBytes += searchDocumentBytes(doc)
		observeIndexProgress(options.ObserveIndexed, progress)
		return nil
	})
}

func indexVaultBatched(index *Index, v *vault.Vault, options Options, progress IndexProgress) error {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.idx == nil {
		return ErrClosed
	}

	batch := index.idx.NewBatch()
	var pendingDocuments, pendingBytes uint64
	flush := func() error {
		if pendingDocuments == 0 {
			return nil
		}
		if err := index.idx.Batch(batch); err != nil {
			return err
		}
		progress.ProcessedDocuments += pendingDocuments
		progress.IndexedBytes += pendingBytes
		observeIndexProgress(options.ObserveIndexed, progress)
		batch = index.idx.NewBatch()
		pendingDocuments, pendingBytes = 0, 0
		return nil
	}

	err := v.ForEachSearchDocument(func(doc vault.SearchDocument) error {
		docBytes := searchDocumentBytes(doc)
		if pendingDocuments > 0 && (pendingDocuments >= options.BatchDocuments || pendingBytes+docBytes > options.BatchBytes) {
			if err := flush(); err != nil {
				return err
			}
		}
		if err := batch.Index(doc.Slug, toNoteDoc(doc)); err != nil {
			return err
		}
		pendingDocuments++
		pendingBytes += docBytes
		if pendingDocuments >= options.BatchDocuments || pendingBytes >= options.BatchBytes {
			return flush()
		}
		return nil
	})
	if err != nil {
		return err
	}
	return flush()
}

func observeIndexProgress(observer func(IndexProgress), progress IndexProgress) {
	if observer != nil {
		observer(progress)
	}
}

func searchDocumentBytes(doc vault.SearchDocument) uint64 {
	bytes := uint64(len(doc.Slug) + len(doc.Title) + len(doc.Body) + len(doc.Excerpt) + len(doc.Path) + len(doc.DateKind) + len(doc.DatePrecision))
	for _, tag := range doc.Tags {
		bytes += uint64(len(tag))
	}
	// Count the keyword-array copy used for exact tag filtering.
	for _, tag := range doc.Tags {
		bytes += uint64(len(tag))
	}
	bytes += uint64(len(doc.Path)) // path_kw lowercased copy
	return bytes
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

	return si.idx.Index(doc.Slug, toNoteDoc(doc))
}

func toNoteDoc(doc vault.SearchDocument) noteDoc {
	tagsKw := make([]string, len(doc.Tags))
	for i, t := range doc.Tags {
		tagsKw[i] = strings.ToLower(t)
	}
	pathKw := strings.ToLower(doc.Path)
	pathKw = strings.TrimPrefix(pathKw, "./")
	pathKw = strings.TrimPrefix(pathKw, "/")
	return noteDoc{
		Title:         doc.Title,
		Body:          doc.Body,
		Tags:          strings.Join(doc.Tags, " "),
		TagsKw:        tagsKw,
		Excerpt:       doc.Excerpt,
		Path:          doc.Path,
		PathKw:        pathKw,
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
		DisplayAt:     doc.DisplayAt,
		DateKind:      doc.DateKind,
		DatePrecision: doc.DatePrecision,
	}
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

	words := tokenizeQuery(query)
	if len(words) == 0 {
		return []SearchResult{}, nil
	}

	bleveQuery := textQueryClause(words)

	req := bleve.NewSearchRequestOptions(bleveQuery, limit, 0, false)
	req.Fields = storedFields
	req.Highlight = bleve.NewHighlight()

	result, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}

	return extractResults(result), nil
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
	bleveQuery := legacyTagQuery(tagQuery)

	req := bleve.NewSearchRequestOptions(bleveQuery, limit, 0, false)
	req.Fields = storedFields
	req.Highlight = bleve.NewHighlight()

	result, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}

	return extractResults(result), nil
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

// buildMapping creates the bleve index mapping with a no-stopword analyzer.
//
// The text fields (title, body, tags, excerpt) are analyzed with a custom
// analyzer ("nostop") that is the unicode tokenizer + the lowercase token
// filter, intentionally WITHOUT the English stop-word filter that bleve's
// built-in "standard" analyzer applies. The stop filter drops common English
// words (what, this, that, with, from, have, your, they, them, ...) from the
// index at indexing time, so they can never be matched by any query. For a
// personal note vault every word can be a legitimate, intentional query, and
// the index-size cost of keeping stopwords is negligible at this scale, so the
// stop filter is the wrong default here. Removing it also aligns the Go bleve
// search path with the static-vault client-side matcher, which is a plain
// substring match and has no stopword logic.
//
// No stemmer is applied: exact token semantics are preserved (code
// identifiers, filenames); fuzziness on MatchQuery covers typos without stem
// surprises.
func buildMapping() mapping.IndexMapping {
	im := bleve.NewIndexMapping()

	dm := bleve.NewDocumentMapping()

	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = nostopAnalyzerName
	titleField.Store = true
	dm.AddFieldMappingsAt("title", titleField)

	bodyField := bleve.NewTextFieldMapping()
	bodyField.Analyzer = nostopAnalyzerName
	bodyField.Store = false
	dm.AddFieldMappingsAt("body", bodyField)

	tagsField := bleve.NewTextFieldMapping()
	tagsField.Analyzer = nostopAnalyzerName
	tagsField.Store = true
	dm.AddFieldMappingsAt("tags", tagsField)

	tagsKwField := bleve.NewKeywordFieldMapping()
	tagsKwField.Store = false
	dm.AddFieldMappingsAt("tags_kw", tagsKwField)

	excerptField := bleve.NewTextFieldMapping()
	excerptField.Analyzer = nostopAnalyzerName
	excerptField.Store = true
	dm.AddFieldMappingsAt("excerpt", excerptField)

	pathField := bleve.NewKeywordFieldMapping()
	pathField.Store = true
	dm.AddFieldMappingsAt("path", pathField)

	pathKwField := bleve.NewKeywordFieldMapping()
	pathKwField.Store = false
	dm.AddFieldMappingsAt("path_kw", pathKwField)

	createdAt := bleve.NewDateTimeFieldMapping()
	createdAt.Store = true
	dm.AddFieldMappingsAt("created_at", createdAt)

	updatedAt := bleve.NewDateTimeFieldMapping()
	updatedAt.Store = true
	dm.AddFieldMappingsAt("updated_at", updatedAt)

	displayAt := bleve.NewDateTimeFieldMapping()
	displayAt.Store = true
	dm.AddFieldMappingsAt("display_at", displayAt)

	dateKind := bleve.NewKeywordFieldMapping()
	dateKind.Store = true
	dm.AddFieldMappingsAt("date_kind", dateKind)

	datePrecision := bleve.NewKeywordFieldMapping()
	datePrecision.Store = true
	dm.AddFieldMappingsAt("date_precision", datePrecision)

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

// storedFields is the set of stored fields requested for result reconstruction.
var storedFields = []string{"title", "excerpt", "tags", "path", "date_kind", "date_precision", "display_at"}

// textQueryClause builds the free-text query: a prefix query for a single short
// word, otherwise a conjunction of fuzziness-one match queries. It searches all
// text fields (no SetField) to preserve existing behavior.
func textQueryClause(words []string) bq.Query {
	if len(words) == 1 && len(words[0]) <= 3 {
		return bleve.NewPrefixQuery(words[0])
	}
	var disjuncts []bq.Query
	for _, w := range words {
		mq := bleve.NewMatchQuery(w)
		mq.SetFuzziness(1)
		disjuncts = append(disjuncts, mq)
	}
	if len(disjuncts) == 1 {
		return disjuncts[0]
	}
	return bleve.NewConjunctionQuery(disjuncts...)
}

// legacyTagQuery builds the analyzed #tag/tag: discovery query over the tags
// field: prefix for short queries, fuzziness-one match for longer ones. This is
// the canonical legacy inclusion contract that static mode must reproduce.
func legacyTagQuery(tagQuery string) bq.Query {
	if len(tagQuery) <= 3 {
		pq := bleve.NewPrefixQuery(tagQuery)
		pq.SetField("tags")
		return pq
	}
	mq := bleve.NewMatchQuery(tagQuery)
	mq.SetField("tags")
	mq.SetFuzziness(1)
	return mq
}

// extractResults converts bleve hits into SearchResults, reconstructing the
// display date from the stored display_at instant and the date_kind/precision
// keyword fields without a second vault lookup.
func extractResults(result *bleve.SearchResult) []SearchResult {
	hits := make([]SearchResult, 0, len(result.Hits))
	for _, hit := range result.Hits {
		sr := SearchResult{
			Slug:  hit.ID,
			Score: hit.Score,
		}
		if v, ok := hit.Fields["title"]; ok {
			sr.Title = asString(v)
		}
		if v, ok := hit.Fields["excerpt"]; ok {
			sr.Excerpt = asString(v)
		}
		if v, ok := hit.Fields["tags"]; ok {
			sr.Tags = splitTags(asString(v))
		}
		if v, ok := hit.Fields["path"]; ok {
			sr.Path = asString(v)
		}
		if d := reconstructDate(hit.Fields); d != nil {
			sr.Date = d
		}
		hits = append(hits, sr)
	}
	return hits
}

// reconstructDate rebuilds the display date projection from stored fields. It
// returns nil when the note has no authored date.
func reconstructDate(fields map[string]interface{}) *SearchResultDate {
	kind := fieldString(fields, "date_kind")
	if kind == "" {
		return nil
	}
	precision := fieldString(fields, "date_precision")
	raw, ok := fields["display_at"]
	if !ok {
		return &SearchResultDate{Kind: kind, Precision: precision}
	}
	t, err := parseStoredTime(raw)
	if err != nil {
		return &SearchResultDate{Kind: kind, Precision: precision}
	}
	return &SearchResultDate{Value: formatAPIValue(t, precision), Kind: kind, Precision: precision}
}

// formatAPIValue renders the display instant as the API projection: the UTC
// calendar date for date precision, or a UTC RFC3339 instant at second
// precision for timestamp precision.
func formatAPIValue(t time.Time, precision string) string {
	if precision == "timestamp" {
		return t.UTC().Format(time.RFC3339)
	}
	return t.UTC().Format("2006-01-02")
}

// parseStoredTime parses a stored datetime field, which Bleve returns as an
// RFC3339 string or a time.Time depending on the version.
func parseStoredTime(v interface{}) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val, nil
	case string:
		return time.Parse(time.RFC3339, val)
	}
	return time.Time{}, fmt.Errorf("unsupported stored time type %T", v)
}

func fieldString(fields map[string]interface{}, key string) string {
	if v, ok := fields[key]; ok {
		return asString(v)
	}
	return ""
}

// buildSearchQuery composes one query tree from independent clauses: text or
// legacy tag discovery, exact tag filters, path prefixes, and a date range.
// Filter-only requests use MatchAll plus structured clauses.
func (si *Index) buildSearchQuery(req SearchRequest) bq.Query {
	var clauses []bq.Query

	if tagQuery, ok := extractTagQuery(req.Query); ok {
		clauses = append(clauses, legacyTagQuery(tagQuery))
	} else if req.Query != "" {
		if q := textQueryClause(tokenizeQuery(req.Query)); q != nil {
			clauses = append(clauses, q)
		}
	}

	if len(req.Tags) > 0 {
		tagQueries := make([]bq.Query, 0, len(req.Tags))
		for _, tag := range req.Tags {
			tq := bleve.NewTermQuery(tag)
			tq.SetField("tags_kw")
			tagQueries = append(tagQueries, tq)
		}
		if req.TagMode == TagModeAny {
			clauses = append(clauses, bleve.NewDisjunctionQuery(tagQueries...))
		} else {
			clauses = append(clauses, bleve.NewConjunctionQuery(tagQueries...))
		}
	}

	if len(req.PathPrefixes) > 0 {
		pathQueries := make([]bq.Query, 0, len(req.PathPrefixes))
		for _, p := range req.PathPrefixes {
			pq := bleve.NewPrefixQuery(p)
			pq.SetField("path_kw")
			pathQueries = append(pathQueries, pq)
		}
		clauses = append(clauses, bleve.NewDisjunctionQuery(pathQueries...))
	}

	if req.DateFrom != nil || req.DateTo != nil {
		if rq := dateRangeQuery(req); rq != nil {
			clauses = append(clauses, rq)
		}
	}

	if len(clauses) == 0 {
		return bleve.NewMatchAllQuery()
	}
	if len(clauses) == 1 {
		return clauses[0]
	}
	return bleve.NewConjunctionQuery(clauses...)
}

// dateRangeQuery builds an inclusive calendar-day range over the selected date
// field. The start is midnight UTC of date_from (inclusive); the end is midnight
// UTC of the day after date_to (exclusive).
func dateRangeQuery(req SearchRequest) bq.Query {
	field := dateFieldName(req.DateField)
	// The range is half-open on the upper bound: start is midnight UTC of
	// date_from (inclusive), end is midnight UTC of the day after date_to
	// (exclusive). Bleve treats a zero time as an open bound regardless of the
	// inclusive flag, so the flags are only meaningful when the bound is set.
	start := time.Time{}
	end := time.Time{}
	if req.DateFrom != nil {
		start = req.DateFrom.StartUTC()
	}
	if req.DateTo != nil {
		end = req.DateTo.NextDayStartUTC()
	}
	inclStart := true
	inclEnd := false
	rq := bleve.NewDateRangeInclusiveQuery(start, end, &inclStart, &inclEnd)
	rq.SetField(field)
	return rq
}

func dateFieldName(field DateField) string {
	switch field {
	case DateFieldCreated:
		return "created_at"
	case DateFieldUpdated:
		return "updated_at"
	case DateFieldDisplay:
		return "display_at"
	default:
		return "display_at"
	}
}

// sortFields returns the deterministic sort order for a request. Relevance sorts
// by descending score then id; newest/oldest sort by display_at then id.
func sortFields(sort SearchSort) []string {
	switch sort {
	case SearchSortNewest:
		return []string{"-display_at", "_id"}
	case SearchSortOldest:
		return []string{"display_at", "_id"}
	case SearchSortRelevance:
		return []string{"-_score", "_id"}
	default:
		return []string{"-_score", "_id"}
	}
}

// SearchAdvanced runs a typed advanced-search request against the index and
// returns a paginated envelope. The request must be normalized
// (NormalizeSearchRequest) before calling.
func (si *Index) SearchAdvanced(req SearchRequest) (SearchResponse, error) {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.idx == nil {
		return SearchResponse{}, ErrClosed
	}

	if !req.Effective() {
		return SearchResponse{Results: []SearchResult{}, Total: 0, Limit: req.Limit, Offset: req.Offset, Sort: req.Sort}, nil
	}

	bleveQuery := si.buildSearchQuery(req)
	searchReq := bleve.NewSearchRequestOptions(bleveQuery, req.Limit, req.Offset, false)
	searchReq.Fields = storedFields
	searchReq.SortBy(sortFields(req.Sort))

	result, err := si.idx.Search(searchReq)
	if err != nil {
		return SearchResponse{}, err
	}

	return SearchResponse{
		Results: extractResults(result),
		Total:   result.Total,
		Limit:   req.Limit,
		Offset:  req.Offset,
		Sort:    req.Sort,
	}, nil
}
