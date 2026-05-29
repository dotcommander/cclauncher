package config

import (
	"context"
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
	AuthRequired   *bool             `json:"authRequired,omitempty" yaml:"authRequired,omitempty"`
	BaseURL        string            `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	Model          string            `json:"model,omitempty" yaml:"model,omitempty"`
	SmallFastModel string            `json:"smallFastModel,omitempty" yaml:"smallFastModel,omitempty"`
	Transformer    TransformerConfig `json:"transformer,omitempty" yaml:"transformer,omitempty"`
}

// HasAuth reports whether the provider has any authentication credential configured.
func (p Provider) HasAuth() bool {
	return p.AuthToken != "" || p.APIKey != "" || p.OAuthToken != ""
}

// RequiresAuth reports whether a missing credential should block launch.
// The default is true so custom providers fail closed unless config opts out.
func (p Provider) RequiresAuth() bool {
	return p.AuthRequired == nil || *p.AuthRequired
}

// AuthCredential returns the preferred auth token for the Anthropic-compatible
// endpoint: AuthToken takes precedence over APIKey. OAuth is handled separately.
func (p Provider) AuthCredential() string {
	if p.AuthToken != "" {
		return p.AuthToken
	}
	return p.APIKey
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

// HasRules reports whether the transformer has any rules configured.
func (t TransformerConfig) HasRules() bool {
	return len(t.Use) > 0 || len(t.Rules) > 0
}

// TransformRule defines a conditional transformation rule
type TransformRule struct {
	ModelPattern   string `json:"modelPattern,omitempty" yaml:"modelPattern,omitempty"`
	MessagePattern string `json:"messagePattern,omitempty" yaml:"messagePattern,omitempty"`
	TokenRange     struct {
		Min int `json:"min,omitempty" yaml:"min,omitempty"`
		Max int `json:"max,omitempty" yaml:"max,omitempty"`
	} `json:"tokenRange,omitempty" yaml:"tokenRange,omitempty"`

	SetModel       string            `json:"setModel,omitempty" yaml:"setModel,omitempty"`
	SetMaxTokens   int               `json:"setMaxTokens,omitempty" yaml:"setMaxTokens,omitempty"`
	SetTemperature float64           `json:"setTemperature,omitempty" yaml:"setTemperature,omitempty"`
	AddHeaders     map[string]string `json:"addHeaders,omitempty" yaml:"addHeaders,omitempty"`
	ModifyBody     map[string]any    `json:"modifyBody,omitempty" yaml:"modifyBody,omitempty"`

	ResponseFormat *ResponseFormat `json:"responseFormat,omitempty" yaml:"responseFormat,omitempty"`
}

// ResponseFormat defines structured output configuration
type ResponseFormat struct {
	Type       string         `json:"type" yaml:"type"`
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

// EnsureExists is a package-level convenience that calls (*Loader).EnsureExists
// with the default loader.
func EnsureExists() (bool, error) {
	return NewLoader().EnsureExists()
}

// EnsureExists creates the config directory and writes the default config
// file if one doesn't already exist. Returns true if a new file was created.
func (l *Loader) EnsureExists() (bool, error) {
	if err := os.MkdirAll(l.homeDir, 0700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	if _, err := os.Stat(l.configFile); err == nil {
		return false, nil
	}

	if err := writeFileAtomic(l.configFile, defaultConfigYAML, 0600); err != nil {
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
	mergeDefaultProviders(&cfg)
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

	if err := writeFileAtomic(l.configFile, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// DefaultConfig returns the default configuration with optimization settings.
// Providers are loaded from the YAML template via EnsureExists or Load.
func DefaultConfig() *Config {
	return &Config{
		Providers:    make(map[string]Provider),
		CLI:          CLIConfig{DefaultProvider: "zai"},
		Optimization: defaultOptimization(),
	}
}

type contextKey string

const cfgContextKey contextKey = "config"

// StoreInContext returns a new context with the config stored.
func StoreInContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, cfgContextKey, cfg)
}

// FromContext extracts the config from a context. Returns nil if not present.
func FromContext(ctx context.Context) *Config {
	cfg, _ := ctx.Value(cfgContextKey).(*Config)
	return cfg
}

// SetDefaultProvider updates only the cli.defaultProvider field in the config
// file on disk. It uses yaml.Node to preserve comments and env-var templates.
func SetDefaultProvider(name string) error {
	path := GetConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected YAML structure in config")
	}

	root := doc.Content[0]
	cliNode := findMapValue(root, "cli")
	if cliNode == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "cli"},
			&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "defaultProvider"},
				{Kind: yaml.ScalarNode, Value: name},
			}},
		)
	} else {
		dpNode := findMapValue(cliNode, "defaultProvider")
		if dpNode == nil {
			cliNode.Content = append(cliNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "defaultProvider"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: name},
			)
		} else {
			dpNode.Value = name
		}
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := writeFileAtomic(path, out, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// findMapValue returns the value node for a given key in a YAML mapping node.
func findMapValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}
