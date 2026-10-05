package launcher

import (
	"context"
	"fmt"
	"io"
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
	argv := append([]string{claudePath}, args...)
	return syscall.Exec(claudePath, argv, env)
}

// LaunchWithProxy starts a local proxy for request transformation, then runs
// Claude Code as a child pointed at the proxy. The Go process stays alive
// until Claude exits so the proxy can serve requests.
func LaunchWithProxy(p config.Provider, opt config.OptimizationConfig, args []string) error {
	return LaunchWithProxyContext(context.Background(), p, opt, args)
}

func LaunchWithProxyContext(ctx context.Context, p config.Provider, opt config.OptimizationConfig, args []string) error {
	return LaunchWithProxyIO(ctx, p, opt, args, os.Stdin, os.Stdout, os.Stderr)
}

// LaunchWithProxyIO keeps the child's terminal and streams while supervising cleanup.
func LaunchWithProxyIO(ctx context.Context, p config.Provider, opt config.OptimizationConfig, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var err error
	p.Transformer, err = config.ResolveTransformerPresets(p.Transformer)
	if err != nil {
		return err
	}
	if err = config.ValidateProvider(p); err != nil {
		return err
	}
	srv := proxy.NewProxy(p.BaseURL, p.AuthCredential(), p.OAuthToken, p.Transformer.Rules)
	port, err := srv.Start()
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}
	slog.Debug("proxy listening", "port", port)
	p = localProxyProvider(p, port)
	claudeErr := runClaudeContext(ctx, proxyEnvironment(SetupEnvironment(p, opt)), args, stdin, stdout, stderr)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("proxy shutdown", "error", err)
	}
	return claudeErr
}

// Preserve each existing exclusion list in the child environment only.
func proxyEnvironment(env []string) []string {
	out := make([]string, 0, len(env)+2)
	values := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		if key == "NO_PROXY" || key == "no_proxy" {
			values[key] = value
			continue
		}
		out = append(out, entry)
	}
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		value := values[key]
		for _, host := range []string{"127.0.0.1", "localhost", "::1"} {
			found := false
			for _, existing := range strings.Split(value, ",") {
				if strings.TrimSpace(existing) == host {
					found = true
					break
				}
			}
			if !found {
				if value != "" {
					value += ","
				}
				value += host
			}
		}
		out = append(out, key+"="+value)
	}
	return out
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
	return runClaudeContext(context.Background(), env, args, os.Stdin, os.Stdout, os.Stderr)
}

func runClaudeContext(ctx context.Context, env, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}
	return superviseChild(ctx, claudePath, env, args, stdin, stdout, stderr)
}

func localProxyProvider(p config.Provider, port int) config.Provider {
	p.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	p.AuthToken, p.APIKey, p.OAuthToken = "ccl-local-proxy", "", ""
	return p
}
