package schema

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed default_template.yml
var defaultSchemaYAML []byte

// DefaultSchemaYAML returns the raw default schema YAML bytes
func DefaultSchemaYAML() []byte {
	return defaultSchemaYAML
}

// LoadDefault returns the bundled default schema
func LoadDefault() (*Schema, error) {
	var raw rawTemplate
	if err := yaml.Unmarshal(defaultSchemaYAML, &raw); err != nil {
		return nil, err
	}

	schema := &Schema{
		Name:        raw.Name,
		Description: raw.Description,
		Labels:      raw.Labels,
		Assignees:   raw.Assignees,
	}

	for _, f := range raw.Body {
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
