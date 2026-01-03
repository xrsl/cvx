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
  cvx.toml            Configuration file
  .cvx/workflows/     Workflow definitions
  .cvx/sessions/      Agent session files
  .cvx/matches/       Match analysis outputs

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
		owner := ""
		if parts := strings.Split(repo, "/"); len(parts) == 2 {
			owner = parts[0]
		}
		cfg := &config.Config{
			GitHub: config.GitHubConfig{Repo: repo, Project: owner + "/1"},
			Agent:  config.AgentConfig{Default: "claude"},
			Schema: config.SchemaConfig{JobAd: workflow.DefaultSchemaPath},
			Paths:  config.PathsConfig{Reference: "reference/"},
			CV:     config.CVConfig{Source: "src/cv.yaml", Output: "out/cv.pdf", Schema: "schema/schema.json"},
			Letter: config.LetterConfig{Source: "src/letter.yaml", Output: "out/letter.pdf", Schema: "schema/schema.json"},
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
		cfg, _ := config.Load()
		schemaPath := cfg.Schema.JobAd
		if schemaPath == "" {
			schemaPath = workflow.DefaultSchemaPath
		}
		if err := workflow.Init(schemaPath); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		}
		return nil
	}

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	fmt.Printf("\n%s\n\n", style.C(style.Gray, "Press Enter to accept defaults shown in brackets."))

	// Step 1: Repository
	repo := cfg.GitHub.Repo
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
			repo = cfg.GitHub.Repo
			continue
		}

		fmt.Print("  Checking... ")
		cli := gh.New()
		if _, err := cli.RepoView(repo, []string{"name"}); err != nil {
			fmt.Println("repository not found or no access")
			repo = cfg.GitHub.Repo
			continue
		}
		fmt.Printf("%s\n\n", style.C(style.Green, "✓"))
		break
	}
	cfg.GitHub.Repo = repo

	// Step 2: CLI Agent
	agents := buildAgentList()
	defaultAgent := agents[0].name
	currentAgent := cfg.Agent.Default
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

	fmt.Printf("%s CLI Agent\n", style.C(style.Green, "?"))
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

	cfg.Agent.Default = selectedAgent
	fmt.Printf("  Using %s\n\n", style.C(style.Cyan, selectedAgent))

	// Step 3: CV source path
	cvSource := cfg.CV.Source
	if cvSource == "" {
		cvSource = "src/cv.yaml"
	}
	fmt.Printf("%s CV source %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+cvSource+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cvSource = input
	}
	cfg.CV.Source = cvSource
	if cfg.CV.Output == "" {
		cfg.CV.Output = "out/cv.pdf"
	}
	if cfg.CV.Schema == "" {
		cfg.CV.Schema = "schema/schema.json"
	}
	fmt.Println()

	// Step 4: Letter source path
	letterSource := cfg.Letter.Source
	if letterSource == "" {
		letterSource = "src/letter.yaml"
	}
	fmt.Printf("%s Letter source %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+letterSource+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		letterSource = input
	}
	cfg.Letter.Source = letterSource
	if cfg.Letter.Output == "" {
		cfg.Letter.Output = "out/letter.pdf"
	}
	if cfg.Letter.Schema == "" {
		cfg.Letter.Schema = "schema/schema.json"
	}
	fmt.Println()

	// Step 5: Reference directory
	refPath := cfg.Paths.Reference
	if refPath == "" {
		refPath = "reference/"
	}
	fmt.Printf("%s Reference directory %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+refPath+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		refPath = input
	}
	cfg.Paths.Reference = refPath
	fmt.Println()

	// Step 6: Job ad schema
	jobAdSchema := cfg.Schema.JobAd
	if jobAdSchema == "" {
		jobAdSchema = workflow.DefaultSchemaPath
	}
	fmt.Printf("%s Job ad schema %s: ", style.C(style.Green, "?"), style.C(style.Cyan, "["+jobAdSchema+"]"))
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		jobAdSchema = input
	}
	cfg.Schema.JobAd = jobAdSchema
	fmt.Println()

	// Step 7: GitHub Project
	if cfg.GitHub.Project == "" {
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
		fmt.Printf("%s GitHub Project %s linked\n\n", style.C(style.Green, "✓"), cfg.GitHub.Project)
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Initialize .cvx directory structure
	if err := workflow.Init(cfg.Schema.JobAd); err != nil {
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
		agents = append(agents, agentOption{"claude", ""})
	}

	if ai.IsGeminiCLIAvailable() {
		agents = append(agents, agentOption{"gemini", ""})
	}

	return agents
}

func validateConfig(cfg *config.Config) {
	warn := func(msg string) {
		fmt.Printf("  %s %s\n", style.C(style.Yellow, "Warning:"), msg)
	}

	cli := gh.New()

	if cfg.GitHub.Repo != "" {
		if _, err := cli.RepoView(cfg.GitHub.Repo, []string{"name"}); err != nil {
			warn(fmt.Sprintf("repo %s not accessible", cfg.GitHub.Repo))
		}
	}

	if cfg.Paths.Reference != "" {
		if _, err := os.Stat(cfg.Paths.Reference); os.IsNotExist(err) {
			warn(fmt.Sprintf("reference path %s does not exist", cfg.Paths.Reference))
		}
	}

	if cfg.GitHub.Project != "" {
		owner := cfg.ProjectOwner()
		number := cfg.ProjectNumber()
		if number > 0 {
			query := fmt.Sprintf(`query { user(login: "%s") { projectV2(number: %d) { id } } }`, owner, number)
			out, err := cli.GraphQLWithJQ(query, ".data.user.projectV2.id")
			if err != nil || strings.TrimSpace(string(out)) == "" || strings.TrimSpace(string(out)) == "null" {
				query = fmt.Sprintf(`query { organization(login: "%s") { projectV2(number: %d) { id } } }`, owner, number)
				out, err = cli.GraphQLWithJQ(query, ".data.organization.projectV2.id")
				if err != nil || strings.TrimSpace(string(out)) == "" || strings.TrimSpace(string(out)) == "null" {
					warn(fmt.Sprintf("project %s not found", cfg.GitHub.Project))
				}
			}
		}
	}

	if cfg.Schema.JobAd != "" && cfg.Schema.JobAd != workflow.DefaultSchemaPath {
		if _, err := os.Stat(cfg.Schema.JobAd); os.IsNotExist(err) {
			warn(fmt.Sprintf("job-ad schema %s does not exist", cfg.Schema.JobAd))
		}
	}
}
