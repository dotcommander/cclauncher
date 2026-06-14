package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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

	if _, err := fmt.Fprintln(out, "Select provider:"); err != nil {
		return "", fmt.Errorf("write provider prompt: %w", err)
	}
	for i, choice := range choices {
		label := choice.Name
		if choice.Model != "" {
			label = fmt.Sprintf("%s (%s)", choice.Name, choice.Model)
		}
		if choice.Name == current {
			label = fmt.Sprintf("%s [current]", label)
		}
		if _, err := fmt.Fprintf(out, "  %d. %s\n", i+1, label); err != nil {
			return "", fmt.Errorf("write provider prompt: %w", err)
		}
	}
	if _, err := fmt.Fprintf(out, "Provider [%d]: ", defaultIndex+1); err != nil {
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
