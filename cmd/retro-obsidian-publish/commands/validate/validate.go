// Package validate implements the read-only vault validation command.
package validate

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	check "github.com/go-go-golems/publish-vault/internal/validate"
)

type Command struct{ *cmds.CommandDescription }
type Settings struct {
	Vault  string `glazed:"vault"`
	Config string `glazed:"config"`
	Strict bool   `glazed:"strict"`
}

func NewCommand() (*cobra.Command, error) {
	c := &Command{cmds.NewCommandDescription("validate", cmds.WithShort("Check published vault wiki links and embeds"), cmds.WithLong("Validate body wiki links and image embeds in the published vault view. Frontmatter references and Markdown images are not checked. The vault is never modified."), cmds.WithFlags(
		fields.New("vault", fields.TypeString, fields.WithHelp("Vault directory; defaults to VAULT_DIR.")),
		fields.New("config", fields.TypeString, fields.WithHelp("Publish config path; defaults to <vault>/.publish/config.yaml.")),
		fields.New("strict", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Exit nonzero when warnings are found.")),
	))}
	return cli.BuildCobraCommandFromCommand(c, cli.WithParserConfig(cli.CobraParserConfig{ShortHelpSections: []string{schema.DefaultSlug}}))
}

func (c *Command) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, processor middlewares.Processor) error {
	var s Settings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	if s.Vault == "" {
		s.Vault, _ = os.LookupEnv("VAULT_DIR")
	}
	if s.Vault == "" {
		return fmt.Errorf("--vault or VAULT_DIR is required")
	}
	diagnostics, err := check.Run(ctx, s.Vault, s.Config)
	if err != nil {
		return err
	}
	for _, d := range diagnostics {
		if err := processor.AddRow(ctx, types.NewRow(types.MRP("rule", d.Rule), types.MRP("severity", d.Severity), types.MRP("path", d.Path), types.MRP("line", d.Line), types.MRP("column", d.Column), types.MRP("message", d.Message), types.MRP("target", d.Target))); err != nil {
			return err
		}
	}
	if s.Strict && len(diagnostics) > 0 {
		return fmt.Errorf("validation found %d warning(s)", len(diagnostics))
	}
	return nil
}
