package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	clog "github.com/xrsl/cvx/pkg/log"
	"github.com/xrsl/cvx/pkg/style"
)

var (
	quiet   bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "cvx",
	Short: "A CLI for CV workflows powered by AI",
	Long: `cvx automates CV-related workflows using AI agents like Claude and Gemini.

Manage job applications from the command line - add postings, analyze matches,
tailor your CV and cover letters, and track everything in GitHub Issues.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure logging based on flags
		clog.SetVerbose(verbose)
		clog.SetQuiet(quiet)
	},
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
}
