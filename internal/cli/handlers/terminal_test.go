//go:build !windows

package handlers

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/xpty"
	"github.com/stretchr/testify/require"
)

func TestTerminalPickerNumberInput(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("terminal integration")
	}
	pty, err := xpty.NewUnixPty(80, 24)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pty.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	inputSent := make(chan error, 1)
	// Reader exits after sending the selection or when closing the PTY wakes Read.
	go func() {
		var screen []byte
		chunk := make([]byte, 1024)
		for len(screen) < 64<<10 {
			n, err := pty.Read(chunk)
			screen = append(screen, chunk[:n]...)
			if bytes.Contains(screen, []byte("2. beta")) {
				_, err = pty.Write([]byte("2\r"))
				inputSent <- err
				return
			}
			if err != nil {
				inputSent <- err
				return
			}
		}
		inputSent <- fmt.Errorf("numbered picker did not render")
	}()
	selector := TerminalProviderSelector{In: pty.Slave(), Out: pty.Slave()}
	selected, err := selector.SelectProvider(ctx, []ProviderChoice{
		{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"},
	}, "gamma")
	_ = pty.Close()
	require.NoError(t, <-inputSent)
	require.NoError(t, err)
	require.Equal(t, "beta", selected)
}

func TestPickerRequiresBareLaunchAndBothTerminals(t *testing.T) {
	t.Parallel()
	pty, err := xpty.NewUnixPty(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pty.Close() })
	tty := pty.Slave()
	if !shouldPickProvider(nil, CodeOptions{Input: tty, Output: tty}) {
		t.Fatal("real terminal not detected")
	}
	for _, tc := range []struct {
		args []string
		opts CodeOptions
	}{
		{[]string{"prompt"}, CodeOptions{Input: tty, Output: tty}},
		{nil, CodeOptions{Input: tty, Output: &bytes.Buffer{}}},
		{nil, CodeOptions{Input: &bytes.Buffer{}, Output: tty}},
	} {
		if shouldPickProvider(tc.args, tc.opts) {
			t.Fatal("picker opened for args or redirected stream")
		}
	}
}

func TestTerminalPickerCtrlC(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	pty, err := xpty.NewUnixPty(80, 24)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pty.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sent := make(chan error, 1)
	go func() {
		var screen []byte
		chunk := make([]byte, 1024)
		for len(screen) < 64<<10 {
			n, err := pty.Read(chunk)
			screen = append(screen, chunk[:n]...)
			if bytes.Contains(screen, []byte("1. alpha")) {
				_, err = pty.Write([]byte{3})
				sent <- err
				return
			}
			if err != nil {
				sent <- err
				return
			}
		}
		sent <- fmt.Errorf("picker did not render")
	}()
	selector := TerminalProviderSelector{In: pty.Slave(), Out: pty.Slave()}
	selected, err := selector.SelectProvider(ctx, []ProviderChoice{{Name: "alpha"}}, "alpha")
	_ = pty.Close()
	require.NoError(t, <-sent)
	require.Empty(t, selected)
	require.ErrorIs(t, err, huh.ErrUserAborted)
}
