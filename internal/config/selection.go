package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// SetDefaultProvider selects a provider and clears any remembered model pair.
// Administrative, so it rewrites every field instead of guarding against edits
// made since some earlier launch.
func SetDefaultProvider(name string) error {
	_, err := NewLoader().SetDefaultSelection(name, Provider{}, nil)
	return err
}

// SetDefaultSelection remembers only the successful selection, never credentials
// or the expanded provider configuration. Fields whose on-disk value changed
// since the session launched keep their on-disk value and are reported by name.
func SetDefaultSelection(name string, provider Provider, launched CLIConfig) ([]string, error) {
	return NewLoader().SetDefaultSelection(name, provider, &launched)
}

// SetDefaultSelection updates CLI defaults in the latest on-disk document while
// preserving comments, environment templates, unknown fields, and symlinks.
// With a non-nil launched baseline, a field is rewritten only when its on-disk
// value still equals the launch-time value; a field the user edited mid-session
// keeps the newer on-disk value and is reported by name. A nil baseline
// rewrites every field.
func (l *Loader) SetDefaultSelection(name string, provider Provider, launched *CLIConfig) ([]string, error) {
	if l.initErr != nil {
		return nil, l.initErr
	}
	if name == "" {
		return nil, fmt.Errorf("default provider must not be empty")
	}
	file, err := os.Open(l.configFile)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	defer file.Close()
	const maxConfigBytes = 10 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if len(data) > maxConfigBytes {
		return nil, fmt.Errorf("config exceeds size limit")
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse config: invalid YAML")
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config must contain a YAML mapping")
	}
	root := doc.Content[0]
	cli := findMapValue(root, "cli")
	if cli == nil {
		cli = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cli"}, cli)
	}
	if cli.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("cli must be a YAML mapping")
	}
	var baseline CLIConfig
	if launched != nil {
		baseline = *launched
	}
	var preserved []string
	for _, field := range []struct{ key, value, baseline string }{
		{"defaultProvider", name, baseline.DefaultProvider},
		{"defaultModel", provider.Model, baseline.DefaultModel},
		{"defaultSmallFastModel", provider.SmallFastModel, baseline.DefaultSmallFastModel},
	} {
		value := findMapValue(cli, field.key)
		current := ""
		if value != nil {
			current = value.Value
		}
		if launched != nil && current != field.baseline {
			// The user edited this field mid-session; keep the newer on-disk value.
			preserved = append(preserved, field.key)
			continue
		}
		if value == nil {
			value = &yaml.Node{}
			cli.Content = append(cli.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field.key}, value)
		}
		value.Kind, value.Tag, value.Value = yaml.ScalarNode, "!!str", field.value
		value.Content, value.Alias = nil, nil
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return preserved, writeFileAtomic(l.configFile, out, 0600)
}
