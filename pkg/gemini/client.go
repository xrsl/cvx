package gemini

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/xrsl/cvx/pkg/retry"
)

const DefaultAgent = "gemini-2.5-flash"

// Rate limiter for API calls (1 request per second, conservative default)
var rateLimiter = retry.NewRateLimiter(1.0)

var SupportedAgents = []string{
	"gemini-3-flash-preview",
	"gemini-3-pro-preview",
	"gemini-2.5-flash",
	"gemini-2.5-pro",
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
	client    *genai.Client
	model     *genai.GenerativeModel
	modelName string
}

func NewClient(model string) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	if model == "" {
		model = DefaultAgent
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	m := client.GenerativeModel(model)
	m.ResponseMIMEType = "application/json"

	return &Client{
		client:    client,
		model:     m,
		modelName: model,
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
	// Retry on rate limits, quota exceeded, and temporary errors
	return strings.Contains(errStr, "RESOURCE_EXHAUSTED") ||
		strings.Contains(errStr, "UNAVAILABLE") ||
		strings.Contains(errStr, "429") ||
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
	case strings.Contains(errStr, "API_KEY_INVALID") || strings.Contains(errStr, "401"):
		return fmt.Errorf("gemini API error: invalid API key. Check GEMINI_API_KEY environment variable")
	case strings.Contains(errStr, "PERMISSION_DENIED") || strings.Contains(errStr, "403"):
		return fmt.Errorf("gemini API error: key does not have access to model %q. Check your Google AI account permissions", model)
	case strings.Contains(errStr, "NOT_FOUND") || strings.Contains(errStr, "404"):
		return fmt.Errorf("gemini API error: model %q not found. Verify the model name is correct", model)
	case strings.Contains(errStr, "RESOURCE_EXHAUSTED") || strings.Contains(errStr, "429"):
		return fmt.Errorf("gemini API error: rate limit exceeded for model %q. Please wait and try again", model)
	case strings.Contains(errStr, "UNAVAILABLE") || strings.Contains(errStr, "503"):
		return fmt.Errorf("gemini API error: service unavailable. Please try again later")
	case strings.Contains(errStr, "unregistered callers"):
		return fmt.Errorf("gemini API error: key not authorized for model %q. Get a new key from https://aistudio.google.com/apikey", model)
	default:
		return fmt.Errorf("gemini API error: %w", err)
	}
}

// GenerateContentWithSystem uses system instruction for the prompt
// Note: Gemini's context caching requires separate cache creation, so this just uses system instruction
func (c *Client) GenerateContentWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Apply rate limiting before making request
	if err := rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}

	if systemPrompt != "" {
		c.model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemPrompt)},
		}
	}

	cfg := retry.DefaultConfig()

	return retry.Do(ctx, cfg, func() (string, error) {
		resp, err := c.model.GenerateContent(ctx, genai.Text(userPrompt))
		if err != nil {
			if isRetryableError(err) {
				return "", retry.Retryable(formatAPIError(err, c.modelName))
			}
			return "", formatAPIError(err, c.modelName)
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("no content generated")
		}

		part := resp.Candidates[0].Content.Parts[0]
		if txt, ok := part.(genai.Text); ok {
			return string(txt), nil
		}

		return "", fmt.Errorf("unexpected response format")
	})
}

func (c *Client) Close() {
	_ = c.client.Close()
}
