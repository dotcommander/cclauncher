package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
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

// applyModelFallbacks fills empty model slots with the interpolated shipped
// defaults. Upgraded installs still carry an older first-run template whose
// bare ${X_MODEL} references resolve empty; this lets shipped model defaults
// (zai's GLM pair) apply without rewriting the user's config file.
func applyModelFallbacks(cfg *Config) error {
	var defaults yaml.Node
	if err := yaml.Unmarshal(defaultConfigYAML, &defaults); err != nil {
		return fmt.Errorf("parse shipped defaults: %w", err)
	}
	if err := interpolateNode(defaults.Content[0], reflect.TypeFor[Config](), ""); err != nil {
		return fmt.Errorf("interpolate shipped defaults: %w", err)
	}
	var shipped Config
	if err := defaults.Decode(&shipped); err != nil {
		return fmt.Errorf("decode shipped defaults: %w", err)
	}
	for name, p := range cfg.Providers {
		d, ok := shipped.Providers[name]
		if !ok {
			continue
		}
		if p.Model == "" && d.Model != "" {
			p.Model = d.Model
		}
		if p.SmallFastModel == "" && d.SmallFastModel != "" {
			p.SmallFastModel = d.SmallFastModel
		}
		cfg.Providers[name] = p
	}
	return nil
}
