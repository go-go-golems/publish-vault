package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-go-golems/publish-vault/pkg/vaultconfig"
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

func TestLoadObserverReportsBoundedStageNoteAndByteProgress(t *testing.T) {
	root := t.TempDir()
	published := []byte("# Published\n\nbody")
	hidden := []byte("---\npublish: false\n---\n# Hidden")
	if err := os.WriteFile(filepath.Join(root, "Published.md"), published, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Hidden.md"), hidden, 0o600); err != nil {
		t.Fatal(err)
	}
	var progress []LoadProgress
	v, err := New(root, WithLoadObserver(func(value LoadProgress) { progress = append(progress, value) }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if v.Count() != 1 {
		t.Fatalf("published notes = %d, want 1", v.Count())
	}
	if len(progress) < 10 {
		t.Fatalf("too few progress events: %#v", progress)
	}
	first, walkDone := progress[0], progress[2]
	wantBytes := uint64(len(published) + len(hidden))
	if first.Stage != LoadStageWalkParse || first.ProcessedNotes != 0 || first.TotalNotes != 2 || first.TotalBytes != wantBytes {
		t.Fatalf("walk start = %#v", first)
	}
	if walkDone.Stage != LoadStageWalkParse || walkDone.ProcessedNotes != 2 || walkDone.ProcessedBytes != wantBytes {
		t.Fatalf("walk completion = %#v", walkDone)
	}
	var stages []LoadStage
	for _, value := range progress {
		if len(stages) == 0 || stages[len(stages)-1] != value.Stage {
			stages = append(stages, value.Stage)
		}
	}
	wantStages := []LoadStage{LoadStageWalkParse, LoadStageNormalize, LoadStageWikiLinks, LoadStageBacklinks, LoadStageRenderHTML}
	if !reflect.DeepEqual(stages, wantStages) {
		t.Fatalf("stages = %v, want %v", stages, wantStages)
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

func TestWikiImageEmbedsResolveToAssets(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Projects/Report.md",
		"# Report\n\n![[shell overview.png]]\n\n![[Attachments/deep/Exact Path.png]]\n\n![[missing.png]]\n")
	writeVaultTestFile(t, root, "Attachments/Shell Overview.png", "png")
	writeVaultTestFile(t, root, "Attachments/deep/Exact Path.png", "png")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, ok := v.GetNote("projects/report")
	if !ok {
		t.Fatal("report note not found")
	}
	// Basename lookup is case-insensitive (Obsidian shortest-path behavior).
	if !strings.Contains(note.HTML, `src="/vault-assets/Attachments/Shell%20Overview.png"`) {
		t.Fatalf("basename embed not resolved, got: %s", note.HTML)
	}
	// Full vault-relative paths resolve too.
	if !strings.Contains(note.HTML, `src="/vault-assets/Attachments/deep/Exact%20Path.png"`) {
		t.Fatalf("path embed not resolved, got: %s", note.HTML)
	}
	if !strings.Contains(note.HTML, "Image not found: missing.png") {
		t.Fatalf("missing asset should render broken marker, got: %s", note.HTML)
	}
	// Image embeds must not appear as backlink sources or wiki links.
	if len(note.WikiLinks) != 0 {
		t.Fatalf("image embeds leaked into WikiLinks: %#v", note.WikiLinks)
	}
}

func TestAssetEmbedResolvesPathSuffixes(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Note.md",
		"# N\n\n![[project-a/pic.png]]\n\n![[project-b/pic.png]]\n\n![[pic.png]]\n")
	writeVaultTestFile(t, root, "Attachments/project-a/pic.png", "a")
	writeVaultTestFile(t, root, "Attachments/project-b/pic.png", "b")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, _ := v.GetNote("note")
	// Obsidian shortest-path targets are suffixes that disambiguate duplicates.
	if !strings.Contains(note.HTML, `src="/vault-assets/Attachments/project-a/pic.png"`) {
		t.Fatalf("suffix project-a/pic.png not resolved: %s", note.HTML)
	}
	if !strings.Contains(note.HTML, `src="/vault-assets/Attachments/project-b/pic.png"`) {
		t.Fatalf("suffix project-b/pic.png not resolved: %s", note.HTML)
	}
	// Ambiguous bare basename resolves deterministically (lexicographically first).
	if p, ok := v.ResolveAssetEmbed("pic.png"); !ok || p != "Attachments/project-a/pic.png" {
		t.Fatalf("ambiguous basename = %q, %v", p, ok)
	}
}

func TestRefreshAssetIndexPicksUpNewFiles(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Note.md", "# N\n\n![[late.png]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := v.ResolveAssetEmbed("late.png"); ok {
		t.Fatal("late.png should not resolve before it exists")
	}
	note, _ := v.GetNote("note")
	if !strings.Contains(note.HTML, "Image not found: late.png") {
		t.Fatalf("expected broken marker before asset exists: %s", note.HTML)
	}

	writeVaultTestFile(t, root, "Attachments/late.png", "png")
	v.RefreshAssetIndex()
	if p, ok := v.ResolveAssetEmbed("late.png"); !ok || p != "Attachments/late.png" {
		t.Fatalf("late.png should resolve after refresh, got %q, %v", p, ok)
	}
	// A note reload (what the watcher triggers on edit) re-renders with the
	// refreshed index.
	if _, err := v.ReloadNote(filepath.Join(root, "Note.md")); err != nil {
		t.Fatalf("ReloadNote: %v", err)
	}
	note, _ = v.GetNote("note")
	if !strings.Contains(note.HTML, `src="/vault-assets/Attachments/late.png"`) {
		t.Fatalf("reloaded note should resolve the new asset: %s", note.HTML)
	}
}

// TestLoadAllRespectsPublishFalseFrontmatter pins the per-note opt-out: a note
// with `publish: false` in frontmatter is parsed but not stored, so it is
// absent from the note index, the file tree, and search documents.
func TestLoadAllRespectsPublishFalseFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Public.md", "---\npublish: false\n---\n# Public\n")
	writeVaultTestFile(t, root, "Notes/AlsoPublic.md", "# Also Public\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, ok := v.GetNote("index"); !ok {
		t.Errorf("index should be published")
	}
	if _, ok := v.GetNote("notes/alsopublic"); !ok {
		t.Errorf("notes/alsopublic should be published")
	}
	if _, ok := v.GetNote("public"); ok {
		t.Errorf("public (publish: false) should be absent from notes")
	}
	if got := v.Count(); got != 2 {
		t.Errorf("Count() = %d, want 2 (publish:false note excluded)", got)
	}

	// File tree must omit the hidden note.
	tree := v.FileTree()
	names := folderAndFileNames(tree)
	if names["Public"] {
		t.Errorf("file tree should omit publish:false note Public")
	}
	if !names["Index"] {
		t.Errorf("file tree should still contain Index")
	}

	// Search documents must omit the hidden note.
	docs, err := v.SearchDocuments()
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}
	for _, d := range docs {
		if d.Slug == "public" {
			t.Errorf("search document %q should have been excluded", d.Slug)
		}
	}
}

// TestLoadAllPublishTrueStringAndBool pins that publish: true (bool) and
// publish: "true" (string) both keep a note eligible, and that case-insensitive
// key lookup works (Publish, PUBLISH).
func TestLoadAllPublishTrueStringAndBool(t *testing.T) {
	cases := []struct {
		name   string
		rel    string
		body   string
		expect bool
	}{
		{"bool true", "Bool.md", "---\npublish: true\n---\n# Bool\n", true},
		{"string true", "Str.md", "---\npublish: \"true\"\n---\n# Str\n", true},
		{"bool false", "Off.md", "---\npublish: false\n---\n# Off\n", false},
		{"uppercase key", "Up.md", "---\nPublish: false\n---\n# Up\n", false},
		{"absent", "NoKey.md", "# NoKey\n", true},
	}
	root := t.TempDir()
	for _, c := range cases {
		writeVaultTestFile(t, root, c.rel, c.body)
	}
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			slug := strings.TrimSuffix(strings.ToLower(c.rel), ".md")
			_, ok := v.GetNote(slug)
			if ok != c.expect {
				t.Errorf("GetNote(%q) = %v, want %v", slug, ok, c.expect)
			}
		})
	}
}

// TestLoadAllRespectsConfigBlacklist pins the config-file blacklist via the
// WithConfig option: a folder matched by a config pattern (here Secrets/**) is
// excluded from notes, the file tree, and search, while an un-matched note
// stays published. This is the headline capability (full ** semantics) that
// the legacy .vault-ignore matcher cannot express.
func TestLoadAllRespectsConfigBlacklist(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")
	writeVaultTestFile(t, root, "Secrets/sub/deep.md", "# Deep\n")
	writeVaultTestFile(t, root, "Notes/Public.md", "# Public\n")

	cfg := &vaultconfig.Config{Ignore: []string{"Secrets/**"}}
	v, err := New(root, WithConfig(cfg))
	if err != nil {
		t.Fatalf("New(WithConfig) error = %v", err)
	}

	if _, ok := v.GetNote("index"); !ok {
		t.Errorf("index should be published")
	}
	if _, ok := v.GetNote("notes/public"); !ok {
		t.Errorf("notes/public should be published")
	}
	for _, slug := range []string{"secrets/secret", "secrets/sub/deep"} {
		if _, ok := v.GetNote(slug); ok {
			t.Errorf("%q should be excluded by config blacklist", slug)
		}
	}

	tree := v.FileTree()
	names := folderAndFileNames(tree)
	if names["Secrets"] {
		t.Errorf("file tree should omit config-blacklisted folder Secrets")
	}
	if !names["Notes"] {
		t.Errorf("file tree should still contain published folder Notes")
	}

	docs, err := v.SearchDocuments()
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}
	for _, d := range docs {
		if strings.HasPrefix(d.Slug, "secrets") {
			t.Errorf("search document %q should have been excluded", d.Slug)
		}
	}
}

// TestIgnoredNoteWithPublishTrueStillHidden pins Decision A: publish is
// opt-out only. A note excluded by an ignore/config rule is NOT resurrected by
// publish: true; exclusion always wins.
func TestIgnoredNoteWithPublishTrueStillHidden(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/forced.md", "---\npublish: true\n---\n# Forced\n")

	cfg := &vaultconfig.Config{Ignore: []string{"Secrets/**"}}
	v, err := New(root, WithConfig(cfg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := v.GetNote("secrets/forced"); ok {
		t.Errorf("publish: true must not resurrect a config-excluded note")
	}
}

// TestReloadNoteDropsPublishFalse pins the watcher's incremental path: a note
// that was published and is then toggled to publish: false is removed from the
// index, and ReloadNote returns ErrIgnored so the watcher drops it from the
// search index.
func TestReloadNoteDropsPublishFalse(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Flip.md", "# Flip\n")
	writeVaultTestFile(t, root, "Other.md", "# Other\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := v.GetNote("flip"); !ok {
		t.Fatalf("flip should start published")
	}

	// Toggle to publish: false on disk.
	writeVaultTestFile(t, root, "Flip.md", "---\npublish: false\n---\n# Flip\n")
	_, err = v.ReloadNote(filepath.Join(root, "Flip.md"))
	if !errors.Is(err, ErrIgnored) {
		t.Fatalf("ReloadNote of publish:false note should return ErrIgnored, got %v", err)
	}
	if _, ok := v.GetNote("flip"); ok {
		t.Errorf("flip should be removed from notes after publish:false reload")
	}
	if got := v.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1 after dropping publish:false note", got)
	}
}

// TestReloadNoteExcludedByConfig pins that the config blacklist is consulted on
// incremental reload, not just full load: a note whose path is blacklisted is
// reported as ErrIgnored.
func TestReloadNoteExcludedByConfig(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/x.md", "# X\n")

	cfg := &vaultconfig.Config{Ignore: []string{"Secrets/**"}}
	v, err := New(root, WithConfig(cfg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := v.GetNote("secrets/x"); ok {
		t.Fatalf("secrets/x should be excluded by config on load")
	}
	_, err = v.ReloadNote(filepath.Join(root, "Secrets", "x.md"))
	if !errors.Is(err, ErrIgnored) {
		t.Errorf("ReloadNote of config-blacklisted path should return ErrIgnored, got %v", err)
	}
}

// TestReadRawExcludedByConfig pins that the raw-source endpoint cannot bypass
// the config blacklist.
func TestReadRawExcludedByConfig(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/secret.md", "# Secret\n")

	cfg := &vaultconfig.Config{Ignore: []string{"Secrets/**"}}
	v, err := New(root, WithConfig(cfg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := v.ReadRaw("Secrets/secret.md"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadRaw of config-blacklisted note should return os.ErrNotExist, got %v", err)
	}
	if _, err := v.ReadRaw("Index.md"); err != nil {
		t.Errorf("ReadRaw of published note should succeed, got %v", err)
	}
}

// TestReloadNoteDropsPublishFalseReportsUnpublished pins that the publish:false
// reload path is distinguishable from a plain ignore: the watcher needs to tell
// "this note just became hidden" (delete it from search) apart from "this path
// was never indexed" (no-op).
func TestReloadNoteDropsPublishFalseReportsUnpublished(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Flip.md", "# Flip\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	writeVaultTestFile(t, root, "Flip.md", "---\npublish: false\n---\n# Flip\n")
	_, err = v.ReloadNote(filepath.Join(root, "Flip.md"))
	if !errors.Is(err, ErrUnpublished) {
		t.Fatalf("ReloadNote of publish:false note = %v, want ErrUnpublished", err)
	}
	if !errors.Is(err, ErrIgnored) {
		t.Errorf("ErrUnpublished must wrap ErrIgnored so ignore-only callers keep working")
	}
	// A path excluded by config, by contrast, is only ErrIgnored: nothing was
	// ever indexed for it, so there is nothing to clean up.
	writeVaultTestFile(t, root, "Secrets/x.md", "# X\n")
	cfgVault, err := New(root, WithConfig(&vaultconfig.Config{Ignore: []string{"Secrets/**"}}))
	if err != nil {
		t.Fatalf("New(WithConfig) error = %v", err)
	}
	_, err = cfgVault.ReloadNote(filepath.Join(root, "Secrets", "x.md"))
	if errors.Is(err, ErrUnpublished) {
		t.Errorf("ReloadNote of config-excluded path = %v, want plain ErrIgnored", err)
	}
	if !errors.Is(err, ErrIgnored) {
		t.Errorf("ReloadNote of config-excluded path = %v, want ErrIgnored", err)
	}
}

// TestSlugForPath pins the slug the watcher addresses secondary indexes with
// for a note the vault has already dropped.
func TestSlugForPath(t *testing.T) {
	root := t.TempDir()
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := v.SlugForPath(filepath.Join(root, "Notes", "My Note.md")); got != "notes/my-note" {
		t.Errorf("SlugForPath = %q, want notes/my-note", got)
	}
}

// TestConfigNegationBelowExcludedDir pins last-match-wins config negations
// beneath an excluded directory: the walk must descend into Secrets/ instead of
// pruning it, otherwise "!Secrets/Public.md" can never be evaluated.
func TestConfigNegationBelowExcludedDir(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Index.md", "# Index\n")
	writeVaultTestFile(t, root, "Secrets/Public.md", "# Public\n")
	writeVaultTestFile(t, root, "Secrets/Private.md", "# Private\n")
	writeVaultTestFile(t, root, "Secrets/diagram.png", "not really a png")

	cfg := &vaultconfig.Config{Ignore: []string{"Secrets/**", "!Secrets/Public.md"}}
	v, err := New(root, WithConfig(cfg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if v.ShouldPruneDir(filepath.Join(root, "Secrets")) {
		t.Errorf("ShouldPruneDir(Secrets) = true; a config negation below it must keep the walk descending")
	}
	if _, ok := v.GetNote("secrets/public"); !ok {
		t.Errorf("secrets/public should be re-included by the negation")
	}
	if _, ok := v.GetNote("secrets/private"); ok {
		t.Errorf("secrets/private should stay excluded")
	}
}

// TestEmbedOfNoteBecomingPublishedResolvesOnReload pins that the broken-embed
// marker is not permanent: rebuilding renders from the parser output, so an
// embed whose target was hidden by publish: false resolves once the target is
// published, without touching the referring note.
func TestEmbedOfNoteBecomingPublishedResolvesOnReload(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Host.md", "# Host\n\n![[Target]]\n")
	writeVaultTestFile(t, root, "Target.md", "---\npublish: false\n---\n# Target\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	host, ok := v.GetNote("host")
	if !ok {
		t.Fatal("host note missing")
	}
	if !strings.Contains(host.HTML, "wiki-embed-broken") {
		t.Fatalf("embed of unpublished target should render a broken marker, got %q", host.HTML)
	}

	// Publish the target; only the target is reloaded, as the watcher would do.
	writeVaultTestFile(t, root, "Target.md", "# Target\n")
	if _, err := v.ReloadNote(filepath.Join(root, "Target.md")); err != nil {
		t.Fatalf("ReloadNote(Target) error = %v", err)
	}
	host, ok = v.GetNote("host")
	if !ok {
		t.Fatal("host note missing after reload")
	}
	if strings.Contains(host.HTML, "wiki-embed-broken") {
		t.Errorf("embed should resolve once the target is published, got %q", host.HTML)
	}
	if !strings.Contains(host.HTML, `data-target="target"`) {
		t.Errorf("embed placeholder should survive the rebuild, got %q", host.HTML)
	}

	// ...and the marker comes back when the target is hidden again.
	writeVaultTestFile(t, root, "Target.md", "---\npublish: false\n---\n# Target\n")
	if _, err := v.ReloadNote(filepath.Join(root, "Target.md")); !errors.Is(err, ErrUnpublished) {
		t.Fatalf("ReloadNote(Target) = %v, want ErrUnpublished", err)
	}
	host, ok = v.GetNote("host")
	if !ok {
		t.Fatal("host note missing after unpublish")
	}
	if !strings.Contains(host.HTML, "wiki-embed-broken") {
		t.Errorf("embed should show the broken marker again once the target is hidden, got %q", host.HTML)
	}
}

// TestMathPlaceholdersSurviveRebuildHTML guards a real invariant rather than a
// hypothetical one: rebuildHTML runs four regex passes over every note's HTML
// on every vault reload (wiki-link targets, wiki-link display text, image
// sources, image embeds). None of them should match math markup — but they are
// regexes over HTML, so the only thing keeping that true is a test.
func TestMathPlaceholdersSurviveRebuildHTML(t *testing.T) {
	root := t.TempDir()
	src := "# Gaussian\n\n" +
		"Density $f(x) = \\frac{1}{\\sigma\\sqrt{2\\pi}}$ and see [[Other]].\n\n" +
		"$$\n\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n$$\n"
	if err := os.WriteFile(filepath.Join(root, "Gaussian.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Other.md"), []byte("# Other"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	note, ok := v.GetNote("gaussian")
	if !ok {
		t.Fatalf("GetNote(gaussian) not found")
	}
	before := note.HTML

	// The wiki link must actually have been resolved, otherwise this test would
	// pass trivially on a pipeline that never ran the rebuild passes at all.
	if !strings.Contains(before, `href="/note/other"`) {
		t.Fatalf("wiki link was not resolved, test is not exercising rebuildHTML:\n%s", before)
	}

	for _, want := range []string{
		`<span class="math math-inline">f(x) = \frac{1}{\sigma\sqrt{2\pi}}</span>`,
		`<div class="math math-display">`,
		`a &amp;= b \\`,
	} {
		if !strings.Contains(before, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, before)
		}
	}

	// A second rebuild (what a live reload triggers) must be a fixed point.
	v.mu.Lock()
	v.rebuildHTML()
	v.mu.Unlock()

	note, _ = v.GetNote("gaussian")
	if note.HTML != before {
		t.Errorf("rebuildHTML() is not idempotent over math markup:\nbefore: %s\nafter:  %s", before, note.HTML)
	}
}

// writeVaultFiles is a small helper for the PV-SLUG-020 lookup tests.
func writeVaultFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCanonicalSlugResolvesUserTypeableVariants covers the permanent half of
// PV-SLUG-020: slugify preserves a trailing "/" and a doubled "//", and GetNote
// is an exact map lookup, so URLs a reader can produce by hand were a hard 404
// on a note that exists.
func TestCanonicalSlugResolvesUserTypeableVariants(t *testing.T) {
	root := t.TempDir()
	writeVaultFiles(t, root, map[string]string{
		"Transcripts/2026/Designing RAG/The_Algebra.md": "# Algebra",
	})
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	const canonical = "transcripts/2026/designing-rag/the_algebra"

	if _, ok := v.GetNote(canonical); !ok {
		t.Fatalf("GetNote(%q) not found; slugs are %v", canonical, slugsOf(v))
	}

	for _, variant := range []string{
		canonical + "/",
		"/" + canonical,
		"transcripts/2026//designing-rag/the_algebra",
		"Transcripts/2026/Designing-RAG/The_Algebra",
	} {
		t.Run(variant, func(t *testing.T) {
			if _, ok := v.GetNote(variant); ok {
				t.Fatalf("GetNote(%q) unexpectedly matched exactly", variant)
			}
			got, ok := v.CanonicalSlug(variant)
			if !ok {
				t.Fatalf("CanonicalSlug(%q) found nothing, want %q", variant, canonical)
			}
			if got != canonical {
				t.Errorf("CanonicalSlug(%q) = %q, want %q", variant, got, canonical)
			}
		})
	}

	// The canonical slug itself must never report a redirect, or the API would
	// 308 a URL to itself and loop.
	if got, ok := v.CanonicalSlug(canonical); ok {
		t.Errorf("CanonicalSlug(%q) = (%q, true), want no redirect for the canonical form", canonical, got)
	}
	// A genuine miss stays a miss.
	if got, ok := v.CanonicalSlug("no/such/note"); ok {
		t.Errorf("CanonicalSlug(no/such/note) = (%q, true), want not found", got)
	}
}

func TestNormalizeSlugIsIdempotent(t *testing.T) {
	for _, in := range []string{
		"a/b", "a/b/", "/a/b", "a//b", "A/B", "  a/b  ", "///", "",
		"transcripts/2026/08/09/designing-rag/the_algebra_of_intervention_fields",
	} {
		once := normalizeSlug(in)
		if twice := normalizeSlug(once); twice != once {
			t.Errorf("normalizeSlug not idempotent for %q: %q -> %q", in, once, twice)
		}
	}
}

// TestCollidingSlugsAreBothPublished: two files whose paths slugify to the same
// string used to resolve last-write-wins, silently discarding one note. Both are
// now published, the lexically-first path keeping the natural slug so existing
// URLs are stable, and the later one getting a suffix derived from its own path.
func TestCollidingSlugsAreBothPublished(t *testing.T) {
	root := t.TempDir()
	writeVaultFiles(t, root, map[string]string{
		"Alpha/Note.md": "# Upper",
		"alpha/note.md": "# Lower",
	})
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(v.AllNotes()) < 2 {
		t.Skip("filesystem folded the two paths into one; nothing to disambiguate")
	}

	if _, ok := v.GetNote("alpha/note"); !ok {
		t.Errorf("the natural slug should still resolve; slugs are %v", slugsOf(v))
	}
	// Every note must be reachable at its own slug, and no two may share one.
	seen := map[string]string{}
	for _, n := range v.AllNotes() {
		if prev, dup := seen[n.Slug]; dup {
			t.Errorf("slug %q is shared by %q and %q", n.Slug, prev, n.Path)
		}
		seen[n.Slug] = n.Path
		if _, ok := v.GetNote(n.Slug); !ok {
			t.Errorf("note %q is not reachable at its own slug %q", n.Path, n.Slug)
		}
	}

	// The suffix must be stable: reloading the same vault must not renumber it.
	before := slugsOf(v)
	if err := v.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	after := slugsOf(v)
	sort.Strings(before)
	sort.Strings(after)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Errorf("slugs changed across reload:\n before %v\n after  %v", before, after)
	}

	// The watcher deletes by path, so a renamed note must still be found.
	for _, n := range v.AllNotes() {
		abs := filepath.Join(root, filepath.FromSlash(n.Path))
		if got := v.SlugForPath(abs); got != n.Slug {
			t.Errorf("SlugForPath(%q) = %q, want %q", n.Path, got, n.Slug)
		}
	}
}

// TestAmbiguousNormalizedKeyIsNotResolved guards buildNormalizedIndex directly:
// when two real slugs share a normalized key and neither is the canonical form,
// picking one would serve the wrong note.
func TestAmbiguousNormalizedKeyIsNotResolved(t *testing.T) {
	v := &Vault{notes: map[string]*Note{
		"a//b": {Slug: "a//b"},
		"a/b/": {Slug: "a/b/"},
	}}
	v.buildNormalizedIndex()
	if got, ok := v.CanonicalSlug("A/B"); ok {
		t.Errorf("CanonicalSlug(A/B) = (%q, true), want no redirect for an ambiguous key", got)
	}

	// With the canonical form present, it owns the key.
	v.notes["a/b"] = &Note{Slug: "a/b"}
	v.buildNormalizedIndex()
	if got, ok := v.CanonicalSlug("A/B"); !ok || got != "a/b" {
		t.Errorf("CanonicalSlug(A/B) = (%q, %v), want (a/b, true)", got, ok)
	}
}

// TestExcludedNotesRecordTheirReason turns the four silent drops into an
// answerable question.
func TestExcludedNotesRecordTheirReason(t *testing.T) {
	root := t.TempDir()
	writeVaultFiles(t, root, map[string]string{
		".vault-ignore": "drafts/\n",
		"drafts/Hid.md": "# Hidden by ignore",
		"Private.md":    "---\npublish: false\n---\n\n# Hidden by frontmatter\n",
		"Привет.md":     "# Non-latin filename",
		"Published.md":  "# Visible",
	})
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, ok := v.GetNote("published"); !ok {
		t.Fatalf("published note missing; slugs are %v", slugsOf(v))
	}
	// The degenerate slug must not have been stored under the empty key, which
	// is where every non-Latin filename used to collide.
	if _, ok := v.GetNote(""); ok {
		t.Errorf("a note was stored under the empty slug")
	}

	for path, want := range map[string]ExclusionReason{
		"drafts/Hid.md": ExcludedByIgnore,
		"Private.md":    ExcludedByPublish,
		"Привет.md":     ExcludedByEmptySlug,
	} {
		got, ok := v.ExclusionReasonFor(path)
		if !ok {
			t.Errorf("ExclusionReasonFor(%q) recorded nothing, want %q", path, want)
			continue
		}
		if got != want {
			t.Errorf("ExclusionReasonFor(%q) = %q, want %q", path, got, want)
		}
	}
	if _, ok := v.ExclusionReasonFor("Published.md"); ok {
		t.Errorf("a published note was recorded as excluded")
	}
}

func slugsOf(v *Vault) []string {
	var out []string
	for _, n := range v.AllNotes() {
		out = append(out, n.Slug)
	}
	return out
}

// TestReloadNotePreservesDisambiguatedSlugs: LoadAll renames the later of two
// colliding notes, but ReloadNote used to recompute the natural slug and assign
// it directly — overwriting the note that owned it while stranding the old
// suffixed entry, so a watched edit made one note vanish and duplicated the
// other until a full reload. (PR #18 review, P2.)
func TestReloadNotePreservesDisambiguatedSlugs(t *testing.T) {
	root := t.TempDir()
	writeVaultFiles(t, root, map[string]string{
		"Alpha/Note.md": "# Upper",
		"alpha/note.md": "# Lower",
	})
	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(v.AllNotes()) < 2 {
		t.Skip("filesystem folded the two paths into one")
	}

	before := map[string]string{}
	for _, n := range v.AllNotes() {
		before[n.Slug] = n.Path
	}

	// Reload each note in turn, as the file watcher does on an edit.
	for slug, rel := range before {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.WriteFile(abs, []byte("# edited "+slug), 0o644); err != nil {
			t.Fatal(err)
		}
		note, err := v.ReloadNote(abs)
		if err != nil {
			t.Fatalf("ReloadNote(%s) error = %v", rel, err)
		}
		if note.Slug != slug {
			t.Errorf("ReloadNote(%s) moved the note from slug %q to %q", rel, slug, note.Slug)
		}
	}

	after := map[string]string{}
	for _, n := range v.AllNotes() {
		after[n.Slug] = n.Path
	}
	if len(after) != len(before) {
		t.Fatalf("note count changed %d -> %d\nbefore %v\nafter  %v", len(before), len(after), before, after)
	}
	for slug, rel := range before {
		if after[slug] != rel {
			t.Errorf("slug %q maps to %q after reload, want %q", slug, after[slug], rel)
		}
	}
	// SlugForPath must keep agreeing with the index, since the watcher deletes
	// search documents by the slug it returns.
	for _, n := range v.AllNotes() {
		abs := filepath.Join(root, filepath.FromSlash(n.Path))
		if got := v.SlugForPath(abs); got != n.Slug {
			t.Errorf("SlugForPath(%q) = %q, want %q", n.Path, got, n.Slug)
		}
	}
}

// TestWikiLinkWithMarkdownExtensionResolvesToTheSameNote is the end-to-end guard
// for PV-WIKILINK-021. The vault deliberately also contains a decoy whose own
// slug is "…/thesis-md" — exactly what an unstripped "[[…/thesis.md]]" target
// slugifies to — so a regression does not merely leave the link unresolved, it
// silently points at the decoy. Both link forms must land on the real note.
func TestWikiLinkWithMarkdownExtensionResolvesToTheSameNote(t *testing.T) {
	root := t.TempDir()
	const dir = "Transcripts/2026/08/06/RAG DSL for Retrieval"
	writeVaultTestFile(t, root, dir+"/thesis.md", "# Doctoral thesis\n\n## Identity is an API decision\n\nbody\n")
	writeVaultTestFile(t, root, dir+"/thesis md.md", "# Decoy\n\nnot the note you wanted\n")
	writeVaultTestFile(t, root, "Research/Zoo.md",
		"# Zoo\n\n"+
			"- with: [["+dir+"/thesis.md#Identity is an API decision]]\n"+
			"- without: [["+dir+"/thesis#Identity is an API decision]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const wantSlug = "transcripts/2026/08/06/rag-dsl-for-retrieval/thesis"
	for _, target := range []string{dir + "/thesis.md", dir + "/thesis"} {
		got, ok := v.ResolveWikiLink(target)
		if !ok || got != wantSlug {
			t.Errorf("ResolveWikiLink(%q) = %q, %v; want %q, true", target, got, ok, wantSlug)
		}
	}

	zoo, ok := v.GetNote("research/zoo")
	if !ok {
		t.Fatal("zoo note missing")
	}
	wantHref := `href="/note/` + wantSlug + `#identity-is-an-api-decision"`
	if strings.Count(zoo.HTML, wantHref) != 2 {
		t.Fatalf("both link forms should render %s, got: %s", wantHref, zoo.HTML)
	}
	if strings.Contains(zoo.HTML, "thesis-md") {
		t.Fatalf("link leaked onto the decoy slug: %s", zoo.HTML)
	}
	if strings.Contains(zoo.HTML, "#unresolved-") {
		t.Fatalf("link stayed unresolved: %s", zoo.HTML)
	}

	// The backlink graph is fed from WikiLink.Target, so it has to agree.
	thesis, ok := v.GetNote(wantSlug)
	if !ok {
		t.Fatal("thesis note missing")
	}
	if len(thesis.Backlinks) != 1 || thesis.Backlinks[0] != "research/zoo" {
		t.Fatalf("backlinks = %#v, want [research/zoo]", thesis.Backlinks)
	}
}

// TestSelfHeadingLinksSurviveRebuild guards the [[#Heading]] fix against the
// vault layer. rebuildHTML re-runs every resolution pass over the parser output
// on each reload, so a same-note anchor that the parser got right could still be
// rewritten back into a /note/ link — ReplaceWikiLinksString rewrites hrefs, and
// ReplaceWikiLinkDisplay rewrites anchor text.
func TestSelfHeadingLinksSurviveRebuild(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Zoo.md",
		"# Zoo\n\n"+
			"- self: [[#9.2 Kernel K0: canonical identity]]\n"+
			"- other: [[Other]]\n\n"+
			"## 9.2 Kernel K0: canonical identity\n")
	writeVaultTestFile(t, root, "Other.md", "# Other\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	zoo, ok := v.GetNote("zoo")
	if !ok {
		t.Fatal("zoo note missing")
	}

	if !strings.Contains(zoo.HTML, `href="#92-kernel-k0-canonical-identity"`) {
		t.Fatalf("self heading anchor did not survive the vault passes: %s", zoo.HTML)
	}
	if !strings.Contains(zoo.HTML, `>9.2 Kernel K0: canonical identity</a>`) {
		t.Fatalf("self heading link lost its display text: %s", zoo.HTML)
	}
	if strings.Contains(zoo.HTML, `href="/note/#`) {
		t.Fatalf("self heading link was routed back through /note/: %s", zoo.HTML)
	}

	// The ordinary link next to it must still resolve, and the self link must
	// not have added a phantom edge to the graph.
	if !strings.Contains(zoo.HTML, `href="/note/other"`) {
		t.Fatalf("neighbouring note link broke: %s", zoo.HTML)
	}
	if len(zoo.WikiLinks) != 1 || zoo.WikiLinks[0].Target != "Other" {
		t.Fatalf("WikiLinks = %#v, want just the Other link", zoo.WikiLinks)
	}
}

// TestCrossNoteHeadingFragmentsUseTheTargetsRenderedIDs is the regression test
// for the cross-note half of the fragment bug. The parser writes a provisional
// fragment with slugify; the target note's heading ids come from goldmark, which
// disagrees with slugify on any heading containing punctuation. Before the fix
// these links opened the right note at the top of the page.
func TestCrossNoteHeadingFragmentsUseTheTargetsRenderedIDs(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Target.md",
		"# Target\n\n"+
			"## 9.2 Kernel K0: canonical identity\n\n"+
			"## Entity–Derivation–Observation Separation\n\n"+
			"## Notes\n\n## Notes\n")
	writeVaultTestFile(t, root, "Hidden.md", "---\npublish: false\n---\n\n# Hidden\n\n## Secret Section\n")
	writeVaultTestFile(t, root, "Source.md",
		"# Source\n\n"+
			"- punct: [[Target#9.2 Kernel K0: canonical identity]]\n"+
			"- dashes: [[Target#Entity–Derivation–Observation Separation]]\n"+
			"- dupe: [[Target#Notes]]\n"+
			"- absent: [[Target#no such heading]]\n"+
			"- hidden: [[Hidden#Secret Section]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	src, ok := v.GetNote("source")
	if !ok {
		t.Fatal("source note missing")
	}

	// goldmark deletes "." and the dashes; slugify would have hyphenated them.
	for _, want := range []string{
		`href="/note/target#92-kernel-k0-canonical-identity"`,
		`href="/note/target#entityderivationobservation-separation"`,
		`href="/note/target#notes"`,
	} {
		if !strings.Contains(src.HTML, want) {
			t.Errorf("expected %s, got: %s", want, src.HTML)
		}
	}
	for _, unwanted := range []string{
		`#9-2-kernel-k0-canonical-identity`,
		`#entity-derivation-observation-separation`,
		`#notes-1`, // duplicate headings: the first one wins
	} {
		if strings.Contains(src.HTML, unwanted) {
			t.Errorf("stale slugified fragment %s survived: %s", unwanted, src.HTML)
		}
	}
	// A heading the target does not have leaves the link working, without a
	// fragment that points at nothing.
	if !strings.Contains(src.HTML, `href="/note/target"`) {
		t.Errorf("absent heading should drop the fragment, not the link: %s", src.HTML)
	}
	if strings.Contains(src.HTML, "#no-such-heading") {
		t.Errorf("fragment for an absent heading should be dropped: %s", src.HTML)
	}
	// An unpublished target is not a note at all: the link is already unresolved
	// and must not be rewritten into a /note/ link by the fragment pass.
	if !strings.Contains(src.HTML, `href="#unresolved-hidden"`) {
		t.Errorf("link to an unpublished note should stay unresolved: %s", src.HTML)
	}
}

// TestCrossNoteHeadingFragmentsFollowTargetEdits guards the reload path: the
// fragment lives in the *linking* note's HTML but is derived from the *target*
// note, so renaming a heading has to re-resolve every link pointing at it.
func TestCrossNoteHeadingFragmentsFollowTargetEdits(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Target.md", "# Target\n\n## 1.1 First\n")
	writeVaultTestFile(t, root, "Source.md", "# Source\n\n[[Target#1.1 First]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	src, _ := v.GetNote("source")
	if !strings.Contains(src.HTML, `href="/note/target#11-first"`) {
		t.Fatalf("initial fragment wrong: %s", src.HTML)
	}

	// Rename the heading; the link now names a heading that no longer exists.
	writeVaultTestFile(t, root, "Target.md", "# Target\n\n## 2.2 Second\n")
	if _, err := v.ReloadNote(filepath.Join(root, "Target.md")); err != nil {
		t.Fatalf("ReloadNote: %v", err)
	}
	src, _ = v.GetNote("source")
	if strings.Contains(src.HTML, "#11-first") {
		t.Fatalf("stale fragment survived the target's edit: %s", src.HTML)
	}
	if !strings.Contains(src.HTML, `href="/note/target"`) {
		t.Fatalf("link should still open the target: %s", src.HTML)
	}

	// Put it back: the link must recover rather than stay dropped.
	writeVaultTestFile(t, root, "Target.md", "# Target\n\n## 1.1 First\n")
	if _, err := v.ReloadNote(filepath.Join(root, "Target.md")); err != nil {
		t.Fatalf("ReloadNote: %v", err)
	}
	src, _ = v.GetNote("source")
	if !strings.Contains(src.HTML, `href="/note/target#11-first"`) {
		t.Fatalf("fragment did not recover after the heading came back: %s", src.HTML)
	}
}

// TestUppercaseMarkdownExtensionIsStrippedEverywhere is the regression test for
// the first P2 on PR #19. The vault walk accepts "Note.MD" case-insensitively,
// but pathToSlug and buildWikiLinkIndex used to trim only a lowercase ".md" —
// so such a note was published at the slug "note-md", and making the wiki-link
// strip case-insensitive on its own would have turned [[Note.MD]] from a
// working link into a broken one. All three spellings must agree.
func TestUppercaseMarkdownExtensionIsStrippedEverywhere(t *testing.T) {
	root := t.TempDir()
	// A title that differs from the filename: otherwise the title-slug entry in
	// the wiki-link index masks the bug.
	writeVaultTestFile(t, root, "Upper.MD", "# A Different Title\n\nbody\n")
	writeVaultTestFile(t, root, "Linker.md", "# Linker\n\n[[Upper.MD]] [[Upper.md]] [[Upper]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	upper, ok := v.GetNote("upper")
	if !ok {
		t.Fatalf("note should be published at 'upper', not with the extension in its slug; slugs: %v", slugsOf(v))
	}
	if upper.Title != "A Different Title" {
		t.Errorf("Title = %q, want the H1", upper.Title)
	}

	linker, ok := v.GetNote("linker")
	if !ok {
		t.Fatal("linker note missing")
	}
	if got := strings.Count(linker.HTML, `href="/note/upper"`); got != 3 {
		t.Fatalf("all three spellings should resolve, got %d: %s", got, linker.HTML)
	}
	if strings.Contains(linker.HTML, "#unresolved-") {
		t.Fatalf("no link should be unresolved: %s", linker.HTML)
	}
}

var headingIDRe = regexp.MustCompile(`<h2 id="([^"]*)"`)

// TestCrossNoteHeadingFragmentWithMath pins that a heading containing math is
// still reachable from another note. The linking note and the target hold
// different sentinels for the same formula, so the two only become comparable
// once both are expressed as TeX.
func TestCrossNoteHeadingFragmentWithMath(t *testing.T) {
	root := t.TempDir()
	writeVaultTestFile(t, root, "Target.md", "# Target\n\n## The $\\sigma$ bound\n\nbody\n")
	writeVaultTestFile(t, root, "Source.md", "# Source\n\n[[Target#The $\\sigma$ bound]]\n")

	v, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	target, _ := v.GetNote("target")
	source, _ := v.GetNote("source")

	id := headingIDRe.FindStringSubmatch(target.HTML)
	if len(id) < 2 {
		t.Fatalf("target has no h2 id: %s", target.HTML)
	}
	want := `href="/note/target#` + id[1] + `"`
	if !strings.Contains(source.HTML, want) {
		t.Fatalf("expected %s, got: %s", want, source.HTML)
	}
	if strings.Contains(source.HTML, `data-heading="The <span`) {
		t.Fatalf("math markup injected into an attribute: %s", source.HTML)
	}
}
