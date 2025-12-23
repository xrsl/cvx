package cmd

import (
	"bytes"
	"context"
	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/workflow"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

var (
	adviseContextFlag     string
	adviseInteractiveFlag bool
	advisePushFlag        bool
)

var adviseCmd = &cobra.Command{
	Use:   "advise <issue-number-or-url>",
	Short: "Get career advice on job match",
	Long: `Analyze job-CV match quality and get strategic career advice.

Takes a GitHub issue number or job posting URL and analyzes
how well your CV matches the position. Uses claude or gemini
CLI based on your agent setting.

Examples:
  cvx advise 42                    # Analyze issue #42
  cvx advise 42 --push             # Analyze and post as comment
  cvx advise https://example.com/job
  cvx advise 42 -c "Focus on backend"
  cvx advise 42 -i                 # Interactive session`,
	Args: cobra.ExactArgs(1),
	RunE: runAdvise,
}

func init() {
	adviseCmd.Flags().StringVarP(&adviseContextFlag, "context", "c", "", "Additional context for analysis")
	adviseCmd.Flags().BoolVarP(&adviseInteractiveFlag, "interactive", "i", false, "Join session interactively")
	adviseCmd.Flags().BoolVarP(&advisePushFlag, "push", "p", false, "Post analysis to GitHub issue")
	rootCmd.AddCommand(adviseCmd)
}

func runAdvise(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Load config
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Get CLI agent name from agent setting
	agent := cfg.AgentCLI()

	// Check if target is URL or issue number
	isURL := strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")

	if isURL {
		return runAdviseURL(cfg, agent, target)
	}
	return runAdviseIssue(cfg, agent, target)
}

func runAdviseURL(cfg *config.Config, agent, url string) error {
	if advisePushFlag {
		fmt.Println("Warning: --push flag is not supported for URL-based analysis")
		fmt.Println("Create an issue first, then run 'cvx advise <issue-number> --push'")
	}
	if adviseInteractiveFlag {
		fmt.Println("Warning: --interactive flag is not supported for URL-based analysis")
	}

	fmt.Printf("Running analysis for: %s\n", url)

	var result string

	// Use API client with caching when agent is not CLI-based
	if !ai.IsAgentCLI(cfg.Agent) {
		systemPrompt, userPrompt, err := buildAdvisePromptParts(cfg, url, "")
		if err != nil {
			return err
		}
		if adviseContextFlag != "" {
			userPrompt = fmt.Sprintf("%s\n\nAdditional context: %s", userPrompt, adviseContextFlag)
		}

		result, err = runAdviseWithAPI(cfg.Agent, systemPrompt, userPrompt)
		if err != nil {
			return err
		}
	} else {
		// CLI agent path
		prompt, err := buildAdvisePrompt(cfg, url, "")
		if err != nil {
			return err
		}
		if adviseContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, adviseContextFlag)
		}

		args := []string{"-p", prompt, "--output-format", "json"}
		output, err := runAgentWithSpinner(agent, args, "Analyzing job posting...")
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}

		var sessionID string
		result, sessionID = parseAgentOutput(output)

		// Clean up session (one-off analysis)
		if sessionID != "" {
			cleanupSession(agent, sessionID)
		}
	}

	// Display result
	fmt.Println("\nMatch Analysis:")
	fmt.Println(result)

	// Save to file
	filename := sanitizeFilename(url) + ".md"
	matchPath := filepath.Join(".cvx", "matches", filename)
	os.MkdirAll(filepath.Dir(matchPath), 0755)
	if err := os.WriteFile(matchPath, []byte(result), 0644); err != nil {
		fmt.Printf("Warning: Could not save analysis: %v\n", err)
	} else {
		fmt.Printf("\n%sAnalysis saved to %s%s\n", cGreen, matchPath, cReset)
	}

	return nil
}

// runAdviseWithAPI runs analysis using API client with caching
func runAdviseWithAPI(agent, systemPrompt, userPrompt string) (string, error) {
	client, err := ai.NewClient(agent)
	if err != nil {
		return "", fmt.Errorf("error creating AI client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Use caching if supported
	if cachingClient, ok := client.(ai.CachingClient); ok {
		return cachingClient.GenerateContentWithSystem(ctx, systemPrompt, userPrompt)
	}

	// Fall back to regular prompt
	prompt := systemPrompt + "\n\n" + userPrompt
	return client.GenerateContent(ctx, prompt)
}

func runAdviseIssue(cfg *config.Config, agent, issueNum string) error {
	sessionKey := issueNum
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")

	// Check for existing session
	sessionID, hasSession := getSession(sessionKey)

	// Handle --push flag
	if advisePushFlag {
		// Check if analysis exists
		content, err := os.ReadFile(matchPath)
		if err != nil && os.IsNotExist(err) {
			// Create analysis first
			fmt.Printf("No existing analysis found, creating new analysis for issue #%s...\n", issueNum)
			if err := runAdviseAnalysis(cfg, agent, issueNum, sessionKey, hasSession, sessionID); err != nil {
				return err
			}
			content, err = os.ReadFile(matchPath)
			if err != nil {
				return fmt.Errorf("error reading analysis: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("error reading %s: %w", matchPath, err)
		}

		// Post as comment
		fmt.Printf("Posting analysis to issue #%s...\n", issueNum)
		ghCmd := exec.Command("gh", "issue", "comment", issueNum,
			"--repo", cfg.Repo,
			"--body", string(content))
		if output, err := ghCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error posting comment: %w\nOutput: %s", err, string(output))
		}
		fmt.Printf("%sAnalysis posted as comment to issue #%s%s\n", cGreen, issueNum, cReset)
		return nil
	}

	// Interactive mode
	if adviseInteractiveFlag {
		return runAdviseInteractive(cfg, agent, issueNum, sessionKey, hasSession, sessionID)
	}

	// Normal analysis
	return runAdviseAnalysis(cfg, agent, issueNum, sessionKey, hasSession, sessionID)
}

func runAdviseAnalysis(cfg *config.Config, agent, issueNum, sessionKey string, hasSession bool, sessionID string) error {
	// Fetch issue body for context
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	var result string

	// Use API client with caching when agent is not CLI-based
	if !ai.IsAgentCLI(cfg.Agent) {
		fmt.Printf("Running analysis for issue #%s...\n", issueNum)

		systemPrompt, userPrompt, err := buildAdvisePromptParts(cfg, "", issueBody)
		if err != nil {
			return err
		}
		if adviseContextFlag != "" {
			userPrompt = fmt.Sprintf("%s\n\nAdditional context: %s", userPrompt, adviseContextFlag)
		}

		result, err = runAdviseWithAPI(cfg.Agent, systemPrompt, userPrompt)
		if err != nil {
			return err
		}
	} else {
		// CLI agent path with session support
		prompt, err := buildAdvisePrompt(cfg, "", issueBody)
		if err != nil {
			return err
		}

		if adviseContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, adviseContextFlag)
		}

		args := []string{"-p", prompt, "--output-format", "json"}
		if hasSession {
			args = append(args, "--resume", sessionID)
			if adviseContextFlag != "" {
				fmt.Printf("Resuming session for issue #%s with new context...\n", issueNum)
			} else {
				fmt.Printf("Resuming existing session for issue #%s...\n", issueNum)
			}
		} else {
			fmt.Printf("Running analysis for issue #%s...\n", issueNum)
		}

		output, err := runAgentWithSpinner(agent, args, "Analyzing job match...")
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}

		var newSessionID string
		result, newSessionID = parseAgentOutput(output)

		// Save session if new
		if !hasSession && newSessionID != "" {
			if err := saveSession(sessionKey, newSessionID); err != nil {
				fmt.Printf("Warning: Could not save session: %v\n", err)
			} else {
				fmt.Printf("%sSession saved. Use 'cvx advise %s -c \"context\"' or 'cvx advise %s -i' to continue.%s\n",
					cGreen, issueNum, issueNum, cReset)
			}
		}
	}

	// Save output
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")
	os.MkdirAll(filepath.Dir(matchPath), 0755)
	if err := os.WriteFile(matchPath, []byte(result), 0644); err != nil {
		fmt.Printf("Warning: Could not save analysis: %v\n", err)
	} else {
		fmt.Printf("%sAnalysis saved to %s%s\n", cGreen, matchPath, cReset)
	}

	fmt.Println("\nMatch Analysis:")
	fmt.Println(result)

	return nil
}

func runAdviseInteractive(cfg *config.Config, agent, issueNum, sessionKey string, hasSession bool, sessionID string) error {
	// Interactive mode requires CLI agent
	if !ai.IsAgentCLI(cfg.Agent) {
		fmt.Printf("Note: Interactive mode requires CLI agent (claude-cli or gemini-cli).\n")
		fmt.Printf("Running single analysis instead.\n\n")
		return runAdviseAnalysis(cfg, agent, issueNum, sessionKey, hasSession, sessionID)
	}

	var cmd *exec.Cmd

	if hasSession {
		fmt.Printf("Resuming interactive session for issue #%s...\n", issueNum)
		cmd = exec.Command(agent, "--resume", sessionID)
	} else {
		fmt.Printf("Starting new interactive session for issue #%s...\n", issueNum)

		// Fetch issue body
		issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
		if err != nil {
			return fmt.Errorf("error fetching issue: %w", err)
		}

		prompt, err := buildAdvisePrompt(cfg, "", issueBody)
		if err != nil {
			return err
		}

		if adviseContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, adviseContextFlag)
		}

		cmd = exec.Command(agent, "-p", prompt)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s: %w", agent, err)
	}

	// Save session if new
	if !hasSession {
		if newSessionID := getMostRecentAgentSession(agent); newSessionID != "" {
			saveSession(sessionKey, newSessionID)
		}
	}

	return nil
}

func buildAdvisePrompt(cfg *config.Config, url, issueBody string) (string, error) {
	system, user, err := buildAdvisePromptParts(cfg, url, issueBody)
	if err != nil {
		return "", err
	}
	return system + "\n\n" + user, nil
}

// buildAdvisePromptParts returns the prompt split for caching:
// - system: workflow template with paths (cacheable)
// - user: job content + context (variable)
func buildAdvisePromptParts(cfg *config.Config, url, issueBody string) (system, user string, err error) {
	// Load workflow
	workflowContent, err := workflow.LoadAdvise()
	if err != nil {
		return "", "", fmt.Errorf("error loading workflow: %w", err)
	}

	// Substitute config paths
	tmpl, err := template.New("match").Parse(workflowContent)
	if err != nil {
		return "", "", fmt.Errorf("error parsing workflow template: %w", err)
	}

	data := struct {
		CVPath         string
		ReferencePath string
	}{
		CVPath:         cfg.CVPath,
		ReferencePath: cfg.ReferencePath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("error executing workflow template: %w", err)
	}

	system = buf.String()

	// Build user part with job content
	if url != "" {
		user = fmt.Sprintf("## Job Posting URL\n%s", url)
	} else if issueBody != "" {
		user = fmt.Sprintf("## Job Posting Content\n%s", issueBody)
	}

	return system, user, nil
}

func fetchIssueBody(repo, issueNum string) (string, error) {
	cmd := exec.Command("gh", "issue", "view", issueNum, "--repo", repo, "--json", "body")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	return result.Body, nil
}

// Session management
func getSession(key string) (string, bool) {
	sessionFile := filepath.Join(".cvx", "sessions", key+".sid")
	content, err := os.ReadFile(sessionFile)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(content)), true
}

func saveSession(key, sessionID string) error {
	os.MkdirAll(filepath.Join(".cvx", "sessions"), 0755)
	sessionFile := filepath.Join(".cvx", "sessions", key+".sid")
	return os.WriteFile(sessionFile, []byte(sessionID), 0644)
}

func getMostRecentAgentSession(agent string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	slug := strings.ReplaceAll(wd, "/", "-")
	sessionDir := filepath.Join(home, "."+agent, "projects", slug)

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return ""
	}

	var mostRecentFile string
	var mostRecentTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(mostRecentTime) {
			mostRecentTime = info.ModTime()
			mostRecentFile = entry.Name()
		}
	}

	if mostRecentFile == "" {
		return ""
	}

	return strings.TrimSuffix(mostRecentFile, ".jsonl")
}

func cleanupSession(agent, sessionID string) {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	slug := strings.ReplaceAll(wd, "/", "-")
	sessionFile := filepath.Join(home, "."+agent, "projects", slug, sessionID+".jsonl")
	os.Remove(sessionFile)
}

// Agent execution
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func runAgentWithSpinner(agent string, args []string, message string) ([]byte, error) {
	cmd := exec.Command(agent, args...)

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
				fmt.Fprintf(os.Stderr, "\r%s%s%s %s", cCyan, spinnerFrames[i%len(spinnerFrames)], cReset, message)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	output, err := cmd.CombinedOutput()

	done <- true
	close(done)

	return output, err
}

func parseAgentOutput(output []byte) (result string, sessionID string) {
	// Try to extract JSON
	str := string(output)
	start := strings.Index(str, "{")
	end := strings.LastIndex(str, "}")

	if start == -1 || end == -1 || end <= start {
		return str, ""
	}

	jsonStr := str[start : end+1]

	var parsed struct {
		SessionID string `json:"session_id"`
		Result    string `json:"result"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return str, ""
	}

	return parsed.Result, parsed.SessionID
}

func sanitizeFilename(s string) string {
	// Remove protocol
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")

	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "",
		"?", "",
		"*", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
		" ", "-",
	)
	s = replacer.Replace(s)

	// Remove multiple consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim and limit length
	s = strings.Trim(s, "-")
	if len(s) > 100 {
		s = s[:100]
	}

	return s
}
