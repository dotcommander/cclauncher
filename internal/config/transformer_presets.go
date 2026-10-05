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
	Min int `json:"min,omitempty" yaml:"min"`
	Max int `json:"max,omitempty" yaml:"max"`
} {
	return struct {
		Min int `json:"min,omitempty" yaml:"min"`
		Max int `json:"max,omitempty" yaml:"max"`
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
		return TransformerConfig{Rules: cloneTransformRules(t.Rules)}, nil
	}

	rules := make([]TransformRule, 0, len(t.Rules)+len(t.Use))
	for _, name := range t.Use {
		presetRules, ok := transformerPresetRegistry[name]
		if !ok {
			return TransformerConfig{}, fmt.Errorf("unknown transformer preset %q", name)
		}
		rules = append(rules, cloneTransformRules(presetRules)...)
	}
	rules = append(rules, cloneTransformRules(t.Rules)...)
	t.Rules = rules
	t.Use = nil
	return t, nil
}

func cloneTransformRules(in []TransformRule) []TransformRule {
	out := make([]TransformRule, len(in))
	for i, rule := range in {
		out[i] = rule
		if rule.SetTemperature != nil {
			v := *rule.SetTemperature
			out[i].SetTemperature = &v
		}
		if rule.ResponseFormat != nil {
			v := *rule.ResponseFormat
			v.JSONSchema = cloneAnyMap(v.JSONSchema)
			out[i].ResponseFormat = &v
		}
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
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneAny(v)
	}
	return out
}
func cloneAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneAnyMap(x)
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = cloneAny(v)
		}
		return out
	default:
		return v
	}
}
