package config

import (
	"fmt"
	"slices"
)

const transformerPresetClampMaxTokens8192 = "clamp-max-tokens-8192"

// transformerPresetRegistry maps stable preset names to concrete rules. Preset
// rules run before inline provider rules, so local config can add final
// provider-specific mutations after the shared preset behavior.
var transformerPresetRegistry = map[string][]TransformRule{
	transformerPresetClampMaxTokens8192: {
		{
			TokenRange:   tokenRange(8193, 0),
			SetMaxTokens: 8192,
		},
	},
}

func tokenRange(min, max int) struct {
	Min int `json:"min,omitempty" yaml:"min,omitempty"`
	Max int `json:"max,omitempty" yaml:"max,omitempty"`
} {
	return struct {
		Min int `json:"min,omitempty" yaml:"min,omitempty"`
		Max int `json:"max,omitempty" yaml:"max,omitempty"`
	}{Min: min, Max: max}
}

// TransformerPresetNames returns the names of built-in transformer presets.
func TransformerPresetNames() []string {
	names := make([]string, 0, len(transformerPresetRegistry))
	for name := range transformerPresetRegistry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ResolveTransformerPresets expands transformer.use into concrete rules.
func ResolveTransformerPresets(t TransformerConfig) (TransformerConfig, error) {
	if len(t.Use) == 0 {
		return t, nil
	}

	rules := make([]TransformRule, 0, len(t.Rules)+len(t.Use))
	for _, name := range t.Use {
		presetRules, ok := transformerPresetRegistry[name]
		if !ok {
			return TransformerConfig{}, fmt.Errorf("unknown transformer preset %q", name)
		}
		rules = append(rules, cloneTransformRules(presetRules)...)
	}
	rules = append(rules, t.Rules...)
	t.Rules = rules
	return t, nil
}

func resolveTransformerPresets(cfg *Config) error {
	for name, provider := range cfg.Providers {
		resolved, err := ResolveTransformerPresets(provider.Transformer)
		if err != nil {
			return fmt.Errorf("provider %q: %w", name, err)
		}
		provider.Transformer = resolved
		cfg.Providers[name] = provider
	}
	return nil
}

func cloneTransformRules(in []TransformRule) []TransformRule {
	out := make([]TransformRule, len(in))
	for i, rule := range in {
		out[i] = rule
		if rule.AddHeaders != nil {
			out[i].AddHeaders = cloneStringMap(rule.AddHeaders)
		}
		if rule.ModifyBody != nil {
			out[i].ModifyBody = cloneAnyMap(rule.ModifyBody)
		}
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
