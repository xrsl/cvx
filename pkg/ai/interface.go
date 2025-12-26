package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/xrsl/cvx/pkg/claude"
	"github.com/xrsl/cvx/pkg/gemini"
)

// Client is the common interface for AI providers
type Client interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
	Close()
}

// CachingClient supports prompt caching (optional interface)
type CachingClient interface {
	Client
	GenerateContentWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// DefaultAgent returns the best available agent
// Prefers claude-code, then gemini-cli, then API agents
func DefaultAgent() string {
	if IsClaudeCLIAvailable() {
		return "claude-code"
	}
	if IsGeminiCLIAvailable() {
		return "gemini-cli"
	}
	return gemini.DefaultAgent
}

// NewClient creates an AI client based on agent prefix
func NewClient(agent string) (Client, error) {
	switch {
	case agent == "claude-code" || strings.HasPrefix(agent, "claude-code:"):
		if !IsClaudeCLIAvailable() {
			return nil, fmt.Errorf("claude CLI not found in PATH")
		}
		// Parse "claude-code:sonnet-4.5" → "sonnet-4.5"
		subAgent := ""
		if idx := strings.Index(agent, ":"); idx != -1 {
			subAgent = agent[idx+1:]
		}
		return NewClaudeCLI(subAgent), nil
	case agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:"):
		if !IsGeminiCLIAvailable() {
			return nil, fmt.Errorf("gemini CLI not found in PATH")
		}
		// Parse "gemini-cli:flash" → "flash"
		subAgent := ""
		if idx := strings.Index(agent, ":"); idx != -1 {
			subAgent = agent[idx+1:]
		}
		return NewGeminiCLI(subAgent), nil
	case strings.HasPrefix(agent, "gemini-"):
		return gemini.NewClient(agent)
	case strings.HasPrefix(agent, "claude-"):
		return claude.NewClient(agent)
	default:
		return nil, fmt.Errorf("unknown agent: %s (use claude-code, gemini-cli, gemini-*, or claude-*)", agent)
	}
}

// IsAgentSupported checks if an agent is supported by any provider
func IsAgentSupported(agent string) bool {
	switch {
	case agent == "claude-code" || strings.HasPrefix(agent, "claude-code:"):
		return IsClaudeCLIAvailable()
	case agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:"):
		return IsGeminiCLIAvailable()
	case strings.HasPrefix(agent, "gemini-"):
		return gemini.IsAgentSupported(agent)
	case strings.HasPrefix(agent, "claude-"):
		return claude.IsAgentSupported(agent)
	default:
		return false
	}
}

// IsAgentCLI returns true if the agent is a CLI agent (claude-code, gemini-cli)
func IsAgentCLI(agent string) bool {
	return agent == "claude-code" || strings.HasPrefix(agent, "claude-code:") ||
		agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:")
}

// IsCLIAgentSupported checks if a CLI agent is available
func IsCLIAgentSupported(agent string) bool {
	switch {
	case agent == "claude-code" || strings.HasPrefix(agent, "claude-code:"):
		return IsClaudeCLIAvailable()
	case agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:"):
		return IsGeminiCLIAvailable()
	default:
		return false
	}
}

// IsModelSupported checks if an API model is supported
func IsModelSupported(model string) bool {
	switch {
	case strings.HasPrefix(model, "gemini-"):
		return gemini.IsAgentSupported(model)
	case strings.HasPrefix(model, "claude-"):
		return claude.IsAgentSupported(model)
	default:
		return false
	}
}

// SupportedAgents returns all supported agents (CLI + API)
func SupportedAgents() []string {
	agents := []string{}
	if IsClaudeCLIAvailable() {
		agents = append(agents, "claude-code")
	}
	if IsGeminiCLIAvailable() {
		agents = append(agents, "gemini-cli")
	}
	agents = append(agents, gemini.SupportedAgents...)
	agents = append(agents, claude.SupportedAgents...)
	return agents
}

// SupportedCLIAgents returns supported CLI agents
func SupportedCLIAgents() []string {
	agents := []string{}
	if IsClaudeCLIAvailable() {
		agents = append(agents, "claude-code")
	}
	if IsGeminiCLIAvailable() {
		agents = append(agents, "gemini-cli")
	}
	return agents
}

// Model represents a model configuration for both CLI and API usage
type Model struct {
	Name    string // Short name (e.g., "sonnet-4")
	CLIName string // CLI parameter name (e.g., "sonnet-4")
	APIName string // Full API model name (e.g., "claude-sonnet-4")
}

// SupportedModelMap maps short model names to their configurations
var SupportedModelMap = map[string]Model{
	"sonnet-4":   {Name: "sonnet-4", CLIName: "sonnet-4", APIName: "claude-sonnet-4"},
	"sonnet-4-5": {Name: "sonnet-4-5", CLIName: "sonnet-4-5", APIName: "claude-sonnet-4-5"},
	"opus-4":     {Name: "opus-4", CLIName: "opus-4", APIName: "claude-opus-4"},
	"opus-4-5":   {Name: "opus-4-5", CLIName: "opus-4-5", APIName: "claude-opus-4-5"},
	"flash":      {Name: "flash", CLIName: "flash", APIName: "gemini-2.5-flash"},
	"pro":        {Name: "pro", CLIName: "pro", APIName: "gemini-2.5-pro"},
	"flash-3":    {Name: "flash-3", CLIName: "flash-3", APIName: "gemini-3-flash-preview"},
	"pro-3":      {Name: "pro-3", CLIName: "pro-3", APIName: "gemini-3-pro-preview"},
}

// GetModel returns the model configuration for a given short name
func GetModel(shortName string) (Model, bool) {
	model, ok := SupportedModelMap[shortName]
	return model, ok
}

// SupportedModelNames returns list of supported short model names
func SupportedModelNames() []string {
	names := make([]string, 0, len(SupportedModelMap))
	for name := range SupportedModelMap {
		names = append(names, name)
	}
	return names
}

// SupportedModels returns supported API models (full names)
func SupportedModels() []string {
	models := []string{}
	models = append(models, claude.SupportedAgents...)
	models = append(models, gemini.SupportedAgents...)
	return models
}
