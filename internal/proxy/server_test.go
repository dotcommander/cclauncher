package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// TestStreamSSE_ForwardsChunksIncrementally verifies that an upstream
// text/event-stream response is relayed chunk-by-chunk with the SSE content
// type preserved.
func TestStreamSSE_ForwardsChunksIncrementally(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl, ok := w.(http.Flusher)
		require.True(t, ok)
		for _, ev := range []string{"event: a\ndata: 1\n\n", "event: b\ndata: 2\n\n"} {
			_, _ = io.WriteString(w, ev)
			fl.Flush()
		}
	}))
	t.Cleanup(upstream.Close)

	p := NewProxy(upstream.URL, "test-token", "", nil)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	proxySrv := httptest.NewServer(mux)
	t.Cleanup(proxySrv.Close)

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, proxySrv.URL+"/v1/messages",
		strings.NewReader(`{"model":"test-model"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"),
		"SSE content type must be preserved")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "data: 1")
	require.Contains(t, string(body), "data: 2")
}

// TestStreamSSE_StopsOnClientDisconnect verifies that when the downstream
// client cancels its request mid-stream, the proxy stops draining the upstream
// body instead of reading it to completion. The upstream blocks after the first
// event until its request context is canceled, which only happens once the
// proxy abandons the stream.
func TestStreamSSE_StopsOnClientDisconnect(t *testing.T) {
	t.Parallel()

	upstreamDone := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl, ok := w.(http.Flusher)
		require.True(t, ok)
		_, _ = io.WriteString(w, "event: a\ndata: 1\n\n")
		fl.Flush()
		// Block until the proxy abandons the stream (its request ctx cancels,
		// propagating to this upstream request's context).
		<-r.Context().Done()
		close(upstreamDone)
	}))
	t.Cleanup(upstream.Close)

	p := NewProxy(upstream.URL, "test-token", "", nil)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	proxySrv := httptest.NewServer(mux)
	t.Cleanup(proxySrv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost, proxySrv.URL+"/v1/messages",
		strings.NewReader(`{"model":"test-model"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read the first event, then disconnect.
	buf := make([]byte, 64)
	_, _ = resp.Body.Read(buf)
	_ = resp.Body.Close()
	cancel()

	select {
	case <-upstreamDone:
		// Proxy abandoned the stream → upstream request context canceled.
	case <-time.After(5 * time.Second):
		t.Fatal("proxy did not stop draining upstream after client disconnect")
	}
}
