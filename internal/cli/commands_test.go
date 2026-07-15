package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnedCommandSurface(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"providers", "doctor", "version", "update"} {
		if !isOwnedCommand(name) {
			t.Fatalf("%s is not owned", name)
		}
	}
	for _, arg := range []string{"--model", "-p", "hello"} {
		if isOwnedCommand(arg) {
			t.Fatalf("%s unexpectedly owned", arg)
		}
	}
}

func TestCompletionCommandIsDisabled(t *testing.T) {
	t.Parallel()
	err := run(context.Background(), []string{"completion"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || err.Error() != `unknown command "completion" for "ccl"` {
		t.Fatalf("error = %v", err)
	}
}

func TestVersionSkipsConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var out bytes.Buffer
	if err := run(context.Background(), []string{"version"}, strings.NewReader(""), &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ccl version") {
		t.Fatalf("output = %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "cclauncher", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("config touched: %v", err)
	}
}

func TestHelpSkipsConfigAndStaysLocal(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root flag", args: []string{"--help"}, want: "Usage: ccl"},
		{name: "help command", args: []string{"help"}, want: "Usage: ccl"},
		{name: "owned help command", args: []string{"help", "doctor"}, want: "Usage: ccl doctor"},
		{name: "owned help flag", args: []string{"update", "--help"}, want: "Usage: ccl update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			var out bytes.Buffer
			if err := run(context.Background(), tt.args, strings.NewReader(""), &out, &bytes.Buffer{}); err != nil {
				t.Fatalf("run(%v): %v", tt.args, err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
			if _, err := os.Stat(filepath.Join(home, ".config", "cclauncher", "config.yaml")); !os.IsNotExist(err) {
				t.Fatalf("help touched config: %v", err)
			}
		})
	}
}

func TestHelpRejectsUnknownOwnedCommand(t *testing.T) {
	t.Parallel()
	err := run(context.Background(), []string{"help", "missing"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), `unknown ccl command "missing"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestDoctorFlagsParse(t *testing.T) {
	t.Parallel()
	var tree commandTree
	p, err := newParser(&tree, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Parse([]string{"doctor", "--provider", "x", "--json", "--check-net"}); err != nil {
		t.Fatal(err)
	}
	if tree.Doctor.Provider != "x" || !tree.Doctor.JSON || !tree.Doctor.CheckNet {
		t.Fatalf("doctor = %#v", tree.Doctor)
	}
}
