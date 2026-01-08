package claude

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/xrsl/cvx/pkg/retry"
)

const DefaultAgent = "claude-sonnet-4"

// Rate limiter for API calls (1 request per second, conservative default)
var rateLimiter = retry.NewRateLimiter(1.0)

var SupportedAgents = []string{
	"claude-sonnet-4",
	"claude-sonnet-4-5",
	"claude-opus-4",
	"claude-opus-4-5",
	"claude-haiku-4",
	"claude-haiku-4-5",
}

// Map friendly agent names to Anthropic model IDs
var modelMapping = map[string]string{
	"claude-sonnet-4":   "claude-sonnet-4-20250514",
	"claude-sonnet-4-5": "claude-sonnet-4-5-20250929",
	"claude-opus-4":     "claude-opus-4-20250514",
	"claude-opus-4-5":   "claude-opus-4-5-20251101",
	"claude-haiku-4":    "claude-haiku-4-20250514",
	"claude-haiku-4-5":  "claude-haiku-4-5-20251001",
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

	// Map agent name to Anthropic model ID
	modelID, ok := modelMapping[model]
	if !ok {
		modelID = model // fallback to raw value if not in mapping
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Client{
		client: client,
		model:  modelID,
	}, nil
}

func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	return c.GenerateContentWithSystem(ctx, "", prompt)
}

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Retry on rate limits, overloaded, and temporary network issues
	return strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, "overloaded") ||
		strings.Contains(errStr, "529") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "timeout")
}

// formatAPIError converts API errors to user-friendly messages
func formatAPIError(err error, model string) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "401") || strings.Contains(errStr, "authentication_error"):
		return fmt.Errorf("claude API error: invalid API key. Check ANTHROPIC_API_KEY environment variable")
	case strings.Contains(errStr, "403") || strings.Contains(errStr, "permission_denied"):
		return fmt.Errorf("claude API error: key does not have access to model %q. Check your Anthropic account permissions", model)
	case strings.Contains(errStr, "404") || strings.Contains(errStr, "not_found"):
		return fmt.Errorf("claude API error: model %q not found. Verify the model name is correct", model)
	case strings.Contains(errStr, "rate_limit"):
		return fmt.Errorf("claude API error: rate limit exceeded for model %q. Please wait and try again", model)
	case strings.Contains(errStr, "overloaded") || strings.Contains(errStr, "529"):
		return fmt.Errorf("claude API error: service overloaded. Please try again later")
	default:
		return fmt.Errorf("claude API error: %w", err)
	}
}

// GenerateContentWithSystem sends a prompt with a cached system message
// The system prompt is marked for caching (5-min TTL, 90% cost reduction on cache hit)
func (c *Client) GenerateContentWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Apply rate limiting before making request
	if err := rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}

	cfg := retry.DefaultConfig()

	return retry.Do(ctx, cfg, func() (string, error) {
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
			if isRetryableError(err) {
				return "", retry.Retryable(formatAPIError(err, c.model))
			}
			return "", formatAPIError(err, c.model)
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
	})
}

func (c *Client) Close() {
	// No cleanup needed for HTTP client
}
