package search

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// TagMode selects whether exact tag filters combine with AND (all) or OR (any).
type TagMode string

const (
	TagModeAll TagMode = "all"
	TagModeAny TagMode = "any"
)

// DateField selects which authored date a range filter applies to.
type DateField string

const (
	DateFieldDisplay DateField = "display"
	DateFieldCreated DateField = "created"
	DateFieldUpdated DateField = "updated"
)

// SearchSort selects result ordering.
type SearchSort string

const (
	SearchSortRelevance SearchSort = "relevance"
	SearchSortNewest    SearchSort = "newest"
	SearchSortOldest    SearchSort = "oldest"
)

// DateOnly is a calendar date without a timezone. It is the public date-filter
// representation; range queries convert it to UTC instants so server, static
// build, SSR, and clients stay deterministic.
type DateOnly struct {
	Year  int
	Month int
	Day   int
}

// ParseDateOnly parses a strict YYYY-MM-DD literal and validates the calendar
// date. It rejects partial dates (2024-1-5) and any value with a time component.
func ParseDateOnly(value string) (DateOnly, error) {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return DateOnly{}, fmt.Errorf("date must be YYYY-MM-DD")
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9} {
		if value[i] < '0' || value[i] > '9' {
			return DateOnly{}, fmt.Errorf("date must be YYYY-MM-DD")
		}
	}
	year := atoi(value[0:4])
	month := atoi(value[5:7])
	day := atoi(value[8:10])
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// time.Date normalizes overflow (e.g. month 13); reject if it did not round-trip.
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return DateOnly{}, fmt.Errorf("invalid calendar date")
	}
	return DateOnly{Year: year, Month: month, Day: day}, nil
}

// StartUTC returns midnight UTC of the calendar date (inclusive range start).
func (d DateOnly) StartUTC() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// NextDayStartUTC returns midnight UTC of the next calendar day (exclusive range
// end), so a single-day filter [d, d] is the half-open interval [d, d+1).
func (d DateOnly) NextDayStartUTC() time.Time {
	return d.StartUTC().AddDate(0, 0, 1)
}

// String returns the canonical YYYY-MM-DD literal.
func (d DateOnly) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// FieldError is one stable, machine-readable validation error. The Message is
// suitable for users but is not the contract; Field and Code are.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SearchRequest is the typed advanced-search request shared by the Bleve
// backend and the static implementation. It is the single source of truth for
// text + structured constraints.
type SearchRequest struct {
	Query        string
	Tags         []string
	TagMode      TagMode
	PathPrefixes []string
	DateField    DateField
	DateFrom     *DateOnly
	DateTo       *DateOnly
	Sort         SearchSort
	Limit        int
	Offset       int
}

// SearchResultDate is the resolved display date projection in a search hit.
type SearchResultDate struct {
	Value     string `json:"value"`
	Kind      string `json:"kind"`
	Precision string `json:"precision"`
}

// SearchResponse is the typed envelope returned by the advanced endpoint.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   uint64         `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
	Sort    SearchSort     `json:"sort"`
}

// Bounded limits that constrain query tree size and error surfaces. They are
// API limits, not Prometheus labels.
const (
	maxQueryBytes   = 1024
	maxTags         = 20
	maxTagBytes     = 128
	maxPathPrefixes = 10
	maxPathBytes    = 512
	defaultLimit    = 30
	maxLimit        = 100
	maxOffset       = 10000
)

// NormalizeSearchRequest trims, validates, applies defaults, and canonicalizes
// an advanced-search request. It returns the normalized request and a list of
// stable field errors. A non-empty error list means the request is invalid; the
// normalized request is still returned for partial inspection.
func NormalizeSearchRequest(raw SearchRequest) (SearchRequest, []FieldError) {
	req := raw
	var errs []FieldError

	req.Query = strings.TrimSpace(req.Query)
	if utf8.RuneCountInString(req.Query) > 0 && len(req.Query) > maxQueryBytes {
		errs = append(errs, FieldError{Field: "q", Code: "query_too_long", Message: "Query is too long."})
	}

	var tagErrs []FieldError
	req.Tags, tagErrs = normalizeTags(req.Tags)
	errs = append(errs, tagErrs...)

	if req.TagMode == "" {
		req.TagMode = TagModeAll
	}
	if req.TagMode != TagModeAll && req.TagMode != TagModeAny {
		errs = append(errs, FieldError{Field: "tag_mode", Code: "tag_mode_invalid", Message: "tag_mode must be all or any."})
	}

	var pathErrs []FieldError
	req.PathPrefixes, pathErrs = normalizePathPrefixes(req.PathPrefixes)
	errs = append(errs, pathErrs...)

	hasRange := req.DateFrom != nil || req.DateTo != nil
	if req.DateField == "" {
		if hasRange {
			req.DateField = DateFieldDisplay
		}
	}
	switch req.DateField {
	case "", DateFieldDisplay, DateFieldCreated, DateFieldUpdated:
	default:
		errs = append(errs, FieldError{Field: "date_field", Code: "date_field_invalid", Message: "date_field must be display, created, or updated."})
	}
	if req.DateField != "" && !hasRange {
		errs = append(errs, FieldError{Field: "date_field", Code: "date_field_without_range", Message: "date_field requires date_from or date_to."})
	}
	if req.DateFrom != nil && req.DateTo != nil && req.DateTo.Before(*req.DateFrom) {
		errs = append(errs, FieldError{Field: "date_to", Code: "before_date_from", Message: "date_to must be on or after date_from."})
	}

	switch req.Sort {
	case SearchSortRelevance, SearchSortNewest, SearchSortOldest, "":
	default:
		errs = append(errs, FieldError{Field: "sort", Code: "sort_invalid", Message: "sort must be relevance, newest, or oldest."})
	}
	if req.Sort == "" {
		if req.Query != "" {
			req.Sort = SearchSortRelevance
		} else {
			req.Sort = SearchSortNewest
		}
	}

	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	if req.Limit < 1 || req.Limit > maxLimit {
		errs = append(errs, FieldError{Field: "limit", Code: "limit_out_of_range", Message: "limit must be between 1 and 100."})
	}
	if req.Offset < 0 || req.Offset > maxOffset {
		errs = append(errs, FieldError{Field: "offset", Code: "offset_out_of_range", Message: "offset must be between 0 and 10000."})
	}

	return req, errs
}

// Effective reports whether the request has any search criterion: non-empty
// text or at least one structured filter. Filter-only requests are valid.
func (req SearchRequest) Effective() bool {
	return req.Query != "" || len(req.Tags) > 0 || len(req.PathPrefixes) > 0 || req.DateFrom != nil || req.DateTo != nil
}

func normalizeTags(tags []string) ([]string, []FieldError) {
	var errs []FieldError
	if len(tags) > maxTags {
		errs = append(errs, FieldError{Field: "tag", Code: "too_many_tags", Message: fmt.Sprintf("At most %d tags are allowed.", maxTags)})
	}
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		t = strings.TrimPrefix(t, "#")
		t = strings.ToLower(t)
		if t == "" {
			errs = append(errs, FieldError{Field: "tag", Code: "tag_invalid", Message: "Tags must not be empty."})
			continue
		}
		if hasControlChars(t) {
			errs = append(errs, FieldError{Field: "tag", Code: "tag_invalid", Message: "Tags must not contain control characters."})
			continue
		}
		if len(t) > maxTagBytes {
			errs = append(errs, FieldError{Field: "tag", Code: "tag_too_long", Message: fmt.Sprintf("Tags must be at most %d bytes.", maxTagBytes)})
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out, errs
}

func normalizePathPrefixes(paths []string) ([]string, []FieldError) {
	var errs []FieldError
	if len(paths) > maxPathPrefixes {
		errs = append(errs, FieldError{Field: "path", Code: "too_many_paths", Message: fmt.Sprintf("At most %d path prefixes are allowed.", maxPathPrefixes)})
	}
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.ToLower(strings.TrimSpace(p))
		p = strings.TrimPrefix(p, "./")
		p = strings.TrimPrefix(p, "/")
		// Reject traversal; a path filter is a folder prefix, not a glob.
		if strings.Contains(p, "..") {
			errs = append(errs, FieldError{Field: "path", Code: "path_invalid", Message: "Path prefixes must not contain traversal segments."})
			continue
		}
		// Collapse duplicate separators and strip a trailing slash before re-adding one.
		p = collapseSlashes(p)
		p = strings.TrimSuffix(p, "/")
		if p == "" {
			errs = append(errs, FieldError{Field: "path", Code: "path_invalid", Message: "Path prefixes must not be empty."})
			continue
		}
		if len(p) > maxPathBytes {
			errs = append(errs, FieldError{Field: "path", Code: "path_too_long", Message: fmt.Sprintf("Path prefixes must be at most %d bytes.", maxPathBytes)})
			continue
		}
		// A trailing slash prevents research/go from matching research/golang-notes.
		p = p + "/"
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out, errs
}

func collapseSlashes(p string) string {
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

func hasControlChars(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// Before reports whether d is strictly before other (calendar order).
func (d DateOnly) Before(other DateOnly) bool {
	if d.Year != other.Year {
		return d.Year < other.Year
	}
	if d.Month != other.Month {
		return d.Month < other.Month
	}
	return d.Day < other.Day
}
