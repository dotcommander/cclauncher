package handlers

import (
	"fmt"
	"os"
	"strings"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/dotcommander/cclauncher/internal/launcher"
	"github.com/spf13/cobra"
)

// GetVersion returns the build-time version string
func GetVersion() string {
	return config.Version
}

func HandleCode(cmd *cobra.Command, args []string) error {
	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("config not loaded (internal error)")
	}

	// Extract provider from config
	providerName, providerConfig, err := extractProvider(cmd, cfg)
	if err != nil {
		return err
	}

	// Validate that provider has required authentication
	if !providerConfig.HasAuth() {
		return fmt.Errorf("provider '%s' requires authentication. Set %s_API_KEY environment variable or configure in config.yaml", providerName, strings.ToUpper(providerName))
	}

	// Setup environment variables for Claude Code
	env := launcher.SetupEnvironment(providerConfig, cfg.Optimization)

	// Show which provider and model is being used
	fmt.Fprintf(os.Stderr, "Launching Claude Code with %s provider using model %s\n", providerName, providerConfig.Model)

	// Execute Claude Code with configured environment
	// This replaces the current process - nothing after this runs
	return launcher.ExecuteClaudeCode(env)
}

// extractProvider extracts and validates provider from command and config
func extractProvider(cmd *cobra.Command, cfg *config.Config) (string, config.Provider, error) {
	provider, _ := cmd.Flags().GetString("provider")

	if provider == "" {
		provider = cfg.CLI.DefaultProvider
	}

	providerConfig, exists := config.GetProvider(cfg, provider)
	if !exists {
		return "", config.Provider{}, fmt.Errorf("unknown provider: %s (check config.yaml at %s)", provider, config.GetConfigPath())
	}

	// Override model from router.default if available (format: "provider:model")
	if cfg.Router.Default != "" {
		parts := strings.SplitN(cfg.Router.Default, ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" && parts[0] == provider {
			providerConfig.Model = parts[1]
			providerConfig.SmallFastModel = parts[1]
		}
	}

	return provider, providerConfig, nil
}
