package launcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// superviseChild keeps a shared foreground process group, so the terminal sends
// Ctrl-C directly to Claude. Parent-only cancellation uses SIGTERM instead.
func superviseChild(ctx context.Context, path string, env, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	return superviseChildWithGrace(ctx, path, env, args, stdin, stdout, stderr, 5*time.Second)
}

func superviseChildWithGrace(ctx context.Context, path string, env, args []string, stdin io.Reader, stdout, stderr io.Writer, grace time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.Command(path, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	signals := make(chan os.Signal, 4)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}
	done := make(chan error, 1)
	// Wait exits exactly once when the child terminates.
	go func() { done <- cmd.Wait() }()
	var timer *time.Timer
	var escalation <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	cancel := ctx.Done()
	terminate := func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		if timer == nil {
			timer = time.NewTimer(grace)
			escalation = timer.C
		}
	}
	for {
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("claude exited: %w", err)
			}
			return nil
		case <-cancel:
			cancel = nil
			terminate()
		case sig := <-signals:
			if sig == syscall.SIGTERM {
				terminate()
				continue
			}
			if !sharedForeground(stdin, cmd.Process.Pid) {
				_ = cmd.Process.Signal(sig)
			}
		case <-escalation:
			escalation = nil
			_ = cmd.Process.Kill()
		}
	}
}

func sharedForeground(stdin io.Reader, pid int) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	foreground, err := unix.IoctlGetInt(int(file.Fd()), unix.TIOCGPGRP)
	if err != nil {
		return false
	}
	childGroup, err := syscall.Getpgid(pid)
	return err == nil && foreground == childGroup && foreground == syscall.Getpgrp()
}
