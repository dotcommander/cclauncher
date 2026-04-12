package handlers

import (
	"errors"
	"fmt"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// errConfigNotLoaded indicates the root PersistentPreRunE did not store the
// config in the command context. It is an internal bug, never a user error.
var errConfigNotLoaded = errors.New("config not loaded (internal error)")

// configFromCmd extracts the loaded config from the command context.
// Returns errConfigNotLoaded if the root pre-runner did not attach it.
func configFromCmd(cmd *cobra.Command) (*config.Config, error) {
	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		return nil, errConfigNotLoaded
	}
	return cfg, nil
}

// unknownProviderError returns a consistently-formatted error for an
// unrecognized provider name, pointing the user at `ccl providers` and the
// config file.
func unknownProviderError(name string) error {
	return fmt.Errorf(
		"unknown provider %q (run 'ccl providers' to list available; config: %s)",
		name, config.GetConfigPath(),
	)
}
