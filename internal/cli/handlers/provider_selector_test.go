package handlers

import (
	"bytes"
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

func TestPickProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":      {Model: "glm-4.7"},
			"deepseek": {Model: "deepseek-v4-pro"},
		},
		CLI: config.CLIConfig{DefaultProvider: "zai"},
	}

	t.Run("picker receives sorted choices and current provider", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{selected: "deepseek"}
		name, err := pickProvider(context.Background(), cfg, selector)
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
		_, err := pickProvider(context.Background(), cfg, selector)
		require.ErrorContains(t, err, "selection canceled")
	})

	t.Run("picker result must be configured", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{selected: "missing"}
		_, err := pickProvider(context.Background(), cfg, selector)
		require.ErrorContains(t, err, "unknown provider")
	})

	t.Run("empty config does not open picker", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{selected: "missing"}
		_, err := pickProvider(context.Background(), &config.Config{}, selector)
		require.ErrorContains(t, err, "no providers configured")
		require.Zero(t, selector.calls)
	})
}

func TestIsInteractive(t *testing.T) {
	t.Parallel()
	require.False(t, isInteractive(&bytes.Buffer{}))
}

func TestTerminalProviderSelectorCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	selector := TerminalProviderSelector{In: &bytes.Buffer{}, Out: &bytes.Buffer{}}
	_, err := selector.SelectProvider(ctx, []ProviderChoice{{Name: "zai"}}, "zai")
	require.ErrorIs(t, err, context.Canceled)
}
