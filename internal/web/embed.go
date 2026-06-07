//go:build embed

package web

import (
	"embed"
	"io/fs"
)

// embeddedFS contains the production web bundle copied from ../../../../web/dist.
//
//go:embed embed/public
var embeddedFS embed.FS

// PublicFS exposes the bundled web assets with index.html at the filesystem root.
var PublicFS fs.FS = mustSub(embeddedFS, "embed/public")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
