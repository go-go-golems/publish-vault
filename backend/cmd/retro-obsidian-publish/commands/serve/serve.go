package serve

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
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/spf13/cobra"

	appserver "retro-obsidian-publish/backend/internal/server"
)

// Command serves the API and optionally the bundled web app.
type Command struct {
	*cmds.CommandDescription
}

// Settings are decoded from the Glazed default command section.
type Settings struct {
	Vault               string `glazed:"vault"`
	Port                string `glazed:"port"`
	ServeWeb            bool   `glazed:"serve-web"`
	Watch               bool   `glazed:"watch"`
	ReloadTokenEnv      string `glazed:"reload-token-env"`
	ReloadAllowLoopback bool   `glazed:"reload-allow-loopback"`
}

// NewCommand creates the Cobra command for the serve verb.
func NewCommand() (*cobra.Command, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmd := &Command{CommandDescription: cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve an Obsidian vault as a retro web app"),
		cmds.WithLong(`Serve scans an Obsidian vault, builds an in-memory search index, watches Markdown files, exposes the /api JSON routes, and can serve the bundled React SPA from the same Go process.

Examples:
  retro-obsidian-publish serve --vault ./vault-example --port 8080
  retro-obsidian-publish serve --vault ./vault-example --port 8080 --serve-web
  VAULT_DIR=/path/to/vault retro-obsidian-publish serve --serve-web
  RETRO_RELOAD_TOKEN=secret retro-obsidian-publish serve --vault /git/root/current --watch=false
  retro-obsidian-publish serve --vault /git/root/current --watch=false --reload-allow-loopback
`),
		cmds.WithFlags(
			fields.New("vault", fields.TypeString,
				fields.WithHelp("Path to an Obsidian vault directory. Defaults to VAULT_DIR when omitted."),
			),
			fields.New("port", fields.TypeString,
				fields.WithDefault("8080"),
				fields.WithHelp("HTTP port for the backend API server."),
			),
			fields.New("serve-web", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Serve the bundled web SPA from the same Go process."),
			),
			fields.New("watch", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Watch Markdown files with fsnotify. Disable in git-sync deployments and use the reload endpoint instead."),
			),
			fields.New("reload-token-env", fields.TypeString,
				fields.WithDefault("RETRO_RELOAD_TOKEN"),
				fields.WithHelp("Environment variable containing the bearer token for POST /api/admin/reload. Empty value disables token auth for the reload endpoint."),
			),
			fields.New("reload-allow-loopback", fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Allow POST /api/admin/reload without a bearer token from loopback clients such as a git-sync sidecar calling 127.0.0.1."),
			),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)}

	return cli.BuildCobraCommandFromCommand(cmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
}

// RunIntoGlazeProcessor runs the long-lived server. It intentionally does not
// emit structured rows; Glazed is used here for schema-backed flags, sections,
// help, logging, and command introspection.
func (c *Command) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, _ middlewares.Processor) error {
	settings := &Settings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	if settings.Vault == "" {
		settings.Vault = os.Getenv("VAULT_DIR")
	}
	if settings.Vault == "" {
		return fmt.Errorf("--vault or VAULT_DIR is required")
	}
	reloadToken := ""
	if settings.ReloadTokenEnv != "" {
		reloadToken = os.Getenv(settings.ReloadTokenEnv)
	}
	return appserver.Run(ctx, appserver.Config{VaultDir: settings.Vault, Port: settings.Port, ServeWeb: settings.ServeWeb, Watch: settings.Watch, ReloadToken: reloadToken, ReloadAllowLoopback: settings.ReloadAllowLoopback})
}
