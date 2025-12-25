package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
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
		return fmt.Errorf("no repo configured. Run: cvx init")
	}

	cli := gh.New()

	// Check if issue exists and get title
	data, err := cli.IssueViewByStr(repo, issueNumber, []string{"number", "title"})
	if err != nil {
		return fmt.Errorf("issue #%s not found in %s", issueNumber, repo)
	}

	var issue struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &issue); err != nil {
		return fmt.Errorf("failed to parse issue: %w", err)
	}

	if err := cli.IssueDeleteByStr(repo, issueNumber); err != nil {
		return fmt.Errorf("gh issue delete failed: %w", err)
	}

	fmt.Printf("%s%s %s\n", style.Success("Deleted"), style.C(style.Cyan, "#"+issueNumber), issue.Title)
	return nil
}
