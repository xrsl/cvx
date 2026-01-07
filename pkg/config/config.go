package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/utils"
)

type GitHubConfig struct {
	Repo    string `toml:"repo" mapstructure:"repo"`
	Project string `toml:"project" mapstructure:"project"`
}

type AgentConfig struct {
	Default string `toml:"default" mapstructure:"default"`
}

type SchemaConfig struct {
	JobAd string `toml:"job-ad" mapstructure:"job-ad"`
}

type PathsConfig struct {
	Reference string `toml:"reference" mapstructure:"reference"`
}

type CVConfig struct {
	Source string `toml:"source" mapstructure:"source"`
	Output string `toml:"output" mapstructure:"output"`
	Schema string `toml:"schema" mapstructure:"schema"`
}

type LetterConfig struct {
	Source string `toml:"source" mapstructure:"source"`
	Output string `toml:"output" mapstructure:"output"`
	Schema string `toml:"schema" mapstructure:"schema"`
}

type Config struct {
	GitHub GitHubConfig `toml:"github" mapstructure:"github"`
	Agent  AgentConfig  `toml:"agent" mapstructure:"agent"`
	Schema SchemaConfig `toml:"schema" mapstructure:"schema"`
	Paths  PathsConfig  `toml:"paths" mapstructure:"paths"`
	CV     CVConfig     `toml:"cv" mapstructure:"cv"`
	Letter LetterConfig `toml:"letter" mapstructure:"letter"`
}

func (c *Config) AgentCLI() string {
	if c.Agent.Default == "gemini" || strings.HasPrefix(c.Agent.Default, "gemini:") {
		return "gemini"
	}
	if c.Agent.Default == "claude" || strings.HasPrefix(c.Agent.Default, "claude:") {
		return "claude"
	}
	return "claude"
}

func (c *Config) ProjectOwner() string {
	parts := strings.Split(c.GitHub.Project, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	parts = strings.Split(c.GitHub.Repo, "/")
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

func (c *Config) ProjectNumber() int {
	parts := strings.Split(c.GitHub.Project, "/")
	if len(parts) == 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil {
			return n
		}
	}
	return 0
}

type FieldIDs struct {
	Status      string `yaml:"status,omitempty"`
	Company     string `yaml:"company,omitempty"`
	Deadline    string `yaml:"deadline,omitempty"`
	AppliedDate string `yaml:"applied_date,omitempty"`
}

type ProjectCache struct {
	ID       string            `yaml:"id"`
	Number   int               `yaml:"number"`
	Title    string            `yaml:"title"`
	Fields   FieldIDs          `yaml:"fields"`
	Statuses map[string]string `yaml:"statuses"`
}

var (
	configFile = "cvx.toml"
	cacheFile  = ".cvx/cache.yaml"
	v          *viper.Viper
)

func init() {
	v = viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")
	v.SetDefault("agent.default", "claude")
	v.SetEnvPrefix("CVX")
	v.AutomaticEnv()
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
	if cfg.Agent.Default != "" && !ai.IsAgentCLI(cfg.Agent.Default) {
		return nil, fmt.Errorf("invalid agent '%s'. Only CLI agents allowed", cfg.Agent.Default)
	}
	return &cfg, nil
}

func Save(c *Config) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(c); err != nil {
		return err
	}
	// Convert single quotes to double quotes
	output := strings.ReplaceAll(buf.String(), "'", "\"")
	if err := os.WriteFile(configFile, []byte(output), 0o644); err != nil {
		return err
	}
	// Reload viper config after writing
	return v.ReadInConfig()
}

func SaveProject(owner string, number int, cache ProjectCache) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}
	cfg.GitHub.Project = fmt.Sprintf("%s/%d", owner, number)
	cache.Number = number
	if err := saveProjectCache(cache); err != nil {
		return fmt.Errorf("failed to save project cache: %w", err)
	}
	return Save(cfg)
}

func saveProjectCache(cache ProjectCache) error {
	if err := utils.EnsureCvxGitignore(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(map[string]ProjectCache{"project": cache})
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0o644)
}

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

func LoadWithCache() (*Config, *ProjectCache, error) {
	cfg, err := Load()
	if err != nil {
		return nil, nil, err
	}
	projectNum := cfg.ProjectNumber()
	if projectNum > 0 {
		if cache, err := LoadProjectCache(); err == nil && cache.Number == projectNum {
			return cfg, cache, nil
		}
	}
	return cfg, nil, nil
}

func ResetForTest(testPath string) {
	configFile = testPath + "/cvx.toml"
	cacheFile = testPath + "/.cvx/cache.yaml"
	v = viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")
	v.SetDefault("agent.default", "claude")
}
