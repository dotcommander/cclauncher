package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// SetDefaultProvider selects a provider and clears any remembered model pair.
func SetDefaultProvider(name string) error {
	return NewLoader().SetDefaultSelection(name, Provider{})
}

// SetDefaultSelection remembers only the successful selection, never credentials
// or the expanded provider configuration.
func SetDefaultSelection(name string, provider Provider) error {
	return NewLoader().SetDefaultSelection(name, provider)
}

// SetDefaultSelection updates CLI defaults in the latest on-disk document while
// preserving comments, environment templates, unknown fields, and symlinks.
func (l *Loader) SetDefaultSelection(name string, provider Provider) error {
	if l.initErr != nil {
		return l.initErr
	}
	if name == "" {
		return fmt.Errorf("default provider must not be empty")
	}
	file, err := os.Open(l.configFile)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	defer file.Close()
	const maxConfigBytes = 10 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if len(data) > maxConfigBytes {
		return fmt.Errorf("config exceeds size limit")
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse config: invalid YAML")
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("config must contain a YAML mapping")
	}
	root := doc.Content[0]
	cli := findMapValue(root, "cli")
	if cli == nil {
		cli = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cli"}, cli)
	}
	if cli.Kind != yaml.MappingNode {
		return fmt.Errorf("cli must be a YAML mapping")
	}
	for _, field := range []struct{ key, value string }{
		{"defaultProvider", name},
		{"defaultModel", provider.Model},
		{"defaultSmallFastModel", provider.SmallFastModel},
	} {
		value := findMapValue(cli, field.key)
		if value == nil {
			value = &yaml.Node{}
			cli.Content = append(cli.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field.key}, value)
		}
		value.Kind, value.Tag, value.Value = yaml.ScalarNode, "!!str", field.value
		value.Content, value.Alias = nil, nil
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return writeFileAtomic(l.configFile, out, 0600)
}
