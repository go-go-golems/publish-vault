// Package validate checks the published view of a vault without changing it.
package validate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vault"
	"github.com/go-go-golems/publish-vault/pkg/vaultconfig"
)

type Diagnostic struct {
	Rule, Severity, Path, Message, Target string
	Line, Column                          int
}

// Run loads one published view and returns diagnostics in stable order.
// It does not inspect frontmatter references or fetch external resources.
func Run(ctx context.Context, root, configPath string) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve vault root: %w", err)
	}
	root, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve vault root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat vault root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault root is not a directory: %s", root)
	}
	if configPath == "" {
		configPath = filepath.Join(root, vaultconfig.DefaultConfigPath)
	}
	cfg, err := vaultconfig.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load vault config: %w", err)
	}
	if _, err = vaultconfig.NewMatcher(cfg); err != nil {
		return nil, fmt.Errorf("compile vault config: %w", err)
	}
	v, err := vault.New(root, vault.WithConfig(cfg), vault.WithStrictLoad())
	if err != nil {
		return nil, fmt.Errorf("load vault for validation: %w", err)
	}
	notes := v.AllNotes()
	sort.Slice(notes, func(i, j int) bool { return notes[i].Path < notes[j].Path })
	var out []Diagnostic
	for _, note := range notes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src, ok := v.SnapshotSource(note.Slug)
		if !ok {
			return nil, fmt.Errorf("loaded note disappeared: %s", note.Path)
		}
		_, doc, err := parser.ParseDocument(src)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", note.Path, err)
		}
		starts := []int{0}
		for i, c := range src {
			if c == '\n' {
				starts = append(starts, i+1)
			}
		}
		for _, ref := range doc.WikiOccurrences() {
			rule, message := "", ""
			if ref.Embed && ref.Image {
				if _, ok := v.ResolveAssetEmbed(ref.Target); !ok {
					rule, message = "WIKI_IMAGE_MISSING", "wiki image not found in published vault"
				}
			} else {
				slug, found := note.Slug, true
				if ref.Target != "" {
					slug, found = v.ResolveWikiLink(ref.Target)
				}
				if !found {
					rule, message = "WIKI_TARGET_MISSING", "wiki target not found in published vault"
				} else if ref.Heading != "" {
					if _, ok := v.ResolveHeading(slug, ref.Heading); !ok {
						rule, message = "WIKI_HEADING_MISSING", "heading not found in published target"
					}
				}
			}
			if rule == "" {
				continue
			}
			line, col := 0, 0
			if ref.Offset >= 0 && ref.Offset < len(src) &&
				(bytes.HasPrefix(src[ref.Offset:], []byte("[[")) || bytes.HasPrefix(src[ref.Offset:], []byte("![["))) {
				n := sort.Search(len(starts), func(i int) bool { return starts[i] > ref.Offset })
				line = n
				col = ref.Offset - starts[n-1] + 1
			}
			target := ref.Target
			if ref.Heading != "" {
				target += "#" + ref.Heading
			}
			out = append(out, Diagnostic{Rule: rule, Severity: "warning", Path: filepath.ToSlash(note.Path), Line: line, Column: col, Message: message, Target: target})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return strings.Compare(a.Rule, b.Rule) < 0
	})
	return out, nil
}
