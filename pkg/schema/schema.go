package schema

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Field represents a single field from GitHub issue template
type Field struct {
	ID          string
	Label       string
	Placeholder string
	Required    bool
	Type        string // "input" or "textarea"
}

// Schema represents parsed GitHub issue template
type Schema struct {
	Name        string
	Description string
	Labels      []string
	Assignees   []string
	Fields      []Field
}

// Raw structures for parsing GitHub issue template YAML
type rawTemplate struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Labels      []string `yaml:"labels"`
	Assignees   []string `yaml:"assignees"`
	Body        []rawField `yaml:"body"`
}

type rawField struct {
	Type       string `yaml:"type"`
	ID         string `yaml:"id"`
	Attributes struct {
		Label       string `yaml:"label"`
		Placeholder string `yaml:"placeholder"`
	} `yaml:"attributes"`
	Validations struct {
		Required bool `yaml:"required"`
	} `yaml:"validations"`
}

// Load parses a GitHub issue template YAML file
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	var raw rawTemplate
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	schema := &Schema{
		Name:        raw.Name,
		Description: raw.Description,
		Labels:      raw.Labels,
		Assignees:   raw.Assignees,
	}

	for _, f := range raw.Body {
		// Only support input and textarea types
		if f.Type != "input" && f.Type != "textarea" {
			continue
		}

		schema.Fields = append(schema.Fields, Field{
			ID:          f.ID,
			Label:       f.Attributes.Label,
			Placeholder: f.Attributes.Placeholder,
			Required:    f.Validations.Required,
			Type:        f.Type,
		})
	}

	return schema, nil
}

// GeneratePrompt creates an AI prompt for extracting fields
func (s *Schema) GeneratePrompt(url, jobText string) string {
	var sb strings.Builder

	sb.WriteString("Extract job posting info. Return ONLY valid JSON with these exact keys:\n")

	for _, f := range s.Fields {
		hint := f.Placeholder
		if hint == "" {
			hint = f.Label
		}

		nullHint := ""
		if !f.Required {
			nullHint = " or null if not found"
		}

		sb.WriteString(fmt.Sprintf("- %s: %s%s\n", f.ID, hint, nullHint))
	}

	sb.WriteString(fmt.Sprintf("\nJob URL: %s\n\nJob posting:\n%s", url, jobText))

	return sb.String()
}

// BuildIssueBody creates GitHub issue body from extracted data
func (s *Schema) BuildIssueBody(data map[string]any) string {
	var sb strings.Builder

	for _, f := range s.Fields {
		sb.WriteString(fmt.Sprintf("### %s\n\n", f.Label))

		val, ok := data[f.ID]
		if !ok || val == nil {
			sb.WriteString("_No response_")
		} else {
			sb.WriteString(fmt.Sprintf("%v", val))
		}

		sb.WriteString("\n\n")
	}

	return strings.TrimSuffix(sb.String(), "\n\n")
}

// GetTitle extracts issue title from data (uses first required field or "title" field)
func (s *Schema) GetTitle(data map[string]any) string {
	// Try common title fields
	for _, key := range []string{"title", "job-title", "position"} {
		if val, ok := data[key]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
	}

	// Fall back to first required field that's not a URL or textarea
	for _, f := range s.Fields {
		if f.Required && f.Type == "input" && !strings.Contains(strings.ToLower(f.ID), "url") {
			if val, ok := data[f.ID]; ok && val != nil {
				return fmt.Sprintf("%v", val)
			}
		}
	}

	return "Job Application"
}
