package handlers

import (
	"fmt"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// HandleUse sets the default provider in config.yaml.
func HandleUse(cmd *cobra.Command, args []string) error {
	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("config not loaded (internal error)")
	}

	providerName := args[0]

	if _, exists := cfg.Providers[providerName]; !exists {
		return fmt.Errorf("unknown provider: %s (check config.yaml at %s)", providerName, config.GetConfigPath())
	}

	if err := config.SetDefaultProvider(providerName); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Default provider set to: %s\n", providerName)
	return nil
}
