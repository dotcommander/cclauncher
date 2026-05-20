package handlers

import (
	"slices"
	"testing"
)

func TestExtractProviderFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantProv   string
		wantClaude []string
	}{
		{
			name:       "no provider flag",
			args:       []string{"--continue", "-p", "hello"},
			wantProv:   "",
			wantClaude: []string{"--continue", "-p", "hello"},
		},
		{
			name:       "provider with space",
			args:       []string{"--provider", "deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "provider with equals",
			args:       []string{"--provider=deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "provider at end",
			args:       []string{"--continue", "--provider", "deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{"--continue"},
		},
		{
			name:       "provider equals at end",
			args:       []string{"--continue", "--provider=deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{"--continue"},
		},
		{
			name:       "provider flag without value at end",
			args:       []string{"--provider"},
			wantProv:   "",
			wantClaude: []string{"--provider"},
		},
		{
			name:       "empty args",
			args:       []string{},
			wantProv:   "",
			wantClaude: []string{},
		},
		{
			name:       "positional prompt only",
			args:       []string{"hello world"},
			wantProv:   "",
			wantClaude: []string{"hello world"},
		},
		{
			name:       "dangerously-skip-permissions passes through",
			args:       []string{"--dangerously-skip-permissions", "hello"},
			wantProv:   "",
			wantClaude: []string{"--dangerously-skip-permissions", "hello"},
		},
		{
			name:       "provider equals empty value",
			args:       []string{"--provider="},
			wantProv:   "",
			wantClaude: []string{},
		},
		{
			name:       "leading short -p claims provider, later -p passes through",
			args:       []string{"-p", "deepseek", "-p", "hello"},
			wantProv:   "deepseek",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "leading short -p claims provider alone",
			args:       []string{"-p", "deepseek"},
			wantProv:   "deepseek",
			wantClaude: []string{},
		},
		{
			name:       "non-leading -p passes through to claude",
			args:       []string{"--continue", "-p", "hello"},
			wantProv:   "",
			wantClaude: []string{"--continue", "-p", "hello"},
		},
		{
			name:       "leading short -p with later --provider override",
			args:       []string{"-p", "deepseek", "--provider", "zai", "-p", "hello"},
			wantProv:   "zai",
			wantClaude: []string{"-p", "hello"},
		},
		{
			name:       "lone -p at start without value passes through",
			args:       []string{"-p"},
			wantProv:   "",
			wantClaude: []string{"-p"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProv, gotClaude := extractProviderFromArgs(tt.args)
			if gotProv != tt.wantProv {
				t.Errorf("provider = %q, want %q", gotProv, tt.wantProv)
			}
			if !slices.Equal(gotClaude, tt.wantClaude) {
				t.Errorf("claudeArgs = %v, want %v", gotClaude, tt.wantClaude)
			}
		})
	}
}
