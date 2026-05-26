package handlers

import (
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// HandleProviders prints the configured providers as a table, marking the
// default and showing whether each has auth credentials available.
func HandleProviders(cmd *cobra.Command, _ []string) error {
	cfg, err := configFromCmd(cmd)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	return writeProviders(out, cfg)
}

func writeProviders(out io.Writer, cfg *config.Config) error {
	const format = "%-16s %-48s %-6s %s\n"
	if _, err := fmt.Fprintf(out, format, "PROVIDER", "MODEL", "AUTH", ""); err != nil {
		return err
	}
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
		if _, err := fmt.Fprintf(out, format, name, p.Model, auth, marker); err != nil {
			return err
		}
	}
	return nil
}
