package cmd

import (
	"context"
	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/project"
	"cvx/pkg/schema"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	textFlag   string
	agentFlag  string
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
  cvx add https://company.com/job -a claude-sonnet-4
  cvx add https://company.com/job -s /path/to/job-app.yml`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&textFlag, "text", "t", "", "Job posting text (skips URL fetch)")
	addCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "AI agent (overrides config)")
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

	// Load config with cached project IDs
	cfg, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve repo (flag > config)
	repo := repoFlag
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" && !dryRunFlag {
		return fmt.Errorf("no repo configured. Run: cvx config")
	}

	// Resolve agent (flag > config > default)
	agent := agentFlag
	if agent == "" {
		agent = cfg.Agent
	}
	if agent == "" {
		agent = ai.DefaultAgent()
	}

	// Validate agent
	if !ai.IsAgentSupported(agent) {
		return fmt.Errorf("unsupported agent: %s (supported: %v)", agent, ai.SupportedAgents())
	}

	// Resolve schema (flag > config > default)
	schemaPath := schemaFlag
	if schemaPath == "" {
		schemaPath = cfg.Schema
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
	log("Extracting with %s...", agent)
	data, err := extractWithSchema(ctx, agent, sch, url, jobText)
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

func extractWithSchema(ctx context.Context, agent string, sch *schema.Schema, url, jobText string) (map[string]any, error) {
	client, err := ai.NewClient(agent)
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
	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSpace(resp)
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var data map[string]any
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return nil, fmt.Errorf("parse failed: %w\nResponse: %s", err, resp)
	}

	return data, nil
}

const (
	cReset = "\033[0m"
	cGreen = "\033[0;32m"
	cCyan  = "\033[0;36m"
)

func printDynamicResult(title string, data map[string]any) {
	company, _ := data["company"]
	location, _ := data["location"]

	fmt.Printf("\n%s%s%s", cGreen, title, cReset)
	if company != nil {
		fmt.Printf(" @ %s%v%s", cCyan, company, cReset)
	}
	if location != nil {
		fmt.Printf(" (%v)", location)
	}
	fmt.Println()
}

func createDynamicIssue(repo string, sch *schema.Schema, title string, data map[string]any) error {
	body := sch.BuildIssueBody(data)

	gh := exec.Command("gh", "issue", "create", "-R", repo, "--title", title, "--body", body)
	output, err := gh.Output()
	if err != nil {
		return fmt.Errorf("gh issue create failed: %w", err)
	}

	issueURL := strings.TrimSpace(string(output))
	fmt.Printf("%sCreated:%s %s\n", cGreen, cReset, issueURL)

	// Add to project if configured
	cfg, _ := config.Load()
	if cfg.Project.ID != "" {
		if err := addToProject(cfg, repo, issueURL, data); err != nil {
			fmt.Printf("Warning: Could not add to project: %v\n", err)
		}
	}

	return nil
}

func addToProject(cfg *config.Config, repo, issueURL string, data map[string]any) error {
	// Extract issue number from URL
	re := regexp.MustCompile(`/issues/(\d+)$`)
	matches := re.FindStringSubmatch(issueURL)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract issue number from URL")
	}
	issueNum := 0
	fmt.Sscanf(matches[1], "%d", &issueNum)

	client := project.New(repo)

	// Get issue node ID
	nodeID, err := client.GetIssueNodeID(issueNum)
	if err != nil {
		return fmt.Errorf("failed to get issue node ID: %w", err)
	}

	// Add to project
	itemID, err := client.AddItem(cfg.Project.ID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to add to project: %w", err)
	}

	// Set Company field
	if company, ok := data["company"].(string); ok && company != "" && cfg.Project.Fields.Company != "" {
		if err := client.SetTextField(cfg.Project.ID, itemID, cfg.Project.Fields.Company, company); err != nil {
			log("Warning: Could not set company field: %v", err)
		}
	}

	// Set Deadline field (default +7 days)
	if cfg.Project.Fields.Deadline != "" {
		deadline := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		if d, ok := data["deadline"].(string); ok && d != "" {
			deadline = d
		}
		if err := client.SetDateField(cfg.Project.ID, itemID, cfg.Project.Fields.Deadline, deadline); err != nil {
			log("Warning: Could not set deadline field: %v", err)
		}
	}

	// Set initial status to "To be Applied"
	if cfg.Project.Fields.Status != "" {
		if statusID, ok := cfg.Project.Statuses["to_be_applied"]; ok {
			if err := client.SetStatusField(cfg.Project.ID, itemID, cfg.Project.Fields.Status, statusID); err != nil {
				log("Warning: Could not set status field: %v", err)
			}
		}
	}

	log("Added to project")
	return nil
}
