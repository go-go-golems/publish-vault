package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
