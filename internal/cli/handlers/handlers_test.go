package handlers

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

func TestExtractProviderFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantProv   string
		wantClaude []string
		wantErr    string
	}{
		{
			name:       "no provider flag",
			args:       []string{"--continue", "-p", "hello"},
			wantProv:   "",
			wantClaude: []string{"--continue", "-p", "hello"},
		},
		{
			name:       "provider with space",
			args:       []string{"--provider", "deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "provider with equals",
			args:       []string{"--provider=deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "provider at end",
			args:       []string{"--continue", "--provider", "deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{"--continue"},
		},
		{
			name:       "provider equals at end",
			args:       []string{"--continue", "--provider=deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{"--continue"},
		},
		{
			name:    "provider flag without value at end",
			args:    []string{"--provider"},
			wantErr: "--provider requires a provider name",
		},
		{
			name:       "empty args",
			args:       []string{},
			wantProv:   "",
			wantClaude: []string{},
		},
		{
			name:       "positional prompt only",
			args:       []string{"hello world"},
			wantProv:   "",
			wantClaude: []string{"hello world"},
		},
		{
			name:       "dangerously-skip-permissions passes through",
			args:       []string{"--dangerously-skip-permissions", "hello"},
			wantProv:   "",
			wantClaude: []string{"--dangerously-skip-permissions", "hello"},
		},
		{
			name:    "provider equals empty value",
			args:    []string{"--provider="},
			wantErr: "--provider requires a provider name",
		},
		{
			name:       "leading short -p claims provider, later -p passes through",
			args:       []string{"-p", "deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "leading short -p claims provider alone",
			args:       []string{"-p", "deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{},
		},
		{
			name:       "non-leading -p passes through to claude",
			args:       []string{"--continue", "-p", "hello"},
			wantProv:   "",
			wantClaude: []string{"--continue", "-p", "hello"},
		},
		{
			name:       "leading short -p with later --provider override",
			args:       []string{"-p", "deepseek", "--provider", "zai", "-p", "hello"},
			wantProv:   "zai",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:    "lone -p at start without value passes through",
			args:    []string{"-p"},
			wantErr: "-p requires a provider name",
		},
		{
			name:    "leading short -p with flag-looking value errors",
			args:    []string{"-p", "--continue"},
			wantErr: "-p requires a provider name",
		},
		{
			name:    "provider flag with flag-looking value errors",
			args:    []string{"--provider", "--continue"},
			wantErr: "--provider requires a provider name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProv, gotClaude, err := extractProviderFromArgs(tt.args)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if gotProv != tt.wantProv {
				t.Errorf("provider = %q, want %q", gotProv, tt.wantProv)
			}
			if !slices.Equal(gotClaude, tt.wantClaude) {
				t.Errorf("claudeArgs = %v, want %v", gotClaude, tt.wantClaude)
			}
		})
	}
}

func TestResolveProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"deepseek": {Model: "deepseek-v4-pro", SmallFastModel: "deepseek-v4-flash"},
			"zai":      {Model: "glm-4.7", SmallFastModel: "glm-4.5-air"},
		},
		CLI: config.CLIConfig{DefaultProvider: "zai"},
		Router: config.RouterConfig{
			Default: "deepseek:deepseek-v4-override",
		},
	}

	t.Run("uses default provider when name empty", func(t *testing.T) {
		t.Parallel()

		name, provider, err := resolveProvider("", cfg)

		require.NoError(t, err)
		require.Equal(t, "zai", name)
		require.Equal(t, "glm-4.7", provider.Model)
		require.Equal(t, "glm-4.5-air", provider.SmallFastModel)
	})

	t.Run("router default overrides matching provider model slots", func(t *testing.T) {
		t.Parallel()

		name, provider, err := resolveProvider("deepseek", cfg)

		require.NoError(t, err)
		require.Equal(t, "deepseek", name)
		require.Equal(t, "deepseek-v4-override", provider.Model)
		require.Equal(t, "deepseek-v4-override", provider.SmallFastModel)
	})

	t.Run("unknown provider returns actionable error", func(t *testing.T) {
		t.Parallel()

		_, _, err := resolveProvider("missing", cfg)

		require.Error(t, err)
		require.Contains(t, err.Error(), "ccl providers")
	})
}

func TestWriteProviders(t *testing.T) {
	t.Parallel()

	authRequired := true
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {
				AuthRequired: &authRequired,
				AuthToken:    "token",
				Model:        "glm-4.7",
			},
			"claude": {
				Model: "claude-cli",
			},
		},
		CLI: config.CLIConfig{DefaultProvider: "zai"},
	}

	var out bytes.Buffer
	if err := writeProviders(&out, cfg); err != nil {
		t.Fatalf("writeProviders returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "PROVIDER") || !strings.Contains(got, "MODEL") || !strings.Contains(got, "AUTH") {
		t.Fatalf("output missing header: %q", got)
	}
	if !strings.Contains(got, "zai") || !strings.Contains(got, "yes") || !strings.Contains(got, "(default)") {
		t.Fatalf("output missing default provider auth row: %q", got)
	}
	if strings.Index(got, "claude") > strings.Index(got, "zai") {
		t.Fatalf("providers are not sorted alphabetically: %q", got)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestWriteProviders_ReturnsWriterError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	err := writeProviders(failingWriter{}, cfg)

	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("writeProviders error = %v, want write failure", err)
	}
}
