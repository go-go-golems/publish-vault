// Command 01-slug-probe loads a real vault with pkg/vault and reports whether
// the slug behind a failing /note/... URL resolves.
//
// Usage:
//
//	go run ./publish-vault/ttmp/2026/08/09/PV-SLUG-020--*/scripts/01-slug-probe \
//	    -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
//	    -slug transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields \
//	    -grep intervention
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	vaultPath := flag.String("vault", "", "absolute path to the vault root")
	slug := flag.String("slug", "", "slug to look up with GetNote")
	grep := flag.String("grep", "", "case-insensitive substring; dump every slug containing it")
	file := flag.String("file", "", "absolute path of a note file; report SlugForPath/IsExcluded for it")
	flag.Parse()

	if *vaultPath == "" {
		fmt.Fprintln(os.Stderr, "-vault is required")
		os.Exit(2)
	}

	fmt.Printf("== vault.New(%q) ==\n", *vaultPath)
	v, err := vault.New(*vaultPath)
	if err != nil {
		fmt.Printf("vault.New FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("loaded notes: %d\n\n", v.Count())

	if *file != "" {
		fmt.Printf("== SlugForPath / IsExcluded for %q ==\n", *file)
		_, statErr := os.Stat(*file)
		fmt.Printf("os.Stat err          : %v\n", statErr)
		fmt.Printf("SlugForPath          : %q\n", v.SlugForPath(*file))
		fmt.Printf("IsExcluded(file)     : %v\n", v.IsExcluded(*file, false))
		dir := filepath.Dir(*file)
		fmt.Printf("IsExcluded(parentDir): %v\n", v.IsExcluded(dir, true))
		fmt.Printf("ShouldPruneDir(parent): %v\n", v.ShouldPruneDir(dir))
		// Walk up to the vault root reporting exclusion for each ancestor.
		for d := dir; strings.HasPrefix(d, *vaultPath) && d != *vaultPath; d = filepath.Dir(d) {
			fmt.Printf("  ancestor %-70s excluded=%v prune=%v\n", d, v.IsExcluded(d, true), v.ShouldPruneDir(d))
		}
		fmt.Println()
	}

	if *slug != "" {
		fmt.Printf("== GetNote(%q) ==\n", *slug)
		n, ok := v.GetNote(*slug)
		fmt.Printf("found: %v\n", ok)
		if ok {
			fmt.Printf("  title   : %q\n", n.Title)
			fmt.Printf("  path    : %q\n", n.Path)
			fmt.Printf("  htmlLen : %d\n", len(n.HTML))
		}
		if wl, ok2 := v.ResolveWikiLink(*slug); ok2 {
			fmt.Printf("ResolveWikiLink -> %q\n", wl)
		} else {
			fmt.Printf("ResolveWikiLink -> (no match)\n")
		}
		fmt.Println()
	}

	if *grep != "" {
		needle := strings.ToLower(*grep)
		fmt.Printf("== slugs containing %q ==\n", needle)
		var got []string
		for _, n := range v.AllNotes() {
			if strings.Contains(strings.ToLower(n.Slug), needle) ||
				strings.Contains(strings.ToLower(n.Path), needle) {
				got = append(got, fmt.Sprintf("%s\n      path=%s", n.Slug, n.Path))
			}
		}
		sort.Strings(got)
		for _, g := range got {
			fmt.Printf("  %s\n", g)
		}
		fmt.Printf("(%d matches)\n\n", len(got))
	}
}
