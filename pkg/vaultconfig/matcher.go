package vaultconfig

import (
	"path/filepath"
	"strings"

	"github.com/sabhiram/go-gitignore"
)

// Matcher answers "is this vault-relative path excluded by the config
// blacklist?" It wraps the sabhiram/go-gitignore library, which implements full
// gitignore semantics including "**" patterns and negations.
//
// A nil or empty Matcher excludes nothing. The matcher is immutable after
// construction, so it is safe to read concurrently without a lock (mirroring the
// internal ignore.Ignore type).
type Matcher struct {
	ig           *ignore.GitIgnore
	hasNegations bool
}

// NewMatcher compiles the config's Ignore patterns. It returns a no-op matcher
// (Match always false) when cfg is nil or has no Ignore patterns. A nil error
// is always returned: the underlying library cannot fail at compile time
// (CompileIgnoreLines does not return an error), and a malformed pattern is
// silently ignored by the library rather than aborting publication, matching
// the internal ignore package's tolerant handling of bad patterns.
func NewMatcher(cfg *Config) (*Matcher, error) {
	if cfg == nil || len(cfg.Ignore) == 0 {
		return &Matcher{}, nil
	}
	return &Matcher{
		ig:           ignore.CompileIgnoreLines(cfg.Ignore...),
		hasNegations: hasNegations(cfg.Ignore),
	}, nil
}

// hasNegations reports whether any pattern re-includes a path with a leading
// "!". Comments and blank lines are skipped, mirroring the library's parsing.
func hasNegations(patterns []string) bool {
	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "!") {
			return true
		}
	}
	return false
}

// Match reports whether relPath (a vault-relative, slash-separated path) is
// excluded by the compiled patterns. isDir indicates whether the path is a
// directory; directory-only patterns (trailing "/") only match directories.
//
// relPath must be slash-separated and relative to the vault root. OS-native
// separators are normalized internally, so callers may pass either form.
//
// Directory-only semantics: the underlying library matches a directory-only
// pattern such as "Drafts/" against the directory itself only when the
// candidate path carries a trailing slash ("Drafts/"). Filesystem walks pass
// directory paths without a trailing slash, so when isDir is true Match also
// probes the candidate with a trailing slash appended. This aligns the
// directory-pruning decision (ShouldPruneDir) with the pattern's intent.
func (m *Matcher) Match(relPath string, isDir bool) bool {
	if m == nil || m.ig == nil {
		return false
	}
	rel := filepath.ToSlash(relPath)
	if rel == "" || rel == "." {
		return false
	}
	if m.ig.MatchesPath(rel) {
		return true
	}
	// For directory candidates, also test with a trailing slash so that
	// directory-only patterns (e.g. "Drafts/") match the bare directory name
	// passed by filepath.Walk.
	if isDir {
		return m.ig.MatchesPath(rel + "/")
	}
	return false
}

// Empty reports whether the matcher would exclude nothing. A nil Matcher is
// empty.
func (m *Matcher) Empty() bool {
	return m == nil || m.ig == nil
}

// HasNegations reports whether the compiled patterns contain a "!" negation.
// Callers that prune whole directories from a filesystem walk must consult this
// first: with last-match-wins semantics a pattern such as "!Secrets/Public.md"
// re-includes a file beneath a directory excluded by "Secrets/**", so the walk
// has to descend and match each file individually instead of pruning.
func (m *Matcher) HasNegations() bool {
	return m != nil && m.hasNegations
}
