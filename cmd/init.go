package cmd

import (
	"bufio"
	"cvx/pkg/ai"
	"cvx/pkg/config"
	"cvx/pkg/project"
	"cvx/pkg/workflow"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	initReset = "\033[0m"
	initGreen = "\033[0;32m"
	initCyan  = "\033[0;36m"
	initGray  = "\033[90m"
	initBold  = "\033[1m"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize cvx for this repository",
	Long: `Initialize cvx configuration and directory structure.

Creates:
  .cvx-config.yaml     Configuration file
  .cvx/workflows/      Workflow definitions
  .cvx/sessions/       Agent session files
  .cvx/matches/        Match analysis outputs

Run this once per repository to set up cvx.`,
	RunE: runInit,
}

var (
	initResetWorkflowsFlag bool
	initDeleteFlag         bool
	initQuietFlag          bool
	initCheckFlag          bool
)

func init() {
	initCmd.Flags().BoolVarP(&initResetWorkflowsFlag, "reset-workflows", "r", false, "Reset workflows to defaults")
	initCmd.Flags().BoolVarP(&initDeleteFlag, "delete", "d", false, "Remove .cvx/ and config file")
	initCmd.Flags().BoolVarP(&initQuietFlag, "quiet", "q", false, "Non-interactive with defaults")
	initCmd.Flags().BoolVarP(&initCheckFlag, "check", "c", false, "Validate config resources exist")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Handle --delete flag
	if initDeleteFlag {
		os.RemoveAll(".cvx")
		os.Remove(config.Path())
		fmt.Printf("%s✓%s Deleted\n", initGreen, initReset)
		return nil
	}

	// Handle --check flag
	if initCheckFlag {
		cfg, _, err := config.LoadWithCache()
		if err != nil {
			return fmt.Errorf("no config found: %w", err)
		}
		validateConfig(cfg)
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	// Handle --reset-workflows flag
	if initResetWorkflowsFlag {
		if err := workflow.ResetWorkflows(); err != nil {
			return fmt.Errorf("failed to reset workflows: %w", err)
		}
		fmt.Printf("%s✓%s Workflows reset to defaults\n", initGreen, initReset)
		return nil
	}

	// Handle --quiet flag
	if initQuietFlag {
		repo := inferRepo()
		if repo == "" {
			return fmt.Errorf("could not infer repo from git remote")
		}
		// Extract owner for project
		owner := ""
		if parts := strings.Split(repo, "/"); len(parts) == 2 {
			owner = parts[0]
		}
		cfg := &config.Config{
			Repo:          repo,
			Agent:         "claude",
			Schema:        workflow.DefaultSchemaPath,
			CVPath:        "src/cv.tex",
			ReferencePath: "reference/",
			Project:       owner + "/1",
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		if err := workflow.Init(workflow.DefaultSchemaPath); err != nil {
			return fmt.Errorf("failed to init workflows: %w", err)
		}
		fmt.Printf("%s✓%s Initialized with defaults\n", initGreen, initReset)
		return nil
	}

	// Check if already initialized
	_, configExists := os.Stat(config.Path())
	_, cvxDirExists := os.Stat(".cvx")

	if configExists == nil && cvxDirExists == nil {
		fmt.Printf("%s✓%s Already initialized\n", initGreen, initReset)
		fmt.Printf("  Config: %s%s%s\n\n", initGray, config.Path(), initReset)

		// Ensure workflow files are up to date
		cfg, _ := config.Load()
		schemaPath := cfg.Schema
		if schemaPath == "" {
			schemaPath = workflow.DefaultSchemaPath
		}
		if err := workflow.Init(schemaPath); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		}
		return nil
	}

	cfg, _ := config.Load()

	fmt.Printf("\n%sPress Enter to accept defaults shown in brackets.%s\n\n", initGray, initReset)

	// Step 1: Repository
	repo := cfg.Repo
	if repo == "" {
		repo = inferRepo()
	}
	for {
		fmt.Printf("%s?%s Repository ", initGreen, initReset)
		if repo != "" {
			fmt.Printf("%s[%s]%s: ", initCyan, repo, initReset)
		} else {
			fmt.Printf("%s[owner/repo]%s: ", initCyan, initReset)
		}

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" && repo != "" {
			fmt.Println()
			break
		} else if input != "" {
			repo = input
		} else {
			fmt.Println("  Repository is required")
			continue
		}

		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			fmt.Println("  Invalid format (expected owner/repo)")
			repo = cfg.Repo
			continue
		}

		fmt.Print("  Checking... ")
		check := exec.Command("gh", "repo", "view", repo, "--json", "name")
		if err := check.Run(); err != nil {
			fmt.Println("repository not found or no access")
			repo = cfg.Repo
			continue
		}
		fmt.Printf("%s✓%s\n\n", initGreen, initReset)
		break
	}
	config.Set("repo", repo)

	// Step 2: AI Agent
	agents := buildAgentList()
	defaultAgent := agents[0].name
	currentAgent := cfg.Agent
	if currentAgent == "" {
		currentAgent = defaultAgent
	}

	currentIdx := 0
	for i, a := range agents {
		if a.name == currentAgent {
			currentIdx = i
			break
		}
	}

	fmt.Printf("%s?%s AI Agent\n", initGreen, initReset)
	for i, a := range agents {
		marker := "   "
		if i == currentIdx {
			marker = fmt.Sprintf("  %s→%s", initGreen, initReset)
		}
		fmt.Printf("%s%s%d)%s %s", marker, initCyan, i+1, initReset, a.name)
		if a.note != "" {
			fmt.Printf(" %s(%s)%s", initGray, a.note, initReset)
		}
		fmt.Println()
	}
	fmt.Printf("\n  Choice %s[%d]%s: ", initCyan, currentIdx+1, initReset)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedAgent := currentAgent
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(agents) {
			selectedAgent = agents[idx-1].name
		}
	}

	config.Set("agent", selectedAgent)
	fmt.Printf("  Using %s%s%s\n\n", initCyan, selectedAgent, initReset)

	// Step 3: CV path (for match command)
	cvPath := cfg.CVPath
	if cvPath == "" {
		cvPath = "src/cv.tex"
	}
	fmt.Printf("%s?%s CV file path %s[%s]%s: ", initGreen, initReset, initCyan, cvPath, initReset)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cvPath = input
	}
	config.Set("cv_path", cvPath)
	fmt.Println()

	// Step 4: Reference directory path
	refPath := cfg.ReferencePath
	if refPath == "" {
		refPath = "reference/"
	}
	fmt.Printf("%s?%s Reference directory %s[%s]%s: ", initGreen, initReset, initCyan, refPath, initReset)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		refPath = input
	}
	config.Set("reference_path", refPath)
	fmt.Println()

	// Step 5: Job ad schema path
	schemaPath := cfg.Schema
	if schemaPath == "" {
		schemaPath = workflow.DefaultSchemaPath
	}
	fmt.Printf("%s?%s Job ad schema (i.e., issue template) path %s[%s]%s: ", initGreen, initReset, initCyan, schemaPath, initReset)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		schemaPath = input
	}
	config.Set("schema", schemaPath)
	fmt.Println()

	// Step 6: GitHub Project
	if cfg.Project == "" {
		fmt.Printf("%s?%s GitHub Project %s(number to use existing, 'new' to create, enter to skip)%s: ", initGreen, initReset, initCyan, initReset)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		client := project.New(repo)

		if input == "new" {
			fmt.Printf("Creating project... ")
			proj, fields, err := client.Create("Job Applications", nil)
			if err != nil {
				fmt.Printf("\n  %sFailed:%s %v\n", initCyan, initReset, err)
			} else {
				fmt.Printf("%s✓%s\n", initGreen, initReset)
				saveProjectConfig(proj, fields, repo)
				fmt.Printf("  Project #%d created\n", proj.Number)
			}
		} else if projNum, err := strconv.Atoi(input); err == nil {
			fmt.Printf("Linking project #%d... ", projNum)
			projects, err := client.ListProjects()
			if err != nil {
				fmt.Printf("\n  %sFailed:%s %v\n", initCyan, initReset, err)
			} else {
				var found *project.ProjectInfo
				for _, p := range projects {
					if p.Number == projNum {
						found = &p
						break
					}
				}
				if found == nil {
					fmt.Printf("\n  %sNot found:%s Project #%d doesn't exist\n", initCyan, initReset, projNum)
				} else {
					fields, err := client.DiscoverFields(found.ID)
					if err != nil {
						fmt.Printf("\n  %sFailed:%s %v\n", initCyan, initReset, err)
					} else {
						fmt.Printf("%s✓%s\n", initGreen, initReset)
						saveProjectConfig(found, fields, repo)
						fmt.Printf("  Linked to \"%s\"\n", found.Title)
					}
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s✓%s GitHub Project %s linked\n\n", initGreen, initReset, cfg.Project)
	}

	// Initialize .cvx directory structure
	if err := workflow.Init(schemaPath); err != nil {
		fmt.Printf("  Warning: Could not initialize .cvx/ directory: %v\n", err)
	}

	fmt.Printf("%s%sReady!%s\n", initBold, initGreen, initReset)
	fmt.Printf("  %scvx add <job-url>%s    Add a job posting\n", initCyan, initReset)
	fmt.Printf("  %scvx advise <issue>%s   Analyze job match\n\n", initCyan, initReset)
	return nil
}

type agentOption struct {
	name string
	note string
}

func saveProjectConfig(proj *project.ProjectInfo, fields map[string]project.FieldInfo, repo string) {
	// Extract owner from repo (owner/name)
	owner := ""
	if parts := strings.Split(repo, "/"); len(parts) == 2 {
		owner = parts[0]
	}

	cache := config.ProjectCache{
		ID:       proj.ID,
		Title:    proj.Title,
		Statuses: make(map[string]string),
		Fields:   config.FieldIDs{},
	}

	// Look for status field - prefer custom "Application Status" over default "Status"
	var statusField *project.FieldInfo
	if f, ok := fields["application_status"]; ok {
		statusField = &f
	} else if f, ok := fields["status"]; ok {
		statusField = &f
	}
	if statusField != nil {
		cache.Fields.Status = statusField.ID
		for _, opt := range statusField.Options {
			key := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
			cache.Statuses[key] = opt.ID
		}
	}
	if f, ok := fields["company"]; ok {
		cache.Fields.Company = f.ID
	}
	if f, ok := fields["deadline"]; ok {
		cache.Fields.Deadline = f.ID
	}
	if f, ok := fields["applied_date"]; ok {
		cache.Fields.AppliedDate = f.ID
	}

	config.SaveProject(owner, proj.Number, cache)
}

func inferRepo() string {
	// Try git remote origin first
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		url := strings.TrimSpace(string(out))
		// Parse git@github.com:owner/repo.git or https://github.com/owner/repo.git
		url = strings.TrimSuffix(url, ".git")
		if strings.Contains(url, "github.com") {
			if strings.HasPrefix(url, "git@") {
				// git@github.com:owner/repo
				parts := strings.Split(url, ":")
				if len(parts) == 2 {
					return parts[1]
				}
			} else {
				// https://github.com/owner/repo
				parts := strings.Split(url, "github.com/")
				if len(parts) == 2 {
					return parts[1]
				}
			}
		}
	}

	// Fallback: gh user + current directory name
	user, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
	if err != nil {
		return ""
	}
	username := strings.TrimSpace(string(user))

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dirname := filepath.Base(wd)

	return username + "/" + dirname
}

func buildAgentList() []agentOption {
	var agents []agentOption

	if ai.IsClaudeCLIAvailable() {
		agents = append(agents, agentOption{"claude", ""})
	}

	if ai.IsGeminiCLIAvailable() {
		agents = append(agents, agentOption{"gemini", ""})
	}

	for _, a := range ai.SupportedAgents() {
		if a == "claude" || a == "gemini" {
			continue
		}
		note := ""
		if strings.HasPrefix(a, "gemini-") {
			note = "GEMINI_API_KEY"
		} else if strings.HasPrefix(a, "claude-") {
			note = "ANTHROPIC_API_KEY"
		}
		agents = append(agents, agentOption{a, note})
	}

	return agents
}

func validateConfig(cfg *config.Config) {
	warn := func(msg string) {
		fmt.Printf("  %sWarning:%s %s\n", initCyan, initReset, msg)
	}

	// Check repo access
	if cfg.Repo != "" {
		cmd := exec.Command("gh", "repo", "view", cfg.Repo, "--json", "name")
		if err := cmd.Run(); err != nil {
			warn(fmt.Sprintf("repo %s not accessible", cfg.Repo))
		}
	}

	// Check cv_path exists
	if cfg.CVPath != "" {
		if _, err := os.Stat(cfg.CVPath); os.IsNotExist(err) {
			warn(fmt.Sprintf("cv_path %s does not exist", cfg.CVPath))
		}
	}

	// Check reference_path exists
	if cfg.ReferencePath != "" {
		if _, err := os.Stat(cfg.ReferencePath); os.IsNotExist(err) {
			warn(fmt.Sprintf("reference_path %s does not exist", cfg.ReferencePath))
		}
	}

	// Check project exists (Projects v2 via GraphQL)
	if cfg.Project != "" {
		owner := cfg.ProjectOwner()
		number := cfg.ProjectNumber()
		if number > 0 {
			query := fmt.Sprintf(`query { user(login: "%s") { projectV2(number: %d) { id } } }`, owner, number)
			cmd := exec.Command("gh", "api", "graphql", "-f", "query="+query, "--jq", ".data.user.projectV2.id")
			out, err := cmd.Output()
			if err != nil || strings.TrimSpace(string(out)) == "" || strings.TrimSpace(string(out)) == "null" {
				// Try org project
				query = fmt.Sprintf(`query { organization(login: "%s") { projectV2(number: %d) { id } } }`, owner, number)
				cmd = exec.Command("gh", "api", "graphql", "-f", "query="+query, "--jq", ".data.organization.projectV2.id")
				out, err = cmd.Output()
				if err != nil || strings.TrimSpace(string(out)) == "" || strings.TrimSpace(string(out)) == "null" {
					warn(fmt.Sprintf("project %s not found", cfg.Project))
				}
			}
		}
	}

	// Check schema exists (if not default)
	if cfg.Schema != "" && cfg.Schema != workflow.DefaultSchemaPath {
		if _, err := os.Stat(cfg.Schema); os.IsNotExist(err) {
			warn(fmt.Sprintf("schema %s does not exist", cfg.Schema))
		}
	}
}
