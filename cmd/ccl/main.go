package main

import (
	"fmt"
	"log/slog"
	"os"

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

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
