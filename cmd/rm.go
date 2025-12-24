package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/style"
)

var rmRepoFlag string

var rmCmd = &cobra.Command{
	Use:   "rm <issue-number>",
	Short: "Remove a job application",
	Long: `Delete a GitHub issue by its number.

Examples:
  cvx rm 123
  cvx rm 123 -r owner/repo`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

func init() {
	rmCmd.Flags().StringVarP(&rmRepoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	issueNumber := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve repo (flag > config)
	repo := rmRepoFlag
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" {
		return fmt.Errorf("no repo configured. Run: cvx config set repo owner/name")
	}

	// Check if issue exists and get title
	check := exec.Command("gh", "issue", "view", issueNumber, "-R", repo, "--json", "number,title", "-q", ".title")
	titleOut, err := check.Output()
	if err != nil {
		return fmt.Errorf("issue #%s not found in %s", issueNumber, repo)
	}
	title := string(titleOut)

	gh := exec.Command("gh", "issue", "delete", issueNumber, "-R", repo, "--yes")
	if err := gh.Run(); err != nil {
		return fmt.Errorf("gh issue delete failed: %w", err)
	}

	fmt.Printf("%s%s %s\n", style.Success("Deleted"), style.C(style.Cyan, "#"+issueNumber), title)
	return nil
}
