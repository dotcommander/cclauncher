package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/dotcommander/cclauncher/internal/config"
)

// ProviderChoice is one selectable provider row.
type ProviderChoice struct {
	Name  string
	Model string
}

// ProviderSelector chooses a provider from the configured provider list.
type ProviderSelector interface {
	SelectProvider(ctx context.Context, choices []ProviderChoice, current string) (string, error)
}

// TerminalProviderSelector selects providers with a simple stdin/stdout prompt.
type TerminalProviderSelector struct {
	In  io.Reader
	Out io.Writer
}

// SelectProvider prompts for one provider name.
func (s TerminalProviderSelector) SelectProvider(ctx context.Context, choices []ProviderChoice, current string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no providers configured")
	}

	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("select provider: %w", err)
	}

	in := s.In
	if in == nil {
		in = os.Stdin
	}
	out := s.Out
	if out == nil {
		out = os.Stdout
	}

	defaultIndex := 0
	for i, choice := range choices {
		if choice.Name == current {
			defaultIndex = i
			break
		}
	}

	w := profileWriter(out)
	numStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	modelStyle := lipgloss.NewStyle().Foreground(colorMuted)
	currentStyle := lipgloss.NewStyle().Foreground(colorPass).Bold(true)
	if _, err := fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Foreground(colorHeader).Render("Select provider:")); err != nil {
		return "", fmt.Errorf("write provider prompt: %w", err)
	}
	for i, choice := range choices {
		pointer := " "
		nameStyle := lipgloss.NewStyle()
		if choice.Name == current {
			pointer = currentStyle.Render("▸")
			nameStyle = nameStyle.Bold(true).Foreground(colorPass)
		}
		line := fmt.Sprintf("%s %s %s", pointer, numStyle.Render(fmt.Sprintf("%d.", i+1)), nameStyle.Render(choice.Name))
		if choice.Model != "" {
			line += " " + modelStyle.Render("("+choice.Model+")")
		}
		if choice.Name == current {
			line += " " + currentStyle.Render("[current]")
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return "", fmt.Errorf("write provider prompt: %w", err)
		}
	}
	if _, err := fmt.Fprintf(w, "%s ", lipgloss.NewStyle().Foreground(colorHeader).Render(fmt.Sprintf("Provider [%d]:", defaultIndex+1))); err != nil {
		return "", fmt.Errorf("write provider prompt: %w", err)
	}

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read provider selection: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("select provider: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return choices[defaultIndex].Name, nil
	}
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(choices) {
		return "", fmt.Errorf("invalid provider selection %q", line)
	}
	return choices[idx-1].Name, nil
}

// isInteractive reports whether in is a terminal (character device). The picker
// only opens for a human at a TTY; piped/CI stdin falls back to the default.
func isInteractive(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// pickProvider opens the interactive picker, pre-selecting the current default
// so Enter relaunches it. The choice is used for this launch only; nothing is
// persisted.
func pickProvider(ctx context.Context, cfg *config.Config, selector ProviderSelector) (string, error) {
	choices := providerChoices(cfg)
	if len(choices) == 0 {
		return "", fmt.Errorf("no providers configured")
	}
	if selector == nil {
		return "", fmt.Errorf("provider selector is not configured")
	}
	name, err := selector.SelectProvider(ctx, choices, cfg.CLI.DefaultProvider)
	if err != nil {
		return "", err
	}
	if _, ok := cfg.Providers[name]; !ok {
		return "", unknownProviderError(name)
	}
	return name, nil
}

// providerChoices returns the configured providers as sorted selectable rows.
func providerChoices(cfg *config.Config) []ProviderChoice {
	names := slices.Sorted(maps.Keys(cfg.Providers))
	choices := make([]ProviderChoice, 0, len(names))
	for _, name := range names {
		choices = append(choices, ProviderChoice{
			Name:  name,
			Model: cfg.Providers[name].Model,
		})
	}
	return choices
}
