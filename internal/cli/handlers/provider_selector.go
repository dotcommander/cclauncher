package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
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
	for i, choice := range choices {
		label := fmt.Sprintf("%d. %s", i+1, choice.Name)
		if choice.Model != "" {
			label += " (" + choice.Model + ")"
		}
		if choice.Name == current {
			selected = current
			label += " [current]"
		}
		options = append(options, huh.NewOption(label, choice.Name))
	}
	field := huh.NewSelect[string]().Title("Select provider").Options(options...).Value(&selected)
	numbers := providerNumberKeys{field: field, options: options, selected: &selected}
	// WithProgramOptions replaces existing options, so set IO after the filter.
	form := huh.NewForm(huh.NewGroup(field)).
		WithProgramOptions(tea.WithFilter(numbers.filter)).
		WithInput(in).WithOutput(out).WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	if err := form.RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("select provider: %w", err)
	}
	return selected, nil
}

// isInteractive reports whether the supplied input is a real terminal.
func isInteractive(in io.Reader) bool {
	f, ok := in.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

func isTerminalOutput(out io.Writer) bool {
	f, ok := out.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

// pickProvider opens the interactive picker, pre-selecting the current default
// so Enter relaunches it. A successful session remembers the selected pair.
func pickProvider(ctx context.Context, cfg *config.Config, selector ProviderSelector) (string, error) {
	client := localProviderClient()
	defer client.CloseIdleConnections()
	choices, err := reachableProviderChoices(ctx, cfg, client)
	if err != nil {
		return "", fmt.Errorf("check provider availability: %w", err)
	}
	if len(choices) == 0 {
		return "", errors.New("no launchable providers: check credentials, configuration, and local servers")
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
	if !slices.ContainsFunc(choices, func(choice ProviderChoice) bool { return choice.Name == name }) {
		return "", fmt.Errorf("provider %q is not launchable", name)
	}
	return name, nil
}

// providerChoices returns sorted rows that pass offline launch prerequisites.
func providerChoices(cfg *config.Config) []ProviderChoice {
	names := slices.Sorted(maps.Keys(cfg.Providers))
	choices := make([]ProviderChoice, 0, len(names))
	for _, name := range names {
		_, provider, _ := resolveProvider(name, cfg)
		if provider.RequiresAuth() && !provider.HasAuth() {
			continue
		}
		if err := config.ValidateProvider(provider); err != nil {
			continue
		}
		choices = append(choices, ProviderChoice{
			Name:  name,
			Model: provider.Model,
		})
	}
	return choices
}

func shouldPickProvider(args []string, opts CodeOptions) bool {
	return len(args) == 0 && isInteractive(opts.Input) && isTerminalOutput(opts.Output)
}
