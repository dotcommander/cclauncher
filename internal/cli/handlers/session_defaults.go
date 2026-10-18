package handlers

import (
	"context"
	"strings"

	"github.com/dotcommander/cclauncher/internal/config"
)

type launchDependencies struct {
	run      func(context.Context, config.Provider, config.OptimizationConfig, []string, CodeOptions) error
	remember func(string, config.Provider) error
}

// selectedModel records command-line model overrides rather than the configured
// model they supersede. CCL maps family aliases through the provider's slots.
func selectedModel(p config.Provider, args []string) config.Provider {
	var model string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		if args[i] == "--model" && i+1 < len(args) {
			i++
			model = args[i]
		} else if value, ok := strings.CutPrefix(args[i], "--model="); ok {
			model = value
		}
	}
	switch model {
	case "", "default", "opus", "sonnet":
		return p
	case "haiku":
		if p.SmallFastModel != "" {
			p.Model = p.SmallFastModel
		}
	default:
		p.Model = model
	}
	return p
}

// Metadata and administrative invocations do not demonstrate a used model.
func remembersSession(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "--version", "-v":
			return false
		}
	}
	if len(args) > 0 {
		switch args[0] {
		case "agents", "attach", "auth", "auto-mode", "doctor", "gateway", "help", "import", "install", "logs", "mcp", "plugin", "plugins", "purge", "respawn", "rm", "setup-token", "stop", "kill", "ultrareview", "update", "upgrade":
			return false
		}
	}
	return true
}
