package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	"charm.land/huh/v2"
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
		return "", errors.New("no providers configured")
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

	selected := choices[0].Name
	options := make([]huh.Option[string], 0, len(choices))
	for _, choice := range choices {
		label := choice.Name
		if choice.Model != "" {
			label += " (" + choice.Model + ")"
		}
		if choice.Name == current {
			selected = current
			label += " [current]"
		}
		options = append(options, huh.NewOption(label, choice.Name))
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Select provider").Options(options...).Value(&selected),
	)).WithInput(in).WithOutput(out).WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	if err := form.RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("select provider: %w", err)
	}
	return selected, nil
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
		return "", errors.New("no providers configured")
	}
	if selector == nil {
		return "", errors.New("provider selector is not configured")
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
