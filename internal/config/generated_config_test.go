package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedConfigExamplesMatchEmbeddedDefault(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join("..", "..", "examples", "config.yaml.example"),
		filepath.Join("testdata", "config.yaml"),
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			generated, err := os.ReadFile(path)
			require.NoError(t, err)

			require.Equal(t, string(defaultConfigYAML), string(generated),
				"%s must be regenerated from internal/config/default-config.yaml", path)
		})
	}
}
