// Package actions holds CCL business logic that is independent of command
// parsing: pure, testable functions over config types.
package actions

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
)

// DoctorNetTimeout bounds each reachability HEAD probe (--check-net).
const DoctorNetTimeout = 5 * time.Second

// Status is the outcome severity of a single diagnostic check.
type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

// CheckResult is the outcome of one diagnostic check on one provider.
// Fields are derived from logic outcomes only — never from credential values.
type CheckResult struct {
	Provider string
	Check    string
	Status   Status
	Reason   string
}

// RunChecks executes the static (offline) checks for one provider in stable
// order: auth, fields, baseurl.
func RunChecks(name string, p config.Provider) []CheckResult {
	return []CheckResult{
		checkAuth(name, p),
		checkFields(name, p),
		checkBaseURL(name, p),
	}
}

func checkAuth(name string, p config.Provider) CheckResult {
	r := CheckResult{Provider: name, Check: "auth"}
	switch {
	case !p.RequiresAuth():
		r.Status = StatusWarn
		r.Reason = "auth not required (authRequired: false)"
	case p.HasAuth():
		r.Status = StatusPass
		r.Reason = "credential present"
	default:
		r.Status = StatusWarn
		r.Reason = "auth required but no token configured — skipping (no credential)"
	}
	return r
}

func checkFields(name string, p config.Provider) CheckResult {
	r := CheckResult{Provider: name, Check: "fields"}
	if err := config.ValidateProviderEndpoint(p); err != nil {
		r.Status, r.Reason = StatusFail, err.Error()
	} else {
		r.Status, r.Reason = StatusPass, "baseUrl and model valid"
	}
	return r
}

func checkBaseURL(name string, p config.Provider) CheckResult {
	r := CheckResult{Provider: name, Check: "baseurl"}
	baseURL := strings.TrimRight(p.BaseURL, "/")
	switch {
	case strings.HasSuffix(baseURL, "/v1"):
		r.Status = StatusWarn
		r.Reason = "baseUrl ends in /v1 — proxy appends /v1/messages, yielding /v1/v1/messages"
	case strings.Contains(p.BaseURL, "/api/paas/v4"):
		r.Status = StatusWarn
		r.Reason = "baseUrl uses /api/paas/v4 — use /api/anthropic for Z.ai (see docs)"
	default:
		r.Status = StatusPass
		r.Reason = "baseUrl format OK"
	}
	return r
}

// ProbeReachability sends an unauthenticated HEAD to the provider's BaseURL and
// classifies reachability. Any HTTP response (incl. 4xx/5xx) means the host is
// live → PASS. Timeout, DNS, or transport error → WARN. NEVER attach auth
// credentials to this request.
func ProbeReachability(ctx context.Context, client *http.Client, name string, p config.Provider) CheckResult {
	r := CheckResult{Provider: name, Check: "net"}
	if client == nil {
		r.Status, r.Reason = StatusWarn, "cannot probe: HTTP client is not configured"
		return r
	}
	ctx, cancel := context.WithTimeout(ctx, DoctorNetTimeout)
	defer cancel()
	if p.BaseURL == "" {
		r.Status = StatusWarn
		r.Reason = "cannot probe: baseUrl is empty"
		return r
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, p.BaseURL, nil)
	if err != nil {
		r.Status = StatusWarn
		r.Reason = fmt.Sprintf("unreachable: %v", err)
		return r
	}
	resp, err := client.Do(req)
	if err != nil {
		r.Status = StatusWarn
		r.Reason = fmt.Sprintf("unreachable: %v", err)
		return r
	}
	defer resp.Body.Close()
	r.Status = StatusPass
	r.Reason = fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)
	return r
}
