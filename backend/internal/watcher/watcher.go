// Package watcher monitors the vault directory for file changes and triggers re-indexing.
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
)

// VaultWatcher wraps fsnotify and debounces rapid file events.
type VaultWatcher struct {
	vault   *vault.Vault
	search  *search.Index
	watcher *fsnotify.Watcher
	done    chan struct{}
}

// Option configures a VaultWatcher.
type Option func(*VaultWatcher)

// WithSearchIndex keeps the search index in sync with file watcher reload and
// remove events.
func WithSearchIndex(si *search.Index) Option {
	return func(vw *VaultWatcher) {
		vw.search = si
	}
}

// New creates and starts a VaultWatcher for the given vault.
func New(v *vault.Vault, opts ...Option) (*VaultWatcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	vw := &VaultWatcher{
		vault:   v,
		watcher: fw,
		done:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(vw)
	}

	// Watch the vault root and all subdirectories
	if err := filepath.Walk(v.Root(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return fw.Add(path)
		}
		return nil
	}); err != nil {
		fw.Close()
		return nil, err
	}

	go vw.loop()
	return vw, nil
}

// Close stops the watcher.
func (vw *VaultWatcher) Close() {
	close(vw.done)
	vw.watcher.Close()
}

// loop processes fsnotify events with debouncing.
func (vw *VaultWatcher) loop() {
	pending := map[string]fsnotify.Op{}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-vw.done:
			return

		case event, ok := <-vw.watcher.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(strings.ToLower(event.Name), ".md") {
				continue
			}
			pending[event.Name] = event.Op

		case err, ok := <-vw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)

		case <-ticker.C:
			if len(pending) == 0 {
				continue
			}
			for path, op := range pending {
				vw.apply(path, op)
			}
			pending = map[string]fsnotify.Op{}
		}
	}
}

func (vw *VaultWatcher) apply(path string, op fsnotify.Op) {
	if op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		log.Printf("vault: removing %s", path)
		slug := vw.vault.RemoveNote(path)
		if vw.search != nil {
			if err := vw.search.Delete(slug); err != nil {
				log.Printf("search: delete error for %s: %v", slug, err)
			}
		}
		return
	}

	log.Printf("vault: reloading %s", path)
	note, err := vw.vault.ReloadNote(path)
	if err != nil {
		log.Printf("vault: reload error: %v", err)
		return
	}
	if vw.search != nil {
		if err := vw.search.Index(note); err != nil {
			log.Printf("search: index error for %s: %v", note.Slug, err)
		}
	}
}
