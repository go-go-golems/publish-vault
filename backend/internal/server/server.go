package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"retro-obsidian-publish/backend/internal/api"
	"retro-obsidian-publish/backend/internal/watcher"
	web "retro-obsidian-publish/backend/internal/web"
)

// Config holds the runtime settings for the Retro Obsidian Publish server.
type Config struct {
	VaultDir            string
	VaultName           string
	Port                string
	ServeWeb            bool
	Watch               bool
	ReloadToken         string
	ReloadAllowLoopback bool
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
		cfg.ReloadToken = os.Getenv("RETRO_RELOAD_TOKEN")
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

	// Derive vault name from directory basename if not explicitly set.
	vaultName := cfg.VaultName
	if vaultName == "" {
		vaultName = filepath.Base(cfg.VaultDir)
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
	h := api.NewWithProvider(state, vaultName)
	h.Register(r)
	r.HandleFunc("/api/healthz", healthHandler(state)).Methods("GET")
	if cfg.ReloadToken != "" || cfg.ReloadAllowLoopback {
		r.HandleFunc("/api/admin/reload", reloadHandler(state, cfg.ReloadToken, cfg.ReloadAllowLoopback)).Methods("POST")
	} else {
		log.Printf("Admin reload endpoint disabled; set RETRO_RELOAD_TOKEN or --reload-token-env, or enable --reload-allow-loopback")
	}
	if cfg.ServeWeb {
		r.PathPrefix("/").Handler(web.NewSPAHandler(&web.SPAOptions{APIPrefix: "/api"}))
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

type healthResponse struct {
	OK             bool   `json:"ok"`
	Notes          int    `json:"notes"`
	VaultRoot      string `json:"vaultRoot"`
	ConfiguredRoot string `json:"configuredRoot"`
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
