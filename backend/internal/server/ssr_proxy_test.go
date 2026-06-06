package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSRProxyProxiesPagesToSidecar(t *testing.T) {
	var ssrPaths []string
	ssr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ssrPaths = append(ssrPaths, r.URL.String())
		_, _ = w.Write([]byte("ssr page"))
	}))
	defer ssr.Close()

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("spa fallback"))
	})

	proxy := newSSRProxy(ssr.URL, spaHandler)

	// Page request should go to SSR sidecar
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/note/my-note", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Body.String() != "ssr page" {
		t.Fatalf("expected SSR response, got %q", rr.Body.String())
	}
	if len(ssrPaths) != 1 || ssrPaths[0] != "/note/my-note" {
		t.Fatalf("expected one SSR request for /note/my-note, got %v", ssrPaths)
	}
}

func TestSSRProxyFallsBackOnSidecarFailure(t *testing.T) {
	// Create a sidecar that returns 500
	ssr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ssr.Close()

	spaCalled := false
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spaCalled = true
		_, _ = w.Write([]byte("spa fallback"))
	})

	proxy := newSSRProxy(ssr.URL, spaHandler)

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/note/some-note", nil))
	if !spaCalled {
		t.Fatal("expected SPA fallback to be called on SSR 500")
	}
	if rr.Body.String() != "spa fallback" {
		t.Fatalf("expected SPA fallback response, got %q", rr.Body.String())
	}
}

func TestSSRProxyFallsBackOnSidecarUnavailable(t *testing.T) {
	// Use a port that's not listening
	spaCalled := false
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spaCalled = true
		_, _ = w.Write([]byte("spa fallback"))
	})

	proxy := newSSRProxy("http://127.0.0.1:1", spaHandler)

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if !spaCalled {
		t.Fatal("expected SPA fallback when sidecar is unavailable")
	}
}

func TestSSRProxyInvalidURLFallsBackToSPA(t *testing.T) {
	spaCalled := false
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spaCalled = true
		_, _ = w.Write([]byte("spa fallback"))
	})

	proxy := newSSRProxy("://invalid-url", spaHandler)

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if !spaCalled {
		t.Fatal("expected SPA fallback for invalid SSR URL")
	}
}

func TestSSRProxyForwardsHeaders(t *testing.T) {
	var receivedAccept string
	ssr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte("ok"))
	}))
	defer ssr.Close()

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	proxy := newSSRProxy(ssr.URL, spaHandler)

	req := httptest.NewRequest(http.MethodGet, "/note/test", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if receivedAccept != "text/html" {
		t.Fatalf("expected Accept header forwarded, got %q", receivedAccept)
	}
}
