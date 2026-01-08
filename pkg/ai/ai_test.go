package ai

import (
	"testing"
)

func TestIsAgentSupported(t *testing.T) {
	// IsAgentSupported only checks CLI agents (claude, gemini) based on CLI availability
	// API model names are handled by the agent, not IsAgentSupported

	// These should always return false (not CLI agent format)
	alwaysFalse := []string{"gemini-2.5-flash", "claude-sonnet-4", "invalid-agent", ""}
	for _, agent := range alwaysFalse {
		if IsAgentSupported(agent) {
			t.Errorf("IsAgentSupported(%q) = true, want false (not a CLI agent)", agent)
		}
	}

	// CLI agents depend on CLI availability
	if IsClaudeCLIAvailable() {
		if !IsAgentSupported("claude") {
			t.Error("IsAgentSupported(claude) = false, want true (CLI available)")
		}
		if !IsAgentSupported("claude:opus-4.5") {
			t.Error("IsAgentSupported(claude:opus-4.5) = false, want true (CLI available)")
		}
	}
	if IsGeminiCLIAvailable() {
		if !IsAgentSupported("gemini") {
			t.Error("IsAgentSupported(gemini) = false, want true (CLI available)")
		}
	}
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()

	// SupportedAgents returns CLI agents that are available
	// In CI, no CLI tools are installed so this may be empty
	if !IsClaudeCLIAvailable() && !IsGeminiCLIAvailable() {
		if len(agents) != 0 {
			t.Errorf("SupportedAgents() = %v, want empty (no CLI available)", agents)
		}
		return
	}

	// If any CLI is available, check the list is correct
	if IsClaudeCLIAvailable() {
		found := false
		for _, a := range agents {
			if a == "claude" {
				found = true
				break
			}
		}
		if !found {
			t.Error("SupportedAgents() should include 'claude' when CLI is available")
		}
	}

	if IsGeminiCLIAvailable() {
		found := false
		for _, a := range agents {
			if a == "gemini" {
				found = true
				break
			}
		}
		if !found {
			t.Error("SupportedAgents() should include 'gemini' when CLI is available")
		}
	}
}

func TestNewClientGemini(t *testing.T) {
	// NewClient only supports CLI agents (gemini, claude)
	// API model names like gemini-2.5-flash go through the agent subprocess
	if !IsGeminiCLIAvailable() {
		t.Skip("Gemini CLI not available")
	}

	client, err := NewClient("gemini")
	if err != nil {
		t.Errorf("NewClient(gemini) error: %v", err)
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
