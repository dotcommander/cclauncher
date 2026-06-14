package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestConfigPath() string {
	return filepath.Join("testdata", "config.yaml")
}

// expectedProviders is the canonical provider set shipped in testdata/config.yaml.
var expectedProviders = []string{
	"synthetic", "deepseek", "minimax", "zai", "openrouter", "wafer", "claude",
	"llamabarn", "lmstudio", "llamacpp", "omlx",
}

func TestGetProvider_AllProvidersExist(t *testing.T) {
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err, "Failed to load test config")

	for _, name := range expectedProviders {
		_, exists := GetProvider(cfg, name)
		assert.True(t, exists, "Provider %s should exist", name)
	}
}

func TestGetProvider_ProvidersHaveBaseURL(t *testing.T) {
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	for _, name := range expectedProviders {
		t.Run(name, func(t *testing.T) {
			provider, exists := GetProvider(cfg, name)
			assert.True(t, exists, "Provider %s should exist", name)
			assert.NotEmpty(t, provider.BaseURL, "Provider %s should have BaseURL", name)
		})
	}
}

func TestGetProvider_BaseURLs(t *testing.T) {
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	tests := []struct {
		name         string
		providerName string
		expectedURL  string
	}{
		{"synthetic", "synthetic", "https://api.synthetic.new/anthropic"},
		{"deepseek", "deepseek", "https://api.deepseek.com/anthropic"},
		{"minimax", "minimax", "https://api.minimax.io/anthropic"},
		{"zai", "zai", "https://api.z.ai/api/anthropic"},
		{"openrouter", "openrouter", "https://openrouter.ai/api"},
		{"wafer", "wafer", "https://pass.wafer.ai"},
		{"claude", "claude", "https://api.anthropic.com"},
		{"llamabarn", "llamabarn", "http://localhost:2276"},
		{"lmstudio", "lmstudio", "http://localhost:1234"},
		{"llamacpp", "llamacpp", "http://localhost:8080"},
		{"omlx", "omlx", "http://localhost:8000"},
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
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	deepseek, _ := GetProvider(cfg, "deepseek")
	assert.Equal(t, "deepseek-v4-pro", deepseek.Model)
	assert.Equal(t, "deepseek-v4-flash", deepseek.SmallFastModel)

	synthetic, _ := GetProvider(cfg, "synthetic")
	assert.Equal(t, "hf:zai-org/GLM-4.7", synthetic.Model)
	assert.Equal(t, "hf:zai-org/GLM-4.7", synthetic.SmallFastModel)

	openrouter, _ := GetProvider(cfg, "openrouter")
	assert.Equal(t, "deepseek/deepseek-v4-pro", openrouter.Model)
	assert.Equal(t, "deepseek/deepseek-v3.2", openrouter.SmallFastModel)
}

func TestGetProvider_AuthenticationMethods(t *testing.T) {
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Providers with bearer-token auth via authToken env-interpolation.
	bearerAuthProviders := []string{"synthetic", "deepseek", "minimax", "zai", "openrouter", "wafer"}
	for _, name := range bearerAuthProviders {
		t.Run(name+" uses auth token", func(t *testing.T) {
			provider, _ := GetProvider(cfg, name)
			// Auth tokens resolve from env vars at load time;
			// may be empty in test environments if env vars aren't set.
			assert.Empty(t, provider.APIKey, "Provider %s should not use APIKey field", name)
			assert.True(t, provider.RequiresAuth(), "Provider %s should require auth", name)
		})
	}

	// Optional-auth providers rely on either the claude CLI keychain or local
	// servers that accept placeholder/no credentials.
	optionalAuthProviders := []string{"claude", "llamabarn", "lmstudio", "llamacpp", "omlx"}
	for _, name := range optionalAuthProviders {
		t.Run(name+" does not require configured auth", func(t *testing.T) {
			provider, _ := GetProvider(cfg, name)
			assert.False(t, provider.RequiresAuth(), "Provider %s should not require auth", name)
		})
	}

	t.Run("claude uses cli oauth", func(t *testing.T) {
		claude, _ := GetProvider(cfg, "claude")
		assert.Empty(t, claude.APIKey)
		assert.Empty(t, claude.AuthToken)
	})

	t.Run("omlx ships without embedded auth token", func(t *testing.T) {
		omlx, _ := GetProvider(cfg, "omlx")
		assert.Empty(t, omlx.AuthToken)
		assert.Empty(t, omlx.APIKey)
	})
}

func TestLoad_MergesDefaultAuthPolicyForExistingConfigs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	data := []byte(`providers:
  claude:
    baseUrl: "https://api.anthropic.com"
  deepseek:
    baseUrl: "https://api.deepseek.com/anthropic"
    model: "deepseek-v4-pro"
  custom:
    baseUrl: "https://api.example.com/anthropic"
cli:
  defaultProvider: "claude"
`)
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	loader := NewLoaderWithPaths(tmpDir, configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)

	require.False(t, cfg.Providers["claude"].RequiresAuth(), "known optional-auth provider policy should merge from defaults")
	require.True(t, cfg.Providers["deepseek"].RequiresAuth(), "known required-auth provider policy should merge from defaults")
	require.True(t, cfg.Providers["custom"].RequiresAuth(), "custom provider should fail closed without authRequired: false")
	require.Contains(t, cfg.Providers, "wafer", "new shipped providers should be available to existing configs")
	require.Contains(t, cfg.Providers, "omlx", "new shipped providers should be available to existing configs")
	require.Equal(t, "https://pass.wafer.ai", cfg.Providers["wafer"].BaseURL)
	require.Equal(t, "https://api.example.com/anthropic", cfg.Providers["custom"].BaseURL)
}

func TestEnsureExists_CreatesDefaultConfig(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// Use the testdata directory which has a config file
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())

	created, err := loader.EnsureExists()
	assert.NoError(t, err)
	assert.False(t, created, "Should not create a new config when one already exists")
}

func TestLoad_Success(t *testing.T) {
	t.Parallel()
	loader := NewLoaderWithPaths("testdata", getTestConfigPath())
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Providers)
	assert.Equal(t, "zai", cfg.CLI.DefaultProvider)
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	loader := NewLoaderWithPaths(tmpDir, filepath.Join(tmpDir, "nonexistent.yaml"))

	_, err := loader.Load()
	assert.Error(t, err)
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Providers: map[string]Provider{
			"test": {BaseURL: "https://example.com"},
		},
	}

	applyDefaults(cfg)

	// Should apply CLI defaults
	assert.Equal(t, "zai", cfg.CLI.DefaultProvider)

	// Should apply optimization defaults
	assert.True(t, cfg.Optimization.DisableNonessentialTraffic)
	assert.True(t, cfg.Optimization.DisableTelemetry)
	assert.Equal(t, 3000000, cfg.Optimization.APITimeoutMs)
	assert.Equal(t, 200000, cfg.Optimization.MaxOutputTokens)
}

func TestApplyOptDefaults_PreservesExplicitBooleansAndFillsNumericZeros(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Optimization: OptimizationConfig{
			DisableTelemetry: true,
			APITimeoutMs:     42,
		},
	}

	applyOptDefaults(cfg)

	assert.False(t, cfg.Optimization.DisableNonessentialTraffic)
	assert.False(t, cfg.Optimization.DisableAutoupdater)
	assert.True(t, cfg.Optimization.DisableTelemetry)
	assert.False(t, cfg.Optimization.DisableErrorReporting)
	assert.False(t, cfg.Optimization.DisableCostWarnings)
	assert.Equal(t, 42, cfg.Optimization.APITimeoutMs)
	assert.Equal(t, 200000, cfg.Optimization.MaxOutputTokens)
	assert.Equal(t, 8192, cfg.Optimization.NodeMaxOldSpaceSize)
}

func TestMergeDefaultProviders_PreservesUserProviderFields(t *testing.T) {
	t.Parallel()

	authRequired := false
	cfg := &Config{
		Providers: map[string]Provider{
			"deepseek": {
				BaseURL:        "https://custom.example.com/anthropic",
				AuthToken:      "custom-token",
				AuthRequired:   &authRequired,
				Model:          "custom-model",
				SmallFastModel: "custom-small",
			},
		},
	}

	mergeDefaultProviders(cfg)

	deepseek := cfg.Providers["deepseek"]
	assert.Equal(t, "https://custom.example.com/anthropic", deepseek.BaseURL)
	assert.Equal(t, "custom-token", deepseek.AuthToken)
	assert.Equal(t, "custom-model", deepseek.Model)
	assert.Equal(t, "custom-small", deepseek.SmallFastModel)
	assert.False(t, deepseek.RequiresAuth())
	assert.Contains(t, cfg.Providers, "claude")
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

func TestOverrideFromEnv_DoesNotClearExistingAPIKeyWhenEnvEmpty(t *testing.T) {
	t.Setenv("CCL_TESTPROVIDER_API_KEY", "")

	cfg := &Config{
		Providers: map[string]Provider{
			"testprovider": {APIKey: "configured-key"},
		},
	}

	overrideFromEnv(cfg)

	assert.Equal(t, "configured-key", cfg.Providers["testprovider"].APIKey)
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
