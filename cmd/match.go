package cmd

import (
	"bytes"
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
	matchContextFlag     string
	matchInteractiveFlag bool
	matchPushFlag        bool
)

var matchCmd = &cobra.Command{
	Use:   "match <issue-number-or-url>",
	Short: "Run job match analysis",
	Long: `Analyze job-CV match quality using AI.

Takes a GitHub issue number or job posting URL and analyzes
how well your CV matches the position. Uses claude or gemini
CLI based on your agent setting.

Examples:
  cvx match 42                    # Analyze issue #42
  cvx match 42 --push             # Analyze and post as comment
  cvx match https://example.com/job
  cvx match 42 -c "Focus on backend"
  cvx match 42 -i                 # Interactive session`,
	Args: cobra.ExactArgs(1),
	RunE: runMatch,
}

func init() {
	matchCmd.Flags().StringVarP(&matchContextFlag, "context", "c", "", "Additional context for analysis")
	matchCmd.Flags().BoolVarP(&matchInteractiveFlag, "interactive", "i", false, "Join session interactively")
	matchCmd.Flags().BoolVarP(&matchPushFlag, "push", "p", false, "Post analysis to GitHub issue")
	rootCmd.AddCommand(matchCmd)
}

func runMatch(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Load config with cached project IDs
	cfg, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Get CLI agent name from agent setting
	agent := cfg.AgentCLI()

	// Check if target is URL or issue number
	isURL := strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")

	if isURL {
		return runMatchURL(cfg, agent, target)
	}
	return runMatchIssue(cfg, agent, target)
}

func runMatchURL(cfg *config.Config, agent, url string) error {
	if matchPushFlag {
		fmt.Println("Warning: --push flag is not supported for URL-based analysis")
		fmt.Println("Create an issue first, then run 'cvx match <issue-number> --push'")
	}
	if matchInteractiveFlag {
		fmt.Println("Warning: --interactive flag is not supported for URL-based analysis")
	}

	// Build prompt
	prompt, err := buildMatchPrompt(cfg, url, "")
	if err != nil {
		return err
	}

	if matchContextFlag != "" {
		prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, matchContextFlag)
	}

	fmt.Printf("Running match analysis for: %s\n", url)

	// Run agent with JSON output
	args := []string{"-p", prompt, "--output-format", "json"}
	output, err := runAgentWithSpinner(agent, args, "Analyzing job posting...")
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	// Parse JSON result
	result, sessionID := parseAgentOutput(output)

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
		fmt.Printf("\n%sMatch analysis saved to %s%s\n", cGreen, matchPath, cReset)
	}

	// Clean up session (one-off analysis)
	if sessionID != "" {
		cleanupSession(agent, sessionID)
	}

	return nil
}

func runMatchIssue(cfg *config.Config, agent, issueNum string) error {
	sessionKey := fmt.Sprintf("match-%s", issueNum)
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")

	// Check for existing session
	sessionID, hasSession := getSession(sessionKey)

	// Handle --push flag
	if matchPushFlag {
		// Check if analysis exists
		content, err := os.ReadFile(matchPath)
		if err != nil && os.IsNotExist(err) {
			// Create analysis first
			fmt.Printf("No existing analysis found, creating new analysis for issue #%s...\n", issueNum)
			if err := runMatchAnalysis(cfg, agent, issueNum, sessionKey, hasSession, sessionID); err != nil {
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
		fmt.Printf("Posting match analysis to issue #%s...\n", issueNum)
		ghCmd := exec.Command("gh", "issue", "comment", issueNum,
			"--repo", cfg.Repo,
			"--body", string(content))
		if output, err := ghCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error posting comment: %w\nOutput: %s", err, string(output))
		}
		fmt.Printf("%sMatch analysis posted as comment to issue #%s%s\n", cGreen, issueNum, cReset)
		return nil
	}

	// Interactive mode
	if matchInteractiveFlag {
		return runMatchInteractive(cfg, agent, issueNum, sessionKey, hasSession, sessionID)
	}

	// Normal analysis
	return runMatchAnalysis(cfg, agent, issueNum, sessionKey, hasSession, sessionID)
}

func runMatchAnalysis(cfg *config.Config, agent, issueNum, sessionKey string, hasSession bool, sessionID string) error {
	// Fetch issue body for context
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	// Build prompt
	prompt, err := buildMatchPrompt(cfg, "", issueBody)
	if err != nil {
		return err
	}

	if matchContextFlag != "" {
		prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, matchContextFlag)
	}

	// Build args
	args := []string{"-p", prompt, "--output-format", "json"}
	if hasSession {
		args = append(args, "--resume", sessionID)
		if matchContextFlag != "" {
			fmt.Printf("Resuming session for issue #%s with new context...\n", issueNum)
		} else {
			fmt.Printf("Resuming existing session for issue #%s...\n", issueNum)
		}
	} else {
		fmt.Printf("Running match analysis for issue #%s...\n", issueNum)
	}

	output, err := runAgentWithSpinner(agent, args, "Analyzing job match...")
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	// Parse result
	result, newSessionID := parseAgentOutput(output)

	// Save session if new
	if !hasSession && newSessionID != "" {
		if err := saveSession(sessionKey, newSessionID); err != nil {
			fmt.Printf("Warning: Could not save session: %v\n", err)
		} else {
			fmt.Printf("%sSession saved. Use 'cvx match %s -c \"context\"' or 'cvx match %s -i' to continue.%s\n",
				cGreen, issueNum, issueNum, cReset)
		}
	}

	// Save output
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")
	os.MkdirAll(filepath.Dir(matchPath), 0755)
	if err := os.WriteFile(matchPath, []byte(result), 0644); err != nil {
		fmt.Printf("Warning: Could not save analysis: %v\n", err)
	} else {
		fmt.Printf("%sMatch analysis saved to %s%s\n", cGreen, matchPath, cReset)
	}

	fmt.Println("\nMatch Analysis:")
	fmt.Println(result)

	return nil
}

func runMatchInteractive(cfg *config.Config, agent, issueNum, sessionKey string, hasSession bool, sessionID string) error {
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

		prompt, err := buildMatchPrompt(cfg, "", issueBody)
		if err != nil {
			return err
		}

		if matchContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, matchContextFlag)
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

func buildMatchPrompt(cfg *config.Config, url, issueBody string) (string, error) {
	// Load workflow
	workflowContent, err := workflow.LoadMatch()
	if err != nil {
		return "", fmt.Errorf("error loading workflow: %w", err)
	}

	// Substitute config paths
	tmpl, err := template.New("match").Parse(workflowContent)
	if err != nil {
		return "", fmt.Errorf("error parsing workflow template: %w", err)
	}

	data := struct {
		CVPath         string
		ExperiencePath string
	}{
		CVPath:         cfg.CVPath,
		ExperiencePath: cfg.ExperiencePath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing workflow template: %w", err)
	}

	prompt := buf.String()

	// Add job content
	if url != "" {
		prompt = fmt.Sprintf("%s\n\n## Job Posting URL\n%s", prompt, url)
	} else if issueBody != "" {
		prompt = fmt.Sprintf("%s\n\n## Job Posting Content\n%s", prompt, issueBody)
	}

	return prompt, nil
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
