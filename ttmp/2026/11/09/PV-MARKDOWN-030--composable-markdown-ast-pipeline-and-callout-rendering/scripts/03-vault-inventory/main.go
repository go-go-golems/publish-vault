// Command 03-vault-inventory produces a content-free JSON summary of parsed
// note features for comparing two implementations on the same vault.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

type inventory struct {
	Links       int `json:"links"`
	Backlinks   int `json:"backlinks"`
	InlineMath  int `json:"inline_math"`
	DisplayMath int `json:"display_math"`
	Callouts    int `json:"callouts"`
	Images      int `json:"images"`
	Broken      int `json:"broken"`
}

func main() {
	root := flag.String("vault", "", "vault directory")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "-vault is required")
		os.Exit(2)
	}
	v, err := vault.New(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result := make(map[string]inventory)
	for _, n := range v.AllNotes() {
		result[n.Slug] = inventory{
			Links: len(n.WikiLinks), Backlinks: len(n.Backlinks),
			InlineMath:  strings.Count(n.HTML, `class="math math-inline"`),
			DisplayMath: strings.Count(n.HTML, `class="math math-display"`),
			Callouts:    strings.Count(n.HTML, `class="callout callout-`),
			Images:      strings.Count(n.HTML, "<img "),
			Broken:      strings.Count(n.HTML, "wiki-embed-broken") + strings.Count(n.HTML, "wiki-link-self broken"),
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
