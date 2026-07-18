// Package vaultdata exposes read-only vault content to JavaScript widget
// page scripts as the goja native module "vault.data".
//
// JS view:
//
//	const vault = require("vault.data");
//	vault.config()            → {vaultName, pageTitle, notes}
//	vault.notes()             → NoteListItem[] (same shape as /api/notes)
//	vault.note(slug)          → full Note or null (same shape as /api/notes/{slug})
//	vault.search(q, {limit})  → SearchResult[] (same shape as /api/search)
//	vault.tree()              → FileNode (same shape as /api/tree)
//	vault.tags()              → TagCount[] (same shape as /api/tags)
//
// Every function reads one immutable snapshot per call (RuntimeState swaps
// snapshots atomically), and every return value is passed through a JSON
// round-trip so scripts see exactly the public wire shapes — never live Go
// objects. The module is read-only by design; write verbs are a deliberate
// non-goal for v1 (see PV-WIDGET-DSL-015 open questions).
package vaultdata

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/publish-vault/internal/api"
)

// ModuleName is the require() specifier for the module.
const ModuleName = "vault.data"

const defaultSearchLimit = 30

// Register registers the vault.data module backed by the given snapshot
// provider and public site config.
func Register(reg *require.Registry, provider api.SnapshotProvider, config api.PublicConfig) {
	reg.RegisterNativeModule(ModuleName, Loader(provider, config))
}

// Loader returns the module loader; exposed separately for tests and hosts
// that manage registries themselves.
func Loader(provider api.SnapshotProvider, config api.PublicConfig) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		mustSet := func(name string, fn any) {
			if err := exports.Set(name, fn); err != nil {
				panic(fmt.Sprintf("vault.data: set %s: %v", name, err))
			}
		}

		mustSet("config", func() (goja.Value, error) {
			v, _ := provider.Snapshot()
			return toJS(vm, api.SiteConfig{
				VaultName: config.VaultName,
				PageTitle: config.PageTitle,
				Notes:     v.Count(),
			})
		})

		mustSet("notes", func() (goja.Value, error) {
			v, _ := provider.Snapshot()
			return toJS(vm, api.NoteList(v))
		})

		mustSet("note", func(slug string) (goja.Value, error) {
			v, _ := provider.Snapshot()
			note, ok := v.GetNote(slug)
			if !ok {
				return goja.Null(), nil
			}
			return toJS(vm, note)
		})

		mustSet("search", func(q string, opts ...map[string]any) (goja.Value, error) {
			limit := defaultSearchLimit
			if len(opts) > 0 {
				if raw, ok := opts[0]["limit"]; ok {
					switch n := raw.(type) {
					case int64:
						limit = int(n)
					case float64:
						limit = int(n)
					}
				}
			}
			if limit <= 0 {
				limit = defaultSearchLimit
			}
			_, si := provider.Snapshot()
			results, err := si.Search(q, limit)
			if err != nil {
				return nil, fmt.Errorf("vault.data search: %w", err)
			}
			if results == nil {
				return toJS(vm, []any{})
			}
			return toJS(vm, results)
		})

		mustSet("tree", func() (goja.Value, error) {
			v, _ := provider.Snapshot()
			return toJS(vm, v.FileTree())
		})

		mustSet("tags", func() (goja.Value, error) {
			v, _ := provider.Snapshot()
			return toJS(vm, api.TagCounts(v))
		})
	}
}

// toJS converts a Go value into a plain goja value via a JSON round-trip so
// scripts receive the exact public wire shape (json tags respected, no Go
// method sets or pointers leaking into the runtime).
func toJS(vm *goja.Runtime, v any) (goja.Value, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("vault.data: marshal: %w", err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("vault.data: unmarshal: %w", err)
	}
	return vm.ToValue(out), nil
}
