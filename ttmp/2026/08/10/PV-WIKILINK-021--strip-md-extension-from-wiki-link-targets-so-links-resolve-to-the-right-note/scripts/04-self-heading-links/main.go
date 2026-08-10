// Command self exercises [[#Heading]] resolution end-to-end through the vault,
// so the rebuildHTML passes (which re-render from the parser output on every
// reload) are proven not to undo it.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/publish-vault/pkg/vault"
)

func main() {
	root, err := os.MkdirTemp("", "selfheading")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	body := `# Zoo

1. [[#Pattern 1 — Semantic Identity as Explicit Projection]]
2. [[#9.2 Kernel K0: canonical identity]]
3. [[#Notes]] and [[#Notes|second one?]]
4. [[#no such heading]]
5. [[Other#9.2 Kernel K0: canonical identity]]

## Pattern 1 — Semantic Identity as Explicit Projection

## 9.2 Kernel K0: canonical identity

## Notes

## Notes
`
	write := func(rel, b string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			panic(err)
		}
	}
	write("Zoo.md", body)
	write("Other.md", "# Other\n\n## 9.2 Kernel K0: canonical identity\n")

	v, err := vault.New(root)
	if err != nil {
		panic(err)
	}
	note, ok := v.GetNote("zoo")
	if !ok {
		panic("zoo missing")
	}
	fmt.Println(note.HTML)
	fmt.Printf("\nWikiLinks: %#v\n", note.WikiLinks)
}
