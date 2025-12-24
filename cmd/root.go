package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/style"
)

var quiet bool

var rootCmd = &cobra.Command{
	Use:   "cvx",
	Short: "A CLI for CV workflows powered by AI",
	Long: `cvx automates CV-related workflows using AI agents like Claude and Gemini.

Manage job applications from the command line - add postings, analyze matches,
tailor your CV and cover letters, and track everything in GitHub Issues.`,
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
	// Setup Typer-style help formatting
	style.SetupHelp(rootCmd)

	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
}
