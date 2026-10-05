package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed default-config.yaml
var defaultConfigYAML []byte

// Version is set at build time via -ldflags
var Version = "dev"

// Config is the main configuration for CCL
type Config struct {
	Doctor       DoctorConfig        `json:"doctor,omitempty" yaml:"doctor,omitempty"`
	Providers    map[string]Provider `json:"providers" yaml:"providers"`
	Router       RouterConfig        `json:"router,omitempty" yaml:"router,omitempty"`
	CLI          CLIConfig           `json:"cli,omitempty" yaml:"cli,omitempty"`
	Optimization OptimizationConfig  `json:"optimization,omitempty" yaml:"optimization,omitempty"`
}

type DoctorConfig struct {
	ProbeConcurrency int `json:"probeConcurrency" yaml:"probeConcurrency"`
}

// RouterConfig defines model routing configuration
type RouterConfig struct {
	Default              string `json:"default,omitempty" yaml:"default"`
	Background           string `json:"background,omitempty" yaml:"background"`
	LongContext          string `json:"longContext,omitempty" yaml:"longContext"`
	Think                string `json:"think,omitempty" yaml:"think"`
	WebSearch            string `json:"webSearch,omitempty" yaml:"webSearch"`
	LongContextThreshold int    `json:"longContextThreshold,omitempty" yaml:"longContextThreshold"`
}

// Provider defines an LLM provider configuration
type Provider struct {
	APIKey         string            `json:"apiKey,omitempty" yaml:"apiKey"`
	AuthToken      string            `json:"authToken,omitempty" yaml:"authToken"`
	OAuthToken     string            `json:"oauthToken,omitempty" yaml:"oauthToken"`
	AuthRequired   *bool             `json:"authRequired,omitempty" yaml:"authRequired,omitempty"`
	BaseURL        string            `json:"baseUrl,omitempty" yaml:"baseUrl"`
	Model          string            `json:"model,omitempty" yaml:"model"`
	SmallFastModel string            `json:"smallFastModel,omitempty" yaml:"smallFastModel"`
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

// OptimizationConfig defines Claude Code optimization and performance settings
type OptimizationConfig struct {
	DisableNonessentialTraffic bool `json:"disableNonessentialTraffic,omitempty" yaml:"disableNonessentialTraffic"`
	DisableAutoupdater         bool `json:"disableAutoupdater,omitempty" yaml:"disableAutoupdater"`
	DisableTelemetry           bool `json:"disableTelemetry,omitempty" yaml:"disableTelemetry"`
	DisableErrorReporting      bool `json:"disableErrorReporting,omitempty" yaml:"disableErrorReporting"`
	DisableCostWarnings        bool `json:"disableCostWarnings,omitempty" yaml:"disableCostWarnings"`
	APITimeoutMs               int  `json:"apiTimeoutMs,omitempty" yaml:"apiTimeoutMs"`
	MaxOutputTokens            int  `json:"maxOutputTokens,omitempty" yaml:"maxOutputTokens"`
	NodeMaxOldSpaceSize        int  `json:"nodeMaxOldSpaceSize,omitempty" yaml:"nodeMaxOldSpaceSize"`
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
	ModelPattern   string `json:"modelPattern,omitempty" yaml:"modelPattern"`
	MessagePattern string `json:"messagePattern,omitempty" yaml:"messagePattern"`
	TokenRange     struct {
		Min int `json:"min,omitempty" yaml:"min"`
		Max int `json:"max,omitempty" yaml:"max"`
	} `json:"tokenRange,omitempty" yaml:"tokenRange,omitempty"`

	SetModel       string            `json:"setModel,omitempty" yaml:"setModel"`
	SetMaxTokens   int               `json:"setMaxTokens,omitempty" yaml:"setMaxTokens"`
	SetTemperature *float64          `json:"setTemperature,omitempty" yaml:"setTemperature,omitempty"`
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
	DefaultProvider string `json:"defaultProvider,omitempty" yaml:"defaultProvider"`
}

// Loader handles configuration loading with injectable paths
type Loader struct {
	homeDir    string
	initErr    error
	configFile string
}

// NewLoader creates a Loader with default paths
func NewLoader() *Loader {
	home, err := defaultHomeDir()
	l := NewLoaderWithPaths(home, filepath.Join(home, "config.yaml"))
	l.initErr = err
	return l
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
	if l.initErr != nil {
		return false, l.initErr
	}
	if _, err := os.Lstat(l.configFile); err == nil {
		if info, err := os.Stat(l.configFile); err != nil {
			return false, fmt.Errorf("stat existing config: %w", err)
		} else if !info.Mode().IsRegular() {
			return false, fmt.Errorf("existing config must be a regular file")
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat config: %w", err)
	}
	if err := os.MkdirAll(l.homeDir, 0700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
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
	if l.initErr != nil {
		return nil, l.initErr
	}
	data, err := os.ReadFile(l.configFile)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return parseConfig(data)

}

// Save writes the config to disk (package-level convenience function)
func Save(cfg *Config) error {
	return NewLoader().Save(cfg)
}

// Save writes the config to disk
func (l *Loader) Save(cfg *Config) error {
	if l.initErr != nil {
		return l.initErr
	}
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
	var cfg Config
	_ = yaml.Unmarshal(defaultConfigYAML, &cfg)
	cfg.Providers = make(map[string]Provider)
	return &cfg
}

// SetDefaultProvider updates only the cli.defaultProvider field in the config
// file on disk. It uses yaml.Node to preserve comments and env-var templates.
func SetDefaultProvider(name string) error {
	l := NewLoader()
	if l.initErr != nil {
		return l.initErr
	}
	path := l.ConfigPath()

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
