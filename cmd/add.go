package cmd

import (
	"context"
	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/gemini"
	"cvx/pkg/schema"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	textFlag   string
	modelFlag  string
	repoFlag   string
	schemaFlag string
	dryRunFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a job application",
	Long: `Fetch job posting, extract details with AI, and create a GitHub issue.

Fields are extracted based on schema (GitHub issue template YAML).

Examples:
  cvx add https://company.com/job
  cvx add https://company.com/job --dry-run
  cvx add https://company.com/job -m claude-sonnet-4
  cvx add https://company.com/job -s /path/to/job-app.yml`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&textFlag, "text", "t", "", "Job posting text (skips URL fetch)")
	addCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "AI model (overrides config)")
	addCmd.Flags().StringVarP(&repoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	addCmd.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Schema file (overrides config)")
	addCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Extract only, don't create issue")
	rootCmd.AddCommand(addCmd)
}

func log(format string, args ...any) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	url := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve model (flag > config > default)
	model := modelFlag
	if model == "" {
		model = cfg.Model
	}
	if model == "" {
		model = gemini.DefaultModel
	}

	// Validate model
	if !ai.IsModelSupported(model) {
		return fmt.Errorf("unsupported model: %s (supported: %v)", model, ai.SupportedModels())
	}

	// Resolve repo (flag > config)
	repo := repoFlag
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" && !dryRunFlag {
		return fmt.Errorf("no repo configured. Run: cvx config set repo owner/name")
	}

	// Resolve schema (flag > config)
	schemaPath := schemaFlag
	if schemaPath == "" {
		schemaPath = cfg.Schema
	}
	if schemaPath == "" {
		return fmt.Errorf("no schema configured. Run: cvx config set schema /path/to/job-app.yml")
	}

	// Load schema
	sch, err := schema.Load(schemaPath)
	if err != nil {
		return fmt.Errorf("schema error: %w", err)
	}

	// Get job text
	jobText, err := getJobText(url)
	if err != nil {
		return err
	}

	// Extract using AI
	log("Extracting with %s...", model)
	data, err := extractWithSchema(ctx, model, sch, url, jobText)
	if err != nil {
		return err
	}

	// Display result
	title := sch.GetTitle(data)
	printDynamicResult(title, data)

	if dryRunFlag {
		log("Dry run - no issue created")
		return nil
	}

	// Create GitHub issue
	return createDynamicIssue(repo, sch, title, data)
}

func getJobText(url string) (string, error) {
	if textFlag != "" {
		log("Using provided text")
		return textFlag, nil
	}

	log("Fetching %s", url)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "cvx/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	return string(body), nil
}

func extractWithSchema(ctx context.Context, model string, sch *schema.Schema, url, jobText string) (map[string]any, error) {
	client, err := ai.NewClient(model)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	prompt := sch.GeneratePrompt(url, jobText)

	resp, err := client.GenerateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Clean markdown code blocks
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var data map[string]any
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return nil, fmt.Errorf("parse failed: %w\nResponse: %s", err, resp)
	}

	return data, nil
}

func printDynamicResult(title string, data map[string]any) {
	fmt.Printf("\n%s\n", title)

	// Print company if available
	if company, ok := data["company"]; ok && company != nil {
		fmt.Printf("Company: %v\n", company)
	}

	// Print location if available
	if location, ok := data["location"]; ok && location != nil {
		fmt.Printf("Location: %v\n", location)
	}

	fmt.Println()
}

func createDynamicIssue(repo string, sch *schema.Schema, title string, data map[string]any) error {
	body := sch.BuildIssueBody(data)

	ghArgs := []string{
		"issue", "create",
		"-R", repo,
		"--title", title,
		"--body", body,
	}

	log("Creating issue in %s...", repo)

	gh := exec.Command("gh", ghArgs...)
	gh.Stdout = os.Stdout
	gh.Stderr = os.Stderr

	if err := gh.Run(); err != nil {
		return fmt.Errorf("gh issue create failed: %w", err)
	}

	return nil
}
