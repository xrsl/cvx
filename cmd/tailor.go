package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/workflow"
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
