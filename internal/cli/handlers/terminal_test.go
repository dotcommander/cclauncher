//go:build !windows

package handlers

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/x/xpty"
)

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
