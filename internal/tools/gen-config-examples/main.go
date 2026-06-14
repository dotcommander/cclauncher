package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

var generatedConfigTargets = []string{
	filepath.Join("examples", "config.yaml.example"),
	filepath.Join("internal", "config", "testdata", "config.yaml"),
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "generate config examples: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}

	sourcePath := filepath.Join(root, "internal", "config", "default-config.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	for _, target := range generatedConfigTargets {
		path := filepath.Join(root, target)
		if err := writeIfChanged(path, source); err != nil {
			return err
		}
		fmt.Printf("generated %s\n", target)
	}

	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	for {
		if fileExists(filepath.Join(dir, "go.mod")) && fileExists(filepath.Join(dir, "internal", "config", "default-config.yaml")) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root not found")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func writeIfChanged(path string, data []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read target %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create target dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write target %s: %w", path, err)
	}
	return nil
}
