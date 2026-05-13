package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileTreeSortedFoldersFirstAlpha(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Zebra.md":           "# Zebra",
		"Apple.md":           "# Apple",
		"mid/Beta.md":        "# Beta",
		"mid/Alpha.md":       "# Alpha",
		"Aardvark/Nested.md": "# Nested",
	}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	tree := v.FileTree()

	// Root children should be sorted: Aardvark (folder), mid (folder), then Apple, Zebra (files)
	if len(tree.Children) < 4 {
		t.Fatalf("expected >= 4 root children, got %d", len(tree.Children))
	}
	// Folders first
	for i := 0; i < 2; i++ {
		if !tree.Children[i].IsFolder {
			t.Fatalf("child %d (%s) should be a folder", i, tree.Children[i].Name)
		}
	}
	// Files after folders
	for i := 2; i < 4; i++ {
		if tree.Children[i].IsFolder {
			t.Fatalf("child %d (%s) should be a file", i, tree.Children[i].Name)
		}
	}
	// Folder order: Aardvark before mid
	if tree.Children[0].Name != "Aardvark" {
		t.Fatalf("first folder should be Aardvark, got %s", tree.Children[0].Name)
	}
	// File order: Apple before Zebra
	if tree.Children[2].Name != "Apple" {
		t.Fatalf("first file should be Apple, got %s", tree.Children[2].Name)
	}

	// Nested: mid folder children should be Alpha, Beta
	midFolder := tree.Children[1]
	if midFolder.Name != "mid" {
		t.Fatalf("second folder should be mid, got %s", midFolder.Name)
	}
	if len(midFolder.Children) < 2 {
		t.Fatalf("mid should have >= 2 children, got %d", len(midFolder.Children))
	}
	if midFolder.Children[0].Name != "Alpha" {
		t.Fatalf("mid/Alpha should come first, got %s", midFolder.Children[0].Name)
	}
}

func TestWikiLinkResolution(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Research/KB/Tribal/App-Auth.md":     "# App Auth\n\nContent here.",
		"Research/KB/Fundamentals/Access.md": "# Access Control\n\nSee [[Tribal/App-Auth]] for details.",
	}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify the short wiki link target resolves to the full slug
	resolved, ok := v.ResolveWikiLink("Tribal/App-Auth")
	if !ok {
		t.Fatal("ResolveWikiLink should find Tribal/App-Auth")
	}
	if resolved != "research/kb/tribal/app-auth" {
		t.Fatalf("expected resolved slug 'research/kb/tribal/app-auth', got '%s'", resolved)
	}

	// Verify backlinks are connected
	authNote, ok := v.GetNote("research/kb/tribal/app-auth")
	if !ok {
		t.Fatal("app-auth note not found")
	}
	if len(authNote.Backlinks) == 0 {
		t.Fatal("app-auth should have a backlink from Access")
	}
	if authNote.Backlinks[0] != "research/kb/fundamentals/access" {
		t.Fatalf("backlink should be from access note, got '%s'", authNote.Backlinks[0])
	}

	// Verify the HTML has the resolved href
	accessNote, ok := v.GetNote("research/kb/fundamentals/access")
	if !ok {
		t.Fatal("access note not found")
	}
	if !strings.Contains(accessNote.HTML, `href="/note/research/kb/tribal/app-auth"`) {
		t.Fatalf("HTML should contain resolved href, got: %s", accessNote.HTML)
	}
	if !strings.Contains(accessNote.HTML, `data-target="research/kb/tribal/app-auth"`) {
		t.Fatalf("HTML should contain resolved data-target, got: %s", accessNote.HTML)
	}
}

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
