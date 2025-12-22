package cmd

import (
	"cvx/pkg/config"
	"cvx/pkg/project"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	statusReset = "\033[0m"
	statusGreen = "\033[0;32m"
	statusCyan  = "\033[0;36m"
)

var statusList bool

var statusCmd = &cobra.Command{
	Use:   "status <issue-number> <status>",
	Short: "Update job application status",
	Long: `Update the status of an issue in the GitHub project.

Available statuses depend on your project configuration.
Run 'cvx status --list' to see available statuses.

Examples:
  cvx status 123 applied
  cvx status 42 todo
  cvx status --list`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusList, "list", false, "List available statuses")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	if cfg.Project.ID == "" {
		return fmt.Errorf("no project configured. Run: cvx init")
	}

	if statusList {
		fmt.Println("Available statuses:")
		for name := range cfg.Project.Statuses {
			fmt.Printf("  %s%s%s\n", statusCyan, name, statusReset)
		}
		return nil
	}

	if len(args) != 2 {
		return fmt.Errorf("usage: cvx status <issue-number> <status>")
	}

	issueNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	statusName := strings.ToLower(strings.ReplaceAll(args[1], " ", "_"))

	optionID, ok := cfg.Project.Statuses[statusName]
	if !ok {
		return fmt.Errorf("unknown status: %s (run 'cvx status --list')", statusName)
	}

	client := project.New(cfg.Repo)

	// Get project item ID
	itemID, err := client.GetItemID(cfg.Project.ID, issueNum)
	if err != nil {
		return fmt.Errorf("issue #%d not in project: %w", issueNum, err)
	}

	// Update status
	if err := client.SetStatusField(cfg.Project.ID, itemID, cfg.Project.Fields.Status, optionID); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// If status is "applied", also set AppliedDate
	if statusName == "applied" && cfg.Project.Fields.AppliedDate != "" {
		today := time.Now().Format("2006-01-02")
		if err := client.SetDateField(cfg.Project.ID, itemID, cfg.Project.Fields.AppliedDate, today); err != nil {
			log("Warning: Could not set applied date: %v", err)
		}
	}

	fmt.Printf("%sStatus updated:%s #%d -> %s%s%s\n", statusGreen, statusReset, issueNum, statusCyan, statusName, statusReset)
	return nil
}
