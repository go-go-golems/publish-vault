// Command server starts the Retro Obsidian Publish backend.
//
// Usage:
//
//	server --vault /path/to/vault --port 8080
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"retro-obsidian-publish/backend/internal/api"
	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
	"retro-obsidian-publish/backend/internal/watcher"
)

func main() {
	vaultDir := flag.String("vault", "", "Path to Obsidian vault directory (required)")
	port := flag.String("port", "8080", "HTTP port to listen on")
	flag.Parse()

	if *vaultDir == "" {
		// Try VAULT_DIR env var
		if env := os.Getenv("VAULT_DIR"); env != "" {
			*vaultDir = env
		} else {
			log.Fatal("--vault flag or VAULT_DIR env var is required")
		}
	}

	absVault, err := filepath.Abs(*vaultDir)
	if err != nil {
		log.Fatalf("invalid vault path: %v", err)
	}

	log.Printf("Loading vault from %s", absVault)
	v, err := vault.New(absVault)
	if err != nil {
		log.Fatalf("failed to load vault: %v", err)
	}
	log.Printf("Loaded %d notes", len(v.AllNotes()))

	// Build search index
	si, err := search.New(v)
	if err != nil {
		log.Fatalf("failed to build search index: %v", err)
	}

	// Start file watcher
	fw, err := watcher.New(v)
	if err != nil {
		log.Printf("warning: could not start file watcher: %v", err)
	} else {
		defer fw.Close()
	}

	// Set up router
	r := mux.NewRouter()
	h := api.New(v, si)
	h.Register(r)

	// CORS — allow the Vite dev server
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})

	addr := ":" + *port
	log.Printf("Server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, c.Handler(r)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
