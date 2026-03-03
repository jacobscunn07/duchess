package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config holds the duchess runtime configuration. Values are populated from
// ~/.duchess/config (YAML) and then overridden by any CLI flags that were
// explicitly set by the user.
type Config struct {
	Profile         string `yaml:"profile"`
	Region          string `yaml:"region"`
	RefreshInterval int    `yaml:"refresh_interval"`
}

// LoadConfig reads configuration from ~/.duchess/config (if present) and
// applies any CLI flags that were explicitly set (using cmd.Flags().Changed)
// over the file values. CLI flags always win, but only when the user actually
// passed them — unset flags do not override file values.
//
// Missing ~/.duchess/config is silently ignored; defaults apply.
// Malformed YAML returns a wrapped error containing "parse config".
func LoadConfig(cmd *cobra.Command) (*Config, error) {
	// 1. Start with defaults.
	cfg := &Config{
		RefreshInterval: 2,
	}

	// 2. Resolve home directory. Never use "~" string expansion — that is
	// shell-only and does not work in Go.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}

	// 3. Attempt to load ~/.duchess/config.
	configPath := filepath.Join(home, ".duchess", "config")
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		// File exists — parse it.
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	// If ReadFile returns an error (file missing, no permission, etc.) we
	// silently skip — defaults remain in effect.

	// 4. Apply flag overrides only for flags the user explicitly set.
	//    cmd.Flags().Changed("name") returns true only when the flag was
	//    passed on the command line, not when it is at its default value.
	if cmd.Flags().Changed("profile") {
		cfg.Profile, _ = cmd.Flags().GetString("profile")
	}
	if cmd.Flags().Changed("region") {
		cfg.Region, _ = cmd.Flags().GetString("region")
	}
	if cmd.Flags().Changed("refresh-interval") {
		cfg.RefreshInterval, _ = cmd.Flags().GetInt("refresh-interval")
	}

	return cfg, nil
}
