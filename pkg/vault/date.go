package vault

import (
	"strings"
	"time"
)

// DatePrecision records whether an authored date was a calendar date or a
// full timestamp. It is retained separately from time.Time because converting a
// date-only value to an instant would lose the author's intent and shift the
// displayed calendar day once rendered in a timezone.
type DatePrecision string

const (
	DatePrecisionDate      DatePrecision = "date"
	DatePrecisionTimestamp DatePrecision = "timestamp"
)

// NoteDateKind identifies which authored concept a date represents.
type NoteDateKind string

const (
	NoteDateCreated NoteDateKind = "created"
	NoteDateUpdated NoteDateKind = "updated"
)

// InvalidDateReason is a finite, content-free reason an authored date was
// rejected. Reasons are safe for metrics and logs; they never encode the
// frontmatter key spelling or the raw value.
type InvalidDateReason string

const (
	InvalidDateWrongType       InvalidDateReason = "wrong_type"
	InvalidDateInvalidFormat   InvalidDateReason = "invalid_format"
	InvalidDateInvalidCalendar InvalidDateReason = "invalid_calendar_date"
)

// NoteDate is one resolved authored date value.
type NoteDate struct {
	// Value is the indexed instant. Date-only values use midnight UTC; this is
	// an indexing representation, not a claim that the author acted at midnight UTC.
	Value time.Time `json:"-"`
	// Precision is date or timestamp.
	Precision DatePrecision `json:"precision"`
	// SourceKey is the canonical lower-case frontmatter key selected by precedence.
	SourceKey string `json:"sourceKey"`
	// Original is the original scalar string, retained for API projection and
	// diagnostics. It is never exposed in metrics.
	Original string `json:"value"`
}

// APIValue returns the value string used in API projections: the original
// literal for date precision, or a normalized UTC RFC3339 instant for
// timestamp precision.
func (d *NoteDate) APIValue() string {
	if d == nil {
		return ""
	}
	if d.Precision == DatePrecisionDate {
		return d.Original
	}
	return d.Value.UTC().Format(time.RFC3339)
}

// NoteDates holds resolved created and updated dates for a note.
type NoteDates struct {
	Created *NoteDate `json:"created,omitempty"`
	Updated *NoteDate `json:"updated,omitempty"`
}

// Display returns the kind and date used for display: updated over created,
// otherwise ("", nil). Missing authored dates stay missing.
func (d NoteDates) Display() (NoteDateKind, *NoteDate) {
	if d.Updated != nil {
		return NoteDateUpdated, d.Updated
	}
	if d.Created != nil {
		return NoteDateCreated, d.Created
	}
	return "", nil
}

// DateWarning is a content-free record that an authored date was rejected. It
// is safe for metrics/logs; the Key is retained only for development diagnostics
// and must not become a metrics label.
type DateWarning struct {
	Concept NoteDateKind
	Reason  InvalidDateReason
	Key     string
}

// createdAliases and updatedAliases are the accepted frontmatter keys, in
// precedence order (highest first).
var (
	createdAliases = []string{"created", "date"}
	updatedAliases = []string{"updated", "modified", "last_updated"}
)

// ResolveNoteDates resolves authored created and updated dates from
// frontmatter using strict YYYY-MM-DD or RFC3339 parsing. Missing or invalid
// values produce nil entries and content-free warnings. A higher-precedence
// invalid value does not silently fall through to a lower alias: an author
// mistake in the selected key stays visible rather than being masked by a
// stale lower-priority value.
func ResolveNoteDates(frontmatter map[string]interface{}) (NoteDates, []DateWarning) {
	var warnings []DateWarning
	created, cw := resolveConcept(frontmatter, NoteDateCreated, createdAliases)
	warnings = append(warnings, cw...)
	updated, uw := resolveConcept(frontmatter, NoteDateUpdated, updatedAliases)
	warnings = append(warnings, uw...)
	return NoteDates{Created: created, Updated: updated}, warnings
}

// resolveConcept selects the first existing alias (case-insensitive) and parses
// its value. It does not fall through to a lower alias when the selected value
// is invalid.
func resolveConcept(fm map[string]interface{}, concept NoteDateKind, aliases []string) (*NoteDate, []DateWarning) {
	canonical, actualKey, ok := lookupAlias(fm, aliases)
	if !ok {
		return nil, nil
	}
	raw := fm[actualKey]
	s, ok := raw.(string)
	if !ok {
		return nil, []DateWarning{{Concept: concept, Reason: InvalidDateWrongType, Key: canonical}}
	}
	nd, reason, ok := parseNoteDate(s, canonical)
	if !ok {
		return nil, []DateWarning{{Concept: concept, Reason: reason, Key: canonical}}
	}
	return &nd, nil
}

// lookupAlias returns the first alias (in precedence order) that exists in fm,
// matched case-insensitively. The first return is the canonical lower-case
// alias; the second is the spelling present in fm.
func lookupAlias(fm map[string]interface{}, aliases []string) (string, string, bool) {
	if fm == nil {
		return "", "", false
	}
	lower := make(map[string]string, len(fm))
	for k := range fm {
		lower[strings.ToLower(k)] = k
	}
	for _, want := range aliases {
		if actual, exists := lower[want]; exists {
			return want, actual, true
		}
	}
	return "", "", false
}

// parseNoteDate parses a strict date-only or RFC3339 value. Date-only values
// become midnight UTC with date precision; timestamps are normalized to UTC
// with timestamp precision. It returns a finite, content-free reason on
// failure so the loader can count rejections without encoding values.
func parseNoteDate(value, sourceKey string) (NoteDate, InvalidDateReason, bool) {
	if isStrictDateOnly(value) {
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return NoteDate{}, InvalidDateInvalidCalendar, false
		}
		return NoteDate{Value: t.UTC(), Precision: DatePrecisionDate, SourceKey: sourceKey, Original: value}, "", true
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return NoteDate{}, InvalidDateInvalidFormat, false
	}
	return NoteDate{Value: t.UTC(), Precision: DatePrecisionTimestamp, SourceKey: sourceKey, Original: value}, "", true
}

// isStrictDateOnly reports whether value is exactly a YYYY-MM-DD literal with
// zero-padded month and day. It rejects partial dates such as 2024-1-5 and any
// value with a time component.
func isStrictDateOnly(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9} {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// noteDateInstant returns the indexed instant for a resolved date, or nil when
// the date is absent. Callers copy the time out so the returned pointer is not
// aliased to the NoteDate value.
func noteDateInstant(d *NoteDate) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Value
	return &t
}

// noteDateDisplayInstant returns the display instant (updated over created), or
// nil when no authored date exists.
func noteDateDisplayInstant(dates NoteDates) *time.Time {
	_, display := dates.Display()
	return noteDateInstant(display)
}

// noteDateKindString returns "created", "updated", or "" for the display date.
func noteDateKindString(dates NoteDates) string {
	kind, _ := dates.Display()
	return string(kind)
}

// dateWarningKey is the content-free metrics/log key for a rejected date.
func dateWarningKey(w DateWarning) string {
	return string(w.Concept) + ":" + string(w.Reason)
}
