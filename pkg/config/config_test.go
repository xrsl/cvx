package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if c.Agent != "claude-cli" {
		t.Errorf("Expected default agent 'claude-cli', got '%s'", c.Agent)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	// Set values
	if err := Set("repo", "owner/repo"); err != nil {
		t.Fatalf("Set repo error: %v", err)
	}
	if err := Set("agent", "claude-cli"); err != nil {
		t.Fatalf("Set agent error: %v", err)
	}

	// Get values
	repo, err := Get("repo")
	if err != nil {
		t.Fatalf("Get repo error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", repo)
	}

	agent, err := Get("agent")
	if err != nil {
		t.Fatalf("Get agent error: %v", err)
	}
	if agent != "claude-cli" {
		t.Errorf("Expected agent 'claude-cli', got '%s'", agent)
	}
}

func TestSetInvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	err := Set("invalid_key", "value")
	if err == nil {
		t.Error("Expected error for invalid key, got nil")
	}
}

func TestGetInvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	_, err := Get("invalid_key")
	if err == nil {
		t.Error("Expected error for invalid key, got nil")
	}
}

func TestSaveProject(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	proj := ProjectConfig{
		ID:     "PVT_test123",
		Number: 1,
		Owner:  "testowner",
		Title:  "Test Project",
		Statuses: map[string]string{
			"todo":    "abc123",
			"applied": "def456",
		},
	}

	if err := SaveProject(proj); err != nil {
		t.Fatalf("SaveProject error: %v", err)
	}

	// Check user-facing config (only number and owner)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c.Project.Number != 1 {
		t.Errorf("Expected project number 1, got %d", c.Project.Number)
	}
	if c.Project.Owner != "testowner" {
		t.Errorf("Expected project owner 'testowner', got '%s'", c.Project.Owner)
	}

	// Check full config with cache (includes internal IDs)
	cFull, err := LoadWithCache()
	if err != nil {
		t.Fatalf("LoadWithCache error: %v", err)
	}
	if cFull.Project.ID != "PVT_test123" {
		t.Errorf("Expected project ID 'PVT_test123', got '%s'", cFull.Project.ID)
	}
	if cFull.Project.Title != "Test Project" {
		t.Errorf("Expected project title 'Test Project', got '%s'", cFull.Project.Title)
	}
	if len(cFull.Project.Statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(cFull.Project.Statuses))
	}
}

func TestConfigPath(t *testing.T) {
	path := Path()
	if path == "" {
		t.Error("Path() returned empty string")
	}
}

func TestConfigFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	if err := Set("repo", "test/repo"); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}
