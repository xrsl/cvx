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
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
)

var (
	adviseAgentFlag           string
	adviseModelFlag           string
	adviseContextFlag         string
	advisePostAsCommentFlag   bool
	adviseCallAPIDirectlyFlag bool
)

var adviseCmd = &cobra.Command{
	Use:   "advise <issue-number-or-url>",
	Short: "Get career advice on job match",
	Long: `Analyze job-CV match quality and get strategic career advice.

Takes a GitHub issue number or job posting URL and analyzes
how well your CV matches the position.

Examples:
  cvx advise 42                                  # Analyze issue #42
  cvx advise 42 --post-as-comment                # Analyze and post as comment
  cvx advise 42 -a gemini-cli                    # Gemini CLI agent
  cvx advise 42 -m sonnet-4                      # Claude CLI with sonnet-4 model
  cvx advise 42 --call-api-directly -m flash     # Gemini API directly with flash model
  cvx advise 42 -c "Focus on backend"`,
	Args: cobra.ExactArgs(1),
	RunE: runAdvise,
}

func init() {
	adviseCmd.Flags().StringVarP(&adviseAgentFlag, "agent", "a", "", "CLI agent: claude-code, gemini-cli")
	adviseCmd.Flags().StringVarP(&adviseModelFlag, "model", "m", "", "Model: sonnet-4, sonnet-4-5, opus-4, opus-4-5, flash, pro, flash-3, pro-3")
	adviseCmd.Flags().BoolVar(&adviseCallAPIDirectlyFlag, "call-api-directly", false, "Explicitly call API directly (requires --model)")
	adviseCmd.Flags().StringVarP(&adviseContextFlag, "context", "c", "", "Additional context for analysis")
	adviseCmd.Flags().BoolVar(&advisePostAsCommentFlag, "post-as-comment", false, "Post analysis to GitHub issue as comment")
	rootCmd.AddCommand(adviseCmd)
}

func runAdvise(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Load config
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve agent/model (flags > config)
	var agentSetting string

	if adviseCallAPIDirectlyFlag {
		// API mode - requires explicit model
		if adviseModelFlag == "" {
			return fmt.Errorf("--call-api-directly requires --model")
		}

		modelConfig, hasModel := ai.GetModel(adviseModelFlag)
		if !hasModel {
			return fmt.Errorf("unsupported model: %s (supported: %v)", adviseModelFlag, ai.SupportedModelNames())
		}

		agentSetting = modelConfig.APIName

	} else {
		// CLI agent mode
		baseAgent := ""
		if adviseAgentFlag != "" {
			if !ai.IsCLIAgentSupported(adviseAgentFlag) {
				return fmt.Errorf("unsupported CLI agent: %s (supported: claude-code, gemini-cli). Use --call-api-directly for API access", adviseAgentFlag)
			}
			baseAgent = adviseAgentFlag
		} else if cfg.Agent != "" {
			baseAgent = cfg.Agent
		} else {
			baseAgent = ai.DefaultAgent()
		}

		// Apply model if specified
		if adviseModelFlag != "" {
			modelConfig, hasModel := ai.GetModel(adviseModelFlag)
			if !hasModel {
				return fmt.Errorf("unsupported model: %s (supported: %v)", adviseModelFlag, ai.SupportedModelNames())
			}
			agentSetting = baseAgent + ":" + modelConfig.CLIName
		} else {
			agentSetting = baseAgent
		}
	}

	// Validate final setting
	if !ai.IsAgentSupported(agentSetting) {
		return fmt.Errorf("unsupported agent/model: %s", agentSetting)
	}

	// Override config agent for this run
	cfg.Agent = agentSetting

	// Get CLI agent name from agent setting
	agent := cfg.AgentCLI()

	// Check if target is URL or issue number
	isURL := strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")

	ctx := cmd.Context()

	if isURL {
		return runAdviseURL(ctx, cfg, agent, target)
	}
	return runAdviseIssue(ctx, cfg, agent, target)
}

func runAdviseURL(ctx context.Context, cfg *config.Config, agent, url string) error {
	if advisePostAsCommentFlag {
		fmt.Println("Warning: --post-as-comment flag is not supported for URL-based analysis")
		fmt.Println("Create an issue first, then run 'cvx advise <issue-number> --post-as-comment'")
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

		result, err = runAdviseWithAPI(ctx, cfg.Agent, systemPrompt, userPrompt)
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

		args := buildCLIArgs(agent, prompt, "", false)
		spinnerMsg := fmt.Sprintf("Analyzing job posting using 🤖 %s...", cfg.Agent)
		output, err := runAgentWithSpinner(agent, args, spinnerMsg)
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
	if err := os.MkdirAll(filepath.Dir(matchPath), 0o755); err != nil {
		return fmt.Errorf("failed to create matches directory: %w", err)
	}
	if err := os.WriteFile(matchPath, []byte(result), 0o644); err != nil {
		fmt.Printf("Warning: Could not save analysis: %v\n", err)
	} else {
		fmt.Printf("\n%s%s\n", style.Success("Analysis saved to"), matchPath)
	}

	return nil
}

// runAdviseWithAPI runs analysis using API client with caching
func runAdviseWithAPI(ctx context.Context, agent, systemPrompt, userPrompt string) (string, error) {
	client, err := ai.NewClient(agent)
	if err != nil {
		return "", fmt.Errorf("error creating AI client: %w", err)
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
				msg := fmt.Sprintf("Analyzing job match using 🤖 %s...", agent)
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), msg)
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
		// Fall back to regular prompt
		prompt := systemPrompt + "\n\n" + userPrompt
		result, err = client.GenerateContent(ctx, prompt)
	}

	done <- true
	close(done)

	return result, err
}

func runAdviseIssue(ctx context.Context, cfg *config.Config, agent, issueNum string) error {
	sessionKey := issueNum
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")

	// Check for existing session
	sessionID, hasSession := getSession(sessionKey)

	// Handle --post-as-comment flag
	if advisePostAsCommentFlag {
		// Check if analysis exists
		content, err := os.ReadFile(matchPath)
		if err != nil && os.IsNotExist(err) {
			// Create analysis first
			fmt.Printf("No existing analysis found, creating new analysis for issue #%s...\n", issueNum)
			if err := runAdviseAnalysis(ctx, cfg, agent, issueNum, sessionKey, hasSession, sessionID); err != nil {
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
		cli := gh.New()
		if err := cli.IssueComment(cfg.Repo, issueNum, string(content)); err != nil {
			return fmt.Errorf("error posting comment: %w", err)
		}
		fmt.Printf("%sissue #%s\n", style.Success("Analysis posted as comment to"), issueNum)
		return nil
	}

	// Run analysis
	return runAdviseAnalysis(ctx, cfg, agent, issueNum, sessionKey, hasSession, sessionID)
}

func runAdviseAnalysis(ctx context.Context, cfg *config.Config, agent, issueNum, sessionKey string, hasSession bool, sessionID string) error {
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

		result, err = runAdviseWithAPI(ctx, cfg.Agent, systemPrompt, userPrompt)
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

		args := buildCLIArgs(agent, prompt, sessionID, hasSession)
		if hasSession {
			if adviseContextFlag != "" {
				fmt.Printf("Resuming session for issue #%s with new context...\n", issueNum)
			} else {
				fmt.Printf("Resuming existing session for issue #%s...\n", issueNum)
			}
		} else {
			fmt.Printf("Running analysis for issue #%s...\n", issueNum)
		}

		// Build spinner message with agent name and model
		spinnerMsg := fmt.Sprintf("Analyzing job match using 🤖 %s...", cfg.Agent)

		output, err := runAgentWithSpinner(agent, args, spinnerMsg)
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}

		// Debug: print raw output
		if os.Getenv("DEBUG") != "" {
			fmt.Printf("\n=== RAW OUTPUT ===\n%s\n=== END RAW OUTPUT ===\n", string(output))
		}

		var newSessionID string
		result, newSessionID = parseAgentOutput(output)

		// Save session if new
		if !hasSession && newSessionID != "" {
			if err := saveSession(sessionKey, newSessionID); err != nil {
				fmt.Printf("Warning: Could not save session: %v\n", err)
			} else {
				fmt.Printf("%sUse 'cvx advise %s -c \"context\"' to continue.\n",
					style.Success("Session saved."), issueNum)
			}
		}
	}

	// Save output
	matchPath := filepath.Join(".cvx", "matches", issueNum+".md")
	_ = os.MkdirAll(filepath.Dir(matchPath), 0o755)
	if err := os.WriteFile(matchPath, []byte(result), 0o644); err != nil {
		fmt.Printf("Warning: Could not save analysis: %v\n", err)
	} else {
		fmt.Printf("%s%s\n", style.Success("Analysis saved to"), matchPath)
	}

	fmt.Println("\nMatch Analysis:")
	fmt.Println(result)

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
		CVPath        string
		ReferencePath string
	}{
		CVPath:        cfg.CVPath,
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
	cli := gh.New()
	output, err := cli.IssueViewByStr(repo, issueNum, []string{"body"})
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
	_ = os.MkdirAll(filepath.Join(".cvx", "sessions"), 0o755)
	sessionFile := filepath.Join(".cvx", "sessions", key+".sid")
	return os.WriteFile(sessionFile, []byte(sessionID), 0o644)
}

func getMostRecentAgentSession(agent string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	isGemini := agent == "gemini" || strings.HasPrefix(agent, "gemini:")

	var sessionDir string
	var fileExt string

	if isGemini {
		// Gemini uses ~/.gemini/history/{project-hash}/
		// Get project hash by listing directories and finding one that contains recent sessions
		historyDir := filepath.Join(home, ".gemini", "history")
		entries, err := os.ReadDir(historyDir)
		if err != nil {
			return ""
		}
		// Find the most recently modified project directory
		var mostRecentDir string
		var mostRecentTime time.Time
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(mostRecentTime) {
				mostRecentTime = info.ModTime()
				mostRecentDir = entry.Name()
			}
		}
		if mostRecentDir == "" {
			return ""
		}
		sessionDir = filepath.Join(historyDir, mostRecentDir)
		fileExt = ".json"
	} else {
		// Claude uses ~/.claude/projects/{slug}/
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		slug := strings.ReplaceAll(wd, "/", "-")
		sessionDir = filepath.Join(home, ".claude", "projects", slug)
		fileExt = ".jsonl"
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return ""
	}

	var mostRecentFile string
	var mostRecentTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), fileExt) {
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

	return strings.TrimSuffix(mostRecentFile, fileExt)
}

func cleanupSession(agent, sessionID string) {
	home, _ := os.UserHomeDir()
	isGemini := agent == "gemini" || strings.HasPrefix(agent, "gemini:")

	if isGemini {
		// Gemini session cleanup - find the session file in history
		historyDir := filepath.Join(home, ".gemini", "history")
		entries, _ := os.ReadDir(historyDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			sessionFile := filepath.Join(historyDir, entry.Name(), sessionID+".json")
			_ = os.Remove(sessionFile)
		}
	} else {
		// Claude session cleanup
		wd, _ := os.Getwd()
		slug := strings.ReplaceAll(wd, "/", "-")
		sessionFile := filepath.Join(home, ".claude", "projects", slug, sessionID+".jsonl")
		_ = os.Remove(sessionFile)
	}
}

// buildCLIArgs constructs CLI arguments for both claude and gemini
func buildCLIArgs(agent, prompt, sessionID string, hasSession bool) []string {
	isGemini := agent == "gemini" || strings.HasPrefix(agent, "gemini:")

	var args []string
	args = append(args, "-p", prompt)

	// Output format flag
	if isGemini {
		args = append(args, "-o", "json")
	} else {
		args = append(args, "--output-format", "json")
	}

	// Resume flag
	if hasSession && sessionID != "" {
		args = append(args, "--resume", sessionID)
	}

	return args
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
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), message)
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
		Response  string `json:"response"` // gemini uses "response" instead of "result"
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return str, ""
	}

	// Try response field first (gemini), then result field (claude)
	resultText := parsed.Response
	if resultText == "" {
		resultText = parsed.Result
	}

	return resultText, parsed.SessionID
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
