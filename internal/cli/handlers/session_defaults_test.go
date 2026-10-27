package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRememberSuccessfulSession(t *testing.T) {
	t.Parallel()
	for _, transformed := range []bool{false, true} {
		name := "direct"
		if transformed {
			name = "transformed"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := config.Provider{BaseURL: "http://localhost:8000", AuthToken: "local", Model: "configured", SmallFastModel: "small"}
			if transformed {
				p.Transformer.Use = []string{"clamp-max-tokens-8192"}
			}
			cfg := &config.Config{Providers: map[string]config.Provider{"omlx": p}, CLI: config.CLIConfig{DefaultProvider: "omlx"}}
			var finished, remembered bool
			deps := launchDependencies{
				run: func(_ context.Context, p config.Provider, _ config.OptimizationConfig, args []string, _ CodeOptions) error {
					require.Equal(t, "override", p.Model)
					require.Equal(t, []string{"--model=override", "--print", "hello"}, args)
					require.Equal(t, transformed, p.Transformer.HasRules())
					finished = true
					return nil
				},
				remember: func(name string, p config.Provider, launched config.CLIConfig) ([]string, error) {
					require.True(t, finished)
					require.Equal(t, "omlx", name)
					require.Equal(t, "override", p.Model)
					require.Equal(t, "small", p.SmallFastModel)
					require.Equal(t, cfg.CLI, launched)
					remembered = true
					return nil, nil
				},
			}
			err := handleCode(context.Background(), cfg, []string{"--provider", "omlx", "--model=override", "--print", "hello"}, CodeOptions{Output: io.Discard, ErrorOutput: io.Discard}, deps)
			require.NoError(t, err)
			require.True(t, remembered)
		})
	}
}

func TestUnsuccessfulAndMetadataLaunchesDoNotRemember(t *testing.T) {
	t.Parallel()
	optional := false
	for _, tc := range []struct {
		name string
		args []string
		err  error
	}{
		{"failure", []string{"--print", "hello"}, errors.New("login expired")},
		{"help", []string{"--help"}, nil},
		{"version", []string{"--version"}, nil},
		{"admin", []string{"mcp", "list"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{Providers: map[string]config.Provider{"local": {BaseURL: "http://localhost:8000", Model: "local", AuthRequired: &optional}}, CLI: config.CLIConfig{DefaultProvider: "local"}}
			deps := launchDependencies{
				run: func(context.Context, config.Provider, config.OptimizationConfig, []string, CodeOptions) error {
					return tc.err
				},
				remember: func(string, config.Provider, config.CLIConfig) ([]string, error) {
					t.Fatal("must not remember")
					return nil, nil
				},
			}
			err := handleCode(context.Background(), cfg, tc.args, CodeOptions{Output: io.Discard, ErrorOutput: io.Discard}, deps)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestRememberedPairIsUsedByNextLaunchAndPicker(t *testing.T) {
	t.Parallel()
	optional := false
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"local": {BaseURL: "http://localhost:8000", Model: "configured", SmallFastModel: "configured-small", AuthRequired: &optional},
			"other": {BaseURL: "http://localhost:1234", Model: "other", AuthRequired: &optional},
		},
		CLI: config.CLIConfig{DefaultProvider: "local", DefaultModel: "remembered", DefaultSmallFastModel: "remembered-small"},
	}
	name, p, err := resolveProvider("", cfg)
	require.NoError(t, err)
	require.Equal(t, "local", name)
	require.Equal(t, "remembered", p.Model)
	require.Equal(t, "remembered-small", p.SmallFastModel)
	require.Equal(t, "remembered", providerChoices(cfg)[0].Model)
	_, p, err = resolveProvider("other", cfg)
	require.NoError(t, err)
	require.Equal(t, "other", p.Model)
	require.Equal(t, "new-model", selectedModel(p, []string{"--model", "new-model"}).Model)
}

func TestRememberFailureWarnsButPreservesSuccess(t *testing.T) {
	t.Parallel()
	optional := false
	cfg := &config.Config{Providers: map[string]config.Provider{"local": {BaseURL: "http://localhost:8000", Model: "local", AuthRequired: &optional}}, CLI: config.CLIConfig{DefaultProvider: "local"}}
	deps := launchDependencies{
		run: func(context.Context, config.Provider, config.OptimizationConfig, []string, CodeOptions) error {
			return nil
		},
		remember: func(string, config.Provider, config.CLIConfig) ([]string, error) {
			return nil, errors.New("disk unavailable")
		},
	}
	var errOut bytes.Buffer
	err := handleCode(context.Background(), cfg, nil, CodeOptions{Output: io.Discard, ErrorOutput: &errOut}, deps)
	require.NoError(t, err)
	require.Contains(t, errOut.String(), "Warning: session succeeded but saving default selection failed")
	require.Contains(t, errOut.String(), "disk unavailable")
}

func TestPreservedFieldsWarnButPreserveSuccess(t *testing.T) {
	t.Parallel()
	optional := false
	cfg := &config.Config{Providers: map[string]config.Provider{"local": {BaseURL: "http://localhost:8000", Model: "local", AuthRequired: &optional}}, CLI: config.CLIConfig{DefaultProvider: "local"}}
	deps := launchDependencies{
		run: func(context.Context, config.Provider, config.OptimizationConfig, []string, CodeOptions) error {
			return nil
		},
		remember: func(string, config.Provider, config.CLIConfig) ([]string, error) {
			return []string{"defaultModel", "defaultSmallFastModel"}, nil
		},
	}
	var errOut bytes.Buffer
	err := handleCode(context.Background(), cfg, nil, CodeOptions{Output: io.Discard, ErrorOutput: &errOut}, deps)
	require.NoError(t, err)
	require.Contains(t, errOut.String(), "Warning: config defaultModel, defaultSmallFastModel changed since launch")
}

func TestExplicitPrimaryAliasRestoresConfiguredModel(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--model", "opus"}, {"--model=sonnet"}, {"--model", "default"}, {"--model", "haiku", "--model", "opus"}} {
		for _, router := range []string{"", "local:routed"} {
			t.Run(strings.Join(args, " ")+router, func(t *testing.T) {
				optional := false
				cfg := &config.Config{
					Providers: map[string]config.Provider{"local": {BaseURL: "http://localhost:8000", Model: "primary", SmallFastModel: "small", AuthRequired: &optional}},
					CLI:       config.CLIConfig{DefaultProvider: "local", DefaultModel: "remembered", DefaultSmallFastModel: "remembered-small"},
				}
				cfg.Router.Default = router
				want, small := "primary", "small"
				if router != "" {
					want, small = "routed", "routed"
				}
				deps := launchDependencies{
					run: func(_ context.Context, p config.Provider, _ config.OptimizationConfig, actual []string, _ CodeOptions) error {
						require.Equal(t, want, p.Model)
						require.Equal(t, small, p.SmallFastModel)
						require.Equal(t, args, actual)
						return nil
					},
					remember: func(_ string, p config.Provider, _ config.CLIConfig) ([]string, error) {
						require.Equal(t, want, p.Model)
						return nil, nil
					},
				}
				require.NoError(t, handleCode(context.Background(), cfg, args, CodeOptions{Output: io.Discard, ErrorOutput: io.Discard}, deps))
				require.Equal(t, "primary", cfg.Providers["local"].Model)
			})
		}
	}
}
