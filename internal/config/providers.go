package config

import "github.com/ccg/ccg/internal/utils"

// ProviderConfig holds the configuration for a provider
type ProviderConfig struct {
	BaseURL        string
	AuthToken      string
	APIKey         string
	OAuthToken     string
	Model          string
	SmallFastModel string
}

// ProviderRegistryInterface defines the contract for provider registry operations
type ProviderRegistryInterface interface {
	Get(name string) (ProviderConfig, bool)
	GetEnvVars(provider string) map[string]string
}

// ProviderRegistry manages provider configurations with dependency injection
type ProviderRegistry struct {
	providers map[string]ProviderConfig
}

// Ensure ProviderRegistry implements ProviderRegistryInterface
var _ ProviderRegistryInterface = (*ProviderRegistry)(nil)

// NewProviderRegistryFromConfig creates a registry from a loaded Config.
// The Config should already have interpolated environment variables (via Load/Init).
func NewProviderRegistryFromConfig(cfg *Config) *ProviderRegistry {
	providers := make(map[string]ProviderConfig, len(cfg.Providers))
	for name, p := range cfg.Providers {
		providers[name] = p.ToProviderConfig()
	}
	return &ProviderRegistry{providers: providers}
}

// NewProviderRegistry creates a registry from DefaultConfig with env vars
// resolved at call time. Prefer NewProviderRegistryFromConfig for production use.
func NewProviderRegistry() *ProviderRegistry {
	cfg := DefaultConfig()
	// Resolve env vars in provider fields directly since DefaultConfig
	// uses ${VAR} template syntax that only resolves through Load/Init
	providers := make(map[string]ProviderConfig, len(cfg.Providers))
	for name, p := range cfg.Providers {
		pc := p.ToProviderConfig()
		pc.BaseURL = resolveEnvTemplate(pc.BaseURL)
		pc.AuthToken = resolveEnvTemplate(pc.AuthToken)
		pc.APIKey = resolveEnvTemplate(pc.APIKey)
		pc.OAuthToken = resolveEnvTemplate(pc.OAuthToken)
		pc.Model = resolveEnvTemplate(pc.Model)
		pc.SmallFastModel = resolveEnvTemplate(pc.SmallFastModel)
		providers[name] = pc
	}
	return &ProviderRegistry{providers: providers}
}

// resolveEnvTemplate resolves a single ${VAR} or ${VAR:-default} template string.
func resolveEnvTemplate(s string) string {
	if s == "" {
		return ""
	}
	result, _ := utils.InterpolateEnvVars(s, false)
	return result
}

// Get retrieves a provider configuration by name
func (r *ProviderRegistry) Get(name string) (ProviderConfig, bool) {
	cfg, ok := r.providers[name]
	return cfg, ok
}

// GetEnvVars returns environment variables for a given provider
func (r *ProviderRegistry) GetEnvVars(provider string) map[string]string {
	config, ok := r.Get(provider)
	if !ok {
		return nil
	}

	envVars := make(map[string]string)

	// Map provider config fields to environment variables
	envMappings := []struct {
		key   string
		value string
	}{
		{"ANTHROPIC_BASE_URL", config.BaseURL},
		{"ANTHROPIC_AUTH_TOKEN", config.AuthToken},
		{"ANTHROPIC_API_KEY", config.APIKey},
		{"CLAUDE_CODE_OAUTH_TOKEN", config.OAuthToken},
		{"ANTHROPIC_MODEL", config.Model},
		{"ANTHROPIC_SMALL_FAST_MODEL", config.SmallFastModel},
		{"ANTHROPIC_DEFAULT_OPUS_MODEL", config.Model},
		{"ANTHROPIC_DEFAULT_SONNET_MODEL", config.Model},
		{"ANTHROPIC_DEFAULT_HAIKU_MODEL", config.SmallFastModel},
		{"CLAUDE_CODE_SUBAGENT_MODEL", config.SmallFastModel},
	}

	for _, m := range envMappings {
		if m.value != "" {
			envVars[m.key] = m.value
		}
	}

	return envVars
}

