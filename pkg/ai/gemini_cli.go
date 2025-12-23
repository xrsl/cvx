package ai

import (
	"context"
	"os/exec"
)

// GeminiCLI implements Client using the gemini CLI
type GeminiCLI struct {
	model string // e.g., "flash", "pro"
}

// NewGeminiCLI creates a Gemini CLI client
func NewGeminiCLI(model string) *GeminiCLI {
	return &GeminiCLI{model: model}
}

// IsGeminiCLIAvailable checks if gemini CLI is installed
func IsGeminiCLIAvailable() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (c *GeminiCLI) GenerateContent(ctx context.Context, prompt string) (string, error) {
	args := []string{"-p", prompt, "-o", "text"}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}
	cmd := exec.CommandContext(ctx, "gemini", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (c *GeminiCLI) Close() {
	// No cleanup needed
}
