package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"retro-obsidian-publish/internal/api"
	"retro-obsidian-publish/internal/watcher"
	web "retro-obsidian-publish/internal/web"
)

// Config holds the runtime settings for the Retro Obsidian Publish server.
type Config struct {
	VaultDir            string
	VaultName           string
	PageTitle           string
	Port                string
	ServeWeb            bool
	Watch               bool
	ReloadToken         string
	ReloadAllowLoopback bool
	SSRURL              string // URL of SSR sidecar (e.g. http://localhost:8089). Empty = no SSR.
	FaviconPath         string // Optional: explicit path to favicon file, overrides vault-root lookup.
}

// Run starts the API server and, when enabled, serves the bundled web SPA from
// the same process.
func Run(ctx context.Context, cfg Config) error {
	if cfg.VaultDir == "" {
		return fmt.Errorf("vault dir is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.ReloadToken == "" {
		cfg.ReloadToken, _ = os.LookupEnv("RETRO_RELOAD_TOKEN")
	}
	if _, err := net.LookupPort("tcp", cfg.Port); err != nil {
		return fmt.Errorf("invalid port %q: %w", cfg.Port, err)
	}

	log.Printf("Loading vault from %s", cfg.VaultDir)
	state, err := NewRuntimeState(cfg.VaultDir)
	if err != nil {
		return err
	}
	v, si := state.Snapshot()

	// Derive vault name and page title from directory basename if not explicitly set.
	vaultName := cfg.VaultName
	if vaultName == "" {
		vaultName = filepath.Base(cfg.VaultDir)
	}
	pageTitle := cfg.PageTitle
	if pageTitle == "" {
		pageTitle = vaultName
	}

	log.Printf("Loaded %d notes from %s", len(v.AllNotes()), state.ResolvedRoot())

	if cfg.Watch {
		fw, err := watcher.New(v, watcher.WithSearchIndex(si))
		if err != nil {
			log.Printf("warning: could not start file watcher: %v", err)
		} else {
			defer fw.Close()
		}
	} else {
		log.Printf("File watcher disabled; expecting explicit reloads")
	}

	r := mux.NewRouter()
	h := api.NewWithProvider(state, api.PublicConfig{VaultName: vaultName, PageTitle: pageTitle})
	h.Register(r)
	r.HandleFunc("/api/healthz", healthHandler(state)).Methods("GET")
	r.PathPrefix("/vault-assets/").Handler(assetHandler(state)).Methods("GET", "HEAD")
	if cfg.ReloadToken != "" || cfg.ReloadAllowLoopback {
		r.HandleFunc("/api/admin/reload", reloadHandler(state, cfg.ReloadToken, cfg.ReloadAllowLoopback)).Methods("POST")
	} else {
		log.Printf("Admin reload endpoint disabled; set RETRO_RELOAD_TOKEN or --reload-token-env, or enable --reload-allow-loopback")
	}
	if cfg.ServeWeb {
		spaHandler := web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"})

		// Favicon handler: serves from CLI override, vault root, or returns 404.
		// This must be registered before the catch-all to avoid serving index.html.
		faviconH := newFaviconHandler(state, cfg.FaviconPath, spaHandler)
		r.HandleFunc("/favicon.ico", faviconH).Methods("GET", "HEAD")
		r.HandleFunc("/favicon.svg", faviconH).Methods("GET", "HEAD")

		if cfg.SSRURL != "" {
			log.Printf("SSR sidecar proxy enabled: %s", cfg.SSRURL)

			// Serve static assets directly from the Go server, not through
			// the SSR proxy. The SSR sidecar only renders page HTML.
			// These routes must be registered before the catch-all proxy.
			r.PathPrefix("/assets/").Handler(spaHandler)
			r.PathPrefix("/__manus__/").Handler(spaHandler)
			r.PathPrefix("/fonts/").Handler(spaHandler)

			ssrProxy := newSSRProxy(cfg.SSRURL, spaHandler)
			pageHandler := newAgentPageHandler(state, api.PublicConfig{VaultName: vaultName, PageTitle: pageTitle}, ssrProxy)
			r.PathPrefix("/").Handler(pageHandler)
		} else {
			pageHandler := newAgentPageHandler(state, api.PublicConfig{VaultName: vaultName, PageTitle: pageTitle}, spaHandler)
			r.PathPrefix("/").Handler(pageHandler)
		}
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           c.Handler(r),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server listening on http://localhost:%s", cfg.Port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func healthHandler(state *RuntimeState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, _ := state.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"notes":%d,"vaultRoot":%q,"configuredRoot":%q}`,
			len(v.AllNotes()), state.ResolvedRoot(), state.ConfiguredRoot())
	}
}

func reloadHandler(state *RuntimeState, token string, allowLoopback bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validReloadRequest(r, token, allowLoopback) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if err := state.Reload(); err != nil {
			log.Printf("reload failed: %v", err)
			http.Error(w, `{"error":"reload failed"}`, http.StatusInternalServerError)
			return
		}
		v, _ := state.Snapshot()
		log.Printf("reload: loaded %d notes from %s", len(v.AllNotes()), state.ResolvedRoot())
		w.WriteHeader(http.StatusNoContent)
	}
}

func assetHandler(state *RuntimeState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/vault-assets/")
		if !validAssetPath(rel) {
			http.NotFound(w, r)
			return
		}

		root, err := os.OpenRoot(state.ResolvedRoot())
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = root.Close() }()

		assetName := filepath.FromSlash(rel)
		file, err := root.Open(assetName)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()

		info, err := file.Stat()
		if err != nil || info.IsDir() || strings.EqualFold(filepath.Ext(assetName), ".md") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})
}

func validAssetPath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.HasPrefix(rel, "/") {
		return false
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}

func validReloadRequest(r *http.Request, token string, allowLoopback bool) bool {
	if validBearerToken(r.Header.Get("Authorization"), token) {
		return true
	}
	if !allowLoopback {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validBearerToken(header, token string) bool {
	if token == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return strings.TrimPrefix(header, prefix) == token
}

// ---------------------------------------------------------------------------
// SSR reverse proxy
// ---------------------------------------------------------------------------

// newSSRProxy returns an http.Handler that reverse-proxies requests to the
// SSR sidecar. If the sidecar returns an error (connection refused, timeout,
// 5xx), the handler falls back to the spaHandler so the site stays functional
// even when the sidecar is unavailable.
func newSSRProxy(ssrURL string, spaHandler http.Handler) http.Handler {
	ssrEndpoint, err := url.Parse(ssrURL)
	if err != nil || ssrEndpoint.Scheme == "" || ssrEndpoint.Host == "" {
		log.Printf("Invalid SSR URL configuration, falling back to SPA")
		return spaHandler
	}
	if ssrEndpoint.Scheme != "http" && ssrEndpoint.Scheme != "https" {
		log.Printf("Unsupported SSR URL scheme, falling back to SPA")
		return spaHandler
	}

	proxy := httputil.NewSingleHostReverseProxy(ssrEndpoint)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = ssrEndpoint.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode >= 500 {
			return errors.New("ssr sidecar returned server error")
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("SSR proxy unavailable, falling back to SPA")
		spaHandler.ServeHTTP(w, r)
	}

	return proxy
}
