package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

type reachabilityTransport func(*http.Request) (*http.Response, error)

func (f reachabilityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestIsLoopbackEndpoint(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		url   string
		local bool
	}{
		{"http://localhost:1234", true},
		{"https://LOCALHOST.:1234/path", true},
		{"http://127.0.0.1:8080", true},
		{"http://127.0.0.2:8080", true},
		{"http://[::1]:8000", true},
		{"https://api.example.com", false},
		{"http://localhost.example.com", false},
		{"http://192.168.1.2:8000", false},
		{"not-a-url", false},
		{"ftp://localhost", false},
	} {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.local, isLoopbackEndpoint(tt.url))
		})
	}
}

func TestReachableProviderChoices(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"cloud":         {BaseURL: "https://api.example.com", Model: "remote", APIKey: "test-key"},
			"local-ready":   {BaseURL: "http://localhost:1234", Model: "local", APIKey: "test-key"},
			"local-stopped": {BaseURL: "http://127.0.0.1:8080", Model: "local", APIKey: "test-key"},
			"missing-auth":  {BaseURL: "http://localhost:8000", Model: "local"},
		},
		Doctor: config.DoctorConfig{ProbeConcurrency: 2},
	}
	client := &http.Client{Transport: reachabilityTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodHead || r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
			return nil, errors.New("probe must be unauthenticated HEAD")
		}
		switch r.URL.Host {
		case "localhost:1234":
			return &http.Response{StatusCode: http.StatusMethodNotAllowed, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		case "127.0.0.1:8080":
			return nil, errors.New("connection refused")
		default:
			t.Errorf("unexpected probe of %s", r.URL.Host)
			return nil, errors.New("unexpected endpoint")
		}
	})}
	choices, err := reachableProviderChoices(context.Background(), cfg, client)
	require.NoError(t, err)
	require.Equal(t, []ProviderChoice{{Name: "cloud", Model: "remote"}, {Name: "local-ready", Model: "local"}}, choices)
}

func TestProviderReachabilityCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	selector := &fakeProviderSelector{}
	_, err := pickProvider(ctx, &config.Config{}, selector)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, selector.calls)
}

func TestPickProviderLocalReachability(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("local HTTP integration")
	}
	ready := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ready.Close)
	stopped := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	stoppedURL := stopped.URL
	stopped.Close()
	optional := false
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"ready":   {BaseURL: ready.URL, Model: "local", AuthRequired: &optional},
			"stopped": {BaseURL: stoppedURL, Model: "local", AuthRequired: &optional},
		},
		Doctor: config.DoctorConfig{ProbeConcurrency: 2},
		CLI:    config.CLIConfig{DefaultProvider: "stopped"},
	}
	selector := &fakeProviderSelector{selected: "ready"}
	name, err := pickProvider(context.Background(), cfg, selector)
	require.NoError(t, err)
	require.Equal(t, "ready", name)
	require.Equal(t, []ProviderChoice{{Name: "ready", Model: "local"}}, selector.choices)
	delete(cfg.Providers, "ready")
	selector.calls = 0
	_, err = pickProvider(context.Background(), cfg, selector)
	require.ErrorContains(t, err, "no launchable providers")
	require.Zero(t, selector.calls)
}

func TestLocalProviderClientBoundaries(t *testing.T) {
	t.Parallel()
	client := localProviderClient()
	t.Cleanup(client.CloseIdleConnections)
	require.Nil(t, client.Transport.(*http.Transport).Proxy)
	require.ErrorIs(t, client.CheckRedirect(&http.Request{}, nil), http.ErrUseLastResponse)
	require.Positive(t, client.Timeout)
}
