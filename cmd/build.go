package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
)

var buildCmd = &cobra.Command{
	Use:   "build [issue-number]",
	Short: "Build tailored CV and cover letter",
	Long: `Build tailored application materials for a job posting.

Generates tailored CV and cover letter based on the job posting.
If issue-number is not provided, it will be inferred from the current branch name.

Examples:
  cvx build                           # Infer issue from branch
  cvx build 42                        # Build for issue #42
  cvx build -o                        # Build and open PDF
  cvx build -c "emphasize Python"     # Continue with feedback
  cvx build -i                        # Interactive session`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

var (
	buildAgentFlag       string
	buildModelFlag       string
	buildContextFlag     string
	buildInteractiveFlag bool
	buildOpenFlag        bool
)

func init() {
	buildCmd.Flags().StringVarP(&buildAgentFlag, "agent", "a", "", "CLI agent: claude, gemini")
	buildCmd.Flags().StringVarP(&buildModelFlag, "model", "m", "", "API model: claude-sonnet-4, gemini-2.5-flash, etc.")
	buildCmd.Flags().StringVarP(&buildContextFlag, "context", "c", "", "Feedback or additional context")
	buildCmd.Flags().BoolVarP(&buildInteractiveFlag, "interactive", "i", false, "Interactive session")
	buildCmd.Flags().BoolVarP(&buildOpenFlag, "open", "o", false, "Open combined.pdf in VSCode after build")
	buildCmd.MarkFlagsMutuallyExclusive("agent", "model")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Get issue number from args or infer from branch
	var issueNum string
	if len(args) > 0 {
		issueNum = args[0]
	} else {
		// Infer from current branch
		currentBranch, err := getCurrentBranch()
		if err != nil {
			return err
		}
		issueNum = extractIssueFromBranch(currentBranch)
		if issueNum == "" {
			return fmt.Errorf("could not infer issue number from branch '%s'. Provide it explicitly: cvx build <issue-number>", currentBranch)
		}
		fmt.Printf("Using issue #%s (from branch %s)\n", issueNum, currentBranch)
	}

	// Resolve agent/model (flags > config > default)
	var agentSetting string
	switch {
	case buildAgentFlag != "":
		if !ai.IsCLIAgentSupported(buildAgentFlag) {
			return fmt.Errorf("unsupported CLI agent: %s (supported: %v)", buildAgentFlag, ai.SupportedCLIAgents())
		}
		agentSetting = buildAgentFlag
	case buildModelFlag != "":
		if !ai.IsModelSupported(buildModelFlag) {
			return fmt.Errorf("unsupported model: %s (supported: %v)", buildModelFlag, ai.SupportedModels())
		}
		agentSetting = buildModelFlag
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
	if buildInteractiveFlag && !ai.IsAgentCLI(agentSetting) {
		return fmt.Errorf("interactive mode requires CLI agent (claude or gemini), got: %s", agentSetting)
	}

	// Override config agent for this run
	cfg.Agent = agentSetting

	// Ensure we're on the correct branch
	if err := ensureIssueBranch(cfg.Repo, issueNum); err != nil {
		return err
	}

	// Interactive mode
	if buildInteractiveFlag {
		return runBuildInteractive(cfg, issueNum)
	}

	// Non-interactive mode (API or CLI)
	if err := runBuildNonInteractive(cmd.Context(), cfg, agentSetting, issueNum); err != nil {
		return err
	}

	// Open PDF if requested
	if buildOpenFlag {
		return openCombinedPDF()
	}

	return nil
}

func runBuildInteractive(cfg *config.Config, issueNum string) error {
	agent := cfg.AgentCLI()

	// Use issue number as unified session key
	sessionID, hasSession := getSession(issueNum)

	var execCmd *exec.Cmd

	if hasSession {
		fmt.Printf("Resuming session for issue #%s...\n", issueNum)
		if buildContextFlag != "" {
			execCmd = exec.Command(agent, "--resume", sessionID, "-p", buildContextFlag)
		} else {
			execCmd = exec.Command(agent, "--resume", sessionID)
		}
	} else {
		fmt.Printf("Starting build session for issue #%s...\n", issueNum)

		// Fetch issue body
		issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
		if err != nil {
			return fmt.Errorf("error fetching issue: %w", err)
		}

		prompt, err := buildBuildPrompt(cfg, issueBody)
		if err != nil {
			return err
		}

		if buildContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, buildContextFlag)
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
			fmt.Printf("%sissue #%s\n", style.Success("Session saved for "), issueNum)
		}
	}

	// Open PDF if requested
	if buildOpenFlag {
		return openCombinedPDF()
	}

	return nil
}

func runBuildNonInteractive(ctx context.Context, cfg *config.Config, agent, issueNum string) error {
	// Fetch issue body
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	fmt.Printf("Building tailored application for issue #%s...\n", issueNum)

	// Path 1: CLI agent (headless) - claude/gemini handles tool use internally
	if ai.IsAgentCLI(agent) {
		return runBuildWithCLI(cfg, agent, issueBody)
	}

	// Path 2: API model - use structured output
	return runBuildWithAPI(ctx, cfg, agent, issueBody)
}

// runBuildWithCLI shells out to claude/gemini CLI in headless mode
func runBuildWithCLI(cfg *config.Config, agent, issueBody string) error {
	prompt, err := buildBuildPrompt(cfg, issueBody)
	if err != nil {
		return err
	}

	if buildContextFlag != "" {
		prompt = fmt.Sprintf("%s\n\nFeedback: %s", prompt, buildContextFlag)
	}

	var args []string
	var cliName string
	if agent == "gemini" || strings.HasPrefix(agent, "gemini:") {
		cliName = "gemini"
		args = []string{"-p", prompt}
	} else {
		cliName = "claude"
		args = []string{"-p", prompt, "--verbose"}
	}

	// Use shared spinner helper
	output, err := runAgentWithSpinner(cliName, args, "Building with 🤖 "+agent+"...")
	if err != nil {
		return fmt.Errorf("error running %s: %w", agent, err)
	}

	// Print output
	if len(output) > 0 {
		fmt.Println(string(output))
	}

	return nil
}

// runBuildWithAPI uses API with structured JSON output
func runBuildWithAPI(ctx context.Context, cfg *config.Config, agent, issueBody string) error {
	systemPrompt, userPrompt, err := buildBuildPromptParts(cfg, issueBody)
	if err != nil {
		return err
	}

	// Add structured output instruction
	structuredInstruction := `

IMPORTANT: You must respond with ONLY a valid JSON object in this exact format:
{"cv": "<full latex content for cv.tex>", "letter": "<full latex content for letter.tex>"}

Do not include any explanation, markdown, or text outside the JSON object.`

	userPrompt += structuredInstruction

	if buildContextFlag != "" {
		userPrompt = fmt.Sprintf("%s\n\nFeedback: %s", userPrompt, buildContextFlag)
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
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), "Building application using 🤖 "+agent+"...")
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

	// Parse structured output
	var output struct {
		CV     string `json:"cv"`
		Letter string `json:"letter"`
	}

	// Try to extract JSON from response (may have markdown code blocks)
	jsonStr := extractJSON(result)
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return fmt.Errorf("failed to parse AI response as JSON: %w\nResponse was: %s", err, result)
	}

	if output.CV == "" || output.Letter == "" {
		return fmt.Errorf("AI response missing cv or letter content")
	}

	// Write files
	cvPath := filepath.Join("src", "cv.tex")
	letterPath := filepath.Join("src", "letter.tex")

	if err := os.WriteFile(cvPath, []byte(output.CV), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", cvPath, err)
	}
	fmt.Printf("%s%s\n", style.Success("Wrote "), cvPath)

	if err := os.WriteFile(letterPath, []byte(output.Letter), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", letterPath, err)
	}
	fmt.Printf("%s%s\n", style.Success("Wrote "), letterPath)

	return nil
}

// extractJSON attempts to extract JSON from a response that may contain markdown
func extractJSON(s string) string {
	// Try to find JSON object directly
	s = strings.TrimSpace(s)

	// Remove markdown code blocks if present
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	}

	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		s = s[start : end+1]
	}

	return strings.TrimSpace(s)
}

// buildBuildPromptParts returns the prompt split for caching
func buildBuildPromptParts(cfg *config.Config, issueBody string) (system, user string, err error) {
	workflowContent, loadErr := workflow.LoadBuild()
	if loadErr != nil {
		err = fmt.Errorf("error loading workflow: %w", loadErr)
		return
	}

	tmpl, parseErr := template.New("build").Parse(workflowContent)
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

func buildBuildPrompt(cfg *config.Config, issueBody string) (string, error) {
	workflowContent, err := workflow.LoadBuild()
	if err != nil {
		return "", fmt.Errorf("error loading workflow: %w", err)
	}

	tmpl, err := template.New("build").Parse(workflowContent)
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

func openCombinedPDF() error {
	pdfPath := "build/combined.pdf"
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return fmt.Errorf("PDF not found at %s - run 'make combined' first", pdfPath)
	}

	cmd := exec.Command("code", pdfPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error opening PDF: %w", err)
	}

	fmt.Printf("%s%s\n", style.Success("Opened "), pdfPath)
	return nil
}
