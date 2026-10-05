package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// normalizeMergeKeys makes inherited YAML fields explicit before applying
// shipped defaults. Explicit fields win; earlier sequence mappings win over
// later mappings, following yaml.v3's merge-key semantics. Aliases have already
// been expanded, so cloned inherited values can be interpolated exactly once.
func normalizeMergeKeys(n *yaml.Node) error {
	for _, child := range n.Content {
		if err := normalizeMergeKeys(child); err != nil {
			return err
		}
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	var inherited *yaml.Node
	explicit := make([]*yaml.Node, 0, len(n.Content))
	present := make(map[string]bool)
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if key.Tag == "!!merge" && key.Value == "<<" {
			if inherited != nil {
				return fmt.Errorf("config contains duplicate YAML merge keys")
			}
			inherited = value
			continue
		}
		explicit = append(explicit, key, value)
		present[key.Value] = true
	}
	if inherited == nil {
		return nil
	}
	sources := []*yaml.Node{inherited}
	if inherited.Kind == yaml.SequenceNode {
		sources = inherited.Content
	}
	for _, source := range sources {
		if source.Kind != yaml.MappingNode {
			return fmt.Errorf("config YAML merge value must contain mappings")
		}
		for i := 0; i+1 < len(source.Content); i += 2 {
			key, value := source.Content[i], source.Content[i+1]
			if present[key.Value] {
				continue
			}
			explicit = append(explicit, cloneNode(key), cloneNode(value))
			present[key.Value] = true
		}
	}
	n.Content = explicit
	return nil
}
