// Command 01-goldmark-frontmatter-contract observes which dash-only delimiter
// forms goldmark-meta recognizes in publish-vault's configured Parse pipeline.
package main

import (
	"fmt"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

func main() {
	cases := []struct {
		name, open, close, newline string
	}{
		{"one dash", "-", "-", "\n"},
		{"two dashes", "--", "--", "\n"},
		{"three dashes", "---", "---", "\n"},
		{"four dashes", "----", "----", "\n"},
		{"whitespace wrapped", "  ----  ", " \t----\t ", "\n"},
		{"four dashes CRLF", "----", "----", "\r\n"},
	}

	for _, tc := range cases {
		src := tc.open + tc.newline +
			"title: Boundary" + tc.newline +
			"marker: preserved" + tc.newline +
			tc.close + tc.newline +
			"Body" + tc.newline
		note, err := parser.Parse([]byte(src))
		if err != nil {
			fmt.Printf("%-20s error=%v\n", tc.name, err)
			continue
		}
		fmt.Printf("%-20s title=%q marker=%q html=%q\n",
			tc.name, note.Title, note.Frontmatter["marker"], note.HTML)
	}
}
