package ai

import (
	"context"
	"os/exec"
)

// ClaudeCLI implements Client using the claude CLI
type ClaudeCLI struct {
	model string // e.g., "sonnet-4.5", "opus-4"
}

// NewClaudeCLI creates a Claude CLI client
func NewClaudeCLI(model string) *ClaudeCLI {
	return &ClaudeCLI{model: model}
}

// IsClaudeCLIAvailable checks if claude CLI is installed
func IsClaudeCLIAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeCLI) GenerateContent(ctx context.Context, prompt string) (string, error) {
	args := []string{"-p", prompt, "--output-format", "text"}
	if c.model != "" {
		args = append(args, "--model", "claude-"+c.model)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (c *ClaudeCLI) Close() {
	// No cleanup needed
}
