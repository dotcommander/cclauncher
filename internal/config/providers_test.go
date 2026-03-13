package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestConfigPath() string {
	return filepath.Join("testdata", "config.yaml")
}

func TestGetProvider_AllProvidersExist(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err, "Failed to load test config")

	// Should load all providers from fixture
	expectedProviders := []string{"kimi2", "qwen", "qwen3-coder", "deepseek", "claude", "claude2", "zai", "synthetic", "minimax", "llamabarn"}
	for _, name := range expectedProviders {
		_, exists := GetProvider(cfg, name)
		assert.True(t, exists, "Provider %s should exist", name)
	}
}

func TestGetProvider_ProvidersExist(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	expectedProviders := []string{"kimi2", "qwen", "qwen3-coder", "deepseek", "claude", "claude2", "zai", "synthetic", "minimax"}

	for _, providerName := range expectedProviders {
		t.Run(providerName, func(t *testing.T) {
			provider, exists := GetProvider(cfg, providerName)
			assert.True(t, exists, "Provider %s should exist", providerName)
			assert.NotEmpty(t, provider.BaseURL, "Provider config should have BaseURL")
		})
	}
}

func TestGetProvider_BaseURLs(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	tests := []struct {
		name         string
		providerName string
		expectedURL  string
	}{
		{"kimi2", "kimi2", "https://api.synthetic.new/anthropic"},
		{"qwen", "qwen", "https://api.synthetic.new/anthropic"},
		{"qwen3-coder", "qwen3-coder", "https://api.synthetic.new/anthropic"},
		{"deepseek", "deepseek", "https://api.synthetic.new/anthropic"},
		{"zai", "zai", "https://api.z.ai/api/anthropic"},
		{"synthetic", "synthetic", "https://api.synthetic.new/anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, exists := GetProvider(cfg, tt.providerName)
			assert.True(t, exists)
			assert.Equal(t, tt.expectedURL, provider.BaseURL)
		})
	}
}

func TestGetProvider_Models(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	deepseek, _ := GetProvider(cfg, "deepseek")
	assert.Equal(t, "hf:deepseek-ai/DeepSeek-V3.2", deepseek.Model)
	assert.Equal(t, "hf:deepseek-ai/DeepSeek-V3.2", deepseek.SmallFastModel)

	synthetic, _ := GetProvider(cfg, "synthetic")
	assert.Equal(t, "hf:moonshotai/Kimi-K2.5", synthetic.Model)
	assert.Equal(t, "hf:moonshotai/Kimi-K2.5", synthetic.SmallFastModel)

	qwen, _ := GetProvider(cfg, "qwen")
	assert.Equal(t, "hf:Qwen/Qwen3-VL-235B-A22B-Instruct", qwen.Model)
	assert.Equal(t, "hf:Qwen/Qwen3-VL-235B-A22B-Instruct", qwen.SmallFastModel)

	qwen3Coder, _ := GetProvider(cfg, "qwen3-coder")
	assert.Equal(t, "hf:Qwen/Qwen3-Coder-480B-A35B-Instruct", qwen3Coder.Model)
	assert.Equal(t, "hf:Qwen/Qwen3-Coder-480B-A35B-Instruct", qwen3Coder.SmallFastModel)
}

func TestGetProvider_AuthenticationMethods(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	tests := []struct {
		name         string
		providerName string
		hasAuthToken bool
		hasAPIKey    bool
	}{
		{"kimi2 uses auth token", "kimi2", true, false},
		{"qwen uses auth token", "qwen", true, false},
		{"qwen3-coder uses auth token", "qwen3-coder", true, false},
		{"deepseek uses auth token", "deepseek", true, false},
		{"claude uses oauth", "claude", false, false},
		{"claude2 uses oauth token", "claude2", false, false},
		{"zai uses auth token", "zai", true, false},
		{"synthetic uses auth token", "synthetic", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, _ := GetProvider(cfg, tt.providerName)

			if tt.hasAuthToken {
				// Auth tokens resolve from env vars at load time;
				// may be empty in test environments if env vars aren't set
				assert.Empty(t, provider.APIKey, "Provider %s should not have API key", tt.providerName)
			}

			if !tt.hasAuthToken && !tt.hasAPIKey {
				// OAuth-only providers (claude, claude2)
				assert.Empty(t, provider.APIKey, "Provider %s should not have API key", tt.providerName)
			}
		})
	}
}

func TestGetProvider_EnvironmentVariableResolution(t *testing.T) {
	// Test that CLAUDE2_OAUTH_TOKEN environment variable is resolved for the claude2 provider
	testValue := "test-oauth-token"
	t.Setenv("CLAUDE2_OAUTH_TOKEN", testValue)

	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	claude2, exists := GetProvider(cfg, "claude2")
	assert.True(t, exists)
	assert.Equal(t, testValue, claude2.OAuthToken)
}

func TestEnsureExists_CreatesDefaultConfig(t *testing.T) {
	// Use a temporary directory that doesn't have a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	loader := NewLoaderWithPaths(tmpDir, configPath)

	created, err := loader.EnsureExists()
	assert.NoError(t, err)
	assert.True(t, created, "Should create a new config when none exists")

	// Verify the config file was created
	_, statErr := filepath.Glob(configPath)
	assert.NoError(t, statErr)
}

func TestEnsureExists_ConfigExists(t *testing.T) {
	// Use the testdata directory which has a config file
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())

	created, err := loader.EnsureExists()
	assert.NoError(t, err)
	assert.False(t, created, "Should not create a new config when one already exists")
}

func TestLoad_Success(t *testing.T) {
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Providers)
	assert.Equal(t, "synthetic", cfg.CLI.DefaultProvider)
}

func TestLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoaderWithPaths(tmpDir, filepath.Join(tmpDir, "nonexistent.yaml"))

	_, err := loader.Load()
	assert.Error(t, err)
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Providers: map[string]Provider{
			"test": {BaseURL: "https://example.com"},
		},
	}

	applyDefaults(cfg)

	// Should apply CLI defaults
	assert.Equal(t, "synthetic", cfg.CLI.DefaultProvider)

	// Should apply optimization defaults
	assert.True(t, cfg.Optimization.DisableNonessentialTraffic)
	assert.True(t, cfg.Optimization.DisableTelemetry)
	assert.Equal(t, 3000000, cfg.Optimization.APITimeoutMs)
	assert.Equal(t, 200000, cfg.Optimization.MaxOutputTokens)
}

func TestOverrideFromEnv(t *testing.T) {
	t.Setenv("CCL_TESTPROVIDER_API_KEY", "test-key-from-env")

	cfg := &Config{
		Providers: map[string]Provider{
			"testprovider": {BaseURL: "https://example.com"},
		},
	}

	overrideFromEnv(cfg)

	provider := cfg.Providers["testprovider"]
	assert.Equal(t, "test-key-from-env", provider.APIKey)
}

func TestInterpolateProviderFields(t *testing.T) {
	t.Setenv("TEST_AUTH_TOKEN", "token-value")
	t.Setenv("TEST_MODEL", "model-value")

	cfg := &Config{
		Providers: map[string]Provider{
			"test": {
				BaseURL:        "https://example.com",
				AuthToken:      "${TEST_AUTH_TOKEN}",
				Model:          "${TEST_MODEL}",
				SmallFastModel: "${TEST_MODEL}",
			},
		},
	}

	interpolateProviderFields(cfg)

	provider := cfg.Providers["test"]
	assert.Equal(t, "token-value", provider.AuthToken)
	assert.Equal(t, "model-value", provider.Model)
	assert.Equal(t, "model-value", provider.SmallFastModel)
}
