package handlers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// GetVersion returns the build-time version string
func GetVersion() string {
	return config.Version
}

func HandleCode(cmd *cobra.Command, args []string) error {
	// Load config (no longer creates defaults)
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Extract provider from config
	providerName, providerConfig, err := extractProvider(cmd, cfg)
	if err != nil {
		return err
	}

	// Validate that provider has required authentication
	if providerConfig.AuthToken == "" && providerConfig.APIKey == "" && providerConfig.OAuthToken == "" {
		return fmt.Errorf("provider '%s' requires authentication. Set %s_API_KEY environment variable or configure in config.yaml", providerName, strings.ToUpper(providerName))
	}

	// Setup environment variables for Claude Code
	env := setupEnvironment(providerConfig, cfg.Optimization)

	// Show which provider and model is being used
	fmt.Fprintf(os.Stderr, "Launching Claude Code with %s provider using model %s\n", providerName, providerConfig.Model)

	// Execute Claude Code with configured environment
	// This replaces the current process - nothing after this runs
	return executeClaudeCode(env)
}

// extractProvider extracts and validates provider from command and config
func extractProvider(cmd *cobra.Command, cfg *config.Config) (string, config.ProviderConfig, error) {
	provider, _ := cmd.Flags().GetString("provider")

	if provider == "" {
		provider = cfg.CLI.DefaultProvider
	}

	registry := config.NewProviderRegistryFromConfig(cfg)
	providerConfig, exists := registry.Get(provider)
	if !exists {
		return "", config.ProviderConfig{}, fmt.Errorf("unknown provider: %s (check config.yaml at %s)", provider, config.GetConfigPath())
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

// setupEnvironment prepares environment variables for Claude Code
func setupEnvironment(providerConfig config.ProviderConfig, opt config.OptimizationConfig) []string {
	// Clear existing Anthropic env vars
	env := clearAnthropicEnvVars()

	// Resolve auth: prefer AuthToken, fall back to APIKey
	authToken := providerConfig.AuthToken
	if authToken == "" {
		authToken = providerConfig.APIKey
	}

	// Generate env vars from provider config
	providerEnvVars := map[string]string{
		"ANTHROPIC_BASE_URL":             providerConfig.BaseURL,
		"ANTHROPIC_AUTH_TOKEN":           authToken,
		"CLAUDE_CODE_OAUTH_TOKEN":        providerConfig.OAuthToken,
		"ANTHROPIC_MODEL":                providerConfig.Model,
		"ANTHROPIC_SMALL_FAST_MODEL":     providerConfig.SmallFastModel,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   providerConfig.Model,
		"ANTHROPIC_DEFAULT_SONNET_MODEL": providerConfig.Model,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  providerConfig.SmallFastModel,
		"CLAUDE_CODE_SUBAGENT_MODEL":     providerConfig.SmallFastModel,
	}

	// Add provider env vars
	env = addProviderEnvVars(env, providerEnvVars)

	// Apply optimization settings from config
	env = applyOptimizationDefaults(env, opt)

	// Increase Node.js heap size to prevent OOM errors
	if opt.NodeMaxOldSpaceSize > 0 {
		env = setEnvVar(env, "NODE_OPTIONS", fmt.Sprintf("--max-old-space-size=%d", opt.NodeMaxOldSpaceSize), false)
	}

	return env
}

// clearAnthropicEnvVars removes existing Claude Code environment variables
// to prevent leaking values from the parent shell
func clearAnthropicEnvVars() []string {
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ANTHROPIC_") ||
			strings.HasPrefix(e, "CLAUDE_CODE_") {
			continue
		}
		env = append(env, e)
	}
	return env
}

// addProviderEnvVars adds provider-specific environment variables
func addProviderEnvVars(env []string, providerEnvVars map[string]string) []string {
	for key, value := range providerEnvVars {
		if value == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

// setEnvVar sets an environment variable. If overwrite is true, replaces existing value.
// If overwrite is false, only adds if the key doesn't exist.
func setEnvVar(env []string, key, value string, overwrite bool) []string {
	for i, e := range env {
		if strings.HasPrefix(e, key+"=") {
			if overwrite {
				env[i] = fmt.Sprintf("%s=%s", key, value)
			}
			return env
		}
	}
	return append(env, fmt.Sprintf("%s=%s", key, value))
}

// applyOptimizationDefaults sets optimization env vars from config
func applyOptimizationDefaults(env []string, opt config.OptimizationConfig) []string {
	if opt.DisableNonessentialTraffic {
		env = setEnvVar(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "1", false)
	}
	if opt.DisableAutoupdater {
		env = setEnvVar(env, "DISABLE_AUTOUPDATER", "1", false)
	}
	if opt.DisableTelemetry {
		env = setEnvVar(env, "DISABLE_TELEMETRY", "1", false)
	}
	if opt.DisableErrorReporting {
		env = setEnvVar(env, "DISABLE_ERROR_REPORTING", "1", false)
	}
	if opt.DisableCostWarnings {
		env = setEnvVar(env, "DISABLE_COST_WARNINGS", "1", false)
	}
	if opt.APITimeoutMs > 0 {
		env = setEnvVar(env, "API_TIMEOUT_MS", fmt.Sprintf("%d", opt.APITimeoutMs), false)
	}
	if opt.MaxOutputTokens > 0 {
		env = setEnvVar(env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS", fmt.Sprintf("%d", opt.MaxOutputTokens), false)
	}
	return env
}

// executeClaudeCode runs the Claude Code CLI with configured environment
// Uses syscall.Exec to replace the current process with Claude Code
// This function never returns on success
func executeClaudeCode(env []string) error {
	// Find the claude executable in PATH
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Prepare argv for exec (argv[0] should be the command name)
	argv := []string{"claude"}

	// syscall.Exec replaces the current process with Claude Code
	// On success, this function never returns
	// On failure, it returns an error
	return syscall.Exec(claudePath, argv, env)
}
