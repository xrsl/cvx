package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// callAgentGeneric calls the agent with an action and input data.
// This is the unified interface for all AI operations.
func callAgentGeneric(action string, input map[string]interface{}, schemaPath string) (map[string]interface{}, error) {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		return nil, fmt.Errorf("uv is not installed. Please install uv: https://docs.astral.sh/uv/")
	}

	// Extract embedded agent to cache directory
	agentDir, err := extractAgentToCache()
	if err != nil {
		return nil, err
	}

	// Regenerate models.py from project schema (for build action)
	if schemaPath != "" {
		regenerateModels(agentDir, schemaPath)
	}

	// Add action to input
	input["action"] = action

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
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
		return nil, fmt.Errorf("agent failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse output JSON
	var output map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w\noutput: %s", err, stdout.String())
	}

	return output, nil
}

// callAgentExtract calls the agent to extract job posting fields.
func callAgentExtract(jobText, url, schemaPrompt string) (map[string]interface{}, error) {
	input := map[string]interface{}{
		"job_text":      jobText,
		"url":           url,
		"schema_prompt": schemaPrompt,
	}
	return callAgentGeneric("extract", input, "")
}

// callAgentAdvise calls the agent to analyze job-CV match.
func callAgentAdvise(jobPosting, cvContent, workflowPrompt, context string) (string, error) {
	input := map[string]interface{}{
		"job_posting":     jobPosting,
		"cv_content":      cvContent,
		"workflow_prompt": workflowPrompt,
		"context":         context,
	}

	output, err := callAgentGeneric("advise", input, "")
	if err != nil {
		return "", err
	}

	if analysis, ok := output["analysis"].(string); ok {
		return analysis, nil
	}

	return "", fmt.Errorf("unexpected output format: %v", output)
}
