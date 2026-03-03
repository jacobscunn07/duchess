package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jacobscunn07/duchess/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCmd creates a cobra.Command with the three flags registered,
// mirroring the real root command flag setup.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "duchess"}
	cmd.PersistentFlags().String("profile", "", "AWS profile name from ~/.aws/config")
	cmd.PersistentFlags().String("region", "", "AWS region (e.g. us-east-1)")
	cmd.PersistentFlags().Int("refresh-interval", 2, "Auto-refresh interval in seconds (default 2)")
	return cmd
}

// setHome overrides HOME so os.UserHomeDir() returns a temp dir.
// Returns a cleanup function.
func setHome(t *testing.T, home string) func() {
	t.Helper()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", home)
	return func() {
		os.Setenv("HOME", orig)
	}
}

// writeDuchessConfig writes content to <home>/.duchess/config.
func writeDuchessConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".duchess")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config"), []byte(content), 0o600))
}

// Test 1: defaults when no file and no flags
func TestLoadConfig_Defaults(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	cmd := newTestCmd()
	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, 2, cfg.RefreshInterval, "default RefreshInterval should be 2")
	assert.Equal(t, "", cfg.Profile, "default Profile should be empty string")
	assert.Equal(t, "", cfg.Region, "default Region should be empty string")
}

// Test 2: file values loaded when no flags are set
func TestLoadConfig_FileValues(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: myprofile\nregion: eu-west-1\nrefresh_interval: 10\n")

	cmd := newTestCmd()
	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, "myprofile", cfg.Profile)
	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, 10, cfg.RefreshInterval)
}

// Test 3: --profile flag overrides YAML file profile
func TestLoadConfig_ProfileFlagOverridesFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: file-profile\nregion: us-east-1\nrefresh_interval: 5\n")

	cmd := newTestCmd()
	require.NoError(t, cmd.Flags().Set("profile", "flag-profile"))

	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, "flag-profile", cfg.Profile, "--profile flag should override file")
	assert.Equal(t, "us-east-1", cfg.Region, "region from file should be preserved")
	assert.Equal(t, 5, cfg.RefreshInterval, "refresh_interval from file should be preserved")
}

// Test 4: --region flag overrides YAML file region
func TestLoadConfig_RegionFlagOverridesFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: file-profile\nregion: ap-northeast-1\nrefresh_interval: 5\n")

	cmd := newTestCmd()
	require.NoError(t, cmd.Flags().Set("region", "us-west-2"))

	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, "file-profile", cfg.Profile, "profile from file should be preserved")
	assert.Equal(t, "us-west-2", cfg.Region, "--region flag should override file")
	assert.Equal(t, 5, cfg.RefreshInterval, "refresh_interval from file should be preserved")
}

// Test 5: --refresh-interval flag overrides YAML file value
func TestLoadConfig_RefreshIntervalFlagOverridesFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: file-profile\nregion: us-east-1\nrefresh_interval: 30\n")

	cmd := newTestCmd()
	require.NoError(t, cmd.Flags().Set("refresh-interval", "7"))

	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, "file-profile", cfg.Profile, "profile from file should be preserved")
	assert.Equal(t, "us-east-1", cfg.Region, "region from file should be preserved")
	assert.Equal(t, 7, cfg.RefreshInterval, "--refresh-interval flag should override file")
}

// Test 6: flags NOT passed do NOT override YAML values
func TestLoadConfig_UnsetFlagsDoNotOverrideFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: keep-this\nregion: ap-south-1\nrefresh_interval: 15\n")

	// Create cmd but do NOT call Set on any flag — none are Changed()
	cmd := newTestCmd()

	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, "keep-this", cfg.Profile, "profile from file must not be overridden by unset flag")
	assert.Equal(t, "ap-south-1", cfg.Region, "region from file must not be overridden by unset flag")
	assert.Equal(t, 15, cfg.RefreshInterval, "refresh_interval from file must not be overridden by unset flag")
}

// Test 7: missing ~/.duchess/config is silently ignored
func TestLoadConfig_MissingFileIgnored(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	// No config file written — directory does not even exist

	cmd := newTestCmd()
	cfg, err := config.LoadConfig(cmd)

	require.NoError(t, err, "missing config file should not return an error")
	assert.Equal(t, 2, cfg.RefreshInterval)
	assert.Equal(t, "", cfg.Profile)
	assert.Equal(t, "", cfg.Region)
}

// Test 8: malformed YAML returns a wrapped error containing "parse config"
func TestLoadConfig_MalformedYAMLReturnsError(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeDuchessConfig(t, home, "profile: [\nbad yaml: : :\n")

	cmd := newTestCmd()
	cfg, err := config.LoadConfig(cmd)

	require.Error(t, err, "malformed YAML should return an error")
	assert.Nil(t, cfg, "cfg should be nil on YAML parse error")
	assert.Contains(t, err.Error(), "parse config", fmt.Sprintf("error message should contain 'parse config', got: %v", err))
}
