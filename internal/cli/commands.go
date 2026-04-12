package cli

import (
	"fmt"

	"github.com/dotcommander/cclauncher/internal/cli/handlers"
	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

// newRootCmd builds the root command tree.
// The root uses DisableFlagParsing so that every flag except --provider is
// forwarded verbatim to claude; see handlers.HandleCode for the parse logic.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ccl",
		Short: "Launch Claude Code with different LLM providers",
		Long: "Launch Claude Code with different LLM providers.\n\n" +
			"All flags except --provider are passed through to Claude Code.\n" +
			"Use --provider to select an LLM provider (or set a default via 'ccl use').",
		Example: `  ccl                                  # Launch with default provider
  ccl --provider deepseek              # Launch with specific provider
  ccl --provider deepseek -p "hello"   # Provider + claude print mode
  ccl -c -p "/dc:next"                 # Continue session + print mode
  ccl --model sonnet "hello"           # Pass claude flags directly`,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		PersistentPreRunE:  loadConfigIntoContext,
		RunE:               handlers.HandleCode,
	}

	root.AddCommand(
		newVersionCmd(),
		newUpdateCmd(),
		newProvidersCmd(),
		newUseCmd(),
	)
	return root
}

// loadConfigIntoContext is the root's PersistentPreRunE: it initializes the
// config file (creating defaults if missing) and stashes the result in the
// command context for handlers. version/update skip this to remain usable
// with a missing or broken config.
func loadConfigIntoContext(cmd *cobra.Command, _ []string) error {
	switch cmd.Name() {
	case "version", "update":
		return nil
	}
	cfg, err := config.Init()
	if err != nil {
		return fmt.Errorf("initialize config: %w", err)
	}
	cmd.SetContext(config.StoreInContext(cmd.Context(), cfg))
	return nil
}

func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List configured providers",
		Long:  "List all configured LLM providers with their model and authentication status.",
		Args:  cobra.NoArgs,
		RunE:  handlers.HandleProviders,
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			fmt.Println("ccl version " + handlers.GetVersion())
		},
	}
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update CCL to the latest version",
		Long: "Update CCL to the latest version from GitHub.\n\n" +
			"This command uses 'go install' to update the binary.\n" +
			"Requires Go to be installed on your system.",
		Example: "  ccl update         # Update to latest version\n" +
			"  ccl update --check # Check for updates without installing",
		Args: cobra.NoArgs,
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
