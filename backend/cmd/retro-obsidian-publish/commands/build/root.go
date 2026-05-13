package build

import "github.com/spf13/cobra"

// NewCommand creates the build command group. Subcommands are files in this
// directory, following the CLI verb structure.
func NewCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build project artifacts",
	}

	webCmd, err := NewWebCommand()
	if err != nil {
		return nil, err
	}
	cmd.AddCommand(webCmd)
	return cmd, nil
}
