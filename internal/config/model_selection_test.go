package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestShippedModelsAreOptIn(t *testing.T) {
	t.Parallel()
	var templates Config
	require.NoError(t, yaml.Unmarshal(defaultConfigYAML, &templates))
	cfg, err := parseConfig(defaultConfigYAML)
	require.NoError(t, err)

	for _, name := range expectedProviders {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			template := templates.Providers[name]
			loaded := cfg.Providers[name]
			if name == "claude" {
				require.Empty(t, template.Model)
				require.Empty(t, template.SmallFastModel)
				require.Empty(t, loaded.Model)
				require.Empty(t, loaded.SmallFastModel)
				return
			}
			primary := strings.ToUpper(name) + "_MODEL"
			small := strings.ToUpper(name) + "_SMALL_MODEL"
			if name == "zai" {
				// zai ships GLM 1M-context defaults; env vars still override.
				envOrDefault := func(key, fallback string) string {
				if value := os.Getenv(key); value != "" {
					return value
				}
				return fallback
			}
			require.Equal(t, "${"+primary+":-glm-5.3[1m]}", template.Model)
			require.Equal(t, "${"+small+":-glm-5.3-flash[1m]}", template.SmallFastModel)
			require.Equal(t, envOrDefault(primary, "glm-5.3[1m]"), loaded.Model)
			require.Equal(t, envOrDefault(small, "glm-5.3-flash[1m]"), loaded.SmallFastModel)
			return
			}
			require.Equal(t, "${"+primary+"}", template.Model)
			require.Equal(t, "${"+small+"}", template.SmallFastModel)
			require.Equal(t, os.Getenv(primary), loaded.Model)
			require.Equal(t, os.Getenv(small), loaded.SmallFastModel)
		})
	}
}

func TestModelSelectionPreservesExplicitValues(t *testing.T) {
	t.Parallel()
	for _, name := range expectedProviders {
		for _, model := range []string{"", "user-selected-model"} {
			t.Run(name+"/"+model, func(t *testing.T) {
				t.Parallel()
				input := Config{Providers: map[string]Provider{
					name: {Model: model, SmallFastModel: model},
				}}
				data, err := yaml.Marshal(input)
				require.NoError(t, err)
				cfg, err := parseConfig(data)
				require.NoError(t, err)
				if name == "zai" && model == "" {
					// Omitted models fall back to zai's shipped GLM defaults.
					require.Equal(t, "glm-5.3[1m]", cfg.Providers[name].Model)
					require.Equal(t, "glm-5.3-flash[1m]", cfg.Providers[name].SmallFastModel)
					return
				}
				require.Equal(t, model, cfg.Providers[name].Model)
				require.Equal(t, model, cfg.Providers[name].SmallFastModel)
			})
		}
	}
}

func TestOmittedModelsInheritOnlyUserEnvironment(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]byte(`providers:
  deepseek:
    baseUrl: "https://api.deepseek.com/anthropic"
  omlx:
    baseUrl: "http://localhost:8000"
`))
	require.NoError(t, err)
	for _, name := range []string{"deepseek", "omlx"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, os.Getenv(strings.ToUpper(name)+"_MODEL"), cfg.Providers[name].Model)
			require.Equal(t, os.Getenv(strings.ToUpper(name)+"_SMALL_MODEL"), cfg.Providers[name].SmallFastModel)
		})
	}
}

func TestUpgradedTemplateGainsShippedModelDefaults(t *testing.T) {
	t.Parallel()
	// Pre-upgrade first-run configs pin the bare template, which resolves
	// empty when ZAI_MODEL is unset; shipped defaults must still apply.
	cfg, err := parseConfig([]byte(`providers:
  zai:
    authRequired: true
    baseUrl: "https://api.z.ai/api/anthropic"
    authToken: "${ZAI_API_KEY}"
    model: "${ZAI_MODEL}"
    smallFastModel: "${ZAI_SMALL_MODEL}"
`))
	require.NoError(t, err)
	envOrDefault := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return fallback
	}
	require.Equal(t, envOrDefault("ZAI_MODEL", "glm-5.3[1m]"), cfg.Providers["zai"].Model)
	require.Equal(t, envOrDefault("ZAI_SMALL_MODEL", "glm-5.3-flash[1m]"), cfg.Providers["zai"].SmallFastModel)
}

func TestFirstRunWritesOptInModels(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	created, err := NewLoaderWithPaths(dir, path).EnsureExists()
	require.NoError(t, err)
	require.True(t, created)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, data)
}

func TestNonNativeModelSelectionRequired(t *testing.T) {
	t.Parallel()
	for _, baseURL := range []string{"https://api.example.com", "http://localhost:8000"} {
		t.Run(baseURL, func(t *testing.T) {
			t.Parallel()
			require.ErrorContains(t, ValidateProvider(Provider{BaseURL: baseURL}), "model must be configured explicitly")
			require.ErrorContains(t, ValidateProvider(Provider{BaseURL: baseURL, Model: "  "}), "model")
			require.NoError(t, ValidateProvider(Provider{BaseURL: baseURL, Model: "user-selected-model"}))
		})
	}
	require.NoError(t, ValidateProvider(Provider{BaseURL: "https://api.anthropic.com"}))
}
