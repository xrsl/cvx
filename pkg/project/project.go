package project

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client handles GitHub Project V2 operations
type Client struct {
	repo string
}

// New creates a new project client
func New(repo string) *Client {
	return &Client{repo: repo}
}

// ProjectInfo contains project details
type ProjectInfo struct {
	ID     string
	Number int
	Title  string
}

// FieldInfo contains field details
type FieldInfo struct {
	ID      string
	Name    string
	Type    string
	Options []OptionInfo // for single-select fields
}

// OptionInfo contains single-select option details
type OptionInfo struct {
	ID   string
	Name string
}

// graphqlWithVars executes a GraphQL query with variables via gh CLI
// Variables are passed safely using -F flags, preventing injection
func graphqlWithVars(query string, vars map[string]any) ([]byte, error) {
	args := []string{"api", "graphql", "-f", "query=" + query}

	// Add variables safely using -F (JSON) or -f (string)
	for key, val := range vars {
		switch v := val.(type) {
		case string:
			args = append(args, "-f", fmt.Sprintf("%s=%s", key, v))
		case int:
			args = append(args, "-F", fmt.Sprintf("%s=%d", key, v))
		default:
			// For complex types, marshal to JSON
			jsonVal, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal variable %s: %w", key, err)
			}
			args = append(args, "-F", fmt.Sprintf("%s=%s", key, string(jsonVal)))
		}
	}

	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, string(out))
	}
	return out, nil
}

// graphql executes a simple GraphQL query without variables
func graphql(query string) ([]byte, error) {
	return graphqlWithVars(query, nil)
}

// GetUserID returns the authenticated user's node ID
func GetUserID() (string, error) {
	query := `query { viewer { id } }`
	out, err := graphql(query)
	if err != nil {
		return "", fmt.Errorf("failed to get user ID: %w", err)
	}

	var result struct {
		Data struct {
			Viewer struct {
				ID string `json:"id"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	return result.Data.Viewer.ID, nil
}

// Create creates a new GitHub Project V2
func (c *Client) Create(title string, statuses []string) (*ProjectInfo, map[string]FieldInfo, error) {
	userID, err := GetUserID()
	if err != nil {
		return nil, nil, err
	}

	query := `mutation($ownerId: ID!, $title: String!) {
		createProjectV2(input: {ownerId: $ownerId, title: $title}) {
			projectV2 { id number title }
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"ownerId": userID,
		"title":   title,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create project: %w", err)
	}

	var createResult struct {
		Data struct {
			CreateProjectV2 struct {
				ProjectV2 struct {
					ID     string `json:"id"`
					Number int    `json:"number"`
					Title  string `json:"title"`
				} `json:"projectV2"`
			} `json:"createProjectV2"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &createResult); err != nil {
		return nil, nil, err
	}

	proj := &ProjectInfo{
		ID:     createResult.Data.CreateProjectV2.ProjectV2.ID,
		Number: createResult.Data.CreateProjectV2.ProjectV2.Number,
		Title:  createResult.Data.CreateProjectV2.ProjectV2.Title,
	}

	// Get existing fields to find the default Status field
	existingFields, err := c.DiscoverFields(proj.ID)
	if err != nil {
		return proj, nil, fmt.Errorf("failed to discover fields: %w", err)
	}

	fields := make(map[string]FieldInfo)

	// Update the existing Status field with job-specific statuses
	jobStatuses := []string{"To be Applied", "Applied", "Interview", "Offered", "Accepted", "Gone", "Let Go"}
	if statusField, ok := existingFields["status"]; ok {
		updatedField, err := c.updateSingleSelectOptions(proj.ID, statusField.ID, statusField.Options, jobStatuses)
		if err != nil {
			return proj, nil, fmt.Errorf("failed to update Status field: %w", err)
		}
		fields["status"] = *updatedField
	} else {
		// Fallback: create new field if Status doesn't exist
		statusField, err := c.createSingleSelectField(proj.ID, "Application Status", jobStatuses)
		if err != nil {
			return proj, nil, fmt.Errorf("failed to create Application Status field: %w", err)
		}
		fields["status"] = *statusField
	}

	// Create Company field (text)
	companyField, err := c.createTextField(proj.ID, "Company")
	if err != nil {
		return proj, nil, fmt.Errorf("failed to create Company field: %w", err)
	}
	fields["company"] = *companyField

	// Create Deadline field (date)
	deadlineField, err := c.createDateField(proj.ID, "Deadline")
	if err != nil {
		return proj, nil, fmt.Errorf("failed to create Deadline field: %w", err)
	}
	fields["deadline"] = *deadlineField

	// Create AppliedDate field (date)
	appliedField, err := c.createDateField(proj.ID, "AppliedDate")
	if err != nil {
		return proj, nil, fmt.Errorf("failed to create AppliedDate field: %w", err)
	}
	fields["applied_date"] = *appliedField

	// Link project to repo
	if err := c.linkToRepo(proj.ID); err != nil {
		// Non-fatal, project still usable
		fmt.Printf("Warning: Could not link project to repo: %v\n", err)
	}

	return proj, fields, nil
}

// SingleSelectOption represents an option for createProjectV2Field
type SingleSelectOption struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

func (c *Client) createSingleSelectField(projectID, name string, options []string) (*FieldInfo, error) {
	// Build options array for the mutation
	opts := make([]SingleSelectOption, len(options))
	for i, opt := range options {
		opts[i] = SingleSelectOption{
			Name:        opt,
			Description: "",
			Color:       "GRAY",
		}
	}

	query := `mutation($projectId: ID!, $name: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
		createProjectV2Field(input: {
			projectId: $projectId
			dataType: SINGLE_SELECT
			name: $name
			singleSelectOptions: $options
		}) {
			projectV2Field {
				... on ProjectV2SingleSelectField {
					id
					name
					options { id name }
				}
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"name":      name,
		"options":   opts,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			CreateProjectV2Field struct {
				ProjectV2Field struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Options []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"options"`
				} `json:"projectV2Field"`
			} `json:"createProjectV2Field"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	field := &FieldInfo{
		ID:   result.Data.CreateProjectV2Field.ProjectV2Field.ID,
		Name: result.Data.CreateProjectV2Field.ProjectV2Field.Name,
		Type: "SINGLE_SELECT",
	}
	for _, opt := range result.Data.CreateProjectV2Field.ProjectV2Field.Options {
		field.Options = append(field.Options, OptionInfo{ID: opt.ID, Name: opt.Name})
	}
	return field, nil
}

func (c *Client) updateSingleSelectOptions(projectID, fieldID string, existingOptions []OptionInfo, newOptions []string) (*FieldInfo, error) {
	// Delete existing options
	deleteQuery := `mutation($projectId: ID!, $fieldId: ID!, $optionId: String!) {
		deleteProjectV2SingleSelectFieldOption(input: {
			projectId: $projectId
			fieldId: $fieldId
			optionId: $optionId
		}) {
			projectV2SingleSelectFieldOption { id }
		}
	}`

	for _, opt := range existingOptions {
		if _, err := graphqlWithVars(deleteQuery, map[string]any{
			"projectId": projectID,
			"fieldId":   fieldID,
			"optionId":  opt.ID,
		}); err != nil {
			return nil, fmt.Errorf("failed to delete option %s: %w", opt.Name, err)
		}
	}

	// Create new options
	createQuery := `mutation($projectId: ID!, $fieldId: ID!, $name: String!) {
		createProjectV2SingleSelectFieldOption(input: {
			projectId: $projectId
			fieldId: $fieldId
			name: $name
			color: GRAY
		}) {
			projectV2SingleSelectFieldOption { id name }
		}
	}`

	var createdOptions []OptionInfo
	for _, optName := range newOptions {
		out, err := graphqlWithVars(createQuery, map[string]any{
			"projectId": projectID,
			"fieldId":   fieldID,
			"name":      optName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create option %s: %w", optName, err)
		}

		var result struct {
			Data struct {
				CreateProjectV2SingleSelectFieldOption struct {
					ProjectV2SingleSelectFieldOption struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"projectV2SingleSelectFieldOption"`
				} `json:"createProjectV2SingleSelectFieldOption"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &result); err != nil {
			return nil, err
		}
		opt := result.Data.CreateProjectV2SingleSelectFieldOption.ProjectV2SingleSelectFieldOption
		createdOptions = append(createdOptions, OptionInfo{ID: opt.ID, Name: opt.Name})
	}

	return &FieldInfo{
		ID:      fieldID,
		Name:    "Status",
		Type:    "SINGLE_SELECT",
		Options: createdOptions,
	}, nil
}

func (c *Client) createTextField(projectID, name string) (*FieldInfo, error) {
	query := `mutation($projectId: ID!, $name: String!) {
		createProjectV2Field(input: {
			projectId: $projectId
			dataType: TEXT
			name: $name
		}) {
			projectV2Field {
				... on ProjectV2Field { id name }
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"name":      name,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			CreateProjectV2Field struct {
				ProjectV2Field struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"projectV2Field"`
			} `json:"createProjectV2Field"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	return &FieldInfo{
		ID:   result.Data.CreateProjectV2Field.ProjectV2Field.ID,
		Name: result.Data.CreateProjectV2Field.ProjectV2Field.Name,
		Type: "TEXT",
	}, nil
}

func (c *Client) createDateField(projectID, name string) (*FieldInfo, error) {
	query := `mutation($projectId: ID!, $name: String!) {
		createProjectV2Field(input: {
			projectId: $projectId
			dataType: DATE
			name: $name
		}) {
			projectV2Field {
				... on ProjectV2Field { id name }
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"name":      name,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			CreateProjectV2Field struct {
				ProjectV2Field struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"projectV2Field"`
			} `json:"createProjectV2Field"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	return &FieldInfo{
		ID:   result.Data.CreateProjectV2Field.ProjectV2Field.ID,
		Name: result.Data.CreateProjectV2Field.ProjectV2Field.Name,
		Type: "DATE",
	}, nil
}

func (c *Client) linkToRepo(projectID string) error {
	parts := strings.Split(c.repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format")
	}

	// Get repo ID
	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) { id }
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"owner": parts[0],
		"name":  parts[1],
	})
	if err != nil {
		return err
	}

	var repoResult struct {
		Data struct {
			Repository struct {
				ID string `json:"id"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &repoResult); err != nil {
		return err
	}

	// Link project to repo
	linkQuery := `mutation($projectId: ID!, $repositoryId: ID!) {
		linkProjectV2ToRepository(input: {projectId: $projectId, repositoryId: $repositoryId}) {
			repository { id }
		}
	}`

	_, err = graphqlWithVars(linkQuery, map[string]any{
		"projectId":    projectID,
		"repositoryId": repoResult.Data.Repository.ID,
	})
	return err
}

// ListProjects returns projects for the repo
func (c *Client) ListProjects() ([]ProjectInfo, error) {
	parts := strings.Split(c.repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format")
	}
	owner := parts[0]

	// Try user projects first, then repo projects
	projects, err := c.listUserProjects(owner)
	if err == nil && len(projects) > 0 {
		return projects, nil
	}

	return c.listRepoProjects(owner, parts[1])
}

func (c *Client) listUserProjects(owner string) ([]ProjectInfo, error) {
	query := `query($login: String!) {
		user(login: $login) {
			projectsV2(first: 20) {
				nodes { id number title }
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"login": owner,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				ProjectsV2 struct {
					Nodes []struct {
						ID     string `json:"id"`
						Number int    `json:"number"`
						Title  string `json:"title"`
					} `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var projects []ProjectInfo
	for _, p := range result.Data.User.ProjectsV2.Nodes {
		projects = append(projects, ProjectInfo{ID: p.ID, Number: p.Number, Title: p.Title})
	}
	return projects, nil
}

func (c *Client) listRepoProjects(owner, name string) ([]ProjectInfo, error) {
	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) {
			projectsV2(first: 20) {
				nodes { id number title }
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"owner": owner,
		"name":  name,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Repository struct {
				ProjectsV2 struct {
					Nodes []struct {
						ID     string `json:"id"`
						Number int    `json:"number"`
						Title  string `json:"title"`
					} `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var projects []ProjectInfo
	for _, p := range result.Data.Repository.ProjectsV2.Nodes {
		projects = append(projects, ProjectInfo{ID: p.ID, Number: p.Number, Title: p.Title})
	}
	return projects, nil
}

// DiscoverFields returns field IDs for an existing project
func (c *Client) DiscoverFields(projectID string) (map[string]FieldInfo, error) {
	query := `query($nodeId: ID!) {
		node(id: $nodeId) {
			... on ProjectV2 {
				fields(first: 30) {
					nodes {
						... on ProjectV2SingleSelectField {
							id name dataType
							options { id name }
						}
						... on ProjectV2Field {
							id name dataType
						}
					}
				}
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"nodeId": projectID,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Node struct {
				Fields struct {
					Nodes []struct {
						ID       string `json:"id"`
						Name     string `json:"name"`
						DataType string `json:"dataType"`
						Options  []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"options"`
					} `json:"nodes"`
				} `json:"fields"`
			} `json:"node"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	fields := make(map[string]FieldInfo)
	for _, f := range result.Data.Node.Fields.Nodes {
		key := strings.ToLower(strings.ReplaceAll(f.Name, " ", "_"))
		field := FieldInfo{
			ID:   f.ID,
			Name: f.Name,
			Type: f.DataType,
		}
		for _, opt := range f.Options {
			field.Options = append(field.Options, OptionInfo{ID: opt.ID, Name: opt.Name})
		}
		fields[key] = field
	}
	return fields, nil
}

// GetIssueNodeID returns the node ID for an issue
func (c *Client) GetIssueNodeID(issueNumber int) (string, error) {
	parts := strings.Split(c.repo, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format")
	}

	query := `query($owner: String!, $name: String!, $number: Int!) {
		repository(owner: $owner, name: $name) {
			issue(number: $number) { id }
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"owner":  parts[0],
		"name":   parts[1],
		"number": issueNumber,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			Repository struct {
				Issue struct {
					ID string `json:"id"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}

	return result.Data.Repository.Issue.ID, nil
}

// AddItem adds an issue to a project
func (c *Client) AddItem(projectID, issueNodeID string) (string, error) {
	query := `mutation($projectId: ID!, $contentId: ID!) {
		addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
			item { id }
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"contentId": issueNodeID,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			AddProjectV2ItemById struct {
				Item struct {
					ID string `json:"id"`
				} `json:"item"`
			} `json:"addProjectV2ItemById"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}

	return result.Data.AddProjectV2ItemById.Item.ID, nil
}

// SetTextField sets a text field value
func (c *Client) SetTextField(projectID, itemID, fieldID, value string) error {
	query := `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: String!) {
		updateProjectV2ItemFieldValue(input: {
			projectId: $projectId
			itemId: $itemId
			fieldId: $fieldId
			value: {text: $value}
		}) {
			projectV2Item { id }
		}
	}`

	_, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"value":     value,
	})
	return err
}

// SetDateField sets a date field value
func (c *Client) SetDateField(projectID, itemID, fieldID, date string) error {
	query := `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $date: Date!) {
		updateProjectV2ItemFieldValue(input: {
			projectId: $projectId
			itemId: $itemId
			fieldId: $fieldId
			value: {date: $date}
		}) {
			projectV2Item { id }
		}
	}`

	_, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"date":      date,
	})
	return err
}

// SetStatusField sets a single-select field value
func (c *Client) SetStatusField(projectID, itemID, fieldID, optionID string) error {
	query := `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
		updateProjectV2ItemFieldValue(input: {
			projectId: $projectId
			itemId: $itemId
			fieldId: $fieldId
			value: {singleSelectOptionId: $optionId}
		}) {
			projectV2Item { id }
		}
	}`

	_, err := graphqlWithVars(query, map[string]any{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"optionId":  optionID,
	})
	return err
}

// GetItemID returns the project item ID for an issue
func (c *Client) GetItemID(projectID string, issueNumber int) (string, error) {
	parts := strings.Split(c.repo, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format")
	}

	query := `query($owner: String!, $name: String!, $number: Int!) {
		repository(owner: $owner, name: $name) {
			issue(number: $number) {
				projectItems(first: 10) {
					nodes {
						id
						project { id }
					}
				}
			}
		}
	}`

	out, err := graphqlWithVars(query, map[string]any{
		"owner":  parts[0],
		"name":   parts[1],
		"number": issueNumber,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID      string `json:"id"`
							Project struct {
								ID string `json:"id"`
							} `json:"project"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}

	for _, item := range result.Data.Repository.Issue.ProjectItems.Nodes {
		if item.Project.ID == projectID {
			return item.ID, nil
		}
	}

	return "", fmt.Errorf("issue not in project")
}
