package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// dateCase is one row of the shared cross-language fixture.
type dateCase struct {
	Name             string                 `json:"name"`
	Frontmatter      map[string]interface{} `json:"frontmatter"`
	Created          *expectedDate          `json:"created"`
	Updated          *expectedDate          `json:"updated"`
	DisplayKind      string                 `json:"display_kind"`
	DisplayValue     *string                `json:"display_value"`
	DisplayPrecision string                 `json:"display_precision"`
	Warnings         []expectedWarning      `json:"warnings"`
}

type expectedDate struct {
	Value     string        `json:"value"`
	Precision DatePrecision `json:"precision"`
	SourceKey string        `json:"source_key"`
}

type expectedWarning struct {
	Concept string `json:"concept"`
	Reason  string `json:"reason"`
}

func loadDateCases(t *testing.T) []dateCase {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "search-date-cases.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var fixture struct {
		Cases []dateCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}
	return fixture.Cases
}

func noteDateMatches(t *testing.T, got *NoteDate, want *expectedDate, label string) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s: expected nil, got %+v", label, got)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s: expected non-nil, got nil", label)
	}
	if got.APIValue() != want.Value {
		t.Fatalf("%s: value = %q, want %q", label, got.APIValue(), want.Value)
	}
	if got.Precision != want.Precision {
		t.Fatalf("%s: precision = %q, want %q", label, got.Precision, want.Precision)
	}
	if got.SourceKey != want.SourceKey {
		t.Fatalf("%s: source_key = %q, want %q", label, got.SourceKey, want.SourceKey)
	}
}

func TestResolveNoteDatesFromFixture(t *testing.T) {
	for _, c := range loadDateCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			dates, warnings := ResolveNoteDates(c.Frontmatter)
			noteDateMatches(t, dates.Created, c.Created, "created")
			noteDateMatches(t, dates.Updated, c.Updated, "updated")

			kind, display := dates.Display()
			if kind != NoteDateKind(c.DisplayKind) {
				t.Fatalf("display kind = %q, want %q", kind, c.DisplayKind)
			}
			if c.DisplayValue == nil {
				if display != nil {
					t.Fatalf("display date = %+v, want nil", display)
				}
			} else {
				if display == nil {
					t.Fatalf("display date = nil, want %q", *c.DisplayValue)
				}
				if display.APIValue() != *c.DisplayValue {
					t.Fatalf("display value = %q, want %q", display.APIValue(), *c.DisplayValue)
				}
				if string(display.Precision) != c.DisplayPrecision {
					t.Fatalf("display precision = %q, want %q", display.Precision, c.DisplayPrecision)
				}
			}

			gotWarnings := make([]expectedWarning, 0, len(warnings))
			for _, w := range warnings {
				gotWarnings = append(gotWarnings, expectedWarning{Concept: string(w.Concept), Reason: string(w.Reason)})
			}
			sortWarnings := func(ws []expectedWarning) {
				sort.Slice(ws, func(i, j int) bool {
					if ws[i].Concept != ws[j].Concept {
						return ws[i].Concept < ws[j].Concept
					}
					return ws[i].Reason < ws[j].Reason
				})
			}
			sortWarnings(gotWarnings)
			sortWarnings(c.Warnings)
			if len(gotWarnings) != len(c.Warnings) {
				t.Fatalf("warnings = %+v, want %+v", gotWarnings, c.Warnings)
			}
			for i := range gotWarnings {
				if gotWarnings[i] != c.Warnings[i] {
					t.Fatalf("warnings = %+v, want %+v", gotWarnings, c.Warnings)
				}
			}
		})
	}
}

func TestParseNoteDate(t *testing.T) {
	tests := []struct {
		value     string
		ok        bool
		precision DatePrecision
		apiValue  string
		reason    InvalidDateReason
	}{
		{"2024-01-15", true, DatePrecisionDate, "2024-01-15", ""},
		{"2024-01-15T13:45:00-05:00", true, DatePrecisionTimestamp, "2024-01-15T18:45:00Z", ""},
		{"2024-01-15T18:45:00Z", true, DatePrecisionTimestamp, "2024-01-15T18:45:00Z", ""},
		{"2024-1-5", false, "", "", InvalidDateInvalidFormat},
		{"01/15/2024", false, "", "", InvalidDateInvalidFormat},
		{"2024-01-15 13:45", false, "", "", InvalidDateInvalidFormat},
		{"2024-13-40", false, "", "", InvalidDateInvalidCalendar},
		{"January 15", false, "", "", InvalidDateInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			nd, reason, ok := parseNoteDate(tt.value, "created")
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (reason %q)", ok, tt.ok, reason)
			}
			if !ok {
				if reason != tt.reason {
					t.Fatalf("reason = %q, want %q", reason, tt.reason)
				}
				return
			}
			if nd.Precision != tt.precision {
				t.Fatalf("precision = %q, want %q", nd.Precision, tt.precision)
			}
			if nd.APIValue() != tt.apiValue {
				t.Fatalf("api value = %q, want %q", nd.APIValue(), tt.apiValue)
			}
		})
	}
}

func TestIsStrictDateOnly(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"2024-01-15", true},
		{"2024-1-5", false},
		{"2024-01-15T00:00:00Z", false},
		{"2024-13-40", true}, // structurally date-only; calendar validity is checked later
		{"01/15/2024", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := isStrictDateOnly(tt.value); got != tt.want {
				t.Fatalf("isStrictDateOnly(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestNoteDateDisplayPrecedence(t *testing.T) {
	if kind, d := (NoteDates{}).Display(); kind != "" || d != nil {
		t.Fatal("empty dates should display as absent")
	}
	created := &NoteDate{Value: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Precision: DatePrecisionDate, SourceKey: "created", Original: "2024-01-15"}
	if kind, d := (NoteDates{Created: created}).Display(); kind != NoteDateCreated || d != created {
		t.Fatalf("created-only display = %q/%v, want created", kind, d)
	}
	updated := &NoteDate{Value: time.Date(2024, 2, 20, 9, 0, 0, 0, time.UTC), Precision: DatePrecisionTimestamp, SourceKey: "updated", Original: "2024-02-20T09:00:00Z"}
	if kind, d := (NoteDates{Created: created, Updated: updated}).Display(); kind != NoteDateUpdated || d != updated {
		t.Fatalf("updated takes precedence, got %q/%v", kind, d)
	}
}
