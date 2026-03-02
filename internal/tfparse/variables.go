package tfparse

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxFileSize     = 1 << 20 // 1 MB
	maxVariables    = 500
)

// DiscoveredVariable represents a variable block parsed from a .tf file.
type DiscoveredVariable struct {
	Name        string  `json:"name"`
	Type        string  `json:"type,omitempty"`
	Description string  `json:"description,omitempty"`
	Default     *string `json:"default,omitempty"`
	Required    bool    `json:"required"`
}

var variableBlockRe = regexp.MustCompile(`(?m)^variable\s+"([^"]+)"\s*\{`)

// ParseVariables extracts variable blocks from HCL content using regex and brace counting.
func ParseVariables(content string) []DiscoveredVariable {
	matches := variableBlockRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}

	var vars []DiscoveredVariable
	for _, m := range matches {
		if len(vars) >= maxVariables {
			break
		}

		name := content[m[2]:m[3]]
		// Find the opening brace
		braceStart := strings.Index(content[m[0]:], "{")
		if braceStart < 0 {
			continue
		}
		blockStart := m[0] + braceStart

		// Count braces to find block end
		depth := 0
		blockEnd := -1
		for i := blockStart; i < len(content); i++ {
			switch content[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					blockEnd = i
					goto found
				}
			case '"':
				// Skip quoted strings to avoid counting braces inside them
				for i++; i < len(content) && content[i] != '"'; i++ {
					if content[i] == '\\' {
						i++
					}
				}
			case '#':
				// Skip single-line comments
				for i++; i < len(content) && content[i] != '\n'; i++ {
				}
			}
		}
	found:
		if blockEnd < 0 {
			continue
		}

		body := content[blockStart+1 : blockEnd]
		v := DiscoveredVariable{
			Name:        name,
			Type:        extractAttribute(body, "type"),
			Description: extractStringAttribute(body, "description"),
		}

		if def, ok := extractDefault(body); ok {
			v.Default = &def
			v.Required = false
		} else {
			v.Required = true
		}

		vars = append(vars, v)
	}

	return vars
}

var (
	stringAttrRe = regexp.MustCompile(`(?m)^\s*(\w+)\s*=\s*"([^"]*)"`)
	typeAttrRe   = regexp.MustCompile(`(?m)^\s*type\s*=\s*(.+)`)
	defaultAttrRe = regexp.MustCompile(`(?m)^\s*default\s*=\s*(.+)`)
)

func extractStringAttribute(body, name string) string {
	for _, m := range stringAttrRe.FindAllStringSubmatch(body, -1) {
		if m[1] == name {
			return m[2]
		}
	}
	return ""
}

func extractAttribute(body, name string) string {
	if name == "type" {
		m := typeAttrRe.FindStringSubmatch(body)
		if m == nil {
			return ""
		}
		val := strings.TrimSpace(m[1])
		// Handle multiline type expressions like list(object({...}))
		if needsBalancing(val) {
			val = balanceValue(body, typeAttrRe, val)
		}
		return strings.TrimSpace(val)
	}
	return extractStringAttribute(body, name)
}

func extractDefault(body string) (string, bool) {
	m := defaultAttrRe.FindStringSubmatchIndex(body)
	if m == nil {
		return "", false
	}
	// Get the value starting position
	valStart := m[2]
	val := strings.TrimSpace(body[valStart:m[3]])

	// If it's a simple quoted string
	if strings.HasPrefix(val, "\"") {
		end := strings.Index(val[1:], "\"")
		if end >= 0 {
			return val[1 : end+1], true
		}
	}

	// If it needs balancing (maps, lists, objects)
	if needsBalancing(val) {
		val = balanceValue(body, defaultAttrRe, val)
	}

	return strings.TrimSpace(val), true
}

func needsBalancing(val string) bool {
	opens := strings.Count(val, "{") + strings.Count(val, "[") + strings.Count(val, "(")
	closes := strings.Count(val, "}") + strings.Count(val, "]") + strings.Count(val, ")")
	return opens > closes
}

func balanceValue(body string, re *regexp.Regexp, initial string) string {
	loc := re.FindStringIndex(body)
	if loc == nil {
		return initial
	}
	// Start scanning from where the value begins
	eqIdx := strings.Index(body[loc[0]:], "=")
	if eqIdx < 0 {
		return initial
	}
	start := loc[0] + eqIdx + 1

	depth := 0
	var end int
	for i := start; i < len(body); i++ {
		switch body[i] {
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
			if depth <= 0 {
				end = i + 1
				return strings.TrimSpace(body[start:end])
			}
		case '"':
			for i++; i < len(body) && body[i] != '"'; i++ {
				if body[i] == '\\' {
					i++
				}
			}
		}
	}
	return initial
}

// ParseDirectory reads all .tf files in a directory (non-recursive) and returns deduplicated variables.
func ParseDirectory(dir string) ([]DiscoveredVariable, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []DiscoveredVariable

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}
		if filepath.Ext(entry.Name()) != ".tf" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Size() > maxFileSize {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		vars := ParseVariables(string(data))
		for _, v := range vars {
			if seen[v.Name] {
				continue
			}
			seen[v.Name] = true
			result = append(result, v)
			if len(result) >= maxVariables {
				return result, nil
			}
		}
	}

	return result, nil
}
