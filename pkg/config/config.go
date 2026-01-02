package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/xrsl/cvx/pkg/ai"
)

type Config struct {
	Repo           string `mapstructure:"repo" yaml:"repo,omitempty"`
	Agent          string `mapstructure:"agent" yaml:"agent,omitempty"`
	DefaultCLIAgent string `mapstructure:"default_cli_agent" yaml:"default_cli_agent,omitempty"`
	Schema         string `mapstructure:"schema" yaml:"schema,omitempty"`
	CVPath         string `mapstructure:"cv_path" yaml:"cv_path,omitempty"`
	ReferencePath  string `mapstructure:"reference_path" yaml:"reference_path,omitempty"`
	Project        string `mapstructure:"project" yaml:"project,omitempty"` // owner/number format
	CVYAMLPath     string `mapstructure:"cv_yaml_path" yaml:"cv_yaml_path,omitempty"`
	LetterYAMLPath string `mapstructure:"letter_yaml_path" yaml:"letter_yaml_path,omitempty"`
}

// AgentCLI returns the CLI agent name derived from the agent setting
func (c *Config) AgentCLI() string {
	if strings.HasPrefix(c.Agent, "gemini") {
		return "gemini"
	}
	if c.Agent == "claude-code" || strings.HasPrefix(c.Agent, "claude-code:") {
		return "claude"
	}
	if c.Agent == "gemini-cli" || strings.HasPrefix(c.Agent, "gemini-cli:") {
		return "gemini"
	}
	return "claude"
}

// ProjectOwner returns the owner from project string (owner/number)
func (c *Config) ProjectOwner() string {
	parts := strings.Split(c.Project, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	// Fall back to repo owner
	parts = strings.Split(c.Repo, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// ProjectNumber returns the number from project string (owner/number)
func (c *Config) ProjectNumber() int {
	parts := strings.Split(c.Project, "/")
	if len(parts) == 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil {
			return n
		}
	}
	return 0
}

type FieldIDs struct {
	Status      string `mapstructure:"status" yaml:"status,omitempty"`
	Company     string `mapstructure:"company" yaml:"company,omitempty"`
	Deadline    string `mapstructure:"deadline" yaml:"deadline,omitempty"`
	AppliedDate string `mapstructure:"applied_date" yaml:"applied_date,omitempty"`
}

// ProjectCache stores internal project IDs (not user-facing)
type ProjectCache struct {
	ID       string            `yaml:"id"`
	Number   int               `yaml:"number"`
	Title    string            `yaml:"title"`
	Fields   FieldIDs          `yaml:"fields"`
	Statuses map[string]string `yaml:"statuses"`
}

var (
	configFile = ".cvx-config.yaml"
	v          *viper.Viper
)

func init() {
	v = viper.New()
	v.SetConfigFile(configFile)

	// Defaults
	v.SetDefault("agent", "claude-code")

	// Environment variables
	v.SetEnvPrefix("CVX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file (ignore if not exists)
	_ = v.ReadInConfig()
}

func Path() string {
	return configFile
}

// validateAgent checks if the configured agent is valid
// Only CLI agents (claude-code, gemini-cli) are allowed in config files
func validateAgent(agent string) error {
	if agent == "" {
		return nil // Empty is ok, will use default
	}

	// Check if it's a CLI agent
	if !ai.IsAgentCLI(agent) {
		return fmt.Errorf("config contains API agent '%s'. Only CLI agents (claude-code, gemini-cli) allowed in config. Use --call-api-directly for API access", agent)
	}

	return nil
}

func Load() (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate agent setting
	if err := validateAgent(cfg.Agent); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Get(key string) (string, error) {
	switch key {
	case "repo", "agent", "default_cli_agent", "schema", "cv_path", "reference_path", "cv_yaml_path", "letter_yaml_path":
		return v.GetString(key), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}

	switch key {
	case "repo":
		cfg.Repo = value
	case "agent":
		// Validate agent before setting
		if err := validateAgent(value); err != nil {
			return err
		}
		cfg.Agent = value
	case "default_cli_agent":
		cfg.DefaultCLIAgent = value
	case "schema":
		cfg.Schema = value
	case "cv_path":
		cfg.CVPath = value
	case "reference_path":
		cfg.ReferencePath = value
	case "cv_yaml_path":
		cfg.CVYAMLPath = value
	case "letter_yaml_path":
		cfg.LetterYAMLPath = value
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, agent, default_cli_agent, schema, cv_path, reference_path, cv_yaml_path, letter_yaml_path)", key)
	}

	v.Set(key, value) // keep viper in sync
	return writeConfig(cfg)
}

func writeConfig(cfg *Config) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(configFile, buf.Bytes(), 0o644)
}

func All() (map[string]string, error) {
	return map[string]string{
		"repo":              v.GetString("repo"),
		"agent":             v.GetString("agent"),
		"default_cli_agent": v.GetString("default_cli_agent"),
		"schema":            v.GetString("schema"),
		"cv_path":           v.GetString("cv_path"),
		"reference_path":    v.GetString("reference_path"),
		"cv_yaml_path":      v.GetString("cv_yaml_path"),
		"letter_yaml_path":  v.GetString("letter_yaml_path"),
	}, nil
}

// Save saves the full config
func Save(c *Config) error {
	return writeConfig(c)
}

// SaveProject saves project configuration
// Project string (owner/number) goes to config file
// Internal IDs go to cache file in .cvx/cache.yaml
func SaveProject(owner string, number int, cache ProjectCache) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}

	// Save as owner/number format
	cfg.Project = fmt.Sprintf("%s/%d", owner, number)
	v.Set("project", cfg.Project)

	// Save internal IDs to cache
	cache.Number = number
	if err := saveProjectCache(cache); err != nil {
		return fmt.Errorf("failed to save project cache: %w", err)
	}

	return writeConfig(cfg)
}

var cacheFile = ".cvx/cache.yaml"

func saveProjectCache(cache ProjectCache) error {
	cacheDir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(map[string]ProjectCache{"project": cache}); err != nil {
		return err
	}
	return os.WriteFile(cacheFile, buf.Bytes(), 0o644)
}

// LoadProjectCache loads cached project IDs
func LoadProjectCache() (*ProjectCache, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	var cache struct {
		Project ProjectCache `yaml:"project"`
	}
	if err := yaml.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache.Project, nil
}

// LoadWithCache loads config and returns it along with cached project data
func LoadWithCache() (*Config, *ProjectCache, error) {
	cfg, err := Load()
	if err != nil {
		return nil, nil, err
	}

	// Try to load cached project IDs
	projectNum := cfg.ProjectNumber()
	if projectNum > 0 {
		if cache, err := LoadProjectCache(); err == nil && cache.Number == projectNum {
			return cfg, cache, nil
		}
	}

	return cfg, nil, nil
}

// ResetForTest resets viper for testing (only use in tests)
func ResetForTest(testPath string) {
	configFile = testPath + "/.cvx-config.yaml"
	cacheFile = testPath + "/.cvx/cache.yaml"
	v = viper.New()
	v.SetConfigFile(configFile)
	v.SetDefault("agent", "claude-code")
}
