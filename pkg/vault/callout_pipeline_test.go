package vault

import (
	"strings"
	"testing"
)

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
