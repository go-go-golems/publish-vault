package search

import (
	"testing"
)

func TestParseDateOnly(t *testing.T) {
	tests := []struct {
		value string
		ok    bool
		str   string
	}{
		{"2024-01-15", true, "2024-01-15"},
		{"2024-1-5", false, ""},
		{"2024-13-40", false, ""},
		{"01/15/2024", false, ""},
		{"2024-01-15T00:00:00Z", false, ""},
		{"", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			d, err := ParseDateOnly(tt.value)
			if tt.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if d.String() != tt.str {
					t.Fatalf("String() = %q, want %q", d.String(), tt.str)
				}
			} else if err == nil {
				t.Fatalf("expected error for %q, got %v", tt.value, d)
			}
		})
	}
}

func TestDateOnlyRangeBounds(t *testing.T) {
	from, _ := ParseDateOnly("2024-01-16")
	to, _ := ParseDateOnly("2024-01-16")
	if !from.StartUTC().Equal(to.StartUTC()) {
		t.Fatal("same day should have equal start")
	}
	// Single-day range is half-open [start, next-day start).
	if !to.NextDayStartUTC().After(to.StartUTC()) {
		t.Fatal("next day start must be after start")
	}
}

func TestNormalizeDefaultsAndEffective(t *testing.T) {
	req, errs := NormalizeSearchRequest(SearchRequest{})
	if len(errs) != 0 {
		t.Fatalf("empty request should not error, got %v", errs)
	}
	if req.Limit != defaultLimit {
		t.Fatalf("Limit = %d, want %d", req.Limit, defaultLimit)
	}
	if req.Sort != SearchSortNewest {
		t.Fatalf("filter-only default Sort = %q, want newest", req.Sort)
	}
	if req.Effective() {
		t.Fatal("empty request should not be effective")
	}

	qreq, _ := NormalizeSearchRequest(SearchRequest{Query: "memory"})
	if qreq.Sort != SearchSortRelevance {
		t.Fatalf("query default Sort = %q, want relevance", qreq.Sort)
	}
	if !qreq.Effective() {
		t.Fatal("query request should be effective")
	}
}

func TestNormalizeTags(t *testing.T) {
	req, errs := NormalizeSearchRequest(SearchRequest{Tags: []string{"Go", "#rust", "go", "  space  "}})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := []string{"go", "rust", "space"}
	if len(req.Tags) != 3 {
		t.Fatalf("Tags = %v, want %v", req.Tags, want)
	}
	for i := range want {
		if req.Tags[i] != want[i] {
			t.Fatalf("Tags = %v, want %v", req.Tags, want)
		}
	}
}

func TestNormalizeTagsErrors(t *testing.T) {
	tooMany := make([]string, maxTags+1)
	for i := range tooMany {
		tooMany[i] = "t"
	}
	_, errs := NormalizeSearchRequest(SearchRequest{Tags: tooMany})
	if !hasFieldError(errs, "tag", "too_many_tags") {
		t.Fatalf("expected too_many_tags, got %v", errs)
	}
	_, errs = NormalizeSearchRequest(SearchRequest{Tags: []string{"   "}})
	if !hasFieldError(errs, "tag", "tag_invalid") {
		t.Fatalf("expected tag_invalid for empty, got %v", errs)
	}
}

func TestNormalizePathPrefixes(t *testing.T) {
	req, errs := NormalizeSearchRequest(SearchRequest{PathPrefixes: []string{"/Research/KB/", "projects//2026", "./Notes"}})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := []string{"research/kb/", "projects/2026/", "notes/"}
	if len(req.PathPrefixes) != 3 {
		t.Fatalf("Paths = %v, want %v", req.PathPrefixes, want)
	}
	for i := range want {
		if req.PathPrefixes[i] != want[i] {
			t.Fatalf("Path[%d] = %q, want %q", i, req.PathPrefixes[i], want[i])
		}
	}
}

func TestNormalizePathRejectsTraversal(t *testing.T) {
	_, errs := NormalizeSearchRequest(SearchRequest{PathPrefixes: []string{"Research/../Private"}})
	if !hasFieldError(errs, "path", "path_invalid") {
		t.Fatalf("expected path_invalid for traversal, got %v", errs)
	}
}

func TestNormalizeDateRange(t *testing.T) {
	from, _ := ParseDateOnly("2024-01-01")
	toLate, _ := ParseDateOnly("2023-12-01")
	_, errs := NormalizeSearchRequest(SearchRequest{DateFrom: &from, DateTo: &toLate})
	if !hasFieldError(errs, "date_to", "before_date_from") {
		t.Fatalf("expected before_date_from, got %v", errs)
	}

	toOk, _ := ParseDateOnly("2024-12-31")
	req, errs := NormalizeSearchRequest(SearchRequest{DateFrom: &from, DateTo: &toOk})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if req.DateField != DateFieldDisplay {
		t.Fatalf("DateField = %q, want display", req.DateField)
	}

	// date_field without a range is ineffective input.
	_, errs = NormalizeSearchRequest(SearchRequest{DateField: DateFieldCreated})
	if !hasFieldError(errs, "date_field", "date_field_without_range") {
		t.Fatalf("expected date_field_without_range, got %v", errs)
	}
}

func TestNormalizeLimitOffsetSort(t *testing.T) {
	_, errs := NormalizeSearchRequest(SearchRequest{Limit: -1})
	if !hasFieldError(errs, "limit", "limit_out_of_range") {
		t.Fatalf("expected limit_out_of_range for -1, got %v", errs)
	}
	_, errs = NormalizeSearchRequest(SearchRequest{Limit: 101})
	if !hasFieldError(errs, "limit", "limit_out_of_range") {
		t.Fatalf("expected limit_out_of_range for 101, got %v", errs)
	}
	_, errs = NormalizeSearchRequest(SearchRequest{Offset: -1})
	if !hasFieldError(errs, "offset", "offset_out_of_range") {
		t.Fatalf("expected offset_out_of_range, got %v", errs)
	}
	_, errs = NormalizeSearchRequest(SearchRequest{Sort: "bogus"})
	if !hasFieldError(errs, "sort", "sort_invalid") {
		t.Fatalf("expected sort_invalid, got %v", errs)
	}
}

func hasFieldError(errs []FieldError, field, code string) bool {
	for _, e := range errs {
		if e.Field == field && e.Code == code {
			return true
		}
	}
	return false
}
