package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dotcommander/cclauncher/internal/utils"
	"gopkg.in/yaml.v3"
)

// parseConfig merges parsed nodes before interpolation so replacement text is
// always scalar data, never YAML syntax or a second environment expression.
func parseConfig(data []byte) (*Config, error) {
	var doc, defaults yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse config: invalid YAML")
	}
	if err := yaml.Unmarshal(defaultConfigYAML, &defaults); err != nil {
		return nil, fmt.Errorf("parse shipped defaults: %w", err)
	}
	if len(doc.Content) == 0 {
		doc = defaults
	} else {
		if err := expandAliases(doc.Content[0], 0); err != nil {
			return nil, err
		}
		if err := normalizeMergeKeys(doc.Content[0]); err != nil {
			return nil, err
		}
		mergeNodes(doc.Content[0], defaults.Content[0])
	}
	if err := interpolateNode(doc.Content[0], reflect.TypeFor[Config](), ""); err != nil {
		return nil, err
	}
	var cfg Config
	if err := doc.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: invalid field type")
	}
	overrideFromEnv(&cfg)
	if cfg.Doctor.ProbeConcurrency <= 0 {
		return nil, fmt.Errorf("doctor.probeConcurrency must be positive")
	}
	for name, p := range cfg.Providers {
		if _, err := ResolveTransformerPresets(p.Transformer); err != nil {
			return nil, fmt.Errorf("provider %q: %w", name, err)
		}
		for i, r := range p.Transformer.Rules {
			if r.ResponseFormat != nil {
				return nil, fmt.Errorf("provider %q rule %d: responseFormat is unsupported", name, i+1)
			}
		}
	}
	return &cfg, nil
}

func mergeNodes(dst, src *yaml.Node) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(src.Content); i += 2 {
		key := src.Content[i].Value
		val := findMapValue(dst, key)
		if val == nil {
			dst.Content = append(dst.Content, src.Content[i], src.Content[i+1])
			continue
		}
		mergeNodes(val, src.Content[i+1])
	}
}

func interpolateNode(n *yaml.Node, typ reflect.Type, path string) error {
	// Preserve YAML null semantics for every field type.
	if n.Tag == "!!null" {
		return nil
	}
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch n.Kind {
	case yaml.AliasNode:
		// Expand aliases to avoid revisiting and reinterpreting substituted values.
		cloned := cloneNode(n.Alias)
		*n = *cloned
		return interpolateNode(n, typ, path)
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			var child reflect.Type
			if typ != nil {
				switch typ.Kind() {
				case reflect.Map:
					child = typ.Elem()
				case reflect.Struct:
					for j := 0; j < typ.NumField(); j++ {
						f := typ.Field(j)
						if strings.Split(f.Tag.Get("yaml"), ",")[0] == key {
							child = f.Type
							break
						}
					}
				}
			}
			field := key
			if path != "" {
				field = path + "." + key
			}
			if err := interpolateNode(n.Content[i+1], child, field); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		var child reflect.Type
		if typ != nil && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) {
			child = typ.Elem()
		}
		for i, v := range n.Content {
			if err := interpolateNode(v, child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		original := n.Value
		value, err := utils.InterpolateEnvVars(original, false)
		if err != nil {
			return fmt.Errorf("interpolate field %s: invalid environment expression", path)
		}
		n.Value = value
		if typ == nil || typ.Kind() == reflect.Interface {
			if value != original {
				n.Tag = "!!str"
			}
			return nil
		}
		switch typ.Kind() {
		case reflect.String:
			n.Tag = "!!str"
		case reflect.Bool:
			if _, err := strconv.ParseBool(value); err != nil {
				return fmt.Errorf("field %s must be boolean", path)
			}
			n.Tag = "!!bool"
		case reflect.Int, reflect.Int64:
			if _, err := strconv.ParseInt(value, 0, 64); err != nil {
				return fmt.Errorf("field %s must be integer", path)
			}
			n.Tag = "!!int"
		case reflect.Float64:
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("field %s must be numeric", path)
			}
			n.Tag = "!!float"
		}
	}
	return nil
}
func cloneNode(n *yaml.Node) *yaml.Node {
	out := *n
	out.Content = make([]*yaml.Node, len(n.Content))
	for i, v := range n.Content {
		out.Content[i] = cloneNode(v)
	}
	return &out
}

func expandAliases(n *yaml.Node, depth int) error {
	if depth > 100 {
		return fmt.Errorf("config contains recursive or excessively nested aliases")
	}
	if n.Kind == yaml.AliasNode {
		*n = *cloneNode(n.Alias)
	}
	for _, v := range n.Content {
		if err := expandAliases(v, depth+1); err != nil {
			return err
		}
	}
	return nil
}
