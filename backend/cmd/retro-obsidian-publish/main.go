package main

import (
	"fmt"
	"os"

	"retro-obsidian-publish/backend/cmd/retro-obsidian-publish/commands"
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
