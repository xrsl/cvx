package claude

import (
	"testing"
)

func TestModelMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude-sonnet-4", "claude-sonnet-4-20250514"},
		{"claude-sonnet-4-5", "claude-sonnet-4-5-20250929"},
		{"claude-opus-4", "claude-opus-4-20250514"},
		{"claude-opus-4-5", "claude-opus-4-5-20251101"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mapped, ok := modelMapping[tt.input]
			if !ok {
				t.Errorf("model %q not found in mapping", tt.input)
				return
			}
			if mapped != tt.expected {
				t.Errorf("model %q mapped to %q, expected %q", tt.input, mapped, tt.expected)
			}
		})
	}
}

func TestIsAgentSupported(t *testing.T) {
	supported := []string{
		"claude-sonnet-4",
		"claude-sonnet-4-5",
		"claude-opus-4",
		"claude-opus-4-5",
	}

	for _, agent := range supported {
		if !IsAgentSupported(agent) {
			t.Errorf("agent %q should be supported", agent)
		}
	}

	unsupported := []string{
		"claude-3",
		"gpt-4",
		"invalid",
	}

	for _, agent := range unsupported {
		if IsAgentSupported(agent) {
			t.Errorf("agent %q should not be supported", agent)
		}
	}
}

func TestNewClientMapsModel(t *testing.T) {
	// Set a dummy API key for testing
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	tests := []struct {
		input    string
		expected string
	}{
		{"claude-sonnet-4", "claude-sonnet-4-20250514"},
		{"claude-sonnet-4-5", "claude-sonnet-4-5-20250929"},
		{"claude-opus-4", "claude-opus-4-20250514"},
		{"claude-opus-4-5", "claude-opus-4-5-20251101"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			client, err := NewClient(tt.input)
			if err != nil {
				t.Fatalf("NewClient(%q) failed: %v", tt.input, err)
			}
			if client.model != tt.expected {
				t.Errorf("NewClient(%q).model = %q, want %q", tt.input, client.model, tt.expected)
			}
		})
	}
}
