// Package ignore parses a .vault-ignore file and answers path-exclusion
// queries for the vault loader, the file watcher, and the static-asset handler.
//
// The supported syntax is a documented subset of gitignore: blank lines and
// "#" comments are ignored; a leading "!" negates a pattern (last match wins);
// a trailing "/" restricts a pattern to directories; a leading "/" or any
// internal "/" anchors the pattern to the vault root; otherwise the pattern
// matches a single path segment at any depth. Globs use stdlib path.Match
// semantics ("*", "?", "[abc]"). Negation cannot re-include a file under an
// excluded *directory* in strict gitignore fashion; this package treats
// negation as a simple last-match override (see ErrIgnored docs in the vault
// package). Nested .vault-ignore files and "**" are not supported.
package ignore

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// IgnoreFile is the conventional name of the ignore file at the vault root.
const IgnoreFile = ".vault-ignore"

// pattern is one compiled line of a .vault-ignore file.
type pattern struct {
	negate   bool   // true if the line started with "!"
	dirOnly  bool   // true if the line ended with "/"
	anchored bool   // true if the line is rooted at the vault root
	pat      string // compiled glob body (no "!", no trailing "/")
	raw      string // original line, for debugging
}

// Ignore holds the compiled patterns from one .vault-ignore file. A nil or
// empty Ignore matches nothing.
type Ignore struct {
	patterns []pattern
}

// Load reads <root>/.vault-ignore. If the file does not exist, Load returns an
// empty Ignore and a nil error so callers can treat "no ignore file" and
// "empty ignore file" identically. Any other read or parse error returns an
// empty Ignore and the error; callers should log and proceed.
func Load(root string) (*Ignore, error) {
	return LoadFromPath(filepath.Join(root, IgnoreFile))
}

// LoadFromPath reads an explicit ignore file path. See Load for absence handling.
func LoadFromPath(p string) (*Ignore, error) {
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Ignore{}, nil
		}
		return &Ignore{}, err
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse compiles patterns from a reader.
func Parse(r io.Reader) (*Ignore, error) {
	ig := &Ignore{}
	sc := bufio.NewScanner(r)
	// Allow long lines (vault paths can be long).
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if pat, ok := parseLine(sc.Text()); ok {
			ig.patterns = append(ig.patterns, pat)
		}
	}
	if err := sc.Err(); err != nil {
		return &Ignore{}, err
	}
	return ig, nil
}

// parseLine compiles one physical line into a pattern. Returns ok=false for
// blank lines, comments, and malformed patterns (which are skipped silently).
func parseLine(line string) (pattern, bool) {
	// Strip trailing whitespace and CR so CRLF files work.
	line = strings.TrimRight(line, " \t\r")
	if line == "" {
		return pattern{}, false
	}
	// Leading whitespace is trimmed only for directive detection; gitignore
	// treats leading spaces as literal, but vault-ignore users do not rely on
	// that and trimming makes "#"/"!" detection predictable.
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return pattern{}, false
	}

	p := pattern{raw: line}
	s := trimmed

	// Unescape leading "\#" / "\!" so literal "#"/"!" filenames can be matched.
	// Otherwise "#" starts a comment and "!" starts a negation.
	switch {
	case strings.HasPrefix(s, `\#`):
		s = "#" + s[2:]
	case strings.HasPrefix(s, `\!`):
		s = "!" + s[2:]
	case strings.HasPrefix(s, "#"):
		return pattern{}, false
	case strings.HasPrefix(s, "!"):
		p.negate = true
		s = s[1:]
	}

	// Trailing "/" => directory-only.
	if strings.HasSuffix(s, "/") {
		p.dirOnly = true
		s = strings.TrimSuffix(s, "/")
	}
	if s == "" {
		// A bare "!" or "/" — nothing to match.
		return pattern{}, false
	}

	// A leading "/" anchors to the vault root (and is otherwise consumed).
	// Any internal "/" also anchors the pattern.
	if strings.HasPrefix(s, "/") {
		p.anchored = true
		s = strings.TrimPrefix(s, "/")
	} else if strings.Contains(s, "/") {
		p.anchored = true
	}
	if s == "" {
		return pattern{}, false
	}

	// Validate the glob so a malformed pattern (e.g. unclosed "[") is skipped
	// rather than panicking later inside path.Match.
	if _, err := path.Match(s, ""); err != nil {
		return pattern{}, false
	}
	p.pat = s
	return p, true
}

// Match reports whether relPath (vault-relative, optionally slash- or
// OS-separated) is ignored. isDir indicates whether the path is a directory;
// directory-only patterns only match directories (or files beneath matched
// directories). Patterns are evaluated in file order with last-match-wins, so a
// later "!" negation overrides an earlier exclusion.
func (ig *Ignore) Match(relPath string, isDir bool) bool {
	if ig == nil || len(ig.patterns) == 0 {
		return false
	}
	rel := cleanRel(relPath)
	if rel == "" || rel == "." {
		return false
	}
	ignored := false
	for _, p := range ig.patterns {
		if p.matches(rel, isDir) {
			ignored = !p.negate
		}
	}
	return ignored
}

// MatchAbs converts absPath to a vault-relative path against root and calls
// Match. absPath must be inside root; if it is not, MatchAbs returns false.
func (ig *Ignore) MatchAbs(root, absPath string, isDir bool) bool {
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return false
	}
	return ig.Match(rel, isDir)
}

// Empty reports whether no patterns were loaded.
func (ig *Ignore) Empty() bool {
	return ig == nil || len(ig.patterns) == 0
}

// HasNegations reports whether any pattern is a negation (!). When negations
// are present, directory pruning is unsafe: a later "!" could re-include a file
// beneath an otherwise-ignored directory, and pruning the directory would
// silently drop that file before it can be matched. Callers that prune ignored
// directories (the vault walk and the file watcher) therefore descend and match
// each file individually when HasNegations is true.
func (ig *Ignore) HasNegations() bool {
	if ig == nil {
		return false
	}
	for _, p := range ig.patterns {
		if p.negate {
			return true
		}
	}
	return false
}

// matches reports whether the pattern excludes rel. A pattern matches rel if it
// matches rel itself or any ancestor directory of rel (so "foo/" excludes
// "foo/bar.md"). Directory-only patterns never match a file directly, but still
// match files via an ancestor directory.
func (p pattern) matches(rel string, isDir bool) bool {
	candidates := ancestors(rel)
	// Include rel itself unless this is a directory-only pattern and rel is a
	// file (dir-only patterns match files only through an ancestor dir).
	if !p.dirOnly || isDir {
		candidates = append(candidates, rel)
	}
	for _, c := range candidates {
		if p.matchSegment(c) {
			return true
		}
	}
	return false
}

// matchSegment matches the pattern against one candidate path. Anchored
// patterns match the full candidate; unanchored patterns match the basename.
func (p pattern) matchSegment(candidate string) bool {
	if p.anchored {
		ok, _ := path.Match(p.pat, candidate)
		return ok
	}
	base := candidate
	if i := strings.LastIndex(candidate, "/"); i >= 0 {
		base = candidate[i+1:]
	}
	ok, _ := path.Match(p.pat, base)
	return ok
}

// ancestors returns the ancestor directory prefixes of rel (excluding rel
// itself). For "a/b/c" it returns ["a", "a/b"]; for "a" it returns [].
func ancestors(rel string) []string {
	parts := strings.Split(rel, "/")
	if len(parts) <= 1 {
		return nil
	}
	out := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		out = append(out, strings.Join(parts[:i], "/"))
	}
	return out
}

// cleanRel normalises a vault-relative path to a clean, slash-separated form
// with no leading "/".
func cleanRel(p string) string {
	p = filepath.ToSlash(p)
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	return p
}
