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

// NewClient creates an AI client based on model prefix
func NewClient(model string) (Client, error) {
	switch {
	case strings.HasPrefix(model, "gemini"):
		return gemini.NewClient(model)
	case strings.HasPrefix(model, "claude"):
		return claude.NewClient(model)
	default:
		return nil, fmt.Errorf("unknown model: %s (use gemini-* or claude-*)", model)
	}
}

// IsModelSupported checks if a model is supported by any provider
func IsModelSupported(model string) bool {
	switch {
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
	models := make([]string, 0, len(gemini.SupportedModels)+len(claude.SupportedModels))
	models = append(models, gemini.SupportedModels...)
	models = append(models, claude.SupportedModels...)
	return models
}
