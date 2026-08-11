// Command 01-code-region-repro shows what a note documenting wiki-link syntax
// currently renders as: the [[...]] pre-pass rewrites the source before goldmark
// runs, so an example inside a code span or a fence becomes anchor HTML, and
// goldmark then escapes that markup into the code block as visible text.
//
//	go run ./ttmp/2026/08/11/PV-WIKICODE-022--*/scripts/01-code-region-repro
package main

import (
	"fmt"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

func main() {
	src := []byte("# Syntax\n\n" +
		"A note refers to another as `[[Some Note]]`, or with a heading:\n\n" +
		"```markdown\n" +
		"[[Target Note#Heading]]\n" +
		"![[Diagram.png]]\n" +
		"```\n\n" +
		"A real link: [[Some Note]].\n")

	parsed, err := parser.Parse(src)
	if err != nil {
		panic(err)
	}
	fmt.Println("== rendered HTML ==")
	fmt.Println(parsed.HTML)
	fmt.Println("== WikiLinks (feeds the backlink graph) ==")
	for _, wl := range parsed.WikiLinks {
		fmt.Printf("  target=%q heading=%q embed=%v\n", wl.Target, wl.Heading, wl.IsEmbed)
	}
}
