package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/style"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system setup for cvx build",
	Long:  `Verify all dependencies and configurations needed for cvx build.`,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Printf("%s Checking cvx setup\n\n", style.C(style.Blue, "→"))

	cfg, configOK := checkConfigFiles()
	_ = checkGitHub(cfg)
	checkPythonDeps()
	checkAgentCLI(cfg)

	fmt.Println()
	hasAPIKeys := checkAPICredentials()
	fmt.Println()

	if configOK && hasAPIKeys {
		fmt.Printf("%s Setup OK\n", style.C(style.Green, "✓"))
		return nil
	}

	if !configOK {
		return fmt.Errorf("setup issues detected")
	}

	return nil
}

func checkConfigFiles() (*config.Config, bool) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("%s cvx.toml not found or invalid\n", style.C(style.Red, "✗"))
		fmt.Printf("  Run: cvx init\n")
		return nil, false
	}

	fmt.Printf("%s cvx.toml configured\n", style.C(style.Green, "✓"))

	allGood := checkSourceFile(cfg.CV.Source, "src/cv.yaml", "CV")
	allGood = checkSourceFile(cfg.Letter.Source, "src/letter.yaml", "Letter") && allGood
	checkSchemaFile(cfg.CV.Schema, "schema/schema.json")

	return cfg, allGood
}

func checkSourceFile(source, defaultPath, label string) bool {
	if source == "" {
		source = defaultPath
	}
	if _, err := os.Stat(source); err != nil {
		fmt.Printf("%s %s source not found: %s\n", style.C(style.Red, "✗"), label, source)
		return false
	}
	fmt.Printf("%s %s source exists: %s\n", style.C(style.Green, "✓"), label, source)
	return true
}

func checkSchemaFile(schemaPath, defaultPath string) {
	if schemaPath == "" {
		schemaPath = defaultPath
	}
	if _, err := os.Stat(schemaPath); err != nil {
		fmt.Printf("%s Schema not found: %s\n", style.C(style.Yellow, "⚠"), schemaPath)
	} else {
		fmt.Printf("%s Schema exists: %s\n", style.C(style.Green, "✓"), schemaPath)
	}
}

func checkGitHub(cfg *config.Config) bool {
	ghInstalled := checkGHInstalled()
	checkGitHubRepo(cfg)
	checkGitHubProject(cfg, ghInstalled)
	return ghInstalled
}

func checkGHInstalled() bool {
	if _, err := exec.LookPath("gh"); err != nil {
		fmt.Printf("%s gh not installed (GitHub CLI)\n", style.C(style.Yellow, "⚠"))
		fmt.Printf("  Install: https://cli.github.com/\n")
		return false
	}
	fmt.Printf("%s gh installed\n", style.C(style.Green, "✓"))
	return true
}

func checkGitHubRepo(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if cfg.GitHub.Repo == "" {
		fmt.Printf("%s GitHub repo not configured\n", style.C(style.Yellow, "⚠"))
	} else {
		fmt.Printf("%s GitHub repo: %s\n", style.C(style.Green, "✓"), cfg.GitHub.Repo)
	}
}

func checkGitHubProject(cfg *config.Config, ghInstalled bool) {
	if cfg == nil {
		return
	}
	if cfg.GitHub.Project == "" {
		fmt.Printf("%s GitHub project not configured\n", style.C(style.Yellow, "⚠"))
		return
	}

	fmt.Printf("%s GitHub project: %s\n", style.C(style.Green, "✓"), cfg.GitHub.Project)

	if !ghInstalled {
		return
	}

	parts := strings.Split(cfg.GitHub.Project, "/")
	if len(parts) != 2 {
		return
	}

	checkCmd := exec.Command("gh", "project", "view", parts[1], "--owner", parts[0], "--format", "json")
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("%s GitHub project not accessible: %s\n", style.C(style.Yellow, "⚠"), cfg.GitHub.Project)
		fmt.Printf("  Check authentication: gh auth status\n")
	}
}

func checkPythonDeps() {
	if _, err := exec.LookPath("uv"); err != nil {
		fmt.Printf("%s uv not installed (required for -m flag)\n", style.C(style.Yellow, "⚠"))
		fmt.Printf("  Install: https://docs.astral.sh/uv/\n")
	} else {
		fmt.Printf("%s uv installed\n", style.C(style.Green, "✓"))
	}
}

func checkAgentCLI(cfg *config.Config) {
	if cfg == nil || cfg.Agent.Default == "" {
		return
	}

	cmdName := ""
	if strings.HasPrefix(cfg.Agent.Default, "claude") {
		cmdName = "claude"
	} else if strings.HasPrefix(cfg.Agent.Default, "gemini") {
		cmdName = "gemini"
	}

	if cmdName == "" {
		return
	}

	if _, err := exec.LookPath(cmdName); err != nil {
		fmt.Printf("%s %s CLI not found (for interactive mode)\n", style.C(style.Yellow, "⚠"), cmdName)
	} else {
		fmt.Printf("%s %s CLI available\n", style.C(style.Green, "✓"), cmdName)
	}
}

func checkAPICredentials() bool {
	fmt.Printf("%s Checking API credentials\n\n", style.C(style.Blue, "→"))

	hasAnthropicKey := checkEnvVar("ANTHROPIC_API_KEY", "required for Claude")
	hasGeminiKey := checkEnvVar("GEMINI_API_KEY", "required for Gemini")
	checkEnvVar("OPENAI_API_KEY", "optional, for GPT models")

	return hasAnthropicKey || hasGeminiKey
}

func checkEnvVar(envVar, description string) bool {
	if os.Getenv(envVar) != "" {
		fmt.Printf("%s %s set\n", style.C(style.Green, "✓"), envVar)
		return true
	}

	symbol := style.C(style.Yellow, "⚠")
	if strings.Contains(description, "optional") {
		symbol = style.C(style.Yellow, "○")
	}
	fmt.Printf("%s %s not set (%s)\n", symbol, envVar, description)
	return false
}
