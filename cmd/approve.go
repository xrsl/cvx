package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/style"
)

var approveCmd = &cobra.Command{
	Use:   "approve [issue-number]",
	Short: "Approve and finalize tailored application",
	Long: `Approve the tailored application and finalize it.

This command:
1. Commits changes with message "Tailored application for [Company] [Role]"
2. Creates a git tag: <issue>-<company>-<role>-YYYY-MM-DD
3. Pushes the tag to origin
4. Updates GitHub project status to "Applied"

If issue-number is not provided, it will be inferred from the current branch name.

Examples:
  cvx approve            # Infer issue from branch (e.g., 42-acme-corp-...)
  cvx approve 42         # Explicit issue number`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApprove,
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(cmd *cobra.Command, args []string) error {
	cfg, cache, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Get current branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	// Get issue number from args or infer from branch
	var issueNum string
	if len(args) > 0 {
		issueNum = args[0]
	} else {
		// Infer from branch name (format: <issue>-<company>-<role>)
		issueNum = extractIssueFromBranch(currentBranch)
		if issueNum == "" {
			return fmt.Errorf("could not infer issue number from branch '%s'. Provide it explicitly: cvx approve <issue-number>", currentBranch)
		}
		fmt.Printf("Using issue #%s (from branch %s)\n", issueNum, currentBranch)
	}

	// Check for uncommitted changes
	hasChanges, err := hasUncommittedChanges()
	if err != nil {
		return err
	}
	if !hasChanges {
		return fmt.Errorf("no uncommitted changes to approve")
	}

	// Get issue details for commit message and tag
	branchName, company, title, err := getIssueBranchName(cfg.Repo, issueNum)
	if err != nil {
		return err
	}

	// Warn if branch doesn't match expected
	if currentBranch != branchName {
		fmt.Printf("Warning: Current branch '%s' doesn't match expected '%s'\n", currentBranch, branchName)
	}

	// Build commit message and tag
	commitMsg := fmt.Sprintf("Tailored application for %s %s", company, title)
	today := time.Now().Format("2006-01-02")
	tagName := fmt.Sprintf("%s-%s-%s-%s", issueNum, sanitizeBranchName(company), sanitizeBranchName(title), today)

	fmt.Printf("Approving: %s\n", commitMsg)
	fmt.Printf("Tag: %s\n", tagName)

	// Stage all changes
	if err := gitAdd(); err != nil {
		return err
	}

	// Commit
	if err := gitCommit(commitMsg); err != nil {
		return err
	}
	fmt.Printf("%s%s\n", style.Success("Committed: "), commitMsg)

	// Create tag
	if err := gitTag(tagName); err != nil {
		return err
	}
	fmt.Printf("%s%s\n", style.Success("Created tag: "), tagName)

	// Push tag
	if err := gitPushTag(tagName); err != nil {
		return err
	}
	fmt.Printf("%s%s\n", style.Success("Pushed tag: "), tagName)

	// Update GitHub project status
	if cache != nil && cache.ID != "" {
		if err := updateProjectStatus(cfg, cache, issueNum); err != nil {
			fmt.Printf("Warning: Could not update project status: %v\n", err)
		} else {
			fmt.Printf("%s\n", style.Success("Updated project status to 'Applied'"))
		}
	}

	return nil
}

func hasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func gitAdd() error {
	cmd := exec.Command("git", "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w\n%s", err, string(output))
	}
	return nil
}

func gitCommit(msg string) error {
	cmd := exec.Command("git", "commit", "-m", msg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w\n%s", err, string(output))
	}
	return nil
}

func gitTag(tag string) error {
	cmd := exec.Command("git", "tag", tag)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git tag failed: %w\n%s", err, string(output))
	}
	return nil
}

func gitPushTag(tag string) error {
	cmd := exec.Command("git", "push", "origin", tag)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push tag failed: %w\n%s", err, string(output))
	}
	return nil
}

func updateProjectStatus(cfg *config.Config, cache *config.ProjectCache, issueNum string) error {
	cli := gh.New()

	// Get project item ID for the issue
	query := fmt.Sprintf(`{
		repository(owner: "%s", name: "%s") {
			issue(number: %s) {
				projectItems(first: 1) {
					nodes { id }
				}
			}
		}
	}`, getRepoOwner(cfg.Repo), getRepoName(cfg.Repo), issueNum)

	output, err := cli.GraphQL(query)
	if err != nil {
		return fmt.Errorf("failed to get project item: %w", err)
	}

	var result struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID string `json:"id"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse project item response: %w", err)
	}

	if len(result.Data.Repository.Issue.ProjectItems.Nodes) == 0 {
		return fmt.Errorf("issue not linked to any project")
	}

	itemID := result.Data.Repository.Issue.ProjectItems.Nodes[0].ID

	// Get "Applied" status option ID
	appliedStatusID, ok := cache.Statuses["Applied"]
	if !ok {
		return fmt.Errorf("'Applied' status not found in cache")
	}

	// Update status field
	mutation := fmt.Sprintf(`mutation {
		updateProjectV2ItemFieldValue(
			input: {
				projectId: "%s"
				itemId: "%s"
				fieldId: "%s"
				value: { singleSelectOptionId: "%s" }
			}
		) {
			projectV2Item { id }
		}
	}`, cache.ID, itemID, cache.Fields.Status, appliedStatusID)

	if _, err := cli.GraphQL(mutation); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Update AppliedDate field if available
	if cache.Fields.AppliedDate != "" {
		today := time.Now().Format("2006-01-02")
		dateMutation := fmt.Sprintf(`mutation {
			updateProjectV2ItemFieldValue(
				input: {
					projectId: "%s"
					itemId: "%s"
					fieldId: "%s"
					value: { date: "%s" }
				}
			) {
				projectV2Item { id }
			}
		}`, cache.ID, itemID, cache.Fields.AppliedDate, today)

		if _, err := cli.GraphQL(dateMutation); err != nil {
			// Non-fatal, just log
			fmt.Printf("Warning: Could not update AppliedDate: %v\n", err)
		}
	}

	return nil
}

func getRepoOwner(repo string) string {
	parts := strings.Split(repo, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func getRepoName(repo string) string {
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// extractIssueFromBranch extracts issue number from branch name
// Branch format: <issue>-<company>-<role> (e.g., "42-acme-corp-senior-engineer")
func extractIssueFromBranch(branch string) string {
	if branch == "" || branch == "main" || branch == "master" {
		return ""
	}

	// Find the first segment before a dash that is all digits
	idx := strings.Index(branch, "-")
	if idx == -1 {
		// No dash, check if entire branch is a number
		if isAllDigits(branch) {
			return branch
		}
		return ""
	}

	prefix := branch[:idx]
	if isAllDigits(prefix) {
		return prefix
	}

	return ""
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
