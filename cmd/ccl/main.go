package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/dotcommander/cclauncher/internal/cli"
)

func main() {
	// Configure logger with appropriate level
	// Use Info level for console output to reduce noise
	logLevel := slog.LevelInfo

	// If DEBUG env var is set, enable debug logging
	if os.Getenv("CCL_DEBUG") != "" {
		logLevel = slog.LevelDebug
	}

	// Create a text handler that writes to stderr
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	if err := cli.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	var child *exec.ExitError
	if errors.As(err, &child) {
		if status, ok := child.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
		return child.ExitCode()
	}
	return 1
}
