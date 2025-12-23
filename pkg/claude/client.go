package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const DefaultAgent = "claude-sonnet-4"

var SupportedAgents = []string{
	"claude-sonnet-4",
	"claude-sonnet-4-5",
	"claude-opus-4",
	"claude-opus-4-5",
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
	return c.GenerateContentWithSystem(ctx, "", prompt)
}

// GenerateContentWithSystem sends a prompt with a cached system message
// The system prompt is marked for caching (5-min TTL, 90% cost reduction on cache hit)
func (c *Client) GenerateContentWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	}

	// Add system prompt with cache control if provided
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: systemPrompt,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		}
	}

	message, err := c.client.Messages.New(ctx, params)
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
