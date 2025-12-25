package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/xrsl/cvx/pkg/config"
	"github.com/xrsl/cvx/pkg/gh"
	"github.com/xrsl/cvx/pkg/style"
)

var (
	listState    string
	listLimit    int
	listRepoFlag string
	listCompany  string
	listApps     bool
)

type Issue struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	State        string `json:"state"`
	ProjectItems struct {
		Nodes []struct {
			FieldValues struct {
				Nodes []struct {
					Field struct {
						Name string `json:"name"`
					} `json:"field"`
					Date string `json:"date"`
				} `json:"nodes"`
			} `json:"fieldValues"`
		} `json:"nodes"`
	} `json:"projectItems"`
}

type IssueWithDeadline struct {
	Number   int
	Title    string
	Company  string
	Deadline string
	Days     int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List job applications",
	Long: `List job application issues from configured GitHub repository.

Examples:
  cvx list
  cvx list --state closed
  cvx list --company google
  cvx list -r owner/repo`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listState, "state", "open", "Issue state (open|closed|all)")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Max issues to list")
	listCmd.Flags().StringVarP(&listRepoFlag, "repo", "r", "", "GitHub repo (overrides config)")
	listCmd.Flags().StringVar(&listCompany, "company", "", "Filter by company name")
	listCmd.Flags().BoolVar(&listApps, "apps", false, "List submitted applications (from git tags)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Handle --apps flag
	if listApps {
		return runListApps(cfg)
	}

	// Resolve repo (flag > config)
	repo := listRepoFlag
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" {
		return fmt.Errorf("no repo configured. Run: cvx init")
	}

	// Parse owner/name
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s (expected owner/name)", repo)
	}
	owner, name := parts[0], parts[1]

	// Build state filter for GraphQL
	stateFilter := "OPEN"
	if listState == "closed" {
		stateFilter = "CLOSED"
	}

	// For "all", we need to fetch both - simplify by just getting issues without state filter
	var query string
	if listState == "all" {
		query = fmt.Sprintf(`query {
  repository(owner: "%s", name: "%s") {
    issues(first: %d, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        number
        title
        body
        state
        projectItems(first: 1) {
          nodes {
            fieldValues(first: 20) {
              nodes {
                ... on ProjectV2ItemFieldDateValue {
                  field {
                    ... on ProjectV2Field {
                      name
                    }
                  }
                  date
                }
              }
            }
          }
        }
      }
    }
  }
}`, owner, name, listLimit)
	} else {
		query = fmt.Sprintf(`query {
  repository(owner: "%s", name: "%s") {
    issues(first: %d, orderBy: {field: CREATED_AT, direction: DESC}, states: %s) {
      nodes {
        number
        title
        body
        state
        projectItems(first: 1) {
          nodes {
            fieldValues(first: 20) {
              nodes {
                ... on ProjectV2ItemFieldDateValue {
                  field {
                    ... on ProjectV2Field {
                      name
                    }
                  }
                  date
                }
              }
            }
          }
        }
      }
    }
  }
}`, owner, name, listLimit, stateFilter)
	}

	cli := gh.New()
	output, err := cli.GraphQL(query)
	if err != nil {
		return fmt.Errorf("gh api failed: %w", err)
	}

	var result struct {
		Data struct {
			Repository struct {
				Issues struct {
					Nodes []Issue `json:"nodes"`
				} `json:"issues"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	issues := make([]IssueWithDeadline, 0)
	today := time.Now()

	for _, issue := range result.Data.Repository.Issues.Nodes {
		deadline := ""
		days := 0

		if len(issue.ProjectItems.Nodes) > 0 {
			for _, field := range issue.ProjectItems.Nodes[0].FieldValues.Nodes {
				if field.Field.Name == "Deadline" && field.Date != "" {
					deadline = field.Date
					deadlineTime, err := time.Parse("2006-01-02", deadline)
					if err == nil {
						days = int(deadlineTime.Sub(today).Hours() / 24)
					}
					break
				}
			}
		}

		if deadline == "" {
			deadline = "No deadline"
			days = 999999 // Sort these last
		}

		// Extract company from body
		company := extractCompany(issue.Body)

		// Apply company filter if specified
		if listCompany != "" && !strings.Contains(strings.ToLower(company), strings.ToLower(listCompany)) {
			continue
		}

		issues = append(issues, IssueWithDeadline{
			Number:   issue.Number,
			Title:    issue.Title,
			Company:  company,
			Deadline: deadline,
			Days:     days,
		})
	}

	// Sort by days remaining
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Days < issues[j].Days
	})

	// Print table header
	fmt.Printf("%s%sIssue | %-35s | %-25s | %-12s | Days%s\n", style.Bold, style.Cyan, "Role", "Company", "Deadline", style.Reset)
	fmt.Printf("%s", style.Cyan)
	fmt.Printf("------+-------------------------------------+---------------------------+--------------+-----\n")
	fmt.Printf("%s", style.Reset)

	// Print table rows
	for _, issue := range issues {
		title := issue.Title
		issueURL := fmt.Sprintf("https://github.com/%s/issues/%d", repo, issue.Number)

		// Truncate title if needed
		if utf8.RuneCountInString(title) > 35 {
			runes := []rune(title)
			title = string(runes[:35])
		}

		company := issue.Company
		if len(company) > 25 {
			company = company[:25]
		}

		var daysColor string
		daysStr := fmt.Sprintf("%d", issue.Days)
		if issue.Deadline == "No deadline" {
			daysStr = "-"
			daysColor = style.Reset
		} else if issue.Days < 0 {
			daysColor = style.Red
		} else if issue.Days <= 3 {
			daysColor = style.Red
		} else if issue.Days <= 7 {
			daysColor = style.Yellow
		} else {
			daysColor = style.Green
		}

		// Make issue number clickable
		issueNumStr := fmt.Sprintf("#%d", issue.Number)
		clickableIssueNum := fmt.Sprintf("\x1b]8;;%s\x1b\\%s%s%s\x1b]8;;\x1b\\", issueURL, style.Cyan, issueNumStr, style.Reset)
		padding := 5 - len(issueNumStr)

		fmt.Printf("%s%s | %-35s | %-25s | %s%-12s | %4s%s\n",
			clickableIssueNum, strings.Repeat(" ", padding),
			title,
			company,
			daysColor, issue.Deadline, daysStr, style.Reset)
	}

	return nil
}

func extractCompany(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "### Company" {
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed != "" && !strings.HasPrefix(trimmed, "###") {
					return trimmed
				}
			}
			break
		}
	}
	return ""
}

type TagInfo struct {
	Tag         string
	IssueNumber string
	Company     string
	Role        string
	AppliedDate string
}

func runListApps(cfg *config.Config) error {
	// Resolve repo (flag > config)
	repo := listRepoFlag
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" {
		return fmt.Errorf("no repo configured. Run: cvx init")
	}

	// Get all tags from git
	cmd := exec.Command("git", "tag")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error listing tags: %w", err)
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		fmt.Println("No applications found")
		return nil
	}

	// Get issue numbers from tags
	issueNumbers := make([]string, 0)
	tagByIssue := make(map[string]string)

	for _, tag := range tags {
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, "-")
		if len(parts) < 4 {
			continue
		}

		issueNumber := parts[0]
		issueNumbers = append(issueNumbers, issueNumber)
		tagByIssue[issueNumber] = tag
	}

	if len(issueNumbers) == 0 {
		fmt.Println("No applications found")
		return nil
	}

	// Fetch all issues
	cli := gh.New()
	issueOutput, err := cli.IssueList(repo, "all", 100)
	if err != nil {
		return fmt.Errorf("error fetching issues: %w", err)
	}

	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}

	if err := json.Unmarshal(issueOutput, &issues); err != nil {
		return fmt.Errorf("error parsing issues: %w", err)
	}

	// Build a map of issue data
	issueData := make(map[string]struct {
		Title   string
		Company string
	})

	for _, issue := range issues {
		issueNumStr := fmt.Sprintf("%d", issue.Number)
		if _, exists := tagByIssue[issueNumStr]; !exists {
			continue
		}

		company := extractCompany(issue.Body)

		issueData[issueNumStr] = struct {
			Title   string
			Company string
		}{
			Title:   issue.Title,
			Company: company,
		}
	}

	// Build tag infos with actual issue data
	tagInfos := make([]TagInfo, 0)

	for issueNum, tag := range tagByIssue {
		data, exists := issueData[issueNum]
		if !exists {
			continue
		}

		// Parse applied date from tag (format: {issueNum}-...-YYYY-MM-DD)
		parts := strings.Split(tag, "-")
		appliedDate := parts[len(parts)-3] + "-" + parts[len(parts)-2] + "-" + parts[len(parts)-1]

		// Apply company filter if specified
		if listCompany != "" && !strings.Contains(strings.ToLower(data.Company), strings.ToLower(listCompany)) {
			continue
		}

		tagInfos = append(tagInfos, TagInfo{
			Tag:         tag,
			IssueNumber: issueNum,
			Company:     data.Company,
			Role:        data.Title,
			AppliedDate: appliedDate,
		})
	}

	if len(tagInfos) == 0 {
		if listCompany != "" {
			fmt.Printf("No applications found for company: %s\n", listCompany)
		} else {
			fmt.Println("No applications found")
		}
		return nil
	}

	// Sort by applied date (most recent first)
	sort.Slice(tagInfos, func(i, j int) bool {
		return tagInfos[i].AppliedDate > tagInfos[j].AppliedDate
	})

	// Print table header
	fmt.Printf("%s%sIssue | %-35s | %-25s | Applied%s\n", style.Bold, style.Cyan, "Role", "Company", style.Reset)
	fmt.Printf("%s", style.Cyan)
	fmt.Printf("------+-------------------------------------+---------------------------+----------\n")
	fmt.Printf("%s", style.Reset)

	// Print table rows
	for _, info := range tagInfos {
		role := info.Role
		issueURL := fmt.Sprintf("https://github.com/%s/issues/%s", repo, info.IssueNumber)

		// Truncate role if needed
		if utf8.RuneCountInString(role) > 35 {
			runes := []rune(role)
			role = string(runes[:35])
		}

		company := info.Company
		if len(company) > 25 {
			company = company[:25]
		}

		// Make issue number clickable
		issueNumStr := fmt.Sprintf("#%s", info.IssueNumber)
		clickableIssueNum := fmt.Sprintf("\x1b]8;;%s\x1b\\%s%s%s\x1b]8;;\x1b\\", issueURL, style.Cyan, issueNumStr, style.Reset)
		padding := 5 - len(issueNumStr)

		fmt.Printf("%s%s | %-35s | %-25s | %s%s%s\n",
			clickableIssueNum, strings.Repeat(" ", padding),
			role,
			company,
			style.Green, info.AppliedDate, style.Reset)
	}

	return nil
}
