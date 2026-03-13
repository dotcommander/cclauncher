package config

import (
	"os"
	"path/filepath"
)

// getDefaultHomeDir returns the default config home directory
func getDefaultHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "cclauncher")
}

// getDefaultConfigFile returns the default config file path
func getDefaultConfigFile() string {
	return filepath.Join(getDefaultHomeDir(), "config.yaml")
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return getDefaultConfigFile()
}
