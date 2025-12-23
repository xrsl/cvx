package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const DefaultAgent = "claude-opus-4-5"

var SupportedAgents = []string{
	"claude-opus-4-5",
	"claude-sonnet-4",
	"claude-opus-4",
	"claude-3-5-sonnet",
	"claude-3-5-haiku",
}

func IsAgentSupported(agent string) bool {
	for _, a := range SupportedAgents {
		if a == agent {
			return true
		}
	}
	return false
}

type Client struct {
	client anthropic.Client
	model  string
}

func NewClient(model string) (*Client, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	if model == "" {
		model = DefaultAgent
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	// Extract text from response
	for _, block := range message.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

func (c *Client) Close() {
	// No cleanup needed for HTTP client
}
