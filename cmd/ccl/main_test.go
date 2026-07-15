package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLICommands(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "test-ccl", "main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-ccl")

	// Test version command
	t.Run("version", func(t *testing.T) {
		cmd := exec.Command("./test-ccl", "version")
		output, err := cmd.Output()
		if err != nil {
			t.Errorf("version command failed: %v", err)
		}
		if !strings.Contains(string(output), "ccl version dev") {
			t.Errorf("unexpected version output: %s", output)
		}
	})

}
