// Command 06-cross-note-fragment-audit measures the *remaining* fragment
// problem, which this ticket does not fix: [[Other Note#Heading]] computes its
// fragment with slugify, while the target note's heading ids come from
// goldmark. Where the two algorithms disagree the link opens the right note at
// the top of the page instead of at the heading.
//
// It parses a note, renders each linked target, and reports how many fragments
// name an id that does not exist in the target.
//
//	go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/06-cross-note-fragment-audit \
//	  -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
//	  -note "Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

var idRe = regexp.MustCompile(`\bid="([^"]*)"`)

func main() {
	vaultRoot := flag.String("vault", "", "path to the source vault")
	notePath := flag.String("note", "", "vault-relative path of the note to audit")
	flag.Parse()
	if *vaultRoot == "" || *notePath == "" {
		flag.Usage()
		os.Exit(2)
	}

	src, err := os.ReadFile(filepath.Join(*vaultRoot, filepath.FromSlash(*notePath)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "read note:", err)
		os.Exit(1)
	}
	parsed, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse note:", err)
		os.Exit(1)
	}

	// Cache each target's heading ids; one target carries many links.
	idsFor := map[string]map[string]bool{}
	loadIDs := func(target string) (map[string]bool, bool) {
		if ids, ok := idsFor[target]; ok {
			return ids, ids != nil
		}
		body, err := os.ReadFile(filepath.Join(*vaultRoot, filepath.FromSlash(target+".md")))
		if err != nil {
			idsFor[target] = nil
			return nil, false
		}
		p, err := parser.Parse(body)
		if err != nil {
			idsFor[target] = nil
			return nil, false
		}
		ids := map[string]bool{}
		for _, m := range idRe.FindAllStringSubmatch(p.HTML, -1) {
			ids[m[1]] = true
		}
		idsFor[target] = ids
		return ids, true
	}

	withHeading, dangling, unreadable := 0, map[string]string{}, 0
	for _, wl := range parsed.WikiLinks {
		if wl.Heading == "" || wl.IsEmbed {
			continue
		}
		withHeading++
		ids, ok := loadIDs(wl.Target)
		if !ok {
			unreadable++
			continue
		}
		want := parser.Slugify(wl.Heading)
		if !ids[want] {
			dangling[wl.Target+"#"+wl.Heading] = want
		}
	}

	fmt.Printf("note:                 %s\n", *notePath)
	fmt.Printf("cross-note links with a #Heading: %d\n", withHeading)
	fmt.Printf("  target unreadable:  %d\n", unreadable)
	fmt.Printf("  fragment dangles:   %d (opens the right note, at the top of the page)\n", len(dangling))
	keys := make([]string, 0, len(dangling))
	for k := range dangling {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("      %s\n        wiki link asks for #%s\n", k, dangling[k])
	}
}
