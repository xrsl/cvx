package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	clog "github.com/xrsl/cvx/pkg/log"
	"github.com/xrsl/cvx/pkg/signal"
	"github.com/xrsl/cvx/pkg/style"
)

// agentFS will be set by the main package
var agentFS *embed.FS

// SetAgentFS sets the embedded agent filesystem
func SetAgentFS(fs *embed.FS) {
	agentFS = fs
}

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
	SilenceErrors: true, // We handle errors ourselves
}

func Execute() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Create context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext()

	err := rootCmd.ExecuteContext(ctx)
	cancel() // Clean up signal handlers before exit

	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "interrupted")
			os.Exit(130) // Standard exit code for SIGINT
		}
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
