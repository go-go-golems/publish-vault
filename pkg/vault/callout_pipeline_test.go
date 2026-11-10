package vault

import (
	"strings"
	"testing"
)

func TestWikiIndexIsDeterministicAndCanonicalSlugsWin(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Target.md", "# Root Target\n")
	writeVaultTestFile(t, root, "A/Target.md", "# A Target\n")
	writeVaultTestFile(t, root, "B/Target.md", "# B Target\n")
	writeVaultTestFile(t, root, "A/Other.md", "# First Common Title\n")
	writeVaultTestFile(t, root, "B/Other.md", "# First Common Title\n")
	v, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		v.buildWikiLinkIndex()
		if got, _ := v.ResolveWikiLink("target"); got != "target" {
			t.Fatalf("canonical slug lost: %q", got)
		}
		if got, _ := v.ResolveWikiLink("other"); got != "a/other" {
			t.Fatalf("ambiguous alias is unstable: %q", got)
		}
		if got, _ := v.ResolveWikiLink("First Common Title"); got != "a/other" {
			t.Fatalf("title collision is unstable: %q", got)
		}
	}
}

func TestCalloutContentSurvivesVaultResolution(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Target.md", "# Target\n\n## Details\n")
	writeVaultTestFile(t, root, "Source.md", "# Source\n\n> [!note] See [[Target]]\n> First line\n> second with [[Target#Details]] and $x$.\n>\n> - item\n")
	v, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	note, ok := v.GetNote("source")
	if !ok {
		t.Fatal("source not found")
	}
	for _, want := range []string{`class="callout callout-note"`, `class="callout-title"`, `class="callout-body"`, `<ul>`, `data-target="target"`, `#details`, `class="math`} {
		if !strings.Contains(note.HTML, want) {
			t.Errorf("missing %q in %s", want, note.HTML)
		}
	}
	if strings.Contains(note.HTML, "<br") {
		t.Errorf("unexpected hard break in %s", note.HTML)
	}
}
