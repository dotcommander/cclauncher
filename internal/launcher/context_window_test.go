package launcher

import (
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

func TestProviderContextWindow(t *testing.T) {
	t.Parallel()
	p := config.Provider{BaseURL: "http://localhost:8000", Model: "opus-omlx", AuthToken: "omlx", MaxContextTokens: 131072}
	env := envMap(SetupEnvironment(p, config.OptimizationConfig{}))
	require.Equal(t, "http://localhost:8000", env["ANTHROPIC_BASE_URL"])
	require.Equal(t, "omlx", env["ANTHROPIC_AUTH_TOKEN"])
	require.Equal(t, "131072", env["CLAUDE_CODE_MAX_CONTEXT_TOKENS"])
	require.Empty(t, env["CLAUDE_CODE_OAUTH_TOKEN"])
	p.MaxContextTokens = 0
	env = envMap(SetupEnvironment(p, config.OptimizationConfig{}))
	require.NotContains(t, env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS")
	p.MaxContextTokens = -1
	require.ErrorContains(t, config.ValidateProvider(p), "maxContextTokens")
}
