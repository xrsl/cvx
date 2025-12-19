package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo   string `yaml:"repo"`
	Model  string `yaml:"model"`
	Schema string `yaml:"schema"`
}

var (
	cfg     *Config
	cfgPath string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	cfgPath = filepath.Join(home, ".config", "cvx", "config.yaml")
}

func Path() string {
	return cfgPath
}

func Load() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		Model: "gemini-3.0-flash", // default
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // return defaults
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func Get(key string) (string, error) {
	c, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case "repo":
		return c.Repo, nil
	case "model":
		return c.Model, nil
	case "schema":
		return c.Schema, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func Set(key, value string) error {
	c, err := Load()
	if err != nil {
		return err
	}

	switch key {
	case "repo":
		c.Repo = value
	case "model":
		c.Model = value
	case "schema":
		c.Schema = value
	default:
		return fmt.Errorf("unknown config key: %s (valid: repo, model, schema)", key)
	}

	return save(c)
}

func save(c *Config) error {
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, data, 0644)
}

func All() (map[string]string, error) {
	c, err := Load()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"repo":   c.Repo,
		"model":  c.Model,
		"schema": c.Schema,
	}, nil
}
