package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestRunChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		provider   config.Provider
		wantCheck  string // check ID to assert
		wantStatus Status
		reasonSub  string // substring required in Reason ("" to skip)
	}{
		{"auth fail: required no token", config.Provider{BaseURL: "https://x", Model: "m"}, "auth", StatusFail, "auth required"},
		{"auth pass: has token", config.Provider{AuthToken: "t", BaseURL: "https://x", Model: "m"}, "auth", StatusPass, "credential present"},
		{"auth warn: not required", config.Provider{AuthRequired: boolPtr(false), BaseURL: "https://x", Model: "m"}, "auth", StatusWarn, "authRequired: false"},
		{"fields fail: empty baseURL", config.Provider{AuthToken: "t", Model: "m"}, "fields", StatusFail, "baseUrl"},
		{"fields fail: empty model", config.Provider{AuthToken: "t", BaseURL: "https://x"}, "fields", StatusFail, "model"},
		{"fields pass", config.Provider{AuthToken: "t", BaseURL: "https://x", Model: "m"}, "fields", StatusPass, ""},
		{"baseurl warn: /v1 suffix", config.Provider{AuthToken: "t", BaseURL: "https://api.example.com/v1", Model: "m"}, "baseurl", StatusWarn, "/v1/v1/messages"},
		{"baseurl warn: paas/v4", config.Provider{AuthToken: "t", BaseURL: "https://api.z.ai/api/paas/v4", Model: "m"}, "baseurl", StatusWarn, "/api/anthropic"},
		{"baseurl pass: anthropic", config.Provider{AuthToken: "t", BaseURL: "https://api.z.ai/api/anthropic", Model: "m"}, "baseurl", StatusPass, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := RunChecks("p", tt.provider)
			require.Len(t, results, 3, "RunChecks must return 3 results")
			assert.Equal(t, "auth", results[0].Check)
			assert.Equal(t, "fields", results[1].Check)
			assert.Equal(t, "baseurl", results[2].Check)

			var got *CheckResult
			for i := range results {
				if results[i].Check == tt.wantCheck {
					got = &results[i]
				}
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantStatus, got.Status)
			if tt.reasonSub != "" {
				assert.Contains(t, got.Reason, tt.reasonSub)
			}
		})
	}
}

func TestProbeReachability(t *testing.T) {
	t.Parallel()

	t.Run("empty baseURL warns", func(t *testing.T) {
		t.Parallel()
		r := ProbeReachability(context.Background(), &http.Client{}, "p", config.Provider{})
		assert.Equal(t, "net", r.Check)
		assert.Equal(t, StatusWarn, r.Status)
		assert.Contains(t, r.Reason, "baseUrl is empty")
	})

	t.Run("reachable server passes", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)
		r := ProbeReachability(context.Background(), srv.Client(), "p", config.Provider{BaseURL: srv.URL})
		assert.Equal(t, StatusPass, r.Status)
		assert.Contains(t, r.Reason, "HTTP")
	})

	t.Run("unreachable server warns", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL
		srv.Close() // nothing listening now
		r := ProbeReachability(context.Background(), &http.Client{Timeout: DoctorNetTimeout}, "p", config.Provider{BaseURL: url})
		assert.Equal(t, StatusWarn, r.Status)
		assert.Contains(t, r.Reason, "unreachable")
	})
}
