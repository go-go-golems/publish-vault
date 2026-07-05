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
	"github.com/spf13/cobra"

	appserver "retro-obsidian-publish/internal/server"
)

// Command serves the API and optionally the bundled web app.
type Command struct {
	*cmds.CommandDescription
}

// Settings are decoded from the Glazed default command section.
type Settings struct {
	Vault               string `glazed:"vault"`
	Port                string `glazed:"port"`
	VaultName           string `glazed:"vault-name"`
	PageTitle           string `glazed:"page-title"`
	ServeWeb            bool   `glazed:"serve-web"`
	Watch               bool   `glazed:"watch"`
	ReloadTokenEnv      string `glazed:"reload-token-env"`
	ReloadAllowLoopback bool   `glazed:"reload-allow-loopback"`
	SSRURL              string `glazed:"ssr-url"`
	Favicon             string `glazed:"favicon"`
	SearchIndexPath     string `glazed:"search-index-path"`
}

// NewCommand creates the Cobra command for the serve verb.
func NewCommand() (*cobra.Command, error) {
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmd := &Command{CommandDescription: cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve an Obsidian vault as a retro web app"),
		cmds.WithLong(`Serve scans an Obsidian vault, builds a search index, watches Markdown files, exposes the /api JSON routes, and can serve the bundled React SPA from the same Go process.

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
			fields.New("vault-name", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Display name for the vault in the web UI. Defaults to the vault directory basename."),
			),
			fields.New("page-title", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Browser page title returned by /api/config. Defaults to --vault-name or the vault directory basename."),
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
			fields.New("ssr-url", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("URL of the SSR sidecar (e.g. http://localhost:8089). When set, page requests are reverse-proxied to the SSR server for server-side rendering. When empty, the SPA fallback serves index.html directly."),
			),
			fields.New("favicon", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to a favicon file (.ico or .svg). When set, overrides vault-root lookup. When empty, the server looks for favicon.ico and favicon.svg in the vault root directory."),
			),
			fields.New("search-index-path", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Optional base directory for per-snapshot persistent bleve indexes. When empty, search stays in memory."),
			),
		),
		cmds.WithSections(commandSettingsSection),
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
		settings.Vault, _ = os.LookupEnv("VAULT_DIR")
	}
	if settings.Vault == "" {
		return fmt.Errorf("--vault or VAULT_DIR is required")
	}
	reloadToken := ""
	if settings.ReloadTokenEnv != "" {
		reloadToken, _ = os.LookupEnv(settings.ReloadTokenEnv)
	}
	return appserver.Run(ctx, appserver.Config{VaultDir: settings.Vault, Port: settings.Port, VaultName: settings.VaultName, PageTitle: settings.PageTitle, ServeWeb: settings.ServeWeb, Watch: settings.Watch, ReloadToken: reloadToken, ReloadAllowLoopback: settings.ReloadAllowLoopback, SSRURL: settings.SSRURL, FaviconPath: settings.Favicon, SearchIndexPath: settings.SearchIndexPath})
}
