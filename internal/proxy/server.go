package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
)

// maxRequestBodyBytes caps the size of /v1/messages request bodies the proxy
// will read into memory before forwarding. Anthropic Messages requests are
// JSON conversation payloads; 10 MiB is comfortably above realistic prompts
// while preventing unbounded reads from a misbehaving or hostile client.
const maxRequestBodyBytes int64 = 10 << 20

// Proxy is a local HTTP server that forwards Anthropic API requests to a provider.
type Proxy struct {
	providerURL string
	authToken   string
	oauthToken  string
	rules       []config.TransformRule
	client      *http.Client
	listener    net.Listener
	server      *http.Server
	lifetime    context.Context
	cancel      context.CancelFunc
}

// NewProxy creates a new proxy that forwards requests to providerURL.
func NewProxy(providerURL, authToken, oauthToken string, rules []config.TransformRule) *Proxy {
	lifetime, cancel := context.WithCancel(context.Background())
	return &Proxy{
		lifetime: lifetime, cancel: cancel,
		providerURL: strings.TrimRight(providerURL, "/"),
		authToken:   authToken,
		oauthToken:  oauthToken,
		rules:       rules,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				DialContext:         (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
				TLSHandshakeTimeout: 15 * time.Second,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Start binds to a random port on localhost and begins serving.
// Returns the port number the proxy is listening on.
func (p *Proxy) Start() (int, error) {
	var err error
	p.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("bind to localhost: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	mux.HandleFunc("POST /v1/messages/count_tokens", p.handleMessages)

	p.server = &http.Server{Handler: mux}

	go func() {
		if err := p.server.Serve(p.listener); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server error", "error", err)
		}
	}()

	port := p.listener.Addr().(*net.TCPAddr).Port
	slog.Debug("proxy started", "port", port, "target", p.providerURL)
	return port, nil
}

// Shutdown gracefully stops the proxy server.
func (p *Proxy) Shutdown(ctx context.Context) error {
	p.cancel()
	if transport, ok := p.client.Transport.(*http.Transport); ok {
		defer transport.CloseIdleConnections()
	}
	if p.server == nil {
		return nil
	}
	if err := p.server.Shutdown(ctx); err != nil {
		_ = p.server.Close()
		return err
	}
	return nil
}

func (p *Proxy) handleMessages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(p.lifetime, cancel)
	defer func() { stop(); cancel() }()
	r = r.WithContext(ctx)
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			slog.Warn("request body exceeds limit", "limit", maxRequestBodyBytes, "error", err)
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
			return
		}
		slog.Error("read request body", "error", err)
		writeError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}

	body, extraHeaders, err := applyRules(p.rules, body, r.URL.Path == "/v1/messages/count_tokens")
	if err != nil {
		slog.Error("apply transform rules", "error", err)
		writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid request body")
		return
	}

	logRequestModel(body)

	resp, err := p.forwardRequest(r, body, extraHeaders)
	if err != nil {
		slog.Error("forward request", "error", err)
		writeError(w, http.StatusBadGateway, "api_error", "failed to forward request")
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w, resp)

	if isSSE(resp) {
		p.streamSSE(r.Context(), w, resp)
		return
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.Error("copy response body", "error", err)
	}
}

func (p *Proxy) forwardRequest(orig *http.Request, body []byte, extraHeaders map[string]string) (*http.Response, error) {
	targetURL := p.providerURL + orig.URL.Path

	req, err := http.NewRequestWithContext(
		orig.Context(), http.MethodPost, targetURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	copyRequestHeader(req, orig, "anthropic-version", "2023-06-01")
	copyRequestHeader(req, orig, "anthropic-beta", "")

	for k, v := range extraHeaders {
		if strings.EqualFold(k, "host") {
			req.Host = v
			continue
		}
		if strings.EqualFold(k, "authorization") || strings.EqualFold(k, "x-api-key") {
			continue
		}
		req.Header.Set(k, v)
	}
	if p.authToken != "" {
		req.Header.Set("x-api-key", p.authToken)
	}
	if p.oauthToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.oauthToken)
	}

	return p.client.Do(req)
}

// copyRequestHeader copies a header from orig to dst. If absent in orig and
// fallback is non-empty, uses the fallback value.
func copyRequestHeader(dst, orig *http.Request, key, fallback string) {
	if v := orig.Header.Get(key); v != "" {
		dst.Header.Set(key, v)
		return
	}
	if fallback != "" {
		dst.Header.Set(key, fallback)
	}
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for _, key := range []string{
		"Content-Type",
		"Anthropic-Version",
		"X-Request-Id",
		"Request-Id",
		"Retry-After",
	} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
		}
	}
	for key, values := range resp.Header {
		if strings.HasPrefix(strings.ToLower(key), "anthropic-ratelimit-") {
			w.Header()[key] = append([]string(nil), values...)
		}
	}
}

func isSSE(resp *http.Response) bool {
	return strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream")
}

func (p *Proxy) streamSSE(ctx context.Context, w http.ResponseWriter, resp *http.Response) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing")
		writeError(w, http.StatusBadGateway, "api_error", "streaming not supported")
		return
	}

	w.WriteHeader(resp.StatusCode)
	flusher.Flush()

	buf := make([]byte, 4096)
	for {
		// Stop promptly when the downstream client disconnects or the proxy is
		// shutting down, instead of draining the full upstream response.
		if err := ctx.Err(); err != nil {
			slog.Debug("SSE stream canceled by client", "error", err)
			return
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				slog.Error("write SSE chunk", "error", writeErr)
				return
			}
			flusher.Flush()
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			// A canceled request context surfaces here as a read error; log it
			// as cancellation rather than a stream fault.
			if ctx.Err() != nil {
				slog.Debug("SSE stream canceled by client", "error", ctx.Err())
				return
			}
			slog.Error("read SSE stream", "error", err)
			return
		}
	}
}

func logRequestModel(body []byte) {
	const marker = `"model":"`
	s := string(body)
	idx := strings.Index(s, marker)
	if idx < 0 {
		slog.Debug("proxy request", "model", "unknown")
		return
	}
	start := idx + len(marker)
	end := strings.IndexByte(s[start:], '"')
	if end < 0 {
		slog.Debug("proxy request", "model", "unknown")
		return
	}
	slog.Debug("proxy request", "model", s[start:start+end])
}

func writeError(w http.ResponseWriter, status int, kind, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error": map[string]string{"type": kind, "message": message}})
}
