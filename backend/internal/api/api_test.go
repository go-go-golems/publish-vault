package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
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
	New(v, si).Register(r)

	cases := []string{
		"/api/notes",
		"/api/notes/index",
		"/api/tree",
		"/api/search?q=unique",
		"/api/tags",
		"/api/graph",
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
