package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPartialProviderCredentialInheritance(t *testing.T) {
	t.Setenv("SYNTHETIC_API_KEY", "template-key")
	t.Setenv("CCL_SYNTHETIC_API_KEY", "")
	require.NoError(t, os.Unsetenv("CCL_SYNTHETIC_API_KEY"))
	t.Setenv("LMSTUDIO_API_KEY", "")
	t.Setenv("CCL_LMSTUDIO_API_KEY", "")
	require.NoError(t, os.Unsetenv("CCL_LMSTUDIO_API_KEY"))
	for _, tc := range []struct{ name, field, want string }{
		{"omitted", "", "template-key"},
		{"empty", `authToken: ""`, ""},
		{"null", "authToken: null", ""},
		{"explicit", "authToken: config-key", "config-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseConfig([]byte("providers:\n  synthetic:\n    model: custom\n    " + tc.field + "\n"))
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Providers["synthetic"].AuthToken)
			t.Setenv("CCL_SYNTHETIC_API_KEY", "override-key")
			cfg, err = parseConfig([]byte("providers:\n  synthetic:\n    authToken: ''\n    apiKey: old-key\n    oauthToken: old-token\n"))
			require.NoError(t, err)
			require.Equal(t, "override-key", cfg.Providers["synthetic"].AuthToken)
			require.Empty(t, cfg.Providers["synthetic"].APIKey)
			require.Empty(t, cfg.Providers["synthetic"].OAuthToken)
		})
	}
	cfg, err := parseConfig([]byte("providers:\n  lmstudio:\n    model: local-model\n"))
	require.NoError(t, err)
	require.Equal(t, "lmstudio", cfg.Providers["lmstudio"].AuthToken)
}

func TestMergeCredentialFieldsRespectPresence(t *testing.T) {
	for _, key := range []string{"authToken", "apiKey", "oauthToken"} {
		for _, value := range []string{"", `""`, "null", "custom"} {
			t.Run(key+"/"+value, func(t *testing.T) {
				var dst, src yaml.Node
				data := "providers:\n  custom:\n    model: custom\n"
				if value != "" {
					data += "    " + key + ": " + value + "\n"
				}
				require.NoError(t, yaml.Unmarshal([]byte(data), &dst))
				require.NoError(t, yaml.Unmarshal([]byte("providers:\n  custom:\n    "+key+": inherited\n"), &src))
				mergeNodes(dst.Content[0], src.Content[0])
				var cfg Config
				require.NoError(t, dst.Decode(&cfg))
				p := cfg.Providers["custom"]
				got := map[string]string{"authToken": p.AuthToken, "apiKey": p.APIKey, "oauthToken": p.OAuthToken}[key]
				want := value
				if value == "" {
					want = "inherited"
				}
				if value == `""` || value == "null" {
					want = ""
				}
				require.Equal(t, want, got)
			})
		}
	}
}
