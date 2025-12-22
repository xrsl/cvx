package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Repo           string        `mapstructure:"repo"`
	Model          string        `mapstructure:"model"`
	Schema         string        `mapstructure:"schema"`
	Project        ProjectConfig `mapstructure:"project"`
	CVPath         string        `mapstructure:"cv_path"`
	ExperiencePath string        `mapstructure:"experience_path"`
}

// Agent returns the CLI agent name derived from the model setting
func (c *Config) Agent() string {
	if strings.HasPrefix(c.Model, "gemini") {
		return "gemini"
	}
	return "claude"
}

type ProjectConfig struct {
	ID       string            `mapstructure:"id"`
	Number   int               `mapstructure:"number"`
	Title    string            `mapstructure:"title"`
	Fields   FieldIDs          `mapstructure:"fields"`
	Statuses map[string]string `mapstructure:"statuses"`
}

type FieldIDs struct {
	Status      string `mapstructure:"status"`
	Company     string `mapstructure:"company"`
	Deadline    string `mapstructure:"deadline"`
	AppliedDate string `mapstructure:"applied_date"`
}

const configFile = ".cvx-config.yaml"

var v *viper.Viper

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
	switch key {
	case "repo", "model", "schema", "cv_path", "experience_path":
		v.Set(key, value)
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, model, schema, cv_path, experience_path)", key)
	}
	return save()
}

func save() error {
	return v.WriteConfigAs(configFile)
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
	v.Set("repo", c.Repo)
	v.Set("model", c.Model)
	v.Set("schema", c.Schema)
	v.Set("project", c.Project)
	v.Set("cv_path", c.CVPath)
	v.Set("experience_path", c.ExperiencePath)
	return save()
}

// SaveProject saves project configuration
func SaveProject(p ProjectConfig) error {
	v.Set("project", p)
	return save()
}

// ResetForTest resets viper for testing (only use in tests)
func ResetForTest(testConfigFile string) {
	v = viper.New()
	v.SetConfigFile(testConfigFile)
	v.SetDefault("model", "claude-cli")
}
