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

func TestChildExitCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		script string
		want   int
	}{{"exit 7", 7}, {"kill -TERM $$", 143}} {
		t.Run(tc.script, func(t *testing.T) {
			t.Parallel()
			err := exec.Command("sh", "-c", tc.script).Run()
			if got := exitCode(err); got != tc.want {
				t.Fatalf("code=%d want=%d err=%v", got, tc.want, err)
			}
		})
	}
}
