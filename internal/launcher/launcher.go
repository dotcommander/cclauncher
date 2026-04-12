package launcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/dotcommander/cclauncher/internal/proxy"
)

// SetupEnvironment prepares environment variables for Claude Code, starting
// from the parent environment with any pre-existing ANTHROPIC_/CLAUDE_CODE_
// entries stripped (to prevent parent-shell leakage).
func SetupEnvironment(p config.Provider, opt config.OptimizationConfig) []string {
	env := filterAnthropicEnv(os.Environ())

	set := func(key, value string) {
		if value != "" {
			env = append(env, key+"="+value)
		}
	}

	// Provider-derived variables
	set("ANTHROPIC_BASE_URL", p.BaseURL)
	set("ANTHROPIC_AUTH_TOKEN", p.AuthCredential())
	set("CLAUDE_CODE_OAUTH_TOKEN", p.OAuthToken)
	set("ANTHROPIC_MODEL", p.Model)
	set("ANTHROPIC_SMALL_FAST_MODEL", p.SmallFastModel)
	set("ANTHROPIC_DEFAULT_OPUS_MODEL", p.Model)
	set("ANTHROPIC_DEFAULT_SONNET_MODEL", p.Model)
	set("ANTHROPIC_DEFAULT_HAIKU_MODEL", p.SmallFastModel)
	set("CLAUDE_CODE_SUBAGENT_MODEL", p.SmallFastModel)

	// Optimization flags — only set when enabled/non-zero so users can still
	// override via parent shell.
	if opt.DisableNonessentialTraffic {
		set("CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "1")
	}
	if opt.DisableAutoupdater {
		set("DISABLE_AUTOUPDATER", "1")
	}
	if opt.DisableTelemetry {
		set("DISABLE_TELEMETRY", "1")
	}
	if opt.DisableErrorReporting {
		set("DISABLE_ERROR_REPORTING", "1")
	}
	if opt.DisableCostWarnings {
		set("DISABLE_COST_WARNINGS", "1")
	}
	if opt.APITimeoutMs > 0 {
		set("API_TIMEOUT_MS", strconv.Itoa(opt.APITimeoutMs))
	}
	if opt.MaxOutputTokens > 0 {
		set("CLAUDE_CODE_MAX_OUTPUT_TOKENS", strconv.Itoa(opt.MaxOutputTokens))
	}
	if opt.NodeMaxOldSpaceSize > 0 {
		set("NODE_OPTIONS", fmt.Sprintf("--max-old-space-size=%d", opt.NodeMaxOldSpaceSize))
	}

	return env
}

// filterAnthropicEnv returns the parent environment minus ANTHROPIC_* and
// CLAUDE_CODE_* entries so the child process sees only our curated values.
// Bare-named Claude flags (DISABLE_AUTOUPDATER, API_TIMEOUT_MS, etc.) are
// intentionally left intact so users can still override them from the parent
// shell — SetupEnvironment only writes them when a config value is non-zero.
func filterAnthropicEnv(parent []string) []string {
	out := make([]string, 0, len(parent))
	for _, e := range parent {
		if strings.HasPrefix(e, "ANTHROPIC_") || strings.HasPrefix(e, "CLAUDE_CODE_") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ExecuteClaudeCode runs the Claude Code CLI with configured environment.
// Uses syscall.Exec to replace the current process — never returns on success.
func ExecuteClaudeCode(env []string, args []string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}
	// argv[0] is the command name by convention
	argv := append([]string{"claude"}, args...)
	return syscall.Exec(claudePath, argv, env)
}

// LaunchWithProxy starts a local proxy for request transformation, then runs
// Claude Code as a child pointed at the proxy. The Go process stays alive
// until Claude exits so the proxy can serve requests.
func LaunchWithProxy(p config.Provider, opt config.OptimizationConfig, args []string) error {
	srv := proxy.NewProxy(p.BaseURL, p.AuthCredential(), p.OAuthToken, p.Transformer.Rules)

	port, err := srv.Start()
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}
	slog.Info("proxy listening", "port", port)

	// Redirect Claude at the local proxy; the proxy handles upstream auth.
	p.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	p.AuthToken, p.APIKey, p.OAuthToken = "", "", ""

	claudeErr := runClaude(SetupEnvironment(p, opt), args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
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

// runClaude spawns claude as a child process, wiring stdio and waiting for exit.
func runClaude(env, args []string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}
	cmd := exec.Command(claudePath, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited: %w", err)
	}
	return nil
}
