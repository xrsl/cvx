package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/project"
	"github.com/xrsl/cvx/pkg/schema"
	"github.com/xrsl/cvx/pkg/style"
)

var (
	// add-specific flags (agentFlag and modelFlag are global in root.go)
	addRepoFlag            string
	addSchemaFlag          string
	addBodyFlag            string
	addDryRunFlag          bool
	addCallAPIDirectlyFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a job application",
	Long: `Fetch job posting, extract details with AI, and create a GitHub issue.

Fields are extracted based on schema (GitHub issue template YAML).
Use --body to read job posting from a file instead of fetching URL.

Two modes (like cvx build):
  1. CLI agent (default): Uses claude/gemini CLI tools
  2. API mode (-m flag): Direct API access with specified model

Examples:
  cvx add https://company.com/job                 # CLI agent (default)
  cvx add https://company.com/job -m sonnet-4     # Claude API
  cvx add https://company.com/job -m flash-2-5    # Gemini API
  cvx add https://company.com/job -a gemini       # Gemini CLI agent
  cvx add https://company.com/job --body          # Read from .cvx/body.md
  cvx add https://company.com/job --dry-run       # Extract only`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	// Note: -m and -a are global flags defined in root.go
	addCmd.Flags().BoolVar(&addCallAPIDirectlyFlag, "call-api-directly", false, "Deprecated: use -m flag instead")
	addCmd.Flags().StringVarP(&addRepoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	addCmd.Flags().StringVarP(&addSchemaFlag, "schema", "s", "", "Schema file (overrides config)")
	addCmd.Flags().StringVarP(&addBodyFlag, "body", "b", "", "Read job posting from file (default: .cvx/body.md)")
	addCmd.Flags().BoolVar(&addDryRunFlag, "dry-run", false, "Extract only, don't create issue")
	rootCmd.AddCommand(addCmd)
}

func log(format string, args ...any) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	url := args[0]

	// Load config
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Resolve repo (flag > config)
	repo, err := resolveAddRepo(cfg)
	if err != nil {
		return err
	}

	// Resolve schema (flag > config > default)
	sch, err := resolveAddSchema(cfg)
	if err != nil {
		return err
	}

	// Resolve body file path if flag was used
	bodyPath := resolveAddBodyPath(cmd)

	// Get job text
	jobText, err := getJobText(ctx, url, bodyPath)
	if err != nil {
		return err
	}

	// Extract using AI - agent for -m flag, CLI for -a flag
	var data map[string]any
	if modelFlag != "" {
		// Agent mode (API)
		if err := os.Setenv("AI_MODEL", modelFlag); err != nil {
			return fmt.Errorf("failed to set AI_MODEL: %w", err)
		}
		log("Extracting with agent (%s)...", modelFlag)
		data, err = extractWithAgent(sch, url, jobText)
	} else {
		// CLI agent mode
		var agent string
		agent, err = resolveAddCLIAgent(cfg)
		if err != nil {
			return err
		}
		log("Extracting with CLI agent (%s)...", agent)
		data, err = extractWithCLI(ctx, agent, sch, url, jobText)
	}
	if err != nil {
		return err
	}

	// Display result
	title := sch.GetTitle(data)
	printDynamicResult(title, data)

	if addDryRunFlag {
		log("Dry run - no issue created")
		return nil
	}

	// Create GitHub issue
	return createDynamicIssue(repo, sch, title, data)
}

func resolveAddRepo(cfg *config.Config) (string, error) {
	repo := addRepoFlag
	if repo == "" {
		repo = cfg.GitHub.Repo
	}
	if repo == "" && !addDryRunFlag {
		return "", fmt.Errorf("no repo configured. Run: cvx init")
	}
	return repo, nil
}

func resolveAddCLIAgent(cfg *config.Config) (string, error) {
	if agentFlag != "" {
		if !ai.IsCLIAgentSupported(agentFlag) {
			return "", fmt.Errorf("unsupported CLI agent: %s (supported: claude, gemini). Use -m for API access", agentFlag)
		}
		return agentFlag, nil
	}
	if cfg.Agent.Default != "" {
		return cfg.Agent.Default, nil
	}
	return "", fmt.Errorf("no CLI agent configured. Run: cvx init")
}

func resolveAddSchema(cfg *config.Config) (*schema.Schema, error) {
	schemaPath := addSchemaFlag
	if schemaPath == "" {
		schemaPath = cfg.Schema.JobAd
	}

	sch, err := schema.Load(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schema error: %w", err)
	}

	return sch, nil
}

func resolveAddBodyPath(cmd *cobra.Command) string {
	if !cmd.Flags().Changed("body") {
		return ""
	}
	if addBodyFlag == "" {
		return ".cvx/body.md"
	}
	return addBodyFlag
}

func getJobText(ctx context.Context, url, bodyPath string) (string, error) {
	// Use body file if specified
	if bodyPath != "" {
		content, err := os.ReadFile(bodyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", bodyPath, err)
		}
		if strings.TrimSpace(string(content)) == "" {
			return "", fmt.Errorf("%s is empty", bodyPath)
		}
		log("Using job posting from %s", bodyPath)
		return string(content), nil
	}

	log("Fetching %s", url)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
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

	return cleanHTML(string(body))
}

func cleanHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove unwanted elements
	doc.Find("script, style, nav, footer, header").Remove()

	// Extract text
	text := doc.Text()

	// Clean up whitespace
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	result := strings.Join(cleaned, "\n")
	log("Extracted %d chars (cleaned from HTML)", len(result))
	return result, nil
}

// extractWithAgent uses the pydantic-ai agent for extraction
func extractWithAgent(sch *schema.Schema, url, jobText string) (map[string]any, error) {
	schemaPrompt, _ := sch.GeneratePromptParts(url, jobText)

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
				msg := fmt.Sprintf("Extracting job details using 🤖 %s...", modelFlag)
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	data, err := callAgentExtract(jobText, url, schemaPrompt)

	done <- true
	close(done)

	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return data, nil
}

// extractWithCLI uses CLI agents (claude/gemini CLI) for extraction
func extractWithCLI(ctx context.Context, agent string, sch *schema.Schema, url, jobText string) (map[string]any, error) {
	client, err := ai.NewClient(agent)
	if err != nil {
		return nil, err
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
				msg := fmt.Sprintf("Extracting job details using 🤖 %s...", agent)
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	var resp string

	// Use prompt caching if client supports it (Claude API)
	if cachingClient, ok := client.(ai.CachingClient); ok {
		systemPrompt, userPrompt := sch.GeneratePromptParts(url, jobText)
		resp, err = cachingClient.GenerateContentWithSystem(ctx, systemPrompt, userPrompt)
	} else {
		prompt := sch.GeneratePrompt(url, jobText)
		resp, err = client.GenerateContent(ctx, prompt)
	}

	done <- true
	close(done)

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

	cli := gh.New()
	output, err := cli.IssueCreate(repo, title, body)
	if err != nil {
		return err
	}

	issueURL := strings.TrimSpace(output)
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
