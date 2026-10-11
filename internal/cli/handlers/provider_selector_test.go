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
			"zai":      {BaseURL: "https://api.z.ai/api/anthropic", Model: "glm-4.7", AuthToken: "test-token"},
			"deepseek": {BaseURL: "https://api.deepseek.com/anthropic", Model: "deepseek-v4.1-flash", APIKey: "test-key"},
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
			{Name: "deepseek", Model: "deepseek-v4.1-flash"},
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
		require.ErrorContains(t, err, "no launchable providers")
		require.Zero(t, selector.calls)
	})
}

func TestProviderChoicesReadiness(t *testing.T) {
	t.Parallel()
	valid := config.Provider{BaseURL: "https://example.com", Model: "test-model", AuthToken: "test-token"}
	tests := []struct {
		name     string
		change   func(*config.Provider)
		eligible bool
	}{
		{"auth token", func(p *config.Provider) {}, true},
		{"missing credentials", func(p *config.Provider) { p.AuthToken = "" }, false},
		{"api key", func(p *config.Provider) { p.AuthToken = ""; p.APIKey = "test-key" }, true},
		{"oauth token", func(p *config.Provider) { p.AuthToken = ""; p.OAuthToken = "test-oauth" }, true},
		{"optional auth", func(p *config.Provider) { optional := false; p.AuthToken = ""; p.AuthRequired = &optional }, true},
		{"invalid endpoint", func(p *config.Provider) { p.BaseURL = "not-a-url" }, false},
		{"missing model", func(p *config.Provider) { p.Model = "" }, false},
		{"native login", func(p *config.Provider) {
			p.BaseURL = "https://api.anthropic.com"
			p.Model = ""
			p.AuthToken = ""
			optional := false
			p.AuthRequired = &optional
		}, true},
		{"invalid transformer", func(p *config.Provider) { p.Transformer.Use = []string{"unknown-preset"} }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := valid
			tt.change(&p)
			choices := providerChoices(&config.Config{Providers: map[string]config.Provider{"test": p}})
			if tt.eligible {
				require.Equal(t, []ProviderChoice{{Name: "test", Model: p.Model}}, choices)
			} else {
				require.Empty(t, choices)
			}
		})
	}
}

func TestPickProviderUnavailable(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"ready":        {BaseURL: "https://example.com", Model: "test-model", APIKey: "test-key"},
			"missing-auth": {BaseURL: "https://example.com", Model: "test-model"},
		},
		CLI: config.CLIConfig{DefaultProvider: "missing-auth"},
	}
	t.Run("unavailable default is excluded", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{selected: "ready"}
		name, err := pickProvider(context.Background(), cfg, selector)
		require.NoError(t, err)
		require.Equal(t, "ready", name)
		require.Equal(t, []ProviderChoice{{Name: "ready", Model: "test-model"}}, selector.choices)
	})
	t.Run("selector cannot return excluded provider", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{selected: "missing-auth"}
		_, err := pickProvider(context.Background(), cfg, selector)
		require.ErrorContains(t, err, "not launchable")
	})
	t.Run("no eligible providers does not open picker", func(t *testing.T) {
		t.Parallel()
		selector := &fakeProviderSelector{}
		unavailable := &config.Config{Providers: map[string]config.Provider{"missing-auth": cfg.Providers["missing-auth"]}}
		_, err := pickProvider(context.Background(), unavailable, selector)
		require.ErrorContains(t, err, "no launchable providers")
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
