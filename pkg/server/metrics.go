package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// startPrivateMetrics serves only the bounded Prometheus projection on a
// separate operator-configured listener. It is never mounted on the public mux.
func startPrivateMetrics(address string, handler http.Handler) (func(context.Context) error, string, error) {
	if address == "" {
		return func(context.Context) error { return nil }, "", nil
	}
	if handler == nil {
		return nil, "", fmt.Errorf("private metrics handler is required")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, "", fmt.Errorf("listen for private metrics: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handler)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() {
		log.Printf("private measure metrics listening on http://%s/metrics", listener.Addr())
		done <- server.Serve(listener)
	}()
	shutdown := func(ctx context.Context) error {
		shutdownErr := server.Shutdown(ctx)
		serveErr := <-done
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(shutdownErr, serveErr)
	}
	return shutdown, listener.Addr().String(), nil
}
