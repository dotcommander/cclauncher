package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccg/ccg/internal/utils"
	"gopkg.in/yaml.v3"
)

// Config is the main configuration for CCG
type Config struct {
	Providers    map[string]Provider `json:"providers" yaml:"providers"`
	Router       RouterConfig        `json:"router,omitempty" yaml:"router,omitempty"`
	CLI          CLIConfig           `json:"cli,omitempty" yaml:"cli,omitempty"`
	Optimization OptimizationConfig  `json:"optimization,omitempty" yaml:"optimization,omitempty"`
}

// RouterConfig defines model routing configuration
type RouterConfig struct {
	Default              string `json:"default,omitempty" yaml:"default,omitempty"`
	Background           string `json:"background,omitempty" yaml:"background,omitempty"`
	LongContext          string `json:"longContext,omitempty" yaml:"longContext,omitempty"`
	Think                string `json:"think,omitempty" yaml:"think,omitempty"`
	WebSearch            string `json:"webSearch,omitempty" yaml:"webSearch,omitempty"`
	LongContextThreshold int    `json:"longContextThreshold,omitempty" yaml:"longContextThreshold,omitempty"`
}

// Provider defines an LLM provider configuration
type Provider struct {
	APIKey         string            `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	AuthToken      string            `json:"authToken,omitempty" yaml:"authToken,omitempty"`
	OAuthToken     string            `json:"oauthToken,omitempty" yaml:"oauthToken,omitempty"`
	BaseURL        string            `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	Model          string            `json:"model,omitempty" yaml:"model,omitempty"`
	SmallFastModel string            `json:"smallFastModel,omitempty" yaml:"smallFastModel,omitempty"`
	Transformer    TransformerConfig `json:"transformer,omitempty" yaml:"transformer,omitempty"`
}

// ToProviderConfig converts a config Provider to a runtime ProviderConfig
func (p Provider) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		BaseURL:        p.BaseURL,
		AuthToken:      p.AuthToken,
		APIKey:         p.APIKey,
		OAuthToken:     p.OAuthToken,
		Model:          p.Model,
		SmallFastModel: p.SmallFastModel,
	}
}

// OptimizationConfig defines Claude Code optimization and performance settings
type OptimizationConfig struct {
	DisableNonessentialTraffic bool `json:"disableNonessentialTraffic,omitempty" yaml:"disableNonessentialTraffic,omitempty"`
	DisableAutoupdater         bool `json:"disableAutoupdater,omitempty" yaml:"disableAutoupdater,omitempty"`
	DisableTelemetry           bool `json:"disableTelemetry,omitempty" yaml:"disableTelemetry,omitempty"`
	DisableErrorReporting      bool `json:"disableErrorReporting,omitempty" yaml:"disableErrorReporting,omitempty"`
	DisableCostWarnings        bool `json:"disableCostWarnings,omitempty" yaml:"disableCostWarnings,omitempty"`
	APITimeoutMs               int  `json:"apiTimeoutMs,omitempty" yaml:"apiTimeoutMs,omitempty"`
	MaxOutputTokens            int  `json:"maxOutputTokens,omitempty" yaml:"maxOutputTokens,omitempty"`
	NodeMaxOldSpaceSize        int  `json:"nodeMaxOldSpaceSize,omitempty" yaml:"nodeMaxOldSpaceSize,omitempty"`
}

// TransformerConfig defines request/response transformation settings
type TransformerConfig struct {
	Use   []string        `json:"use,omitempty" yaml:"use,omitempty"`
	Rules []TransformRule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// TransformRule defines a conditional transformation rule
type TransformRule struct {
	ModelPattern   string `json:"modelPattern,omitempty" yaml:"modelPattern,omitempty"`
	MessagePattern string `json:"messagePattern,omitempty" yaml:"messagePattern,omitempty"`
	TokenRange     struct {
		Min int `json:"min,omitempty" yaml:"min,omitempty"`
		Max int `json:"max,omitempty" yaml:"max,omitempty"`
	} `json:"tokenRange,omitempty" yaml:"tokenRange,omitempty"`

	SetModel       string                 `json:"setModel,omitempty" yaml:"setModel,omitempty"`
	SetMaxTokens   int                    `json:"setMaxTokens,omitempty" yaml:"setMaxTokens,omitempty"`
	SetTemperature float64                `json:"setTemperature,omitempty" yaml:"setTemperature,omitempty"`
	AddHeaders     map[string]string      `json:"addHeaders,omitempty" yaml:"addHeaders,omitempty"`
	ModifyBody     map[string]any `json:"modifyBody,omitempty" yaml:"modifyBody,omitempty"`

	ResponseFormat *ResponseFormat `json:"responseFormat,omitempty" yaml:"responseFormat,omitempty"`
}

// ResponseFormat defines structured output configuration
type ResponseFormat struct {
	Type       string                 `json:"type" yaml:"type"`
	JSONSchema map[string]any `json:"jsonSchema,omitempty" yaml:"jsonSchema,omitempty"`
}

// CLIConfig contains CLI-specific configuration
type CLIConfig struct {
	DefaultProvider string `json:"defaultProvider,omitempty" yaml:"defaultProvider,omitempty"`
	Version         string `json:"version,omitempty" yaml:"version,omitempty"`
}

// getDefaultHomeDir returns the default config home directory
func getDefaultHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "cclauncher")
}

// getDefaultConfigFile returns the default config file path
func getDefaultConfigFile() string {
	return filepath.Join(getDefaultHomeDir(), "config.yaml")
}

// Loader handles configuration loading with injectable paths
type Loader struct {
	homeDir    string
	configFile string
}

// NewLoader creates a Loader with default paths
func NewLoader() *Loader {
	return NewLoaderWithPaths(getDefaultHomeDir(), getDefaultConfigFile())
}

// NewLoaderWithPaths creates a Loader with custom paths (for testing)
func NewLoaderWithPaths(homeDir, configFile string) *Loader {
	return &Loader{
		homeDir:    homeDir,
		configFile: configFile,
	}
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return getDefaultConfigFile()
}

// ConfigPath returns the path to the config file for this loader
func (l *Loader) ConfigPath() string {
	return l.configFile
}

// EnsureExists creates the config directory and default config file if they don't exist.
// Returns true if a new config file was created.
func EnsureExists() (bool, error) {
	return NewLoader().EnsureExists()
}

// EnsureExists creates the config directory and default config file if they don't exist.
// Returns true if a new config file was created.
func (l *Loader) EnsureExists() (bool, error) {
	if err := os.MkdirAll(l.homeDir, 0700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	if _, err := os.Stat(l.configFile); err == nil {
		return false, nil // already exists
	}

	cfg := DefaultConfig()
	if err := l.Save(cfg); err != nil {
		return false, fmt.Errorf("save default config: %w", err)
	}

	return true, nil
}

// Init initializes config directory and loads or creates config
func Init() (*Config, error) {
	return NewLoader().Init()
}

// Init initializes config directory and loads or creates config.
// Uses EnsureExists to create defaults if needed, then loads the config.
func (l *Loader) Init() (*Config, error) {
	if _, err := l.EnsureExists(); err != nil {
		return nil, err
	}
	return l.Load()
}

// Load reads and parses the config file (package-level convenience function)
func Load() (*Config, error) {
	return NewLoader().Load()
}

// Load reads and parses the config file
func (l *Loader) Load() (*Config, error) {
	data, err := os.ReadFile(l.configFile)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	dataStr, err := utils.InterpolateEnvVars(string(data), false)
	if err != nil {
		return nil, fmt.Errorf("interpolate environment variables: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(dataStr), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)
	interpolateProviderFields(&cfg)
	overrideFromEnv(&cfg)

	return &cfg, nil
}

// Save writes the config to disk (package-level convenience function)
func Save(cfg *Config) error {
	return NewLoader().Save(cfg)
}

// Save writes the config to disk
func (l *Loader) Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(l.configFile, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// DefaultConfig returns the default configuration with all providers
// and optimization settings. Auth fields use ${VAR} syntax for
// environment variable interpolation when saved to YAML.
func DefaultConfig() *Config {
	return &Config{
		Providers:    defaultProviders(),
		CLI:          CLIConfig{DefaultProvider: "synthetic", Version: "1.0.0"},
		Optimization: defaultOptimization(),
	}
}

func defaultProviders() map[string]Provider {
	synth := func(model string) Provider {
		return Provider{
			BaseURL:        "https://api.synthetic.new/anthropic",
			AuthToken:      "${SYNTHETIC_API_KEY}",
			Model:          model,
			SmallFastModel: model,
		}
	}

	return map[string]Provider{
		"synthetic":   synth("hf:moonshotai/Kimi-K2.5"),
		"qwen":        synth("hf:Qwen/Qwen3-VL-235B-A22B-Instruct"),
		"qwen3-coder": synth("hf:Qwen/Qwen3-Coder-480B-A35B-Instruct"),
		"deepseek":    synth("hf:deepseek-ai/DeepSeek-V3.2"),
		"kimi2":       synth("hf:moonshotai/Kimi-K2.5"),
		"minimax":     synth("hf:MiniMaxAI/MiniMax-M2"),
		"claude": {
			BaseURL: "https://api.anthropic.com",
		},
		"claude2": {
			BaseURL:    "https://api.anthropic.com",
			OAuthToken: "${CLAUDE2_OAUTH_TOKEN}",
		},
		"zai": {
			BaseURL:        "https://api.z.ai/api/anthropic",
			AuthToken:      "${ZAI_API_KEY}",
			Model:          "${ZAI_MODEL:-glm-4.7}",
			SmallFastModel: "${ZAI_SMALL_MODEL:-glm-4.5-air}",
		},
		"llamabarn": {
			BaseURL:        "${LLAMABARN_BASE_URL:-http://localhost:2276/v1}",
			AuthToken:      "${LLAMABARN_API_KEY}",
			Model:          "${LLAMABARN_MODEL:-local}",
			SmallFastModel: "${LLAMABARN_SMALL_MODEL:-local}",
		},
	}
}

func defaultOptimization() OptimizationConfig {
	return OptimizationConfig{
		DisableNonessentialTraffic: true,
		DisableAutoupdater:         true,
		DisableTelemetry:           true,
		DisableErrorReporting:      true,
		DisableCostWarnings:        true,
		APITimeoutMs:               3000000,
		MaxOutputTokens:            200000,
		NodeMaxOldSpaceSize:        8192,
	}
}

func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	// CLI defaults
	if cfg.CLI.DefaultProvider == "" {
		cfg.CLI.DefaultProvider = defaults.CLI.DefaultProvider
	}
	if cfg.CLI.Version == "" {
		cfg.CLI.Version = defaults.CLI.Version
	}

	// Optimization defaults: if entire section is zero-valued, apply all defaults
	if cfg.Optimization == (OptimizationConfig{}) {
		cfg.Optimization = defaults.Optimization
	} else {
		// Fill in zero-valued int fields from defaults
		if cfg.Optimization.APITimeoutMs == 0 {
			cfg.Optimization.APITimeoutMs = defaults.Optimization.APITimeoutMs
		}
		if cfg.Optimization.MaxOutputTokens == 0 {
			cfg.Optimization.MaxOutputTokens = defaults.Optimization.MaxOutputTokens
		}
		if cfg.Optimization.NodeMaxOldSpaceSize == 0 {
			cfg.Optimization.NodeMaxOldSpaceSize = defaults.Optimization.NodeMaxOldSpaceSize
		}
	}

	// Provider defaults: add missing providers and fill missing fields
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]Provider)
	}
	for name, defaultProvider := range defaults.Providers {
		if existing, exists := cfg.Providers[name]; !exists {
			cfg.Providers[name] = defaultProvider
		} else {
			// Merge: fill empty fields from defaults
			if existing.BaseURL == "" {
				existing.BaseURL = defaultProvider.BaseURL
			}
			if existing.AuthToken == "" {
				existing.AuthToken = defaultProvider.AuthToken
			}
			if existing.OAuthToken == "" {
				existing.OAuthToken = defaultProvider.OAuthToken
			}
			if existing.Model == "" {
				existing.Model = defaultProvider.Model
			}
			if existing.SmallFastModel == "" {
				existing.SmallFastModel = defaultProvider.SmallFastModel
			}
			cfg.Providers[name] = existing
		}
	}
}

// interpolateProviderFields resolves any ${VAR} templates in provider fields
// that were filled by applyDefaults after the initial YAML interpolation pass.
func interpolateProviderFields(cfg *Config) {
	for name, p := range cfg.Providers {
		p.BaseURL, _ = utils.InterpolateEnvVars(p.BaseURL, false)
		p.AuthToken, _ = utils.InterpolateEnvVars(p.AuthToken, false)
		p.OAuthToken, _ = utils.InterpolateEnvVars(p.OAuthToken, false)
		p.APIKey, _ = utils.InterpolateEnvVars(p.APIKey, false)
		p.Model, _ = utils.InterpolateEnvVars(p.Model, false)
		p.SmallFastModel, _ = utils.InterpolateEnvVars(p.SmallFastModel, false)
		cfg.Providers[name] = p
	}
}

func overrideFromEnv(cfg *Config) {
	for name, provider := range cfg.Providers {
		envKey := "CCL_" + strings.ToUpper(name) + "_API_KEY"
		if apiKey := os.Getenv(envKey); apiKey != "" {
			provider.APIKey = apiKey
			cfg.Providers[name] = provider
		}
	}
}
