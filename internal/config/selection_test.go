package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRememberSelectionPreservesConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "# keep comment\nproviders:\n  omlx:\n    authToken: '${OMLX_API_KEY:-omlx}'\n    model: '${OMLX_MODEL:-configured}'\n    maxContextTokens: '${CCL_TEST_CONTEXT_981:-131072}'\ncli:\n  defaultProvider: zai # keep default comment\nextra: keep-me\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0600))
	loader := NewLoaderWithPaths(dir, path)
	selected := Provider{Model: "opus-omlx", SmallFastModel: "small-local", AuthToken: "must-not-persist"}
	require.NoError(t, loader.SetDefaultSelection("omlx", selected))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, preserved := range []string{"# keep comment", "# keep default comment", "${OMLX_API_KEY:-omlx}", "${OMLX_MODEL:-configured}", "extra: keep-me"} {
		require.Contains(t, string(data), preserved)
	}
	require.NotContains(t, string(data), selected.AuthToken)
	cfg, err := loader.Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "omlx", DefaultModel: selected.Model, DefaultSmallFastModel: selected.SmallFastModel}, cfg.CLI)
	require.Equal(t, 131072, cfg.Providers["omlx"].MaxContextTokens)

	// A later successful provider with an empty native model clears the old pair.
	require.NoError(t, loader.SetDefaultSelection("claude", Provider{}))
	cfg, err = loader.Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "claude"}, cfg.CLI)
}

func TestRememberSelectionCreatesCLIAndPreservesSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yaml")
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(target, []byte("providers: {}\n"), 0640))
	require.NoError(t, os.Symlink(target, path))
	require.NoError(t, NewLoaderWithPaths(dir, path).SetDefaultSelection("local", Provider{Model: "custom"}))
	info, err := os.Lstat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink)
	info, err = os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	var cfg Config
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.Equal(t, "local", cfg.CLI.DefaultProvider)
	require.Equal(t, "custom", cfg.CLI.DefaultModel)
}

func TestRememberSelectionRejectsMalformedConfig(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"- not-a-mapping\n", "cli: scalar\n", "cli: [invalid\n"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(input), 0600))
			require.Error(t, NewLoaderWithPaths(dir, path).SetDefaultSelection("omlx", Provider{Model: "local"}))
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, input, string(data))
		})
	}
}
