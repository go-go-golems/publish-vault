package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
