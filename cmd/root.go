package cmd

import (
	"bufio"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	quiet     bool
	verbose   bool
	envFile   string
	modelFlag string // Global: Use API with specified model
	agentFlag string // Global: Use specified CLI agent
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
	// Load .env files with priority:
	// 1. Explicit --env-file flag (if provided)
	// 2. Current directory .env
	// 3. Git worktree main repo .env
	// 4. Parent directories .env
	// 5. User config ~/.config/cvx/env
	loadEnvFiles()

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

// loadEnvFiles loads environment variables from .env files with fallback locations.
// Priority: explicit flag > current dir > git worktree main repo > parent dirs > user config
func loadEnvFiles() {
	// 1. Load user-level config first (lowest priority, can be overridden)
	userConfigDir := filepath.Join(os.Getenv("HOME"), ".config", "cvx")
	_ = godotenv.Load(filepath.Join(userConfigDir, "env"))

	// 2. Search parent directories for .env
	if envPath := findEnvInAncestors(); envPath != "" {
		_ = godotenv.Overload(envPath)
	}

	// 3. If in a git worktree, load .env from main repo (for sibling worktrees)
	if envPath := findEnvFromGitWorktree(); envPath != "" {
		_ = godotenv.Overload(envPath)
	}

	// 4. Current directory .env (overrides parent/worktree)
	_ = godotenv.Overload(".env")

	// 5. Explicit --env-file flag (highest priority)
	// Note: This is parsed before cobra runs, so we check os.Args directly
	if path := getEnvFileFromArgs(); path != "" {
		if err := godotenv.Overload(path); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load env file %s: %v\n", path, err)
		}
	}
}

// findEnvInAncestors searches parent directories for .env file
func findEnvInAncestors() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Skip current directory (handled separately)
	dir = filepath.Dir(dir)

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return ""
}

// findEnvFromGitWorktree detects if we're in a git worktree and returns
// the path to .env in the main repository. Git worktrees have a .git file
// (not directory) that points to the main repo's .git/worktrees/<name>.
func findEnvFromGitWorktree() string {
	gitPath := ".git"
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}

	// Regular repos have .git as a directory; worktrees have it as a file
	if info.IsDir() {
		return ""
	}

	// Read the .git file to find the main repo
	file, err := os.Open(gitPath)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return ""
	}

	line := scanner.Text()
	// Format: "gitdir: /path/to/main-repo/.git/worktrees/<worktree-name>"
	if !strings.HasPrefix(line, "gitdir:") {
		return ""
	}

	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))

	// Navigate from .git/worktrees/<name> to the main repo root
	// Go up 3 levels: worktree-name -> worktrees -> .git -> repo-root
	mainRepoDir := filepath.Dir(filepath.Dir(filepath.Dir(gitDir)))

	envPath := filepath.Join(mainRepoDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return envPath
	}

	return ""
}

// getEnvFileFromArgs parses --env-file or -e from os.Args before cobra runs
func getEnvFileFromArgs() string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--env-file" || arg == "-e" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if len(arg) > 11 && arg[:11] == "--env-file=" {
			return arg[11:]
		}
	}
	return ""
}

func init() {
	// Setup Typer-style help formatting
	style.SetupHelp(rootCmd)

	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&envFile, "env-file", "e", "", "Path to .env file (overrides default locations)")
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "", "Use API with specified model (sonnet-4, flash-2-5, etc.)")
	rootCmd.PersistentFlags().StringVarP(&agentFlag, "agent", "a", "", "Use CLI agent (claude, gemini)")
}
