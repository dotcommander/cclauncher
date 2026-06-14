package handlers

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// HandleUse persists the default provider to config.yaml.
func HandleUse(cmd *cobra.Command, args []string) error {
	return handleUse(cmd, args, TerminalProviderSelector{
		In:  cmd.InOrStdin(),
		Out: cmd.OutOrStdout(),
	})
}

func handleUse(cmd *cobra.Command, args []string, selector ProviderSelector) error {
	cfg, err := configFromCmd(cmd)
	if err != nil {
		return err
	}

	name, err := providerNameForUse(cmd.Context(), cfg, args, selector)
	if err != nil {
		return err
	}

	if err := config.SetDefaultProvider(name); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Default provider set to: %s\n", name)
	return nil
}

func providerNameForUse(ctx context.Context, cfg *config.Config, args []string, selector ProviderSelector) (string, error) {
	if len(args) == 1 {
		name := args[0]
		if _, ok := cfg.Providers[name]; !ok {
			return "", unknownProviderError(name)
		}
		return name, nil
	}

	choices := providerChoices(cfg)
	if len(choices) == 0 {
		return "", fmt.Errorf("no providers configured")
	}
	if selector == nil {
		return "", fmt.Errorf("provider selector is not configured")
	}

	name, err := selector.SelectProvider(ctx, choices, cfg.CLI.DefaultProvider)
	if err != nil {
		return "", err
	}
	if _, ok := cfg.Providers[name]; !ok {
		return "", unknownProviderError(name)
	}
	return name, nil
}

func providerChoices(cfg *config.Config) []ProviderChoice {
	names := slices.Sorted(maps.Keys(cfg.Providers))
	choices := make([]ProviderChoice, 0, len(names))
	for _, name := range names {
		choices = append(choices, ProviderChoice{
			Name:  name,
			Model: cfg.Providers[name].Model,
		})
	}
	return choices
}
