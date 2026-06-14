package handlers

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/dotcommander/cclauncher/internal/actions"
	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestConfigFromCmdRequiresLoadedConfig(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	cfg, err := configFromCmd(cmd)

	require.Nil(t, cfg)
	require.ErrorIs(t, err, errConfigNotLoaded)
}

func TestUnknownProviderErrorIncludesRecoveryHint(t *testing.T) {
	t.Parallel()

	err := unknownProviderError("missing")

	require.ErrorContains(t, err, `unknown provider "missing"`)
	require.ErrorContains(t, err, "ccl providers")
	require.ErrorContains(t, err, "config:")
}

func TestSelectProviders(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":      {Model: "glm-4.7"},
			"deepseek": {Model: "deepseek-v4-pro"},
		},
		CLI: config.CLIConfig{DefaultProvider: "zai"},
	}

	t.Run("all providers sorted", func(t *testing.T) {
		t.Parallel()

		got, err := selectProviders("", cfg)

		require.NoError(t, err)
		require.Equal(t, []string{"deepseek", "zai"}, got)
	})

	t.Run("single provider validates name", func(t *testing.T) {
		t.Parallel()

		got, err := selectProviders("zai", cfg)

		require.NoError(t, err)
		require.Equal(t, []string{"zai"}, got)
	})

	t.Run("unknown provider returns hint", func(t *testing.T) {
		t.Parallel()

		_, err := selectProviders("missing", cfg)

		require.ErrorContains(t, err, "ccl providers")
	})
}

func TestWriteDoctorTableAndAnyFail(t *testing.T) {
	t.Parallel()

	results := []actions.CheckResult{
		{Provider: "alpha", Check: "auth", Status: actions.StatusPass, Reason: "ok"},
		{Provider: "beta", Check: "auth", Status: actions.StatusFail, Reason: "missing token"},
	}

	var out bytes.Buffer
	err := writeDoctorTable(&out, results)

	require.NoError(t, err)
	require.Contains(t, out.String(), "PROVIDER")
	require.Contains(t, out.String(), "alpha")
	require.Contains(t, out.String(), "beta")
	require.True(t, anyFail(results))
	require.False(t, anyFail(results[:1]))
}

func TestExitCodeErrorSupportsErrorAs(t *testing.T) {
	t.Parallel()

	err := error(&ExitCodeError{Code: 7, Msg: "failed"})

	var got *ExitCodeError
	require.True(t, errors.As(err, &got))
	require.Equal(t, 7, got.Code)
	require.Equal(t, "failed", got.Error())
}
