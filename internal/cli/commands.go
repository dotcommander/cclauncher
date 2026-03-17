package cli

import (
	"fmt"

	"github.com/dotcommander/cclauncher/internal/cli/handlers"
	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// Execute runs the root command
func Execute() error {
	return newRootCmd().Execute()
}

// newRootCmd builds the root command tree with all subcommands and flags
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ccl",
		Short: "CCL - Claude Code Launcher",
		Long: "Launch Claude Code with different LLM providers.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config check for version and update commands
			if cmd.Name() == "version" || cmd.Name() == "update" {
				return nil
			}

			cfg, err := config.Init()
			if err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			cmd.SetContext(config.StoreInContext(cmd.Context(), cfg))
			return nil
		},
		Example: `  ccl                  # Launch with default provider
  ccl -p <provider>    # Launch with specific provider (see config.yaml)`,
		Args: cobra.ArbitraryArgs,
		RunE: handlers.HandleCode,
	}

	rootCmd.Flags().StringP("provider", "p", "",
		"LLM provider (see providers in config.yaml)")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newProvidersCmd())
	rootCmd.AddCommand(newUseCmd())

	return rootCmd
}

func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List configured providers",
		Long:  "List all configured LLM providers with their model and authentication status.",
		RunE:  handlers.HandleProviders,
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ccl version " + handlers.GetVersion())
		},
	}
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update CCL to the latest version",
		Long: `Update CCL to the latest version from GitHub.

This command uses 'go install' to update the binary.
Requires Go to be installed on your system.`,
		Example: `  ccl update         # Update to latest version
  ccl update --check # Check for updates without installing`,
		RunE: handlers.HandleUpdate,
	}

	cmd.Flags().Bool("check", false, "Check for updates without installing")

	return cmd
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use <provider>",
		Short:   "Set the default provider",
		Long:    "Set the default LLM provider. This persists to config.yaml.",
		Example: "  ccl use deepseek\n  ccl use synthetic",
		Args:    cobra.ExactArgs(1),
		RunE:    handlers.HandleUse,
	}
}
