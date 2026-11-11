package validate

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIJSONLAndStrict(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "retro")
	build := exec.Command("go", "build", "-o", bin, "./cmd/retro-obsidian-publish")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	vault := t.TempDir()
	if err := os.WriteFile(filepath.Join(vault, "Note.md"), []byte("# Note\n\n[[Missing]]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, strict := range []bool{false, true} {
		args := []string{"validate", "--vault", vault, "--format", "jsonl"}
		if strict {
			args = append(args, "--strict")
		}
		cmd := exec.Command(bin, args...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if strict && err == nil || !strict && err != nil {
			t.Fatalf("strict %t: %v stderr=%s", strict, err, stderr.String())
		}
		var row map[string]any
		if e := json.Unmarshal(out, &row); e != nil {
			t.Fatalf("strict %t invalid JSONL %q: %v", strict, out, e)
		}
		if row["rule"] != "WIKI_TARGET_MISSING" {
			t.Fatalf("row=%v", row)
		}
		if strict && !strings.Contains(stderr.String(), "validation found") {
			t.Fatalf("missing strict error: %s", stderr.String())
		}
	}
}
