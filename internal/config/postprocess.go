package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func defaultOptimization() OptimizationConfig {
	return OptimizationConfig{
		DisableNonessentialTraffic: true,
		DisableAutoupdater:         true,
		DisableTelemetry:           true,
		DisableErrorReporting:      true,
		DisableCostWarnings:        true,
		APITimeoutMs:               3000000,
		MaxOutputTokens:            200000,
		NodeMaxOldSpaceSize:        8192,
	}
}

func applyDefaults(cfg *Config) {
	applyCLIDefaults(cfg)
	applyOptDefaults(cfg)
}

func applyCLIDefaults(cfg *Config) {
	if cfg.CLI.DefaultProvider == "" {
		cfg.CLI.DefaultProvider = DefaultConfig().CLI.DefaultProvider
	}
}

func applyOptDefaults(cfg *Config) {
	defaults := defaultOptimization()

	if cfg.Optimization == (OptimizationConfig{}) {
		cfg.Optimization = defaults
		return
	}
	if cfg.Optimization.APITimeoutMs == 0 {
		cfg.Optimization.APITimeoutMs = defaults.APITimeoutMs
	}
	if cfg.Optimization.MaxOutputTokens == 0 {
		cfg.Optimization.MaxOutputTokens = defaults.MaxOutputTokens
	}
	if cfg.Optimization.NodeMaxOldSpaceSize == 0 {
		cfg.Optimization.NodeMaxOldSpaceSize = defaults.NodeMaxOldSpaceSize
	}
}

// mergeDefaultProviderMetadata carries non-secret provider policy from the
// embedded defaults into existing user configs that predate those fields.
func mergeDefaultProviderMetadata(cfg *Config) {
	var defaults Config
	if err := yaml.Unmarshal(defaultConfigYAML, &defaults); err != nil {
		return
	}

	for name, provider := range cfg.Providers {
		defaultProvider, ok := defaults.Providers[name]
		if !ok {
			continue
		}
		if provider.AuthRequired == nil {
			provider.AuthRequired = defaultProvider.AuthRequired
			cfg.Providers[name] = provider
		}
	}
}

// interpolateProviderFields resolves any ${VAR} templates in provider fields
// that were filled by applyDefaults after the initial YAML interpolation pass.
func interpolateProviderFields(cfg *Config) {
	for name, p := range cfg.Providers {
		cfg.Providers[name] = p.interpolated()
	}
}

func overrideFromEnv(cfg *Config) {
	for name, provider := range cfg.Providers {
		envKey := "CCL_" + strings.ToUpper(name) + "_API_KEY"
		if apiKey := os.Getenv(envKey); apiKey != "" {
			provider.APIKey = apiKey
			cfg.Providers[name] = provider
		}
	}
}
