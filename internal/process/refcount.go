package process

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// getDefaultHomeDir returns the default config home directory
func getDefaultHomeDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "ccl")
}

// getDefaultRefCountFile returns the default refcount file path
func getDefaultRefCountFile() string {
	return filepath.Join(getDefaultHomeDir(), ".ccl.refcount")
}

// RefCountManager manages filesystem-based reference counting for concurrent ccl sessions.
// This tracks how many ccl CLI instances are currently running.
type RefCountManager struct {
	homeDir      string
	refCountFile string
	mu           sync.Mutex
}

// NewRefCountManager creates a new reference count manager with default paths
func NewRefCountManager() *RefCountManager {
	return NewRefCountManagerWithPaths(getDefaultHomeDir(), getDefaultRefCountFile())
}

// NewRefCountManagerWithPaths creates a reference count manager with custom paths (for testing)
func NewRefCountManagerWithPaths(homeDir, refCountFile string) *RefCountManager {
	return &RefCountManager{
		homeDir:      homeDir,
		refCountFile: refCountFile,
	}
}

// IncrementRefCount increments the session reference count
// Call this when a CLI command starts
func (m *RefCountManager) IncrementRefCount() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count, err := m.readRefCount()
	if err != nil {
		slog.Warn("Failed to read refcount, assuming 0", "error", err)
		count = 0
	}

	count++

	if err := m.writeRefCount(count); err != nil {
		return fmt.Errorf("failed to write refcount: %w", err)
	}

	slog.Debug("Incremented reference count", "refcount", count)

	return nil
}

// DecrementRefCount decrements the session reference count
// Call this when a CLI command exits
// Returns the new count
func (m *RefCountManager) DecrementRefCount() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count, err := m.readRefCount()
	if err != nil {
		slog.Warn("Failed to read refcount during decrement", "error", err)
		return 0, err
	}

	if count <= 0 {
		slog.Warn("Reference count already at 0")
		return 0, nil
	}

	count--

	if err := m.writeRefCount(count); err != nil {
		return count, fmt.Errorf("failed to write refcount: %w", err)
	}

	slog.Debug("Decremented reference count", "refcount", count)

	return count, nil
}

// GetRefCount returns the current reference count
func (m *RefCountManager) GetRefCount() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.readRefCount()
}

// ResetRefCount resets the reference count to 0
// Use this for manual cleanup of leaked references
func (m *RefCountManager) ResetRefCount() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.writeRefCount(0); err != nil {
		return fmt.Errorf("failed to reset refcount: %w", err)
	}

	slog.Info("Reference count reset to 0")
	return nil
}

// readRefCount reads the reference count from the filesystem
// Must be called with lock held
func (m *RefCountManager) readRefCount() (int, error) {
	data, err := os.ReadFile(m.refCountFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // File doesn't exist means count is 0
		}
		return 0, fmt.Errorf("failed to read refcount file: %w", err)
	}

	count, err := strconv.Atoi(string(data))
	if err != nil {
		slog.Warn("Invalid refcount file content, resetting to 0",
			"error", err,
			"content", string(data))
		return 0, nil
	}

	return count, nil
}

// writeRefCount writes the reference count to the filesystem
// Must be called with lock held
func (m *RefCountManager) writeRefCount(count int) error {
	// Ensure directory exists
	if err := os.MkdirAll(m.homeDir, 0700); err != nil {
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	// Write count to file
	data := strconv.Itoa(count)
	if err := os.WriteFile(m.refCountFile, []byte(data), 0600); err != nil {
		return fmt.Errorf("failed to write refcount file: %w", err)
	}

	return nil
}

// GetRefCountFile returns the path to the refcount file
func GetRefCountFile() string {
	return getDefaultRefCountFile()
}

// RefCountFile returns the path to the refcount file for this manager instance
func (m *RefCountManager) RefCountFile() string {
	return m.refCountFile
}
