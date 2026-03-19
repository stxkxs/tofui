package tfstate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TFState represents the top-level structure of an OpenTofu/Terraform state file.
type TFState struct {
	Version          int                        `json:"version"`
	TerraformVersion string                     `json:"terraform_version"`
	Serial           int                        `json:"serial"`
	Lineage          string                     `json:"lineage"`
	Outputs          map[string]TFStateOutput   `json:"outputs"`
	Resources        []TFStateResource          `json:"resources"`
}

// TFStateOutput represents an output value in the state file.
type TFStateOutput struct {
	Value interface{} `json:"value"`
	Type  interface{} `json:"type"`
}

// Output is a simplified output representation returned by the API.
type Output struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// TFStateResource represents a resource block in the state file.
type TFStateResource struct {
	Module    string            `json:"module"`
	Mode      string            `json:"mode"`
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Provider  string            `json:"provider"`
	Instances []TFStateInstance  `json:"instances"`
}

// TFStateInstance represents a single instance of a resource.
type TFStateInstance struct {
	SchemaVersion int                    `json:"schema_version"`
	Attributes    map[string]interface{} `json:"attributes"`
	AttributesFlat map[string]string     `json:"attributes_flat"`
}

// Resource is a simplified representation returned by the API.
type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Module     string                 `json:"module"`
	Provider   string                 `json:"provider"`
	Mode       string                 `json:"mode"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ParseResources extracts a flat list of resources from raw state JSON.
func ParseResources(data []byte) ([]Resource, error) {
	var state TFState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	var resources []Resource
	for _, r := range state.Resources {
		provider := cleanProviderName(r.Provider)

		for _, inst := range r.Instances {
			attrs := inst.Attributes
			if attrs == nil {
				attrs = map[string]interface{}{}
			}
			resources = append(resources, Resource{
				Type:       r.Type,
				Name:       r.Name,
				Module:     r.Module,
				Provider:   provider,
				Mode:       r.Mode,
				Attributes: attrs,
			})
		}

		// If no instances, still include the resource shell
		if len(r.Instances) == 0 {
			resources = append(resources, Resource{
				Type:       r.Type,
				Name:       r.Name,
				Module:     r.Module,
				Provider:   provider,
				Mode:       r.Mode,
				Attributes: map[string]interface{}{},
			})
		}
	}

	if resources == nil {
		resources = []Resource{}
	}
	return resources, nil
}

// ParseOutputs extracts output values from raw state JSON.
func ParseOutputs(data []byte) ([]Output, error) {
	var state TFState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	var outputs []Output
	for name, out := range state.Outputs {
		// Convert value to string representation
		var valueStr string
		switch v := out.Value.(type) {
		case string:
			valueStr = v
		default:
			b, _ := json.Marshal(v)
			valueStr = string(b)
		}

		// Convert type to string
		var typeStr string
		switch t := out.Type.(type) {
		case string:
			typeStr = t
		default:
			b, _ := json.Marshal(t)
			typeStr = string(b)
		}

		outputs = append(outputs, Output{
			Name:  name,
			Value: valueStr,
			Type:  typeStr,
		})
	}

	if outputs == nil {
		outputs = []Output{}
	}
	return outputs, nil
}

// cleanProviderName strips the registry prefix from a provider string.
// e.g. "registry.terraform.io/hashicorp/aws" → "aws"
func cleanProviderName(provider string) string {
	// Format: registry.terraform.io/hashicorp/aws or provider["registry.terraform.io/hashicorp/aws"]
	p := strings.TrimPrefix(provider, "provider[\"")
	p = strings.TrimSuffix(p, "\"]")
	parts := strings.Split(p, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return provider
}
