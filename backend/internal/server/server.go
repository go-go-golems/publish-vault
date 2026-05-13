package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"retro-obsidian-publish/backend/internal/api"
	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
	"retro-obsidian-publish/backend/internal/watcher"
	web "retro-obsidian-publish/backend/internal/web"
)

// Config holds the runtime settings for the Retro Obsidian Publish server.
type Config struct {
	VaultDir string
	Port     string
	ServeWeb bool
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
	if _, err := net.LookupPort("tcp", cfg.Port); err != nil {
		return fmt.Errorf("invalid port %q: %w", cfg.Port, err)
	}

	absVault, err := filepath.Abs(cfg.VaultDir)
	if err != nil {
		return fmt.Errorf("invalid vault path: %w", err)
	}

	log.Printf("Loading vault from %s", absVault)
	v, err := vault.New(absVault)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}
	log.Printf("Loaded %d notes", len(v.AllNotes()))

	si, err := search.New(v)
	if err != nil {
		return fmt.Errorf("failed to build search index: %w", err)
	}

	fw, err := watcher.New(v, watcher.WithSearchIndex(si))
	if err != nil {
		log.Printf("warning: could not start file watcher: %v", err)
	} else {
		defer fw.Close()
	}

	r := mux.NewRouter()
	h := api.New(v, si)
	h.Register(r)
	if cfg.ServeWeb {
		r.PathPrefix("/").Handler(web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"}))
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
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
