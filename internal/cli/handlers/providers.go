package handlers

import (
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/cobra"
)

// HandleProviders prints the configured providers as a table, marking the
// default and showing whether each has auth credentials available.
func HandleProviders(cmd *cobra.Command, _ []string) error {
	cfg, err := configFromCmd(cmd)
	if err != nil {
		return err
	}

	const format = "%-16s %-48s %-6s %s\n"
	fmt.Printf(format, "PROVIDER", "MODEL", "AUTH", "")
	for _, name := range slices.Sorted(maps.Keys(cfg.Providers)) {
		p := cfg.Providers[name]
		auth := "no"
		if p.HasAuth() {
			auth = "yes"
		}
		marker := ""
		if name == cfg.CLI.DefaultProvider {
			marker = "(default)"
		}
		fmt.Printf(format, name, p.Model, auth, marker)
	}
	return nil
}
