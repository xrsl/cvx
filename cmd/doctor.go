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

	// Check 1: uv is installed
	if _, err := exec.LookPath("uv"); err != nil {
		fmt.Printf("%s uv is not installed\n", style.C(style.Red, "✗"))
		fmt.Printf("  Install: https://docs.astral.sh/uv/\n")
		allGood = false
	} else {
		fmt.Printf("%s uv installed\n", style.C(style.Green, "✓"))
	}

	// Check 2: Python available via uv
	cmd2 := exec.Command("uv", "python", "list")
	if err := cmd2.Run(); err != nil {
		fmt.Printf("%s Python not available via uv\n", style.C(style.Red, "✗"))
		fmt.Printf("  Fix: uv python install\n")
		allGood = false
	} else {
		fmt.Printf("%s Python available via uv\n", style.C(style.Green, "✓"))
	}

	// Check 3: cvx-agent runnable
	cmd3 := exec.Command("uv", "tool", "run", "--help")
	if err := cmd3.Run(); err != nil {
		fmt.Printf("%s uv tool run not working\n", style.C(style.Red, "✗"))
		allGood = false
	} else {
		fmt.Printf("%s uv tool run available\n", style.C(style.Green, "✓"))
	}

	// Check 4: Configured CLI agent availability
	cfg, err := config.Load()
	if err == nil && cfg.DefaultCLIAgent != "" {
		// Determine which command to check based on agent
		cmdName := ""
		if strings.HasPrefix(cfg.DefaultCLIAgent, "claude-code") {
			cmdName = "claude"
		} else if strings.HasPrefix(cfg.DefaultCLIAgent, "gemini-cli") {
			cmdName = "gemini"
		}

		if cmdName != "" {
			if _, err := exec.LookPath(cmdName); err != nil {
				fmt.Printf("%s %s not found in PATH\n", style.C(style.Red, "✗"), cfg.DefaultCLIAgent)
				fmt.Printf("  Install %s CLI to use interactive features\n", cmdName)
				allGood = false
			} else {
				fmt.Printf("%s %s available\n", style.C(style.Green, "✓"), cfg.DefaultCLIAgent)
			}
		}
	}

	fmt.Println()

	// Check environment variables
	fmt.Printf("%s Checking API credentials\n\n", style.C(style.Blue, "→"))

	hasAnthropicKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasGoogleKey := os.Getenv("GOOGLE_API_KEY") != ""
	hasOpenAIKey := os.Getenv("OPENAI_API_KEY") != ""

	if hasAnthropicKey {
		fmt.Printf("%s ANTHROPIC_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s ANTHROPIC_API_KEY not set (required for Claude)\n", style.C(style.Yellow, "⚠"))
	}

	if hasGoogleKey {
		fmt.Printf("%s GOOGLE_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s GOOGLE_API_KEY not set (required for Gemini)\n", style.C(style.Yellow, "⚠"))
	}

	if hasOpenAIKey {
		fmt.Printf("%s OPENAI_API_KEY set\n", style.C(style.Green, "✓"))
	} else {
		fmt.Printf("%s OPENAI_API_KEY not set (optional, for GPT models)\n", style.C(style.Yellow, "○"))
	}

	fmt.Println()

	if allGood && (hasAnthropicKey || hasGoogleKey) {
		fmt.Printf("%s Setup OK\n", style.C(style.Green, "✓"))
		return nil
	}

	if !allGood {
		return fmt.Errorf("setup issues detected")
	}

	// Warnings don't cause exit code
	return nil
}
