package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	web "github.com/go-go-golems/publish-vault/internal/web"
)

const (
	faviconICO = "favicon.ico"
	faviconSVG = "favicon.svg"
)

// newFaviconHandler returns an HTTP handler that serves favicon.ico or
// favicon.svg from (in order): a CLI-configured override path with a matching
// extension, the vault root directory, or the bundled web filesystem. When none
// of these sources contain the requested file, the handler returns a clean 404
// with text/plain body instead of falling through to the SPA catch-all (which
// would serve index.html).
func newFaviconHandler(state *RuntimeState, faviconOverride string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename, ok := faviconNameFromRequest(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		// 1. CLI override (--favicon flag). Only serve an override when its
		// extension matches the requested favicon URL. This prevents serving ICO
		// bytes from /favicon.svg or SVG bytes from /favicon.ico.
		if faviconOverride != "" && faviconOverrideMatchesRequest(faviconOverride, filename) {
			if serveConfiguredFavicon(w, r, faviconOverride) {
				return
			}
			log.Printf("warning: --favicon path %q does not exist", faviconOverride)
		}

		// 2. Vault root lookup. filename is one of two fixed constants from
		// faviconNameFromRequest, not an arbitrary path component.
		if serveVaultFavicon(w, r, state, filename) {
			return
		}

		// 3. Bundled web filesystem fallback. Check for the exact file before
		// delegating to the SPA/static handler so missing favicons do not fall
		// through to index.html.
		if fallback != nil && web.PublicFileExists(filename) {
			fallback.ServeHTTP(w, r)
			return
		}

		// 4. Not found
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "favicon not found")
	}
}

func faviconNameFromRequest(requestPath string) (string, bool) {
	switch requestPath {
	case "/" + faviconICO:
		return faviconICO, true
	case "/" + faviconSVG:
		return faviconSVG, true
	default:
		return "", false
	}
}

func faviconOverrideMatchesRequest(overridePath, requestedName string) bool {
	return strings.EqualFold(filepath.Ext(overridePath), filepath.Ext(requestedName))
}

// serveConfiguredFavicon serves an operator-configured favicon path. This path
// comes from trusted process configuration (--favicon), not from the HTTP URL.
func serveConfiguredFavicon(w http.ResponseWriter, r *http.Request, configuredPath string) bool {
	info, err := os.Stat(configuredPath)
	if err != nil || info.IsDir() {
		return false
	}

	file, err := os.Open(configuredPath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	serveFaviconContent(w, r, info.Name(), info.ModTime(), file)
	return true
}

// serveVaultFavicon serves a favicon from the vault root. The filename argument
// must come from faviconNameFromRequest so it is restricted to favicon.ico or
// favicon.svg and cannot contain separators or traversal components.
func serveVaultFavicon(w http.ResponseWriter, r *http.Request, state *RuntimeState, filename string) bool {
	root, err := os.OpenRoot(state.ResolvedRoot())
	if err != nil {
		return false
	}
	defer func() { _ = root.Close() }()

	file, err := root.Open(filename)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	serveFaviconContent(w, r, info.Name(), info.ModTime(), file)
	return true
}

func serveFaviconContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, name, modTime, content)
}
