package cmd

import (
	"fmt"

	"github.com/jacobscunn07/duchess/internal/config"
	session "github.com/jacobscunn07/duchess/internal/aws"
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
// Loads config, creates an AWS session, calls STS GetCallerIdentity, and prints the result.
// Phase 2 will replace the print statement with a Bubble Tea program launch.
func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	awsCfg, err := session.NewAWSConfig(cmd.Context(), cfg.Profile, cfg.Region)
	if err != nil {
		return err
	}

	account, arn, err := session.GetCallerIdentity(cmd.Context(), awsCfg)
	if err != nil {
		return session.ClassifyCredentialError(err, cfg.Profile)
	}

	fmt.Printf("Account: %s\nARN:     %s\n", account, arn)
	return nil
}
