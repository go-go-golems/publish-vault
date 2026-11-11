package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidationSnapshotAccess(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Note.md", "# Note\n\n## Details\n")
	v, err := New(root, WithStrictLoad())
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := v.ResolveHeading("note", "Details"); !ok || id != "details" {
		t.Fatalf("heading=%q %t", id, ok)
	}
	src, ok := v.SnapshotSource("note")
	if !ok {
		t.Fatal("missing source")
	}
	src[0] = '!'
	again, ok := v.SnapshotSource("note")
	if !ok || again[0] != '#' {
		t.Fatal("source copy mutated snapshot")
	}
	if _, ok := v.SnapshotSource("missing"); ok {
		t.Fatal("missing source found")
	}
}
func TestStrictLoadRejectsUnreadableIgnore(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vault-ignore"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, WithStrictLoad()); err == nil {
		t.Fatal("strict load accepted unreadable ignore")
	}
}
