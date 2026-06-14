package handlers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDoctorTestCmd(cfg *config.Config) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{RunE: HandleDoctor}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("check-net", false, "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(config.StoreInContext(context.Background(), cfg))
	return cmd, &out
}

func doctorTestConfig() *config.Config {
	authReq := true
	return &config.Config{
		Providers: map[string]config.Provider{
			"alpha": {AuthRequired: &authReq, AuthToken: "secret-token", BaseURL: "https://a.example.com/anthropic", Model: "m"},
			"beta":  {AuthRequired: &authReq, BaseURL: "https://b.example.com/anthropic", Model: "m"},
		},
		CLI: config.CLIConfig{DefaultProvider: "alpha"},
	}
}

func TestHandleDoctor_UnknownProvider(t *testing.T) {
	t.Parallel()
	cmd, _ := newDoctorTestCmd(doctorTestConfig())
	require.NoError(t, cmd.Flags().Set("provider", "nope"))
	err := HandleDoctor(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestHandleDoctor_ProviderScoping(t *testing.T) {
	t.Parallel()
	cmd, out := newDoctorTestCmd(doctorTestConfig())
	require.NoError(t, cmd.Flags().Set("provider", "alpha"))
	_ = HandleDoctor(cmd, nil) // alpha has auth → no FAIL
	got := out.String()
	assert.Contains(t, got, "alpha")
	assert.NotContains(t, got, "beta")
}

func TestHandleDoctor_AllProviders(t *testing.T) {
	t.Parallel()
	cmd, out := newDoctorTestCmd(doctorTestConfig())
	_ = HandleDoctor(cmd, nil)
	got := out.String()
	assert.Contains(t, got, "alpha")
	assert.Contains(t, got, "beta")
}

func TestHandleDoctor_FailExitOnMissingAuth(t *testing.T) {
	t.Parallel()
	cmd, _ := newDoctorTestCmd(doctorTestConfig())
	require.NoError(t, cmd.Flags().Set("provider", "beta")) // requires auth, no token
	err := HandleDoctor(cmd, nil)
	require.Error(t, err)
	var ece *ExitCodeError
	require.ErrorAs(t, err, &ece)
	assert.Equal(t, 1, ece.Code)
}

func TestHandleDoctor_JSONNoSecrets(t *testing.T) {
	t.Parallel()
	cmd, out := newDoctorTestCmd(doctorTestConfig())
	require.NoError(t, cmd.Flags().Set("json", "true"))
	_ = HandleDoctor(cmd, nil)
	got := out.String()
	assert.True(t, strings.HasPrefix(strings.TrimSpace(got), "["), "json output should be an array")
	require.NotContains(t, got, "secret-token")
}
