package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/project"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
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
		_ = os.RemoveAll(".cvx")
		_ = os.Remove(config.Path())
		fmt.Printf("%s Deleted\n", style.C(style.Green, "✓"))
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
		fmt.Printf("%s Workflows reset to defaults\n", style.C(style.Green, "✓"))
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
		// Auto-detect default CLI agent
		defaultCLI := detectAvailableCLI()
		if defaultCLI == "" {
			defaultCLI = "claude-code" // fallback
		}
		cfg := &config.Config{
			Repo:            repo,
			Agent:           "claude-code",
			DefaultCLIAgent: defaultCLI,
			Schema:          workflow.DefaultSchemaPath,
			CVPath:          "src/cv.tex",
			ReferencePath:   "reference/",
			Project:         owner + "/1",
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		if err := workflow.Init(workflow.DefaultSchemaPath); err != nil {
			return fmt.Errorf("failed to init workflows: %w", err)
		}
		fmt.Printf("%s Initialized with defaults\n", style.C(style.Green, "✓"))
		return nil
	}

	// Check if already initialized
	_, configExists := os.Stat(config.Path())
	_, cvxDirExists := os.Stat(".cvx")

	if configExists == nil && cvxDirExists == nil {
		fmt.Printf("%s Already initialized\n", style.C(style.Green, "✓"))
		fmt.Printf("  Config: %s\n\n", style.C(style.Gray, config.Path()))

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

	fmt.Printf("\n%s\n\n", style.C(style.Gray, "Press Enter to accept defaults shown in brackets."))

	// Step 1: Repository
	repo := cfg.Repo
	if repo == "" {
		repo = inferRepo()
	}
	for {
		fmt.Printf("%s Repository ", style.C(style.Green, "?"))
		if repo != "" {
			fmt.Printf("%s: ", style.C(style.Cyan, "["+repo+"]"))
		} else {
			fmt.Printf("%s: ", style.C(style.Cyan, "[owner/repo]"))
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
		cli := gh.New()
		if _, err := cli.RepoView(repo, []string{"name"}); err != nil {
			fmt.Println("repository not found or no access")
			repo = cfg.Repo
			continue
		}
		fmt.Printf("%s\n\n", style.C(style.Green, "✓"))
		break
	}
	_ = config.Set("repo", repo)

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

	fmt.Printf("%s AI Agent\n", style.C(style.Green, "?"))
	for i, a := range agents {
		marker := "   "
		if i == currentIdx {
			marker = "  " + style.C(style.Green, "→")
		}
		fmt.Printf("%s%s %s", marker, style.C(style.Cyan, fmt.Sprintf("%d)", i+1)), a.name)
		if a.note != "" {
			fmt.Printf(" %s", style.C(style.Gray, "("+a.note+")"))
		}
		fmt.Println()
	}
	fmt.Printf("\n  Choice %s: ", style.C(style.Cyan, fmt.Sprintf("[%d]", currentIdx+1)))

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedAgent := currentAgent
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(agents) {
			selectedAgent = agents[idx-1].name
		}
	}

	_ = config.Set("agent", selectedAgent)
	fmt.Printf("  Using %s\n\n", style.C(style.Cyan, selectedAgent))

	// Step 2.5: Default CLI Agent (for -i mode)
	cliAgents := []agentOption{}
	if isCommandAvailable("claude") {
		cliAgents = append(cliAgents, agentOption{"claude-code", ""})
	}
	if isCommandAvailable("gemini") {
		cliAgents = append(cliAgents, agentOption{"gemini-cli", ""})
	}

	if len(cliAgents) > 0 {
		defaultCLI := cfg.DefaultCLIAgent
		if defaultCLI == "" {
			defaultCLI = cliAgents[0].name
		}

		currentCLIIdx := 0
		for i, a := range cliAgents {
			if a.name == defaultCLI {
				currentCLIIdx = i
				break
			}
		}

		fmt.Printf("%s Default CLI Agent (for -i mode)\n", style.C(style.Green, "?"))
		for i, a := range cliAgents {
			marker := "   "
			if i == currentCLIIdx {
				marker = "  " + style.C(style.Green, "→")
			}
			fmt.Printf("%s%s %s\n", marker, style.C(style.Cyan, fmt.Sprintf("%d)", i+1)), a.name)
		}
		fmt.Printf("\n  Choice %s: ", style.C(style.Cyan, fmt.Sprintf("[%d]", currentCLIIdx+1)))

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		selectedCLI := defaultCLI
		if input != "" {
			if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(cliAgents) {
				selectedCLI = cliAgents[idx-1].name
			}
		}

		_ = config.Set("default_cli_agent", selectedCLI)
		fmt.Printf("  Using %s for interactive mode\n\n", style.C(style.Cyan, selectedCLI))
	}

	// Step 3: CV path (for match command)
	cvPath := cfg.CVPath
	if cvPath == "" {
		cvPath = "src/cv.tex"
	}
	fmt.Printf("%s CV file path %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+cvPath+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cvPath = input
	}
	_ = config.Set("cv_path", cvPath)
	fmt.Println()

	// Step 4: Reference directory path
	refPath := cfg.ReferencePath
	if refPath == "" {
		refPath = "reference/"
	}
	fmt.Printf("%s Reference directory %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+refPath+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		refPath = input
	}
	_ = config.Set("reference_path", refPath)
	fmt.Println()

	// Step 5: Job ad schema path
	schemaPath := cfg.Schema
	if schemaPath == "" {
		schemaPath = workflow.DefaultSchemaPath
	}
	fmt.Printf("%s Job ad schema (issue template) path %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+schemaPath+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		schemaPath = input
	}
	_ = config.Set("schema", schemaPath)
	fmt.Println()

	// Step 6: GitHub Project
	if cfg.Project == "" {
		fmt.Printf("%s GitHub Project %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "(number to use existing, 'new' to create, enter to skip)"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		client := project.New(repo)

		if input == "new" {
			fmt.Printf("Creating project... ")
			proj, fields, err := client.Create("Job Applications", nil)
			if err != nil {
				fmt.Printf("\n  %s %v\n", style.C(style.Cyan, "Failed:"), err)
			} else {
				fmt.Printf("%s\n", style.C(style.Green, "✓"))
				saveProjectConfig(proj, fields, repo)
				fmt.Printf("  Project #%d created\n", proj.Number)
			}
		} else if projNum, err := strconv.Atoi(input); err == nil {
			fmt.Printf("Linking project #%d... ", projNum)
			projects, err := client.ListProjects()
			if err != nil {
				fmt.Printf("\n  %s %v\n", style.C(style.Cyan, "Failed:"), err)
			} else {
				var found *project.ProjectInfo
				for _, p := range projects {
					if p.Number == projNum {
						found = &p
						break
					}
				}
				if found == nil {
					fmt.Printf("\n  %s Project #%d doesn't exist\n", style.C(style.Cyan, "Not found:"), projNum)
				} else {
					fields, err := client.DiscoverFields(found.ID)
					if err != nil {
						fmt.Printf("\n  %s %v\n", style.C(style.Cyan, "Failed:"), err)
					} else {
						fmt.Printf("%s\n", style.C(style.Green, "✓"))
						saveProjectConfig(found, fields, repo)
						fmt.Printf("  Linked to \"%s\"\n", found.Title)
					}
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s GitHub Project %s linked\n\n", style.C(style.Green, "✓"), cfg.Project)
	}

	// Initialize .cvx directory structure
	if err := workflow.Init(schemaPath); err != nil {
		fmt.Printf("  Warning: Could not initialize .cvx/ directory: %v\n", err)
	}

	fmt.Printf("%s\n", style.C(style.Green, style.Bold+"Ready!"))
	fmt.Printf("  %s    Add a job posting\n", style.C(style.Cyan, "cvx add <job-url>"))
	fmt.Printf("  %s   Analyze job match\n\n", style.C(style.Cyan, "cvx advise <issue>"))
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

	_ = config.SaveProject(owner, proj.Number, cache)
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
	cli := gh.New()
	user, err := cli.APIUser()
	if err != nil {
		return ""
	}
	username := strings.TrimSpace(user)

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
		agents = append(agents, agentOption{"claude-code", ""})
	}

	if ai.IsGeminiCLIAvailable() {
		agents = append(agents, agentOption{"gemini-cli", ""})
	}

	for _, a := range ai.SupportedAgents() {
		if a == "claude-code" || a == "gemini-cli" {
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
		fmt.Printf("  %s %s\n", style.C(style.Yellow, "Warning:"), msg)
	}

	cli := gh.New()

	// Check repo access
	if cfg.Repo != "" {
		if _, err := cli.RepoView(cfg.Repo, []string{"name"}); err != nil {
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
			out, err := cli.GraphQLWithJQ(query, ".data.user.projectV2.id")
			if err != nil || strings.TrimSpace(string(out)) == "" || strings.TrimSpace(string(out)) == "null" {
				// Try org project
				query = fmt.Sprintf(`query { organization(login: "%s") { projectV2(number: %d) { id } } }`, owner, number)
				out, err = cli.GraphQLWithJQ(query, ".data.organization.projectV2.id")
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
