package cmd

import (
	"cvx/pkg/config"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	listState    string
	listLimit    int
	listRepoFlag string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List job applications",
	Long: `List job application issues from configured GitHub repository.

Examples:
  cvx list
  cvx list --state closed
  cvx list --state all --limit 50
  cvx list -r owner/repo`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		// Resolve repo (flag > config)
		repo := listRepoFlag
		if repo == "" {
			repo = cfg.Repo
		}
		if repo == "" {
			return fmt.Errorf("no repo configured. Run: cvx config set repo owner/name")
		}

		ghArgs := []string{"issue", "list", "-R", repo}

		if listState != "" {
			ghArgs = append(ghArgs, "--state", listState)
		}
		if listLimit > 0 {
			ghArgs = append(ghArgs, "--limit", fmt.Sprintf("%d", listLimit))
		}

		gh := exec.Command("gh", ghArgs...)
		gh.Stdout = os.Stdout
		gh.Stderr = os.Stderr

		if err := gh.Run(); err != nil {
			return fmt.Errorf("gh failed: %w", err)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listState, "state", "open", "Issue state (open|closed|all)")
	listCmd.Flags().IntVar(&listLimit, "limit", 30, "Max issues to list")
	listCmd.Flags().StringVarP(&listRepoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	rootCmd.AddCommand(listCmd)
}
