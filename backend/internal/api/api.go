// Package api provides the HTTP REST API for the Obsidian vault.
//
// Endpoints:
//
//	GET /api/notes          — list all notes (slug, title, tags, excerpt, modTime)
//	GET /api/notes/{slug}   — full note (html, frontmatter, wikiLinks, backlinks)
//	GET /api/tree           — hierarchical file tree
//	GET /api/search?q=...   — full-text search
//	GET /api/tags           — all tags with counts
package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"retro-obsidian-publish/backend/internal/search"
	"retro-obsidian-publish/backend/internal/vault"
)

// Handler holds dependencies for the API.
type Handler struct {
	vault  *vault.Vault
	search *search.Index
}

// New creates a new API Handler.
func New(v *vault.Vault, si *search.Index) *Handler {
	return &Handler{vault: v, search: si}
}

// Register mounts all routes on the given router.
func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/notes", h.listNotes).Methods("GET")
	r.HandleFunc("/api/notes/{slug:.*}", h.getNote).Methods("GET")
	r.HandleFunc("/api/tree", h.getTree).Methods("GET")
	r.HandleFunc("/api/search", h.searchNotes).Methods("GET")
	r.HandleFunc("/api/tags", h.listTags).Methods("GET")
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

// listNotes returns all notes as a list.
func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	notes := h.vault.AllNotes()
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
	jsonResponse(w, items)
}

// getNote returns a single note by slug.
func (h *Handler) getNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	note, ok := h.vault.GetNote(slug)
	if !ok {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}
	jsonResponse(w, note)
}

// getTree returns the vault file tree.
func (h *Handler) getTree(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, h.vault.FileTree())
}

// searchNotes performs full-text search.
func (h *Handler) searchNotes(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		jsonResponse(w, []search.SearchResult{})
		return
	}
	results, err := h.search.Search(q, 30)
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusInternalServerError)
		return
	}
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

// listTags returns all tags with their note counts.
func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	counts := map[string]int{}
	for _, n := range h.vault.AllNotes() {
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
	jsonResponse(w, tags)
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
