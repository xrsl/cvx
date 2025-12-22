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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	cfgReset = "\033[0m"
	cfgGreen = "\033[0;32m"
	cfgCyan  = "\033[0;36m"
	cfgBold  = "\033[1m"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cvx configuration",
	Long: `Interactive setup wizard or direct config access.

Run without subcommand for interactive setup:
  cvx config

Or use subcommands:
  cvx config list
  cvx config get <key>
  cvx config set <key> <value>`,
	RunE: runConfigWizard,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a configuration value.

Keys:
  repo    GitHub repo (owner/name)
  model   AI model (gemini-3.0-flash, claude-sonnet-4, etc.)
  schema  Path to GitHub issue template YAML

Examples:
  cvx config set repo myuser/cv
  cvx config set model gemini-2.5-pro
  cvx config set schema /path/to/.github/ISSUE_TEMPLATE/job-app.yml`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		if err := config.Set(key, value); err != nil {
			return err
		}
		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		value, err := config.Get(args[0])
		if err != nil {
			return err
		}
		if value == "" {
			fmt.Println("(not set)")
		} else {
			fmt.Println(value)
		}
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Printf("\n%s%scvx config%s\n", cfgBold, cfgCyan, cfgReset)
		fmt.Printf("%s%s%s\n\n", cfgGray, config.Path(), cfgReset)

		// AI settings
		fmt.Printf("%sai%s\n", cfgCyan, cfgReset)
		printConfigRow("agent", cfg.Model, "bundled default")

		// Schema
		schemaDisplay := cfg.Schema
		if schemaDisplay == "" {
			schemaDisplay = ""
		}
		printConfigRow("schema", schemaDisplay, "bundled default")

		// GitHub settings
		fmt.Printf("\n%sgh%s\n", cfgCyan, cfgReset)
		if cfg.Repo != "" {
			repoURL := fmt.Sprintf("https://github.com/%s", cfg.Repo)
			fmt.Printf("  %-9s %s%s%s\n", "repo", cfgGreen, repoURL, cfgReset)
		} else {
			fmt.Printf("  %-9s %s(not set)%s\n", "repo", cfgGray, cfgReset)
		}

		if cfg.Project.ID != "" {
			// Extract owner from repo for project URL
			parts := strings.Split(cfg.Repo, "/")
			owner := parts[0]
			projectURL := fmt.Sprintf("https://github.com/users/%s/projects/%d", owner, cfg.Project.Number)
			title := cfg.Project.Title
			if title == "" {
				title = fmt.Sprintf("Project #%d", cfg.Project.Number)
			}
			fmt.Printf("  %-9s %s%s%s %s(%s)%s\n", "project", cfgGreen, projectURL, cfgReset, cfgGray, title, cfgReset)
			if len(cfg.Project.Statuses) > 0 {
				var statuses []string
				for k := range cfg.Project.Statuses {
					statuses = append(statuses, k)
				}
				fmt.Printf("  %-9s %s%s%s\n", "statuses", cfgGray, strings.Join(statuses, ", "), cfgReset)
			}
		} else {
			fmt.Printf("  %-9s %s(not configured)%s\n", "project", cfgGray, cfgReset)
		}

		fmt.Println()
		return nil
	},
}

func printConfigRow(key, value, defaultHint string) {
	if value == "" {
		if defaultHint != "" {
			fmt.Printf("  %-9s %s(%s)%s\n", key, cfgGray, defaultHint, cfgReset)
		} else {
			fmt.Printf("  %-9s %s(not set)%s\n", key, cfgGray, cfgReset)
		}
	} else {
		fmt.Printf("  %-9s %s%s%s\n", key, cfgGreen, value, cfgReset)
	}
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

const cfgGray = "\033[90m"

type modelOption struct {
	name string
	note string
}

// buildModelList returns a flat list of all available models with notes
func buildModelList() []modelOption {
	var models []modelOption

	// Claude CLI options (if available)
	if ai.IsClaudeCLIAvailable() {
		models = append(models,
			modelOption{"claude-cli", "uses CLI-configured model"},
			modelOption{"claude-cli:opus-4.5", ""},
			modelOption{"claude-cli:sonnet-4", ""},
		)
	}

	// Gemini CLI (if available)
	if ai.IsGeminiCLIAvailable() {
		models = append(models, modelOption{"gemini-cli", "uses CLI-configured model"})
	}

	// API models
	for _, m := range ai.SupportedModels() {
		// Skip CLI models already added
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

func runConfigWizard(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	cfg, _ := config.Load()

	fmt.Printf("\n%s%scvx setup%s\n\n", cfgBold, cfgCyan, cfgReset)

	// Step 1: Repository
	repo := cfg.Repo
	for {
		fmt.Printf("%s?%s Repository ", cfgGreen, cfgReset)
		if repo != "" {
			fmt.Printf("%s(%s)%s: ", cfgCyan, repo, cfgReset)
		} else {
			fmt.Printf("%s(owner/repo)%s: ", cfgCyan, cfgReset)
		}

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" && repo != "" {
			break // keep existing
		} else if input != "" {
			repo = input
		} else {
			fmt.Println("  Repository is required")
			continue
		}

		// Validate
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
		fmt.Printf("%s✓%s\n\n", cfgGreen, cfgReset)
		break
	}
	config.Set("repo", repo)

	// Step 2: AI Model
	models := buildModelList()
	defaultModel := models[0].name // first is default
	currentModel := cfg.Model
	if currentModel == "" {
		currentModel = defaultModel
	}

	// Find current model index
	currentIdx := 0
	for i, m := range models {
		if m.name == currentModel {
			currentIdx = i
			break
		}
	}

	fmt.Printf("%s?%s AI Model\n", cfgGreen, cfgReset)
	for i, m := range models {
		marker := "   "
		if i == currentIdx {
			marker = fmt.Sprintf("  %s→%s", cfgGreen, cfgReset)
		}
		fmt.Printf("%s%s%d)%s %s", marker, cfgCyan, i+1, cfgReset, m.name)
		if m.note != "" {
			fmt.Printf(" %s(%s)%s", cfgGray, m.note, cfgReset)
		}
		fmt.Println()
	}
	fmt.Printf("\n  Choice %s(%d)%s: ", cfgCyan, currentIdx+1, cfgReset)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selectedModel := currentModel
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(models) {
			selectedModel = models[idx-1].name
		}
	}

	config.Set("model", selectedModel)
	fmt.Printf("  Using %s%s%s\n\n", cfgCyan, selectedModel, cfgReset)

	// Step 3: CV Path (for match command)
	cvPath := cfg.CVPath
	fmt.Printf("%s?%s CV file path ", cfgGreen, cfgReset)
	if cvPath != "" {
		fmt.Printf("%s(%s)%s: ", cfgCyan, cvPath, cfgReset)
	} else {
		fmt.Printf("%s(e.g. src/cv.tex)%s: ", cfgCyan, cfgReset)
	}
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cvPath = input
	}
	if cvPath != "" {
		config.Set("cv_path", cvPath)
	}

	// Step 4: Experience/Skills file path
	expPath := cfg.ExperiencePath
	fmt.Printf("%s?%s Experience file path ", cfgGreen, cfgReset)
	if expPath != "" {
		fmt.Printf("%s(%s)%s: ", cfgCyan, expPath, cfgReset)
	} else {
		fmt.Printf("%s(e.g. reference/EXPERIENCE.md)%s: ", cfgCyan, cfgReset)
	}
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		expPath = input
	}
	if expPath != "" {
		config.Set("experience_path", expPath)
	}
	fmt.Println()

	// Step 5: GitHub Project
	if cfg.Project.ID == "" {
		fmt.Printf("%s?%s Create GitHub Project for tracking? %s(Y/n)%s: ", cfgGreen, cfgReset, cfgCyan, cfgReset)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "n" && input != "no" {
			fmt.Printf("\nCreating project... ")
			client := project.New(repo)

			proj, fields, err := client.Create("Job Applications", nil)
			if err != nil {
				fmt.Printf("\n  %sFailed:%s %v\n", cfgCyan, cfgReset, err)
			} else {
				fmt.Printf("%s✓%s\n", cfgGreen, cfgReset)

				projCfg := config.ProjectConfig{
					ID:       proj.ID,
					Number:   proj.Number,
					Title:    proj.Title,
					Statuses: make(map[string]string),
					Fields: config.FieldIDs{
						Status:      fields["status"].ID,
						Company:     fields["company"].ID,
						Deadline:    fields["deadline"].ID,
						AppliedDate: fields["applied_date"].ID,
					},
				}

				for _, opt := range fields["status"].Options {
					key := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
					projCfg.Statuses[key] = opt.ID
				}

				config.SaveProject(projCfg)
				fmt.Printf("  Project #%d created\n", proj.Number)
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s✓%s GitHub Project #%d linked\n\n", cfgGreen, cfgReset, cfg.Project.Number)
	}

	// Initialize workflow directory structure
	if err := workflow.Init(); err != nil {
		fmt.Printf("  Warning: Could not initialize .cvx/ directory: %v\n", err)
	}

	// Done
	fmt.Printf("%s%sReady!%s Try: %scvx add <job-url>%s\n\n", cfgBold, cfgGreen, cfgReset, cfgCyan, cfgReset)
	return nil
}
