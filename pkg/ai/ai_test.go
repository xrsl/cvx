package ai

import (
	"strings"
	"testing"
)

func TestIsModelSupported(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"gemini-2.5-flash", true},
		{"gemini-2.5-pro", true},
		{"claude-sonnet-4", true},
		{"claude-opus-4", true},
		{"invalid-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := IsModelSupported(tt.model)
			if got != tt.expected {
				t.Errorf("IsModelSupported(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestSupportedModels(t *testing.T) {
	models := SupportedModels()

	if len(models) == 0 {
		t.Error("SupportedModels() returned empty list")
	}

	// Should include Gemini models
	hasGemini := false
	for _, m := range models {
		if strings.HasPrefix(m, "gemini") {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Error("SupportedModels() should include Gemini models")
	}
}

func TestDefaultModel(t *testing.T) {
	model := DefaultModel()

	if model == "" {
		t.Error("DefaultModel() returned empty string")
	}

	// Default should be a supported model
	if !IsModelSupported(model) {
		t.Errorf("DefaultModel() returned unsupported model: %s", model)
	}
}

func TestNewClientGemini(t *testing.T) {
	// Skip if no API key (just test that it doesn't panic)
	client, err := NewClient("gemini-2.5-flash")
	if err != nil {
		// Expected if no API key
		if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Errorf("Unexpected error: %v", err)
		}
		return
	}
	defer client.Close()
}

func TestNewClientInvalid(t *testing.T) {
	_, err := NewClient("invalid-model")
	if err == nil {
		t.Error("Expected error for invalid model")
	}
}

func TestNewClientClaudeCLI(t *testing.T) {
	if !IsClaudeCLIAvailable() {
		t.Skip("Claude CLI not available")
	}

	client, err := NewClient("claude-cli")
	if err != nil {
		t.Errorf("NewClient(claude-cli) error: %v", err)
		return
	}
	defer client.Close()
}

func TestNewClientClaudeCLIWithModel(t *testing.T) {
	if !IsClaudeCLIAvailable() {
		t.Skip("Claude CLI not available")
	}

	client, err := NewClient("claude-cli:opus-4.5")
	if err != nil {
		t.Errorf("NewClient(claude-cli:opus-4.5) error: %v", err)
		return
	}
	defer client.Close()
}

func TestIsModelSupportedCLI(t *testing.T) {
	// CLI models should be supported if CLI is available
	if IsClaudeCLIAvailable() {
		if !IsModelSupported("claude-cli") {
			t.Error("claude-cli should be supported when CLI is available")
		}
		if !IsModelSupported("claude-cli:opus-4.5") {
			t.Error("claude-cli:opus-4.5 should be supported when CLI is available")
		}
	}

	if IsGeminiCLIAvailable() {
		if !IsModelSupported("gemini-cli") {
			t.Error("gemini-cli should be supported when CLI is available")
		}
	}
}
