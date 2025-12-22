package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo           string        `mapstructure:"repo" yaml:"repo,omitempty"`
	Model          string        `mapstructure:"model" yaml:"model,omitempty"`
	Schema         string        `mapstructure:"schema" yaml:"schema,omitempty"`
	CVPath         string        `mapstructure:"cv_path" yaml:"cv_path,omitempty"`
	ExperiencePath string        `mapstructure:"experience_path" yaml:"experience_path,omitempty"`
	Project        ProjectConfig `mapstructure:"project" yaml:"project,omitempty"`
}

// Agent returns the CLI agent name derived from the model setting
func (c *Config) Agent() string {
	if strings.HasPrefix(c.Model, "gemini") {
		return "gemini"
	}
	return "claude"
}

type ProjectConfig struct {
	ID       string            `mapstructure:"id" yaml:"id,omitempty"`
	Number   int               `mapstructure:"number" yaml:"number,omitempty"`
	Title    string            `mapstructure:"title" yaml:"title,omitempty"`
	Fields   FieldIDs          `mapstructure:"fields" yaml:"fields,omitempty"`
	Statuses map[string]string `mapstructure:"statuses" yaml:"statuses,omitempty"`
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
	v.SetDefault("model", "claude-cli")

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
	case "repo", "model", "schema", "cv_path", "experience_path":
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
	case "model":
		cfg.Model = value
	case "schema":
		cfg.Schema = value
	case "cv_path":
		cfg.CVPath = value
	case "experience_path":
		cfg.ExperiencePath = value
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, model, schema, cv_path, experience_path)", key)
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
		"repo":            v.GetString("repo"),
		"model":           v.GetString("model"),
		"schema":          v.GetString("schema"),
		"cv_path":         v.GetString("cv_path"),
		"experience_path": v.GetString("experience_path"),
	}, nil
}

// Save saves the full config
func Save(c *Config) error {
	return writeConfig(c)
}

// SaveProject saves project configuration
func SaveProject(p ProjectConfig) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}
	cfg.Project = p

	// Keep viper in sync so subsequent Load() calls include project data
	v.Set("project", map[string]interface{}{
		"id":     p.ID,
		"number": p.Number,
		"title":  p.Title,
		"fields": map[string]string{
			"status":       p.Fields.Status,
			"company":      p.Fields.Company,
			"deadline":     p.Fields.Deadline,
			"applied_date": p.Fields.AppliedDate,
		},
		"statuses": p.Statuses,
	})

	return writeConfig(cfg)
}

// ResetForTest resets viper for testing (only use in tests)
func ResetForTest(testPath string) {
	configFile = testPath + "/.cvx-config.yaml"
	v = viper.New()
	v.SetConfigFile(configFile)
	v.SetDefault("model", "claude-cli")
}
