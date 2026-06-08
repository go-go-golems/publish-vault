package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// newFaviconHandler returns an HTTP handler that serves favicon.ico or
// favicon.svg from (in order): a CLI-configured override path, the vault root
// directory, or the embedded web bundle. When none of these sources contain the
// requested file, the handler returns a clean 404 with text/plain body instead
// of falling through to the SPA catch-all (which would serve index.html).
func newFaviconHandler(state *RuntimeState, faviconOverride string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path) // "favicon.ico" or "favicon.svg"

		// 1. CLI override (--favicon flag)
		if faviconOverride != "" {
			if serveFileIfExists(w, r, faviconOverride) {
				return
			}
			log.Printf("warning: --favicon path %q does not exist", faviconOverride)
		}

		// 2. Vault root lookup
		vaultPath := filepath.Join(state.ResolvedRoot(), filename)
		if serveFileIfExists(w, r, vaultPath) {
			return
		}

		// 3. Embedded web bundle — delegate to the SPA handler's FS.
		//    We cannot call fallback directly because it would serve
		//    index.html for missing files. Instead, we let it through
		//    only if the embedded FS actually contains the file.
		//    For now, fall through to 404; a future change can add
		//    embedded default favicon support.

		// 4. Not found
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "favicon not found")
	}
}

// serveFileIfExists serves a file if it exists, is regular, and is readable.
// Returns true if the file was served, false otherwise.
func serveFileIfExists(w http.ResponseWriter, r *http.Request, absPath string) bool {
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false
	}

	file, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return true
}
