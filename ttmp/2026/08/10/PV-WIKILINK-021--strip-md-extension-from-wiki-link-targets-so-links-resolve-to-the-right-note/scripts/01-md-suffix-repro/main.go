// Command 01-md-suffix-repro reproduces the ".md in a wiki-link target" bug
// against a throwaway vault built in a temp directory.
//
// It builds the same shape as the real vault (Transcripts/YYYY/MM/DD/<topic>/<note>.md),
// writes one note whose links use the ".md#Heading" form that Obsidian emits when
// you link into a heading of a file in another folder, and prints the rendered
// HTML plus the raw index lookups so we can see exactly where resolution fails.
//
//	go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/01-md-suffix-repro
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	root, err := os.MkdirTemp("", "mdsuffix")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	write("Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md",
		"# Doctoral thesis\n\n## Identity is an API decision\n\nbody\n")
	// Decoy: its own path already slugifies to "...doctoral-thesis-md", which is
	// exactly what the buggy ".md" target slugifies to. This is the silent
	// wrong-note case, as opposed to the merely-unresolved case above.
	write("Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis md.md",
		"# Unrelated decoy\n\nnot the note you wanted\n")
	write("Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md", `# Pattern zoo

| Name | Sighting |
|---|---|
| with .md | [[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md#Identity is an API decision]] |
| without .md | [[Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis#Identity is an API decision]] |
`)

	v, err := vault.New(root)
	if err != nil {
		panic(err)
	}

	fmt.Println("== slugs in vault ==")
	for _, n := range v.AllNotes() {
		fmt.Printf("  %s\n", n.Slug)
	}

	targets := []string{
		"Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis.md",
		"Transcripts/2026/08/06/RAG DSL for Retrieval/rag-ttc-p01-p03-doctoral-thesis",
	}
	fmt.Println("\n== ResolveWikiLink ==")
	for _, t := range targets {
		got, ok := v.ResolveWikiLink(t)
		fmt.Printf("  target=%q\n    slugify=%q\n    resolved=%q ok=%v\n", t, parser.Slugify(t), got, ok)
	}

	n, ok := v.GetNote("transcripts/research/09-rag-maths-pattern-zoo")
	if !ok {
		fmt.Println("\n!! pattern-zoo note missing")
		return
	}
	fmt.Println("\n== rendered HTML ==")
	fmt.Println(n.HTML)
}
