package cmd

import (
	"cvx/pkg/config"
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cvx configuration",
	Long:  `Get and set cvx configuration values.`,
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
		all, err := config.All()
		if err != nil {
			return err
		}
		fmt.Printf("Config: %s\n\n", config.Path())
		for k, v := range all {
			if v == "" {
				v = "(not set)"
			}
			fmt.Printf("  %s: %s\n", k, v)
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}
