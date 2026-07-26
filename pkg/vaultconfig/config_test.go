package vaultconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes content to <root>/.publish/config.yaml and returns root.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		dir := filepath.Join(root, ".publish")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadAbsentFileIsEmpty(t *testing.T) {
	root := t.TempDir() // no config
	cfg, err := LoadFromVaultRoot(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Ignore) != 0 {
		t.Fatalf("expected empty Ignore, got %v", cfg.Ignore)
	}
}

func TestLoadExplicitPathAbsent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load(missing) error = %v", err)
	}
	if len(cfg.Ignore) != 0 {
		t.Fatalf("expected empty Ignore for missing file, got %v", cfg.Ignore)
	}
}

func TestLoadParsesIgnoreList(t *testing.T) {
	content := "ignore:\n  - Secrets/\n  - \"*.draft.md\"\n  - \"!Drafts/Pinned.draft.md\"\n"
	root := writeConfig(t, content)
	cfg, err := LoadFromVaultRoot(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Ignore) != 3 {
		t.Fatalf("expected 3 patterns, got %d: %v", len(cfg.Ignore), cfg.Ignore)
	}
}

func TestLoadMalformedYAMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(bad, []byte("ignore: [unterminated"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(bad)
	if err == nil {
		t.Fatalf("expected error for malformed YAML, got nil")
	}
	if cfg == nil || len(cfg.Ignore) != 0 {
		t.Fatalf("expected empty Config on error, got %+v", cfg)
	}
}

func TestLoadIgnoresUnknownKeys(t *testing.T) {
	content := "ignore:\n  - Secrets/\nfutureField: 42\nanother:\n  nested: true\n"
	root := writeConfig(t, content)
	cfg, err := LoadFromVaultRoot(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Ignore) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(cfg.Ignore))
	}
}

// matchCase describes one expectation for the matcher.
type matchCase struct {
	name   string
	rel    string
	isDir  bool
	ignore bool
}

func runMatchTable(t *testing.T, patterns []string, cases []matchCase) {
	t.Helper()
	m, err := NewMatcher(&Config{Ignore: patterns})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := m.Match(c.rel, c.isDir)
			if got != c.ignore {
				t.Errorf("Match(%q, isDir=%v) = %v, want %v", c.rel, c.isDir, got, c.ignore)
			}
		})
	}
}

func TestMatcherEmptyConfigExcludesNothing(t *testing.T) {
	m, err := NewMatcher(&Config{})
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}
	if !m.Empty() {
		t.Fatalf("empty config matcher should be Empty()")
	}
	if m.Match("anything.md", false) {
		t.Fatalf("empty matcher should match nothing")
	}
}

func TestMatcherNilConfigExcludesNothing(t *testing.T) {
	m, err := NewMatcher(nil)
	if err != nil {
		t.Fatalf("NewMatcher(nil) error = %v", err)
	}
	if !m.Empty() {
		t.Fatalf("nil config matcher should be Empty()")
	}
}

func TestMatcherNilSafe(t *testing.T) {
	var m *Matcher
	if !m.Empty() {
		t.Fatalf("nil Matcher should be Empty()")
	}
	if m.Match("x.md", false) {
		t.Fatalf("nil Matcher should match nothing")
	}
}

// TestMatcherDoubleStarPattern pins the headline capability of the config
// blacklist: the "**" glob, which the legacy internal/ignore matcher
// deliberately does not support.
func TestMatcherDoubleStarPattern(t *testing.T) {
	runMatchTable(t, []string{"Secrets/**"}, []matchCase{
		{"file directly under", "Secrets/x.md", false, true},
		{"file nested deep", "Secrets/sub/deep/y.md", false, true},
		{"dir itself", "Secrets", true, true},
		{"unrelated file", "Notes.md", false, false},
		{"similar prefix not matched", "Secrets-backup/x.md", false, false},
	})
}

// TestMatcherDoubleStarPrefix covers "**/node_modules/" which matches the
// directory at any depth.
func TestMatcherDoubleStarPrefix(t *testing.T) {
	runMatchTable(t, []string{"**/node_modules/"}, []matchCase{
		{"top-level", "node_modules/c", false, true},
		{"nested", "a/b/node_modules/c", false, true},
		{"deep nested", "x/y/z/node_modules/pkg", false, true},
		{"unrelated", "src/index.ts", false, false},
	})
}

func TestMatcherNegation(t *testing.T) {
	runMatchTable(t, []string{"*.draft.md", "!Drafts/Pinned.draft.md"}, []matchCase{
		{"excluded draft", "Drafts/Other.draft.md", false, true},
		{"re-included draft", "Drafts/Pinned.draft.md", false, false},
		{"top-level draft still excluded", "WIP.draft.md", false, true},
	})
}

func TestMatcherDirectoryOnly(t *testing.T) {
	runMatchTable(t, []string{"Drafts/"}, []matchCase{
		{"dir matches", "Drafts", true, true},
		{"file of same name does not", "Drafts", false, false},
		{"file under dir matches", "Drafts/WIP.md", false, true},
		{"file under nested dir matches", "Drafts/sub/WIP.md", false, true},
		{"unrelated file does not", "Notes.md", false, false},
	})
}

func TestMatcherAnchored(t *testing.T) {
	runMatchTable(t, []string{"/Secrets"}, []matchCase{
		{"root-level dir", "Secrets", true, true},
		{"root-level file", "Secrets", false, true},
		{"nested same name NOT matched", "a/Secrets", false, false},
	})
}
