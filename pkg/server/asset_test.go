package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestAssetHandler_404OnIgnored verifies that a non-.md asset located under a
// .vault-ignore-excluded directory is not served (404), closing the static-asset
// loophole that bypasses the notes index.
func TestAssetHandler_404OnIgnored(t *testing.T) {
	vaultDir := t.TempDir()

	// A published image.
	publishedImg := filepath.Join(vaultDir, "img", "public.png")
	if err := os.MkdirAll(filepath.Dir(publishedImg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publishedImg, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An image inside an ignored directory.
	ignoredDir := filepath.Join(vaultDir, "Secrets")
	if err := os.MkdirAll(ignoredDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "secret.png"), []byte("SECRET_PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Ignore the whole Secrets folder.
	if err := os.WriteFile(filepath.Join(vaultDir, ".vault-ignore"), []byte("/Secrets/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	h := assetHandler(state)

	// Ignored asset -> 404.
	req := httptest.NewRequest(http.MethodGet, "/vault-assets/Secrets/secret.png", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("ignored asset: expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() == "SECRET_PNG" {
		t.Fatalf("ignored asset body should be empty, got %q", rr.Body.String())
	}

	// Published asset -> 200.
	req = httptest.NewRequest(http.MethodGet, "/vault-assets/img/public.png", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("published asset: expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "PNG" {
		t.Fatalf("published asset body = %q, want %q", rr.Body.String(), "PNG")
	}
}

// TestAssetHandler_NoIgnoreFileServesAll confirms the handler is unchanged
// when no .vault-ignore file is present.
func TestAssetHandler_NoIgnoreFileServesAll(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vaultDir, "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "img", "a.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	h := assetHandler(state)

	req := httptest.NewRequest(http.MethodGet, "/vault-assets/img/a.png", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 without ignore file, got %d", rr.Code)
	}
}

// TestAssetHandler_ServesReIncludedAsset confirms the asset handler agrees with
// the permissive matcher: a "!" re-include under an excluded directory is served,
// while a sibling asset stays 404. This keeps the static-asset path consistent
// with the note index and the raw-source endpoint.
func TestAssetHandler_ServesReIncludedAsset(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vaultDir, "Secrets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "Secrets", "secret.png"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "Secrets", "public.png"), []byte("PUBLIC"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, ".vault-ignore"), []byte("/Secrets/\n!Secrets/public.png\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	h := assetHandler(state)

	// Re-included asset -> 200.
	req := httptest.NewRequest(http.MethodGet, "/vault-assets/Secrets/public.png", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("re-included asset: expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "PUBLIC" {
		t.Errorf("re-included asset body = %q, want PUBLIC", rr.Body.String())
	}

	// Excluded sibling -> 404.
	req = httptest.NewRequest(http.MethodGet, "/vault-assets/Secrets/secret.png", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("excluded sibling asset: expected 404, got %d", rr.Code)
	}
}
