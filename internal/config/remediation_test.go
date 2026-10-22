package config

import (
	"errors"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestParsedPresenceAndLiteralValues(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]byte(`providers:
  deepseek:
    model: ""
    authRequired: false
    authToken: '${CCL_TEST_MISSING_442:-a: b # c}'
optimization:
  disableTelemetry: false
  apiTimeoutMs: '${CCL_TEST_MISSING_442:-42}'
doctor:
  probeConcurrency: '${CCL_TEST_MISSING_442:-1}'
`))
	require.NoError(t, err)
	p := cfg.Providers["deepseek"]
	require.Empty(t, p.Model)
	require.False(t, p.RequiresAuth())
	require.NotEmpty(t, p.BaseURL)
	require.Equal(t, "a: b # c", p.AuthToken)
	require.False(t, cfg.Optimization.DisableTelemetry)
	require.True(t, cfg.Optimization.DisableAutoupdater)
	require.Equal(t, 42, cfg.Optimization.APITimeoutMs)
	require.Equal(t, 1, cfg.Doctor.ProbeConcurrency)
}
func TestScalarReplacementIsNotReinterpreted(t *testing.T) {
	t.Setenv("CCL_CONFIG_LITERAL", "quote\"\n# comment: yes\n${OTHER_SECRET}")
	t.Setenv("OTHER_SECRET", "should-not-appear")
	cfg, err := parseConfig([]byte("providers:\n  custom:\n    authToken: '${CCL_CONFIG_LITERAL}'\n    model: 123\n"))
	require.NoError(t, err)
	require.Equal(t, os.Getenv("CCL_CONFIG_LITERAL"), cfg.Providers["custom"].AuthToken)
	require.Equal(t, "123", cfg.Providers["custom"].Model)
}
func TestCredentialOverrideNormalizedPresence(t *testing.T) {
	t.Setenv("CCL_MY_PROVIDER_API_KEY", "")
	t.Setenv("CCL_MY-PROVIDER_API_KEY", "legacy")
	cfg := &Config{Providers: map[string]Provider{"my-provider": {APIKey: "key", AuthToken: "token", OAuthToken: "oauth"}}}
	overrideFromEnv(cfg)
	require.False(t, cfg.Providers["my-provider"].HasAuth())
}
func TestConfigFieldErrorsDoNotExposeValues(t *testing.T) {
	t.Parallel()
	_, err := parseConfig([]byte("doctor:\n  probeConcurrency: '${CCL_TEST_MISSING_442:-private-secret}'\n"))
	require.ErrorContains(t, err, "doctor.probeConcurrency")
	require.NotContains(t, err.Error(), "private-secret")
	_, err = parseConfig([]byte("doctor:\n  probeConcurrency: 0\n"))
	require.Error(t, err)
}
func TestPresetsDeclarativeRoundTripAndIsolation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("providers:\n  custom:\n    transformer:\n      use: [clamp-max-tokens-8192]\n      rules:\n        - modifyBody:\n            nested: [one]\n"), 0600))
	l := NewLoaderWithPaths(dir, path)
	cfg, err := l.Load()
	require.NoError(t, err)
	original := cfg.Providers["custom"].Transformer
	resolved, err := ResolveTransformerPresets(original)
	require.NoError(t, err)
	require.Len(t, resolved.Rules, 2)
	require.Empty(t, resolved.Use)
	again, err := ResolveTransformerPresets(resolved)
	require.NoError(t, err)
	require.Len(t, again.Rules, 2)
	resolved.Rules[1].ModifyBody["nested"].([]any)[0] = "changed"
	require.Equal(t, "one", original.Rules[0].ModifyBody["nested"].([]any)[0])
	require.NoError(t, l.Save(cfg))
	cfg, err = l.Load()
	require.NoError(t, err)
	require.Len(t, cfg.Providers["custom"].Transformer.Rules, 1)
	require.Equal(t, []string{"clamp-max-tokens-8192"}, cfg.Providers["custom"].Transformer.Use)
}
func TestAtomicSymlinksAndFailureCleanup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(target, []byte("old"), 0640))
	require.NoError(t, os.Symlink(target, link))
	require.NoError(t, writeFileAtomic(link, []byte("new"), 0600))
	info, err := os.Lstat(link)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink)
	info, err = os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
	dangling := filepath.Join(dir, "dangling")
	require.NoError(t, os.Symlink(filepath.Join(dir, "absent"), dangling))
	created, err := NewLoaderWithPaths(dir, dangling).EnsureExists()
	require.Error(t, err)
	require.False(t, created)
	require.Error(t, writeFileAtomic(dangling, []byte("bad"), 0600))
	leftovers, err := filepath.Glob(filepath.Join(dir, ".config-*.tmp"))
	require.NoError(t, err)
	require.Empty(t, leftovers)
}
func TestLoaderInitializationFailure(t *testing.T) {
	t.Parallel()
	l := NewLoaderWithPaths(t.TempDir(), "unused")
	l.initErr = errors.New("home unavailable")
	_, err := l.Load()
	require.ErrorContains(t, err, "home unavailable")
	_, err = l.EnsureExists()
	require.ErrorContains(t, err, "home unavailable")
	require.ErrorContains(t, l.Save(&Config{}), "home unavailable")
}
func TestSharedProviderValidation(t *testing.T) {
	t.Parallel()
	for _, p := range []Provider{{BaseURL: "https://api.anthropic.com"}, {BaseURL: "http://localhost:8080", Model: "local"}} {
		require.NoError(t, ValidateProvider(p))
	}
	for _, p := range []Provider{{BaseURL: "relative", Model: "x"}, {BaseURL: "ftp://host", Model: "x"}, {BaseURL: "https://host"}, {BaseURL: "https://api.anthropic.com", Transformer: TransformerConfig{Use: []string{"clamp-max-tokens-8192"}}}} {
		require.Error(t, ValidateProvider(p))
	}
	require.ErrorContains(t, ValidateProvider(Provider{BaseURL: "https://host", Model: "x", Transformer: TransformerConfig{Rules: []TransformRule{{ResponseFormat: &ResponseFormat{}}}}}), "rule 1")
}

func TestYAMLMergePresencePrecedenceAndLiteralKeys(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]byte(`anchor: &custom
  baseUrl: "http://localhost:9999"
  model: ""
  authRequired: false
  authToken: '${CCL_TEST_MISSING_442:-literal: #value}'
second: &second
  baseUrl: "http://localhost:8888"
  model: "second"
providers:
  deepseek:
    <<: [*custom, *second]
  synthetic:
    <<: *custom
    baseUrl: ""
    model: "explicit"
  '${CCL_TEST_MISSING_442:-changed}':
    <<: *custom
`))
	require.NoError(t, err)
	first := cfg.Providers["deepseek"]
	require.Equal(t, "http://localhost:9999", first.BaseURL)
	require.Empty(t, first.Model)
	require.False(t, first.RequiresAuth())
	require.Equal(t, "literal: #value", first.AuthToken)
	require.Equal(t, os.Getenv("DEEPSEEK_SMALL_MODEL"), first.SmallFastModel, "omitted models inherit only the user's environment selection")
	second := cfg.Providers["synthetic"]
	require.Empty(t, second.BaseURL)
	require.Equal(t, "explicit", second.Model)
	require.False(t, second.RequiresAuth())
	require.Contains(t, cfg.Providers, "${CCL_TEST_MISSING_442:-changed}")
	require.NotContains(t, cfg.Providers, "changed")
}

func TestYAMLMergeNestedAndQuotedKey(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]byte(`base: &base
  disableTelemetry: false
middle: &middle
  <<: *base
  disableAutoupdater: false
optimization:
  <<: *middle
providers:
  custom:
    transformer:
      rules:
        - modifyBody:
            "<<": "literal-key"
`))
	require.NoError(t, err)
	require.False(t, cfg.Optimization.DisableTelemetry)
	require.False(t, cfg.Optimization.DisableAutoupdater)
	require.True(t, cfg.Optimization.DisableErrorReporting)
	require.Equal(t, "literal-key", cfg.Providers["custom"].Transformer.Rules[0].ModifyBody["<<"])
}

func TestYAMLNullPreservesDecoderSemantics(t *testing.T) {
	for _, null := range []string{"null", "~", ""} {
		t.Run(null, func(t *testing.T) {
			cfg, err := parseConfig([]byte("providers:\n  custom:\n    model: " + null + "\n    authToken: " + null + "\n    authRequired: " + null + "\n    maxContextTokens: " + null + "\noptimization:\n  disableTelemetry: " + null + "\n  apiTimeoutMs: " + null + "\n"))
			require.NoError(t, err)
			require.Empty(t, cfg.Providers["custom"].Model)
			require.Empty(t, cfg.Providers["custom"].AuthToken)
			require.Nil(t, cfg.Providers["custom"].AuthRequired)
			require.Zero(t, cfg.Providers["custom"].MaxContextTokens)
			require.False(t, cfg.Optimization.DisableTelemetry)
			require.Zero(t, cfg.Optimization.APITimeoutMs)
		})
	}
	cfg, err := parseConfig([]byte("providers:\n  custom:\n    model: 'null'\n    authToken: '~'\n"))
	require.NoError(t, err)
	require.Equal(t, "null", cfg.Providers["custom"].Model)
	require.Equal(t, "~", cfg.Providers["custom"].AuthToken)
}
