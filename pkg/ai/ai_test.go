package ai

import (
	"strings"
	"testing"
)

func TestIsAgentSupported(t *testing.T) {
	tests := []struct {
		agent    string
		expected bool
	}{
		{"gemini-2.5-flash", true},
		{"gemini-2.5-pro", true},
		{"claude-sonnet-4", true},
		{"claude-sonnet-4-5", true},
		{"claude-opus-4", true},
		{"claude-opus-4-5", true},
		{"invalid-agent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := IsAgentSupported(tt.agent)
			if got != tt.expected {
				t.Errorf("IsAgentSupported(%q) = %v, want %v", tt.agent, got, tt.expected)
			}
		})
	}
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()

	if len(agents) == 0 {
		t.Error("SupportedAgents() returned empty list")
	}

	// Should include Gemini agents
	hasGemini := false
	for _, a := range agents {
		if strings.HasPrefix(a, "gemini") {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Error("SupportedAgents() should include Gemini agents")
	}
}

func TestDefaultAgent(t *testing.T) {
	agent := DefaultAgent()

	if agent == "" {
		t.Error("DefaultAgent() returned empty string")
	}

	// Default should be a supported agent
	if !IsAgentSupported(agent) {
		t.Errorf("DefaultAgent() returned unsupported agent: %s", agent)
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
	_, err := NewClient("invalid-agent")
	if err == nil {
		t.Error("Expected error for invalid agent")
	}
}

func TestNewClientClaude(t *testing.T) {
	if !IsClaudeCLIAvailable() {
		t.Skip("Claude CLI not available")
	}

	client, err := NewClient("claude")
	if err != nil {
		t.Errorf("NewClient(claude) error: %v", err)
		return
	}
	defer client.Close()
}

func TestNewClientClaudeWithSubAgent(t *testing.T) {
	if !IsClaudeCLIAvailable() {
		t.Skip("Claude CLI not available")
	}

	client, err := NewClient("claude:opus-4.5")
	if err != nil {
		t.Errorf("NewClient(claude:opus-4.5) error: %v", err)
		return
	}
	defer client.Close()
}

func TestIsAgentSupportedCLI(t *testing.T) {
	// CLI agents should be supported if CLI is available
	if IsClaudeCLIAvailable() {
		if !IsAgentSupported("claude") {
			t.Error("claude should be supported when CLI is available")
		}
		if !IsAgentSupported("claude:opus-4.5") {
			t.Error("claude:opus-4.5 should be supported when CLI is available")
		}
	}

	if IsGeminiCLIAvailable() {
		if !IsAgentSupported("gemini") {
			t.Error("gemini should be supported when CLI is available")
		}
	}
}
