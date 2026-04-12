package handlers

import (
	"fmt"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// HandleUse persists the default provider to config.yaml.
func HandleUse(cmd *cobra.Command, args []string) error {
	cfg, err := configFromCmd(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	if _, ok := cfg.Providers[name]; !ok {
		return unknownProviderError(name)
	}

	if err := config.SetDefaultProvider(name); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Default provider set to: %s\n", name)
	return nil
}
