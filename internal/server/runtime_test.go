package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/publish-vault/internal/search"
	"github.com/go-go-golems/publish-vault/internal/vault"
)

func TestRuntimeStateResolvesSymlinkRootAndReloads(t *testing.T) {
	root := t.TempDir()
	worktree1 := filepath.Join(root, "wt1")
	worktree2 := filepath.Join(root, "wt2")
	link := filepath.Join(root, "current")
	writeVaultNote(t, worktree1, "Index.md", "# First\n\nold body")
	writeVaultNote(t, worktree2, "Index.md", "# Second\n\nnew body")
	if err := os.Symlink(worktree1, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	state, err := NewRuntimeState(link)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}
	v, _ := state.Snapshot()
	note, ok := v.GetNote("index")
	if !ok || note.Title != "First" {
		t.Fatalf("initial note = %#v ok=%v, want First", note, ok)
	}
	if state.ResolvedRoot() != worktree1 {
		t.Fatalf("ResolvedRoot() = %q, want %q", state.ResolvedRoot(), worktree1)
	}

	if err := os.Remove(link); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	if err := os.Symlink(worktree2, link); err != nil {
		t.Fatalf("symlink new: %v", err)
	}
	if err := state.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	v, _ = state.Snapshot()
	note, ok = v.GetNote("index")
	if !ok || note.Title != "Second" {
		t.Fatalf("reloaded note = %#v ok=%v, want Second", note, ok)
	}
	if state.ResolvedRoot() != worktree2 {
		t.Fatalf("ResolvedRoot() = %q, want %q", state.ResolvedRoot(), worktree2)
	}
}

func TestHealthHandlerIncludesMemoryStats(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n")

	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rr := httptest.NewRecorder()
	healthHandler(state).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health status = %d body=%s", rr.Code, rr.Body.String())
	}
	var got healthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("health json: %v body=%s", err, rr.Body.String())
	}
	if !got.OK {
		t.Fatal("health ok=false, want true")
	}
	if got.Notes != 1 {
		t.Fatalf("health notes = %d, want 1", got.Notes)
	}
	if got.VaultRoot != root || got.ConfiguredRoot != root {
		t.Fatalf("health roots = (%q, %q), want %q", got.VaultRoot, got.ConfiguredRoot, root)
	}
	if got.HeapSysBytes == 0 || got.NextGCBytes == 0 {
		t.Fatalf("health memory stats missing: %#v", got.memoryStats)
	}
}

func TestRuntimeStatePersistentSearchReloadDropsDeletedNotes(t *testing.T) {
	root := t.TempDir()
	worktree1 := filepath.Join(root, "wt1")
	worktree2 := filepath.Join(root, "wt2")
	link := filepath.Join(root, "current")
	writeVaultNote(t, worktree1, "Gone.md", "# Gone\n\nvanishingterm")
	writeVaultNote(t, worktree2, "Kept.md", "# Kept\n\nordinary content")
	if err := os.Symlink(worktree1, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	state, err := NewRuntimeStateWithOptions(link, RuntimeOptions{SearchIndexPath: filepath.Join(root, "search")})
	if err != nil {
		t.Fatalf("NewRuntimeStateWithOptions() error = %v", err)
	}
	_, si := state.Snapshot()
	results, err := si.Search("vanishingterm", 10)
	if err != nil {
		t.Fatalf("initial search error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("initial search did not find vanishingterm")
	}

	if err := os.Remove(link); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	if err := os.Symlink(worktree2, link); err != nil {
		t.Fatalf("symlink new: %v", err)
	}
	if err := state.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	v, si := state.Snapshot()
	results, err = si.Search("vanishingterm", 10)
	if err != nil {
		t.Fatalf("reloaded search error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("stale deleted note remained searchable: %#v", results)
	}
	kept, ok := v.GetNote("kept")
	if !ok || kept.Title != "Kept" {
		t.Fatalf("active vault missing kept note after reload: %#v ok=%v", kept, ok)
	}
}

func TestBuildSearchIndexReopensPersistentIndexAtFinalPath(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n\ninitialterm")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	searchBase := filepath.Join(t.TempDir(), "search")
	si, finalDir, err := buildSearchIndex(v, searchBase, "rev-final-path")
	if err != nil {
		t.Fatalf("buildSearchIndex() error = %v", err)
	}
	if finalDir != filepath.Join(searchBase, "snapshots", "rev-final-path") {
		t.Fatalf("finalDir = %q, want final snapshot dir", finalDir)
	}
	if _, err := os.Stat(filepath.Join(searchBase, "snapshots", "rev-final-path.building")); !os.IsNotExist(err) {
		t.Fatalf("building directory still exists or stat failed unexpectedly: %v", err)
	}

	writeVaultNote(t, root, "Changed.md", "# Changed\n\nwatcherterm")
	note, err := v.ReloadNote(filepath.Join(root, "Changed.md"))
	if err != nil {
		t.Fatalf("ReloadNote() error = %v", err)
	}
	doc, err := v.SearchDocument(note)
	if err != nil {
		t.Fatalf("SearchDocument() error = %v", err)
	}
	if err := si.Index(doc); err != nil {
		t.Fatalf("Index() after final move error = %v", err)
	}
	if err := si.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := search.OpenPersistent(filepath.Join(finalDir, "index"))
	if err != nil {
		t.Fatalf("OpenPersistent(final index) error = %v", err)
	}
	defer func() { _ = reopened.Close() }()
	results, err := reopened.Search("watcherterm", 10)
	if err != nil {
		t.Fatalf("Search(watcherterm) error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("watcher-style index update was not persisted in final index path")
	}
}

func TestRuntimeStatePersistentSearchCleansOldSnapshotIndexDir(t *testing.T) {
	oldDelay := oldSnapshotCloseDelay
	oldSnapshotCloseDelay = 0
	defer func() { oldSnapshotCloseDelay = oldDelay }()

	root := t.TempDir()
	worktree1 := filepath.Join(root, "wt1")
	worktree2 := filepath.Join(root, "wt2")
	link := filepath.Join(root, "current")
	writeVaultNote(t, worktree1, "First.md", "# First\n")
	writeVaultNote(t, worktree2, "Second.md", "# Second\n")
	if err := os.Symlink(worktree1, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	state, err := NewRuntimeStateWithOptions(link, RuntimeOptions{SearchIndexPath: filepath.Join(root, "search")})
	if err != nil {
		t.Fatalf("NewRuntimeStateWithOptions() error = %v", err)
	}
	oldDir := state.currentSnapshot().IndexDir
	if oldDir == "" {
		t.Fatal("persistent runtime snapshot has empty IndexDir")
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("old index dir does not exist before reload: %v", err)
	}

	if err := os.Remove(link); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	if err := os.Symlink(worktree2, link); err != nil {
		t.Fatalf("symlink new: %v", err)
	}
	if err := state.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	newDir := state.currentSnapshot().IndexDir
	if newDir == "" || newDir == oldDir {
		t.Fatalf("new IndexDir = %q, old = %q", newDir, oldDir)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(oldDir); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("old index dir still exists after cleanup delay: %s", oldDir)
}

func TestAssetHandlerServesVaultFiles(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n")
	writeVaultFile(t, root, "images/planet.png", "png-bytes")

	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/vault-assets/images/planet.png", nil)
	rr := httptest.NewRecorder()
	assetHandler(state).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("asset status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); got != "png-bytes" {
		t.Fatalf("asset body = %q, want png-bytes", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestAssetHandlerRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n")
	writeVaultFile(t, outside, "secret.png", "secret")
	if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(root, "leak.png")); err != nil {
		t.Fatalf("symlink file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-dir")); err != nil {
		t.Fatalf("symlink dir: %v", err)
	}

	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}
	for _, target := range []string{
		"/vault-assets/leak.png",
		"/vault-assets/linked-dir/secret.png",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		assetHandler(state).ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", target, rr.Code)
		}
	}
}

func TestAssetHandlerRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n")
	writeVaultFile(t, root, "images/planet.png", "png-bytes")
	writeVaultFile(t, root, ".hidden/secret.png", "secret")

	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}
	for _, target := range []string{
		"/vault-assets/../images/planet.png",
		"/vault-assets/.hidden/secret.png",
		"/vault-assets/Index.md",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		assetHandler(state).ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", target, rr.Code)
		}
	}
}

func TestValidBearerToken(t *testing.T) {
	if !validBearerToken("Bearer secret", "secret") {
		t.Fatal("valid token rejected")
	}
	if validBearerToken("Bearer wrong", "secret") {
		t.Fatal("wrong token accepted")
	}
	if validBearerToken("secret", "secret") {
		t.Fatal("missing bearer prefix accepted")
	}
	if validBearerToken("Bearer secret", "") {
		t.Fatal("empty configured token accepted")
	}
}

func TestReloadHandlerRunsBeforeReloadHook(t *testing.T) {
	root := t.TempDir()
	writeVaultNote(t, root, "Index.md", "# Index\n")
	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState() error = %v", err)
	}

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reload", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	reloadHandler(state, "", true, func() { called = true }).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("reload status = %d body=%s, want 204", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatal("beforeReload hook was not called")
	}
}

func TestValidReloadRequestAllowsLoopback(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/admin/reload", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	if !validReloadRequest(req, "", true) {
		t.Fatal("loopback reload request rejected")
	}

	req.RemoteAddr = "10.0.0.5:12345"
	if validReloadRequest(req, "", true) {
		t.Fatal("non-loopback reload request accepted")
	}
}

func writeVaultNote(t *testing.T, root, rel, body string) {
	writeVaultFile(t, root, rel, body)
}

func writeVaultFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
