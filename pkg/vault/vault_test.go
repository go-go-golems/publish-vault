package vault

import (
	"encoding/json"
	"errors"
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

func TestCountReturnsLoadedNoteCount(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Notes/Second.md", "# Second\n")
	writeVaultTestFile(t, root, "Notes/ignored.txt", "not markdown")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := v.Count(); got != 2 {
		t.Fatalf("Count() = %d, want 2", got)
	}
}

func TestSearchDocumentsUsePlainMarkdownBody(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n\nSearchable **bold** text with [[Second|alias]] and `retroctl publish`.")
	writeVaultTestFile(t, root, "Second.md", "# Second\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	docs, err := v.SearchDocuments()
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}
	var indexDoc SearchDocument
	for _, doc := range docs {
		if doc.Slug == "index" {
			indexDoc = doc
			break
		}
	}
	if indexDoc.Slug == "" {
		t.Fatal("index search document not found")
	}
	if strings.Contains(indexDoc.Body, "<") || strings.Contains(indexDoc.Body, ">") {
		t.Fatalf("search body should not contain rendered HTML: %q", indexDoc.Body)
	}
	if !strings.Contains(indexDoc.Body, "Searchable bold text with alias and retroctl publish") {
		t.Fatalf("search body = %q, want markdown stripped body", indexDoc.Body)
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

func TestImageSourcesRewriteToAssets(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Projects/Article.md", "# Article\n\n![Planet](images/planet.png)\n![Absolute](/global/map.png)\n![Remote](https://example.com/remote.png)\n")
	writeVaultTestFile(t, root, "Projects/images/planet.png", "png")
	writeVaultTestFile(t, root, "global/map.png", "map")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, ok := v.GetNote("projects/article")
	if !ok {
		t.Fatal("article note not found")
	}
	if !strings.Contains(note.HTML, `src="/vault-assets/Projects/images/planet.png"`) {
		t.Fatalf("relative image was not rewritten relative to note path, got: %s", note.HTML)
	}
	if !strings.Contains(note.HTML, `src="/vault-assets/global/map.png"`) {
		t.Fatalf("root-relative image was not rewritten as vault-root asset, got: %s", note.HTML)
	}
	if !strings.Contains(note.HTML, `src="https://example.com/remote.png"`) {
		t.Fatalf("remote image should be preserved, got: %s", note.HTML)
	}
}

func TestResolveAssetURLRejectsTraversal(t *testing.T) {
	v := &Vault{}
	if got := v.ResolveAssetURL("Projects/Article.md", "../../secret.png"); got != "../../secret.png" {
		t.Fatalf("escaping traversal should be preserved, got: %q", got)
	}
	if got := v.ResolveAssetURL("Projects/Article.md", "../shared/image.png"); got != "/vault-assets/shared/image.png" {
		t.Fatalf("in-vault parent traversal should resolve, got: %q", got)
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

func writeVaultTestFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAllRespectsVaultIgnore(t *testing.T) {
	root := t.TempDir()
	// Published content.
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Notes/Public.md", "# Public\n")
	// Authoring scaffolding that must never be published.
	writeVaultTestFile(t, root, "ttmp/_guidelines/Style.md", "# Style\n")
	writeVaultTestFile(t, root, "ttmp/_templates/Note.md", "# Template\n")
	// A draft excluded by glob, with one re-included note.
	writeVaultTestFile(t, root, "Drafts/WIP.draft.md", "# WIP\n")
	writeVaultTestFile(t, root, "Drafts/Pinned.draft.md", "# Pinned\n")
	// A private folder.
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")

	ignore := "# scaffolding\nttmp/_guidelines/\nttmp/_templates/\n# drafts\n*.draft.md\n!Drafts/Pinned.draft.md\n# private\n/Secrets/\n"
	if err := os.WriteFile(filepath.Join(root, ".vault-ignore"), []byte(ignore), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	wantPresent := []string{"index", "notes/public", "drafts/pinned-draft"}
	for _, slug := range wantPresent {
		if _, ok := v.GetNote(slug); !ok {
			t.Errorf("expected note %q to be published, but it is absent", slug)
		}
	}

	wantAbsent := []string{"ttmp/_guidelines/style", "ttmp/_templates/note", "drafts/wip-draft", "secrets/secret"}
	for _, slug := range wantAbsent {
		if _, ok := v.GetNote(slug); ok {
			t.Errorf("expected note %q to be ignored, but it is present", slug)
		}
	}

	if got := v.Count(); got != len(wantPresent) {
		t.Errorf("Count() = %d, want %d (only published notes)", got, len(wantPresent))
	}

	// File tree must not contain ignored folders.
	tree := v.FileTree()
	names := folderAndFileNames(tree)
	for _, bad := range []string{"_guidelines", "_templates", "Secrets"} {
		if names[bad] {
			t.Errorf("file tree should omit ignored entry %q", bad)
		}
	}
	if !names["Notes"] {
		t.Errorf("file tree should still contain published folder Notes")
	}

	// Search documents must omit ignored notes.
	docs, err := v.SearchDocuments()
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}
	for _, d := range docs {
		if strings.HasPrefix(d.Slug, "ttmp/_guidelines") || strings.HasPrefix(d.Slug, "ttmp/_templates") || strings.HasPrefix(d.Slug, "secrets") {
			t.Errorf("search document %q should have been excluded", d.Slug)
		}
		if d.Slug == "drafts/wip-draft" {
			t.Errorf("draft WIP should have been excluded from search docs")
		}
	}
}

func TestLoadAllWithoutIgnoreFileIsUnchanged(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := v.Count(); got != 2 {
		t.Errorf("without .vault-ignore, Count() = %d, want 2", got)
	}
}

func TestReloadNoteIgnoresExcludedPath(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "tmp/_guidelines/Style.md", "# Style\n")
	if err := os.WriteFile(filepath.Join(root, ".vault-ignore"), []byte("tmp/_guidelines/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	// Edit the ignored file and reload; it must not enter the index.
	target := filepath.Join(root, "tmp/_guidelines/Style.md")
	if err := os.WriteFile(target, []byte("# Updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	note, err := v.ReloadNote(target)
	if !errors.Is(err, ErrIgnored) {
		t.Fatalf("ReloadNote(ignored) err = %v, want ErrIgnored", err)
	}
	if note != nil {
		t.Errorf("ReloadNote(ignored) should return nil note, got %v", note)
	}
	if _, ok := v.GetNote("ttmp/_guidelines/style"); ok {
		t.Errorf("ignored note must not appear in the index after ReloadNote")
	}
}

func TestReadRawRejectsIgnoredSlug(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")
	if err := os.WriteFile(filepath.Join(root, ".vault-ignore"), []byte("/Secrets/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := v.ReadRaw("Secrets/secret.md"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadRaw(ignored) err = %v, want os.ErrNotExist", err)
	}
	// A published note still reads fine.
	if _, err := v.ReadRaw("Index.md"); err != nil {
		t.Errorf("ReadRaw(published) err = %v, want nil", err)
	}
}

func TestIsIgnoredIsNilSafeWithoutIgnoreFile(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if v.IsIgnored(filepath.Join(root, "Index.md"), false) {
		t.Errorf("IsIgnored must be false when no ignore file is present")
	}
}

// TestLoadAllNegationUnderExcludedDir verifies the consistency fix for the
// permissive matcher: when a "!" re-includes a file beneath an excluded
// directory, LoadAll must NOT prune the directory, so the re-included file is
// actually visited and published. Other files under the excluded dir stay
// excluded. (Without the ShouldPruneDir guard, SkipDir would drop Public.md.)
func TestLoadAllNegationUnderExcludedDir(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")
	writeVaultTestFile(t, root, "Secrets/Public.md", "# Public\n")
	ignore := "/Secrets/\n!Secrets/Public.md\n"
	if err := os.WriteFile(filepath.Join(root, ".vault-ignore"), []byte(ignore), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// The negated file is re-included and published.
	if _, ok := v.GetNote("secrets/public"); !ok {
		t.Errorf("re-included note secrets/public should be published, but it is absent")
	}
	// The non-negated sibling stays excluded.
	if _, ok := v.GetNote("secrets/secret"); ok {
		t.Errorf("secrets/secret should remain ignored")
	}
	if got := v.Count(); got != 2 { // index + secrets/public
		t.Errorf("Count() = %d, want 2", got)
	}

	// The file tree contains the re-included note but not the excluded sibling.
	names := folderAndFileNames(v.FileTree())
	if !names["Secrets"] {
		t.Errorf("file tree should contain the Secrets folder (it holds a published note)")
	}
	if !names["Public"] {
		t.Errorf("file tree should contain the re-included Public note")
	}
	if names["secret"] {
		t.Errorf("file tree should omit the excluded secret note")
	}

	// The matcher, raw endpoint, and loader all agree: Public.md is not ignored.
	if v.IsIgnored(filepath.Join(root, "Secrets/Public.md"), false) {
		t.Errorf("IsIgnored(Secrets/Public.md) should be false (re-included)")
	}
	if _, err := v.ReadRaw("Secrets/Public.md"); err != nil {
		t.Errorf("ReadRaw(re-included) err = %v, want nil", err)
	}
	if _, err := v.ReadRaw("Secrets/secret.md"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadRaw(excluded) err = %v, want os.ErrNotExist", err)
	}
}

// folderAndFileNames flattens a FileNode tree into a set of entry names.
func folderAndFileNames(n *FileNode) map[string]bool {
	out := map[string]bool{}
	var walk func(*FileNode)
	walk = func(node *FileNode) {
		if node == nil {
			return
		}
		if node.Name != "root" {
			out[node.Name] = true
		}
		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}
