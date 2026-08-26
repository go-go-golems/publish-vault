package search

import (
	"testing"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

// buildAdvancedVault builds a small vault covering tags, paths, dates, and a
// missing-date note so the advanced-search contract can be exercised.
func buildAdvancedVault(t *testing.T) *vault.Vault {
	t.Helper()
	root := t.TempDir()
	writeTestNote(t, root, "Research/KB/Alpha.md",
		"---\ntitle: Alpha\ntags: [go, performance]\ncreated: 2024-01-15\nupdated: 2024-02-20T09:00:00Z\n---\n# Alpha\n\ncommon content\n")
	writeTestNote(t, root, "Research/KB/Beta.md",
		"---\ntitle: Beta\ntags: [go]\ncreated: 2024-03-01\n---\n# Beta\n\ncommon content\n")
	writeTestNote(t, root, "Projects/Gamma.md",
		"---\ntitle: Gamma\ntags: [rust]\ncreated: 2024-01-10\n---\n# Gamma\n\ncommon content\n")
	writeTestNote(t, root, "Notes/Plain.md", "# Plain\n\ncommon content\n")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	return v
}

func slugs(resp SearchResponse) []string {
	out := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		out = append(out, r.Slug)
	}
	return out
}

func mustIndex(t *testing.T, v *vault.Vault) *Index {
	t.Helper()
	si, err := New(v)
	if err != nil {
		t.Fatalf("search.New: %v", err)
	}
	t.Cleanup(func() { _ = si.Close() })
	return si
}

func datePtr(t *testing.T, s string) *DateOnly {
	t.Helper()
	d, err := ParseDateOnly(s)
	if err != nil {
		t.Fatalf("ParseDateOnly(%s): %v", s, err)
	}
	return &d
}

func TestSearchAdvancedExactTagAll(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Tags: []string{"go", "performance"}, TagMode: TagModeAll})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 1 || len(resp.Results) != 1 || resp.Results[0].Slug != "research/kb/alpha" {
		t.Fatalf("all-tags = total=%d slugs=%v, want only alpha", resp.Total, slugs(resp))
	}
}

func TestSearchAdvancedExactTagAny(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Tags: []string{"go", "rust"}, TagMode: TagModeAny})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("any-tags total = %d slugs=%v, want 3", resp.Total, slugs(resp))
	}
}

func TestSearchAdvancedPathPrefix(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{PathPrefixes: []string{"research/kb/"}})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("path total = %d slugs=%v, want 2", resp.Total, slugs(resp))
	}
	for _, r := range resp.Results {
		if r.Path == "" {
			t.Fatalf("result %q has empty path", r.Slug)
		}
	}
}

func TestSearchAdvancedDateRangeDisplay(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{DateFrom: datePtr(t, "2024-02-01"), DateTo: datePtr(t, "2024-03-01")})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	got := slugs(resp)
	// Alpha (display 2024-02-20) and Beta (display 2024-03-01); Gamma (01-10) excluded.
	if resp.Total != 2 {
		t.Fatalf("date range total = %d slugs=%v, want 2", resp.Total, got)
	}
}

func TestSearchAdvancedSortNewest(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Query: "content", Sort: SearchSortNewest})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	want := []string{"research/kb/beta", "research/kb/alpha", "projects/gamma", "notes/plain"}
	if !equalStrings(slugs(resp), want) {
		t.Fatalf("newest order = %v, want %v", slugs(resp), want)
	}
}

func TestSearchAdvancedSortOldestMissingLast(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Query: "content", Sort: SearchSortOldest})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	// Ascending display_at with the undated note last.
	want := []string{"projects/gamma", "research/kb/alpha", "research/kb/beta", "notes/plain"}
	if !equalStrings(slugs(resp), want) {
		t.Fatalf("oldest order = %v, want %v (missing date must sort last)", slugs(resp), want)
	}
}

func TestSearchAdvancedPagination(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Query: "content", Sort: SearchSortNewest, Limit: 2, Offset: 0})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 4 || len(resp.Results) != 2 {
		t.Fatalf("page1 total=%d len=%d, want total=4 len=2", resp.Total, len(resp.Results))
	}
	req2, _ := NormalizeSearchRequest(SearchRequest{Query: "content", Sort: SearchSortNewest, Limit: 2, Offset: 2})
	resp2, err := si.SearchAdvanced(req2)
	if err != nil {
		t.Fatalf("SearchAdvanced page2: %v", err)
	}
	if len(resp2.Results) != 2 {
		t.Fatalf("page2 len=%d, want 2", len(resp2.Results))
	}
}

func TestSearchAdvancedResultDateReconstructed(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Tags: []string{"performance"}})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 1 || resp.Results[0].Date == nil {
		t.Fatalf("expected reconstructed date, got total=%d results=%v", resp.Total, resp.Results)
	}
	d := resp.Results[0].Date
	if d.Kind != "updated" || d.Precision != "timestamp" || d.Value != "2024-02-20T09:00:00Z" {
		t.Fatalf("date = %+v, want updated/timestamp/2024-02-20T09:00:00Z", d)
	}
}

func TestSearchAdvancedLegacyTagDiscovery(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{Query: "#go", Sort: SearchSortRelevance})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	// #go is a short prefix query over the analyzed tags field: matches Alpha and Beta.
	if resp.Total != 2 {
		t.Fatalf("#go total = %d slugs=%v, want 2", resp.Total, slugs(resp))
	}
}

func TestSearchAdvancedNotEffectiveReturnsEmpty(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	req, _ := NormalizeSearchRequest(SearchRequest{})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 0 || len(resp.Results) != 0 {
		t.Fatalf("ineffective request total=%d results=%v, want empty", resp.Total, resp.Results)
	}
}

func TestSearchAdvancedCompoundQuery(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	// go tag AND research/kb/ prefix AND display range -> only Alpha.
	req, _ := NormalizeSearchRequest(SearchRequest{
		Tags:         []string{"go"},
		PathPrefixes: []string{"research/kb/"},
		DateFrom:     datePtr(t, "2024-02-01"),
		DateTo:       datePtr(t, "2024-02-28"),
	})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	if resp.Total != 1 || resp.Results[0].Slug != "research/kb/alpha" {
		t.Fatalf("compound total=%d slugs=%v, want only alpha", resp.Total, slugs(resp))
	}
}

func TestSearchAdvancedDateRangeExclusiveUpperBound(t *testing.T) {
	si := mustIndex(t, buildAdvancedVault(t))
	// date_to=2024-02-20 must include a note whose display instant is exactly
	// 2024-02-20 (Alpha's updated timestamp) but must NOT include a note at
	// 2024-02-21T00:00:00Z. Alpha's updated is 2024-02-20T09:00:00Z, which is
	// inside the [2024-02-20, 2024-02-21) window.
	to, _ := ParseDateOnly("2024-02-20")
	req, _ := NormalizeSearchRequest(SearchRequest{DateTo: &to, Sort: SearchSortNewest})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
	for _, r := range resp.Results {
		if r.Slug == "research/kb/beta" {
			t.Fatalf("date_to=2024-02-20 must exclude beta (2024-03-01); got %v", slugs(resp))
		}
	}
	found := false
	for _, r := range resp.Results {
		if r.Slug == "research/kb/alpha" {
			found = true
		}
	}
	if !found {
		t.Fatalf("date_to=2024-02-20 must include alpha (2024-02-20T09:00:00Z); got %v", slugs(resp))
	}
}
