package utils

import (
	"os"
	"path/filepath"
)

func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// EnsureCvxGitignore creates .cvx/.gitignore to keep .cvx/ out of git
func EnsureCvxGitignore() error {
	gitignorePath := ".cvx/.gitignore"

	// Skip if already exists
	if FileExists(gitignorePath) {
		return nil
	}

	// Create .cvx directory if needed
	if err := os.MkdirAll(".cvx", 0o755); err != nil {
		return err
	}

	// Create .gitignore that ignores everything
	content := "*\n"
	return os.WriteFile(gitignorePath, []byte(content), 0o644)
}
