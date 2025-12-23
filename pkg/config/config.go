package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo          string        `mapstructure:"repo" yaml:"repo,omitempty"`
	Agent         string        `mapstructure:"agent" yaml:"agent,omitempty"`
	Schema        string        `mapstructure:"schema" yaml:"schema,omitempty"`
	CVPath        string        `mapstructure:"cv_path" yaml:"cv_path,omitempty"`
	ReferencePath string        `mapstructure:"reference_path" yaml:"reference_path,omitempty"`
	Project       ProjectConfig `mapstructure:"project" yaml:"project,omitempty"`
}

// AgentCLI returns the CLI agent name derived from the agent setting
func (c *Config) AgentCLI() string {
	if strings.HasPrefix(c.Agent, "gemini") {
		return "gemini"
	}
	return "claude"
}

// ProjectConfig contains user-facing project configuration
// Only Number and Owner are saved to config file; IDs are cached separately
type ProjectConfig struct {
	Number int    `mapstructure:"number" yaml:"number,omitempty"`
	Owner  string `mapstructure:"owner" yaml:"owner,omitempty"`
	// Internal fields - not saved to user config, loaded from cache
	ID       string            `mapstructure:"id" yaml:"-"`
	Title    string            `mapstructure:"title" yaml:"-"`
	Fields   FieldIDs          `mapstructure:"fields" yaml:"-"`
	Statuses map[string]string `mapstructure:"statuses" yaml:"-"`
}

type FieldIDs struct {
	Status      string `mapstructure:"status" yaml:"status,omitempty"`
	Company     string `mapstructure:"company" yaml:"company,omitempty"`
	Deadline    string `mapstructure:"deadline" yaml:"deadline,omitempty"`
	AppliedDate string `mapstructure:"applied_date" yaml:"applied_date,omitempty"`
}

var (
	configFile = ".cvx-config.yaml"
	v          *viper.Viper
)

func init() {
	v = viper.New()
	v.SetConfigFile(configFile)

	// Defaults
	v.SetDefault("agent", "claude-cli")

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

func Load() (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

func Get(key string) (string, error) {
	switch key {
	case "repo", "agent", "schema", "cv_path", "reference_path":
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
		cfg.Agent = value
	case "schema":
		cfg.Schema = value
	case "cv_path":
		cfg.CVPath = value
	case "reference_path":
		cfg.ReferencePath = value
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, agent, schema, cv_path, reference_path)", key)
	}

	v.Set(key, value) // keep viper in sync
	return writeConfig(cfg)
}

func save() error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}
	return writeConfig(cfg)
}

func writeConfig(cfg *Config) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(configFile, buf.Bytes(), 0644)
}

func All() (map[string]string, error) {
	return map[string]string{
		"repo":           v.GetString("repo"),
		"agent":          v.GetString("agent"),
		"schema":         v.GetString("schema"),
		"cv_path":        v.GetString("cv_path"),
		"reference_path": v.GetString("reference_path"),
	}, nil
}

// Save saves the full config
func Save(c *Config) error {
	return writeConfig(c)
}

// SaveProject saves project configuration
// User-facing fields (number, owner) go to config file
// Internal IDs go to cache file in .cvx/cache.yaml
func SaveProject(p ProjectConfig) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}

	// Only save user-facing fields to config
	cfg.Project.Number = p.Number
	cfg.Project.Owner = p.Owner

	// Keep viper in sync for user-facing fields only
	v.Set("project", map[string]interface{}{
		"number": p.Number,
		"owner":  p.Owner,
	})

	// Save internal IDs to cache
	cache := ProjectCache{
		ID:       p.ID,
		Number:   p.Number,
		Title:    p.Title,
		Fields:   p.Fields,
		Statuses: p.Statuses,
	}
	if err := saveProjectCache(cache); err != nil {
		return fmt.Errorf("failed to save project cache: %w", err)
	}

	return writeConfig(cfg)
}

// ProjectCache stores internal project IDs (not user-facing)
type ProjectCache struct {
	ID       string            `yaml:"id"`
	Number   int               `yaml:"number"`
	Title    string            `yaml:"title"`
	Fields   FieldIDs          `yaml:"fields"`
	Statuses map[string]string `yaml:"statuses"`
}

var cacheFile = ".cvx/cache.yaml"

func saveProjectCache(cache ProjectCache) error {
	cacheDir := filepath.Dir(cacheFile)
	os.MkdirAll(cacheDir, 0755)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(map[string]ProjectCache{"project": cache}); err != nil {
		return err
	}
	return os.WriteFile(cacheFile, buf.Bytes(), 0644)
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

// LoadWithCache loads config and merges in cached project IDs
func LoadWithCache() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Try to load cached project IDs
	if cfg.Project.Number > 0 {
		if cache, err := LoadProjectCache(); err == nil && cache.Number == cfg.Project.Number {
			cfg.Project.ID = cache.ID
			cfg.Project.Title = cache.Title
			cfg.Project.Fields = cache.Fields
			cfg.Project.Statuses = cache.Statuses
		}
	}

	return cfg, nil
}

// ResetForTest resets viper for testing (only use in tests)
func ResetForTest(testPath string) {
	configFile = testPath + "/.cvx-config.yaml"
	cacheFile = testPath + "/.cvx/cache.yaml"
	v = viper.New()
	v.SetConfigFile(configFile)
	v.SetDefault("agent", "claude-cli")
}
