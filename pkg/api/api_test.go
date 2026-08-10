package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func TestRoutesSmoke(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "Index.md", `---
title: Index
tags: [home]
---
# Index

Welcome to [[Second Note]].
`)
	writeNote(t, root, "Second Note.md", `# Second Note

Searchable unique phrase.
`)

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}

	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)

	cases := []string{
		"/api/config",
		"/api/notes",
		"/api/notes/index",
		"/api/tree",
		"/api/search?q=unique",
		"/api/tags",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d body=%s", path, rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("GET %s Content-Type = %q, want JSON", path, ct)
		}
	}
}

func TestNoteRawReadsMarkdownFromDisk(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "Index.md", "# Index\n\nraw body")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)

	req := httptest.NewRequest(http.MethodGet, "/api/notes/index/raw", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET raw status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); got != "# Index\n\nraw body" {
		t.Fatalf("raw body = %q", got)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/markdown") {
		t.Fatalf("raw Content-Type = %q, want text/markdown", ct)
	}
}

func TestNoteRawReturnsNotFoundWhenSourceFileIsGone(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "Index.md")
	writeNote(t, root, "Index.md", "# Index\n")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	if err := os.Remove(notePath); err != nil {
		t.Fatalf("remove note source: %v", err)
	}
	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)

	req := httptest.NewRequest(http.MethodGet, "/api/notes/index/raw", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET raw missing source status = %d body=%s, want 404", rr.Code, rr.Body.String())
	}
}

func TestGetNoteOmitsRawMarkdown(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "Index.md", "# Index\n\nraw body")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)

	req := httptest.NewRequest(http.MethodGet, "/api/notes/index", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET note status = %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode note: %v", err)
	}
	if _, ok := body["rawMarkdown"]; ok {
		t.Fatalf("note response unexpectedly contains rawMarkdown: %s", rr.Body.String())
	}
}

func TestConfigIncludesPageTitle(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "Index.md", "# Index\n")
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}

	r := mux.NewRouter()
	NewWithProvider(staticProvider{vault: v, search: si}, PublicConfig{VaultName: "PARC", PageTitle: "PARC Notes"}).Register(r)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/config status = %d body=%s", rr.Code, rr.Body.String())
	}
	var cfg SiteConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.VaultName != "PARC" || cfg.PageTitle != "PARC Notes" || cfg.Notes != 1 {
		t.Fatalf("config = %#v, want vaultName/pageTitle/notes", cfg)
	}
}

func writeNote(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestGetNoteRouteShapes pins the status code for each URL shape a reader or
// the SSR sidecar can produce for the same note (PV-SLUG-020 design doc 12.3).
// The %2F-encoded row is the SSR contract: server.mjs builds note URLs with
// encodeURIComponent, so the whole slug arrives as one percent-encoded segment.
func TestGetNoteRouteShapes(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "Transcripts/2026/Designing RAG/The_Algebra.md", "# Algebra\n\nBody.\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New() error = %v", err)
	}
	si, err := search.New(v)
	if err != nil {
		t.Fatalf("search.New() error = %v", err)
	}
	r := mux.NewRouter()
	New(v, si, "TestVault").Register(r)

	const canonical = "transcripts/2026/designing-rag/the_algebra"
	cases := []struct {
		name         string
		path         string
		wantStatus   int
		wantLocation string
	}{
		{"raw slashes", "/api/notes/" + canonical, http.StatusOK, ""},
		{"percent encoded", "/api/notes/" + url.PathEscape(canonical), http.StatusOK, ""},
		{"trailing slash", "/api/notes/" + canonical + "/", http.StatusPermanentRedirect, "/api/notes/" + canonical},
		// gorilla/mux cleans "//" and redirects (301) before the handler runs, so
		// this shape never reaches CanonicalSlug. Pinned so a future router change
		// that stops cleaning is caught rather than silently 404ing.
		{"doubled slash", "/api/notes/transcripts/2026//designing-rag/the_algebra", http.StatusMovedPermanently, "/api/notes/" + canonical},
		{"uppercase", "/api/notes/TRANSCRIPTS/2026/DESIGNING-RAG/THE_ALGEBRA", http.StatusPermanentRedirect, "/api/notes/" + canonical},
		{"genuinely missing", "/api/notes/no/such/note", http.StatusNotFound, ""},
		{"md suffix", "/api/notes/" + canonical + ".md", http.StatusNotFound, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("GET %s = %d, want %d (body %q)", tc.path, rr.Code, tc.wantStatus, rr.Body.String())
			}
			if tc.wantLocation == "" {
				return
			}
			if got := rr.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("GET %s Location = %q, want %q", tc.path, got, tc.wantLocation)
			}
			// Following the redirect must land on the note, not on another
			// redirect: a non-idempotent normalizer would loop here.
			follow := httptest.NewRequest(http.MethodGet, rr.Header().Get("Location"), nil)
			followRR := httptest.NewRecorder()
			r.ServeHTTP(followRR, follow)
			if followRR.Code != http.StatusOK {
				t.Errorf("following Location %q = %d, want 200", rr.Header().Get("Location"), followRR.Code)
			}
		})
	}
}

// TestSafeRedirectSlug pins the guard that keeps the canonical-slug redirect
// same-origin. Slugify's alphabet makes these cases unreachable today; the
// guard exists so widening that alphabet cannot silently turn the redirect
// into an open redirect.
func TestSafeRedirectSlug(t *testing.T) {
	safe := []string{"a", "a/b", "notes/2026/08/09/thing", "a-b_c"}
	unsafe := []string{
		"",           // empty target
		"/evil.com",  // becomes //evil.com after the prefix: protocol-relative
		"//evil.com", //
		`\evil.com`,  // some agents normalise backslashes to slashes
		"../../etc",  // climbs out of /api/notes/
		"a/../../b",  //
	}
	for _, s := range safe {
		if !safeRedirectSlug(s) {
			t.Errorf("safeRedirectSlug(%q) = false, want true", s)
		}
	}
	for _, s := range unsafe {
		if safeRedirectSlug(s) {
			t.Errorf("safeRedirectSlug(%q) = true, want false", s)
		}
	}
}
