package handlers

import (
	"fmt"
	"sort"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

func HandleProviders(cmd *cobra.Command, args []string) error {
	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("config not loaded (internal error)")
	}

	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("%-16s %-48s %-6s %s\n", "PROVIDER", "MODEL", "AUTH", "")
	for _, name := range names {
		p := cfg.Providers[name]

		auth := "no"
		if p.HasAuth() {
			auth = "yes"
		}

		marker := ""
		if name == cfg.CLI.DefaultProvider {
			marker = "(default)"
		}

		fmt.Printf("%-16s %-48s %-6s %s\n", name, p.Model, auth, marker)
	}

	return nil
}
