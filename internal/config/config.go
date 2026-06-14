package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PhaseConfig represents a single phase in the workflow
type PhaseConfig struct {
	Name         string   `yaml:"name"`
	Prompt       string   `yaml:"prompt"`
	NextPhase    string   `yaml:"next_phase,omitempty"`
	ReadOnly     bool     `yaml:"read_only"`
	AllowedTools []string `yaml:"allowed_tools"`
}

// GetAllowedTools returns the list of tools allowed in this phase
func (p *PhaseConfig) GetAllowedTools() []string {
	return p.AllowedTools
}

// SecurityPolicy holds security settings for tools
type SecurityPolicy struct {
	CommandExecution CommandExecutionPolicy `yaml:"command_execution"`
}

// CommandExecutionPolicy configures command execution security
type CommandExecutionPolicy struct {
	Enabled              bool             `yaml:"enabled"`
	AllowedCommands      []AllowedCommand `yaml:"allowed_commands"`
	DefaultTimeoutSec    int              `yaml:"default_timeout_seconds"`
	MaxTimeoutSec        int              `yaml:"max_timeout_seconds"`
	MaxConcurrent        int              `yaml:"max_concurrent_commands"`
	BlockedDirectories   []string         `yaml:"blocked_directories"`
	MaxOutputSizeBytes   int              `yaml:"max_output_size_bytes"`
}

// AllowedCommand defines a whitelisted command with restrictions
type AllowedCommand struct {
	Prefix        string   `yaml:"prefix"`
	Subcommands   []string `yaml:"subcommands,omitempty"`
	BlockedArgs   []string `yaml:"blocked_args,omitempty"`
	AllowedPhases []string `yaml:"allowed_phases,omitempty"`
}

// ModelProfile holds per-model overrides selectable via --model flag
type ModelProfile struct {
	Model       string  `yaml:"model,omitempty"`
	BaseURL     string  `yaml:"base_url,omitempty"`
	APIKey      string  `yaml:"api_key,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
	Temperature float32 `yaml:"temperature,omitempty"`
}

// Config holds the application configuration
type Config struct {
	APIKey             string                    `yaml:"api_key"`
	BaseURL            string                    `yaml:"base_url"`
	InsecureSkipVerify bool                      `yaml:"insecure_skip_verify"`
	Model              string                    `yaml:"model"`
	Temperature        float32                   `yaml:"temperature"`
	MaxTokens          int                       `yaml:"max_tokens"`
	SystemPrompt       string                    `yaml:"system_prompt"`
	Phases             []PhaseConfig             `yaml:"phases,omitempty"`
	InitialPhase       string                    `yaml:"initial_phase,omitempty"`
	Security           SecurityPolicy            `yaml:"security,omitempty"`
	Models             map[string]ModelProfile   `yaml:"models,omitempty"`
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	// Initialize with defaults
	cfg := &Config{
		BaseURL:            DefaultBaseURL,
		InsecureSkipVerify: DefaultInsecureSkipVerify,
		Model:              DefaultModel,
		Temperature:        DefaultTemperature,
		MaxTokens:          DefaultMaxTokens,
		SystemPrompt:       DefaultSystemPrompt,
		Phases:             DefaultPhases,
		InitialPhase:       DefaultInitialPhase,
		Security:           DefaultSecurityPolicy(),
	}

	// Try to load from config file
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	// Environment variables take precedence
	
	// API Key - can be empty for custom endpoints
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}
	
	// Base URL - support both OPENAI_BASE_URL and OPENAI_API_BASE
	// OPENAI_BASE_URL takes precedence
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	} else if baseURL := os.Getenv("OPENAI_API_BASE"); baseURL != "" {
		cfg.BaseURL = baseURL
	}
	
	// Insecure TLS skip verify
	if skipVerify := os.Getenv("OPENAI_INSECURE_SKIP_VERIFY"); skipVerify != "" {
		cfg.InsecureSkipVerify = skipVerify == "true" || skipVerify == "1"
	}

	// Normalize base URL (auto-append /v1 if missing)
	cfg.BaseURL = normalizeBaseURL(cfg.BaseURL)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// API key is only required for default OpenAI endpoint
	// Custom endpoints (LocalAI, etc.) often don't need authentication
	if c.APIKey == "" && c.BaseURL == DefaultBaseURL {
		return fmt.Errorf("OPENAI_API_KEY is required when using default OpenAI endpoint (set via environment variable or config file)")
	}

	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got: %f", c.Temperature)
	}

	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive, got: %d", c.MaxTokens)
	}

	if c.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	// Validate base URL format
	if err := validateBaseURL(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}

	return nil
}

// ApplyProfile overlays a named model profile's non-zero fields onto the config.
func (c *Config) ApplyProfile(name string) error {
	p, ok := c.Models[name]
	if !ok {
		return fmt.Errorf("model profile %q not found in config", name)
	}
	if p.Model != ""         { c.Model = p.Model }
	if p.BaseURL != ""       { c.BaseURL = p.BaseURL }
	if p.APIKey != ""        { c.APIKey = p.APIKey }
	if p.MaxTokens > 0   { c.MaxTokens = p.MaxTokens }
	if p.Temperature > 0 { c.Temperature = p.Temperature }
	return nil
}

// NormalizeBaseURL ensures the base URL ends with /v1.
func NormalizeBaseURL(baseURL string) string {
	return normalizeBaseURL(baseURL)
}

// normalizeBaseURL ensures the base URL ends with /v1
func normalizeBaseURL(baseURL string) string {
	// Remove trailing slashes first
	baseURL = strings.TrimRight(baseURL, "/")
	
	// Check if it already ends with /v1
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL = baseURL + "/v1"
	}
	
	return baseURL
}

// validateBaseURL performs strict URL validation
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base_url cannot be empty")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Must use http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Must have a host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// IsCustomEndpoint checks if using non-default OpenAI endpoint
func (c *Config) IsCustomEndpoint() bool {
	return c.BaseURL != DefaultBaseURL
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ConfigDir, ConfigFileName)
}

// GetConfigDir returns the path to the config directory
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ConfigDir)
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	dir := GetConfigDir()
	return os.MkdirAll(dir, 0755)
}
