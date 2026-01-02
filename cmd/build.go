package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/cache"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
)

var buildCmd = &cobra.Command{
	Use:   "build [issue-number]",
	Short: "Build tailored CV and cover letter",
	Long: `Build tailored CV and cover letter for a job posting.

Two build modes:
  1. Python Agent (default): Structured YAML with caching and validation
  2. Interactive CLI: Real-time editing with auto-detected CLI tools

Examples:
  cvx build -m sonnet-4          # Python agent mode
  cvx build -m flash             # Python agent with Gemini
  cvx build -m sonnet-4 --dry-run  # Preview without AI call
  cvx build -m sonnet-4 --no-cache # Skip cache

  cvx build -i                   # Interactive mode, auto-detect CLI
  cvx build 42 -i                # Interactive for issue #42
  cvx build -i -c "focus on ML"  # Interactive with context`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

var (
	buildModelFlag        string
	buildContextFlag      string
	buildInteractiveFlag  bool
	buildNoCacheFlag      bool
	buildRefreshCacheFlag bool
	buildDryRunFlag       bool
)

func init() {
	buildCmd.Flags().StringVarP(&buildModelFlag, "model", "m", "", "Model: sonnet-4, sonnet-4-5, opus-4, opus-4-5, flash, pro, flash-3, pro-3")
	buildCmd.Flags().BoolVarP(&buildInteractiveFlag, "interactive", "i", false, "Interactive CLI mode (auto-detects claude-code or gemini-cli)")
	buildCmd.Flags().StringVarP(&buildContextFlag, "context", "c", "", "Feedback or additional context")
	buildCmd.Flags().BoolVar(&buildNoCacheFlag, "no-cache", false, "Skip cache (Python agent mode only)")
	buildCmd.Flags().BoolVar(&buildRefreshCacheFlag, "refresh-cache", false, "Recompute and overwrite cache (Python agent mode only)")
	buildCmd.Flags().BoolVar(&buildDryRunFlag, "dry-run", false, "Preview without AI call (Python agent mode only)")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	issueNum, err := resolveIssueNumber(args)
	if err != nil {
		return err
	}

	// Flag validation
	if buildInteractiveFlag && buildModelFlag != "" {
		return fmt.Errorf("cannot use -i and -m together (choose one mode)")
	}
	if buildInteractiveFlag && (buildDryRunFlag || buildNoCacheFlag || buildRefreshCacheFlag) {
		return fmt.Errorf("--dry-run, --no-cache, and --refresh-cache only work with -m mode")
	}
	if buildNoCacheFlag && buildRefreshCacheFlag {
		return fmt.Errorf("cannot use --no-cache and --refresh-cache together")
	}

	// Interactive CLI mode
	if buildInteractiveFlag {
		agentSetting, err := resolveAgent(cfg)
		if err != nil {
			return err
		}
		cfg.Agent = agentSetting

		if err := ensureIssueBranch(cfg.Repo, issueNum); err != nil {
			return err
		}

		return runBuildInteractive(cfg, issueNum)
	}

	// Python Agent Mode (requires -m)
	if buildModelFlag == "" {
		return fmt.Errorf("model not specified. Use -m (e.g., cvx build -m sonnet-4)")
	}

	if err := os.Setenv("AI_MODEL", buildModelFlag); err != nil {
		return fmt.Errorf("failed to set AI_MODEL: %w", err)
	}

	return runBuildWithPythonAgent(cfg, issueNum)
}

func resolveIssueNumber(args []string) (string, error) {
	if len(args) > 0 {
		issueNum := args[0]
		if _, err := fmt.Sscanf(issueNum, "%d", new(int)); err != nil {
			return "", fmt.Errorf("invalid issue number: %s (must be numeric)", issueNum)
		}
		return issueNum, nil
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", err
	}
	issueNum := extractIssueFromBranch(currentBranch)
	if issueNum == "" {
		return "", fmt.Errorf("could not infer issue number from branch '%s'. Provide it explicitly: cvx build <issue-number>", currentBranch)
	}
	fmt.Printf("Using issue #%s (from branch %s)\n", issueNum, currentBranch)
	return issueNum, nil
}

func resolveAgent(cfg *config.Config) (string, error) {
	var baseAgent string

	// Priority 1: Use configured default CLI agent if available
	if cfg.DefaultCLIAgent != "" {
		// Check if the configured CLI is actually available
		cliName := ""
		switch cfg.DefaultCLIAgent {
		case "claude-code":
			cliName = "claude"
		case "gemini-cli":
			cliName = "gemini"
		}
		if cliName != "" && isCommandAvailable(cliName) {
			baseAgent = cfg.DefaultCLIAgent
		}
	}

	// Priority 2: Auto-detect if no config or configured CLI not available
	if baseAgent == "" {
		baseAgent = detectAvailableCLI()
	}

	// Priority 3: Fallback to config.Agent if it's a CLI agent
	if baseAgent == "" {
		if cfg.Agent != "" && ai.IsCLIAgentSupported(cfg.Agent) {
			baseAgent = cfg.Agent
		} else {
			return "", fmt.Errorf("no CLI tool found. Install claude-code or gemini-cli")
		}
	}

	if !ai.IsAgentSupported(baseAgent) {
		return "", fmt.Errorf("unsupported agent: %s", baseAgent)
	}

	return baseAgent, nil
}

func runBuildInteractive(cfg *config.Config, issueNum string) error {
	agent := cfg.AgentCLI()

	// Use issue number as unified session key
	sessionID, hasSession := getSession(issueNum + "-build")

	var execCmd *exec.Cmd

	if hasSession {
		fmt.Printf("%s Resuming session for issue %s\n", style.C(style.Cyan, "↻"), style.C(style.Cyan, "#"+issueNum))
		if buildContextFlag != "" {
			execCmd = exec.Command(agent, "--resume", sessionID, "-p", buildContextFlag)
		} else {
			execCmd = exec.Command(agent, "--resume", sessionID)
		}
	} else {
		fmt.Printf("%s Starting build session for issue %s\n", style.C(style.Green, "▶"), style.C(style.Cyan, "#"+issueNum))

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
		if agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:") {
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
			_ = saveSession(issueNum+"-build", newSessionID)
			fmt.Printf("%s Session saved for issue %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, "#"+issueNum))
		}
	}

	return nil
}

func runBuildNonInteractive(ctx context.Context, cfg *config.Config, agent, issueNum string) error {
	// Fetch issue body
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	// Path 1: CLI agent (headless) - claude-code/gemini-cli handles tool use internally
	if ai.IsAgentCLI(agent) {
		return runBuildWithCLI(cfg, agent, issueNum, issueBody)
	}

	// Path 2: API model - use structured output
	fmt.Printf("%s Building application for issue %s\n", style.C(style.Green, "▶"), style.C(style.Cyan, "#"+issueNum))
	return runBuildWithAPI(ctx, cfg, agent, issueBody)
}

// runBuildWithCLI shells out to claude-code/gemini-cli CLI in headless mode
func runBuildWithCLI(cfg *config.Config, agent, issueNum, issueBody string) error {
	var cliName string
	if agent == "gemini-cli" || strings.HasPrefix(agent, "gemini-cli:") {
		cliName = "gemini"
	} else {
		cliName = "claude"
	}

	// Check for existing session
	sessionID, hasSession := getSession(issueNum + "-build")

	var args []string
	if hasSession {
		fmt.Printf("%s Resuming session for issue %s\n", style.C(style.Cyan, "↻"), style.C(style.Cyan, "#"+issueNum))
		// Resume existing session
		if buildContextFlag != "" {
			args = []string{"--resume", sessionID, "-p", buildContextFlag}
		} else {
			args = []string{"--resume", sessionID, "-p", "continue"}
		}
	} else {
		fmt.Printf("%s Starting build session for issue %s\n", style.C(style.Green, "▶"), style.C(style.Cyan, "#"+issueNum))
		// Start new session
		prompt, err := buildBuildPrompt(cfg, issueBody)
		if err != nil {
			return err
		}

		if buildContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nFeedback: %s", prompt, buildContextFlag)
		}

		args = []string{"-p", prompt}
	}

	// Add CLI-specific flags
	if cliName == "claude" {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Use shared spinner helper
	spinnerMsg := fmt.Sprintf("Building with 🤖 %s...", agent)
	output, err := runAgentWithSpinner(cliName, args, spinnerMsg)
	if err != nil {
		return fmt.Errorf("error running %s: %w", agent, err)
	}

	// Print output in gray
	if len(output) > 0 {
		fmt.Println(style.C(style.Gray, string(output)))
	}

	// Save session if new
	if !hasSession {
		if newSessionID := getMostRecentAgentSession(cliName); newSessionID != "" {
			_ = saveSession(issueNum+"-build", newSessionID)
			fmt.Printf("%s Session saved for issue %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, "#"+issueNum))
		}
	}

	return nil
}

// runBuildWithAPI uses API with structured JSON output
func runBuildWithAPI(ctx context.Context, cfg *config.Config, agent, issueBody string) error {
	systemPrompt, userPrompt, err := buildBuildPromptParts(cfg, issueBody)
	if err != nil {
		return err
	}

	// Read existing templates to extract preambles
	cvTemplate, err := os.ReadFile(filepath.Join("src", "cv.tex"))
	if err != nil {
		return fmt.Errorf("failed to read cv.tex template: %w", err)
	}
	letterTemplate, err := os.ReadFile(filepath.Join("src", "letter.tex"))
	if err != nil {
		return fmt.Errorf("failed to read letter.tex template: %w", err)
	}

	// Extract preambles (everything before \begin{document})
	cvPreamble := extractPreamble(string(cvTemplate))
	letterPreamble := extractPreamble(string(letterTemplate))

	// Add structured output instruction - only ask for document body
	structuredInstruction := fmt.Sprintf(`

IMPORTANT: You must respond with ONLY a valid JSON object containing the document BODY content (everything between \begin{document} and \end{document}).

The preambles are fixed and will be preserved. You MUST use the exact same LaTeX commands from the templates.

CV preamble (preserved, for reference of available commands):
%s

Letter preamble (preserved, for reference of available commands):
%s

Return JSON in this exact format:
{
  "cv_body": "<content between \\begin{document} and \\end{document} for cv.tex>",
  "letter_body": "<content between \\begin{document} and \\end{document} for letter.tex>"
}

Do not include \\begin{document} or \\end{document} in your response.
Do not include any explanation, markdown, or text outside the JSON object.`, cvPreamble, letterPreamble)

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
				msg := fmt.Sprintf("Building application using 🤖 %s...", agent)
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
		CVBody     string `json:"cv_body"`
		LetterBody string `json:"letter_body"`
	}

	// Try to extract JSON from response (may have markdown code blocks)
	jsonStr := extractJSON(result)
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return fmt.Errorf("failed to parse AI response as JSON: %w\nResponse was: %s", err, result)
	}

	if output.CVBody == "" || output.LetterBody == "" {
		return fmt.Errorf("AI response missing cv_body or letter_body")
	}

	// Combine preambles with AI-generated bodies
	cvContent := cvPreamble + "\n\\begin{document}\n" + output.CVBody + "\n\\end{document}\n"
	letterContent := letterPreamble + "\n\\begin{document}\n" + output.LetterBody + "\n\\end{document}\n"

	// Write files
	cvPath := filepath.Join("src", "cv.tex")
	letterPath := filepath.Join("src", "letter.tex")

	if err := os.WriteFile(cvPath, []byte(cvContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", cvPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, cvPath))

	if err := os.WriteFile(letterPath, []byte(letterContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", letterPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, letterPath))

	return nil
}

// extractPreamble returns everything before \begin{document}
func extractPreamble(content string) string {
	marker := "\\begin{document}"
	if idx := strings.Index(content, marker); idx != -1 {
		return strings.TrimSpace(content[:idx])
	}
	return content
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

	fmt.Printf("%s Opened build/combined.pdf in VSCode\n", style.C(style.Green, "✓"))
	return nil
}

func commitBuildChanges(repo, issueNum string) error {
	// Verify we're on the expected issue branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}
	expectedBranch, _, _, err := getIssueBranchName(repo, issueNum)
	if err != nil {
		return fmt.Errorf("error getting expected branch name: %w", err)
	}
	if currentBranch != expectedBranch {
		return fmt.Errorf("refusing to commit: expected branch %s, but on %s", expectedBranch, currentBranch)
	}

	// Stage src/ and build/ changes
	addCmd := exec.Command("git", "add", "src/", "build/")
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("error staging changes: %w", err)
	}

	// Check if there are changes to commit
	diffCmd := exec.Command("git", "diff", "--cached", "--quiet")
	if err := diffCmd.Run(); err == nil {
		// No changes to commit
		fmt.Printf("%s No changes to commit\n", style.C(style.Yellow, "⚠"))
		return nil
	}

	// Set git identity for CI environments
	_ = exec.Command("git", "config", "user.name", "cvx").Run()
	_ = exec.Command("git", "config", "user.email", "cvx@automated").Run()

	// Commit with message
	commitMsg := fmt.Sprintf("build: update application for issue #%s", issueNum)
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}

	fmt.Printf("%s Committed changes for issue #%s\n", style.C(style.Green, "✓"), issueNum)
	return nil
}

func pushChanges() error {
	pushCmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("error pushing changes: %w", err)
	}

	fmt.Printf("%s Pushed changes to remote\n", style.C(style.Green, "✓"))
	return nil
}

func buildPDF() error {
	fmt.Printf("%s Building PDF...\n", style.C(style.Cyan, "⧗"))
	makeCmd := exec.Command("make", "combined")
	if output, err := makeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error building PDF: %w\n%s", err, string(output))
	}
	fmt.Printf("%s PDF built successfully\n", style.C(style.Green, "✓"))
	return nil
}

// ========================================
// Python Agent Mode Functions
// ========================================

// readYAMLCV reads cv.yaml and extracts the cv field
func readYAMLCV(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		CV map[string]interface{} `yaml:"cv"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.CV, nil
}

// readYAMLLetter reads letter.yaml and extracts the letter field
func readYAMLLetter(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Letter map[string]interface{} `yaml:"letter"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Letter, nil
}

// writeYAMLCV writes cv data back to cv.yaml
func writeYAMLCV(path string, cv map[string]interface{}) error {
	wrapper := struct {
		CV map[string]interface{} `yaml:"cv"`
	}{CV: cv}

	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// writeYAMLLetter writes letter data back to letter.yaml
func writeYAMLLetter(path string, letter map[string]interface{}) error {
	wrapper := struct {
		Letter map[string]interface{} `yaml:"letter"`
	}{Letter: letter}

	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// callPythonAgent calls the Python agent subprocess with JSON stdin/stdout
// extractAgentToCache extracts the embedded agent to a cache directory for reuse
func extractAgentToCache() (string, error) {
	if agentFS == nil {
		return "", fmt.Errorf("agent filesystem not initialized")
	}

	// Use ~/.cache/cvx/agent as persistent cache
	cacheDir := filepath.Join(os.ExpandEnv("$HOME"), ".cache", "cvx", "agent")

	// Check if already extracted (simple version check via stat)
	if stat, err := os.Stat(cacheDir); err == nil && stat.IsDir() {
		// Cache exists, reuse it
		return cacheDir, nil
	}

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Walk the embedded FS and extract all files
	err := fs.WalkDir(*agentFS, "agent", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path (remove "agent" or "agent/" prefix)
		relPath := strings.TrimPrefix(path, "agent")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			return nil // skip the agent directory itself
		}

		targetPath := filepath.Join(cacheDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		// Read file from embedded FS
		data, err := fs.ReadFile(*agentFS, path)
		if err != nil {
			return err
		}

		// Write to cache directory
		return os.WriteFile(targetPath, data, 0o644)
	})

	if err != nil {
		_ = os.RemoveAll(cacheDir) // cleanup on error
		return "", fmt.Errorf("failed to extract agent: %w", err)
	}

	return cacheDir, nil
}

// Go boundary: subprocess management, JSON marshaling
// Python boundary: AI calls, validation, caching
func callPythonAgent(jobPosting string, cv, letter map[string]interface{}) (cvOut, letterOut map[string]interface{}, err error) {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		return nil, nil, fmt.Errorf("uv is not installed. Please install uv: https://docs.astral.sh/uv/")
	}

	// Extract embedded agent to cache directory
	agentDir, err := extractAgentToCache()
	if err != nil {
		return nil, nil, err
	}

	// Prepare input JSON
	input := map[string]interface{}{
		"job_posting": jobPosting,
		"cv":          cv,
		"letter":      letter,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Call Python agent via uvx
	cmd := exec.Command("uvx", "--from", agentDir, "cvx-agent")
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("python agent failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse output JSON
	var output struct {
		CV     map[string]interface{} `json:"cv"`
		Letter map[string]interface{} `json:"letter"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, nil, fmt.Errorf("failed to parse output: %w\noutput: %s", err, stdout.String())
	}

	// Validate output has required fields
	if output.CV == nil || output.Letter == nil {
		return nil, nil, fmt.Errorf("invalid output: missing cv or letter fields")
	}

	return output.CV, output.Letter, nil
}

// runBuildWithPythonAgent executes the build command using the Python agent
// This mode is triggered when -m flag is used without --call-api-directly
func runBuildWithPythonAgent(cfg *config.Config, issueNum string) error {
	fmt.Printf("%s Building with Python agent for issue %s\n",
		style.C(style.Green, "▶"), style.C(style.Cyan, "#"+issueNum))

	// 1. Fetch job posting from GitHub
	issueBody, err := fetchIssueBody(cfg.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	// 2. Read YAML files
	cvPath := cfg.CVYAMLPath
	if cvPath == "" {
		cvPath = "src/cv.yaml"
	}
	letterPath := cfg.LetterYAMLPath
	if letterPath == "" {
		letterPath = "src/letter.yaml"
	}

	cv, err := readYAMLCV(cvPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", cvPath, err)
	}

	letter, err := readYAMLLetter(letterPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", letterPath, err)
	}

	// 3. Read schema if available
	schemaContent := ""
	schemaPath := "schema/schema.json"
	if data, err := os.ReadFile(schemaPath); err == nil {
		schemaContent = string(data)
	}

	// 4. Compute cache key using canonical JSON
	cvJSON, err := json.Marshal(cv)
	if err != nil {
		return fmt.Errorf("failed to marshal CV to JSON: %w", err)
	}
	letterJSON, err := json.Marshal(letter)
	if err != nil {
		return fmt.Errorf("failed to marshal letter to JSON: %w", err)
	}

	// Convert issue number to int for cache key
	var issueNumInt int
	if _, err := fmt.Sscanf(issueNum, "%d", &issueNumInt); err != nil {
		return fmt.Errorf("invalid issue number %q: %w", issueNum, err)
	}

	cacheKey := cache.CacheKey(issueNumInt, issueBody, string(cvJSON), string(letterJSON), schemaContent, buildModelFlag)

	// 5. Handle --dry-run
	if buildDryRunFlag {
		fmt.Printf("%s Dry run mode\n", style.C(style.Cyan, "►"))
		fmt.Printf("  Model: %s\n", buildModelFlag)
		fmt.Printf("  Cache key: %s\n", cacheKey[:16]+"...")
		if cache.Exists(cacheKey) && !buildNoCacheFlag && !buildRefreshCacheFlag {
			fmt.Printf("  Cache: %s (hit)\n", style.C(style.Green, "✓"))
		} else {
			fmt.Printf("  Cache: %s (miss)\n", style.C(style.Yellow, "○"))
		}
		fmt.Printf("  Agent command: uvx --from <agent-dir> cvx-agent\n")
		return nil
	}

	// 6. Check cache (unless --no-cache or --refresh-cache)
	var cvOut, letterOut map[string]interface{}
	useCache := !buildNoCacheFlag && !buildRefreshCacheFlag

	if useCache && cache.Exists(cacheKey) {
		fmt.Printf("%s Cache hit\n", style.C(style.Green, "✓"))
		cached, err := cache.Read(cacheKey)
		if err == nil {
			cvOut = cached["cv"].(map[string]interface{})
			letterOut = cached["letter"].(map[string]interface{})
		} else {
			fmt.Printf("%s Cache read failed, calling agent\n", style.C(style.Yellow, "⚠"))
			cvOut, letterOut, err = callPythonAgent(issueBody, cv, letter)
			if err != nil {
				return err
			}
		}
	} else {
		// 7. Call Python agent
		if buildRefreshCacheFlag {
			fmt.Printf("%s Refreshing cache, calling Python agent...\n", style.C(style.Cyan, "⧗"))
		} else {
			fmt.Printf("%s Calling Python agent...\n", style.C(style.Cyan, "⧗"))
		}
		var err error
		cvOut, letterOut, err = callPythonAgent(issueBody, cv, letter)
		if err != nil {
			return err
		}

		// 8. Write to cache (unless --no-cache)
		if !buildNoCacheFlag {
			if err := cache.Write(cacheKey, cvOut, letterOut); err != nil {
				fmt.Printf("%s Failed to write cache: %v\n", style.C(style.Yellow, "⚠"), err)
			}
		}
	}

	// 9. Write output YAML
	if err := writeYAMLCV(cvPath, cvOut); err != nil {
		return fmt.Errorf("failed to write %s: %w", cvPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), cvPath)

	if err := writeYAMLLetter(letterPath, letterOut); err != nil {
		return fmt.Errorf("failed to write %s: %w", letterPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), letterPath)

	return nil
}
