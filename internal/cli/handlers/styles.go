package handlers

import (
	"fmt"
	"io"
	"os"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
)

// Shared lipgloss palette for ccl's CLI tables and prompts. Output is written
// through a colorprofile.Writer whose profile is detected from the real stdout,
// so ANSI is emitted in a terminal but stripped when piped, in CI, or captured
// by a test buffer.
var (
	colorPass   = lipgloss.Color("42")  // green
	colorWarn   = lipgloss.Color("214") // amber
	colorFail   = lipgloss.Color("196") // red
	colorHeader = lipgloss.Color("99")  // purple, matches fang's help
	colorMuted  = lipgloss.Color("240") // dim border / secondary text
	colorAccent = lipgloss.Color("212") // pink accent

	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(colorHeader).Padding(0, 1)
	styleCell   = lipgloss.NewStyle().Padding(0, 1)
	styleBorder = lipgloss.NewStyle().Foreground(colorMuted)
)

// newTable returns a table preconfigured with ccl's border and border style.
func newTable() *table.Table {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(styleBorder)
}

// renderStyled writes s to out followed by a newline, downsampling/stripping
// ANSI to match the real stdout's color profile (plain when stdout is not a TTY).
func renderStyled(out io.Writer, s string) error {
	w := colorprofile.NewWriter(out, os.Environ())
	w.Profile = colorprofile.Detect(os.Stdout, os.Environ())
	_, err := fmt.Fprintln(w, s)
	return err
}
