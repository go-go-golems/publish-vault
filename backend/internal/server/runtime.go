package server

import (
	"fmt"
	"path/filepath"
	"sync"

	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
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
	configured := s.ConfiguredRoot()
	v, si, resolved, err := loadVaultAndSearch(configured)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.vault = v
	s.search = si
	s.resolvedRoot = resolved
	s.mu.Unlock()
	return nil
}

func loadVaultAndSearch(configuredRoot string) (*vault.Vault, *search.Index, string, error) {
	absRoot, err := filepath.Abs(configuredRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("invalid vault path: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolve vault path %q: %w", absRoot, err)
	}

	v, err := vault.New(resolvedRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to load vault: %w", err)
	}
	si, err := search.New(v)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to build search index: %w", err)
	}
	return v, si, resolvedRoot, nil
}
