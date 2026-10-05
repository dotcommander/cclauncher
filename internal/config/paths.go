package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func defaultHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "cclauncher"), nil
}
func getDefaultHomeDir() string { home, _ := defaultHomeDir(); return home }
func getDefaultConfigFile() string {
	home, err := defaultHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "config.yaml")
}
func GetConfigPath() string { return getDefaultConfigFile() }
