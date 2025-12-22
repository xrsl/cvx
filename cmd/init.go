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

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

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

	// Step 2: AI Model
	models := buildModelList()
	defaultModel := models[0].name
	currentModel := cfg.Model
	if currentModel == "" {
		currentModel = defaultModel
	}

	currentIdx := 0
	for i, m := range models {
		if m.name == currentModel {
			currentIdx = i
			break
		}
	}

	fmt.Printf("%s?%s AI Model\n", initGreen, initReset)
	for i, m := range models {
		marker := "   "
		if i == currentIdx {
			marker = fmt.Sprintf("  %s→%s", initGreen, initReset)
		}
		fmt.Printf("%s%s%d)%s %s", marker, initCyan, i+1, initReset, m.name)
		if m.note != "" {
			fmt.Printf(" %s(%s)%s", initGray, m.note, initReset)
		}
		fmt.Println()
	}
	fmt.Printf("\n  Choice %s[%d]%s: ", initCyan, currentIdx+1, initReset)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedModel := currentModel
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(models) {
			selectedModel = models[idx-1].name
		}
	}

	config.Set("model", selectedModel)
	fmt.Printf("  Using %s%s%s\n\n", initCyan, selectedModel, initReset)

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

	// Step 4: Experience file path
	expPath := cfg.ExperiencePath
	if expPath == "" {
		expPath = "reference/EXPERIENCE.md"
	}
	fmt.Printf("%s?%s Experience file path %s[%s]%s: ", initGreen, initReset, initCyan, expPath, initReset)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		expPath = input
	}
	config.Set("experience_path", expPath)
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
	if cfg.Project.ID == "" {
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
				saveProjectConfig(proj, fields)
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
						saveProjectConfig(found, fields)
						fmt.Printf("  Linked to \"%s\"\n", found.Title)
					}
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s✓%s GitHub Project #%d linked\n\n", initGreen, initReset, cfg.Project.Number)
	}

	// Initialize .cvx directory structure
	if err := workflow.Init(schemaPath); err != nil {
		fmt.Printf("  Warning: Could not initialize .cvx/ directory: %v\n", err)
	}

	fmt.Printf("%s%sReady!%s\n", initBold, initGreen, initReset)
	fmt.Printf("  %scvx add <job-url>%s    Add a job posting\n", initCyan, initReset)
	fmt.Printf("  %scvx match <issue>%s    Analyze job match\n\n", initCyan, initReset)
	return nil
}

type modelOption struct {
	name string
	note string
}

func saveProjectConfig(proj *project.ProjectInfo, fields map[string]project.FieldInfo) {
	projCfg := config.ProjectConfig{
		ID:       proj.ID,
		Number:   proj.Number,
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
		projCfg.Fields.Status = statusField.ID
		for _, opt := range statusField.Options {
			key := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
			projCfg.Statuses[key] = opt.ID
		}
	}
	if f, ok := fields["company"]; ok {
		projCfg.Fields.Company = f.ID
	}
	if f, ok := fields["deadline"]; ok {
		projCfg.Fields.Deadline = f.ID
	}
	if f, ok := fields["applied_date"]; ok {
		projCfg.Fields.AppliedDate = f.ID
	}

	config.SaveProject(projCfg)
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

func buildModelList() []modelOption {
	var models []modelOption

	if ai.IsClaudeCLIAvailable() {
		models = append(models,
			modelOption{"claude-cli", "uses CLI-configured model"},
			modelOption{"claude-cli:opus-4.5", ""},
			modelOption{"claude-cli:sonnet-4", ""},
		)
	}

	if ai.IsGeminiCLIAvailable() {
		models = append(models, modelOption{"gemini-cli", "uses CLI-configured model"})
	}

	for _, m := range ai.SupportedModels() {
		if m == "claude-cli" || m == "gemini-cli" {
			continue
		}
		note := ""
		if strings.HasPrefix(m, "gemini") {
			note = "requires GEMINI_API_KEY"
		} else if strings.HasPrefix(m, "claude") {
			note = "requires ANTHROPIC_API_KEY"
		}
		models = append(models, modelOption{m, note})
	}

	return models
}
