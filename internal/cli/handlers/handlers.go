package handlers

import (
	"fmt"
	"os"
	"slices"
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
	// Handle --help / -h manually since DisableFlagParsing bypasses cobra's help
	if slices.Contains(args, "--help") || slices.Contains(args, "-h") {
		cmd.Help()
		return nil
	}

	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("config not loaded (internal error)")
	}

	// Extract --provider from raw args; everything else passes through to claude
	providerName, claudeArgs := extractProviderFromArgs(args)

	// Resolve provider: explicit --provider flag, then config default
	providerName, providerConfig, err := resolveProvider(providerName, cfg)
	if err != nil {
		return err
	}

	// Validate that provider has required authentication
	if !providerConfig.HasAuth() {
		return fmt.Errorf("provider '%s' requires authentication. Set %s_API_KEY environment variable or configure in config.yaml", providerName, strings.ToUpper(providerName))
	}

	// Proxy mode: local proxy handles request transformation
	if providerConfig.Transformer.HasRules() {
		fmt.Fprintf(os.Stderr, "Launching Claude Code with %s provider (proxy mode) using model %s\n", providerName, providerConfig.Model)
		return launcher.LaunchWithProxy(providerConfig, cfg.Optimization, claudeArgs)
	}

	// Setup environment variables for Claude Code
	env := launcher.SetupEnvironment(providerConfig, cfg.Optimization)

	// Show which provider and model is being used
	fmt.Fprintf(os.Stderr, "Launching Claude Code with %s provider using model %s\n", providerName, providerConfig.Model)

	// Execute Claude Code with configured environment
	// This replaces the current process - nothing after this runs
	return launcher.ExecuteClaudeCode(env, claudeArgs)
}

// extractProviderFromArgs finds and removes --provider from raw args.
// Supports both "--provider value" and "--provider=value" forms.
// Returns the provider name (empty if not found) and the remaining args.
func extractProviderFromArgs(args []string) (string, []string) {
	result := make([]string, 0, len(args))
	var provider string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--provider" && i+1 < len(args) {
			provider = args[i+1]
			i++ // skip the value
			continue
		}

		if strings.HasPrefix(arg, "--provider=") {
			provider = strings.TrimPrefix(arg, "--provider=")
			continue
		}

		result = append(result, arg)
	}

	return provider, result
}

// resolveProvider validates a provider name and returns its resolved name and config.
// If name is empty, uses the config default.
func resolveProvider(name string, cfg *config.Config) (string, config.Provider, error) {
	if name == "" {
		name = cfg.CLI.DefaultProvider
	}

	providerConfig, exists := config.GetProvider(cfg, name)
	if !exists {
		return "", config.Provider{}, fmt.Errorf("unknown provider: %s (check config.yaml at %s)", name, config.GetConfigPath())
	}

	// Override model from router.default if available (format: "provider:model")
	if cfg.Router.Default != "" {
		parts := strings.SplitN(cfg.Router.Default, ":", 2)
		if len(parts) == 2 && parts[0] == name && parts[1] != "" {
			providerConfig.Model = parts[1]
			providerConfig.SmallFastModel = parts[1]
		}
	}

	return name, providerConfig, nil
}
