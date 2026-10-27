package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProviderEndpointRejectsQueryAndFragment(t *testing.T) {
	t.Parallel()
	for _, baseURL := range []string{
		"https://api.example.com/?key=secret",
		"https://api.example.com?key=secret",
		"https://api.example.com/?", // bare trailing "?" — ForceQuery
		"https://api.example.com/#/dashboard",
		"https://api.example.com/anthropic?key=x#frag",
	} {
		t.Run(baseURL, func(t *testing.T) {
			t.Parallel()
			err := ValidateProviderEndpoint(Provider{BaseURL: baseURL, Model: "m"})
			require.ErrorContains(t, err, "baseUrl must not include")
		})
	}
	require.NoError(t, ValidateProviderEndpoint(Provider{BaseURL: "https://api.example.com/anthropic", Model: "m"}))
	require.NoError(t, ValidateProviderEndpoint(Provider{BaseURL: "http://localhost:8000", Model: "m"}))
}
