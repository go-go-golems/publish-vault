// Package vaultwidgets exposes note-domain widget builders to JavaScript
// page scripts as the goja native module "vault.widgets" (ticket
// PV-VAULT-WIDGETS-016).
//
// JS view:
//
//	const vw = require("vault.widgets");
//	vw.noteHtml(note, {embeds, anchors, highlight, mermaid})  → NoteHtml node
//	vw.frontmatter(note, {onTagClick})                        → FrontmatterPanel node
//	vw.breadcrumb(note)                                       → BreadcrumbBar node
//	vw.backlinks(note, {onSelect})                            → BacklinksPanel node (entries resolved server-side)
//	vw.tagList(tags, {onSelect})                              → TagCloud node
//	vw.noteCard(noteListItem, {onSelect})                     → NoteCard node
//
// Every helper returns a plain Widget IR component node
// ({kind:"component", type, props}) — exactly what widget.raw.component
// produces — so results compose with widget.dsl v3 builders (.view(), use()).
// This module is the sibling-module workaround decided in -016 D1: when
// rag-evaluation-system#28 lands a namespace extension API, these builders
// graduate to a first-class widget.vault.* namespace.
//
// Helpers are read-only. vw.backlinks consults one vault snapshot per call
// (decision D4) so scripts don't loop vault.note(slug) per backlink.
package vaultwidgets

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/publish-vault/pkg/api"
)

// ModuleName is the require() specifier for the module.
const ModuleName = "vault.widgets"

// Register registers the vault.widgets module backed by the given snapshot
// provider (used only for backlink resolution).
func Register(reg *require.Registry, provider api.SnapshotProvider, config api.PublicConfig) {
	reg.RegisterNativeModule(ModuleName, Loader(provider, config))
}

// Loader returns the module loader; exposed separately for tests.
func Loader(provider api.SnapshotProvider, _ api.PublicConfig) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		mustSet := func(name string, fn any) {
			if err := exports.Set(name, fn); err != nil {
				panic(fmt.Sprintf("vault.widgets: set %s: %v", name, err))
			}
		}

		// Accepts either a note object (from vault.data.note()) or a slug
		// string, in which case the note is fetched from the current
		// snapshot. Unknown slugs throw so page authors fail loudly; guard
		// with vault.note(slug) for soft behavior.
		mustSet("noteHtml", func(noteOrSlug goja.Value, opts ...map[string]any) (map[string]any, error) {
			o := firstOpts(opts)
			var html, slug string
			switch v := noteOrSlug.Export().(type) {
			case string:
				vaultSnap, _ := provider.Snapshot()
				target, ok := vaultSnap.GetNote(v)
				if !ok {
					return nil, fmt.Errorf("vault.widgets.noteHtml: unknown note %q", v)
				}
				html, slug = target.HTML, target.Slug
			case map[string]any:
				html, slug = stringField(v, "html"), stringField(v, "slug")
			default:
				return nil, fmt.Errorf("vault.widgets.noteHtml: expected note object or slug string, got %T", v)
			}
			return node("NoteHtml", map[string]any{
				"html":      html,
				"slug":      slug,
				"embeds":    boolOpt(o, "embeds", true),
				"anchors":   boolOpt(o, "anchors", true),
				"highlight": boolOpt(o, "highlight", true),
				"mermaid":   boolOpt(o, "mermaid", true),
			}), nil
		})

		mustSet("frontmatter", func(note map[string]any, opts ...map[string]any) map[string]any {
			o := firstOpts(opts)
			props := map[string]any{
				"frontmatter": note["frontmatter"],
				"tags":        note["tags"],
				"modTime":     note["modTime"],
			}
			if action, ok := o["onTagClick"]; ok {
				props["onTagClick"] = action
			}
			return node("FrontmatterPanel", props)
		})

		mustSet("breadcrumb", func(note map[string]any) map[string]any {
			path := stringField(note, "path")
			trimmed := strings.TrimSuffix(path, ".md")
			segments := []any{}
			for _, part := range strings.Split(trimmed, "/") {
				if part == "" {
					continue
				}
				segments = append(segments, map[string]any{"label": part})
			}
			return node("BreadcrumbBar", map[string]any{"segments": segments})
		})

		mustSet("backlinks", func(note map[string]any, opts ...map[string]any) map[string]any {
			o := firstOpts(opts)
			v, _ := provider.Snapshot()
			entries := []any{}
			if raw, ok := note["backlinks"].([]any); ok {
				for _, item := range raw {
					slug, ok := item.(string)
					if !ok {
						continue
					}
					entry := map[string]any{"slug": slug, "title": slug}
					if target, found := v.GetNote(slug); found {
						entry["title"] = target.Title
						entry["excerpt"] = target.Excerpt
					}
					entries = append(entries, entry)
				}
			}
			props := map[string]any{"entries": entries}
			if action, ok := o["onSelect"]; ok {
				props["onSelect"] = action
			}
			return node("BacklinksPanel", props)
		})

		mustSet("tagList", func(tags []any, opts ...map[string]any) map[string]any {
			o := firstOpts(opts)
			props := map[string]any{"tags": tags}
			if action, ok := o["onSelect"]; ok {
				props["onSelect"] = action
			}
			return node("TagCloud", props)
		})

		mustSet("noteCard", func(item map[string]any, opts ...map[string]any) map[string]any {
			o := firstOpts(opts)
			props := map[string]any{
				"slug":    stringField(item, "slug"),
				"title":   stringField(item, "title"),
				"excerpt": stringField(item, "excerpt"),
				"tags":    item["tags"],
			}
			if action, ok := o["onSelect"]; ok {
				props["onSelect"] = action
			}
			return node("NoteCard", props)
		})
	}
}

func node(componentType string, props map[string]any) map[string]any {
	return map[string]any{"kind": "component", "type": componentType, "props": props}
}

func firstOpts(opts []map[string]any) map[string]any {
	if len(opts) > 0 && opts[0] != nil {
		return opts[0]
	}
	return map[string]any{}
}

func stringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func boolOpt(m map[string]any, key string, fallback bool) bool {
	if b, ok := m[key].(bool); ok {
		return b
	}
	return fallback
}
