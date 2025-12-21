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

// DefaultModel returns the best available model
// Prefers claude-cli, then gemini-cli, then gemini API
func DefaultModel() string {
	if IsClaudeCLIAvailable() {
		return "claude-cli"
	}
	if IsGeminiCLIAvailable() {
		return "gemini-cli"
	}
	return gemini.DefaultModel
}

// NewClient creates an AI client based on model prefix
func NewClient(model string) (Client, error) {
	switch {
	case model == "claude-cli" || strings.HasPrefix(model, "claude-cli:"):
		if !IsClaudeCLIAvailable() {
			return nil, fmt.Errorf("claude CLI not found in PATH")
		}
		// Parse "claude-cli:sonnet-4.5" → "sonnet-4.5"
		subModel := ""
		if idx := strings.Index(model, ":"); idx != -1 {
			subModel = model[idx+1:]
		}
		return NewClaudeCLI(subModel), nil
	case model == "gemini-cli" || strings.HasPrefix(model, "gemini-cli:"):
		if !IsGeminiCLIAvailable() {
			return nil, fmt.Errorf("gemini CLI not found in PATH")
		}
		// Parse "gemini-cli:flash" → "flash"
		subModel := ""
		if idx := strings.Index(model, ":"); idx != -1 {
			subModel = model[idx+1:]
		}
		return NewGeminiCLI(subModel), nil
	case strings.HasPrefix(model, "gemini"):
		return gemini.NewClient(model)
	case strings.HasPrefix(model, "claude"):
		return claude.NewClient(model)
	default:
		return nil, fmt.Errorf("unknown model: %s (use claude-cli, gemini-cli, gemini-*, or claude-*)", model)
	}
}

// IsModelSupported checks if a model is supported by any provider
func IsModelSupported(model string) bool {
	switch {
	case model == "claude-cli" || strings.HasPrefix(model, "claude-cli:"):
		return IsClaudeCLIAvailable()
	case model == "gemini-cli" || strings.HasPrefix(model, "gemini-cli:"):
		return IsGeminiCLIAvailable()
	case strings.HasPrefix(model, "gemini"):
		return gemini.IsModelSupported(model)
	case strings.HasPrefix(model, "claude"):
		return claude.IsModelSupported(model)
	default:
		return false
	}
}

// SupportedModels returns all supported models
func SupportedModels() []string {
	models := []string{}
	if IsClaudeCLIAvailable() {
		models = append(models, "claude-cli")
	}
	if IsGeminiCLIAvailable() {
		models = append(models, "gemini-cli")
	}
	models = append(models, gemini.SupportedModels...)
	models = append(models, claude.SupportedModels...)
	return models
}
