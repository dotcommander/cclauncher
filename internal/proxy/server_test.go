package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleMessages_RejectsOversizedBody verifies that a request body larger
// than maxRequestBodyBytes is rejected with 413 before any upstream forwarding
// happens. The upstream server is wired to fail the test if it is ever called.
func TestHandleMessages_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	upstreamHit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	p := NewProxy(upstream.URL, "test-token", "", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	proxySrv := httptest.NewServer(mux)
	t.Cleanup(proxySrv.Close)

	oversized := bytes.Repeat([]byte("a"), int(maxRequestBodyBytes)+1)

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, proxySrv.URL+"/v1/messages",
		bytes.NewReader(oversized))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode,
		"oversized request body must be rejected with 413")
	require.False(t, upstreamHit, "upstream must not be contacted for oversized request")
}

// TestHandleMessages_AcceptsBodyAtLimit verifies a request body at exactly the
// limit is accepted and forwarded upstream.
func TestHandleMessages_AcceptsBodyAtLimit(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, int(maxRequestBodyBytes), len(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	p := NewProxy(upstream.URL, "test-token", "", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	proxySrv := httptest.NewServer(mux)
	t.Cleanup(proxySrv.Close)

	// JSON body padded with whitespace to exactly the size limit so ApplyRules
	// has valid (no-op) input and the read consumes the full bounded reader.
	const prefix = `{"model":"test-model","x":"`
	const suffix = `"}`
	padLen := int(maxRequestBodyBytes) - len(prefix) - len(suffix)
	require.Positive(t, padLen)
	body := prefix + strings.Repeat("a", padLen) + suffix
	require.Equal(t, int(maxRequestBodyBytes), len(body))

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, proxySrv.URL+"/v1/messages",
		strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
