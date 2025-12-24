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

	if c.Agent != "claude" {
		t.Errorf("Expected default agent 'claude', got '%s'", c.Agent)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	// Set values
	if err := Set("repo", "owner/repo"); err != nil {
		t.Fatalf("Set repo error: %v", err)
	}
	if err := Set("agent", "claude"); err != nil {
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
	if agent != "claude" {
		t.Errorf("Expected agent 'claude', got '%s'", agent)
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

	cache := ProjectCache{
		ID:     "PVT_test123",
		Number: 1,
		Title:  "Test Project",
		Statuses: map[string]string{
			"todo":    "abc123",
			"applied": "def456",
		},
	}

	if err := SaveProject("testowner", 1, cache); err != nil {
		t.Fatalf("SaveProject error: %v", err)
	}

	// Check user-facing config (only number and owner)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c.ProjectNumber() != 1 {
		t.Errorf("Expected project number 1, got %d", c.ProjectNumber())
	}
	if c.ProjectOwner() != "testowner" {
		t.Errorf("Expected project owner 'testowner', got '%s'", c.ProjectOwner())
	}

	// Check full config with cache (includes internal IDs)
	_, projectCache, err := LoadWithCache()
	if err != nil {
		t.Fatalf("LoadWithCache error: %v", err)
	}
	if projectCache == nil {
		t.Fatal("Expected project cache to be loaded, got nil")
	}
	if projectCache.ID != "PVT_test123" {
		t.Errorf("Expected project ID 'PVT_test123', got '%s'", projectCache.ID)
	}
	if projectCache.Title != "Test Project" {
		t.Errorf("Expected project title 'Test Project', got '%s'", projectCache.Title)
	}
	if len(projectCache.Statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(projectCache.Statuses))
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
