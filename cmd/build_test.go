package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadYAMLCV(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	cvPath := filepath.Join(tmpDir, "cv.yaml")

	cvContent := `cv:
  name: John Doe
  email: john@example.com
  phone: "+1234567890"
  sections:
    experience:
      - company: Tech Corp
        position: Software Engineer
        start_date: "2020-01"
        end_date: present
`

	if err := os.WriteFile(cvPath, []byte(cvContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading
	cv, err := readYAMLCV(cvPath)
	if err != nil {
		t.Fatalf("readYAMLCV failed: %v", err)
	}

	// Validate structure
	if cv["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", cv["name"])
	}
	if cv["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", cv["email"])
	}

	// Check nested sections
	sections, ok := cv["sections"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected sections to be a map")
	}
	if sections["experience"] == nil {
		t.Errorf("Expected experience section to exist")
	}
}

func TestReadYAMLLetter(t *testing.T) {
	tmpDir := t.TempDir()
	letterPath := filepath.Join(tmpDir, "letter.yaml")

	letterContent := `letter:
  sender:
    name: Jane Doe
    email: jane@example.com
    phone: "+9876543210"
  recipient:
    name: Hiring Manager
    company: Target Corp
  content:
    salutation: Dear Hiring Manager
    opening: I am writing to apply...
    closing: Sincerely
  metadata:
    date: auto
`

	if err := os.WriteFile(letterPath, []byte(letterContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	letter, err := readYAMLLetter(letterPath)
	if err != nil {
		t.Fatalf("readYAMLLetter failed: %v", err)
	}

	// Validate sender
	sender, ok := letter["sender"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected sender to be a map")
	}
	if sender["name"] != "Jane Doe" {
		t.Errorf("Expected sender name 'Jane Doe', got %v", sender["name"])
	}

	// Validate recipient
	recipient, ok := letter["recipient"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected recipient to be a map")
	}
	if recipient["company"] != "Target Corp" {
		t.Errorf("Expected company 'Target Corp', got %v", recipient["company"])
	}
}

func TestWriteYAMLCV(t *testing.T) {
	tmpDir := t.TempDir()
	cvPath := filepath.Join(tmpDir, "cv.yaml")

	// Create test CV data
	cv := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
		"sections": map[string]interface{}{
			"experience": []interface{}{
				map[string]interface{}{
					"company":  "Test Corp",
					"position": "Engineer",
				},
			},
		},
	}

	// Write CV
	if err := writeYAMLCV(cvPath, cv, ""); err != nil {
		t.Fatalf("writeYAMLCV failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cvPath); os.IsNotExist(err) {
		t.Fatalf("CV file was not created")
	}

	// Read it back and verify
	cvRead, err := readYAMLCV(cvPath)
	if err != nil {
		t.Fatalf("Failed to read written CV: %v", err)
	}

	if cvRead["name"] != "Test User" {
		t.Errorf("Expected name 'Test User', got %v", cvRead["name"])
	}
}

func TestWriteYAMLLetter(t *testing.T) {
	tmpDir := t.TempDir()
	letterPath := filepath.Join(tmpDir, "letter.yaml")

	letter := map[string]interface{}{
		"sender": map[string]interface{}{
			"name":  "Test Sender",
			"email": "sender@example.com",
		},
		"recipient": map[string]interface{}{
			"name":    "Test Recipient",
			"company": "Test Company",
		},
		"content": map[string]interface{}{
			"salutation": "Dear Test",
			"opening":    "Test opening",
			"closing":    "Test closing",
		},
		"metadata": map[string]interface{}{
			"date": "auto",
		},
	}

	if err := writeYAMLLetter(letterPath, letter, ""); err != nil {
		t.Fatalf("writeYAMLLetter failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(letterPath); os.IsNotExist(err) {
		t.Fatalf("Letter file was not created")
	}

	// Read it back
	letterRead, err := readYAMLLetter(letterPath)
	if err != nil {
		t.Fatalf("Failed to read written letter: %v", err)
	}

	sender, ok := letterRead["sender"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected sender to be a map")
	}
	if sender["name"] != "Test Sender" {
		t.Errorf("Expected sender name 'Test Sender', got %v", sender["name"])
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cvPath := filepath.Join(tmpDir, "cv.yaml")

	// Original data
	original := map[string]interface{}{
		"name":  "Round Trip",
		"email": "roundtrip@example.com",
		"sections": map[string]interface{}{
			"skills": []interface{}{
				map[string]interface{}{
					"label":   "Programming",
					"details": "Go, Python, JavaScript",
				},
			},
		},
	}

	// Write
	if err := writeYAMLCV(cvPath, original, ""); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	readBack, err := readYAMLCV(cvPath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Compare
	if readBack["name"] != original["name"] {
		t.Errorf("Name mismatch: expected %v, got %v", original["name"], readBack["name"])
	}
	if readBack["email"] != original["email"] {
		t.Errorf("Email mismatch: expected %v, got %v", original["email"], readBack["email"])
	}
}

func TestReadYAMLCV_FileNotFound(t *testing.T) {
	_, err := readYAMLCV("/nonexistent/path/cv.yaml")
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got nil")
	}
}

func TestReadYAMLCV_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cvPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `cv:
  this is: not
  valid yaml: [
`

	if err := os.WriteFile(cvPath, []byte(invalidContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := readYAMLCV(cvPath)
	if err == nil {
		t.Errorf("Expected error for invalid YAML, got nil")
	}
}

func TestReadYAMLCV_MissingCVField(t *testing.T) {
	tmpDir := t.TempDir()
	cvPath := filepath.Join(tmpDir, "no-cv-field.yaml")

	content := `name: Test
email: test@example.com
`

	if err := os.WriteFile(cvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cv, err := readYAMLCV(cvPath)
	if err != nil {
		t.Fatalf("Should not error on missing cv field: %v", err)
	}

	// Should return nil/empty map for missing cv field
	if len(cv) > 0 {
		t.Logf("Got CV data even without cv field wrapper: %v", cv)
	}
}
