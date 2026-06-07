package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAOptions configures the single-page-app handler.
type SPAOptions struct {
	// APIPrefix is excluded from SPA fallback handling. Usually /api.
	APIPrefix string
}

// NewSPAHandler returns an http.Handler that serves static web assets and falls
// back to index.html for client-side routes.
func NewSPAHandler(opts *SPAOptions) http.Handler {
	return newSPAHandler(PublicFS, opts)
}

func newSPAHandler(fsys fs.FS, opts *SPAOptions) http.Handler {
	apiPrefix := "/api"
	if opts != nil && opts.APIPrefix != "" {
		apiPrefix = opts.APIPrefix
	}
	files := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, apiPrefix) {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			serveIndex(fsys, w)
			return
		}

		if fileExists(fsys, cleanPath) {
			files.ServeHTTP(w, r)
			return
		}

		serveIndex(fsys, w)
	})
}

func serveIndex(fsys fs.FS, w http.ResponseWriter) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "web bundle not found; run `retro-obsidian-publish build web` first", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func fileExists(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}
