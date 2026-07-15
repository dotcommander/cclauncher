package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/dotcommander/cclauncher/internal/actions"
	"github.com/dotcommander/cclauncher/internal/config"
)

// ExitCodeError carries an intended process exit code alongside a message.
// cmd/ccl/main.go exits 1 on any non-nil Execute() error; Code is reserved for
// future per-code handling.
type ExitCodeError struct {
	Code int
	Msg  string
}

func (e *ExitCodeError) Error() string { return e.Msg }

// DoctorOptions controls provider selection, output format, and network checks.
type DoctorOptions struct {
	Provider string
	JSON     bool
	CheckNet bool
}

// HandleDoctor runs preflight diagnostics for the selected provider(s).
func HandleDoctor(ctx context.Context, out io.Writer, cfg *config.Config, opts DoctorOptions) error {
	names, err := selectProviders(opts.Provider, cfg)
	if err != nil {
		return err
	}

	var client *http.Client
	if opts.CheckNet {
		client = &http.Client{Timeout: actions.DoctorNetTimeout}
	}

	var results []actions.CheckResult
	for _, name := range names {
		p := cfg.Providers[name]
		results = append(results, actions.RunChecks(name, p)...)
		if opts.CheckNet {
			results = append(results, actions.ProbeReachability(ctx, client, name, p))
		}
	}

	if opts.JSON {
		if err := json.NewEncoder(out).Encode(results); err != nil {
			return fmt.Errorf("write doctor JSON: %w", err)
		}
	} else {
		if err := writeDoctorTable(out, results); err != nil {
			return fmt.Errorf("write doctor table: %w", err)
		}
	}
	if anyFail(results) {
		return &ExitCodeError{Code: 1, Msg: "one or more providers failed preflight checks"}
	}
	return nil
}

// selectProviders returns the provider name(s) to check: a single validated name
// when flag is non-empty, otherwise all providers in sorted order.
func selectProviders(flag string, cfg *config.Config) ([]string, error) {
	if flag != "" {
		name, _, err := resolveProvider(flag, cfg)
		if err != nil {
			return nil, err
		}
		return []string{name}, nil
	}
	return slices.Sorted(maps.Keys(cfg.Providers)), nil
}

func writeDoctorTable(out io.Writer, results []actions.CheckResult) error {
	const statusCol = 2
	t := newTable().
		Headers("PROVIDER", "CHECK", "STATUS", "REASON").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle()
			}
			style := cellStyle()
			if col == statusCol && row >= 0 && row < len(results) {
				switch results[row].Status {
				case actions.StatusPass:
					style = style.Foreground(lipgloss.Color(colorPass))
				case actions.StatusWarn:
					style = style.Foreground(lipgloss.Color(colorWarn))
				case actions.StatusFail:
					style = style.Foreground(lipgloss.Color(colorFail)).Bold(true)
				}
			}
			return style
		})
	for _, r := range results {
		t.Row(r.Provider, r.Check, string(r.Status), r.Reason)
	}
	return renderStyled(out, t.Render())
}

func anyFail(results []actions.CheckResult) bool {
	for _, r := range results {
		if r.Status == actions.StatusFail {
			return true
		}
	}
	return false
}
