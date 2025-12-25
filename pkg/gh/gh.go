// Package gh provides an interface for GitHub CLI operations
package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// CLI defines the interface for GitHub CLI operations
type CLI interface {
	// IssueCreate creates a new issue and returns the issue URL
	IssueCreate(repo, title, body string) (string, error)
	// IssueView returns issue details as JSON
	IssueView(repo string, number int, fields []string) ([]byte, error)
	// IssueViewByStr returns issue details by string identifier (number or URL)
	IssueViewByStr(repo, issue string, fields []string) ([]byte, error)
	// IssueList lists issues with optional filters
	IssueList(repo, state string, limit int) ([]byte, error)
	// IssueDelete deletes an issue
	IssueDelete(repo string, number int) error
	// IssueDeleteByStr deletes an issue by string identifier
	IssueDeleteByStr(repo, issue string) error
	// IssueComment adds a comment to an issue
	IssueComment(repo, issue, body string) error
	// RepoView returns repo details as JSON
	RepoView(repo string, fields []string) ([]byte, error)
	// APIUser returns the current user's login
	APIUser() (string, error)
	// GraphQL executes a GraphQL query and returns the response
	GraphQL(query string) ([]byte, error)
	// GraphQLWithJQ executes a GraphQL query with jq filter
	GraphQLWithJQ(query, jq string) ([]byte, error)
}

// DefaultCLI implements CLI using the gh command
type DefaultCLI struct{}

// New returns a new DefaultCLI instance
func New() *DefaultCLI {
	return &DefaultCLI{}
}

// IssueCreate creates a new issue
func (c *DefaultCLI) IssueCreate(repo, title, body string) (string, error) {
	cmd := exec.Command("gh", "issue", "create", "-R", repo, "--title", title, "--body", body)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh issue create failed: %w", err)
	}
	return string(output), nil
}

// IssueView returns issue details
func (c *DefaultCLI) IssueView(repo string, number int, fields []string) ([]byte, error) {
	args := []string{"issue", "view", fmt.Sprintf("%d", number), "--repo", repo, "--json"}
	if len(fields) > 0 {
		fieldStr := ""
		for i, f := range fields {
			if i > 0 {
				fieldStr += ","
			}
			fieldStr += f
		}
		args = append(args, fieldStr)
	}
	cmd := exec.Command("gh", args...)
	return cmd.Output()
}

// IssueList lists issues
func (c *DefaultCLI) IssueList(repo, state string, limit int) ([]byte, error) {
	args := []string{"issue", "list", "--repo", repo, "--json", "number,title,state,labels"}
	if state != "" {
		args = append(args, "--state", state)
	}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", limit))
	}
	cmd := exec.Command("gh", args...)
	return cmd.Output()
}

// IssueDelete deletes an issue
func (c *DefaultCLI) IssueDelete(repo string, number int) error {
	return c.IssueDeleteByStr(repo, fmt.Sprintf("%d", number))
}

// IssueDeleteByStr deletes an issue by string identifier
func (c *DefaultCLI) IssueDeleteByStr(repo, issue string) error {
	cmd := exec.Command("gh", "issue", "delete", issue, "--repo", repo, "--yes")
	_, err := cmd.Output()
	return err
}

// IssueViewByStr returns issue details by string identifier
func (c *DefaultCLI) IssueViewByStr(repo, issue string, fields []string) ([]byte, error) {
	args := []string{"issue", "view", issue, "--repo", repo, "--json"}
	if len(fields) > 0 {
		fieldStr := ""
		for i, f := range fields {
			if i > 0 {
				fieldStr += ","
			}
			fieldStr += f
		}
		args = append(args, fieldStr)
	}
	cmd := exec.Command("gh", args...)
	return cmd.Output()
}

// IssueComment adds a comment to an issue
func (c *DefaultCLI) IssueComment(repo, issue, body string) error {
	cmd := exec.Command("gh", "issue", "comment", issue, "--repo", repo, "--body", body)
	_, err := cmd.CombinedOutput()
	return err
}

// RepoView returns repo details
func (c *DefaultCLI) RepoView(repo string, fields []string) ([]byte, error) {
	args := []string{"repo", "view", repo, "--json"}
	if len(fields) > 0 {
		fieldStr := ""
		for i, f := range fields {
			if i > 0 {
				fieldStr += ","
			}
			fieldStr += f
		}
		args = append(args, fieldStr)
	}
	cmd := exec.Command("gh", args...)
	return cmd.Output()
}

// APIUser returns the current user's login
func (c *DefaultCLI) APIUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GraphQL executes a GraphQL query
func (c *DefaultCLI) GraphQL(query string) ([]byte, error) {
	cmd := exec.Command("gh", "api", "graphql", "-f", fmt.Sprintf("query=%s", query))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, string(out))
	}
	return out, nil
}

// GraphQLWithJQ executes a GraphQL query with jq filter
func (c *DefaultCLI) GraphQLWithJQ(query, jq string) ([]byte, error) {
	cmd := exec.Command("gh", "api", "graphql", "-f", "query="+query, "--jq", jq)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, string(out))
	}
	return out, nil
}

// Issue represents a GitHub issue
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// ParseIssue parses issue JSON
func ParseIssue(data []byte) (*Issue, error) {
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}
