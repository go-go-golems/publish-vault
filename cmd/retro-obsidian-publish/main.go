package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/publish-vault/cmd/retro-obsidian-publish/commands"
)

func main() {
	rootCmd, err := commands.NewRootCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
