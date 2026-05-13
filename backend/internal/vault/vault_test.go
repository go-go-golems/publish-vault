package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBacklinksMarshalAsEmptyArray(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Index.md")
	if err := os.WriteFile(path, []byte("# Index\n\nNo incoming links."), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, ok := v.GetNote("index")
	if !ok {
		t.Fatal("index note not found")
	}
	if note.Backlinks == nil {
		t.Fatal("Backlinks is nil, want empty slice")
	}
	if note.Tags == nil {
		t.Fatal("Tags is nil, want empty slice")
	}
	if note.WikiLinks == nil {
		t.Fatal("WikiLinks is nil, want empty slice")
	}
	if note.Frontmatter == nil {
		t.Fatal("Frontmatter is nil, want empty object")
	}
	data, err := json.Marshal(note)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(data), `"backlinks":null`) {
		t.Fatalf("backlinks marshaled as null: %s", string(data))
	}
	jsonText := string(data)
	for _, field := range []string{"backlinks", "tags", "wikiLinks"} {
		if strings.Contains(jsonText, `"`+field+`":null`) {
			t.Fatalf("%s marshaled as null: %s", field, jsonText)
		}
		if !strings.Contains(jsonText, `"`+field+`":[]`) {
			t.Fatalf("%s did not marshal as []: %s", field, jsonText)
		}
	}
	if !strings.Contains(jsonText, `"frontmatter":{}`) {
		t.Fatalf("frontmatter did not marshal as {}: %s", jsonText)
	}
}
