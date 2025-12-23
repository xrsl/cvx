package workflow

import (
	"cvx/pkg/schema"
	_ "embed"
	"os"
)

//go:embed defaults/advise.md
var DefaultAdvise string

//go:embed defaults/tailor.md
var DefaultTailor string

const (
	DefaultSchemaPath = ".github/ISSUE_TEMPLATE/job-ad-schema.yaml"
	AdvisePath        = ".cvx/workflows/advise.md"
	TailorPath        = ".cvx/workflows/tailor.md"
)

// Init creates .cvx/ directory structure with default workflow files.
// If schemaPath is empty, uses DefaultSchemaPath.
func Init(schemaPath string) error {
	if schemaPath == "" {
		schemaPath = DefaultSchemaPath
	}

	// Create directories
	dirs := []string{".cvx/workflows", ".cvx/sessions", ".cvx/matches", ".github/ISSUE_TEMPLATE"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create default job-ad-schema.yaml if it doesn't exist
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		if err := os.WriteFile(schemaPath, schema.DefaultSchemaYAML(), 0644); err != nil {
			return err
		}
	}

	// Create default advise.md if it doesn't exist
	if _, err := os.Stat(AdvisePath); os.IsNotExist(err) {
		if err := os.WriteFile(AdvisePath, []byte(DefaultAdvise), 0644); err != nil {
			return err
		}
	}

	// Create default tailor.md if it doesn't exist
	if _, err := os.Stat(TailorPath); os.IsNotExist(err) {
		if err := os.WriteFile(TailorPath, []byte(DefaultTailor), 0644); err != nil {
			return err
		}
	}

	return nil
}

// LoadAdvise loads the advise workflow from .cvx/workflows/advise.md
func LoadAdvise() (string, error) {
	content, err := os.ReadFile(AdvisePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// LoadTailor loads the tailor workflow from .cvx/workflows/tailor.md
func LoadTailor() (string, error) {
	content, err := os.ReadFile(TailorPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ResetWorkflows overwrites workflow files with defaults
func ResetWorkflows() error {
	os.MkdirAll(".cvx/workflows", 0755)
	if err := os.WriteFile(AdvisePath, []byte(DefaultAdvise), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(TailorPath, []byte(DefaultTailor), 0644); err != nil {
		return err
	}
	return nil
}
