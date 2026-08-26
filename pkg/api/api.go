// Package api provides the HTTP REST API for the Obsidian vault.
//
// Endpoints:
//
//	GET /api/notes          — list all notes (slug, title, tags, excerpt, modTime)
//	GET /api/notes/{slug}   — full note (html, frontmatter, wikiLinks, backlinks)
//	GET /api/tree           — hierarchical file tree
//	GET /api/search?q=...        — full-text search (legacy bare array)
//	GET /api/search/advanced      — typed advanced search (envelope)
//	GET /api/tags                 — all tags with counts
package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"github.com/go-go-golems/publish-vault/pkg/search"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

// SnapshotProvider returns the currently active vault and search index.
// Implementations may swap both values atomically during a reload.
type SnapshotProvider interface {
	Snapshot() (*vault.Vault, *search.Index)
}

type staticProvider struct {
	vault  *vault.Vault
	search *search.Index
}

func (p staticProvider) Snapshot() (*vault.Vault, *search.Index) {
	return p.vault, p.search
}

// Handler holds dependencies for the API.
type Handler struct {
	provider SnapshotProvider
	config   PublicConfig
}

// New creates a new API Handler backed by fixed vault/search pointers.
func New(v *vault.Vault, si *search.Index, vaultName string) *Handler {
	return NewWithProvider(staticProvider{vault: v, search: si}, PublicConfig{VaultName: vaultName, PageTitle: vaultName})
}

// NewWithProvider creates a new API Handler backed by a dynamic provider.
func NewWithProvider(provider SnapshotProvider, config PublicConfig) *Handler {
	if config.PageTitle == "" {
		config.PageTitle = config.VaultName
	}
	return &Handler{provider: provider, config: config}
}

// Register mounts all routes on the given router.
func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/config", h.getConfig).Methods("GET")
	r.HandleFunc("/api/notes", h.listNotes).Methods("GET")
	r.HandleFunc("/api/notes/{slug:.*}/raw", h.getNoteRaw).Methods("GET")
	r.HandleFunc("/api/notes/{slug:.*}", h.getNote).Methods("GET")
	r.HandleFunc("/api/tree", h.getTree).Methods("GET")
	r.HandleFunc("/api/search", h.searchNotes).Methods("GET")
	r.HandleFunc("/api/search/advanced", h.searchAdvanced).Methods("GET")
	r.HandleFunc("/api/tags", h.listTags).Methods("GET")
}

// PublicConfig is the operator-configured public site metadata returned by /api/config.
type PublicConfig struct {
	VaultName string `json:"vaultName"`
	PageTitle string `json:"pageTitle"`
}

// SiteConfig is the public site configuration returned by /api/config.
type SiteConfig struct {
	VaultName string `json:"vaultName"`
	PageTitle string `json:"pageTitle"`
	Notes     int    `json:"notes"`
}

// getConfig returns public site configuration.
func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	v, _ := h.provider.Snapshot()
	jsonResponse(w, SiteConfig{
		VaultName: h.config.VaultName,
		PageTitle: h.config.PageTitle,
		Notes:     v.Count(),
	})
}

// NoteListItem is the lightweight note representation for listing.
type NoteListItem struct {
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Excerpt string   `json:"excerpt"`
	ModTime string   `json:"modTime"`
	Path    string   `json:"path"`
}

// NoteList returns the sorted list representation served by /api/notes.
func NoteList(v *vault.Vault) []NoteListItem {
	notes := v.AllNotes()
	items := make([]NoteListItem, 0, len(notes))
	for _, n := range notes {
		items = append(items, NoteListItem{
			Slug:    n.Slug,
			Title:   n.Title,
			Tags:    nonNilStrings(n.Tags),
			Excerpt: n.Excerpt,
			ModTime: n.ModTime.Format("2006-01-02"),
			Path:    n.Path,
		})
	}
	// Sort alphabetically by title
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	return items
}

// listNotes returns all notes as a list.
func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	v, _ := h.provider.Snapshot()
	jsonResponse(w, NoteList(v))
}

// getNote returns a single note by slug.
//
// On an exact miss it tries the normalized index before 404ing: slugs preserve
// a trailing "/" and a doubled "//", so URLs a reader can produce by hand — a
// copy-paste that kept the trailing slash, a mixed-case path — used to be a
// permanent 404 on a note that was one normalization step away. Redirecting
// rather than serving keeps one canonical URL per note for caches and crawlers.
func (h *Handler) getNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	v, _ := h.provider.Snapshot()
	note, ok := v.GetNote(slug)
	if !ok {
		if canonical, found := v.CanonicalSlug(slug); found && safeRedirectSlug(canonical) {
			// 308 rather than 301: the method and body must be preserved, and
			// the redirect is a property of the slug, not of this request.
			//
			// #nosec G710 -- not an open redirect. The target is a key from the
			// vault's own slug index; the request slug is only a lookup key and
			// never reaches the Location header. safeRedirectSlug enforces the
			// property gosec's taint analysis cannot see.
			http.Redirect(w, r, "/api/notes/"+canonicalSlugPath(canonical), http.StatusPermanentRedirect)
			return
		}
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}
	jsonResponse(w, note)
}

// safeRedirectSlug reports whether a slug can be appended to "/api/notes/" and
// still yield a same-origin, same-prefix path.
//
// Slugify already restricts slugs to [a-z0-9-_/], so today no vault slug can
// carry a scheme, a host, or a traversal segment. This check does not trust
// that: a slug beginning with "/" would make the Location "//host"-shaped and
// therefore protocol-relative, which leaves the origin, and ".." would climb
// out of the prefix. Both are cheap to reject and keep the redirect safe if the
// slug alphabet is ever widened.
func safeRedirectSlug(slug string) bool {
	if slug == "" || strings.HasPrefix(slug, "/") || strings.HasPrefix(slug, `\`) {
		return false
	}
	for _, segment := range strings.Split(slug, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

// canonicalSlugPath escapes a slug for use in a redirect Location while
// leaving the "/" separators intact — the route is registered as
// {slug:.*}, so the path segments are part of the slug rather than
// sub-resources.
func canonicalSlugPath(slug string) string {
	parts := strings.Split(slug, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// getNoteRaw returns the raw markdown source for a note.
func (h *Handler) getNoteRaw(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	v, _ := h.provider.Snapshot()
	note, ok := v.GetNote(slug)
	if !ok {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	raw, err := v.ReadRaw(note.Path)
	if err != nil {
		http.Error(w, "note source not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+note.Slug+".md\"")
	// #nosec G705 -- the raw endpoint intentionally returns Markdown source as text/markdown, not rendered HTML.
	_, _ = w.Write(raw)
}

// getTree returns the vault file tree.
func (h *Handler) getTree(w http.ResponseWriter, r *http.Request) {
	v, _ := h.provider.Snapshot()
	jsonResponse(w, v.FileTree())
}

// searchNotes performs the legacy full-text search and returns a bare array.
// It delegates to the same typed SearchAdvanced implementation as the advanced
// endpoint so there is one search path during the compatibility window.
func (h *Handler) searchNotes(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		jsonResponse(w, []search.SearchResult{})
		return
	}
	_, si := h.provider.Snapshot()
	req, _ := search.NormalizeSearchRequest(search.SearchRequest{Query: q, Limit: 30})
	resp, err := si.SearchAdvanced(req)
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusInternalServerError)
		return
	}
	results := resp.Results
	if results == nil {
		results = []search.SearchResult{}
	}
	jsonResponse(w, results)
}

// TagCount holds a tag and its note count.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// TagCounts returns tag counts sorted by count, as served by /api/tags.
func TagCounts(v *vault.Vault) []TagCount {
	counts := map[string]int{}
	for _, n := range v.AllNotes() {
		for _, t := range n.Tags {
			counts[t]++
		}
	}
	tags := make([]TagCount, 0, len(counts))
	for t, c := range counts {
		tags = append(tags, TagCount{Tag: t, Count: c})
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Count > tags[j].Count
	})
	return tags
}

// listTags returns all tags with their note counts.
func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	v, _ := h.provider.Snapshot()
	jsonResponse(w, TagCounts(v))
}

// jsonResponse writes v as JSON with proper content-type.
func jsonResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":"encoding failed"}`, http.StatusInternalServerError)
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
