package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/project"
	"github.com/xrsl/cvx/pkg/schema"
	"github.com/xrsl/cvx/pkg/style"
)

var (
	agentFlag  string
	repoFlag   string
	schemaFlag string
	bodyFlag   string
	dryRunFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a job application",
	Long: `Fetch job posting, extract details with AI, and create a GitHub issue.

Fields are extracted based on schema (GitHub issue template YAML).
Use --body to read job posting from a file instead of fetching URL.

Examples:
  cvx add https://company.com/job
  cvx add https://company.com/job --dry-run
  cvx add https://company.com/job -a gemini
  cvx add https://company.com/job --body        # use .cvx/body.md
  cvx add https://company.com/job --body job.md # use custom file`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "AI agent (overrides config)")
	addCmd.Flags().StringVarP(&repoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	addCmd.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Schema file (overrides config)")
	addCmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "Read job posting from file (default: .cvx/body.md)")
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
	cfg, _, err := config.LoadWithCache()
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

	// Resolve body file path if flag was used
	var bodyPath string
	if cmd.Flags().Changed("body") {
		bodyPath = bodyFlag
		if bodyPath == "" {
			bodyPath = ".cvx/body.md"
		}
	}

	// Get job text
	jobText, err := getJobText(url, bodyPath)
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

func getJobText(url, bodyPath string) (string, error) {
	// Use body file if specified
	if bodyPath != "" {
		content, err := os.ReadFile(bodyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", bodyPath, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			return "", fmt.Errorf("%s is empty", bodyPath)
		}
		log("Using job posting from %s", bodyPath)
		return string(content), nil
	}

	log("Fetching %s", url)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "cvx/"+getVersion())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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

	var resp string

	// Use prompt caching if client supports it (Claude API)
	if cachingClient, ok := client.(ai.CachingClient); ok {
		systemPrompt, userPrompt := sch.GeneratePromptParts(url, jobText)
		resp, err = cachingClient.GenerateContentWithSystem(ctx, systemPrompt, userPrompt)
	} else {
		prompt := sch.GeneratePrompt(url, jobText)
		resp, err = client.GenerateContent(ctx, prompt)
	}
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

func printDynamicResult(title string, data map[string]any) {
	company := data["company"]
	location := data["location"]

	fmt.Printf("\n%s", style.C(style.Green, title))
	if company != nil {
		fmt.Printf(" @ %s", style.C(style.Cyan, fmt.Sprintf("%v", company)))
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
	fmt.Printf("%s%s\n", style.Success("Created"), issueURL)

	// Add to project if configured
	_, projectCache, _ := config.LoadWithCache()
	if projectCache != nil && projectCache.ID != "" {
		if err := addToProject(projectCache, repo, issueURL, data); err != nil {
			fmt.Printf("Warning: Could not add to project: %v\n", err)
		}
	}

	return nil
}

func addToProject(proj *config.ProjectCache, repo, issueURL string, data map[string]any) error {
	// Extract issue number from URL
	re := regexp.MustCompile(`/issues/(\d+)$`)
	matches := re.FindStringSubmatch(issueURL)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract issue number from URL")
	}
	issueNum := 0
	if _, err := fmt.Sscanf(matches[1], "%d", &issueNum); err != nil {
		return fmt.Errorf("failed to parse issue number: %w", err)
	}

	client := project.New(repo)

	// Get issue node ID
	nodeID, err := client.GetIssueNodeID(issueNum)
	if err != nil {
		return fmt.Errorf("failed to get issue node ID: %w", err)
	}

	// Add to project
	itemID, err := client.AddItem(proj.ID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to add to project: %w", err)
	}

	fmt.Printf("%s%s\n", style.Success("Added to project"), proj.Title)

	// Set Company field
	company := ""
	if c, ok := data["company"].(string); ok && c != "" && proj.Fields.Company != "" {
		company = c
		if err := client.SetTextField(proj.ID, itemID, proj.Fields.Company, company); err != nil {
			log("Warning: Could not set company field: %v", err)
		}
	}

	// Set Deadline field (default +7 days if not provided)
	deadline := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	if d, ok := data["deadline"].(string); ok && d != "" {
		deadline = d
	}
	if proj.Fields.Deadline != "" {
		if err := client.SetDateField(proj.ID, itemID, proj.Fields.Deadline, deadline); err != nil {
			log("Warning: Could not set deadline field: %v", err)
		}
	}

	// Set initial status to "To be Applied"
	if proj.Fields.Status != "" {
		if statusID, ok := proj.Statuses["to_be_applied"]; ok {
			if err := client.SetStatusField(proj.ID, itemID, proj.Fields.Status, statusID); err != nil {
				log("Warning: Could not set status field: %v", err)
			}
		}
	}

	// Print fields that were set
	fmt.Printf("%scompany, deadline: %s\n", style.Success("Set fields"), deadline)

	return nil
}
