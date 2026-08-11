// Command 06-cross-note-fragment-audit checks that every [[Note#Heading]] link
// in a real note ends up pointing at a heading id that actually exists in the
// note it links to.
//
// It reads the *rendered* href rather than recomputing it, so it measures what
// the reader gets. It stages the note and its link targets into a temp vault —
// the fragment is resolved in the vault layer, and loading go-go-parc in full is
// its own memory story (PV-MEMORY-019).
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
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

var (
	idRe   = regexp.MustCompile(`\bid="([^"]*)"`)
	hrefRe = regexp.MustCompile(`<a href="/note/([^"#]*)(#[^"]*)?" class="wiki-link" data-target="[^"]*" data-raw="([^"]*)" data-heading="([^"]*)"`)
	// The pre-fix anchor carried no data-heading at all; match it so the same
	// script can measure the "before" state after a git checkout of the parser.
	legacyHrefRe = regexp.MustCompile(`<a href="/note/([^"#]*)(#[^"]*)?" class="wiki-link" data-target="[^"]*" data-raw="([^"]*)" data-alias=`)
)

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

	tmp, err := os.MkdirTemp("", "fragaudit")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	copyIn := func(rel string) bool {
		body, err := os.ReadFile(filepath.Join(*vaultRoot, filepath.FromSlash(rel)))
		if err != nil {
			return false
		}
		to := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(to, body, 0o644); err != nil {
			panic(err)
		}
		return true
	}
	if !copyIn(*notePath) {
		fmt.Fprintln(os.Stderr, "could not stage the note itself")
		os.Exit(1)
	}
	for _, wl := range parsed.WikiLinks {
		copyIn(wl.Target + ".md")
		copyIn(wl.Target) // pre-fix targets still carry their own ".md"
	}

	v, err := vault.New(tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load temp vault:", err)
		os.Exit(1)
	}
	note, ok := v.GetNote(parser.Slugify(strings.TrimSuffix(*notePath, ".md")))
	if !ok {
		fmt.Fprintln(os.Stderr, "note missing from temp vault")
		os.Exit(1)
	}

	// Every id present in each target note's rendered HTML.
	idsFor := map[string]map[string]bool{}
	for _, n := range v.AllNotes() {
		ids := map[string]bool{}
		for _, m := range idRe.FindAllStringSubmatch(n.HTML, -1) {
			ids[m[1]] = true
		}
		idsFor[n.Slug] = ids
	}

	matches := hrefRe.FindAllStringSubmatch(note.HTML, -1)
	legacy := false
	if len(matches) == 0 {
		matches = legacyHrefRe.FindAllStringSubmatch(note.HTML, -1)
		legacy = true
	}

	withFragment, dangling, dropped := 0, map[string]string{}, 0
	for _, m := range matches {
		slug, frag, raw := m[1], m[2], m[3]
		heading := ""
		if !legacy && len(m) > 4 {
			heading = m[4]
		}
		if frag == "" {
			if heading != "" {
				dropped++
			}
			continue
		}
		withFragment++
		if !idsFor[slug][frag[1:]] {
			dangling[raw+" -> "+slug+frag] = heading
		}
	}

	if legacy {
		fmt.Println("(pre-fix markup: anchors carry no data-heading)")
	}
	fmt.Printf("note:                        %s\n", *notePath)
	fmt.Printf("cross-note links w/ fragment: %d\n", withFragment)
	fmt.Printf("  fragment dangles:           %d (opens the right note, at the top of the page)\n", len(dangling))
	fmt.Printf("  fragment dropped:           %d (heading not found in the target; link still opens the note)\n", dropped)
	keys := make([]string, 0, len(dangling))
	for k := range dangling {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("      %s\n", k)
	}
	if len(dangling) > 0 {
		os.Exit(1)
	}
}
