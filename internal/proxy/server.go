package proxy

import (
	"bytes"
	"context"
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
}

// NewProxy creates a new proxy that forwards requests to providerURL.
func NewProxy(providerURL, authToken, oauthToken string, rules []config.TransformRule) *Proxy {
	return &Proxy{
		providerURL: strings.TrimRight(providerURL, "/"),
		authToken:   authToken,
		oauthToken:  oauthToken,
		rules:       rules,
		client: &http.Client{
			Transport: &http.Transport{
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

	p.server = &http.Server{Handler: mux}

	go func() {
		if err := p.server.Serve(p.listener); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server error", "error", err)
		}
	}()

	port := p.listener.Addr().(*net.TCPAddr).Port
	slog.Info("proxy started", "port", port, "target", p.providerURL)
	return port, nil
}

// Shutdown gracefully stops the proxy server.
func (p *Proxy) Shutdown(ctx context.Context) error {
	if p.server == nil {
		return nil
	}
	return p.server.Shutdown(ctx)
}

func (p *Proxy) handleMessages(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			slog.Warn("request body exceeds limit", "limit", maxRequestBodyBytes, "error", err)
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		slog.Error("read request body", "error", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	body, extraHeaders, err := ApplyRules(p.rules, body)
	if err != nil {
		slog.Error("apply transform rules", "error", err)
		http.Error(w, "failed to apply transform rules", http.StatusInternalServerError)
		return
	}

	logRequestModel(body)

	resp, err := p.forwardRequest(r, body, extraHeaders)
	if err != nil {
		slog.Error("forward request", "error", err)
		http.Error(w, "failed to forward request", http.StatusBadGateway)
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
	targetURL := p.providerURL + "/v1/messages"

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

	if p.authToken != "" {
		req.Header.Set("x-api-key", p.authToken)
	}
	if p.oauthToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.oauthToken)
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
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
	} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
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
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	flusher.Flush()

	buf := make([]byte, 4096)
	for {
		// Stop promptly when the downstream client disconnects or the proxy is
		// shutting down, instead of draining the full upstream response.
		if err := ctx.Err(); err != nil {
			slog.Info("SSE stream canceled by client", "error", err)
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
				slog.Info("SSE stream canceled by client", "error", ctx.Err())
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
		slog.Info("proxy request", "model", "unknown")
		return
	}
	start := idx + len(marker)
	end := strings.IndexByte(s[start:], '"')
	if end < 0 {
		slog.Info("proxy request", "model", "unknown")
		return
	}
	slog.Info("proxy request", "model", s[start:start+end])
}
