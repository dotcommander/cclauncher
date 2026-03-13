package config

// GetProvider retrieves a provider configuration by name from the config.
func GetProvider(cfg *Config, name string) (Provider, bool) {
	p, ok := cfg.Providers[name]
	return p, ok
}
