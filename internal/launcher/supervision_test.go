package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSupervisionHelper(t *testing.T) {
	t.Parallel()
	role := os.Getenv("CCL_SUPERVISION_FIXTURE")
	if role == "" {
		return
	}
	switch role {
	case "hup-parent", "hup-parent-ignore":
		childRole := "hup-child"
		if role == "hup-parent-ignore" {
			childRole = "hup-child-ignore"
		}
		env := append(os.Environ(), "CCL_SUPERVISION_FIXTURE="+childRole)
		err := superviseChildWithGrace(context.Background(), os.Args[0], env, []string{"-test.run=^TestSupervisionHelper$"}, nil, os.Stdout, os.Stderr, 100*time.Millisecond)
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			os.Exit(2)
		}
		want := syscall.SIGTERM
		if role == "hup-parent-ignore" {
			want = syscall.SIGKILL
		}
		if exit.Sys().(syscall.WaitStatus).Signal() != want {
			os.Exit(3)
		}
		os.Exit(0)
	case "hup-child", "hup-child-ignore":
		if role == "hup-child-ignore" {
			signal.Ignore(syscall.SIGTERM)
		}
		fmt.Println(os.Getpid())
		<-time.After(time.Hour)
	case "exit":
		os.Exit(7)
	case "signal":
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		<-time.After(time.Hour)
	case "cancel":
		fmt.Println("ready")
		<-time.After(time.Hour)
	case "ignore":
		signal.Ignore(syscall.SIGTERM)
		fmt.Println("ready")
		<-time.After(time.Hour)
	}
}

func TestChildExitStatus(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"exit", "signal"} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			env := append(os.Environ(), "CCL_SUPERVISION_FIXTURE="+role)
			err := superviseChild(context.Background(), os.Args[0], env, []string{"-test.run=^TestSupervisionHelper$"}, nil, io.Discard, io.Discard)
			var exit *exec.ExitError
			require.True(t, errors.As(err, &exit))
			status := exit.Sys().(syscall.WaitStatus)
			if role == "exit" {
				require.Equal(t, 7, status.ExitStatus())
			} else {
				require.Equal(t, syscall.SIGTERM, status.Signal())
			}
		})
	}
}

func TestChildCancellationAndEscalation(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"cancel", "ignore"} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			reader, writer, err := os.Pipe()
			require.NoError(t, err)
			t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
			result := make(chan error, 1)
			go func() {
				result <- superviseChildWithGrace(ctx, os.Args[0], append(os.Environ(), "CCL_SUPERVISION_FIXTURE="+role), []string{"-test.run=^TestSupervisionHelper$"}, nil, writer, io.Discard, 20*time.Millisecond)
			}()
			scanner := bufio.NewScanner(reader)
			require.True(t, scanner.Scan())
			require.Equal(t, "ready", scanner.Text())
			cancel()
			select {
			case err := <-result:
				var exit *exec.ExitError
				require.True(t, errors.As(err, &exit))
				want := syscall.SIGTERM
				if role == "ignore" {
					want = syscall.SIGKILL
				}
				require.Equal(t, want, exit.Sys().(syscall.WaitStatus).Signal())
			case <-time.After(2 * time.Second):
				t.Fatal("child did not terminate")
			}
		})
	}
}

func TestProxyEnvironmentPreservesExclusions(t *testing.T) {
	t.Parallel()
	input := []string{"NO_PROXY=internal.test", "no_proxy=existing.test,localhost", "HTTP_PROXY=http://example.test", "OTHER=value"}
	got := envMap(proxyEnvironment(input))
	require.Equal(t, "internal.test,127.0.0.1,localhost,::1", got["NO_PROXY"])
	require.Equal(t, "existing.test,localhost,127.0.0.1,::1", got["no_proxy"])
	require.Equal(t, "http://example.test", got["HTTP_PROXY"])
	require.Equal(t, "NO_PROXY=internal.test", input[0])
}

func TestLocalProxyCredentialIsolation(t *testing.T) {
	t.Parallel()
	original := config.Provider{BaseURL: "https://upstream.test", AuthToken: "upstream-secret", APIKey: "other-secret", OAuthToken: "oauth-secret", Model: "model"}
	local := localProxyProvider(original, 12345)
	env := envMap(SetupEnvironment(local, config.OptimizationConfig{}))
	require.Equal(t, "http://127.0.0.1:12345", env["ANTHROPIC_BASE_URL"])
	require.Equal(t, "ccl-local-proxy", env["ANTHROPIC_AUTH_TOKEN"])
	require.Empty(t, env["CLAUDE_CODE_OAUTH_TOKEN"])
	require.Equal(t, "upstream-secret", original.AuthToken)
	for _, value := range env {
		require.NotEqual(t, "upstream-secret", value)
		require.NotEqual(t, "other-secret", value)
		require.NotEqual(t, "oauth-secret", value)
	}
}

func TestSIGHUPTerminatesSupervisedChild(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"hup-parent", "hup-parent-ignore"} {
		t.Run(role, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSupervisionHelper$")
			cmd.Env = append(os.Environ(), "CCL_SUPERVISION_FIXTURE="+role)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			require.NoError(t, cmd.Start())
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			ready := make(chan string, 1)
			go func() { scanner := bufio.NewScanner(stdout); scanner.Scan(); ready <- scanner.Text() }()
			var line string
			select {
			case line = <-ready:
			case <-time.After(5 * time.Second):
				t.Fatal("child did not become ready")
			}
			pid, err := strconv.Atoi(line)
			require.NoError(t, err)
			child, err := os.FindProcess(pid)
			require.NoError(t, err)
			t.Cleanup(func() { _ = child.Kill() })
			require.NoError(t, cmd.Process.Signal(syscall.SIGHUP))
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("SIGHUP did not stop supervisor")
			}
		})
	}
}
