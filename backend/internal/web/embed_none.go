//go:build !embed

package web

import (
	"io/fs"
	"os"
	"path/filepath"
)

// PublicFS serves the built web bundle from disk in development builds.
var PublicFS fs.FS = os.DirFS(filepath.Join(findRepoRoot(), "web", "dist"))

func findRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if exists(filepath.Join(dir, "backend", "go.mod")) && exists(filepath.Join(dir, "web", "package.json")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
