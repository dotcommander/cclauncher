package handlers

import (
	"fmt"

	"github.com/dotcommander/cclauncher/internal/config"
)

// unknownProviderError returns a consistently-formatted error for an
// unrecognized provider name, pointing the user at `ccl providers` and the
// config file.
func unknownProviderError(name string) error {
	return fmt.Errorf(
		"unknown provider %q (run 'ccl providers' to list available; config: %s)",
		name, config.GetConfigPath(),
	)
}
