// Command 03-heading-id-divergence compares goldmark's auto heading IDs with
// parser.Slugify for real heading text, which is why a [[#Heading]] fragment has
// to be read back out of the rendered HTML rather than computed.
//
//	go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/03-heading-id-divergence
package main

import (
	"fmt"
	"regexp"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

var idRe = regexp.MustCompile(`<h[1-6] id="([^"]*)"`)

func main() {
	headings := []string{
		"Identity is an API decision",
		"Pattern 1 — Semantic Identity as Explicit Projection",
		"9.2 Kernel K0: canonical identity",
		"P01 - Semantic identity and cache fingerprints",
		"7.3 Domain-separated hashes",
		"13. Behavior identity and causal identity",
		"Entity–Derivation–Observation Separation",
	}
	for _, h := range headings {
		p, err := parser.Parse([]byte("## " + h + "\n"))
		if err != nil {
			panic(err)
		}
		m := idRe.FindStringSubmatch(p.HTML)
		got := ""
		if len(m) > 1 {
			got = m[1]
		}
		mark := "OK "
		if got != parser.Slugify(h) {
			mark = "!! "
		}
		fmt.Printf("%s%-55q goldmark=%-50q slugify=%q\n", mark, h, got, parser.Slugify(h))
	}
}
