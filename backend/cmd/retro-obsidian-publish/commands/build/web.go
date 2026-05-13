package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/spf13/cobra"
)

const (
	defaultBuilderImage = "node:22"
	defaultPNPMVersion  = "10.4.1"
)

var errDaggerUnavailable = errors.New("dagger: engine not reachable")

// WebCommand builds the web frontend and copies it into internal/web/embed/public.
type WebCommand struct {
	*cmds.CommandDescription
}

type WebSettings struct {
	Local        bool   `glazed:"local"`
	BuilderImage string `glazed:"builder-image"`
	PNPMVersion  string `glazed:"pnpm-version"`
}

func NewWebCommand() (*cobra.Command, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmd := &WebCommand{CommandDescription: cmds.NewCommandDescription(
		"web",
		cmds.WithShort("Build the web frontend for embedding in the Go binary"),
		cmds.WithLong(`Build the React/Vite web app and copy web/dist into backend/internal/web/embed/public so the Go binary can be built with -tags embed.

Examples:
  retro-obsidian-publish build web
  BUILD_WEB_LOCAL=1 retro-obsidian-publish build web
  retro-obsidian-publish build web --local
`),
		cmds.WithFlags(
			fields.New("local", fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Use local pnpm instead of Dagger."),
			),
			fields.New("builder-image", fields.TypeString,
				fields.WithDefault(defaultBuilderImage),
				fields.WithHelp("Container image for Dagger builds."),
			),
			fields.New("pnpm-version", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("pnpm version to activate with corepack. Defaults to web/package.json packageManager."),
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

func (c *WebCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, _ middlewares.Processor) error {
	settings := &WebSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	if settings.PNPMVersion == "" {
		settings.PNPMVersion = readPNPMVersion(filepath.Join(repoRoot, "web"))
	}
	if settings.PNPMVersion == "" {
		settings.PNPMVersion = defaultPNPMVersion
	}
	settings.PNPMVersion = strings.Split(settings.PNPMVersion, "+")[0]

	if settings.Local || os.Getenv("BUILD_WEB_LOCAL") == "1" {
		return runLocal(repoRoot)
	}
	if err := runDagger(ctx, repoRoot, settings.BuilderImage, settings.PNPMVersion); err != nil {
		if errors.Is(err, errDaggerUnavailable) {
			fmt.Fprintln(os.Stderr, "dagger unavailable, falling back to local pnpm")
			return runLocal(repoRoot)
		}
		return err
	}
	return nil
}

func runDagger(ctx context.Context, repoRoot, builderImage, pnpmVersion string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("%w: %v", errDaggerUnavailable, err)
	}
	defer func() { _ = client.Close() }()

	webDir := filepath.Join(repoRoot, "web")
	source := client.Host().Directory(webDir, dagger.HostDirectoryOpts{Exclude: []string{"dist", "node_modules", ".git"}})
	pnpmStore := client.CacheVolume("retro-obsidian-publish-pnpm-store")
	pathEnv := "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	container := client.Container().
		From(builderImage).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", pathEnv).
		WithMountedCache("/pnpm/store", pnpmStore).
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-lc", "corepack enable && corepack prepare pnpm@" + pnpmVersion + " --activate"}).
		WithExec([]string{"pnpm", "config", "set", "store-dir", "/pnpm/store"}).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile", "--prefer-offline"}).
		WithExec([]string{"pnpm", "run", "build"})

	tmpDir, err := os.MkdirTemp("", "retro-obsidian-publish-web-dist-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if _, err := container.Directory("/src/dist").Export(ctx, tmpDir); err != nil {
		return fmt.Errorf("export dist: %w", err)
	}
	return copyDistToEmbed(repoRoot, tmpDir, "Dagger")
}

func runLocal(repoRoot string) error {
	if err := runCmd(filepath.Join(repoRoot, "web"), "pnpm", "install", "--frozen-lockfile"); err != nil {
		return fmt.Errorf("pnpm install (local): %w", err)
	}
	if err := runCmd(filepath.Join(repoRoot, "web"), "pnpm", "run", "build"); err != nil {
		return fmt.Errorf("pnpm build (local): %w", err)
	}
	return copyDistToEmbed(repoRoot, filepath.Join(repoRoot, "web", "dist"), "local pnpm")
}

func copyDistToEmbed(repoRoot, src, mode string) error {
	dst := filepath.Join(repoRoot, "backend", "internal", "web", "embed", "public")
	if err := recreate(dst); err != nil {
		return fmt.Errorf("recreate dst: %w", err)
	}
	if err := copyTree(src, dst); err != nil {
		return fmt.Errorf("copy to embed/public: %w", err)
	}
	log.Printf("Successfully exported web dist to %s (%s)", dst, mode)
	return nil
}

func readPNPMVersion(webDir string) string {
	data, err := os.ReadFile(filepath.Join(webDir, "package.json"))
	if err != nil {
		return ""
	}
	prefix := `"packageManager": "pnpm@`
	idx := strings.Index(string(data), prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := start
	for end < len(data) && data[end] != '"' {
		end++
	}
	return string(data[start:end])
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 12; i++ {
		if exists(filepath.Join(dir, "backend", "go.mod")) && exists(filepath.Join(dir, "web", "package.json")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func recreate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == ".gitkeep" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
