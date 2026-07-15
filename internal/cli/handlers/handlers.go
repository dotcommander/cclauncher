package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/dotcommander/cclauncher/internal/launcher"
)

const (
	developmentVersion = "dev"
	providerFlag       = "--provider"
	providerShortFlag  = "-p"
)

// GetVersion returns the ldflags-provided version, the installed module
// version, or "dev" for an untagged local build, in that order.
func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	return resolvedVersion(config.Version, info, ok)
}

func resolvedVersion(configured string, info *debug.BuildInfo, ok bool) string {
	if configured != "" && configured != developmentVersion {
		return configured
	}
	if ok && info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return developmentVersion
}

// CodeOptions contains the launcher's process streams.
type CodeOptions struct {
	Input       io.Reader
	Output      io.Writer
	ErrorOutput io.Writer
}

// HandleCode resolves a provider, sets up the environment, and execs Claude
// Code with all non-CCL arguments passed through unchanged.
func HandleCode(ctx context.Context, cfg *config.Config, args []string, opts CodeOptions) error {

	providerName, claudeArgs, err := extractProviderFromArgs(args)
	if err != nil {
		return err
	}

	if providerName == "" && isInteractive(opts.Input) {
		selector := TerminalProviderSelector{In: opts.Input, Out: opts.Output}
		providerName, err = pickProvider(ctx, cfg, selector)
		if err != nil {
			return err
		}
	}

	providerName, provider, err := resolveProvider(providerName, cfg)
	if err != nil {
		return err
	}

	if provider.RequiresAuth() && !provider.HasAuth() {
		return fmt.Errorf(
			"provider %q requires authentication: set %s_API_KEY or configure authToken in %s",
			providerName, strings.ToUpper(providerName), config.GetConfigPath(),
		)
	}

	if _, err := fmt.Fprintf(opts.ErrorOutput, "Launching Claude Code with %s provider using model %s\n",
		providerName, provider.Model); err != nil {
		return fmt.Errorf("write launch status: %w", err)
	}

	if provider.Transformer.HasRules() {
		return launcher.LaunchWithProxy(provider, cfg.Optimization, claudeArgs)
	}

	return launcher.ExecuteClaudeCode(
		launcher.SetupEnvironment(provider, cfg.Optimization),
		claudeArgs,
	)
}

// extractProviderFromArgs finds and removes the CCL provider selector from raw
// args. Supports "--provider value", "--provider=value", and the short form
// "-p value" — but the short form is only claimed when it is the FIRST
// argument pair, because claude itself uses `-p` for print mode and any later
// `-p` must pass through verbatim.
// Returns the provider name (empty if not found), the remaining args, and a
// usage error if a CCL-owned provider selector is missing its value.
func extractProviderFromArgs(args []string) (string, []string, error) {
	result := make([]string, 0, len(args))
	var provider string

	// Short -p is only owned by CCL when it leads the argument list.
	start := 0
	if len(args) > 0 && args[0] == providerShortFlag {
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return "", nil, errors.New("-p requires a provider name")
		}
		provider = args[1]
		start = 2
	}

	for i := start; i < len(args); i++ {
		arg := args[i]

		if arg == providerFlag {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", nil, errors.New("--provider requires a provider name")
			}
			provider = args[i+1]
			i++ // skip the value
			continue
		}

		if v, ok := strings.CutPrefix(arg, providerFlag+"="); ok {
			if v == "" {
				return "", nil, errors.New("--provider requires a provider name")
			}
			provider = v
			continue
		}

		result = append(result, arg)
	}

	return provider, result, nil
}

// resolveProvider validates a provider name and returns its resolved name and config.
// If name is empty, uses the config default.
func resolveProvider(name string, cfg *config.Config) (string, config.Provider, error) {
	if name == "" {
		name = cfg.CLI.DefaultProvider
	}

	provider, ok := cfg.Providers[name]
	if !ok {
		return "", config.Provider{}, unknownProviderError(name)
	}

	// Override model from router.default if it targets this provider.
	// Format: "provider:model" (e.g. "zai:glm-5.1").
	if prov, model, ok := strings.Cut(cfg.Router.Default, ":"); ok && prov == name && model != "" {
		provider.Model = model
		provider.SmallFastModel = model
	}

	return name, provider, nil
}
