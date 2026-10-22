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

// HandleCode resolves a provider, supervises Claude Code, and remembers the
// provider/model pair only after a successful session.
func HandleCode(ctx context.Context, cfg *config.Config, args []string, opts CodeOptions) error {
	return handleCode(ctx, cfg, args, opts, launchDependencies{
		run: func(ctx context.Context, p config.Provider, opt config.OptimizationConfig, args []string, opts CodeOptions) error {
			return launcher.LaunchIO(ctx, p, opt, args, opts.Input, opts.Output, opts.ErrorOutput)
		},
		remember: config.SetDefaultSelection,
	})
}

func handleCode(ctx context.Context, cfg *config.Config, args []string, opts CodeOptions, deps launchDependencies) error {

	providerName, claudeArgs, err := extractProviderFromArgs(args)
	if err != nil {
		return err
	}

	if shouldPickProvider(args, opts) {
		selector := TerminalProviderSelector{In: opts.Input, Out: opts.Output}
		providerName, err = pickProvider(ctx, cfg, selector)
		if err != nil {
			return err
		}
	}

	providerName, provider, err := resolveProviderForModel(providerName, cfg, commandLineModel(claudeArgs))
	if err != nil {
		return err
	}

	provider = selectedModel(provider, claudeArgs)
	provider.Transformer, err = config.ResolveTransformerPresets(provider.Transformer)
	if err != nil {
		return fmt.Errorf("provider %q: %w", providerName, err)
	}
	if err := config.ValidateProvider(provider); err != nil {
		return fmt.Errorf("provider %q: %w", providerName, err)
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

	if err := deps.run(ctx, provider, cfg.Optimization, claudeArgs, opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if remembersSession(claudeArgs) {
		if err := deps.remember(providerName, provider); err != nil {
			return fmt.Errorf("session succeeded but saving default selection failed: %w", err)
		}
	}
	return nil
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

		if arg == "--" {
			result = append(result, args[i+1:]...)
			break
		}

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
	return resolveProviderForModel(name, cfg, "")
}

func resolveProviderForModel(name string, cfg *config.Config, model string) (string, config.Provider, error) {
	if name == "" {
		name = cfg.CLI.DefaultProvider
	}

	provider, ok := cfg.Providers[name]
	if !ok {
		return "", config.Provider{}, unknownProviderError(name)
	}

	// Explicit primary aliases restore configured slots instead of remembered ones.
	primaryAlias := model == "default" || model == "opus" || model == "sonnet"
	if !primaryAlias && name == cfg.CLI.DefaultProvider && cfg.CLI.DefaultModel != "" {
		provider.Model = cfg.CLI.DefaultModel
		provider.SmallFastModel = cfg.CLI.DefaultSmallFastModel
	}

	// Override model from router.default if it targets this provider.
	// Format: "provider:model" (e.g. "zai:glm-5.1").
	if prov, model, ok := strings.Cut(cfg.Router.Default, ":"); ok && prov == name && model != "" {
		provider.Model = model
		provider.SmallFastModel = model
	}

	return name, provider, nil
}
