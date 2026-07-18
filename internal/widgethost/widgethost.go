// Package widgethost executes widget.dsl v3 page scripts (see ticket
// PV-WIDGET-DSL-015) and serves the resulting Widget IR over the same HTTP
// contract as rag-evaluation-system:
//
//	GET  /api/widget/pages           → [{id, title, path}]
//	GET  /api/widget/pages/{id}      → WidgetPage IR JSON
//	POST /api/widget/actions/{name}  → run a page-exported action handler
//
// Page scripts live in a directory of .js files (page id = file basename).
// Each script must define `const page = widget.page(...)` and may export
// action handlers via `const actions = { name: (payload, context) => result }`.
// Action names are addressed as "pageId.actionName"; a bare name is resolved
// by scanning pages in sorted order.
//
// Every render runs in a fresh goja VM (correctness over throughput — see
// design doc §9 Phase 3). Scripts are trusted, author-owned content, the same
// trust level as the vault markdown itself.
package widgethost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/rag-evaluation-system/pkg/widgetdsl"
	"github.com/gorilla/mux"

	"github.com/go-go-golems/publish-vault/internal/api"
	"github.com/go-go-golems/publish-vault/internal/vaultdata"
	"github.com/go-go-golems/publish-vault/internal/vaultwidgets"
)

// Host renders widget pages from a script directory against the live vault.
type Host struct {
	provider api.SnapshotProvider
	config   api.PublicConfig
	pagesDir string
}

// PageInfo describes one discovered page script.
type PageInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// ActionResult is the server-action response contract the frontend
// dispatcher understands ({ok, refresh, toast, patch, data, error, ...}).
type ActionResult map[string]any

// New creates a host reading page scripts from pagesDir.
func New(provider api.SnapshotProvider, config api.PublicConfig, pagesDir string) *Host {
	return &Host{provider: provider, config: config, pagesDir: pagesDir}
}

// RegisterRoutes mounts the widget HTTP contract on the router.
func (h *Host) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/widget/pages", h.handleListPages).Methods("GET")
	r.HandleFunc("/api/widget/pages/{id}", h.handleGetPage).Methods("GET")
	r.HandleFunc("/api/widget/actions/{name}", h.handleAction).Methods("POST")
}

// ListPages scans the pages directory. The title comes from a cheap static
// match on `widget.page("Title"` so listing never executes scripts.
func (h *Host) ListPages() ([]PageInfo, error) {
	entries, err := os.ReadDir(h.pagesDir)
	if err != nil {
		return nil, err
	}
	var pages []PageInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".js")
		path := filepath.Join(h.pagesDir, entry.Name())
		pages = append(pages, PageInfo{ID: id, Title: staticPageTitle(path, id), Path: path})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].ID < pages[j].ID })
	return pages, nil
}

var pageTitleRe = regexp.MustCompile(`widget\.page\(\s*(?:"([^"]*)"|'([^']*)')`)

func staticPageTitle(path, fallback string) string {
	source, err := os.ReadFile(path) // #nosec G304 -- operator-configured pages dir
	if err != nil {
		return fallback
	}
	if m := pageTitleRe.FindSubmatch(source); m != nil {
		if len(m[1]) > 0 {
			return string(m[1])
		}
		if len(m[2]) > 0 {
			return string(m[2])
		}
	}
	return fallback
}

// RenderPage executes the page script and returns the lowered WidgetPage IR.
func (h *Host) RenderPage(id string, query url.Values) (json.RawMessage, error) {
	out, _, err := h.evalPage(id, query)
	if err != nil {
		return nil, err
	}
	pageVal := out.Get("page")
	if pageVal == nil || goja.IsUndefined(pageVal) || goja.IsNull(pageVal) {
		return nil, fmt.Errorf("page script %q did not define `const page`", id)
	}
	data, err := json.Marshal(pageVal.Export())
	if err != nil {
		return nil, fmt.Errorf("marshal page %q: %w", id, err)
	}
	return data, nil
}

// HandleAction resolves and invokes a page-exported action handler.
// name is either "pageId.actionName" or a bare action name searched across
// all pages in sorted order.
func (h *Host) HandleAction(name string, payload, context json.RawMessage) (ActionResult, error) {
	pageID, actionName, explicit := strings.Cut(name, ".")
	if !explicit {
		actionName = name
		pages, err := h.ListPages()
		if err != nil {
			return nil, err
		}
		pageID = ""
		for _, p := range pages {
			if h.pageExportsAction(p.ID, actionName) {
				pageID = p.ID
				break
			}
		}
		if pageID == "" {
			return nil, fmt.Errorf("no page exports action %q", actionName)
		}
	}
	return h.invokeAction(pageID, actionName, payload, context)
}

func (h *Host) pageExportsAction(pageID, actionName string) bool {
	out, vm, err := h.evalPage(pageID, nil)
	if err != nil {
		return false
	}
	actionsVal := out.Get("actions")
	if actionsVal == nil || goja.IsUndefined(actionsVal) || goja.IsNull(actionsVal) {
		return false
	}
	handler := actionsVal.ToObject(vm).Get(actionName)
	_, ok := goja.AssertFunction(handler)
	return ok
}

func (h *Host) invokeAction(pageID, actionName string, payload, context json.RawMessage) (ActionResult, error) {
	out, vm, err := h.evalPage(pageID, nil)
	if err != nil {
		return nil, err
	}
	actionsVal := out.Get("actions")
	if actionsVal == nil || goja.IsUndefined(actionsVal) || goja.IsNull(actionsVal) {
		return nil, fmt.Errorf("page %q exports no actions", pageID)
	}
	handlerVal := actionsVal.ToObject(vm).Get(actionName)
	handler, ok := goja.AssertFunction(handlerVal)
	if !ok {
		return nil, fmt.Errorf("page %q has no action %q", pageID, actionName)
	}

	payloadVal, err := jsonToValue(vm, payload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	contextVal, err := jsonToValue(vm, context)
	if err != nil {
		return nil, fmt.Errorf("decode context: %w", err)
	}

	resultVal, err := handler(goja.Undefined(), payloadVal, contextVal)
	if err != nil {
		return nil, fmt.Errorf("action %s.%s: %w", pageID, actionName, err)
	}
	if resultVal == nil || goja.IsUndefined(resultVal) || goja.IsNull(resultVal) {
		return ActionResult{"ok": true}, nil
	}
	data, err := json.Marshal(resultVal.Export())
	if err != nil {
		return nil, fmt.Errorf("marshal action result: %w", err)
	}
	var result ActionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("action %s.%s returned a non-object result", pageID, actionName)
	}
	if _, ok := result["ok"]; !ok {
		result["ok"] = true
	}
	return result, nil
}

// evalPage runs the page script in a fresh VM and returns the wrapper output
// object ({page: <lowered IR>, actions: <exported handler functions>}) as
// live goja values, plus the VM that owns them.
func (h *Host) evalPage(id string, query url.Values) (*goja.Object, *goja.Runtime, error) {
	vm, err := h.newRuntime()
	if err != nil {
		return nil, nil, err
	}
	if err := injectRequest(vm, id, query); err != nil {
		return nil, nil, err
	}
	source, err := h.readPage(id)
	if err != nil {
		return nil, nil, err
	}
	value, err := vm.RunString(wrapPageScript(source))
	if err != nil {
		return nil, nil, fmt.Errorf("run page %q: %w", id, err)
	}
	return value.ToObject(vm), vm, nil
}

func (h *Host) newRuntime() (*goja.Runtime, error) {
	vm := goja.New()
	reg := require.NewRegistry()
	widgetdsl.Register(reg)
	vaultdata.Register(reg, h.provider, h.config)
	vaultwidgets.Register(reg, h.provider, h.config)
	reg.Enable(vm)
	return vm, nil
}

func (h *Host) readPage(id string) (string, error) {
	if !validPageID(id) {
		return "", fmt.Errorf("invalid page id %q", id)
	}
	// os.Root confines every open to the pages directory, so a crafted id
	// cannot escape it regardless of validation above.
	root, err := os.OpenRoot(h.pagesDir)
	if err != nil {
		return "", fmt.Errorf("pages dir unavailable: %w", err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(id + ".js")
	if err != nil {
		return "", fmt.Errorf("unknown page %q", id)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read page %q: %w", id, err)
	}
	return string(data), nil
}

func validPageID(id string) bool {
	if id == "" || strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") || strings.HasPrefix(id, ".") {
		return false
	}
	return true
}

// wrapPageScript wraps a page script with the same run convention as
// rag-evaluation-system's widgetdsl-v3-preview host: the script defines
// `const page` (a PageBuilder or plain object) and optionally `const actions`.
func wrapPageScript(source string) string {
	return `(function(){
` + source + `
var __out = {};
if (typeof page !== "undefined") {
  __out.page = page && typeof page.toPage === "function" ? page.toPage() : page;
}
if (typeof actions !== "undefined") {
  __out.actions = actions;
}
if (__out.page === undefined && __out.actions === undefined) {
  throw new Error("page script must define const page (and may define const actions)");
}
return __out;
})()`
}

// injectRequest exposes the request query to page scripts as
// globalThis.request = {pageId, query:{k:v}} so pages can parameterize.
func injectRequest(vm *goja.Runtime, id string, query url.Values) error {
	q := map[string]string{}
	for key, values := range query {
		if len(values) > 0 {
			q[key] = values[0]
		}
	}
	return vm.Set("request", map[string]any{"pageId": id, "query": q})
}

func jsonToValue(vm *goja.Runtime, raw json.RawMessage) (goja.Value, error) {
	if len(raw) == 0 {
		return goja.Undefined(), nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return vm.ToValue(v), nil
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (h *Host) handleListPages(w http.ResponseWriter, r *http.Request) {
	pages, err := h.ListPages()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "listing pages failed")
		return
	}
	if pages == nil {
		pages = []PageInfo{}
	}
	writeJSON(w, pages)
}

func (h *Host) handleGetPage(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	page, err := h.RenderPage(id, r.URL.Query())
	if err != nil {
		if strings.Contains(err.Error(), "unknown page") || strings.Contains(err.Error(), "invalid page id") {
			httpError(w, http.StatusNotFound, err.Error())
			return
		}
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Encoder-mediated write: page is json.Marshal output from RenderPage;
	// Encode revalidates it as JSON on the way out.
	_ = json.NewEncoder(w).Encode(page)
}

const maxActionBodyBytes = 1 << 20 // 1 MiB

func (h *Host) handleAction(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var body struct {
		Payload json.RawMessage `json:"payload"`
		Context json.RawMessage `json:"context"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxActionBodyBytes)).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, "invalid action body")
		return
	}
	result, err := h.HandleAction(name, body.Payload, body.Context)
	if err != nil {
		writeJSON(w, ActionResult{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, result)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
