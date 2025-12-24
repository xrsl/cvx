package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/workflow"
)

var tailorCmd = &cobra.Command{
	Use:   "tailor <issue-number>",
	Short: "Tailor CV and cover letter for a job",
	Long: `Tailor application materials for a job posting.

Starts an interactive session to tailor your CV and cover letter
based on the job posting. Resumes existing session if available.

Examples:
  cvx tailor 42                    # Tailor for issue #42
  cvx tailor 42 -a gemini          # Use Gemini CLI
  cvx tailor 42 -c "Emphasize Python"`,
	Args: cobra.ExactArgs(1),
	RunE: runTailor,
}

var (
	tailorAgentFlag   string
	tailorContextFlag string
)

func init() {
	tailorCmd.Flags().StringVarP(&tailorAgentFlag, "agent", "a", "", "AI agent (overrides config)")
	tailorCmd.Flags().StringVarP(&tailorContextFlag, "context", "c", "", "Additional context")
	rootCmd.AddCommand(tailorCmd)
}

func runTailor(cmd *cobra.Command, args []string) error {
	issueNum := args[0]

	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve agent (flag > config > default)
	agentSetting := tailorAgentFlag
	if agentSetting == "" {
		agentSetting = cfg.Agent
	}
	if agentSetting == "" {
		agentSetting = ai.DefaultAgent()
	}

	// Validate agent
	if !ai.IsAgentSupported(agentSetting) {
		return fmt.Errorf("unsupported agent: %s (supported: %v)", agentSetting, ai.SupportedAgents())
	}

	// Tailor requires CLI for interactive feedback loop
	if !ai.IsAgentCLI(agentSetting) {
		fmt.Printf("Note: tailor requires CLI agent for interactive feedback.\n")
		fmt.Printf("Use -a claude or -a gemini, or configure in .cvx-config.yaml\n")
		return fmt.Errorf("tailor requires CLI agent (claude or gemini)")
	}

	// Override config agent for this run
	cfg.Agent = agentSetting
	agent := cfg.AgentCLI()

	// Ensure we're on the correct branch
	if err := ensureIssueBranch(cfg.Repo, issueNum); err != nil {
		return err
	}

	// Use issue number as unified session key
	sessionID, hasSession := getSession(issueNum)

	var execCmd *exec.Cmd

	if hasSession {
		fmt.Printf("Resuming session for issue #%s...\n", issueNum)
		if tailorContextFlag != "" {
			execCmd = exec.Command(agent, "--resume", sessionID, "-p", tailorContextFlag)
		} else {
			execCmd = exec.Command(agent, "--resume", sessionID)
		}
	} else {
		fmt.Printf("Starting tailor session for issue #%s...\n", issueNum)

		// Fetch issue body
		issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
		if err != nil {
			return fmt.Errorf("error fetching issue: %w", err)
		}

		prompt, err := buildTailorPrompt(cfg, issueBody)
		if err != nil {
			return err
		}

		if tailorContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, tailorContextFlag)
		}

		// Use -i for gemini (prompt-interactive), -p for claude
		if agent == "gemini" || strings.HasPrefix(agent, "gemini:") {
			execCmd = exec.Command("gemini", "-i", prompt)
		} else {
			execCmd = exec.Command("claude", "-p", prompt)
		}
	}

	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("error running %s: %w", agent, err)
	}

	// Save session if new
	if !hasSession {
		if newSessionID := getMostRecentAgentSession(agent); newSessionID != "" {
			_ = saveSession(issueNum, newSessionID)
			fmt.Printf("%sSession saved for issue #%s%s\n", cGreen, issueNum, cReset)
		}
	}

	return nil
}

func buildTailorPrompt(cfg *config.Config, issueBody string) (string, error) {
	workflowContent, err := workflow.LoadTailor()
	if err != nil {
		return "", fmt.Errorf("error loading workflow: %w", err)
	}

	tmpl, err := template.New("tailor").Parse(workflowContent)
	if err != nil {
		return "", fmt.Errorf("error parsing workflow template: %w", err)
	}

	data := struct {
		CVPath        string
		ReferencePath string
	}{
		CVPath:        cfg.CVPath,
		ReferencePath: cfg.ReferencePath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing workflow template: %w", err)
	}

	return fmt.Sprintf("%s\n\n## Job Posting\n%s", buf.String(), issueBody), nil
}

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
		fmt.Printf("On branch %s\n", branchName)
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
		fmt.Printf("%sSwitched to branch '%s'%s\n", cGreen, branchName, cReset)
	} else {
		// Create new branch from main
		gitCmd := exec.Command("git", "checkout", "-b", branchName, "main")
		if output, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error creating branch: %w\n%s", err, string(output))
		}
		fmt.Printf("%sCreated branch '%s'%s\n", cGreen, branchName, cReset)
	}

	fmt.Printf("Issue #%s: %s at %s\n", issueNumber, title, company)
	return nil
}

// getIssueBranchName fetches issue details and creates branch name
func getIssueBranchName(repo, issueNumber string) (branchName, company, title string, err error) {
	// Fetch issue details
	ghCmd := exec.Command("gh", "issue", "view", issueNumber, "--repo", repo, "--json", "title,body")
	output, execErr := ghCmd.Output()
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
