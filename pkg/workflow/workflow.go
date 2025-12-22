package workflow

import (
	"cvx/pkg/schema"
	_ "embed"
	"os"
)

//go:embed defaults/match.md
var DefaultMatch string

const (
	DefaultSchemaPath = ".github/ISSUE_TEMPLATE/job-ad-schema.yaml"
	MatchPath         = ".cvx/workflows/match.md"
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

	// Create default match.md if it doesn't exist
	if _, err := os.Stat(MatchPath); os.IsNotExist(err) {
		if err := os.WriteFile(MatchPath, []byte(DefaultMatch), 0644); err != nil {
			return err
		}
	}

	return nil
}

// LoadMatch loads the match workflow from .cvx/workflows/match.md
func LoadMatch() (string, error) {
	content, err := os.ReadFile(".cvx/workflows/match.md")
	if err != nil {
		return "", err
	}
	return string(content), nil
}
