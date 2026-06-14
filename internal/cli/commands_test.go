package cli

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

func TestNewRootCmd_CommandSurface(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	if cmd.Use != "ccl" {
		t.Fatalf("Use = %q, want ccl", cmd.Use)
	}
	if !cmd.DisableFlagParsing {
		t.Fatal("root command must keep DisableFlagParsing so Claude flags pass through")
	}
	if !cmd.SilenceErrors {
		t.Fatal("root command should silence Cobra errors so main controls error output")
	}
	if cmd.PersistentPreRunE == nil {
		t.Fatal("root command must preload config before handler execution")
	}
	if cmd.RunE == nil {
		t.Fatal("root command must dispatch to the code launcher handler")
	}

	got := commandUses(cmd.Commands())
	want := []string{
		"doctor",
		"providers",
		"update",
		"version",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("subcommand uses = %v, want %v", got, want)
	}
}

func TestSubcommandFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	doctor := findCommand(t, cmd, "doctor")
	for _, name := range []string{"provider", "json", "check-net"} {
		if doctor.Flags().Lookup(name) == nil {
			t.Fatalf("doctor missing --%s flag", name)
		}
	}
	if !doctor.SilenceUsage {
		t.Fatal("doctor should silence usage on diagnostic failures")
	}

	update := findCommand(t, cmd, "update")
	if update.Flags().Lookup("check") == nil {
		t.Fatal("update missing --check flag")
	}
}

func TestLoadConfigIntoContext_SkipsConfigIndependentCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	for _, name := range []string{"version", "update"} {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: name}
			cmd.SetContext(context.Background())

			if err := loadConfigIntoContext(cmd, nil); err != nil {
				t.Fatalf("loadConfigIntoContext(%s) returned error: %v", name, err)
			}
			if cfg := config.FromContext(cmd.Context()); cfg != nil {
				t.Fatalf("%s stored config in context, want nil", name)
			}
			if _, err := os.Stat(filepath.Join(home, ".config", "cclauncher", "config.yaml")); !os.IsNotExist(err) {
				t.Fatalf("%s touched config file, stat err = %v", name, err)
			}
		})
	}
}

func TestLoadConfigIntoContext_InitializesConfigForRuntimeCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := &cobra.Command{Use: "providers"}
	cmd.SetContext(context.Background())
	if err := loadConfigIntoContext(cmd, nil); err != nil {
		t.Fatalf("loadConfigIntoContext returned error: %v", err)
	}

	if cfg := config.FromContext(cmd.Context()); cfg == nil {
		t.Fatal("config was not stored in command context")
	} else if len(cfg.Providers) == 0 {
		t.Fatal("loaded config has no providers")
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "cclauncher", "config.yaml")); err != nil {
		t.Fatalf("expected default config to be created: %v", err)
	}
}

func commandUses(commands []*cobra.Command) []string {
	uses := make([]string, 0, len(commands))
	for _, cmd := range commands {
		uses = append(uses, cmd.Use)
	}
	slices.Sort(uses)
	return uses
}

func findCommand(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("command %q not found", name)
	return nil
}
