package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/alecthomas/kong"
	"github.com/dotcommander/cclauncher/internal/cli/handlers"
	"github.com/dotcommander/cclauncher/internal/config"
)

const (
	commandDoctor    = "doctor"
	commandHelp      = "help"
	commandProviders = "providers"
	commandUpdate    = "update"
	commandVersion   = "version"
	longHelpFlag     = "--help"
)

type commandTree struct {
	Providers providersCommand `cmd:"" help:"List configured providers."`
	Doctor    doctorCommand    `cmd:"" help:"Run preflight diagnostics on configured providers."`
	Version   versionCommand   `cmd:"" help:"Show version information."`
	Update    updateCommand    `cmd:"" help:"Update CCL to the latest version."`
}

type providersCommand struct{}
type versionCommand struct{}
type doctorCommand struct {
	Provider string `help:"Check only this provider."`
	JSON     bool   `help:"Emit results as JSON."`
	CheckNet bool   `name:"check-net" help:"Probe provider reachability over the network."`
}

func (doctorCommand) Help() string {
	return "Run local provider checks for authentication, required fields, and base URL problems. " +
		"Use --check-net to also probe provider reachability. The command fails only when a check reports FAIL."
}

type updateCommand struct {
	Check bool `help:"Check for updates without installing."`
}

func (updateCommand) Help() string {
	return "Check GitHub for the latest CCL release and install it with go install. " +
		"Use --check to report availability without changing the installed binary."
}

// Execute routes CCL-owned subcommands through Kong and forwards every other
// argument verbatim to Claude Code.
func Execute(ctx context.Context) error {
	return run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

func run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	helpTarget, helpRequested, err := requestedHelp(args)
	if err != nil {
		return err
	}
	if helpRequested {
		return writeCommandHelp(helpTarget, out, errOut)
	}

	if len(args) > 0 && args[0] == "completion" {
		return fmt.Errorf("unknown command %q for %q", "completion", "ccl")
	}
	if len(args) == 0 || !isOwnedCommand(args[0]) {
		cfg, loadErr := loadConfig()
		if loadErr != nil {
			return loadErr
		}
		return handlers.HandleCode(ctx, cfg, args, handlers.CodeOptions{
			Input:       in,
			Output:      out,
			ErrorOutput: errOut,
		})
	}

	return runOwnedCommand(ctx, args, out, errOut)
}

func runOwnedCommand(ctx context.Context, args []string, out, errOut io.Writer) error {
	var tree commandTree
	parser, err := newParser(&tree, out, errOut)
	if err != nil {
		return err
	}
	parsed, err := parser.Parse(args)
	if err != nil {
		return err
	}

	switch parsed.Command() {
	case commandVersion:
		_, err = fmt.Fprintln(out, "ccl version "+handlers.GetVersion())
		return err
	case commandUpdate:
		return handlers.HandleUpdate(ctx, out, errOut, tree.Update.Check)
	case commandProviders:
		cfg, loadErr := loadConfig()
		if loadErr != nil {
			return loadErr
		}
		return handlers.HandleProviders(out, cfg)
	case commandDoctor:
		cfg, loadErr := loadConfig()
		if loadErr != nil {
			return loadErr
		}
		return handlers.HandleDoctor(ctx, out, cfg, handlers.DoctorOptions{
			Provider: tree.Doctor.Provider,
			JSON:     tree.Doctor.JSON,
			CheckNet: tree.Doctor.CheckNet,
		})
	default:
		return fmt.Errorf("unknown command %q", parsed.Command())
	}
}

func isOwnedCommand(arg string) bool {
	switch arg {
	case commandProviders, commandDoctor, commandVersion, commandUpdate:
		return true
	default:
		return false
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Init()
	if err != nil {
		return nil, fmt.Errorf("initialize config: %w", err)
	}
	return cfg, nil
}

func newParser(tree *commandTree, out, errOut io.Writer) (*kong.Kong, error) {
	return kong.New(tree,
		kong.Name("ccl"),
		kong.Description("Launch Claude Code with different LLM providers."),
		kong.Writers(out, errOut),
		kong.NoDefaultHelp(),
		kong.Exit(func(int) {}),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:   true,
			Tree:      true,
			Summary:   true,
			FlagsLast: true,
		}),
	)
}

func requestedHelp(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, nil
	}
	if args[0] == commandHelp {
		if len(args) == 1 {
			return "", true, nil
		}
		if len(args) > 2 {
			return "", false, errors.New("help accepts at most one command")
		}
		if !isOwnedCommand(args[1]) {
			return "", false, fmt.Errorf("unknown ccl command %q", args[1])
		}
		return args[1], true, nil
	}
	if !slices.ContainsFunc(args, isHelpFlag) {
		return "", false, nil
	}
	if isOwnedCommand(args[0]) {
		return args[0], true, nil
	}
	return "", true, nil
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == longHelpFlag
}

func writeCommandHelp(command string, out, errOut io.Writer) error {
	if command == "" {
		return writeRootHelp(out)
	}
	var tree commandTree
	parser, err := newParser(&tree, out, errOut)
	if err != nil {
		return fmt.Errorf("create command help: %w", err)
	}
	parsed, err := parser.Parse([]string{command})
	if err != nil {
		return fmt.Errorf("parse command help: %w", err)
	}
	if err := parsed.PrintUsage(false); err != nil {
		return fmt.Errorf("write command help: %w", err)
	}
	return nil
}

func writeRootHelp(out io.Writer) error {
	_, err := fmt.Fprint(out, `Usage: ccl [flags passed to Claude Code]
       ccl <command> [flags]

Launch Claude Code with different LLM providers.

All flags except --provider are passed through to Claude Code.
Use --provider to select an LLM provider, or run bare ccl to pick interactively.

Commands:
  providers  List configured providers
  doctor     Run preflight diagnostics
  version    Show version information
  update     Update CCL

Run ccl help <command> for command-specific help.
`)
	return err
}
