package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func writeAPINote(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func advancedVault(t *testing.T) (*vault.Vault, *search.Index) {
	t.Helper()
	root := t.TempDir()
	writeAPINote(t, root, "Research/Alpha.md",
		"---\ntitle: Alpha\ntags: [go, performance]\ncreated: 2024-01-15\nupdated: 2024-02-20T09:00:00Z\n---\n# Alpha\n\ncommon content\n")
	writeAPINote(t, root, "Research/Beta.md",
		"---\ntitle: Beta\ntags: [go]\ncreated: 2024-03-01\n---\n# Beta\n\ncommon content\n")
	writeAPINote(t, root, "Projects/Gamma.md",
		"---\ntitle: Gamma\ntags: [rust]\ncreated: 2024-01-10\n---\n# Gamma\n\ncommon content\n")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New: %v", err)
	}
	t.Cleanup(func() { _ = si.Close() })
	return v, si
}

func advancedHandler(t *testing.T) http.Handler {
	t.Helper()
	v, si := advancedVault(t)
	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)
	return r
}

func doAdvanced(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestAdvancedSearchReturnsEnvelope(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=content&sort=newest&limit=2")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var resp search.SearchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != 3 || len(resp.Results) != 2 {
		t.Fatalf("total=%d len=%d, want total=3 len=2", resp.Total, len(resp.Results))
	}
	if resp.Limit != 2 || resp.Offset != 0 {
		t.Fatalf("limit=%d offset=%d, want 2/0", resp.Limit, resp.Offset)
	}
	if resp.Sort != search.SearchSortNewest {
		t.Fatalf("sort = %q, want newest", resp.Sort)
	}
	if resp.Results[0].Date == nil {
		t.Fatal("first result should have a reconstructed date")
	}
}

func TestAdvancedSearchEmptyResultsAreArray(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=nomatch")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	// Results must serialize as [] not null.
	if !strings.Contains(rr.Body.String(), `"results":[]`) {
		t.Fatalf("expected empty results array, got %s", rr.Body.String())
	}
}

func TestAdvancedSearchFilterOnly(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?tag=go")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp search.SearchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("filter-only total = %d, want 2 (Alpha, Beta)", resp.Total)
	}
}

func TestAdvancedSearchInvalidBeforeDateFrom(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?date_from=2024-02-01&date_to=2024-01-01")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", rr.Code, rr.Body.String())
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Code != "invalid_search_request" {
		t.Fatalf("code = %q, want invalid_search_request", body.Error.Code)
	}
	if !hasAPIField(body.Error.Fields, "date_to", "before_date_from") {
		t.Fatalf("fields = %+v, want date_to/before_date_from", body.Error.Fields)
	}
}

func TestAdvancedSearchUnknownParameterRejected(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=x&bogus=1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !hasAPIField(body.Error.Fields, "bogus", "unknown_parameter") {
		t.Fatalf("fields = %+v, want bogus/unknown_parameter", body.Error.Fields)
	}
}

func TestAdvancedSearchRepeatedSingletonRejected(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=a&q=b")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !hasAPIField(body.Error.Fields, "q", "repeated_parameter") {
		t.Fatalf("fields = %+v, want q/repeated_parameter", body.Error.Fields)
	}
}

func TestAdvancedSearchInvalidLimit(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=x&limit=999")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !hasAPIField(body.Error.Fields, "limit", "limit_out_of_range") {
		t.Fatalf("fields = %+v, want limit/limit_out_of_range", body.Error.Fields)
	}
}

func TestAdvancedSearchExplicitZeroLimitRejected(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?q=x&limit=0")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for explicit limit=0", rr.Code)
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !hasAPIField(body.Error.Fields, "limit", "limit_out_of_range") {
		t.Fatalf("fields = %+v, want limit/limit_out_of_range for explicit zero", body.Error.Fields)
	}
}

func TestAdvancedSearchInvalidDate(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search/advanced?date_from=01/15/2024")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body advancedError
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !hasAPIField(body.Error.Fields, "date_from", "date_from_invalid") {
		t.Fatalf("fields = %+v, want date_from/date_from_invalid", body.Error.Fields)
	}
}

func TestLegacySearchStillBareArray(t *testing.T) {
	rr := doAdvanced(t, advancedHandler(t), "/api/search?q=content")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var results []search.SearchResult
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode legacy array: %v body=%s", err, rr.Body.String())
	}
	if len(results) == 0 {
		t.Fatal("legacy search should return results")
	}
	// A bare array decodes without a "total" wrapper; confirm it is not an envelope.
	if strings.Contains(rr.Body.String(), `"total":`) {
		t.Fatalf("legacy endpoint should return a bare array, got envelope: %s", rr.Body.String())
	}
}

func hasAPIField(fields []search.FieldError, field, code string) bool {
	for _, f := range fields {
		if f.Field == field && f.Code == code {
			return true
		}
	}
	return false
}
