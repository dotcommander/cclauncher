package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/dotcommander/cclauncher/internal/utils"
	"gopkg.in/yaml.v3"
)

//go:embed default-config.yaml
var defaultConfigYAML []byte

// Version is set at build time via -ldflags
var Version = "dev"

// Config is the main configuration for CCL
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

// HasAuth reports whether the provider has any authentication credential configured.
func (p Provider) HasAuth() bool {
	return p.AuthToken != "" || p.APIKey != "" || p.OAuthToken != ""
}

// interpolated returns a copy of the Provider with all ${VAR} templates resolved.
func (p Provider) interpolated() Provider {
	p.BaseURL, _ = utils.InterpolateEnvVars(p.BaseURL, false)
	p.AuthToken, _ = utils.InterpolateEnvVars(p.AuthToken, false)
	p.OAuthToken, _ = utils.InterpolateEnvVars(p.OAuthToken, false)
	p.APIKey, _ = utils.InterpolateEnvVars(p.APIKey, false)
	p.Model, _ = utils.InterpolateEnvVars(p.Model, false)
	p.SmallFastModel, _ = utils.InterpolateEnvVars(p.SmallFastModel, false)
	return p
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

// ConfigPath returns the path to the config file for this loader
func (l *Loader) ConfigPath() string {
	return l.configFile
}

// EnsureExists creates the config directory and default config file if they don't exist.
// Returns true if a new config file was created.
func EnsureExists() (bool, error) {
	return NewLoader().EnsureExists()
}

// EnsureExists creates the config directory and copies the example config if needed.
// Returns true if a new config file was created.
func (l *Loader) EnsureExists() (bool, error) {
	if err := os.MkdirAll(l.homeDir, 0700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	if _, err := os.Stat(l.configFile); err == nil {
		return false, nil
	}

	if err := os.WriteFile(l.configFile, defaultConfigYAML, 0600); err != nil {
		return false, fmt.Errorf("write default config: %w", err)
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

// DefaultConfig returns the default configuration with optimization settings.
// Providers are loaded from the YAML template via EnsureExists or Load.
func DefaultConfig() *Config {
	return &Config{
		Providers:    make(map[string]Provider),
		CLI:          CLIConfig{DefaultProvider: "synthetic"},
		Optimization: defaultOptimization(),
	}
}

