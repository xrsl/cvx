package ai

import (
	"context"
	"fmt"
	"strings"
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

// NewClient creates an AI client based on agent prefix
// Only supports CLI agents (claude, gemini). API calls go through agent subprocess.
func NewClient(agent string) (Client, error) {
	switch {
	case agent == "claude" || strings.HasPrefix(agent, "claude:"):
		if !IsClaudeCLIAvailable() {
			return nil, fmt.Errorf("claude CLI not found in PATH")
		}
		// Parse "claude:sonnet-4.5" → "sonnet-4.5"
		subAgent := ""
		if idx := strings.Index(agent, ":"); idx != -1 {
			subAgent = agent[idx+1:]
		}
		return NewClaudeCLI(subAgent), nil
	case agent == "gemini" || strings.HasPrefix(agent, "gemini:"):
		if !IsGeminiCLIAvailable() {
			return nil, fmt.Errorf("gemini CLI not found in PATH")
		}
		// Parse "gemini:flash" → "flash"
		subAgent := ""
		if idx := strings.Index(agent, ":"); idx != -1 {
			subAgent = agent[idx+1:]
		}
		return NewGeminiCLI(subAgent), nil
	default:
		return nil, fmt.Errorf("unknown CLI agent: %s (use claude or gemini for CLI, or use -m flag for API)", agent)
	}
}

// IsAgentSupported checks if an agent is supported (CLI agents only)
func IsAgentSupported(agent string) bool {
	switch {
	case agent == "claude" || strings.HasPrefix(agent, "claude:"):
		return IsClaudeCLIAvailable()
	case agent == "gemini" || strings.HasPrefix(agent, "gemini:"):
		return IsGeminiCLIAvailable()
	default:
		return false
	}
}

// IsAgentCLI returns true if the agent is a CLI agent (claude, gemini)
func IsAgentCLI(agent string) bool {
	return agent == "claude" || strings.HasPrefix(agent, "claude:") ||
		agent == "gemini" || strings.HasPrefix(agent, "gemini:")
}

// IsCLIAgentSupported checks if a CLI agent is available
func IsCLIAgentSupported(agent string) bool {
	switch {
	case agent == "claude" || strings.HasPrefix(agent, "claude:"):
		return IsClaudeCLIAvailable()
	case agent == "gemini" || strings.HasPrefix(agent, "gemini:"):
		return IsGeminiCLIAvailable()
	default:
		return false
	}
}

// SupportedAgents returns all supported CLI agents
func SupportedAgents() []string {
	agents := []string{}
	if IsClaudeCLIAvailable() {
		agents = append(agents, "claude")
	}
	if IsGeminiCLIAvailable() {
		agents = append(agents, "gemini")
	}
	return agents
}

// SupportedCLIAgents returns supported CLI agents
func SupportedCLIAgents() []string {
	return SupportedAgents()
}

// Model represents a model configuration for both CLI and API usage
type Model struct {
	Name    string // Short name (e.g., "sonnet-4")
	CLIName string // CLI parameter name (e.g., "sonnet-4")
	APIName string // Full API model name (e.g., "claude-sonnet-4")
}

// SupportedModelMap maps short model names to their configurations
// Note: When using -m flag, the model name is passed directly to the agent
// which supports any model pydantic-ai supports. These short names are for convenience.
var SupportedModelMap = map[string]Model{
	"sonnet-4":     {Name: "sonnet-4", CLIName: "sonnet-4", APIName: "claude-sonnet-4"},
	"sonnet-4-5":   {Name: "sonnet-4-5", CLIName: "sonnet-4-5", APIName: "claude-sonnet-4-5"},
	"opus-4":       {Name: "opus-4", CLIName: "opus-4", APIName: "claude-opus-4"},
	"opus-4-5":     {Name: "opus-4-5", CLIName: "opus-4-5", APIName: "claude-opus-4-5"},
	"haiku-4":      {Name: "haiku-4", CLIName: "haiku-4", APIName: "claude-haiku-4"},
	"haiku-4-5":    {Name: "haiku-4-5", CLIName: "haiku-4-5", APIName: "claude-haiku-4-5"},
	"flash-2-5":    {Name: "flash-2-5", CLIName: "flash-2-5", APIName: "gemini-2.5-flash"},
	"pro-2-5":      {Name: "pro-2-5", CLIName: "pro-2-5", APIName: "gemini-2.5-pro"},
	"flash-3":      {Name: "flash-3", CLIName: "flash-3", APIName: "gemini-3-flash-preview"},
	"pro-3":        {Name: "pro-3", CLIName: "pro-3", APIName: "gemini-3-pro-preview"},
	"gpt-oss-120b": {Name: "gpt-oss-120b", CLIName: "gpt-oss-120b", APIName: "openai/gpt-oss-120b"},
	"qwen3-32b":    {Name: "qwen3-32b", CLIName: "qwen3-32b", APIName: "qwen/qwen3-32b"},
}

// GetModel returns the model configuration for a given short name
// If not found, returns the model name as-is (pass-through to agent)
func GetModel(shortName string) (Model, bool) {
	model, ok := SupportedModelMap[shortName]
	if !ok {
		// Pass through unknown models - agent can handle any model
		return Model{Name: shortName, CLIName: shortName, APIName: shortName}, true
	}
	return model, true
}

// SupportedModelNames returns list of supported short model names
func SupportedModelNames() []string {
	names := make([]string, 0, len(SupportedModelMap))
	for name := range SupportedModelMap {
		names = append(names, name)
	}
	return names
}
