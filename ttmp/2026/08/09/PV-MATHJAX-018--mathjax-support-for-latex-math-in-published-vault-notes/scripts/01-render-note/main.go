// 01-render-note renders a single Markdown file through the real parser and
// prints the HTML, so math placeholder emission can be eyeballed without
// starting a server. Used to find the bug where "$30 … `$` …" closed inline
// math inside a code span.
//
//	go run ./ttmp/2026/08/09/PV-MATHJAX-018--*/scripts/01-render-note <file.md>
package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: 01-render-note <file.md>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	note, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(note.HTML)
}
