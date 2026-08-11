// Command 02-real-note-check validates the fix against a real note from a real
// vault, without loading the whole vault (go-go-parc is large enough that a full
// load is its own memory story — see PV-MEMORY-019).
//
// It parses the note, walks its wiki links, copies each existing target out of
// the source vault into a temp vault alongside the note, then loads that temp
// vault and counts how many links still render as "#unresolved-".
//
//	go run ./ttmp/2026/08/10/PV-WIKILINK-021--*/scripts/02-real-note-check \
//	  -vault /home/manuel/code/wesen/go-go-golems/go-go-parc \
//	  -note "Transcripts/Research/09 - RAG-MATHS Pattern Zoo.md"
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	vaultRoot := flag.String("vault", "", "path to the source vault")
	notePath := flag.String("note", "", "vault-relative path of the note to check")
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

	tmp, err := os.MkdirTemp("", "realnote")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	copyIn := func(rel string) bool {
		from := filepath.Join(*vaultRoot, filepath.FromSlash(rel))
		body, err := os.ReadFile(from)
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

	// Targets come out of the parser already stripped of ".md", so put it back
	// to find the file. Anything that is not a vault-relative path (bare titles,
	// links to notes living elsewhere) simply fails to copy and is reported.
	staged, missing := 0, []string{}
	for _, wl := range parsed.WikiLinks {
		if copyIn(wl.Target + ".md") {
			staged++
			continue
		}
		missing = append(missing, wl.Target)
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

	unresolved := regexp.MustCompile(`href="#unresolved-([^"]*)"`).FindAllStringSubmatch(note.HTML, -1)
	fmt.Printf("note:        %s\n", *notePath)
	fmt.Printf("wiki links:  %d\n", len(parsed.WikiLinks))
	fmt.Printf("staged:      %d target files copied into the temp vault\n", staged)
	fmt.Printf("not on disk: %d (link targets with no matching file, expected for cross-vault or bare-title links)\n", len(missing))
	for _, m := range missing {
		fmt.Printf("               %s\n", m)
	}
	fmt.Printf("unresolved:  %d links rendered as #unresolved-\n", len(unresolved))
	for _, m := range unresolved {
		fmt.Printf("               %s\n", m[1])
	}
	if len(unresolved) > 0 {
		os.Exit(1)
	}
}
