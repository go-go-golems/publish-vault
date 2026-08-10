// Command 05-real-self-heading-check reports how the [[#Heading]] links in a
// real note render: how many reach a heading that exists in the rendered HTML,
// and how many are left visibly broken (and which).
//
// Unlike 02-real-note-check this needs no vault at all — same-note resolution
// depends only on the note itself, which is the whole reason it can live in
// Parse rather than in the vault layer.
//
//	go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/05-real-self-heading-check \
//	  -note "/home/manuel/code/wesen/go-go-golems/go-go-parc/Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

var (
	selfRe   = regexp.MustCompile(`<a href="(#[^"]*)" class="wiki-link wiki-link-self( broken)?" data-heading="([^"]*)"`)
	legacyRe = regexp.MustCompile(`<a href="/note/#[^"]*"`)
	emptyRe  = regexp.MustCompile(`class="wiki-link[^"]*"[^>]*></a>`)
)

func main() {
	notePath := flag.String("note", "", "absolute path of the note to check")
	flag.Parse()
	if *notePath == "" {
		flag.Usage()
		os.Exit(2)
	}

	src, err := os.ReadFile(*notePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read note:", err)
		os.Exit(1)
	}
	parsed, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse note:", err)
		os.Exit(1)
	}

	resolved, broken := 0, []string{}
	for _, m := range selfRe.FindAllStringSubmatch(parsed.HTML, -1) {
		if m[2] != "" {
			broken = append(broken, m[3])
			continue
		}
		resolved++
	}

	fmt.Printf("note:            %s\n", *notePath)
	fmt.Printf("self links:      %d\n", resolved+len(broken))
	fmt.Printf("  resolved:      %d (href points at a heading id present in the rendered HTML)\n", resolved)
	fmt.Printf("  broken:        %d\n", len(broken))
	for _, b := range broken {
		fmt.Printf("                   %s\n", b)
	}
	fmt.Printf("legacy /note/#:  %d (the old empty-slug form; must be 0)\n", len(legacyRe.FindAllString(parsed.HTML, -1)))
	fmt.Printf("empty anchors:   %d (wiki links rendering with no text; must be 0)\n", len(emptyRe.FindAllString(parsed.HTML, -1)))

	// Every resolved fragment must actually exist as an id in this document.
	idRe := regexp.MustCompile(`\bid="([^"]*)"`)
	ids := map[string]bool{}
	for _, m := range idRe.FindAllStringSubmatch(parsed.HTML, -1) {
		ids[m[1]] = true
	}
	dangling := 0
	for _, m := range selfRe.FindAllStringSubmatch(parsed.HTML, -1) {
		if m[2] != "" {
			continue
		}
		if !ids[m[1][1:]] {
			fmt.Printf("DANGLING:        %s\n", m[1])
			dangling++
		}
	}
	fmt.Printf("dangling:        %d (resolved links whose target id is absent; must be 0)\n", dangling)
	if dangling > 0 {
		os.Exit(1)
	}
}
