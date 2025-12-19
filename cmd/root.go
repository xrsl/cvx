package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var quiet bool

var rootCmd = &cobra.Command{
	Use:   "cvx",
	Short: "A CLI for CV workflows powered by AI",
	Long:  `cvx automates CV-related workflows using AI agents like Gemini and Claude.`,
}

func Execute() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
}
