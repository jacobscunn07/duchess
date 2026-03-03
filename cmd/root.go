package cmd

import (
	"fmt"

	"github.com/jacobscunn07/duchess/internal/config"
	"github.com/spf13/cobra"
)

// rootCmd is the cobra root command for the duchess binary.
var rootCmd = &cobra.Command{
	Use:   "duchess",
	Short: "Navigate your AWS resources across accounts and regions",
	RunE:  runRoot,
}

// Execute runs the root cobra command and returns any error.
// This is the only function called from main.go.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().String("profile", "", "AWS profile name from ~/.aws/config")
	rootCmd.PersistentFlags().String("region", "", "AWS region (e.g. us-east-1)")
	rootCmd.PersistentFlags().Int("refresh-interval", 2, "Auto-refresh interval in seconds (default 2)")
}

// runRoot is the RunE handler for the root command.
// Phase 1 placeholder: loads config and prints values to stdout.
// Phase 2 will replace this with Bubble Tea program launch.
func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	fmt.Printf("Profile: %s\nRegion: %s\nRefreshInterval: %d\n", cfg.Profile, cfg.Region, cfg.RefreshInterval)
	return nil
}
