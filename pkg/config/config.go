package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Repo    string        `mapstructure:"repo"`
	Model   string        `mapstructure:"model"`
	Schema  string        `mapstructure:"schema"`
	Project ProjectConfig `mapstructure:"project"`
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

var (
	v       *viper.Viper
	cfgPath string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	cfgPath = filepath.Join(home, ".config", "cvx")

	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(cfgPath)

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
	return filepath.Join(cfgPath, "config.yaml")
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
	case "repo", "model", "schema":
		return v.GetString(key), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func Set(key, value string) error {
	switch key {
	case "repo", "model", "schema":
		v.Set(key, value)
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, model, schema)", key)
	}
	return save()
}

func save() error {
	// Ensure config directory exists
	if err := os.MkdirAll(cfgPath, 0755); err != nil {
		return err
	}

	return v.WriteConfigAs(Path())
}

func All() (map[string]string, error) {
	return map[string]string{
		"repo":   v.GetString("repo"),
		"model":  v.GetString("model"),
		"schema": v.GetString("schema"),
	}, nil
}

// Save saves the full config
func Save(c *Config) error {
	v.Set("repo", c.Repo)
	v.Set("model", c.Model)
	v.Set("schema", c.Schema)
	v.Set("project", c.Project)
	return save()
}

// SaveProject saves project configuration
func SaveProject(p ProjectConfig) error {
	v.Set("project", p)
	return save()
}

// ResetForTest resets viper for testing (only use in tests)
func ResetForTest(testPath string) {
	cfgPath = testPath
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(cfgPath)
	v.SetDefault("model", "claude-cli")
}
