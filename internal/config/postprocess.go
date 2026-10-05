package config

import (
	"os"
	"strings"
)

func defaultOptimization() OptimizationConfig {
	return DefaultConfig().Optimization
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

func overrideFromEnv(cfg *Config) {
	for name, p := range cfg.Providers {
		legacy := "CCL_" + strings.ToUpper(name) + "_API_KEY"
		normalized := strings.ReplaceAll(legacy, "-", "_")
		value, ok := os.LookupEnv(normalized)
		if !ok {
			value, ok = os.LookupEnv(legacy)
		}
		if ok {
			p.AuthToken = value
			p.APIKey = ""
			p.OAuthToken = ""
			cfg.Providers[name] = p
		}
	}
}
