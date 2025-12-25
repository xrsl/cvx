package gemini

import (
	"testing"
)

func TestIsAgentSupported(t *testing.T) {
	supported := []string{
		"gemini-3-flash-preview",
		"gemini-3-pro-preview",
		"gemini-2.5-flash",
		"gemini-2.5-pro",
	}

	for _, agent := range supported {
		if !IsAgentSupported(agent) {
			t.Errorf("agent %q should be supported", agent)
		}
	}

	unsupported := []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gpt-4",
		"invalid",
	}

	for _, agent := range unsupported {
		if IsAgentSupported(agent) {
			t.Errorf("agent %q should not be supported", agent)
		}
	}
}

func TestNewClientDefaultAgent(t *testing.T) {
	if DefaultAgent != "gemini-2.5-flash" {
		t.Errorf("DefaultAgent = %q, want %q", DefaultAgent, "gemini-2.5-flash")
	}
}
