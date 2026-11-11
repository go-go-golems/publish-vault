package validate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, name, value string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestPublishedWikiDiagnostics(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".vault-ignore", "Private/\n")
	put(t, root, ".publish/config.yaml", "ignore:\n  - Drafts/\n")
	put(t, root, "Private/Secret.md", "[[PrivateMissing]]")
	put(t, root, "Drafts/Draft.md", "[[DraftMissing]]")
	put(t, root, "Hidden.md", "---\npublish: false\n---\n[[HiddenMissing]]")
	put(t, root, "Target.md", "# Target\n\n## Existing\n")
	put(t, root, "diagram.png", "image")
	put(t, root, "Source.md", "# Source\n\n[[Target#Existing]] [[Missing#H]]\n[[Target#Gone]] [[#Nope]]\n![[diagram.png]] ![[missing.png]]\n`[[NotALink]]` [[Missing#H]]\n")
	got, err := Run(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d diagnostics: %#v", len(got), got)
	}
	for i, want := range []struct {
		rule string
		line int
	}{{"WIKI_TARGET_MISSING", 3}, {"WIKI_HEADING_MISSING", 4}, {"WIKI_HEADING_MISSING", 4}, {"WIKI_IMAGE_MISSING", 5}, {"WIKI_TARGET_MISSING", 6}} {
		if got[i].Rule != want.rule || got[i].Line != want.line || got[i].Path != "Source.md" || got[i].Column < 1 {
			t.Errorf("%d: %#v want %#v", i, got[i], want)
		}
	}
}
func TestValidationErrors(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".publish/config.yaml", "ignore: [\n")
	if _, err := Run(context.Background(), root, ""); err == nil || !strings.Contains(err.Error(), "config") {
		t.Fatalf("config error: %v", err)
	}
	if _, err := Run(context.Background(), filepath.Join(root, "missing"), ""); err == nil {
		t.Fatal("missing root accepted")
	}
}
func TestCancelled(t *testing.T) {
	root := t.TempDir()
	put(t, root, "note.md", "# Hi")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, root, ""); err == nil {
		t.Fatal("cancel ignored")
	}
}
