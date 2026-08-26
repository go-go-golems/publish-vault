package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-go-golems/publish-vault/pkg/search"
)

// advancedParamSpec declares each accepted advanced-search query parameter and
// whether it may repeat. Unknown keys are rejected so misspelled filters do not
// produce unexpectedly broad results; repeated singletons are rejected rather
// than taking the first or last value.
var advancedParamSpec = map[string]bool{
	"q":          false,
	"tag":        true,
	"tag_mode":   false,
	"path":       true,
	"date_field": false,
	"date_from":  false,
	"date_to":    false,
	"sort":       false,
	"limit":      false,
	"offset":     false,
}

// advancedErrorBody is the machine-readable error contract for invalid
// advanced-search requests. Field and Code are stable; Message is for users.
type advancedErrorBody struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  []search.FieldError `json:"fields,omitempty"`
}

// advancedError wraps the error body under an "error" key.
type advancedError struct {
	Error advancedErrorBody `json:"error"`
}

// parseAdvancedParams converts HTTP query parameters into a raw SearchRequest
// plus parse-time field errors (unknown parameters, repeated singletons, and
// non-numeric/invalid-date values). The caller runs NormalizeSearchRequest to
// apply defaults and semantic validation.
func parseAdvancedParams(values url.Values) (search.SearchRequest, []search.FieldError) {
	var req search.SearchRequest
	var errs []search.FieldError

	for k := range values {
		if _, ok := advancedParamSpec[k]; !ok {
			errs = append(errs, search.FieldError{Field: k, Code: "unknown_parameter", Message: "Unknown search parameter."})
		}
	}

	singleton := func(field, code string) (string, bool) {
		vs := values[field]
		if len(vs) > 1 {
			errs = append(errs, search.FieldError{Field: field, Code: "repeated_parameter", Message: "Parameter must appear at most once."})
			return "", false
		}
		if len(vs) == 1 {
			return vs[0], true
		}
		return "", false
	}

	if v, ok := singleton("q", "q"); ok {
		req.Query = v
	}
	req.Tags = values["tag"]
	if v, ok := singleton("tag_mode", "tag_mode"); ok {
		req.TagMode = search.TagMode(v)
	}
	req.PathPrefixes = values["path"]
	if v, ok := singleton("date_field", "date_field"); ok {
		req.DateField = search.DateField(v)
	}
	if v, ok := singleton("date_from", "date_from"); ok {
		if d, err := search.ParseDateOnly(v); err == nil {
			req.DateFrom = &d
		} else {
			errs = append(errs, search.FieldError{Field: "date_from", Code: "date_from_invalid", Message: "date_from must be YYYY-MM-DD."})
		}
	}
	if v, ok := singleton("date_to", "date_to"); ok {
		if d, err := search.ParseDateOnly(v); err == nil {
			req.DateTo = &d
		} else {
			errs = append(errs, search.FieldError{Field: "date_to", Code: "date_to_invalid", Message: "date_to must be YYYY-MM-DD."})
		}
	}
	if v, ok := singleton("sort", "sort"); ok {
		req.Sort = search.SearchSort(v)
	}
	if v, ok := singleton("limit", "limit"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			errs = append(errs, search.FieldError{Field: "limit", Code: "limit_out_of_range", Message: "limit must be an integer between 1 and 100."})
		} else if n == 0 {
			// 0 is the internal omitted-value sentinel; an explicitly supplied
			// zero is out of the documented 1-100 range, not a request for the
			// default. Reject it here so normalization does not silently turn
			// it into the default of 30.
			errs = append(errs, search.FieldError{Field: "limit", Code: "limit_out_of_range", Message: "limit must be between 1 and 100."})
		} else {
			req.Limit = n
		}
	}
	if v, ok := singleton("offset", "offset"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			req.Offset = n
		} else {
			errs = append(errs, search.FieldError{Field: "offset", Code: "offset_out_of_range", Message: "offset must be an integer between 0 and 10000."})
		}
	}

	return req, errs
}

// jsonStatusResponse writes a JSON body with an explicit status code. The
// Content-Type must be set before WriteHeader.
func jsonStatusResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// searchAdvanced handles GET /api/search/advanced. It parses and validates the
// typed request, returns a stable 400 envelope on invalid input, and delegates
// to the shared typed search implementation. Raw query values are never logged.
func (h *Handler) searchAdvanced(w http.ResponseWriter, r *http.Request) {
	rawReq, parseErrs := parseAdvancedParams(r.URL.Query())
	normReq, normErrs := search.NormalizeSearchRequest(rawReq)
	errs := append(parseErrs, normErrs...)
	if len(errs) > 0 {
		jsonStatusResponse(w, http.StatusBadRequest, advancedError{
			Error: advancedErrorBody{
				Code:    "invalid_search_request",
				Message: "One or more search parameters are invalid.",
				Fields:  errs,
			},
		})
		return
	}

	_, si := h.provider.Snapshot()
	resp, err := si.SearchAdvanced(normReq)
	if err != nil {
		jsonStatusResponse(w, http.StatusInternalServerError, advancedError{
			Error: advancedErrorBody{
				Code:    "search_unavailable",
				Message: "Search is temporarily unavailable.",
			},
		})
		return
	}
	if resp.Results == nil {
		resp.Results = []search.SearchResult{}
	}
	jsonResponse(w, resp)
}
