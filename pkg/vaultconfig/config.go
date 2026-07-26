// Package vaultconfig loads the publish-vault site configuration file
// (.publish/config.yaml) and answers gitignore-style path-exclusion queries
// derived from its `ignore` list.
//
// The config file is vault-scoped: it travels with the vault in version
// control and configures publishing policy that applies to that specific
// vault. It is distinct from the .vault-ignore file (handled by the internal
// ignore package): the config blacklist uses full gitignore semantics via a
// library, so it supports "**" patterns and nested matches that the legacy
// .vault-ignore matcher deliberately omits.
//
// A missing config file is harmless (Load returns an empty Config and a nil
// error), mirroring the internal ignore package's treatment of a missing
// .vault-ignore file. A malformed file returns an empty Config and the error;
// callers should log and proceed so publishing is not blocked by a bad config.
package vaultconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath is the conventional config location relative to a vault
// root. Callers resolve it against the vault root with filepath.Join.
const DefaultConfigPath = ".publish/config.yaml"

// Config is the deserialized vault configuration. Unknown keys are ignored so
// future fields can be added without breaking older binaries.
type Config struct {
	// Ignore is a list of gitignore-style patterns applied to vault-relative
	// paths. Patterns use FULL gitignore semantics (**, negations, nested
	// matches) via github.com/sabhiram/go-gitignore. An empty list means
	// "ignore nothing" (everything is eligible, subject to per-note publish
	// frontmatter and .vault-ignore).
	Ignore []string `yaml:"ignore"`
}

// Load reads a config file from path. If the file does not exist, Load returns
// an empty Config and a nil error so callers can treat "no config file" and
// "empty config file" identically. Any other read or parse error returns an
// empty Config and the error; callers should log and proceed.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return &Config{}, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return &Config{}, err
	}
	return cfg, nil
}

// LoadFromVaultRoot reads <root>/.publish/config.yaml.
func LoadFromVaultRoot(root string) (*Config, error) {
	return Load(filepath.Join(root, DefaultConfigPath))
}
