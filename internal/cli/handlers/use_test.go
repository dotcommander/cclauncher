package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

type fakeProviderSelector struct {
	selected string
	err      error
	calls    int
	choices  []ProviderChoice
	current  string
}

func (f *fakeProviderSelector) SelectProvider(_ context.Context, choices []ProviderChoice, current string) (string, error) {
	f.calls++
	f.choices = append([]ProviderChoice(nil), choices...)
	f.current = current
	if f.err != nil {
		return "", f.err
	}
	return f.selected, nil
}

func TestProviderNameForUse(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":      {Model: "glm-4.7"},
			"deepseek": {Model: "deepseek-v4-pro"},
		},
		CLI: config.CLIConfig{DefaultProvider: "zai"},
	}

	t.Run("argument bypasses selector", func(t *testing.T) {
		t.Parallel()

		selector := &fakeProviderSelector{selected: "zai"}
		name, err := providerNameForUse(context.Background(), cfg, []string{"deepseek"}, selector)

		require.NoError(t, err)
		require.Equal(t, "deepseek", name)
		require.Zero(t, selector.calls)
	})

	t.Run("picker receives sorted choices and current provider", func(t *testing.T) {
		t.Parallel()

		selector := &fakeProviderSelector{selected: "deepseek"}
		name, err := providerNameForUse(context.Background(), cfg, nil, selector)

		require.NoError(t, err)
		require.Equal(t, "deepseek", name)
		require.Equal(t, 1, selector.calls)
		require.Equal(t, "zai", selector.current)
		require.Equal(t, []ProviderChoice{
			{Name: "deepseek", Model: "deepseek-v4-pro"},
			{Name: "zai", Model: "glm-4.7"},
		}, selector.choices)
	})

	t.Run("picker error is returned", func(t *testing.T) {
		t.Parallel()

		selector := &fakeProviderSelector{err: errors.New("selection canceled")}
		_, err := providerNameForUse(context.Background(), cfg, nil, selector)

		require.ErrorContains(t, err, "selection canceled")
	})

	t.Run("picker result must be configured", func(t *testing.T) {
		t.Parallel()

		selector := &fakeProviderSelector{selected: "missing"}
		_, err := providerNameForUse(context.Background(), cfg, nil, selector)

		require.ErrorContains(t, err, "unknown provider")
	})

	t.Run("empty config does not open picker", func(t *testing.T) {
		t.Parallel()

		selector := &fakeProviderSelector{selected: "missing"}
		_, err := providerNameForUse(context.Background(), &config.Config{}, nil, selector)

		require.ErrorContains(t, err, "no providers configured")
		require.Zero(t, selector.calls)
	})
}
