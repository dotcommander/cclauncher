package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/dotcommander/cclauncher/internal/actions"
	"github.com/dotcommander/cclauncher/internal/config"
)

func localProviderClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &http.Client{
		Transport: transport,
		Timeout:   actions.DoctorNetTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// reachableProviderChoices probes only eligible loopback endpoints, without
// credentials. Any HTTP response proves reachability, not model availability.
func reachableProviderChoices(ctx context.Context, cfg *config.Config, client *http.Client) ([]ProviderChoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	choices := providerChoices(cfg)
	local := make([]string, 0, len(choices))
	for _, choice := range choices {
		if isLoopbackEndpoint(cfg.Providers[choice.Name].BaseURL) {
			local = append(local, choice.Name)
		}
	}
	if len(local) == 0 {
		return choices, nil
	}
	if cfg.Doctor.ProbeConcurrency <= 0 {
		return nil, fmt.Errorf("doctor.probeConcurrency must be positive")
	}
	results, err := probeProviders(ctx, client, local, cfg.Providers, cfg.Doctor.ProbeConcurrency)
	if err != nil {
		return nil, err
	}
	unreachable := make(map[string]bool, len(results))
	for _, result := range results {
		if result.Status != actions.StatusPass {
			unreachable[result.Provider] = true
		}
	}
	return slices.DeleteFunc(choices, func(choice ProviderChoice) bool {
		return unreachable[choice.Name]
	}), nil
}

func isLoopbackEndpoint(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
