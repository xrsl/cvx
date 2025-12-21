package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	sch, err := Load("")
	if err != nil {
		t.Fatalf("Load('') error: %v", err)
	}

	if sch.Name == "" {
		t.Error("Default schema has no name")
	}

	if len(sch.Fields) == 0 {
		t.Error("Default schema has no fields")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp schema file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.yml")

	content := `name: Test Schema
description: Test description
body:
  - type: input
    id: company
    attributes:
      label: Company
  - type: input
    id: role
    attributes:
      label: Role
`
	if err := os.WriteFile(schemaPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test schema: %v", err)
	}

	sch, err := Load(schemaPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if sch.Name != "Test Schema" {
		t.Errorf("Expected name 'Test Schema', got '%s'", sch.Name)
	}

	if len(sch.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(sch.Fields))
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/path/schema.yml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestGeneratePrompt(t *testing.T) {
	sch, _ := Load("")

	prompt := sch.GeneratePrompt("https://example.com/job", "Software Engineer at Acme Corp")

	if !strings.Contains(prompt, "https://example.com/job") {
		t.Error("Prompt should contain URL")
	}

	if !strings.Contains(prompt, "Software Engineer") {
		t.Error("Prompt should contain job text")
	}

	if !strings.Contains(prompt, "JSON") {
		t.Error("Prompt should mention JSON format")
	}
}

func TestGetTitle(t *testing.T) {
	sch, _ := Load("")

	data := map[string]any{
		"company": "Acme Corp",
		"role":    "Software Engineer",
	}

	title := sch.GetTitle(data)

	if title == "" {
		t.Error("GetTitle returned empty string")
	}
}

func TestBuildIssueBody(t *testing.T) {
	sch, _ := Load("")

	data := map[string]any{
		"company":  "Acme Corp",
		"role":     "Software Engineer",
		"location": "Remote",
	}

	body := sch.BuildIssueBody(data)

	if body == "" {
		t.Error("BuildIssueBody returned empty string")
	}

	if !strings.Contains(body, "Acme Corp") {
		t.Error("Body should contain company")
	}
}

func TestSchemaFields(t *testing.T) {
	sch, _ := Load("")

	// Default schema should have common job fields
	fieldIDs := make(map[string]bool)
	for _, field := range sch.Fields {
		fieldIDs[field.ID] = true
	}

	expectedFields := []string{"company", "title", "location"}
	for _, f := range expectedFields {
		if !fieldIDs[f] {
			t.Errorf("Default schema should have '%s' field", f)
		}
	}
}
