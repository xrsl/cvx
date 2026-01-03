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

	if c.Agent.Default != "claude" {
		t.Errorf("Expected default agent 'claude', got '%s'", c.Agent.Default)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	cfg := &Config{
		GitHub: GitHubConfig{Repo: "owner/repo", Project: "owner/1"},
		Agent:  AgentConfig{Default: "claude"},
		Schema: SchemaConfig{JobAd: ".github/ISSUE_TEMPLATE/job-ad.yaml"},
		Paths:  PathsConfig{Reference: "reference/"},
		CV:     CVConfig{Source: "src/cv.yaml", Output: "out/cv.pdf", Schema: "schema/schema.json"},
		Letter: LetterConfig{Source: "src/letter.yaml", Output: "out/letter.pdf", Schema: "schema/schema.json"},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.GitHub.Repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", loaded.GitHub.Repo)
	}
	if loaded.Agent.Default != "claude" {
		t.Errorf("Expected agent 'claude', got '%s'", loaded.Agent.Default)
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
	if path == "cvx.toml" {
		return
	}
	t.Errorf("Expected path 'cvx.toml', got '%s'", path)
}

func TestConfigFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	ResetForTest(tmpDir)

	cfg := &Config{
		GitHub: GitHubConfig{Repo: "test/repo"},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}
