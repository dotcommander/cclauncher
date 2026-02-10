# Default recipe - show menu
default:
    @just --list

# Variables
BINARY_NAME := "ccl"
GO := "go"
VERSION := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
GOFLAGS := "-ldflags='-s -w -X github.com/dotcommander/cclauncher/internal/config.Version=" + VERSION + "'"
GOPATH := env_var_or_default("GOPATH", env_var("HOME") + "/go")

# Build the binary
build:
    {{GO}} build {{GOFLAGS}} -o {{BINARY_NAME}} ./cmd/ccl

# Run tests
test:
    {{GO}} test -v ./...

# Clean build artifacts
clean:
    {{GO}} clean
    rm -f {{BINARY_NAME}}
    rm -f test-ccl

# Install binary to GOPATH/bin
install: build
    ln -sf {{justfile_directory()}}/{{BINARY_NAME}} {{GOPATH}}/bin/{{BINARY_NAME}}

# Format code
fmt:
    {{GO}} fmt ./...

# Run linter
lint:
    golangci-lint run

# Build and run
run: build
    ./{{BINARY_NAME}} start

# Run directly without building
dev:
    {{GO}} run cmd/ccl/main.go
