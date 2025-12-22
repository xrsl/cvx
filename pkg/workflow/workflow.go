package workflow

import (
	_ "embed"
	"os"
)

//go:embed defaults/match.md
var DefaultMatch string

// Init creates .cvx/ directory structure with default workflow files
func Init() error {
	// Create directories
	dirs := []string{".cvx/workflows", ".cvx/sessions", ".cvx/matches"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create default match.md if it doesn't exist
	matchPath := ".cvx/workflows/match.md"
	if _, err := os.Stat(matchPath); os.IsNotExist(err) {
		if err := os.WriteFile(matchPath, []byte(DefaultMatch), 0644); err != nil {
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
