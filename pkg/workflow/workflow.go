package workflow

import (
	_ "embed"
	"os"

	"github.com/xrsl/cvx/pkg/schema"
	"github.com/xrsl/cvx/pkg/utils"
)

//go:embed defaults/add.md
var DefaultAdd string

//go:embed defaults/advise.md
var DefaultAdvise string

//go:embed defaults/build.md
var DefaultBuild string

const (
	DefaultSchemaPath = ".github/ISSUE_TEMPLATE/job-ad-schema.yaml"
	AddPath           = ".cvx/workflows/add.md"
	AdvisePath        = ".cvx/workflows/advise.md"
	BuildPath         = ".cvx/workflows/build.md"
)

// Init creates .cvx/ directory structure with default workflow files.
// If schemaPath is empty, uses DefaultSchemaPath.
func Init(schemaPath string) error {
	if schemaPath == "" {
		schemaPath = DefaultSchemaPath
	}

	// Ensure .cvx/.gitignore exists
	if err := utils.EnsureCvxGitignore(); err != nil {
		return err
	}

	// Create directories
	dirs := []string{".cvx/workflows", ".cvx/sessions", ".cvx/matches", ".github/ISSUE_TEMPLATE"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// Create default job-ad-schema.yaml if it doesn't exist
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		if err := os.WriteFile(schemaPath, schema.DefaultSchemaYAML(), 0o644); err != nil {
			return err
		}
	}

	// Create default add.md if it doesn't exist
	if _, err := os.Stat(AddPath); os.IsNotExist(err) {
		if err := os.WriteFile(AddPath, []byte(DefaultAdd), 0o644); err != nil {
			return err
		}
	}

	// Create default advise.md if it doesn't exist
	if _, err := os.Stat(AdvisePath); os.IsNotExist(err) {
		if err := os.WriteFile(AdvisePath, []byte(DefaultAdvise), 0o644); err != nil {
			return err
		}
	}

	// Create default build.md if it doesn't exist
	if _, err := os.Stat(BuildPath); os.IsNotExist(err) {
		if err := os.WriteFile(BuildPath, []byte(DefaultBuild), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// LoadAdvise loads the advise workflow, falling back to embedded default
func LoadAdvise() (string, error) {
	content, err := os.ReadFile(AdvisePath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAdvise, nil
		}
		return "", err
	}
	return string(content), nil
}

// LoadBuild loads the build workflow, falling back to embedded default
func LoadBuild() (string, error) {
	content, err := os.ReadFile(BuildPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultBuild, nil
		}
		return "", err
	}
	return string(content), nil
}

// LoadAdd loads the add workflow, falling back to embedded default
func LoadAdd() (string, error) {
	content, err := os.ReadFile(AddPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAdd, nil
		}
		return "", err
	}
	return string(content), nil
}

// ResetWorkflows overwrites workflow files with defaults
func ResetWorkflows() error {
	// Ensure .cvx/.gitignore exists
	if err := utils.EnsureCvxGitignore(); err != nil {
		return err
	}

	if err := os.MkdirAll(".cvx/workflows", 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(AddPath, []byte(DefaultAdd), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(AdvisePath, []byte(DefaultAdvise), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(BuildPath, []byte(DefaultBuild), 0o644); err != nil {
		return err
	}
	return nil
}
