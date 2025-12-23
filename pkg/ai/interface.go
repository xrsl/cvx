package ai

import (
	"context"
	"fmt"
	"strings"

	"cvx/pkg/claude"
	"cvx/pkg/gemini"
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
// Prefers claude, then gemini, then API agents
func DefaultAgent() string {
	if IsClaudeCLIAvailable() {
		return "claude"
	}
	if IsGeminiCLIAvailable() {
		return "gemini"
	}
	return gemini.DefaultAgent
}

// NewClient creates an AI client based on agent prefix
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
	case strings.HasPrefix(agent, "gemini-"):
		return gemini.NewClient(agent)
	case strings.HasPrefix(agent, "claude-"):
		return claude.NewClient(agent)
	default:
		return nil, fmt.Errorf("unknown agent: %s (use claude, gemini, gemini-*, or claude-*)", agent)
	}
}

// IsAgentSupported checks if an agent is supported by any provider
func IsAgentSupported(agent string) bool {
	switch {
	case agent == "claude" || strings.HasPrefix(agent, "claude:"):
		return IsClaudeCLIAvailable()
	case agent == "gemini" || strings.HasPrefix(agent, "gemini:"):
		return IsGeminiCLIAvailable()
	case strings.HasPrefix(agent, "gemini-"):
		return gemini.IsAgentSupported(agent)
	case strings.HasPrefix(agent, "claude-"):
		return claude.IsAgentSupported(agent)
	default:
		return false
	}
}

// IsAgentCLI returns true if the agent is a CLI agent (claude, gemini)
func IsAgentCLI(agent string) bool {
	return agent == "claude" || strings.HasPrefix(agent, "claude:") ||
		agent == "gemini" || strings.HasPrefix(agent, "gemini:")
}

// SupportedAgents returns all supported agents
func SupportedAgents() []string {
	agents := []string{}
	if IsClaudeCLIAvailable() {
		agents = append(agents, "claude")
	}
	if IsGeminiCLIAvailable() {
		agents = append(agents, "gemini")
	}
	agents = append(agents, gemini.SupportedAgents...)
	agents = append(agents, claude.SupportedAgents...)
	return agents
}
