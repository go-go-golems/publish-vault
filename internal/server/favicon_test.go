package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	web "github.com/go-go-golems/publish-vault/internal/web"
)

func TestFaviconHandler_ServesFromVaultRoot(t *testing.T) {
	vaultDir := t.TempDir()
	// Write a fake favicon.ico into the vault root
	icoContent := []byte("FAKE_ICO_DATA")
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.ico"), icoContent, 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, "", nil)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", resp.StatusCode, w.Body.String())
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "image/x-icon" && ct != "image/vnd.microsoft.icon" {
		t.Fatalf("expected image/x-icon content-type, got %q", ct)
	}
	if w.Body.String() != "FAKE_ICO_DATA" {
		t.Fatalf("expected favicon content, got %q", w.Body.String())
	}
}

func TestFaviconHandler_ServesSVGFromVaultRoot(t *testing.T) {
	vaultDir := t.TempDir()
	svgContent := []byte("<svg></svg>")
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.svg"), svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, "", nil)

	req := httptest.NewRequest("GET", "/favicon.svg", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Fatalf("expected image/svg+xml content-type, got %q", ct)
	}
}

func TestFaviconHandler_Returns404WhenMissing(t *testing.T) {
	vaultDir := t.TempDir()
	// No favicon files in vault

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, "", nil)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain content-type, got %q", ct)
	}
	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty 404 body")
	}
	// Must NOT be HTML
	if len(body) > 0 && body[0] == '<' {
		t.Fatalf("404 body should not be HTML, got: %s", body)
	}
}

func TestFaviconHandler_CLIOverrideTakesPrecedence(t *testing.T) {
	vaultDir := t.TempDir()
	// Put a favicon in vault root
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.ico"), []byte("VAULT_ICO"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an override file in a different location
	overrideDir := t.TempDir()
	overrideContent := []byte("OVERRIDE_ICO")
	overridePath := filepath.Join(overrideDir, "custom-favicon.ico")
	if err := os.WriteFile(overridePath, overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, overridePath, nil)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if w.Body.String() != "OVERRIDE_ICO" {
		t.Fatalf("expected override content, got %q", w.Body.String())
	}
}

func TestFaviconHandler_CLIOverrideMissingFallsThroughToVault(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.ico"), []byte("VAULT_ICO"), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	// Point --favicon to a nonexistent file — should fall through to vault
	handler := newFaviconHandler(state, "/nonexistent/path/favicon.ico", nil)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (fallthrough to vault), got %d", resp.StatusCode)
	}
	if w.Body.String() != "VAULT_ICO" {
		t.Fatalf("expected vault content, got %q", w.Body.String())
	}
}

func TestFaviconHandler_CLIOverrideOnlyMatchesRequestedExtension(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.svg"), []byte("<svg>VAULT</svg>"), 0644); err != nil {
		t.Fatal(err)
	}

	overrideDir := t.TempDir()
	overridePath := filepath.Join(overrideDir, "custom-favicon.ico")
	if err := os.WriteFile(overridePath, []byte("OVERRIDE_ICO"), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, overridePath, nil)

	req := httptest.NewRequest("GET", "/favicon.svg", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if w.Body.String() != "<svg>VAULT</svg>" {
		t.Fatalf("expected vault SVG content, got %q", w.Body.String())
	}
}

func TestFaviconHandler_ServesBundledFallbackWhenPresent(t *testing.T) {
	oldPublicFS := web.PublicFS
	web.PublicFS = fstest.MapFS{
		"favicon.ico": &fstest.MapFile{Data: []byte("BUNDLED_ICO")},
	}
	defer func() { web.PublicFS = oldPublicFS }()

	vaultDir := t.TempDir()
	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("BUNDLED_ICO"))
	})
	handler := newFaviconHandler(state, "", fallback)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if w.Body.String() != "BUNDLED_ICO" {
		t.Fatalf("expected bundled fallback content, got %q", w.Body.String())
	}
}

func TestFaviconHandler_SetsCacheControl(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.ico"), []byte("DATA"), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, "", nil)

	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	cc := resp.Header.Get("Cache-Control")
	if cc != "public, max-age=3600" {
		t.Fatalf("expected Cache-Control public, max-age=3600, got %q", cc)
	}
}

func TestFaviconHandler_HeadRequest(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "favicon.ico"), []byte("DATA"), 0644); err != nil {
		t.Fatal(err)
	}

	state, err := NewRuntimeState(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	handler := newFaviconHandler(state, "", nil)

	req := httptest.NewRequest("HEAD", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for HEAD, got %d", resp.StatusCode)
	}
}
