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
	preserved, err := loader.SetDefaultSelection("omlx", selected, &CLIConfig{DefaultProvider: "zai"})
	require.NoError(t, err)
	require.Empty(t, preserved)
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
	launched := cfg.CLI
	preserved, err = loader.SetDefaultSelection("claude", Provider{}, &launched)
	require.NoError(t, err)
	require.Empty(t, preserved)
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
	preserved, err := NewLoaderWithPaths(dir, path).SetDefaultSelection("local", Provider{Model: "custom"}, &CLIConfig{})
	require.NoError(t, err)
	require.Empty(t, preserved)
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
			_, err := NewLoaderWithPaths(dir, path).SetDefaultSelection("omlx", Provider{Model: "local"}, &CLIConfig{})
			require.Error(t, err)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, input, string(data))
		})
	}
}

func TestRememberSelectionAcceptsEmptyDocument(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", " \n", "# comment only\n"} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(input), 0600))
			loader := NewLoaderWithPaths(dir, path)
			preserved, err := loader.SetDefaultSelection("omlx", Provider{Model: "chosen"}, &CLIConfig{})
			require.NoError(t, err)
			require.Empty(t, preserved)
			cfg, err := loader.Load()
			require.NoError(t, err)
			require.Equal(t, "omlx", cfg.CLI.DefaultProvider)
			require.Equal(t, "chosen", cfg.CLI.DefaultModel)
		})
	}
}

func TestRememberSelectionKeepsMidSessionEdits(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "providers: {}\ncli:\n  defaultProvider: zai\n  defaultModel: fresh\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0600))
	loader := NewLoaderWithPaths(dir, path)
	// The session launched with zai and no remembered model, but the user
	// switched defaultModel on disk while it ran: keep the fresh edit.
	preserved, err := loader.SetDefaultSelection("zai", Provider{Model: "stale", SmallFastModel: "stale-small"}, &CLIConfig{DefaultProvider: "zai"})
	require.NoError(t, err)
	require.Equal(t, []string{"defaultModel"}, preserved)
	cfg, err := loader.Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "zai", DefaultModel: "fresh", DefaultSmallFastModel: "stale-small"}, cfg.CLI)

	// Untouched fields still take the session's values once the baseline
	// matches what the user left on disk.
	preserved, err = loader.SetDefaultSelection("omlx", Provider{Model: "stale", SmallFastModel: "stale-small"}, &CLIConfig{DefaultProvider: "zai", DefaultModel: "fresh", DefaultSmallFastModel: "stale-small"})
	require.NoError(t, err)
	require.Empty(t, preserved)
	cfg, err = loader.Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "omlx", DefaultModel: "stale", DefaultSmallFastModel: "stale-small"}, cfg.CLI)

	// A field the user deleted mid-session (absent on disk) stays deleted
	// instead of being restored from the session's values.
	require.NoError(t, os.WriteFile(path, []byte("providers: {}\ncli:\n  defaultProvider: omlx\n"), 0600))
	preserved, err = loader.SetDefaultSelection("omlx", Provider{Model: "newmodel", SmallFastModel: "newsmall"}, &CLIConfig{DefaultProvider: "omlx", DefaultModel: "stale", DefaultSmallFastModel: "stale-small"})
	require.NoError(t, err)
	require.Equal(t, []string{"defaultModel", "defaultSmallFastModel"}, preserved)
	cfg, err = loader.Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "omlx"}, cfg.CLI)
}

func TestSetDefaultProviderReplacesFieldsAndClearsPair(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("providers: {}\ncli:\n  defaultProvider: zai\n  defaultModel: old\n  defaultSmallFastModel: old-small\n"), 0600))
	// The nil baseline is SetDefaultProvider's administrative path: every field
	// is rewritten even though the on-disk values differ from any baseline.
	preserved, err := NewLoaderWithPaths(dir, path).SetDefaultSelection("omlx", Provider{}, nil)
	require.NoError(t, err)
	require.Empty(t, preserved)
	cfg, err := NewLoaderWithPaths(dir, path).Load()
	require.NoError(t, err)
	require.Equal(t, CLIConfig{DefaultProvider: "omlx"}, cfg.CLI)
}
