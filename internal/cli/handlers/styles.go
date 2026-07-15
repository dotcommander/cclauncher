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
const (
	colorPass   = "42"  // green
	colorWarn   = "214" // amber
	colorFail   = "196" // red
	colorHeader = "99"  // purple help accent
	colorMuted  = "240" // dim border / secondary text
	colorAccent = "212" // pink accent
)

func headerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorHeader)).Padding(0, 1)
}

func cellStyle() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1)
}

func borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
}

// newTable returns a table preconfigured with ccl's border and border style.
func newTable() *table.Table {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle())
}

// profileWriter wraps out so emitted ANSI matches the real stdout's color
// profile (plain when stdout is not a TTY: pipes, CI, captured test buffers).
func profileWriter(out io.Writer) *colorprofile.Writer {
	w := colorprofile.NewWriter(out, os.Environ())
	w.Profile = colorprofile.Detect(os.Stdout, os.Environ())
	return w
}

// renderStyled writes s to out followed by a newline, via profileWriter.
func renderStyled(out io.Writer, s string) error {
	_, err := fmt.Fprintln(profileWriter(out), s)
	return err
}
