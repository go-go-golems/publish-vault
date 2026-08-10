// Command 05-vault-slug-audit walks a real vault and reports every way a note
// on disk can fail to be reachable at /note/<slug>:
//
//	1. the file is on disk but not in v.notes  (excluded, publish:false, or a
//	   parse error that LoadAll silently swallows)
//	2. two different files slugify to the same key (last writer wins; the other
//	   note is permanently unreachable)
//	3. the slug is empty (non-Latin filenames collapse to "")
//
// Usage:
//
//	go run ./publish-vault/ttmp/2026/08/09/PV-SLUG-020--*/scripts/05-vault-slug-audit \
//	    -vault /home/manuel/code/wesen/go-go-golems/go-go-parc
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	vaultPath := flag.String("vault", "", "absolute path to the vault root")
	limit := flag.Int("limit", 25, "max rows to print per section")
	flag.Parse()
	if *vaultPath == "" {
		fmt.Fprintln(os.Stderr, "-vault is required")
		os.Exit(2)
	}

	v, err := vault.New(*vaultPath)
	if err != nil {
		fmt.Printf("vault.New FAILED: %v\n", err)
		os.Exit(1)
	}

	indexed := map[string]string{} // slug -> relPath
	for _, n := range v.AllNotes() {
		indexed[n.Slug] = n.Path
	}

	// Walk the disk the same way LoadAll does, but without any of its filters,
	// so we can diff "markdown on disk" against "markdown reachable by URL".
	type diskNote struct{ rel, slug string }
	var disk []diskNote
	bySlug := map[string][]string{}
	_ = filepath.Walk(*vaultPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && p != *vaultPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(*vaultPath, p)
		slug := parser.Slugify(strings.TrimSuffix(filepath.ToSlash(rel), ".md"))
		disk = append(disk, diskNote{rel: rel, slug: slug})
		bySlug[slug] = append(bySlug[slug], rel)
		return nil
	})

	fmt.Printf("vault root          : %s\n", *vaultPath)
	fmt.Printf("markdown files on disk (excluding dot-dirs): %d\n", len(disk))
	fmt.Printf("notes indexed by vault.LoadAll            : %d\n", v.Count())
	fmt.Printf("difference (unreachable at /note/<slug>)  : %d\n\n", len(disk)-v.Count())

	// --- 1. on disk but not served ---
	var missing []diskNote
	for _, d := range disk {
		if _, ok := indexed[d.slug]; !ok {
			missing = append(missing, d)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].rel < missing[j].rel })
	fmt.Printf("== on disk but NOT reachable (%d) ==\n", len(missing))
	for i, m := range missing {
		if i >= *limit {
			fmt.Printf("  ... and %d more\n", len(missing)-*limit)
			break
		}
		abs := filepath.Join(*vaultPath, m.rel)
		fmt.Printf("  slug=%-45q excluded=%-5v path=%s\n", m.slug, v.IsExcluded(abs, false), m.rel)
	}
	fmt.Println()

	// --- 2. slug collisions ---
	var collisions []string
	for slug, paths := range bySlug {
		if len(paths) > 1 {
			sort.Strings(paths)
			collisions = append(collisions, fmt.Sprintf("  slug=%q (%d files)\n      %s", slug, len(paths), strings.Join(paths, "\n      ")))
		}
	}
	sort.Strings(collisions)
	fmt.Printf("== slug collisions: distinct files mapping to one slug (%d slugs) ==\n", len(collisions))
	for i, c := range collisions {
		if i >= *limit {
			fmt.Printf("  ... and %d more\n", len(collisions)-*limit)
			break
		}
		fmt.Println(c)
	}
	fmt.Println()

	// --- 3. empty slugs ---
	fmt.Printf("== files whose slug is the empty string (%d) ==\n", len(bySlug[""]))
	for i, p := range bySlug[""] {
		if i >= *limit {
			fmt.Printf("  ... and %d more\n", len(bySlug[""])-*limit)
			break
		}
		fmt.Printf("  %s\n", p)
	}
}
