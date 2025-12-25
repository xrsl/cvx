package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/style"
)

// ensureIssueBranch checks if we're on the correct branch for the issue,
// and creates/switches to it if not
func ensureIssueBranch(repo, issueNumber string) error {
	// Get expected branch name
	branchName, company, title, err := getIssueBranchName(repo, issueNumber)
	if err != nil {
		return err
	}

	// Check current branch
	currentCmd := exec.Command("git", "branch", "--show-current")
	output, err := currentCmd.Output()
	if err != nil {
		return fmt.Errorf("error getting current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(string(output))

	// Already on correct branch
	if currentBranch == branchName {
		fmt.Printf("%s On branch %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, branchName))
		return nil
	}

	// Check if branch exists
	checkCmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if err := checkCmd.Run(); err == nil {
		// Branch exists, switch to it
		gitCmd := exec.Command("git", "checkout", branchName)
		if output, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error switching to branch: %w\n%s", err, string(output))
		}
		fmt.Printf("%s Switched to branch %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, branchName))
	} else {
		// Create new branch from main
		gitCmd := exec.Command("git", "checkout", "-b", branchName, "main")
		if output, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error creating branch: %w\n%s", err, string(output))
		}
		fmt.Printf("%s Created branch %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, branchName))
	}

	fmt.Printf("  Issue %s: %s @ %s\n", style.C(style.Cyan, "#"+issueNumber), title, style.C(style.Cyan, company))
	return nil
}

// getIssueBranchName fetches issue details and creates branch name
func getIssueBranchName(repo, issueNumber string) (branchName, company, title string, err error) {
	// Fetch issue details
	cli := gh.New()
	output, execErr := cli.IssueViewByStr(repo, issueNumber, []string{"title", "body"})
	if execErr != nil {
		err = fmt.Errorf("error fetching issue #%s: %w", issueNumber, execErr)
		return
	}

	var issue struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err = json.Unmarshal(output, &issue); err != nil {
		err = fmt.Errorf("error parsing issue: %w", err)
		return
	}

	// Extract company from body
	company = extractCompany(issue.Body)
	if company == "" {
		err = fmt.Errorf("could not extract company name from issue")
		return
	}

	// Create branch name: issue-number-company-role
	branchName = fmt.Sprintf("%s-%s-%s",
		issueNumber,
		sanitizeBranchName(company),
		sanitizeBranchName(issue.Title))

	title = issue.Title
	return
}

// sanitizeBranchName converts a string to a valid git branch name component
func sanitizeBranchName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, ".", "-")

	// Remove multiple consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return strings.Trim(s, "-")
}
