// Package compose provides docker compose orchestration and environment variable parsing.
package compose

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// EnvVarReference represents an environment variable reference found in a compose file.
type EnvVarReference struct {
	Name         string   // Variable name (e.g., "DATABASE_URL")
	HasDefault   bool     // Has default value (e.g., ${VAR:-default} or ${VAR-default})
	DefaultValue string   // The default value if HasDefault is true
	IsRequired   bool     // Explicitly marked required (e.g., ${VAR:?error} or ${VAR?error})
	ErrorMessage string   // Custom error message if IsRequired is true
	Locations    []string // Where in compose file this var appears (service names)
}

// envVarPattern matches all forms of environment variable references:
// - ${VAR} or $VAR - basic substitution
// - ${VAR:-default} - default if unset or empty
// - ${VAR-default} - default if unset (not if empty)
// - ${VAR:?error} - error if unset or empty
// - ${VAR?error} - error if unset (not if empty)
var envVarPattern = regexp.MustCompile(
	// Group 1: ${VAR...} form - variable name
	// Group 2: operator (:-|:?|?|-)
	// Group 3: value after operator
	`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:(:-|:\?|\?|-)([^}]*))?\}` +
		// OR Group 4: $VAR form
		`|\$([A-Za-z_][A-Za-z0-9_]*)`,
)

// ParseEnvVars extracts all environment variable references from a compose file.
// It returns a deduplicated list of variables with metadata about defaults and requirements.
func ParseEnvVars(composeFilePath string) ([]EnvVarReference, error) {
	// Read compose file
	data, err := os.ReadFile(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	// Parse YAML to get service structure for better location tracking
	var composeData map[string]interface{}
	if err := yaml.Unmarshal(data, &composeData); err != nil {
		return nil, fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Extract service names for location tracking
	serviceNames := extractServiceNames(composeData)

	// Find all variable references in the raw content with positions
	content := string(data)
	matchesWithIndex := envVarPattern.FindAllStringSubmatchIndex(content, -1)
	matches := envVarPattern.FindAllStringSubmatch(content, -1)

	// Use map to deduplicate variables
	// Key: variable name, Value: EnvVarReference with aggregated metadata
	vars := make(map[string]*EnvVarReference)

	for i, match := range matches {
		var name, operator, value string

		// Check which form matched
		if match[1] != "" {
			// ${VAR...} form
			name = match[1]
			operator = match[2]
			value = match[3]
		} else if match[4] != "" {
			// $VAR form (no operator, no value)
			name = match[4]
		} else {
			continue // Shouldn't happen with our regex, but be safe
		}

		// Get or create reference
		ref, exists := vars[name]
		if !exists {
			ref = &EnvVarReference{
				Name:      name,
				Locations: []string{},
			}
			vars[name] = ref
		}

		// Parse operator to determine type
		switch operator {
		case ":-", "-":
			// Default value forms
			ref.HasDefault = true
			// Use the longest/most specific default value seen
			if value != "" && (ref.DefaultValue == "" || len(value) > len(ref.DefaultValue)) {
				ref.DefaultValue = value
			}
		case ":?", "?":
			// Required forms with error message
			ref.IsRequired = true
			// Use the most specific error message seen
			if value != "" && ref.ErrorMessage == "" {
				ref.ErrorMessage = value
			}
		default:
			// Basic ${VAR} or $VAR form - no special handling
			// These are "soft" requirements (compose uses empty string if unset)
		}

		// Track approximate location (which service likely uses it)
		// Use the position of this specific match
		matchStartPos := matchesWithIndex[i][0]
		location := findVariableLocationAt(content, matchStartPos, serviceNames)
		if location != "" && !contains(ref.Locations, location) {
			ref.Locations = append(ref.Locations, location)
		}
	}

	// Convert map to sorted slice
	result := make([]EnvVarReference, 0, len(vars))
	for _, ref := range vars {
		// Sort locations for deterministic output
		sort.Strings(ref.Locations)
		result = append(result, *ref)
	}

	// Sort by name for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetMissingVars compares required variables against stored variables
// and returns a list of variables that are missing or empty.
func GetMissingVars(required []EnvVarReference, stored map[string]string) []EnvVarReference {
	var missing []EnvVarReference

	for _, req := range required {
		value, exists := stored[req.Name]

		// Consider variable missing if:
		// 1. Not in stored map, OR
		// 2. Empty string in stored map
		if !exists || value == "" {
			missing = append(missing, req)
		}
	}

	return missing
}

// extractServiceNames extracts service names from parsed compose data.
// Returns a slice of service names found in the "services" section.
func extractServiceNames(composeData map[string]interface{}) []string {
	var names []string

	services, ok := composeData["services"].(map[string]interface{})
	if !ok {
		return names
	}

	for name := range services {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// findVariableLocationAt attempts to determine which service uses a variable
// by finding the service definition closest to where the variable appears at the given position.
func findVariableLocationAt(content string, position int, serviceNames []string) string {
	if position < 0 || position >= len(content) {
		return ""
	}

	// Look backwards for service definition
	// Find the most recent "servicename:" pattern before this variable
	beforeContent := content[:position]

	var closestService string
	closestDistance := len(beforeContent)

	for _, serviceName := range serviceNames {
		// Look for "  servicename:" pattern (2-space indent for service definition)
		pattern := fmt.Sprintf("  %s:", serviceName)
		lastIndex := strings.LastIndex(beforeContent, pattern)

		if lastIndex != -1 {
			distance := position - lastIndex
			if distance < closestDistance {
				closestDistance = distance
				closestService = serviceName
			}
		}
	}

	return closestService
}

// contains checks if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GenerateEnvExample creates a .env.example file from environment variable references.
// The file contains all variables with comments indicating type (required/optional) and defaults.
func GenerateEnvExample(vars []EnvVarReference, outputPath string) error {
	var lines []string

	// Group by required vs optional
	var required, optional []EnvVarReference
	for _, v := range vars {
		if v.IsRequired {
			required = append(required, v)
		} else if v.HasDefault {
			optional = append(optional, v)
		} else {
			// Basic variables (neither required nor optional) - treat as required
			required = append(required, v)
		}
	}

	// Header
	lines = append(lines, "# Environment Variables")
	lines = append(lines, "# Generated by OtterStack")
	lines = append(lines, "")

	// Required section
	if len(required) > 0 {
		lines = append(lines, "# Required Variables")
		lines = append(lines, "# These must be set for the application to work")
		lines = append(lines, "")

		for _, v := range required {
			if len(v.Locations) > 0 {
				lines = append(lines, fmt.Sprintf("# Used by: %s", strings.Join(v.Locations, ", ")))
			}
			if v.ErrorMessage != "" {
				lines = append(lines, fmt.Sprintf("# Error if unset: %s", v.ErrorMessage))
			}
			lines = append(lines, fmt.Sprintf("%s=", v.Name))
			lines = append(lines, "")
		}
	}

	// Optional section
	if len(optional) > 0 {
		lines = append(lines, "# Optional Variables")
		lines = append(lines, "# These have default values but can be overridden")
		lines = append(lines, "")

		for _, v := range optional {
			if len(v.Locations) > 0 {
				lines = append(lines, fmt.Sprintf("# Used by: %s", strings.Join(v.Locations, ", ")))
			}
			lines = append(lines, fmt.Sprintf("# Default: %s", v.DefaultValue))
			lines = append(lines, fmt.Sprintf("# %s=", v.Name))
			lines = append(lines, "")
		}
	}

	// Write to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .env.example: %w", err)
	}

	return nil
}
