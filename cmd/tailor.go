package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
)

var tailorCmd = &cobra.Command{
	Use:   "tailor <issue-number>",
	Short: "Tailor CV and cover letter for a job",
	Long: `Tailor application materials for a job posting.

Prepares tailored CV and cover letter based on the job posting.
Use -i for interactive session with the AI agent.

Examples:
  cvx tailor 42                        # Prep tailored application
  cvx tailor 42 -m claude-sonnet-4     # Use Claude API
  cvx tailor 42 -i                     # Interactive session
  cvx tailor 42 -a gemini -i           # Interactive with Gemini CLI
  cvx tailor 42 -c "Emphasize Python"`,
	Args: cobra.ExactArgs(1),
	RunE: runTailor,
}

var (
	tailorAgentFlag       string
	tailorModelFlag       string
	tailorContextFlag     string
	tailorInteractiveFlag bool
)

func init() {
	tailorCmd.Flags().StringVarP(&tailorAgentFlag, "agent", "a", "", "CLI agent: claude, gemini")
	tailorCmd.Flags().StringVarP(&tailorModelFlag, "model", "m", "", "API model: claude-sonnet-4, gemini-2.5-flash, etc.")
	tailorCmd.Flags().StringVarP(&tailorContextFlag, "context", "c", "", "Additional context")
	tailorCmd.Flags().BoolVarP(&tailorInteractiveFlag, "interactive", "i", false, "Interactive session")
	tailorCmd.MarkFlagsMutuallyExclusive("agent", "model")
	rootCmd.AddCommand(tailorCmd)
}

func runTailor(cmd *cobra.Command, args []string) error {
	issueNum := args[0]

	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve agent/model (flags > config > default)
	var agentSetting string
	switch {
	case tailorAgentFlag != "":
		if !ai.IsCLIAgentSupported(tailorAgentFlag) {
			return fmt.Errorf("unsupported CLI agent: %s (supported: %v)", tailorAgentFlag, ai.SupportedCLIAgents())
		}
		agentSetting = tailorAgentFlag
	case tailorModelFlag != "":
		if !ai.IsModelSupported(tailorModelFlag) {
			return fmt.Errorf("unsupported model: %s (supported: %v)", tailorModelFlag, ai.SupportedModels())
		}
		agentSetting = tailorModelFlag
	case cfg.Agent != "":
		agentSetting = cfg.Agent
	default:
		agentSetting = ai.DefaultAgent()
	}

	// Validate final setting
	if !ai.IsAgentSupported(agentSetting) {
		return fmt.Errorf("unsupported agent/model: %s (supported: %v)", agentSetting, ai.SupportedAgents())
	}

	// Interactive mode requires CLI agent
	if tailorInteractiveFlag && !ai.IsAgentCLI(agentSetting) {
		return fmt.Errorf("interactive mode requires CLI agent (claude or gemini), got: %s", agentSetting)
	}

	// Override config agent for this run
	cfg.Agent = agentSetting

	// Ensure we're on the correct branch
	if err := ensureIssueBranch(cfg.Repo, issueNum); err != nil {
		return err
	}

	// Interactive mode
	if tailorInteractiveFlag {
		return runTailorInteractive(cfg, issueNum)
	}

	// Non-interactive mode (API or CLI)
	return runTailorNonInteractive(cmd.Context(), cfg, agentSetting, issueNum)
}

func runTailorInteractive(cfg *config.Config, issueNum string) error {
	agent := cfg.AgentCLI()

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
			fmt.Printf("%sissue #%s\n", style.Success("Session saved for"), issueNum)
		}
	}

	return nil
}

func runTailorNonInteractive(ctx context.Context, cfg *config.Config, agent, issueNum string) error {
	fmt.Printf("Tailoring application for issue #%s...\n", issueNum)

	// Fetch issue body
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	systemPrompt, userPrompt, err := buildTailorPromptParts(cfg, issueBody)
	if err != nil {
		return err
	}
	if tailorContextFlag != "" {
		userPrompt = fmt.Sprintf("%s\n\nAdditional context: %s", userPrompt, tailorContextFlag)
	}

	client, err := ai.NewClient(agent)
	if err != nil {
		return fmt.Errorf("error creating AI client: %w", err)
	}
	defer client.Close()

	// Start spinner
	done := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), "Tailoring application using 🤖 "+agent+"...")
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	var result string

	// Use caching if supported
	if cachingClient, ok := client.(ai.CachingClient); ok {
		result, err = cachingClient.GenerateContentWithSystem(ctx, systemPrompt, userPrompt)
	} else {
		prompt := systemPrompt + "\n\n" + userPrompt
		result, err = client.GenerateContent(ctx, prompt)
	}

	done <- true
	close(done)

	if err != nil {
		return err
	}

	fmt.Println("\nTailoring suggestions:")
	fmt.Println(result)

	return nil
}

// buildTailorPromptParts returns the prompt split for caching
func buildTailorPromptParts(cfg *config.Config, issueBody string) (system, user string, err error) {
	workflowContent, loadErr := workflow.LoadTailor()
	if loadErr != nil {
		err = fmt.Errorf("error loading workflow: %w", loadErr)
		return
	}

	tmpl, parseErr := template.New("tailor").Parse(workflowContent)
	if parseErr != nil {
		err = fmt.Errorf("error parsing workflow template: %w", parseErr)
		return
	}

	data := struct {
		CVPath        string
		ReferencePath string
	}{
		CVPath:        cfg.CVPath,
		ReferencePath: cfg.ReferencePath,
	}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		err = fmt.Errorf("error executing workflow template: %w", execErr)
		return
	}

	system = buf.String()
	user = fmt.Sprintf("## Job Posting\n%s", issueBody)
	return
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
		fmt.Printf("%s'%s'\n", style.Success("Switched to branch"), branchName)
	} else {
		// Create new branch from main
		gitCmd := exec.Command("git", "checkout", "-b", branchName, "main")
		if output, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error creating branch: %w\n%s", err, string(output))
		}
		fmt.Printf("%s'%s'\n", style.Success("Created branch"), branchName)
	}

	fmt.Printf("Issue #%s: %s at %s\n", issueNumber, title, company)
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
