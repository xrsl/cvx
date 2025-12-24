package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
)

var (
	viewLetterFlag bool
	viewCVFlag     bool
)

var viewCmd = &cobra.Command{
	Use:   "view <issue-number>",
	Short: "View submitted application documents",
	Long: `Open submitted CV or cover letter for a job application.

Finds the git tag for the issue and extracts the PDF documents
that were submitted with the application.

By default, opens the combined PDF (CV + letter) if available,
otherwise falls back to CV only.

Examples:
  cvx view 42              # Open combined or CV
  cvx view 42 -l           # Open cover letter
  cvx view 42 -c           # Open CV only`,
	Args: cobra.ExactArgs(1),
	RunE: runView,
}

func init() {
	viewCmd.Flags().BoolVarP(&viewLetterFlag, "letter", "l", false, "Open cover letter")
	viewCmd.Flags().BoolVarP(&viewCVFlag, "cv", "c", false, "Open CV only")
	rootCmd.AddCommand(viewCmd)
}

func runView(cmd *cobra.Command, args []string) error {
	issueNum := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Find tag for this issue
	gitCmd := exec.Command("git", "tag")
	output, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("error listing tags: %w", err)
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	var foundTag string
	prefix := issueNum + "-"

	for _, tag := range tags {
		if strings.HasPrefix(tag, prefix) {
			foundTag = tag
			break
		}
	}

	if foundTag == "" {
		return fmt.Errorf("no tag found for issue #%s (application not yet submitted?)", issueNum)
	}

	// Fetch issue details
	ghCmd := exec.Command("gh", "issue", "view", issueNum, "--repo", cfg.Repo, "--json", "title,body")
	output, err = ghCmd.Output()
	if err != nil {
		return fmt.Errorf("error fetching issue #%s: %w", issueNum, err)
	}

	var issue struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := json.Unmarshal(output, &issue); err != nil {
		return fmt.Errorf("error parsing issue: %w", err)
	}

	// Extract company from body
	company := extractCompany(issue.Body)

	// Determine document type and path
	var gitPath, tmpPath, docType string

	if viewLetterFlag {
		docType = "letter"
		gitPath = fmt.Sprintf("%s:build/letter.pdf", foundTag)
		tmpPath = fmt.Sprintf("/tmp/%s-letter.pdf", issueNum)
	} else if viewCVFlag {
		docType = "cv"
		gitPath = fmt.Sprintf("%s:build/cv.pdf", foundTag)
		tmpPath = fmt.Sprintf("/tmp/%s-cv.pdf", issueNum)
	} else {
		// Try combined first, fallback to cv
		gitPath = fmt.Sprintf("%s:build/combined.pdf", foundTag)
		tmpPath = fmt.Sprintf("/tmp/%s-combined.pdf", issueNum)
		docType = "combined"

		// Check if combined exists
		checkCmd := exec.Command("git", "show", gitPath)
		if _, err := checkCmd.Output(); err != nil {
			// Fallback to cv
			gitPath = fmt.Sprintf("%s:build/cv.pdf", foundTag)
			tmpPath = fmt.Sprintf("/tmp/%s-cv.pdf", issueNum)
			docType = "cv"
		}
	}

	// Extract PDF from git
	extractCmd := exec.Command("git", "show", gitPath)
	pdfData, err := extractCmd.Output()
	if err != nil {
		return fmt.Errorf("error extracting %s PDF: %w", docType, err)
	}

	if err := os.WriteFile(tmpPath, pdfData, 0644); err != nil {
		return fmt.Errorf("error writing PDF: %w", err)
	}

	// Open with VS Code
	openCmd := exec.Command("code", tmpPath)
	if err := openCmd.Run(); err != nil {
		return fmt.Errorf("error opening PDF: %w", err)
	}

	// Print formatted message
	if company != "" {
		fmt.Printf("Opened %s submitted to %s%s%s for %s%s%s\n",
			docType,
			cCyan, company, cReset,
			cGreen, issue.Title, cReset)
	} else {
		fmt.Printf("Opened %s submitted for %s%s%s\n",
			docType,
			cGreen, issue.Title, cReset)
	}

	return nil
}
