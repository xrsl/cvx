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

	if c.Model != "claude-cli" {
		t.Errorf("Expected default model 'claude-cli', got '%s'", c.Model)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	// Set values
	if err := Set("repo", "owner/repo"); err != nil {
		t.Fatalf("Set repo error: %v", err)
	}
	if err := Set("model", "claude-cli"); err != nil {
		t.Fatalf("Set model error: %v", err)
	}

	// Get values
	repo, err := Get("repo")
	if err != nil {
		t.Fatalf("Get repo error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", repo)
	}

	model, err := Get("model")
	if err != nil {
		t.Fatalf("Get model error: %v", err)
	}
	if model != "claude-cli" {
		t.Errorf("Expected model 'claude-cli', got '%s'", model)
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
		Title:  "Test Project",
		Statuses: map[string]string{
			"todo":    "abc123",
			"applied": "def456",
		},
	}

	if err := SaveProject(proj); err != nil {
		t.Fatalf("SaveProject error: %v", err)
	}

	// Reload and check
	c, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if c.Project.ID != "PVT_test123" {
		t.Errorf("Expected project ID 'PVT_test123', got '%s'", c.Project.ID)
	}
	if c.Project.Number != 1 {
		t.Errorf("Expected project number 1, got %d", c.Project.Number)
	}
	if c.Project.Title != "Test Project" {
		t.Errorf("Expected project title 'Test Project', got '%s'", c.Project.Title)
	}
	if len(c.Project.Statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(c.Project.Statuses))
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
