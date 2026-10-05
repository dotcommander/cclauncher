//go:build darwin || linux

package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/xpty"
	"github.com/stretchr/testify/require"
)

func TestTerminalHelper(t *testing.T) {
	t.Parallel()
	role := os.Getenv("CCL_TERMINAL_FIXTURE")
	switch role {
	case "parent":
		env := append(os.Environ(), "CCL_TERMINAL_FIXTURE=child")
		err := superviseChild(context.Background(), os.Args[0], env, []string{"-test.run=^TestTerminalHelper$"}, os.Stdin, os.Stdout, os.Stderr)
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(99)
		}
		os.Exit(0)
	case "child":
		signals := make(chan os.Signal, 2)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		fmt.Printf("foreground=%t\n", sharedForeground(os.Stdin, os.Getpid()))
		received := <-signals
		if received == syscall.SIGINT {
			fmt.Println("child-interrupt")
			os.Exit(24)
		}
		fmt.Println("child-terminate")
		os.Exit(23)
	}
}

func TestForegroundTerminalAndParentSignals(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"terminal-interrupt", "parent-terminate"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			terminal, err := xpty.NewUnixPty(80, 24)
			require.NoError(t, err)
			t.Cleanup(func() { _ = terminal.Close() })
			command := exec.Command(os.Args[0], "-test.run=^TestTerminalHelper$")
			command.Env = append(os.Environ(), "CCL_TERMINAL_FIXTURE=parent")
			command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
			require.NoError(t, terminal.Start(command))
			finished := false
			t.Cleanup(func() {
				if !finished {
					_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
				}
			})
			lines := make(chan string, 16)
			// The reader ends when the owned PTY closes; writes cannot block on abandoned callers.
			go func() {
				scanner := bufio.NewScanner(terminal)
				for scanner.Scan() {
					select {
					case lines <- scanner.Text():
					default:
					}
				}
				close(lines)
			}()
			deadline := time.NewTimer(3 * time.Second)
			t.Cleanup(func() { deadline.Stop() })
			nextLine := func() string {
				select {
				case line, ok := <-lines:
					require.True(t, ok, "PTY closed before expected marker")
					return strings.TrimSpace(line)
				case <-deadline.C:
					t.Fatal("terminal fixture timed out")
					return ""
				}
			}
			require.Equal(t, "foreground=true", nextLine())
			wantStatus := 24
			wantMarker := "child-interrupt"
			if scenario == "terminal-interrupt" {
				_, err = terminal.Write([]byte{3})
				require.NoError(t, err)
			} else {
				wantStatus = 23
				wantMarker = "child-terminate"
				require.NoError(t, command.Process.Signal(syscall.SIGTERM))
			}
			found := false
			for !found {
				found = strings.Contains(nextLine(), wantMarker)
			}
			result := make(chan error, 1)
			go func() { result <- command.Wait() }()
			select {
			case err := <-result:
				finished = true
				var exit *exec.ExitError
				require.True(t, errors.As(err, &exit))
				require.Equal(t, wantStatus, exit.ExitCode())
			case <-deadline.C:
				t.Fatal("supervisor did not preserve child status")
			}
		})
	}
}
