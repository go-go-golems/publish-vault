package commands

import (
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	buildcmd "retro-obsidian-publish/backend/cmd/retro-obsidian-publish/commands/build"
	servecmd "retro-obsidian-publish/backend/cmd/retro-obsidian-publish/commands/serve"
)

// NewRootCommand builds the top-level command tree. Subdirectories mirror the
// command verbs: serve lives in commands/serve and build web lives in
// commands/build/web.go.
func NewRootCommand() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "retro-obsidian-publish",
		Short: "Publish an Obsidian vault with a retro web UI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	if err := logging.AddLoggingSectionToRootCommand(rootCmd, "retro-obsidian-publish"); err != nil {
		return nil, err
	}

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	serveCommand, err := servecmd.NewCommand()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(serveCommand)

	buildCommand, err := buildcmd.NewCommand()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(buildCommand)

	return rootCmd, nil
}
