package handlers

import (
	"io"
	"maps"
	"slices"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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
	const authCol = 2
	names := slices.Sorted(maps.Keys(cfg.Providers))
	rows := make([][]string, 0, len(names))
	defaultRow := -1
	for _, name := range names {
		p := cfg.Providers[name]
		auth := "no"
		if p.HasAuth() {
			auth = "yes"
		}
		label := name
		if name == cfg.CLI.DefaultProvider {
			label = name + " (default)"
			defaultRow = len(rows)
		}
		rows = append(rows, []string{label, p.Model, auth})
	}
	t := newTable().
		Headers("PROVIDER", "MODEL", "AUTH").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleHeader
			}
			if row < 0 || row >= len(rows) {
				return styleCell
			}
			style := styleCell
			switch col {
			case 0:
				if row == defaultRow {
					style = style.Bold(true).Foreground(colorAccent)
				}
			case authCol:
				if rows[row][authCol] == "yes" {
					style = style.Foreground(colorPass)
				} else {
					style = style.Foreground(colorMuted)
				}
			}
			return style
		})
	t.Rows(rows...)
	return renderStyled(out, t.Render())
}
