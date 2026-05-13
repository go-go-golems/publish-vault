package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandlerServesIndexForRootAndClientRoutes(t *testing.T) {
	h := newSPAHandler(testFS(), &SPAOptions{APIPrefix: "/api"})

	for _, path := range []string{"/", "/note/index", "/search"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "app shell") {
			t.Fatalf("GET %s body = %q", path, rr.Body.String())
		}
	}
}

func TestSPAHandlerServesAssets(t *testing.T) {
	h := newSPAHandler(testFS(), nil)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("asset status = %d", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "console.log('ok')" {
		t.Fatalf("asset body = %q", got)
	}
}

func TestSPAHandlerDoesNotSwallowAPIRoutes(t *testing.T) {
	h := newSPAHandler(testFS(), &SPAOptions{APIPrefix: "/api"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/notes", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("/api/notes status = %d, want 404 from SPA handler", rr.Code)
	}
}

func testFS() fs.FS {
	return fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>app shell</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}
}
