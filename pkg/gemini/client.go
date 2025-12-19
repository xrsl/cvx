package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const DefaultModel = "gemini-3.0-flash"

var SupportedModels = []string{
	"gemini-3.0-flash",
	"gemini-2.5-pro",
	"gemini-2.5-flash",
	"gemini-2.0-pro",
	"gemini-2.0-flash",
	"gemini-1.5-pro",
	"gemini-1.5-flash",
}

func IsModelSupported(model string) bool {
	for _, m := range SupportedModels {
		if m == model {
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
		model = DefaultModel
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
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
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
	c.client.Close()
}
