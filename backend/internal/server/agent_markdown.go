package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
	"time"

	"retro-obsidian-publish/backend/internal/api"
	"retro-obsidian-publish/backend/internal/vault"
)

const markdownDocVersion = "1"

func newAgentPageHandler(state *RuntimeState, publicConfig api.PublicConfig, pageHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		baseURL := deriveBaseURL(r)

		switch path {
		case "/AGENTS.md":
			writeMarkdownResponse(w, r, baseURL+path, renderAgentsMarkdown(state, publicConfig, baseURL))
			return
		case "/llms.txt":
			writeTextResponse(w, r, "text/plain; charset=utf-8", renderLLMSTxt(state, publicConfig, baseURL))
			return
		case "/sitemap.md":
			writeMarkdownResponse(w, r, baseURL+path, renderSitemapMarkdown(state, publicConfig, baseURL))
			return
		case "/sitemap.xml":
			writeTextResponse(w, r, "application/xml; charset=utf-8", renderSitemapXML(state, baseURL))
			return
		case "/index.md":
			writeMarkdownResponse(w, r, baseURL+"/", renderHomeMarkdown(state, publicConfig, baseURL))
			return
		}

		if strings.HasPrefix(path, "/note/") && strings.HasSuffix(path, ".md") {
			slug := strings.TrimSuffix(strings.TrimPrefix(path, "/note/"), ".md")
			if body, ok := renderNoteMarkdown(state, publicConfig, baseURL, slug); ok {
				writeMarkdownResponse(w, r, baseURL+"/note/"+slug, body)
				return
			}
			http.NotFound(w, r)
			return
		}

		if wantsMarkdown(r) {
			if path == "/" {
				writeMarkdownResponse(w, r, baseURL+"/", renderHomeMarkdown(state, publicConfig, baseURL))
				return
			}
			if strings.HasPrefix(path, "/note/") {
				slug := strings.TrimPrefix(path, "/note/")
				if body, ok := renderNoteMarkdown(state, publicConfig, baseURL, slug); ok {
					writeMarkdownResponse(w, r, baseURL+"/note/"+slug, body)
					return
				}
				http.NotFound(w, r)
				return
			}
		}

		pageHandler.ServeHTTP(w, r)
	})
}

func wantsMarkdown(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/markdown") || strings.Contains(accept, "text/x-markdown")
}

func deriveBaseURL(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host
}

func writeMarkdownResponse(w http.ResponseWriter, r *http.Request, canonicalURL, body string) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", canonicalURL))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(body))
}

func writeTextResponse(w http.ResponseWriter, r *http.Request, contentType, body string) {
	w.Header().Set("Content-Type", contentType)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(body))
}

func renderAgentsMarkdown(state *RuntimeState, cfg api.PublicConfig, baseURL string) string {
	v, _ := state.Snapshot()
	vaultName := publicVaultName(cfg)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Agent Guide\n\n", vaultName)
	fmt.Fprintf(&b, "%s is a read-only published Obsidian vault served by Retro Obsidian Publish.\n\n", vaultName)
	b.WriteString("## Installation\n\n")
	b.WriteString("Readers do not need to install anything to browse this site. Operators run the project as a Go binary with an optional Node.js SSR sidecar.\n\n")
	b.WriteString("Local operator commands:\n\n```bash\n")
	b.WriteString("retro-obsidian-publish serve --vault /path/to/vault --port 8080\n")
	b.WriteString("devctl up --profile example\n")
	b.WriteString("```\n\n")
	b.WriteString("## Configuration\n\n")
	fmt.Fprintf(&b, "- HTML home: %s/\n", baseURL)
	fmt.Fprintf(&b, "- Markdown home mirror: %s/index.md\n", baseURL)
	fmt.Fprintf(&b, "- Markdown sitemap: %s/sitemap.md\n", baseURL)
	fmt.Fprintf(&b, "- XML sitemap: %s/sitemap.xml\n", baseURL)
	fmt.Fprintf(&b, "- LLM index: %s/llms.txt\n", baseURL)
	fmt.Fprintf(&b, "- Notes: %d published Markdown notes\n\n", len(v.AllNotes()))
	b.WriteString("URL scheme:\n\n")
	b.WriteString("- HTML note: `/note/{slug}`\n")
	b.WriteString("- Markdown note mirror: `/note/{slug}.md`\n")
	b.WriteString("- Content negotiation: send `Accept: text/markdown` to `/` or `/note/{slug}`\n\n")
	b.WriteString("## Usage\n\n")
	b.WriteString("Agents should start at `/llms.txt`, then read `/sitemap.md` to discover note pages. For a note, prefer `/note/{slug}.md` when structured Markdown is needed, or `/note/{slug}` when rendered HTML is needed.\n\n")
	b.WriteString("## Glossary\n\n")
	b.WriteString("- **Vault:** The read-only Obsidian Markdown directory used as the source of truth.\n")
	b.WriteString("- **Slug:** URL-safe note identifier derived from the note path.\n")
	b.WriteString("- **Wiki link:** Obsidian link syntax such as `[[Target]]`, resolved to a note URL.\n")
	b.WriteString("- **Markdown mirror:** A `.md` representation of an HTML page, intended for agents and text tools.\n")
	b.WriteString("\n## Sitemap\n\n")
	appendNoteLinks(&b, sortedNotes(v), baseURL, true)
	return b.String()
}

func renderLLMSTxt(state *RuntimeState, cfg api.PublicConfig, baseURL string) string {
	v, _ := state.Snapshot()
	vaultName := publicVaultName(cfg)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", vaultName)
	fmt.Fprintf(&b, "> Read-only Obsidian vault with %d published notes.\n\n", len(v.AllNotes()))
	b.WriteString("## Key resources\n\n")
	fmt.Fprintf(&b, "- [Agent guide](%s/AGENTS.md)\n", baseURL)
	fmt.Fprintf(&b, "- [Markdown sitemap](%s/sitemap.md)\n", baseURL)
	fmt.Fprintf(&b, "- [XML sitemap](%s/sitemap.xml)\n", baseURL)
	fmt.Fprintf(&b, "- [Home markdown mirror](%s/index.md)\n", baseURL)
	b.WriteString("\n## Notes\n\n")
	appendNoteLinks(&b, sortedNotes(v), baseURL, false)
	return b.String()
}

func renderSitemapMarkdown(state *RuntimeState, cfg api.PublicConfig, baseURL string) string {
	v, _ := state.Snapshot()
	var b strings.Builder
	b.WriteString("# Sitemap\n\n")
	fmt.Fprintf(&b, "Sitemap for %s.\n\n", publicVaultName(cfg))
	b.WriteString("## Site resources\n\n")
	fmt.Fprintf(&b, "- [Home](%s/)\n", baseURL)
	fmt.Fprintf(&b, "- [Home markdown mirror](%s/index.md)\n", baseURL)
	fmt.Fprintf(&b, "- [Agent guide](%s/AGENTS.md)\n", baseURL)
	fmt.Fprintf(&b, "- [LLM index](%s/llms.txt)\n", baseURL)
	fmt.Fprintf(&b, "- [XML sitemap](%s/sitemap.xml)\n\n", baseURL)
	b.WriteString("## Notes\n\n")
	appendNoteLinks(&b, sortedNotes(v), baseURL, true)
	return b.String()
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func renderSitemapXML(state *RuntimeState, baseURL string) string {
	v, _ := state.Snapshot()
	urls := []sitemapURL{{Loc: baseURL + "/", LastMod: time.Now().Format("2006-01-02")}}
	for _, note := range sortedNotes(v) {
		urls = append(urls, sitemapURL{Loc: baseURL + "/note/" + note.Slug, LastMod: note.ModTime.Format("2006-01-02")})
		urls = append(urls, sitemapURL{Loc: baseURL + "/note/" + note.Slug + ".md", LastMod: note.ModTime.Format("2006-01-02")})
	}
	out, err := xml.MarshalIndent(sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}, "", "  ")
	if err != nil {
		return ""
	}
	return xml.Header + string(out) + "\n"
}

func renderHomeMarkdown(state *RuntimeState, cfg api.PublicConfig, baseURL string) string {
	v, _ := state.Snapshot()
	notes := sortedNotes(v)
	vaultName := publicVaultName(cfg)
	desc := fmt.Sprintf("Markdown index for %s, listing %d published notes.", vaultName, len(notes))
	var b strings.Builder
	writeFrontmatter(&b, vaultName, desc, time.Now(), baseURL+"/")
	fmt.Fprintf(&b, "# %s\n\n", vaultName)
	fmt.Fprintf(&b, "%s\n\n", desc)
	b.WriteString("## Notes\n\n")
	appendNoteLinks(&b, notes, baseURL, true)
	b.WriteString("\n## Sitemap\n\n")
	fmt.Fprintf(&b, "- [Markdown sitemap](%s/sitemap.md)\n", baseURL)
	fmt.Fprintf(&b, "- [XML sitemap](%s/sitemap.xml)\n", baseURL)
	fmt.Fprintf(&b, "- [Agent guide](%s/AGENTS.md)\n", baseURL)
	return b.String()
}

func renderNoteMarkdown(state *RuntimeState, cfg api.PublicConfig, baseURL, slug string) (string, bool) {
	v, _ := state.Snapshot()
	note, ok := v.GetNote(slug)
	if !ok {
		return "", false
	}
	desc := note.Excerpt
	if strings.TrimSpace(desc) == "" {
		desc = fmt.Sprintf("Markdown mirror for %s in %s.", note.Title, publicVaultName(cfg))
	}
	var b strings.Builder
	writeFrontmatter(&b, note.Title, desc, note.ModTime, baseURL+"/note/"+note.Slug)
	fmt.Fprintf(&b, "# %s\n\n", note.Title)
	if note.Excerpt != "" {
		fmt.Fprintf(&b, "%s\n\n", note.Excerpt)
	}
	if len(note.Tags) > 0 {
		b.WriteString("## Tags\n\n")
		for _, tag := range note.Tags {
			fmt.Fprintf(&b, "- `%s`\n", tag)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Content\n\n")
	b.WriteString("This mirror is derived from the rendered note HTML. Use the canonical HTML page for the fully styled view.\n\n")
	b.WriteString("```html\n")
	b.WriteString(strings.TrimSpace(note.HTML))
	b.WriteString("\n```\n\n")
	if len(note.WikiLinks) > 0 {
		b.WriteString("## Wiki Links\n\n")
		for _, wl := range note.WikiLinks {
			fmt.Fprintf(&b, "- %s\n", wl.Target)
		}
		b.WriteString("\n")
	}
	if len(note.Backlinks) > 0 {
		b.WriteString("## Backlinks\n\n")
		for _, backlink := range note.Backlinks {
			fmt.Fprintf(&b, "- [%s](%s/note/%s)\n", backlink, baseURL, backlink)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Sitemap\n\n")
	fmt.Fprintf(&b, "- [Home](%s/)\n", baseURL)
	fmt.Fprintf(&b, "- [Home markdown mirror](%s/index.md)\n", baseURL)
	fmt.Fprintf(&b, "- [Markdown sitemap](%s/sitemap.md)\n", baseURL)
	return b.String(), true
}

func writeFrontmatter(b *strings.Builder, title, description string, modTime time.Time, canonicalURL string) {
	if modTime.IsZero() {
		modTime = time.Now()
	}
	b.WriteString("---\n")
	fmt.Fprintf(b, "title: %q\n", title)
	fmt.Fprintf(b, "description: %q\n", trimDescription(description))
	fmt.Fprintf(b, "doc_version: %s\n", markdownDocVersion)
	fmt.Fprintf(b, "last_updated: %s\n", modTime.Format("2006-01-02"))
	fmt.Fprintf(b, "canonical_url: %q\n", canonicalURL)
	b.WriteString("---\n\n")
}

func appendNoteLinks(b *strings.Builder, notes []*vault.Note, baseURL string, includeMirrors bool) {
	for _, note := range notes {
		fmt.Fprintf(b, "- [%s](%s/note/%s)", note.Title, baseURL, note.Slug)
		if includeMirrors {
			fmt.Fprintf(b, " ([markdown](%s/note/%s.md))", baseURL, note.Slug)
		}
		b.WriteString("\n")
	}
}

func sortedNotes(v *vault.Vault) []*vault.Note {
	notes := v.AllNotes()
	sort.Slice(notes, func(i, j int) bool {
		return strings.ToLower(notes[i].Title) < strings.ToLower(notes[j].Title)
	})
	return notes
}

func publicVaultName(cfg api.PublicConfig) string {
	if cfg.PageTitle != "" {
		return cfg.PageTitle
	}
	if cfg.VaultName != "" {
		return cfg.VaultName
	}
	return "Vault"
}

func trimDescription(s string) string {
	s = strings.Join(strings.Fields(html.UnescapeString(stripHTMLTags(s))), " ")
	if len(s) > 280 {
		return s[:277] + "..."
	}
	return s
}

func stripHTMLTags(s string) string {
	inTag := false
	var out []rune
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
			out = append(out, ' ')
		default:
			if !inTag {
				out = append(out, r)
			}
		}
	}
	return string(out)
}
