# Bug Spec: Proxy SSE and Cancellation Hardening

## Objective Note
- Objective: harden the local proxy so streamed `/v1/messages` responses stop promptly and cleanly when the downstream client disconnects or the launcher shuts down.
- Invariants: preserve existing transformer behavior, auth/header forwarding, request body limit, and non-SSE response behavior.
- Scope boundary: `internal/proxy` request forwarding and SSE streaming; launcher shutdown may be touched only if required to prove cancellation behavior.
- Expected artifact: focused Go change plus regression tests.
- Stop condition: proxy tests and full repo tests pass, and the cancellation tests fail on the old behavior or explicitly characterize the guarded behavior.

## Evidence
- `internal/proxy/server.go:42` - proxy uses a dedicated `http.Client` with transport timeouts but no response-header timeout or stream-specific cancellation policy.
- `internal/proxy/server.go:85` - `handleMessages` owns request body reading, transformation, upstream forwarding, response copying, and SSE dispatch.
- `internal/proxy/server.go:120` - SSE responses are routed to `streamSSE`.
- `internal/proxy/server.go:134` - upstream requests already use `orig.Context()`, so downstream request cancellation is the native cancellation source for upstream work.
- `internal/proxy/server.go:189` - `streamSSE` reads from `resp.Body` and writes chunks until EOF, read error, or write error.
- `internal/proxy/server.go:200` - the SSE loop blocks on `resp.Body.Read(buf)` with no explicit `r.Context().Done()` select or cancellation-aware close path at the stream boundary.
- `internal/proxy/server_test.go:15` - existing proxy tests use `httptest` and binary status assertions for edge behavior.
- `internal/launcher/launcher.go:99` - launcher keeps the Go process alive while Claude talks through the proxy.
- `internal/launcher/launcher.go:117` - launcher shutdown currently uses a 5 second timeout and then calls `srv.Shutdown(ctx)`.
- `go.mod:1` - module is `github.com/dotcommander/cclauncher`.
- `go.mod:3` - project targets Go `1.24.0` with toolchain `go1.26.2`.

## Problem / Goal
The proxy forwards provider SSE streams, but stream ownership is implicit. The upstream request carries the downstream context, yet the SSE copy loop has no explicit cancellation boundary that closes the upstream body when the downstream request is canceled. This can leave tests and future maintainers unable to prove that client cancellation, write failure, and launcher shutdown all release the streaming upstream promptly.

Goal: make SSE streaming cancellation explicit, testable, and bounded without changing request transformation semantics or non-streaming proxy behavior.

## Computation Graph
- Stateful artifacts:
  - Downstream HTTP request context from Claude Code to the local proxy.
  - Upstream provider HTTP request created by `forwardRequest`.
  - Upstream SSE response body owned by `handleMessages` and consumed by `streamSSE`.
  - Local proxy server lifetime owned by `LaunchWithProxy`.
- Semantic boundaries:
  - `handleMessages`: HTTP boundary for `/v1/messages`, request size cap, transform application, and response routing.
  - `forwardRequest`: upstream provider boundary, auth/header application, context binding.
  - `streamSSE`: byte streaming boundary between provider SSE and downstream client.
  - `LaunchWithProxy`: process lifetime boundary between Claude Code and proxy server.
- Transforms:
  - Request body may be transformed by `ApplyRules`; SSE bytes must not be transformed.
  - Response headers are selectively copied before streaming.
  - Cancellation should transform a downstream disconnect into upstream stream closure, not into a synthetic success response.
- Consumers:
  - Claude Code consumes the local proxy as an Anthropic-compatible endpoint.
  - Provider consumes upstream `/v1/messages`.
  - Tests consume proxy behavior through `httptest`.
- Invalidation rules:
  - A stream becomes invalid when `r.Context()` is canceled, downstream write fails, upstream body read fails, upstream EOF arrives, or server shutdown begins.
  - A cancellation hardening fix is invalid if it buffers full SSE responses, rewrites event payloads, drops status/header forwarding, or changes non-SSE response semantics.

## Scope
In:
- Add explicit cancellation handling for SSE streams.
- Ensure downstream cancellation closes or otherwise unblocks the upstream response body promptly.
- Preserve streaming chunk forwarding and flushing.
- Add focused `internal/proxy` tests for downstream cancellation and normal SSE forwarding.
- Run `gofmt` on touched Go files.

Out:
- Changing transformer rule behavior.
- Adding retries, buffering, backpressure queues, reconnect support, or SSE parsing.
- Changing provider config, model selection, auth semantics, or CLI flags.
- Adding new dependencies.
- Creating CI/CD workflows.

## Target Shape
- `handleMessages` passes the downstream request context into SSE streaming, or `streamSSE` otherwise observes the request context directly.
- `streamSSE` exits on the first terminal condition:
  1. downstream context cancellation,
  2. downstream write error,
  3. upstream EOF,
  4. upstream read error.
- The implementation closes the upstream response body or uses a context-bound upstream request so cancellation releases blocked reads promptly.
- Logging distinguishes expected cancellation from unexpected upstream read/write failures where practical.
- Tests use `httptest` servers and channels/timers, not sleeps-only assertions.

## Requirements
1. SSE success path: a provider response with `Content-Type: text/event-stream` must be forwarded with status, copied headers, chunk bytes, and flush-capable streaming behavior preserved.
2. Downstream cancellation: when the downstream request context is canceled during an active SSE stream, the proxy must stop forwarding and the upstream handler must observe request cancellation or closed write/read state without waiting for launcher shutdown.
3. Upstream cleanup: the upstream response body must be closed on every SSE exit path.
4. Non-SSE preservation: existing JSON/non-SSE proxy behavior must remain unchanged.
5. No new dependencies: use the standard library and existing test packages only.

## Verification Surface
1. Focused proxy tests:
   ```sh
   go test ./internal/proxy
   ```
2. Full repo tests:
   ```sh
   go test ./...
   ```
3. Race check if implementation introduces goroutines or shared state beyond existing server goroutines:
   ```sh
   go test -race ./internal/proxy
   ```
4. Optional manual smoke for a configured transformer provider:
   ```sh
   ccl --provider <provider-with-transformer> --help
   ```
   Only run this when the local config has a safe provider fixture; source tests are the required gate for this spec.

## Phases
1. A0: Add characterization tests in `internal/proxy/server_test.go` for normal SSE forwarding and downstream cancellation. The cancellation test should fail or hang under the current implicit loop unless the implementation is added in the same patch.
2. A: Thread request context into `streamSSE`, make the stream loop cancellation-aware, and ensure upstream response bodies are closed promptly on exit.
3. B: Improve cancellation logging only if tests expose noisy expected errors; keep log changes minimal and avoid broad observability work.

## reject_if
- Reject if the solution reads an entire SSE response into memory before writing downstream.
- Reject if it parses, rewrites, coalesces, or drops SSE event bytes.
- Reject if it changes request transformer semantics, auth headers, provider URL construction, or non-SSE response status/body handling.
- Reject if tests depend on real network providers, wall-clock sleeps as the only synchronization, or machine-local Claude credentials.
- Reject if the proxy can continue reading from upstream after downstream cancellation has been observed.
- Reject if implementation adds dependencies or CI workflow files.

## Minimal Acceptance Checklist
- [ ] Evidence-backed tests cover normal SSE forwarding and downstream cancellation in `internal/proxy`.
- [ ] Cancellation closes/unblocks upstream stream work promptly and deterministically.
- [ ] Non-SSE and oversized-body tests still pass.
- [ ] `gofmt` was run on touched Go files.
- [ ] `go test ./internal/proxy` and `go test ./...` pass.
