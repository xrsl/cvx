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

	allGood := true

	// Check 1: cvx.toml exists
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("%s cvx.toml not found or invalid\n", style.C(style.Red, "✗"))
		fmt.Printf("  Run: cvx init\n")
		allGood = false
	} else {
		fmt.Printf("%s cvx.toml configured\n", style.C(style.Green, "✓"))

		// Check 2: CV source file exists
		cvSource := cfg.CV.Source
		if cvSource == "" {
			cvSource = "src/cv.yaml"
		}
		if _, err := os.Stat(cvSource); err != nil {
			fmt.Printf("%s CV source not found: %s\n", style.C(style.Red, "✗"), cvSource)
			allGood = false
		} else {
			fmt.Printf("%s CV source exists: %s\n", style.C(style.Green, "✓"), cvSource)
		}

		// Check 3: Letter source file exists
		letterSource := cfg.Letter.Source
		if letterSource == "" {
			letterSource = "src/letter.yaml"
		}
		if _, err := os.Stat(letterSource); err != nil {
			fmt.Printf("%s Letter source not found: %s\n", style.C(style.Red, "✗"), letterSource)
			allGood = false
		} else {
			fmt.Printf("%s Letter source exists: %s\n", style.C(style.Green, "✓"), letterSource)
		}

		// Check 4: Schema file exists
		schemaPath := cfg.CV.Schema
		if schemaPath == "" {
			schemaPath = "schema/schema.json"
		}
		if _, err := os.Stat(schemaPath); err != nil {
			fmt.Printf("%s Schema not found: %s\n", style.C(style.Yellow, "⚠"), schemaPath)
		} else {
			fmt.Printf("%s Schema exists: %s\n", style.C(style.Green, "✓"), schemaPath)
		}

		// Check 5: GitHub repo configured
		if cfg.GitHub.Repo == "" {
			fmt.Printf("%s GitHub repo not configured\n", style.C(style.Yellow, "⚠"))
		} else {
			fmt.Printf("%s GitHub repo: %s\n", style.C(style.Green, "✓"), cfg.GitHub.Repo)
		}

		// Check 6: Agent configured
		if cfg.Agent.Default == "" {
			fmt.Printf("%s No default agent configured\n", style.C(style.Yellow, "⚠"))
		} else {
			// Check if CLI agent is available
			cmdName := ""
			if strings.HasPrefix(cfg.Agent.Default, "claude") {
				cmdName = "claude"
			} else if strings.HasPrefix(cfg.Agent.Default, "gemini") {
				cmdName = "gemini"
			}

			if cmdName != "" {
				if _, err := exec.LookPath(cmdName); err != nil {
					fmt.Printf("%s %s CLI not found (for interactive mode)\n", style.C(style.Yellow, "⚠"), cmdName)
				} else {
					fmt.Printf("%s %s CLI available\n", style.C(style.Green, "✓"), cmdName)
				}
			}
		}
	}

	// Check 7: uv installed (required for Python agent mode)
	if _, err := exec.LookPath("uv"); err != nil {
		fmt.Printf("%s uv not installed (required for -m flag)\n", style.C(style.Yellow, "⚠"))
		fmt.Printf("  Install: https://docs.astral.sh/uv/\n")
	} else {
		fmt.Printf("%s uv installed\n", style.C(style.Green, "✓"))
	}

	fmt.Println()

	// Check environment variables
	fmt.Printf("%s Checking API credentials\n\n", style.C(style.Blue, "→"))

	hasAnthropicKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasGeminiKey := os.Getenv("GEMINI_API_KEY") != ""
	hasOpenAIKey := os.Getenv("OPENAI_API_KEY") != ""

	if hasAnthropicKey {
		fmt.Printf("%s ANTHROPIC_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s ANTHROPIC_API_KEY not set (required for Claude)\n", style.C(style.Yellow, "⚠"))
	}

	if hasGeminiKey {
		fmt.Printf("%s GEMINI_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s GEMINI_API_KEY not set (required for Gemini)\n", style.C(style.Yellow, "⚠"))
	}

	if hasOpenAIKey {
		fmt.Printf("%s OPENAI_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s OPENAI_API_KEY not set (optional, for GPT models)\n", style.C(style.Yellow, "○"))
	}

	fmt.Println()

	if allGood && (hasAnthropicKey || hasGeminiKey) {
		fmt.Printf("%s Setup OK\n", style.C(style.Green, "✓"))
		return nil
	}

	if !allGood {
		return fmt.Errorf("setup issues detected")
	}

	// Warnings don't cause exit code
	return nil
}
