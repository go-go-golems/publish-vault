package server

import (
	"fmt"
	"log"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"

	"retro-obsidian-publish/internal/search"
	"retro-obsidian-publish/internal/vault"
)

// RuntimeState holds the currently active vault and search index. It supports
// full reloads for git-sync style deployments where a sidecar atomically flips
// a symlink to a new checkout and then calls an admin reload endpoint.
type RuntimeState struct {
	mu             sync.RWMutex
	configuredRoot string
	resolvedRoot   string
	vault          *vault.Vault
	search         *search.Index
}

// NewRuntimeState loads a vault from configuredRoot and builds the initial
// search index. Symlinks are resolved before loading so a git-sync link such as
// /git/root/current is treated as the concrete worktree directory.
func NewRuntimeState(configuredRoot string) (*RuntimeState, error) {
	v, si, resolved, err := loadVaultAndSearch(configuredRoot)
	if err != nil {
		return nil, err
	}
	return &RuntimeState{
		configuredRoot: configuredRoot,
		resolvedRoot:   resolved,
		vault:          v,
		search:         si,
	}, nil
}

// Snapshot returns the active vault and search index.
func (s *RuntimeState) Snapshot() (*vault.Vault, *search.Index) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vault, s.search
}

// ResolvedRoot returns the currently active concrete vault directory.
func (s *RuntimeState) ResolvedRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolvedRoot
}

// ConfiguredRoot returns the stable configured vault path.
func (s *RuntimeState) ConfiguredRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configuredRoot
}

// Reload builds a new vault and search index, then atomically swaps them into
// service. If loading or indexing fails, the previous state remains active.
func (s *RuntimeState) Reload() error {
	started := time.Now()
	configured := s.ConfiguredRoot()
	logMemoryPhase("reload_start", "configuredRoot", configured)
	v, si, resolved, err := loadVaultAndSearch(configured)
	if err != nil {
		logMemoryPhase("reload_failed", "configuredRoot", configured, "error", err.Error())
		return err
	}

	s.mu.Lock()
	s.vault = v
	s.search = si
	s.resolvedRoot = resolved
	s.mu.Unlock()
	logMemoryPhase("reload_swapped", "configuredRoot", configured, "resolvedRoot", resolved, "duration", time.Since(started).String())
	return nil
}

func loadVaultAndSearch(configuredRoot string) (*vault.Vault, *search.Index, string, error) {
	started := time.Now()
	logMemoryPhase("load_start", "configuredRoot", configuredRoot)

	absRoot, err := filepath.Abs(configuredRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("invalid vault path: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolve vault path %q: %w", absRoot, err)
	}
	logMemoryPhase("load_resolved_root", "configuredRoot", configuredRoot, "resolvedRoot", resolvedRoot)

	vaultStarted := time.Now()
	v, err := vault.New(resolvedRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to load vault: %w", err)
	}
	logMemoryPhase("load_vault_done", "resolvedRoot", resolvedRoot, "notes", fmt.Sprint(v.Count()), "duration", time.Since(vaultStarted).String())

	searchStarted := time.Now()
	si, err := search.New(v)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to build search index: %w", err)
	}
	logMemoryPhase("load_search_done", "resolvedRoot", resolvedRoot, "notes", fmt.Sprint(v.Count()), "duration", time.Since(searchStarted).String())
	logMemoryPhase("load_done", "configuredRoot", configuredRoot, "resolvedRoot", resolvedRoot, "duration", time.Since(started).String())
	return v, si, resolvedRoot, nil
}

type memoryStats struct {
	HeapAllocBytes uint64 `json:"heapAllocBytes"`
	HeapSysBytes   uint64 `json:"heapSysBytes"`
	HeapInuseBytes uint64 `json:"heapInuseBytes"`
	NextGCBytes    uint64 `json:"nextGCBytes"`
	NumGC          uint32 `json:"numGC"`
}

func currentMemoryStats() memoryStats {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	return memoryStats{
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		HeapInuseBytes: m.HeapInuse,
		NextGCBytes:    m.NextGC,
		NumGC:          m.NumGC,
	}
}

func logMemoryPhase(phase string, keyValues ...string) {
	m := currentMemoryStats()
	fields := ""
	for i := 0; i+1 < len(keyValues); i += 2 {
		fields += fmt.Sprintf(" %s=%q", keyValues[i], keyValues[i+1])
	}
	log.Printf("memory phase=%s heapAllocBytes=%d heapSysBytes=%d heapInuseBytes=%d nextGCBytes=%d numGC=%d%s",
		phase, m.HeapAllocBytes, m.HeapSysBytes, m.HeapInuseBytes, m.NextGCBytes, m.NumGC, fields)
}
