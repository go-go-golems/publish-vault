package ignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeIgnore writes content to <root>/.vault-ignore and returns root.
func writeIgnore(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, IgnoreFile), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadAbsentFileIsEmpty(t *testing.T) {
	root := t.TempDir() // no .vault-ignore
	ig, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ig.Empty() {
		t.Fatalf("Ignore should be empty when file is absent, got %d patterns", len(ig.patterns))
	}
	if ig.Match("anything.md", false) {
		t.Fatalf("empty Ignore should match nothing")
	}
	if ig.MatchAbs(root, filepath.Join(root, "deep/notes.md"), false) {
		t.Fatalf("empty Ignore should match nothing (abs)")
	}
}

func TestLoadFromPathExplicit(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "my-ignore")
	if err := os.WriteFile(custom, []byte("Secrets/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ig, err := LoadFromPath(custom)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}
	if !ig.Match("Secrets/keep.md", false) {
		t.Fatalf("expected Secrets/keep.md to be ignored")
	}
}

func TestLoadReadErrorIsReturned(t *testing.T) {
	// Point LoadFromPath at a directory (not a file) to trigger a non-NotExist error.
	dir := t.TempDir()
	ig, err := LoadFromPath(dir)
	if err == nil {
		t.Fatalf("expected an error when opening a directory, got nil")
	}
	if ig == nil || !ig.Empty() {
		t.Fatalf("expected an empty Ignore on error")
	}
}

// matchCase describes one expectation for the matcher.
type matchCase struct {
	name   string
	rel    string
	isDir  bool
	ignore bool
}

func runMatchTable(t *testing.T, content string, cases []matchCase) {
	t.Helper()
	root := writeIgnore(t, content)
	ig, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := ig.Match(c.rel, c.isDir)
			if got != c.ignore {
				t.Errorf("Match(%q, isDir=%v) = %v, want %v", c.rel, c.isDir, got, c.ignore)
			}
		})
	}
}

func TestMatchCommentsAndBlankLines(t *testing.T) {
	runMatchTable(t, `# a comment

   # indented comment
Index.md
`, []matchCase{
		{"comment not a pattern", "# a comment.md", false, false},
		{"real pattern matches", "Index.md", false, true},
		{"real pattern dir", "Index.md", true, true},
	})
}

func TestMatchDirectoryOnly(t *testing.T) {
	runMatchTable(t, "Drafts/\n", []matchCase{
		{"dir matches", "Drafts", true, true},
		{"file of same name does not", "Drafts", false, false},
		{"file under dir matches", "Drafts/WIP.md", false, true},
		{"file under nested dir matches", "Drafts/sub/WIP.md", false, true},
		{"unrelated file does not", "Notes.md", false, false},
	})
}

func TestMatchAnchored(t *testing.T) {
	runMatchTable(t, "ttmp/_guidelines\ntmp/_templates\n", []matchCase{
		{"anchored dir itself", "ttmp/_guidelines", true, true},
		{"file under anchored dir", "ttmp/_guidelines/Index.md", false, true},
		{"deep file under anchored dir", "ttmp/_guidelines/a/b.md", false, true},
		{"sibling-prefixed dir NOT matched", "ttmp/_guidelines-backup", true, false},
		{"file under sibling-prefixed dir NOT matched", "ttmp/_guidelines-backup/x.md", false, false},
		{"second anchored pattern", "tmp/_templates/y.md", false, true},
		{"unrelated top-level", "Notes.md", false, false},
	})
}

func TestMatchLeadingSlashAnchorsToRoot(t *testing.T) {
	runMatchTable(t, "/Secrets\n", []matchCase{
		{"root-level dir", "Secrets", true, true},
		{"root-level file", "Secrets", false, true},
		{"nested same name NOT matched", "a/Secrets", false, false},
	})
}

func TestMatchUnanchoredBasename(t *testing.T) {
	runMatchTable(t, "*.draft.md\n", []matchCase{
		{"top-level draft", "WIP.draft.md", false, true},
		{"nested draft", "Projects/Pinned.draft.md", false, true},
		{"non-draft", "Projects/Note.md", false, false},
		{"dir named like pattern still matches", "Old.draft.md", true, true},
	})
}

func TestMatchNegationLastWins(t *testing.T) {
	runMatchTable(t, "*.draft.md\n!Projects/Pinned.draft.md\n", []matchCase{
		{"excluded draft", "Projects/Other.draft.md", false, true},
		{"re-included draft", "Projects/Pinned.draft.md", false, false},
		{"top-level draft still excluded", "WIP.draft.md", false, true},
	})
}

func TestMatchEscapedBangAndHash(t *testing.T) {
	runMatchTable(t, `\!keep.md
\#hashy.md
`, []matchCase{
		{"literal bang filename", "!keep.md", false, true},
		{"literal hash filename", "#hashy.md", false, true},
		{"plain keep not matched", "keep.md", false, false},
	})
}

func TestMatchMalformedPatternSkipped(t *testing.T) {
	// Unclosed character class is a bad pattern; it must be skipped, not panic.
	runMatchTable(t, "[unfinished\nDrafts/\n", []matchCase{
		{"bad pattern ignored safely", "unfinished.md", false, false},
		{"valid pattern still works", "Drafts/x.md", false, true},
	})
}

func TestMatchAbsWithRoot(t *testing.T) {
	content := "ttmp/_guidelines/\n"
	root := writeIgnore(t, content)
	ig, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ig.MatchAbs(root, filepath.Join(root, "ttmp/_guidelines/Index.md"), false) {
		t.Errorf("MatchAbs should ignore file under excluded dir")
	}
	if ig.MatchAbs(root, filepath.Join(root, "Notes.md"), false) {
		t.Errorf("MatchAbs should not ignore unrelated file")
	}
	// Path outside root -> false (safe default).
	if ig.MatchAbs(root, "/etc/passwd", false) {
		t.Errorf("MatchAbs should return false for paths outside root")
	}
}

func TestMatchRootItselfNeverIgnored(t *testing.T) {
	// "." is the vault root; it must never be ignored.
	runMatchTable(t, "*\n", []matchCase{
		{"root not ignored", ".", true, false},
		{"empty not ignored", "", true, false},
		{"everything else ignored", "Notes.md", false, true},
	})
}

func TestMatchCRLFFile(t *testing.T) {
	// CRLF line endings must not leave a trailing "\r" on patterns.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, IgnoreFile), []byte("Drafts/\r\n*.draft.md\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ig, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ig.Match("Drafts/x.md", false) {
		t.Errorf("CRLF Drafts/ should match Drafts/x.md")
	}
	if !ig.Match("a/b.draft.md", false) {
		t.Errorf("CRLF *.draft.md should match a/b.draft.md")
	}
}

func TestParseFromReader(t *testing.T) {
	ig, err := Parse(strings.NewReader("# hi\nSecrets/\n!Secrets/keep.md\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !ig.Match("Secrets/secret.md", false) {
		t.Errorf("Secrets/secret.md should be ignored")
	}
	if ig.Match("Secrets/keep.md", false) {
		t.Errorf("Secrets/keep.md should be re-included by negation")
	}
}

// TestMatchNegationUnderExcludedDir pins the permissive, last-match-wins
// behavior: a "!" re-includes a file even when its parent directory is excluded
// by an earlier pattern. (LoadAll only prunes ignored directories when no
// negation exists, so this re-included file is actually visited and published.)
func TestMatchNegationUnderExcludedDir(t *testing.T) {
	runMatchTable(t, "/Secrets/\n!Secrets/Public.md\n", []matchCase{
		{"other file under dir still ignored", "Secrets/secret.md", false, true},
		{"re-included file is published", "Secrets/Public.md", false, false},
		{"excluded dir itself is ignored", "Secrets", true, true},
	})
}

func TestHasNegations(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"no patterns", "# only a comment\n", false},
		{"only positive patterns", "Drafts/\n*.draft.md\n", false},
		{"has a negation", "*.draft.md\n!Keep.draft.md\n", true},
		{"negation under dir", "Secrets/\n!Secrets/Public.md\n", true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			root := writeIgnore(t, c.content)
			ig, err := Load(root)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got := ig.HasNegations(); got != c.want {
				t.Errorf("HasNegations() = %v, want %v", got, c.want)
			}
		})
	}
}
