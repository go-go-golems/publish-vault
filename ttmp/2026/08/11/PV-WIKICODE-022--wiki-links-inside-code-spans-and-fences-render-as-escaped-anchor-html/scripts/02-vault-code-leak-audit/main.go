// Command 02-vault-code-leak-audit renders every note in a vault and counts the
// ones whose output contains anchor markup escaped into a code element — the
// signature of a wiki link that was substituted inside a code sample.
//
// It also counts the phantom entries a code sample contributes to WikiLinks,
// which is what gives a note a backlink from a note that only mentioned it in
// an example.
//
// Parses notes directly rather than loading a Vault: nothing here depends on
// cross-note state, and a full load of a large vault is its own memory story
// (PV-MEMORY-019).
//
//	go run ./ttmp/2026/08/11/PV-WIKICODE-022--*/scripts/02-vault-code-leak-audit \
//	  -vault /home/manuel/code/wesen/go-go-golems/go-go-parc
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

// Anchor or embed markup that has been escaped into a code element.
//
// Matching this alone is not enough: a note explaining the renderer may quote
// the very same markup in an ```html block on purpose, and that output is
// correct. To separate the two, each note is parsed twice — once as written,
// and once with every "[[" neutralised so no wiki link can match. Markup that
// appears in both renderings was written by the author; markup that appears
// only in the first was injected by the pre-pass, which is the defect.
var leakRe = regexp.MustCompile(`&lt;(a href=&quot;/note/|img class=&quot;wiki-embed-image|div class=&quot;wiki-embed)`)

// neutralised stands in for "[[" while measuring the author-written baseline.
// It is not Markdown-active and cannot open a wiki link.
var neutralised = []byte("\u27e6\u27e6")

func leakCount(body []byte) int {
	parsed, err := parser.Parse(body)
	if err != nil {
		return 0
	}
	return len(leakRe.FindAllString(parsed.HTML, -1))
}

func main() {
	vaultRoot := flag.String("vault", "", "path to the vault to audit")
	limit := flag.Int("show", 15, "how many affected notes to list")
	flag.Parse()
	if *vaultRoot == "" {
		flag.Usage()
		os.Exit(2)
	}

	type hit struct {
		path  string
		leaks int
	}
	var hits []hit
	notes, leaks := 0, 0

	err := filepath.Walk(*vaultRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(info.Name()), ".md") {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		notes++
		n := leakCount(body) - leakCount(bytes.ReplaceAll(body, []byte("[["), neutralised))
		if n > 0 {
			leaks += n
			rel, _ := filepath.Rel(*vaultRoot, p)
			hits = append(hits, hit{rel, n})
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "walk:", err)
		os.Exit(1)
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].leaks > hits[j].leaks })

	fmt.Printf("vault:              %s\n", *vaultRoot)
	fmt.Printf("notes parsed:       %d\n", notes)
	fmt.Printf("injected markup:    %d occurrences across %d notes\n", leaks, len(hits))
	fmt.Printf("                    (author-written HTML samples are excluded by the two-pass baseline)\n")
	for i, h := range hits {
		if i >= *limit {
			fmt.Printf("                    … and %d more\n", len(hits)-*limit)
			break
		}
		fmt.Printf("  %4d  %s\n", h.leaks, h.path)
	}
	if leaks > 0 {
		os.Exit(1)
	}
}
