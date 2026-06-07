package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"retro-obsidian-publish/internal/api"
)

func newTestRuntimeState(t *testing.T) *RuntimeState {
	t.Helper()
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	write("Index.md", "---\ntags: [home]\n---\n# Index\n\nWelcome to the test vault.\n")
	write("Notes/Second.md", "# Second Note\n\nLinks to [[Index]].\n")
	state, err := NewRuntimeState(root)
	if err != nil {
		t.Fatalf("NewRuntimeState: %v", err)
	}
	return state
}

func TestAgentPageHandlerDiscoveryEndpoints(t *testing.T) {
	state := newTestRuntimeState(t)
	h := newAgentPageHandler(state, api.PublicConfig{VaultName: "test-vault", PageTitle: "Test Vault"}, http.NotFoundHandler())

	cases := []struct {
		path        string
		contentType string
		contains    []string
	}{
		{path: "/AGENTS.md", contentType: "text/markdown", contains: []string{"## Installation", "## Configuration", "## Usage", "## Glossary"}},
		{path: "/llms.txt", contentType: "text/plain", contains: []string{"# Test Vault", "AGENTS.md", "sitemap.md"}},
		{path: "/sitemap.md", contentType: "text/markdown", contains: []string{"# Sitemap", "## Notes", "/note/index"}},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d", rr.Code)
			}
			if !strings.Contains(rr.Header().Get("Content-Type"), tc.contentType) {
				t.Fatalf("content-type = %q, want contains %q", rr.Header().Get("Content-Type"), tc.contentType)
			}
			body := rr.Body.String()
			for _, want := range tc.contains {
				if !strings.Contains(body, want) {
					t.Fatalf("body missing %q:\n%s", want, body)
				}
			}
		})
	}
}

func TestAgentPageHandlerSitemapXML(t *testing.T) {
	state := newTestRuntimeState(t)
	h := newAgentPageHandler(state, api.PublicConfig{VaultName: "test-vault"}, http.NotFoundHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/xml") {
		t.Fatalf("content-type = %q", rr.Header().Get("Content-Type"))
	}
	var parsed struct {
		XMLName xml.Name `xml:"urlset"`
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("xml unmarshal: %v\n%s", err, rr.Body.String())
	}
	if parsed.XMLName.Local != "urlset" || len(parsed.URLs) == 0 {
		t.Fatalf("unexpected sitemap: name=%s urls=%d", parsed.XMLName.Local, len(parsed.URLs))
	}
}

func TestAgentPageHandlerMarkdownMirrors(t *testing.T) {
	state := newTestRuntimeState(t)
	h := newAgentPageHandler(state, api.PublicConfig{VaultName: "test-vault", PageTitle: "Test Vault"}, http.NotFoundHandler())

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "home mirror", path: "/index.md"},
		{name: "note mirror", path: "/note/index.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d", rr.Code)
			}
			body := rr.Body.String()
			for _, want := range []string{"---", "title:", "description:", "doc_version:", "last_updated:", "## Sitemap"} {
				if !strings.Contains(body, want) {
					t.Fatalf("body missing %q:\n%s", want, body)
				}
			}
			if !strings.Contains(rr.Header().Get("Link"), `rel="canonical"`) {
				t.Fatalf("missing canonical Link header: %q", rr.Header().Get("Link"))
			}
		})
	}
}

func TestAgentPageHandlerMarkdownContentNegotiation(t *testing.T) {
	state := newTestRuntimeState(t)
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		_, _ = w.Write([]byte("html fallback"))
	})
	h := newAgentPageHandler(state, api.PublicConfig{VaultName: "test-vault", PageTitle: "Test Vault"}, fallback)

	req := httptest.NewRequest(http.MethodGet, "/note/index", nil)
	req.Header.Set("Accept", "text/markdown")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if fallbackCalled {
		t.Fatal("fallback should not be called for markdown negotiation")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/markdown") {
		t.Fatalf("content-type = %q", rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(rr.Body.String(), "# Index") {
		t.Fatalf("expected note markdown, got:\n%s", rr.Body.String())
	}
}
