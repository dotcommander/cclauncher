package launcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/dotcommander/cclauncher/internal/proxy"
)

type optEnvRule struct {
	envKey   string
	getValue func(config.OptimizationConfig) string
}

func boolEnv(b bool) string {
	if b {
		return "1"
	}
	return ""
}

func intEnv(n int) string {
	if n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return ""
}

var optEnvRules = []optEnvRule{
	{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", func(o config.OptimizationConfig) string { return boolEnv(o.DisableNonessentialTraffic) }},
	{"DISABLE_AUTOUPDATER", func(o config.OptimizationConfig) string { return boolEnv(o.DisableAutoupdater) }},
	{"DISABLE_TELEMETRY", func(o config.OptimizationConfig) string { return boolEnv(o.DisableTelemetry) }},
	{"DISABLE_ERROR_REPORTING", func(o config.OptimizationConfig) string { return boolEnv(o.DisableErrorReporting) }},
	{"DISABLE_COST_WARNINGS", func(o config.OptimizationConfig) string { return boolEnv(o.DisableCostWarnings) }},
	{"API_TIMEOUT_MS", func(o config.OptimizationConfig) string { return intEnv(o.APITimeoutMs) }},
	{"CLAUDE_CODE_MAX_OUTPUT_TOKENS", func(o config.OptimizationConfig) string { return intEnv(o.MaxOutputTokens) }},
}

// SetupEnvironment prepares environment variables for Claude Code.
func SetupEnvironment(providerConfig config.Provider, opt config.OptimizationConfig) []string {
	// Clear existing Anthropic env vars
	env := clearAnthropicEnvVars()

	// Resolve auth: prefer AuthToken, fall back to APIKey
	authToken := resolveAuthToken(providerConfig)

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
	env := make([]string, 0, len(os.Environ()))
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
	for _, rule := range optEnvRules {
		if v := rule.getValue(opt); v != "" {
			env = setEnvVar(env, rule.envKey, v, false)
		}
	}
	return env
}

// ExecuteClaudeCode runs the Claude Code CLI with configured environment.
// Uses syscall.Exec to replace the current process with Claude Code.
// This function never returns on success.
func ExecuteClaudeCode(env []string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	// Prepare argv for exec (argv[0] should be the command name)
	argv := []string{"claude"}

	// syscall.Exec replaces the current process with Claude Code
	// On success, this function never returns
	// On failure, it returns an error
	return syscall.Exec(claudePath, argv, env)
}

// LaunchWithProxy starts a local proxy for request transformation, then
// launches Claude Code pointing at the proxy. The Go process stays alive
// to serve the proxy until Claude exits.
func LaunchWithProxy(providerConfig config.Provider, opt config.OptimizationConfig) error {
	authToken := resolveAuthToken(providerConfig)

	p := proxy.NewProxy(
		providerConfig.BaseURL,
		authToken,
		providerConfig.OAuthToken,
		providerConfig.Transformer.Rules,
	)

	port, err := p.Start()
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}
	slog.Info("proxy listening", "port", port)

	// Point Claude at the local proxy; proxy handles real auth
	providerConfig.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	providerConfig.AuthToken = ""
	providerConfig.APIKey = ""
	providerConfig.OAuthToken = ""

	env := SetupEnvironment(providerConfig, opt)

	claudeErr := runClaude(env)

	// Shut down proxy with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Shutdown(ctx); err != nil {
		slog.Error("proxy shutdown", "error", err)
	}

	return claudeErr
}

// findClaude locates the claude binary in PATH.
func findClaude() (string, error) {
	p, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH: %w", err)
	}
	return p, nil
}

// resolveAuthToken returns the preferred auth credential for a provider.
func resolveAuthToken(p config.Provider) string {
	if p.AuthToken != "" {
		return p.AuthToken
	}
	return p.APIKey
}

// runClaude spawns claude as a child process, piping stdio, and waits for exit.
func runClaude(env []string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	cmd := exec.Command(claudePath)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited: %w", err)
	}
	return nil
}
