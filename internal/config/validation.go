package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateProviderEndpoint is shared by launch and doctor field checks.
func ValidateProviderEndpoint(p Provider) error {
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return fmt.Errorf("baseUrl must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(p.Model) == "" && !isNativeAnthropic(p) {
		return fmt.Errorf("model must be configured")
	}
	return nil
}
func isNativeAnthropic(p Provider) bool {
	u, err := url.Parse(p.BaseURL)
	return err == nil && strings.EqualFold(u.Hostname(), "api.anthropic.com")
}

// ValidateProvider rejects unsupported transformations before launching Claude.
func ValidateProvider(p Provider) error {
	if err := ValidateProviderEndpoint(p); err != nil {
		return err
	}
	resolved, err := ResolveTransformerPresets(p.Transformer)
	if err != nil {
		return err
	}
	for i, r := range resolved.Rules {
		if r.ResponseFormat != nil {
			return fmt.Errorf("rule %d: responseFormat is unsupported", i+1)
		}
	}
	if p.Transformer.HasRules() && isNativeAnthropic(p) && p.AuthCredential() == "" {
		return fmt.Errorf("transformed native Anthropic launches require a configured upstream key")
	}
	return nil
}
