package tfparse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVariables_RequiredVar(t *testing.T) {
	content := `
variable "region" {
  type        = string
  description = "AWS region"
}
`
	vars := ParseVariables(content)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(vars))
	}
	v := vars[0]
	if v.Name != "region" {
		t.Errorf("expected name 'region', got %q", v.Name)
	}
	if v.Type != "string" {
		t.Errorf("expected type 'string', got %q", v.Type)
	}
	if v.Description != "AWS region" {
		t.Errorf("expected description 'AWS region', got %q", v.Description)
	}
	if !v.Required {
		t.Error("expected Required=true")
	}
	if v.Default != nil {
		t.Errorf("expected Default=nil, got %q", *v.Default)
	}
}

func TestParseVariables_AllFields(t *testing.T) {
	content := `
variable "instance_type" {
  type        = string
  description = "EC2 instance type"
  default     = "t3.micro"
}
`
	vars := ParseVariables(content)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(vars))
	}
	v := vars[0]
	if v.Name != "instance_type" {
		t.Errorf("expected name 'instance_type', got %q", v.Name)
	}
	if v.Required {
		t.Error("expected Required=false")
	}
	if v.Default == nil || *v.Default != "t3.micro" {
		t.Errorf("expected Default='t3.micro', got %v", v.Default)
	}
}

func TestParseVariables_NestedBraces(t *testing.T) {
	content := `
variable "tags" {
  type = map(object({
    name  = string
    value = string
  }))
  description = "Resource tags"
  default     = {}
}
`
	vars := ParseVariables(content)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(vars))
	}
	v := vars[0]
	if v.Name != "tags" {
		t.Errorf("expected name 'tags', got %q", v.Name)
	}
	if v.Required {
		t.Error("expected Required=false for var with default")
	}
}

func TestParseVariables_Multiple(t *testing.T) {
	content := `
variable "name" {
  type = string
}

variable "count" {
  type    = number
  default = 1
}

variable "enabled" {
  type    = bool
  default = true
}
`
	vars := ParseVariables(content)
	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}
	if vars[0].Name != "name" || !vars[0].Required {
		t.Errorf("var[0]: expected name='name', required=true; got name=%q, required=%v", vars[0].Name, vars[0].Required)
	}
	if vars[1].Name != "count" || vars[1].Required {
		t.Errorf("var[1]: expected name='count', required=false; got name=%q, required=%v", vars[1].Name, vars[1].Required)
	}
	if vars[2].Name != "enabled" || vars[2].Required {
		t.Errorf("var[2]: expected name='enabled', required=false; got name=%q, required=%v", vars[2].Name, vars[2].Required)
	}
}

func TestParseVariables_NoVars(t *testing.T) {
	content := `
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
}
`
	vars := ParseVariables(content)
	if len(vars) != 0 {
		t.Fatalf("expected 0 variables, got %d", len(vars))
	}
}

func TestParseVariables_EmptyFile(t *testing.T) {
	vars := ParseVariables("")
	if len(vars) != 0 {
		t.Fatalf("expected 0 variables, got %d", len(vars))
	}
}

func TestParseVariables_ListDefault(t *testing.T) {
	content := `
variable "subnets" {
  type    = list(string)
  default = ["a", "b"]
}
`
	vars := ParseVariables(content)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(vars))
	}
	if vars[0].Required {
		t.Error("expected Required=false for var with list default")
	}
}

func TestParseDirectory(t *testing.T) {
	dir := t.TempDir()

	// Write a .tf file
	tf1 := `
variable "region" {
  type = string
}
variable "env" {
  type    = string
  default = "dev"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tf1), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a second .tf file with duplicate
	tf2 := `
variable "region" {
  type = string
}
variable "bucket" {
  type = string
}
`
	if err := os.WriteFile(filepath.Join(dir, "vars.tf"), []byte(tf2), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-tf file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 3 unique vars: region, env, bucket (region deduped)
	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}

	names := map[string]bool{}
	for _, v := range vars {
		names[v.Name] = true
	}
	for _, expected := range []string{"region", "env", "bucket"} {
		if !names[expected] {
			t.Errorf("expected variable %q not found", expected)
		}
	}
}
