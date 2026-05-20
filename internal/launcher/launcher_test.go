package launcher

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
)

// envSnapshot wraps os.Environ() so tests can document why they peek at the
// real parent environment (to distinguish inherited vars from ones the SUT
// appended).
func envSnapshot() []string { return os.Environ() }

// envMap converts a KEY=VALUE slice into a map for stable lookup. Later entries
// win, matching how a child process resolves environment lookups via libc.
func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		m[k] = v
	}
	return m
}

func TestFilterAnthropicEnv_RemovesAnthropicAndClaudeCodePrefixes(t *testing.T) {
	t.Parallel()

	parent := []string{
		"ANTHROPIC_BASE_URL=https://leaked.example",
		"ANTHROPIC_API_KEY=leak",
		"CLAUDE_CODE_OAUTH_TOKEN=leak",
		"CLAUDE_CODE_SUBAGENT_MODEL=leak",
		"PATH=/usr/bin:/bin",
		"HOME=/home/user",
		// Bare optimization variables — must be preserved so parent shell can
		// still override them when SetupEnvironment leaves them unset.
		"DISABLE_AUTOUPDATER=1",
		"DISABLE_TELEMETRY=1",
		"DISABLE_ERROR_REPORTING=1",
		"DISABLE_COST_WARNINGS=1",
		"DISABLE_NONESSENTIAL_TRAFFIC=1",
		"API_TIMEOUT_MS=60000",
		"NODE_OPTIONS=--max-old-space-size=4096",
	}

	got := filterAnthropicEnv(parent)
	gotMap := envMap(got)

	// ANTHROPIC_* and CLAUDE_CODE_* entries must be gone.
	for _, e := range got {
		if strings.HasPrefix(e, "ANTHROPIC_") || strings.HasPrefix(e, "CLAUDE_CODE_") {
			t.Errorf("filterAnthropicEnv left prefixed entry %q in output", e)
		}
	}

	// Unrelated entries must survive.
	wantPreserved := map[string]string{
		"PATH":                         "/usr/bin:/bin",
		"HOME":                         "/home/user",
		"DISABLE_AUTOUPDATER":          "1",
		"DISABLE_TELEMETRY":            "1",
		"DISABLE_ERROR_REPORTING":      "1",
		"DISABLE_COST_WARNINGS":        "1",
		"DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"API_TIMEOUT_MS":               "60000",
		"NODE_OPTIONS":                 "--max-old-space-size=4096",
	}
	for k, want := range wantPreserved {
		if gotMap[k] != want {
			t.Errorf("filterAnthropicEnv dropped or rewrote %s: got %q want %q", k, gotMap[k], want)
		}
	}
}

func TestFilterAnthropicEnv_EmptyInput(t *testing.T) {
	t.Parallel()

	got := filterAnthropicEnv(nil)
	if len(got) != 0 {
		t.Errorf("filterAnthropicEnv(nil) = %v, want empty slice", got)
	}
}

func TestSetupEnvironment_SetsProviderAuthAndModelVars(t *testing.T) {
	t.Parallel()

	p := config.Provider{
		BaseURL:        "https://api.example.test",
		AuthToken:      "auth-tok",
		OAuthToken:     "oauth-tok",
		Model:          "model-large",
		SmallFastModel: "model-small",
	}
	opt := config.OptimizationConfig{}

	env := SetupEnvironment(p, opt)
	m := envMap(env)

	want := map[string]string{
		"ANTHROPIC_BASE_URL":             "https://api.example.test",
		"ANTHROPIC_AUTH_TOKEN":           "auth-tok",
		"CLAUDE_CODE_OAUTH_TOKEN":        "oauth-tok",
		"ANTHROPIC_MODEL":                "model-large",
		"ANTHROPIC_SMALL_FAST_MODEL":     "model-small",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "model-large",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "model-large",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "model-small",
		"CLAUDE_CODE_SUBAGENT_MODEL":     "model-small",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("env[%s] = %q, want %q", k, m[k], v)
		}
	}

	// Optimization vars must not be APPENDED by SetupEnvironment when
	// OptimizationConfig is zero. (Bare optimization vars inherited from the
	// parent environment are intentionally preserved — see filterAnthropicEnv
	// docs — so we cannot just assert the keys are absent. We instead assert
	// that SetupEnvironment did not push its own entry on top of the parent's.)
	parentInherited := map[string]bool{}
	for _, e := range filterAnthropicEnv(envSnapshot()) {
		k, _, ok := strings.Cut(e, "=")
		if ok {
			parentInherited[k] = true
		}
	}
	zeroVars := []string{
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		"DISABLE_AUTOUPDATER",
		"DISABLE_TELEMETRY",
		"DISABLE_ERROR_REPORTING",
		"DISABLE_COST_WARNINGS",
		"API_TIMEOUT_MS",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS",
		"NODE_OPTIONS",
	}
	for _, k := range zeroVars {
		if _, ok := m[k]; ok && !parentInherited[k] {
			t.Errorf("env[%s] should be unset when OptimizationConfig is zero, got %q (not in parent env, so SetupEnvironment appended it)", k, m[k])
		}
	}
}

func TestSetupEnvironment_AuthTokenPreferredOverAPIKey(t *testing.T) {
	t.Parallel()

	p := config.Provider{
		BaseURL:   "https://api.example.test",
		AuthToken: "auth-tok",
		APIKey:    "api-key",
	}

	env := SetupEnvironment(p, config.OptimizationConfig{})
	m := envMap(env)

	if got := m["ANTHROPIC_AUTH_TOKEN"]; got != "auth-tok" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %q, want %q (AuthToken should win over APIKey)", got, "auth-tok")
	}
}

func TestSetupEnvironment_APIKeyUsedWhenNoAuthToken(t *testing.T) {
	t.Parallel()

	p := config.Provider{
		BaseURL: "https://api.example.test",
		APIKey:  "api-key",
	}

	env := SetupEnvironment(p, config.OptimizationConfig{})
	m := envMap(env)

	if got := m["ANTHROPIC_AUTH_TOKEN"]; got != "api-key" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %q, want %q (APIKey should fall through)", got, "api-key")
	}
}

func TestSetupEnvironment_OptimizationFlagsSet(t *testing.T) {
	t.Parallel()

	p := config.Provider{BaseURL: "https://api.example.test"}
	opt := config.OptimizationConfig{
		DisableNonessentialTraffic: true,
		DisableAutoupdater:         true,
		DisableTelemetry:           true,
		DisableErrorReporting:      true,
		DisableCostWarnings:        true,
		APITimeoutMs:               60000,
		MaxOutputTokens:            8192,
		NodeMaxOldSpaceSize:        4096,
	}

	env := SetupEnvironment(p, opt)
	m := envMap(env)

	want := map[string]string{
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"DISABLE_AUTOUPDATER":                      "1",
		"DISABLE_TELEMETRY":                        "1",
		"DISABLE_ERROR_REPORTING":                  "1",
		"DISABLE_COST_WARNINGS":                    "1",
		"API_TIMEOUT_MS":                           "60000",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS":            "8192",
		"NODE_OPTIONS":                             fmt.Sprintf("--max-old-space-size=%d", 4096),
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("env[%s] = %q, want %q", k, m[k], v)
		}
	}
}

func TestSetupEnvironment_EmptyProviderFieldsAreOmitted(t *testing.T) {
	t.Parallel()

	// Provider with no credentials/models — set() should skip empty strings so
	// child shell never sees blank ANTHROPIC_* vars masking parent values.
	p := config.Provider{BaseURL: "https://api.example.test"}

	env := SetupEnvironment(p, config.OptimizationConfig{})
	m := envMap(env)

	mustAbsent := []string{
		"ANTHROPIC_AUTH_TOKEN",
		"CLAUDE_CODE_OAUTH_TOKEN",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"CLAUDE_CODE_SUBAGENT_MODEL",
	}
	for _, k := range mustAbsent {
		if v, ok := m[k]; ok {
			t.Errorf("env[%s] should be absent for empty provider, got %q", k, v)
		}
	}

	if m["ANTHROPIC_BASE_URL"] != "https://api.example.test" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", m["ANTHROPIC_BASE_URL"], "https://api.example.test")
	}
}

func TestSetupEnvironment_StripsInheritedAnthropicEnv(t *testing.T) {
	// No t.Parallel — t.Setenv is incompatible with parallel tests, but this
	// test only needs serial execution for the env-restore guarantee.
	t.Setenv("ANTHROPIC_LEAKED_FROM_PARENT", "leaked-value")

	p := config.Provider{BaseURL: "https://api.example.test"}
	env := SetupEnvironment(p, config.OptimizationConfig{})

	// The leaked var must not survive into the child env.
	if slices.ContainsFunc(env, func(e string) bool {
		return strings.HasPrefix(e, "ANTHROPIC_LEAKED_FROM_PARENT=")
	}) {
		t.Errorf("SetupEnvironment did not strip inherited ANTHROPIC_LEAKED_FROM_PARENT from output")
	}
}
