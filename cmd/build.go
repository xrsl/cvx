package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/xrsl/cvx/pkg/ai"
	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/style"
	"github.com/xrsl/cvx/pkg/workflow"
)

// buildModelList generates a formatted list of supported models for help text
func buildModelList() string {
	// Collect model names and sort them for consistent output
	type modelEntry struct {
		short string
		long  string
	}
	entries := make([]modelEntry, 0, len(ai.SupportedModelMap))
	for short, model := range ai.SupportedModelMap {
		entries = append(entries, modelEntry{short: short, long: model.APIName})
	}

	// Custom sort order: flash, pro, gpt, qwen, haiku, sonnet, opus
	familyOrder := map[string]int{"flash": 0, "pro": 1, "gpt": 2, "qwen": 3, "haiku": 4, "sonnet": 5, "opus": 6}
	getFamily := func(name string) string {
		for family := range familyOrder {
			if strings.HasPrefix(name, family) {
				return family
			}
		}
		return name
	}

	sort.Slice(entries, func(i, j int) bool {
		famI, famJ := getFamily(entries[i].short), getFamily(entries[j].short)
		if familyOrder[famI] != familyOrder[famJ] {
			return familyOrder[famI] < familyOrder[famJ]
		}
		// Within same family, sort by name (version)
		return entries[i].short < entries[j].short
	})

	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %-12s → %s\n", e.short, e.long))
	}
	return sb.String()
}

var buildCmd = &cobra.Command{
	Use:   "build [issue-number]",
	Short: "Build tailored CV and cover letter",
	Long: `Build tailored CV and cover letter for a job posting.

Two build modes:
  1. Interactive CLI (default): Real-time editing with auto-detected CLI tools
  2. Agent Mode: Structured output with validation

Examples:
  cvx build                      # Interactive mode (default)
  cvx build 42                   # Interactive for issue #42
  cvx build -c "focus on ML"     # Interactive with context

  cvx build -m sonnet-4          # Agent mode
  cvx build -m flash-2-5         # Agent mode with Gemini

Supported models (short → full name):
` + buildModelList(),
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

var (
	// build-specific flags (modelFlag is global in root.go)
	buildContextFlag string
	buildSchemaFlag  string
	buildBranchFlag  bool
)

func init() {
	// Note: -m is a global flag defined in root.go
	buildCmd.Flags().StringVarP(&buildContextFlag, "context", "c", "", "Feedback or additional context")
	buildCmd.Flags().StringVarP(&buildSchemaFlag, "schema", "s", "", "Schema path (defaults to schema from config)")
	buildCmd.Flags().BoolVarP(&buildBranchFlag, "branch", "b", false, "Switch to issue branch (creates if not exists, format: issue_number-company_name-role)")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.LoadWithCache()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	issueNum, err := resolveIssueNumber(args)
	if err != nil {
		return err
	}

	// Handle branch switching if requested
	if buildBranchFlag {
		if err := ensureIssueBranch(cfg.GitHub.Repo, issueNum); err != nil {
			return err
		}
	}

	// Agent Mode (use -m flag)
	if modelFlag != "" {
		// Resolve short model name to full API name
		modelConfig, ok := ai.GetModel(modelFlag)
		if !ok {
			return fmt.Errorf("unsupported model: %s (supported: %v)", modelFlag, ai.SupportedModelNames())
		}
		if err := os.Setenv("AI_MODEL", modelConfig.APIName); err != nil {
			return fmt.Errorf("failed to set AI_MODEL: %w", err)
		}
		return runBuildWithAgent(cfg, issueNum)
	}

	// Interactive CLI mode (default)
	if cfg.Agent.Default == "" {
		return fmt.Errorf("no CLI agent configured. Run 'cvx init' to configure")
	}

	return runBuildInteractive(cfg, issueNum)
}

func resolveIssueNumber(args []string) (string, error) {
	if len(args) > 0 {
		issueNum := args[0]
		if _, err := fmt.Sscanf(issueNum, "%d", new(int)); err != nil {
			return "", fmt.Errorf("invalid issue number: %s (must be numeric)", issueNum)
		}
		return issueNum, nil
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", err
	}
	issueNum := extractIssueFromBranch(currentBranch)
	if issueNum == "" {
		return "", fmt.Errorf("could not infer issue number from branch '%s'. Provide it explicitly: cvx build <issue-number>", currentBranch)
	}
	fmt.Printf("Using issue #%s (from branch %s)\n", issueNum, currentBranch)
	return issueNum, nil
}

func runBuildInteractive(cfg *config.Config, issueNum string) error {
	agent := cfg.AgentCLI()

	// Use issue number as unified session key
	sessionID, hasSession := getSession(issueNum + "-build")

	var execCmd *exec.Cmd

	if hasSession {
		fmt.Printf("%s Resuming session for issue %s\n", style.C(style.Cyan, "↻"), style.C(style.Cyan, "#"+issueNum))
		if buildContextFlag != "" {
			execCmd = exec.Command(agent, "--resume", sessionID, "-p", buildContextFlag)
		} else {
			execCmd = exec.Command(agent, "--resume", sessionID)
		}
	} else {
		fmt.Printf("%s Starting build session for issue %s\n", style.C(style.Green, "▶"), style.C(style.Cyan, "#"+issueNum))

		// Fetch issue body
		issueBody, err := fetchIssueBody(cfg.GitHub.Repo, issueNum)
		if err != nil {
			return fmt.Errorf("error fetching issue: %w", err)
		}

		prompt, err := buildBuildPrompt(cfg, issueBody)
		if err != nil {
			return err
		}

		if buildContextFlag != "" {
			prompt = fmt.Sprintf("%s\n\nAdditional context: %s", prompt, buildContextFlag)
		}

		// Use -i for gemini (prompt-interactive), -p for claude
		if agent == "gemini" || strings.HasPrefix(agent, "gemini:") {
			execCmd = exec.Command("gemini", "-i", prompt)
		} else {
			execCmd = exec.Command("claude", "-p", prompt)
		}
	}

	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("error running %s: %w", agent, err)
	}

	// Save session if new
	if !hasSession {
		if newSessionID := getMostRecentAgentSession(agent); newSessionID != "" {
			_ = saveSession(issueNum+"-build", newSessionID)
			fmt.Printf("%s Session saved for issue %s\n", style.C(style.Green, "✓"), style.C(style.Cyan, "#"+issueNum))
		}
	}

	return nil
}

func buildBuildPrompt(cfg *config.Config, issueBody string) (string, error) {
	workflowContent, err := workflow.LoadBuild()
	if err != nil {
		return "", fmt.Errorf("error loading workflow: %w", err)
	}

	tmpl, err := template.New("build").Parse(workflowContent)
	if err != nil {
		return "", fmt.Errorf("error parsing workflow template: %w", err)
	}

	data := struct {
		CVYAMLPath    string
		ReferencePath string
	}{
		CVYAMLPath:    cfg.CV.Source,
		ReferencePath: cfg.Paths.Reference,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing workflow template: %w", err)
	}

	return fmt.Sprintf("%s\n\n## Job Posting\n%s", buf.String(), issueBody), nil
}

// ========================================
// Agent Mode Functions
// ========================================

// readYAMLCV reads cv.yaml and extracts the cv field
func readYAMLCV(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		CV map[string]interface{} `yaml:"cv" toml:"cv"`
	}

	// Detect format based on file extension
	if strings.HasSuffix(path, ".toml") {
		if err := toml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	}
	return wrapper.CV, nil
}

// readYAMLLetter reads letter.yaml/toml and extracts the letter field
func readYAMLLetter(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Letter map[string]interface{} `yaml:"letter" toml:"letter"`
	}

	// Detect format based on file extension
	if strings.HasSuffix(path, ".toml") {
		if err := toml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	}
	return wrapper.Letter, nil
}

// removeNilValues recursively removes nil values from maps and slices
// This prevents "null" from appearing in YAML output
func removeNilValues(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if val != nil {
				cleaned := removeNilValues(val)
				if cleaned != nil {
					result[key] = cleaned
				}
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, val := range v {
			if val != nil {
				cleaned := removeNilValues(val)
				if cleaned != nil {
					result = append(result, cleaned)
				}
			}
		}
		return result
	default:
		return v
	}
}

// writeYAMLData is a generic function to write CV or Letter data with schema support
func writeYAMLData(path string, data map[string]interface{}, fieldName, schemaPath string) error {
	// Remove nil values to avoid "null" in output
	cleaned := removeNilValues(data).(map[string]interface{})

	// Create wrapper with dynamic field name
	wrapper := map[string]interface{}{
		fieldName: cleaned,
	}

	var marshaledData []byte
	var err error

	// Detect format based on file extension
	if strings.HasSuffix(path, ".toml") {
		marshaledData, err = toml.Marshal(wrapper)
	} else {
		marshaledData, err = yaml.Marshal(wrapper)
	}
	if err != nil {
		return err
	}

	// Prepend schema comment for IDE support (if schema path provided)
	finalData := marshaledData
	if schemaPath != "" {
		// Calculate relative path from file to schema
		relPath, err := filepath.Rel(filepath.Dir(path), schemaPath)
		if err != nil {
			relPath = schemaPath // fallback to absolute path
		}

		// Use different comment format for TOML vs YAML
		var schemaComment string
		if strings.HasSuffix(path, ".toml") {
			schemaComment = fmt.Sprintf("#:schema %s\n", relPath)
		} else {
			schemaComment = fmt.Sprintf("# yaml-language-server: $schema=%s\n", relPath)
		}
		finalData = append([]byte(schemaComment), marshaledData...)
	}

	if err := os.WriteFile(path, finalData, 0o644); err != nil {
		return err
	}

	// Auto-format TOML files with tombi if available
	if strings.HasSuffix(path, ".toml") {
		formatTOML(path)
	}

	return nil
}

// writeYAMLCV writes cv data back to cv.yaml or cv.toml
func writeYAMLCV(path string, cv map[string]interface{}, schemaPath string) error {
	return writeYAMLData(path, cv, "cv", schemaPath)
}

// writeYAMLLetter writes letter data back to letter.yaml or letter.toml
func writeYAMLLetter(path string, letter map[string]interface{}, schemaPath string) error {
	return writeYAMLData(path, letter, "letter", schemaPath)
}

// formatTOML formats a TOML file using tombi if available
func formatTOML(path string) {
	if _, err := exec.LookPath("tombi"); err != nil {
		return // tombi not available, skip formatting
	}

	cmd := exec.Command("tombi", "format", path)
	_ = cmd.Run() // ignore errors - formatting is best-effort
}

// callAgent calls the agent subprocess with JSON stdin/stdout
// extractAgentToCache extracts the embedded agent to a cache directory for reuse
func extractAgentToCache() (string, error) {
	if agentFS == nil {
		return "", fmt.Errorf("agent filesystem not initialized")
	}

	// Use ~/.cache/cvx/agent as persistent cache
	cacheDir := filepath.Join(os.ExpandEnv("$HOME"), ".cache", "cvx", "agent")

	// Check if already extracted (simple version check via stat)
	if stat, err := os.Stat(cacheDir); err == nil && stat.IsDir() {
		// Cache exists, reuse it
		return cacheDir, nil
	}

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Walk the embedded FS and extract all files
	err := fs.WalkDir(*agentFS, "agent", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path (remove "agent" or "agent/" prefix)
		relPath := strings.TrimPrefix(path, "agent")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			return nil // skip the agent directory itself
		}

		targetPath := filepath.Join(cacheDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		// Read file from embedded FS
		data, err := fs.ReadFile(*agentFS, path)
		if err != nil {
			return err
		}

		// Write to cache directory
		return os.WriteFile(targetPath, data, 0o644)
	})

	if err != nil {
		_ = os.RemoveAll(cacheDir) // cleanup on error
		return "", fmt.Errorf("failed to extract agent: %w", err)
	}

	return cacheDir, nil
}

// regenerateModels regenerates models.py from schema.json in the agent directory
// Only regenerates if the schema has changed since last generation
func regenerateModels(agentDir, schemaPath string) {
	if schemaPath == "" || agentDir == "" {
		return // skip if no schema provided
	}

	// Check if schema exists
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return // skip if schema doesn't exist
	}

	// Compute schema hash
	schemaHash := fmt.Sprintf("%x", sha256.Sum256(schemaData))
	hashPath := filepath.Join(agentDir, ".schema_hash")

	// Check if models.py needs regeneration
	if existingHash, err := os.ReadFile(hashPath); err == nil {
		if string(existingHash) == schemaHash {
			// Schema hasn't changed, skip regeneration
			return
		}
	}

	modelsPath := filepath.Join(agentDir, "cvx_agent", "models.py")

	// Preprocess schema: remove '#' prefixes from definitions
	// This fixes the infinite recursion issue with datamodel-codegen
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse schema: %v\n", err)
		return
	}

	// Clean the schema to work with datamodel-codegen
	cleanSchema(schema)

	// Write cleaned schema to temp file
	cleanedData, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal cleaned schema: %v\n", err)
		return
	}

	tmpSchemaPath := filepath.Join(agentDir, ".schema_cleaned.json")
	if err := os.WriteFile(tmpSchemaPath, cleanedData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write cleaned schema: %v\n", err)
		return
	}
	defer func() { _ = os.Remove(tmpSchemaPath) }() // cleanup temp file

	// Run datamodel-codegen via uv run from the agent directory
	cmd := exec.Command(
		"uv", "run",
		"datamodel-codegen",
		"--input", tmpSchemaPath,
		"--input-file-type", "jsonschema",
		"--output", modelsPath,
		"--output-model-type", "pydantic_v2.BaseModel",
		"--disable-warnings",
	)
	cmd.Dir = agentDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Just log warning, don't fail the build
		fmt.Fprintf(os.Stderr, "Warning: failed to regenerate models.py: %v\n%s\n", err, stderr.String())
		return
	}

	// Save the schema hash
	_ = os.WriteFile(hashPath, []byte(schemaHash), 0o644)
}

// cleanSchema removes '#' prefixes from definition names to fix datamodel-codegen issues
func cleanSchema(schema map[string]interface{}) {
	// Process $defs: rename keys to remove '#' prefix
	if defs, ok := schema["$defs"].(map[string]interface{}); ok {
		newDefs := make(map[string]interface{})
		for key, value := range defs {
			cleanKey := strings.TrimPrefix(key, "#")
			newDefs[cleanKey] = value
		}
		schema["$defs"] = newDefs
	}

	// Recursively fix all $ref values
	fixRefs(schema)
}

// fixRefs recursively replaces $ref values to remove '#' prefixes
func fixRefs(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		// Check if this object has a $ref
		if ref, ok := val["$ref"].(string); ok {
			// Replace #/$defs/#Name with #/$defs/Name
			val["$ref"] = strings.ReplaceAll(ref, "#/$defs/#", "#/$defs/")
		}
		// Recurse into all values
		for _, child := range val {
			fixRefs(child)
		}
	case []interface{}:
		// Recurse into array elements
		for _, child := range val {
			fixRefs(child)
		}
	}
}

// Go boundary: subprocess management, JSON marshaling
// Agent boundary: AI calls, validation, caching
func callAgent(jobPosting string, cv, letter map[string]interface{}, schemaPath string) (cvOut, letterOut map[string]interface{}, err error) {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		return nil, nil, fmt.Errorf("uv is not installed. Please install uv: https://docs.astral.sh/uv/")
	}

	// Extract embedded agent to cache directory
	agentDir, err := extractAgentToCache()
	if err != nil {
		return nil, nil, err
	}

	// Regenerate models.py from project schema
	regenerateModels(agentDir, schemaPath)

	// Prepare input JSON
	input := map[string]interface{}{
		"job_posting": jobPosting,
		"cv":          cv,
		"letter":      letter,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Call agent via uvx
	cmd := exec.Command("uvx", "--from", agentDir, "cvx-agent")
	cmd.Stdin = bytes.NewReader(inputJSON)

	// Pass environment variables to subprocess (including API keys)
	cmd.Env = os.Environ()

	// Groq models use OpenAI-compatible API, so set OPENAI_API_KEY from GROQ_API_KEY if needed
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		if os.Getenv("OPENAI_API_KEY") == "" {
			cmd.Env = append(cmd.Env, "OPENAI_API_KEY="+groqKey)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("agent failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse output JSON
	var output struct {
		CV     map[string]interface{} `json:"cv"`
		Letter map[string]interface{} `json:"letter"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, nil, fmt.Errorf("failed to parse output: %w\noutput: %s", err, stdout.String())
	}

	// Validate output has required fields
	if output.CV == nil || output.Letter == nil {
		// Debug: show what we actually got
		fmt.Fprintf(os.Stderr, "Debug: agent output:\n%s\n", stdout.String())
		return nil, nil, fmt.Errorf("invalid output: missing cv or letter fields (cv=%v, letter=%v)", output.CV == nil, output.Letter == nil)
	}

	return output.CV, output.Letter, nil
}

// callAgentWithSpinner wraps callAgent with a spinner
func callAgentWithSpinner(modelName string, jobPosting string, cv, letter map[string]interface{}, schemaPath string) (cvOut, letterOut map[string]interface{}, err error) {
	// Start spinner
	done := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
				msg := fmt.Sprintf("Building with 🤖 %s...", modelName)
				fmt.Fprintf(os.Stderr, "\r%s %s", style.C(style.Cyan, spinnerFrames[i%len(spinnerFrames)]), msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	cvOut, letterOut, err = callAgent(jobPosting, cv, letter, schemaPath)

	done <- true
	close(done)

	return cvOut, letterOut, err
}

// runBuildWithAgent executes the build command using the agent
// This mode is triggered when -m flag is used without --call-api-directly
func runBuildWithAgent(cfg *config.Config, issueNum string) error {
	// Get full model name from AI_MODEL env var (already set to full API name)
	modelFullName := os.Getenv("AI_MODEL")

	fmt.Printf("%s Building with 🤖 %s for issue %s\n",
		style.C(style.Green, "▶"), style.C(style.Cyan, modelFullName), style.C(style.Cyan, "#"+issueNum))

	// 1. Fetch job posting from GitHub
	issueBody, err := fetchIssueBody(cfg.GitHub.Repo, issueNum)
	if err != nil {
		return fmt.Errorf("error fetching issue: %w", err)
	}

	// 2. Read YAML files
	cvPath := cfg.CV.Source
	if cvPath == "" {
		cvPath = "src/cv.yaml"
	}
	letterPath := cfg.Letter.Source
	if letterPath == "" {
		letterPath = "src/letter.yaml"
	}

	cv, err := readYAMLCV(cvPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", cvPath, err)
	}

	letter, err := readYAMLLetter(letterPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", letterPath, err)
	}

	// 3. Read schema if available
	schemaPath := buildSchemaFlag
	if schemaPath == "" {
		schemaPath = cfg.CV.Schema
	}
	if schemaPath == "" {
		schemaPath = "schema/schema.json" // fallback default
	}

	// 4. Call agent with spinner
	cvOut, letterOut, err := callAgentWithSpinner(modelFullName, issueBody, cv, letter, schemaPath)
	if err != nil {
		return err
	}

	// 5. Write output files
	if err := writeYAMLCV(cvPath, cvOut, schemaPath); err != nil {
		return fmt.Errorf("failed to write %s: %w", cvPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), cvPath)

	if err := writeYAMLLetter(letterPath, letterOut, schemaPath); err != nil {
		return fmt.Errorf("failed to write %s: %w", letterPath, err)
	}
	fmt.Printf("%s Wrote %s\n", style.C(style.Green, "✓"), letterPath)

	return nil
}
