// Command 04-slug-algebra prints the exact output of parser.Slugify for a table
// of representative inputs, so the design doc quotes measured behaviour instead
// of a reading of the regexes.
//
// Usage: go run ./publish-vault/ttmp/2026/08/09/PV-SLUG-020--*/scripts/04-slug-algebra
package main

import (
	"fmt"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

func main() {
	cases := []struct{ label, in string }{
		{"the real note path", "Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields"},
		{"plain lowercase", "notes/hello"},
		{"uppercase", "Notes/Hello World"},
		{"single space", "hello world"},
		{"repeated spaces", "hello    world"},
		{"underscore kept", "the_algebra_of_fields"},
		{"hyphen kept", "already-a-slug"},
		{"slash kept", "a/b/c"},
		{"leading/trailing space", "  padded  "},
		{"leading/trailing dash", "--edges--"},
		{"dot", "v1.2.3 release"},
		{"ampersand", "Cats & Dogs"},
		{"apostrophe", "Manuel's Notes"},
		{"colon in title", "Design: Part 1"},
		{"parens/brackets", "Branch (copy) [6a785ead]"},
		{"plus and hash", "C++ #tips"},
		{"unicode accents", "Café Münster"},
		{"cyrillic", "Привет мир"},
		{"cjk", "日本語ノート"},
		{"emoji", "done ✅ shipped 🚀"},
		{"percent + encoded", "50% off %20 test"},
		{"trailing slash", "a/b/"},
		{"double slash", "a//b"},
		{"only punctuation", "!!!"},
		{"empty string", ""},
		{"nbsp (U+00A0)", "hello world"},
		{"tab", "hello\tworld"},
	}

	fmt.Printf("%-24s | %-52s | %s\n", "CASE", "INPUT", "Slugify(INPUT)")
	fmt.Printf("%-24s-+-%-52s-+-%s\n", "------------------------", "----------------------------------------------------", "--------------")
	for _, c := range cases {
		fmt.Printf("%-24s | %-52q | %q\n", c.label, c.in, parser.Slugify(c.in))
	}

	fmt.Println()
	fmt.Println("Round-trip check: does Slugify(Slugify(x)) == Slugify(x)? (idempotence)")
	for _, c := range cases {
		once := parser.Slugify(c.in)
		twice := parser.Slugify(once)
		if once != twice {
			fmt.Printf("  NOT IDEMPOTENT: %q -> %q -> %q\n", c.in, once, twice)
		}
	}
	fmt.Println("  (no output above means Slugify is idempotent for these inputs)")

	fmt.Println()
	fmt.Println("Collision check: distinct inputs that produce the SAME slug")
	seen := map[string]string{}
	collide := []string{
		"Designing RAG Abstractions", "Designing-RAG-Abstractions", "Designing/RAG Abstractions",
		"Designing_RAG_Abstractions",
		"Cats & Dogs", "Cats and Dogs", "Cats   Dogs", "Cats+Dogs", "Cats!Dogs",
		"Café", "Cafe", "C-a-f-e",
	}
	for _, in := range collide {
		s := parser.Slugify(in)
		if prev, ok := seen[s]; ok {
			fmt.Printf("  COLLISION: %-30q and %-30q both -> %q\n", prev, in, s)
		} else {
			seen[s] = in
		}
	}
}
