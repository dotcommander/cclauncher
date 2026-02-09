package handlers

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ccg/ccg/internal/config"
	"github.com/ccg/ccg/internal/process"
	"github.com/spf13/cobra"
)

// RefCounter abstracts reference counting operations for testability
type RefCounter interface {
	IncrementRefCount() error
	DecrementRefCount() (int, error)
	GetRefCount() (int, error)
	ResetRefCount() error
}

// Ensure concrete types implement interfaces
var _ RefCounter = (*process.RefCountManager)(nil)

// GetVersion returns the configured version string
func GetVersion() string {
	cfg, err := config.Load()
	if err != nil {
		return "1.0.0"
	}
	if cfg.CLI.Version == "" {
		return "1.0.0"
	}
	return cfg.CLI.Version
}

func HandleCode(cmd *cobra.Command, args []string) error {
	// Load config once (creates defaults if needed)
	cfg, err := config.Init()
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "error", err)
		cfg = config.DefaultConfig()
	}

	// Extract provider from config
	providerName, providerConfig, err := extractProvider(cmd, cfg)
	if err != nil {
		return err
	}

	// Reference count management (increment only, since exec never returns)
	_, _ = setupReferenceCounting()

	// Setup environment variables for Claude Code
	env := setupEnvironment(providerConfig, cfg.Optimization)

	// Show which provider and model is being used
	fmt.Fprintf(os.Stderr, "🚀 Using %s provider with model %s\n", providerName, providerConfig.Model)

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
		return "", config.ProviderConfig{}, fmt.Errorf("unknown provider: %s", provider)
	}

	// Override model from router.default if available (format: "provider:model")
	if cfg.Router.Default != "" {
		parts := strings.SplitN(cfg.Router.Default, ":", 2)
		if len(parts) == 2 && parts[0] == provider {
			providerConfig.Model = parts[1]
			providerConfig.SmallFastModel = parts[1]
		}
	}

	return provider, providerConfig, nil
}

// setupReferenceCounting manages process reference counting
// Note: Cleanup function is ignored since syscall.Exec() never returns
func setupReferenceCounting() (*process.RefCountManager, func()) {
	refMgr := process.NewRefCountManager()

	if err := refMgr.IncrementRefCount(); err != nil {
		slog.Warn("Failed to increment reference count", "error", err)
	}

	cleanup := func() {
		// This will never be called due to exec, but kept for interface consistency
		count, err := refMgr.DecrementRefCount()
		if err != nil {
			slog.Warn("Failed to decrement reference count", "error", err)
		} else {
			slog.Debug("Session completed", "remaining_sessions", count)
		}
	}

	return refMgr, cleanup
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

func HandleCleanup(cmd *cobra.Command, args []string) error {
	refMgr := process.NewRefCountManager()

	// Get current refcount
	count, err := refMgr.GetRefCount()
	if err != nil {
		slog.Warn("Failed to get reference count", "error", err)
		count = 0
	}

	if count == 0 {
		fmt.Println("✅ No active sessions, nothing to cleanup")
		return nil
	}

	fmt.Printf("⚠️  Found %d active session reference(s)\n", count)
	fmt.Println("This could be from:")
	fmt.Println("  - Currently running ccg sessions")
	fmt.Println("  - Crashed sessions that didn't cleanup")
	fmt.Println()

	// Ask for confirmation
	fmt.Print("Reset reference count to 0? (yes/no): ")
	var response string
	_, _ = fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
		fmt.Println("Cleanup cancelled")
		return nil
	}

	// Reset refcount
	if err := refMgr.ResetRefCount(); err != nil {
		return fmt.Errorf("failed to reset reference count: %w", err)
	}

	fmt.Println("✅ Reference count reset to 0")
	return nil
}

func HandleRefs(cmd *cobra.Command, args []string) error {
	refMgr := process.NewRefCountManager()

	count, err := refMgr.GetRefCount()
	if err != nil {
		return fmt.Errorf("failed to get reference count: %w", err)
	}

	if count == 0 {
		fmt.Println("📊 Reference count: 0 (no active sessions)")
	} else {
		fmt.Printf("📊 Reference count: %d (active sessions)\n", count)
	}

	fmt.Printf("📁 Reference file: %s\n", process.GetRefCountFile())

	return nil
}
