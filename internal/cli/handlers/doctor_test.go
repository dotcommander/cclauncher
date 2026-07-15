package handlers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runDoctorTest(cfg *config.Config, opts DoctorOptions) (*bytes.Buffer, error) {
	var out bytes.Buffer
	err := HandleDoctor(context.Background(), &out, cfg, opts)
	return &out, err
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
	_, err := runDoctorTest(doctorTestConfig(), DoctorOptions{Provider: "nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestHandleDoctor_ProviderScoping(t *testing.T) {
	t.Parallel()
	out, _ := runDoctorTest(doctorTestConfig(), DoctorOptions{Provider: "alpha"})
	got := out.String()
	assert.Contains(t, got, "alpha")
	assert.NotContains(t, got, "beta")
}

func TestHandleDoctor_AllProviders(t *testing.T) {
	t.Parallel()
	out, _ := runDoctorTest(doctorTestConfig(), DoctorOptions{})
	got := out.String()
	assert.Contains(t, got, "alpha")
	assert.Contains(t, got, "beta")
}

func TestHandleDoctor_NoFailOnMissingAuth(t *testing.T) {
	t.Parallel()
	_, err := runDoctorTest(doctorTestConfig(), DoctorOptions{Provider: "beta"})
	require.NoError(t, err)
}

func TestHandleDoctor_JSONNoSecrets(t *testing.T) {
	t.Parallel()
	out, _ := runDoctorTest(doctorTestConfig(), DoctorOptions{JSON: true})
	got := out.String()
	assert.True(t, strings.HasPrefix(strings.TrimSpace(got), "["), "json output should be an array")
	require.NotContains(t, got, "secret-token")
}
