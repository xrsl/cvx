package cmd

import (
	"bytes"
	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/workflow"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/spf13/cobra"
)

var tailorCmd = &cobra.Command{
	Use:   "tailor <issue-number>",
	Short: "Tailor CV and cover letter for a job",
	Long: `Tailor application materials for a job posting.

Starts an interactive session to tailor your CV and cover letter
based on the job posting. Resumes existing session if available.

Examples:
  cvx tailor 42                    # Tailor for issue #42
  cvx tailor 42 -c "Emphasize Python"`,
	Args: cobra.ExactArgs(1),
	RunE: runTailor,
}

var tailorContextFlag string

func init() {
	tailorCmd.Flags().StringVarP(&tailorContextFlag, "context", "c", "", "Additional context")
	rootCmd.AddCommand(tailorCmd)
}

func runTailor(cmd *cobra.Command, args []string) error {
	issueNum := args[0]

	cfg, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Tailor requires CLI for interactive feedback loop
	if !ai.IsAgentCLI(cfg.Agent) {
		fmt.Printf("Note: tailor requires CLI agent for interactive feedback.\n")
		fmt.Printf("Configure claude-cli or gemini-cli with 'cvx config set agent claude-cli'\n")
		return fmt.Errorf("tailor requires CLI agent")
	}

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

		execCmd = exec.Command(agent, "-p", prompt)
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
			saveSession(issueNum, newSessionID)
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
