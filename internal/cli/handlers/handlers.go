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

// HandleCode is the root command's runner: resolves a provider, sets up the
// environment, and execs claude. Help handling is manual because the root
// command uses DisableFlagParsing for transparent flag passthrough.
func HandleCode(cmd *cobra.Command, args []string) error {
	if slices.ContainsFunc(args, isHelpFlag) {
		return cmd.Help()
	}

	cfg, err := configFromCmd(cmd)
	if err != nil {
		return err
	}

	providerName, claudeArgs := extractProviderFromArgs(args)
	providerName, provider, err := resolveProvider(providerName, cfg)
	if err != nil {
		return err
	}

	if !provider.HasAuth() {
		return fmt.Errorf(
			"provider %q requires authentication: set %s_API_KEY or configure authToken in %s",
			providerName, strings.ToUpper(providerName), config.GetConfigPath(),
		)
	}

	fmt.Fprintf(os.Stderr, "Launching Claude Code with %s provider using model %s\n",
		providerName, provider.Model)

	if provider.Transformer.HasRules() {
		return launcher.LaunchWithProxy(provider, cfg.Optimization, claudeArgs)
	}

	return launcher.ExecuteClaudeCode(
		launcher.SetupEnvironment(provider, cfg.Optimization),
		claudeArgs,
	)
}

func isHelpFlag(s string) bool { return s == "-h" || s == "--help" }

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

		if v, ok := strings.CutPrefix(arg, "--provider="); ok {
			provider = v
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

	provider, ok := cfg.Providers[name]
	if !ok {
		return "", config.Provider{}, unknownProviderError(name)
	}

	// Override model from router.default if it targets this provider.
	// Format: "provider:model" (e.g. "zai:glm-5.1").
	if prov, model, ok := strings.Cut(cfg.Router.Default, ":"); ok && prov == name && model != "" {
		provider.Model = model
		provider.SmallFastModel = model
	}

	return name, provider, nil
}
