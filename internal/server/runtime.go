package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"retro-obsidian-publish/internal/search"
	"retro-obsidian-publish/internal/vault"
)

var oldSnapshotCloseDelay = 30 * time.Second

// RuntimeOptions configures how runtime snapshots are loaded.
type RuntimeOptions struct {
	SearchIndexPath string
}

// Snapshot is an immutable runtime view of one resolved vault root and its
// matching search index. RuntimeState swaps whole snapshots so request handlers
// never see a vault from one revision and a search index from another.
type Snapshot struct {
	Revision     string
	ResolvedRoot string
	Vault        *vault.Vault
	Search       *search.Index
	IndexDir     string
	BuiltAt      time.Time
}

// RuntimeState holds the currently active vault and search index. It supports
// full reloads for git-sync style deployments where a sidecar atomically flips
// a symlink to a new checkout and then calls an admin reload endpoint.
type RuntimeState struct {
	mu              sync.RWMutex
	configuredRoot  string
	searchIndexPath string
	snapshot        *Snapshot
}

// NewRuntimeState loads a vault from configuredRoot and builds the initial
// search index. Symlinks are resolved before loading so a git-sync link such as
// /git/root/current is treated as the concrete worktree directory.
func NewRuntimeState(configuredRoot string) (*RuntimeState, error) {
	return NewRuntimeStateWithOptions(configuredRoot, RuntimeOptions{})
}

func NewRuntimeStateWithOptions(configuredRoot string, opts RuntimeOptions) (*RuntimeState, error) {
	snap, err := loadSnapshot(configuredRoot, opts.SearchIndexPath)
	if err != nil {
		return nil, err
	}
	return &RuntimeState{
		configuredRoot:  configuredRoot,
		searchIndexPath: opts.SearchIndexPath,
		snapshot:        snap,
	}, nil
}

// Snapshot returns the active vault and search index.
func (s *RuntimeState) Snapshot() (*vault.Vault, *search.Index) {
	snap := s.currentSnapshot()
	return snap.Vault, snap.Search
}

func (s *RuntimeState) currentSnapshot() *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

// ResolvedRoot returns the currently active concrete vault directory.
func (s *RuntimeState) ResolvedRoot() string {
	snap := s.currentSnapshot()
	return snap.ResolvedRoot
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
	next, err := loadSnapshot(configured, s.searchIndexPath)
	if err != nil {
		logMemoryPhase("reload_failed", "configuredRoot", configured, "error", err.Error())
		return err
	}

	s.mu.Lock()
	old := s.snapshot
	s.snapshot = next
	s.mu.Unlock()
	closeSnapshotAfter(old, oldSnapshotCloseDelay)
	logMemoryPhase("reload_swapped", "configuredRoot", configured, "resolvedRoot", next.ResolvedRoot, "revision", next.Revision, "duration", time.Since(started).String())
	return nil
}

func loadSnapshot(configuredRoot, searchIndexPath string) (*Snapshot, error) {
	started := time.Now()
	logMemoryPhase("load_start", "configuredRoot", configuredRoot, "persistentSearch", fmt.Sprint(searchIndexPath != ""))

	absRoot, err := filepath.Abs(configuredRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid vault path: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve vault path %q: %w", absRoot, err)
	}
	revision := snapshotRevision(resolvedRoot, time.Now())
	logMemoryPhase("load_resolved_root", "configuredRoot", configuredRoot, "resolvedRoot", resolvedRoot, "revision", revision)

	vaultStarted := time.Now()
	v, err := vault.New(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault: %w", err)
	}
	logMemoryPhase("load_vault_done", "resolvedRoot", resolvedRoot, "revision", revision, "notes", fmt.Sprint(v.Count()), "duration", time.Since(vaultStarted).String())

	searchStarted := time.Now()
	si, indexDir, err := buildSearchIndex(v, searchIndexPath, revision)
	if err != nil {
		return nil, fmt.Errorf("failed to build search index: %w", err)
	}
	logMemoryPhase("load_search_done", "resolvedRoot", resolvedRoot, "revision", revision, "indexDir", indexDir, "notes", fmt.Sprint(v.Count()), "duration", time.Since(searchStarted).String())
	logMemoryPhase("load_done", "configuredRoot", configuredRoot, "resolvedRoot", resolvedRoot, "revision", revision, "duration", time.Since(started).String())

	return &Snapshot{
		Revision:     revision,
		ResolvedRoot: resolvedRoot,
		Vault:        v,
		Search:       si,
		IndexDir:     indexDir,
		BuiltAt:      time.Now(),
	}, nil
}

func buildSearchIndex(v *vault.Vault, searchIndexPath, revision string) (*search.Index, string, error) {
	if searchIndexPath == "" {
		si, err := search.New(v)
		return si, "", err
	}

	base, err := filepath.Abs(searchIndexPath)
	if err != nil {
		return nil, "", err
	}
	snapshotsDir := filepath.Join(base, "snapshots")
	buildDir := filepath.Join(snapshotsDir, revision+".building")
	finalDir := filepath.Join(snapshotsDir, revision)
	indexDir := filepath.Join(buildDir, "index")
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return nil, "", err
	}
	if err := os.RemoveAll(buildDir); err != nil {
		return nil, "", err
	}
	if err := os.RemoveAll(finalDir); err != nil {
		return nil, "", err
	}

	si, err := search.NewPersistent(v, indexDir)
	if err != nil {
		_ = os.RemoveAll(buildDir)
		return nil, "", err
	}
	if err := os.Rename(buildDir, finalDir); err != nil {
		_ = si.Close()
		_ = os.RemoveAll(buildDir)
		return nil, "", err
	}
	return si, finalDir, nil
}

func closeSnapshotAfter(snap *Snapshot, delay time.Duration) {
	if snap == nil {
		return
	}
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		if snap.Search != nil {
			if err := snap.Search.Close(); err != nil {
				log.Printf("warning: close search index for revision %s: %v", snap.Revision, err)
			}
		}
		if snap.IndexDir != "" {
			if err := os.RemoveAll(snap.IndexDir); err != nil {
				log.Printf("warning: remove old search index dir %s: %v", snap.IndexDir, err)
			}
		}
	}()
}

func snapshotRevision(resolvedRoot string, now time.Time) string {
	base := strings.ToLower(filepath.Base(resolvedRoot))
	var b strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	prefix := strings.Trim(b.String(), "-")
	if prefix == "" {
		prefix = "vault"
	}
	return fmt.Sprintf("%s-%d", prefix, now.UnixNano())
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
