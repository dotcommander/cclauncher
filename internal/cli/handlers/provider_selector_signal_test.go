//go:build !windows

package handlers

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/xpty"
)

func TestTerminalProviderSelectorSignalAbort(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("signal subprocess")
	}
	pty, err := xpty.NewUnixPty(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pty.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalProviderSelectorSignalHelper$")
	cmd.Env = append(os.Environ(), "CCL_PICKER_SIGNAL_FIXTURE=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pty.Slave(), pty.Slave(), pty.Slave()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Only the child retains the slave, so its exit also wakes the reader.
	_ = pty.Slave().Close()
	var screen []byte
	chunk := make([]byte, 1024)
	sent := false
	var signalErr error
	for len(screen) < 64<<10 {
		n, readErr := pty.Read(chunk)
		screen = append(screen, chunk[:n]...)
		if bytes.Contains(screen, []byte("Select provider")) {
			sent = true
			signalErr = cmd.Process.Signal(syscall.SIGTERM)
			break
		}
		if readErr != nil {
			break
		}
	}
	if !sent || signalErr != nil {
		_ = cmd.Process.Kill()
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("signal fixture failed: %v\n%s", err, screen)
	}
	if !sent || signalErr != nil {
		t.Fatalf("signal sent=%v error=%v", sent, signalErr)
	}
}

func TestTerminalProviderSelectorSignalHelper(t *testing.T) {
	t.Parallel()
	if os.Getenv("CCL_PICKER_SIGNAL_FIXTURE") != "1" {
		t.Skip("subprocess fixture")
	}
	selector := TerminalProviderSelector{In: os.Stdin, Out: os.Stdout}
	selected, err := selector.SelectProvider(context.Background(), []ProviderChoice{{Name: "first"}, {Name: "second"}}, "first")
	if selected != "" || !errors.Is(err, huh.ErrUserAborted) {
		t.Fatalf("unexpected selection %q: %v", selected, err)
	}
}
