// Command 01-parser-edge-probe records edge behavior of the current Markdown
// and wiki-link pipeline for PV-MARKDOWN-023. It is an observation tool, not a
// proposed parser implementation.
package main

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

func main() {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "four-dash frontmatter delimiter",
			src:  "----\ntitle: Four Dashes\nrelated: '[[Meta Link]]'\n----\nBody [[Body Link]] with $x$.\n",
		},
		{
			name: "same target, distinct headings",
			src:  "[[Note#One]] and [[Note#Two]]\n",
		},
		{
			name: "same target as link and embed",
			src:  "[[Note]] and ![[Note]]\n",
		},
		{
			name: "wiki link crossing a line boundary",
			src:  "Before [[Note\ncontinued]] after.\n",
		},
		{
			name: "HTML in alias display",
			src:  "[[Note|<em data-probe=\"yes\">Alias</em>]]\n",
		},
	}

	for _, tc := range cases {
		parsed, err := parser.Parse([]byte(tc.src))
		fmt.Printf("\n=== %s ===\n", tc.name)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}
		fmt.Printf("title=%q frontmatter=%#v\n", parsed.Title, parsed.Frontmatter)
		fmt.Printf("links=%#v\n", parsed.WikiLinks)
		fmt.Printf("html=%s\n", strings.TrimSpace(parsed.HTML))
	}

	fmt.Println("\n=== broad HTML rewrite scope ===")
	in := `<div data-target="short">unrelated</div><a href="/note/short">plain link</a>`
	out := parser.ReplaceWikiLinksString(in, func(target string) string {
		if target == "short" {
			return "resolved/full"
		}
		return ""
	})
	fmt.Printf("before=%s\nafter =%s\n", in, out)
}
