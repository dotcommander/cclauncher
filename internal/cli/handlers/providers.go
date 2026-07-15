package handlers

import (
	"io"
	"maps"
	"slices"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/dotcommander/cclauncher/internal/config"
)

// HandleProviders prints the configured providers as a table, marking the
// default and showing whether each has auth credentials available.
func HandleProviders(out io.Writer, cfg *config.Config) error {
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
				return headerStyle()
			}
			if row < 0 || row >= len(rows) {
				return cellStyle()
			}
			style := cellStyle()
			switch col {
			case 0:
				if row == defaultRow {
					style = style.Bold(true).Foreground(lipgloss.Color(colorAccent))
				}
			case authCol:
				if rows[row][authCol] == "yes" {
					style = style.Foreground(lipgloss.Color(colorPass))
				} else {
					style = style.Foreground(lipgloss.Color(colorMuted))
				}
			}
			return style
		})
	t.Rows(rows...)
	return renderStyled(out, t.Render())
}
