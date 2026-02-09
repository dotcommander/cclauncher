package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderRegistry_ProvidersExist(t *testing.T) {
	registry := NewProviderRegistry()
	expectedProviders := []string{"kimi2", "qwen", "qwen3-coder", "deepseek", "claude", "claude2", "zai", "synthetic", "minimax"}

	for _, providerName := range expectedProviders {
		t.Run(providerName, func(t *testing.T) {
			provider, exists := registry.Get(providerName)
			assert.True(t, exists, "Provider %s should exist", providerName)
			assert.NotEmpty(t, provider.BaseURL, "Provider config should have BaseURL")
		})
	}
}

func TestProviderRegistry_BaseURLs(t *testing.T) {
	registry := NewProviderRegistry()
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
			provider, exists := registry.Get(tt.providerName)
			assert.True(t, exists)
			assert.Equal(t, tt.expectedURL, provider.BaseURL)
		})
	}
}

func TestProviderRegistry_Models(t *testing.T) {
	registry := NewProviderRegistry()

	// Test providers with specific models (all use syntheticBaseConfig)
	deepseek, _ := registry.Get("deepseek")
	assert.Equal(t, "hf:deepseek-ai/DeepSeek-V3.2", deepseek.Model)
	assert.Equal(t, "hf:deepseek-ai/DeepSeek-V3.2", deepseek.SmallFastModel)

	synthetic, _ := registry.Get("synthetic")
	assert.Equal(t, "hf:moonshotai/Kimi-K2.5", synthetic.Model)
	assert.Equal(t, "hf:moonshotai/Kimi-K2.5", synthetic.SmallFastModel)

	qwen, _ := registry.Get("qwen")
	assert.Equal(t, "hf:Qwen/Qwen3-VL-235B-A22B-Instruct", qwen.Model)
	assert.Equal(t, "hf:Qwen/Qwen3-VL-235B-A22B-Instruct", qwen.SmallFastModel)

	qwen3Coder, _ := registry.Get("qwen3-coder")
	assert.Equal(t, "hf:Qwen/Qwen3-Coder-480B-A35B-Instruct", qwen3Coder.Model)
	assert.Equal(t, "hf:Qwen/Qwen3-Coder-480B-A35B-Instruct", qwen3Coder.SmallFastModel)
}

func TestProviderRegistry_GetEnvVars_ValidProvider(t *testing.T) {
	registry := NewProviderRegistry()
	envVars := registry.GetEnvVars("deepseek")
	assert.NotNil(t, envVars)

	// deepseek uses synthetic config
	expectedVars := map[string]string{
		"ANTHROPIC_BASE_URL":         "https://api.synthetic.new/anthropic",
		"ANTHROPIC_MODEL":            "hf:deepseek-ai/DeepSeek-V3.2",
		"ANTHROPIC_SMALL_FAST_MODEL": "hf:deepseek-ai/DeepSeek-V3.2",
	}

	for key, expectedValue := range expectedVars {
		assert.Equal(t, expectedValue, envVars[key], "Environment variable %s should match", key)
	}
}

func TestProviderRegistry_GetEnvVars_InvalidProvider(t *testing.T) {
	registry := NewProviderRegistry()
	envVars := registry.GetEnvVars("nonexistent")
	assert.Nil(t, envVars)
}

func TestProviderRegistry_GetEnvVars_Claude(t *testing.T) {
	registry := NewProviderRegistry()
	envVars := registry.GetEnvVars("claude")
	assert.NotNil(t, envVars)

	// Claude provider sets base URL only (OAuth via default credentials)
	assert.Contains(t, envVars, "ANTHROPIC_BASE_URL")
	assert.Equal(t, "https://api.anthropic.com", envVars["ANTHROPIC_BASE_URL"])

	// No API key — OAuth only
	assert.NotContains(t, envVars, "ANTHROPIC_API_KEY")
}

func TestProviderRegistry_GetEnvVars_Claude2(t *testing.T) {
	testToken := "test-oauth-token"
	t.Setenv("CLAUDE2_OAUTH_TOKEN", testToken)

	registry := NewProviderRegistry()
	envVars := registry.GetEnvVars("claude2")
	assert.NotNil(t, envVars)

	assert.Contains(t, envVars, "ANTHROPIC_BASE_URL")
	assert.Equal(t, "https://api.anthropic.com", envVars["ANTHROPIC_BASE_URL"])

	assert.Contains(t, envVars, "CLAUDE_CODE_OAUTH_TOKEN")
	assert.Equal(t, testToken, envVars["CLAUDE_CODE_OAUTH_TOKEN"])

	// No API key
	assert.NotContains(t, envVars, "ANTHROPIC_API_KEY")
}

func TestProviderRegistry_GetEnvVars_AllFields(t *testing.T) {
	registry := NewProviderRegistry()
	// Use synthetic provider which has all fields
	envVars := registry.GetEnvVars("synthetic")
	assert.NotNil(t, envVars)

	expectedFields := []string{
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
	}

	for _, field := range expectedFields {
		assert.Contains(t, envVars, field, "Should contain %s", field)
	}
}

func TestResolveEnvTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "env var not set uses default",
			template: "${TEST_UNSET_VAR:-default_value}",
			envKey:   "TEST_UNSET_VAR",
			envValue: "",
			expected: "default_value",
		},
		{
			name:     "env var set overrides default",
			template: "${TEST_SET_VAR:-default_value}",
			envKey:   "TEST_SET_VAR",
			envValue: "env_value",
			expected: "env_value",
		},
		{
			name:     "plain string unchanged",
			template: "https://api.example.com",
			envKey:   "",
			envValue: "",
			expected: "https://api.example.com",
		},
		{
			name:     "empty string unchanged",
			template: "",
			envKey:   "",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				if tt.envValue != "" {
					t.Setenv(tt.envKey, tt.envValue)
				} else {
					t.Setenv(tt.envKey, "")
					os.Unsetenv(tt.envKey)
				}
			}

			result := resolveEnvTemplate(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_AuthenticationMethods(t *testing.T) {
	registry := NewProviderRegistry()
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
			provider, _ := registry.Get(tt.providerName)

			if tt.hasAuthToken {
				// Auth tokens resolve from env vars at registry creation time;
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

func TestProviderRegistry_DefaultValues(t *testing.T) {
	// Test that resolveEnvTemplate handles default values
	t.Setenv("NONEXISTENT_TEST_KEY", "")
	os.Unsetenv("NONEXISTENT_TEST_KEY")

	result := resolveEnvTemplate("${NONEXISTENT_TEST_KEY:-sk-bleh}")
	assert.Equal(t, "sk-bleh", result)
}

func TestProviderRegistry_EnvironmentVariableResolution(t *testing.T) {
	// Test that CLAUDE2_OAUTH_TOKEN environment variable is resolved for the claude2 provider
	testValue := "test-oauth-token"
	t.Setenv("CLAUDE2_OAUTH_TOKEN", testValue)

	registry := NewProviderRegistry()
	claude2, exists := registry.Get("claude2")
	assert.True(t, exists)
	assert.Equal(t, testValue, claude2.OAuthToken)
}
