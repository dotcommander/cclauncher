package cli

import (
	"fmt"
	"os"

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
			created, err := config.EnsureExists()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize config: %v\n", err)
				return nil
			}
			if created {
				fmt.Fprintf(os.Stderr, "Created default config at %s\n", config.GetConfigPath())
			}
			return nil
		},
		Example: `  ccl                  # Launch with default provider
  ccl -p kimi2         # Launch with Kimi-K2.5
  ccl -p deepseek      # Launch with DeepSeek-V3.2
  ccl -p minimax       # Launch with MiniMax-M2
  ccl -p claude        # Launch with Claude (OAuth)
  ccl -p claude2       # Launch with Claude (second account)
  ccl -p qwen          # Launch with Qwen3-VL
  ccl -p qwen3-coder   # Launch with Qwen3-Coder
  ccl -p zai           # Launch with Z.ai
  ccl -p llamabarn     # Launch with LlamaBarn
  ccl -p synthetic     # Launch with Synthetic.new`,
		Args: cobra.ArbitraryArgs,
		RunE: handlers.HandleCode,
	}

	rootCmd.Flags().StringP("provider", "p", "",
		"LLM provider (synthetic, kimi2, deepseek, minimax, qwen, qwen3-coder, claude, claude2, zai, llamabarn)")

	rootCmd.AddCommand(newVersionCmd(), newCleanupCmd(), newRefsCmd())

	return rootCmd
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

func newCleanupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup stale session references",
		Long: `Reset the reference count for stale sessions.

This is useful when ccl sessions crash and leave stale reference counts.
The reference count is used to track concurrent sessions.`,
		RunE: handlers.HandleCleanup,
	}
}

func newRefsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refs",
		Short: "Show current reference count",
		Long: `Display the current session reference count.

This shows how many active ccl sessions are currently tracked.
Useful for debugging concurrent session issues.`,
		RunE: handlers.HandleRefs,
	}
}
