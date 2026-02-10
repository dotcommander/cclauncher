package config

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

// Get retrieves a provider configuration by name
func (r *ProviderRegistry) Get(name string) (ProviderConfig, bool) {
	cfg, ok := r.providers[name]
	return cfg, ok
}


