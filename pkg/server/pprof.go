package server

import (
	"log"
	"net/http"
	"net/http/pprof"
	"time"
)

// startPprof serves net/http/pprof on its own listener when addr is non-empty.
//
// Diagnosing the OOMKill in PV-MEMORY-019 required rebuilding the vault under a
// local harness because the running process exposed no way to ask where its
// memory had gone. A heap profile answers that in one command:
//
//	go tool pprof -top http://127.0.0.1:6060/debug/pprof/heap
//
// It is a SEPARATE listener, never routes registered on the public mux, so
// binding it to loopback or a private interface keeps profiles off the internet
// regardless of how the main port is exposed. The default is off.
func startPprof(addr string) {
	if addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("pprof listening on http://%s/debug/pprof/", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("warning: pprof listener stopped: %v", err)
		}
	}()
}
