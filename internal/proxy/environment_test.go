package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentProxyHelper(t *testing.T) {
	t.Parallel()
	mode := os.Getenv("CCL_PROXY_ENV_FIXTURE")
	if mode == "" {
		return
	}
	p := NewProxy("http://upstream.invalid", "", "", nil)
	request := httptest.NewRequest(http.MethodPost, "http://upstream.invalid/v1/messages", strings.NewReader(`{}`))
	selection, err := p.client.Transport.(*http.Transport).Proxy(request)
	require.NoError(t, err)
	if mode == "excluded" {
		require.Nil(t, selection)
		return
	}
	require.NotNil(t, selection)
	require.Equal(t, os.Getenv("HTTP_PROXY"), selection.String())
	response, err := p.forwardRequest(request, []byte(`{}`), nil)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "environment-routed", string(body))
}

func TestEnvironmentProxySelection(t *testing.T) {
	t.Parallel()
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "http://upstream.invalid/v1/messages", r.RequestURI)
		_, _ = io.WriteString(w, "environment-routed")
	}))
	t.Cleanup(fixture.Close)
	for _, mode := range []string{"routed", "excluded"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			t.Cleanup(cancel)
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEnvironmentProxyHelper$")
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				switch strings.ToLower(key) {
				case "http_proxy", "https_proxy", "all_proxy", "no_proxy", "request_method", "ccl_proxy_env_fixture":
					continue
				}
				command.Env = append(command.Env, entry)
			}
			command.Env = append(command.Env, "CCL_PROXY_ENV_FIXTURE="+mode, "HTTP_PROXY="+fixture.URL)
			if mode == "excluded" {
				command.Env = append(command.Env, "NO_PROXY=upstream.invalid")
			}
			output, err := command.CombinedOutput()
			require.NoError(t, err, fmt.Sprintf("environment fixture: %s", output))
		})
	}
}
