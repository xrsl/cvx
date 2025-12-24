package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const DefaultAgent = "gemini-2.5-flash"

var SupportedAgents = []string{
	"gemini-2.5-flash",
	"gemini-2.5-pro",
	"gemini-2.0-flash",
	"gemini-1.5-pro",
	"gemini-1.5-flash",
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
	client *genai.Client
	model  *genai.GenerativeModel
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
		client: client,
		model:  m,
	}, nil
}

func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	return c.GenerateContentWithSystem(ctx, "", prompt)
}

// GenerateContentWithSystem uses system instruction for the prompt
// Note: Gemini's context caching requires separate cache creation, so this just uses system instruction
func (c *Client) GenerateContentWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if systemPrompt != "" {
		c.model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemPrompt)},
		}
	}

	resp, err := c.model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	part := resp.Candidates[0].Content.Parts[0]
	if txt, ok := part.(genai.Text); ok {
		return string(txt), nil
	}

	return "", fmt.Errorf("unexpected response format")
}

func (c *Client) Close() {
	_ = c.client.Close()
}
