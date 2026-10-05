package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRuleChainingAndNumbers(t *testing.T) {
	t.Parallel()
	rules := []config.TransformRule{
		{ModelPattern: "*", ModifyBody: map[string]any{"model": "second", "max_tokens": 200}},
		{ModelPattern: "second", SetModel: "third", SetMaxTokens: 300},
	}
	rules[1].TokenRange.Min = 200
	body := []byte(`{"model":"first","max_tokens":10,"integer":9007199254740993}`)
	out, _, err := ApplyRules(rules, body)
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"third"`)
	require.Contains(t, string(out), `"max_tokens":300`)
	require.Contains(t, string(out), `9007199254740993`)
	_, _, err = ApplyRules(rules, append(body, []byte(` {}`)...))
	require.Error(t, err)
	unmatched := []byte(" {\"model\":\"first\"} \n")
	out, _, err = ApplyRules([]config.TransformRule{{ModelPattern: "unknown"}}, unmatched)
	require.NoError(t, err)
	require.Equal(t, unmatched, out)
}

func TestCountingRouting(t *testing.T) {
	t.Parallel()
	temperature := 0.5
	rules := []config.TransformRule{
		{ModelPattern: "*", SetModel: "routed", SetMaxTokens: 10, SetTemperature: &temperature, ModifyBody: map[string]any{"top_p": 1, "top_k": 2, "stop_sequences": []any{"stop"}, "stream": true, "metadata": map[string]any{"user_id": "user"}}, AddHeaders: map[string]string{"Host": "virtual.test", "X-Routing": "yes", "x-api-key": "placeholder", "Authorization": "placeholder"}},
		{SetModel: "incorrect"},
	}
	rules[1].TokenRange.Max = 10
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages/count_tokens", r.URL.Path)
		require.Equal(t, "virtual.test", r.Host)
		require.Equal(t, "configured", r.Header.Get("X-Api-Key"))
		require.Empty(t, r.Header.Get("Authorization"))
		require.Equal(t, "yes", r.Header.Get("X-Routing"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "routed", body["model"])
		for _, key := range []string{"max_tokens", "temperature", "top_p", "top_k", "stop_sequences", "stream"} {
			require.NotContains(t, body, key)
		}
		w.Header().Set("Retry-After", "3")
		w.Header().Set("Request-Id", "fixture")
		w.Header().Set("Anthropic-Ratelimit-Requests-Remaining", "0")
		w.Header().Set("X-Private", "hidden")
		w.WriteHeader(429)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"rate_limit_error"}}`)
	}))
	t.Cleanup(upstream.Close)
	p := NewProxy(upstream.URL, "configured", "", rules)
	port, err := p.Start()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, p.Shutdown(context.Background())) })
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/v1/messages/count_tokens", port), strings.NewReader(`{"model":"first","messages":[]}`))
	require.NoError(t, err)
	req.Header.Set("X-Api-Key", "incoming")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, 429, resp.StatusCode)
	require.Equal(t, "3", resp.Header.Get("Retry-After"))
	require.Equal(t, "fixture", resp.Header.Get("Request-Id"))
	require.Equal(t, "0", resp.Header.Get("Anthropic-Ratelimit-Requests-Remaining"))
	require.Empty(t, resp.Header.Get("X-Private"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "rate_limit_error")
}

func TestGeneratedErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		body   []byte
		status int
	}{
		{"invalid", []byte(`{} {}`), 400},
		{"array", []byte(`[]`), 400},
		{"oversized", bytes.Repeat([]byte("x"), int(maxRequestBodyBytes)+1), 413},
		{"transport", []byte(`{}`), 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewProxy("http://127.0.0.1:1", "", "", nil)
			w := httptest.NewRecorder()
			p.handleMessages(w, httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(tc.body)))
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
			require.Equal(t, "error", envelope["type"])
		})
	}
}

func TestShutdownCancelsCommittedStream(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(done)
	}))
	t.Cleanup(upstream.Close)
	p := NewProxy(upstream.URL, "", "", nil)
	port, err := p.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(context.Background()) })
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/v1/messages", port), "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	prefix := make([]byte, len("data: first\n\n"))
	_, err = io.ReadFull(resp.Body, prefix)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(ctx))
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("upstream stream not canceled")
	}
	tail, _ := io.ReadAll(resp.Body)
	require.NotContains(t, string(tail), `"type":"error"`)
}

func TestTransportUsesConfiguredProxy(t *testing.T) {
	t.Parallel()
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "http://upstream.invalid/v1/messages", r.RequestURI)
		require.Equal(t, "upstream.invalid", r.Host)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(proxyServer.Close)
	p := NewProxy("http://upstream.invalid", "", "", nil)
	transport := p.client.Transport.(*http.Transport)
	require.NotNil(t, transport.Proxy, "production transport must consult environment proxies")
	proxyURL, err := url.Parse(proxyServer.URL)
	require.NoError(t, err)
	transport.Proxy = http.ProxyURL(proxyURL)
	w := httptest.NewRecorder()
	p.handleMessages(w, httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"ok":true}`, w.Body.String())
	transport.CloseIdleConnections()
}

func TestUniversalMessagePatternAndAbsentTokens(t *testing.T) {
	t.Parallel()
	rule := config.TransformRule{MessagePattern: "*", SetModel: "universal"}
	out, _, err := ApplyRules([]config.TransformRule{rule}, []byte(`{"model":"first"}`))
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"universal"`)
	rule.TokenRange.Max = 100
	input := []byte(`{ "model": "first" }`)
	out, _, err = ApplyRules([]config.TransformRule{rule}, input)
	require.NoError(t, err)
	require.Equal(t, input, out)
}
