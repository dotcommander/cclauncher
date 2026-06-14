package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveTransformerPresets(t *testing.T) {
	t.Parallel()

	t.Run("known preset expands before inline rules", func(t *testing.T) {
		t.Parallel()

		inlineRule := TransformRule{SetModel: "provider-model"}
		cfg := TransformerConfig{
			Use:   []string{"clamp-max-tokens-8192"},
			Rules: []TransformRule{inlineRule},
		}

		resolved, err := ResolveTransformerPresets(cfg)

		require.NoError(t, err)
		require.Len(t, resolved.Rules, 2)
		require.Equal(t, 8193, resolved.Rules[0].TokenRange.Min)
		require.Equal(t, 8192, resolved.Rules[0].SetMaxTokens)
		require.Equal(t, "provider-model", resolved.Rules[1].SetModel)
		require.Equal(t, []string{"clamp-max-tokens-8192"}, resolved.Use)
	})

	t.Run("unknown preset rejects config", func(t *testing.T) {
		t.Parallel()

		_, err := ResolveTransformerPresets(TransformerConfig{
			Use: []string{"does-not-exist"},
		})

		require.ErrorContains(t, err, `unknown transformer preset "does-not-exist"`)
	})

	t.Run("preset rules are cloned", func(t *testing.T) {
		t.Parallel()

		first, err := ResolveTransformerPresets(TransformerConfig{Use: []string{"clamp-max-tokens-8192"}})
		require.NoError(t, err)
		first.Rules[0].SetMaxTokens = 123

		second, err := ResolveTransformerPresets(TransformerConfig{Use: []string{"clamp-max-tokens-8192"}})
		require.NoError(t, err)
		require.Equal(t, 8192, second.Rules[0].SetMaxTokens)
	})
}

func TestLoad_TransformerUseRejectsUnknownPreset(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	data := []byte(`providers:
  custom:
    baseUrl: "https://api.example.com/anthropic"
    authRequired: false
    transformer:
      use:
        - missing-preset
cli:
  defaultProvider: "custom"
`)
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	_, err := NewLoaderWithPaths(tmpDir, configPath).Load()

	require.ErrorContains(t, err, `provider "custom": unknown transformer preset "missing-preset"`)
}
