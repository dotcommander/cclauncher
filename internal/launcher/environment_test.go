package launcher

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChildProxyEnvironmentHelper(t *testing.T) {
	t.Parallel()
	if os.Getenv("CCL_CHILD_PROXY_FIXTURE") == "" {
		return
	}
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		exclusions := strings.Split(os.Getenv(key), ",")
		require.Contains(t, exclusions, "127.0.0.1")
		require.Contains(t, exclusions, "localhost")
		require.Contains(t, exclusions, "::1")
	}
	for _, host := range []string{"127.0.0.1", "localhost", "[::1]", "kept.internal"} {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+host+"/", nil)
		require.NoError(t, err)
		selection, err := http.ProxyFromEnvironment(request)
		require.NoError(t, err)
		require.Nil(t, selection, "child exclusion %s", host)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://upstream.invalid/", nil)
	require.NoError(t, err)
	selection, err := http.ProxyFromEnvironment(request)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.Equal(t, os.Getenv("HTTP_PROXY"), selection.String())
}

func TestChildEnvironmentProxySelection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestChildProxyEnvironmentHelper$")
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToLower(key) {
		case "http_proxy", "https_proxy", "all_proxy", "no_proxy", "request_method", "ccl_child_proxy_fixture":
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "CCL_CHILD_PROXY_FIXTURE=1", "HTTP_PROXY=http://127.0.0.1:9", "NO_PROXY=kept.internal", "no_proxy=kept.lower")
	command.Env = proxyEnvironment(command.Env)
	output, err := command.CombinedOutput()
	require.NoError(t, err, fmt.Sprintf("child proxy environment: %s", output))
}
