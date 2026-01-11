// Package validate provides validation functions for OtterStack.
package validate

import (
	"fmt"
	"strings"

	"github.com/jayteealao/otterstack/internal/compose"
)

// ValidationResult represents the outcome of environment variable validation.
type ValidationResult struct {
	AllPresent bool         // True if all required variables are present
	Missing    []MissingVar // Variables that are missing and required
	Optional   []MissingVar // Variables that are missing but have defaults
}

// MissingVar represents an environment variable that is not set.
type MissingVar struct {
	Name         string   // Variable name
	IsRequired   bool     // True if explicitly marked required (${VAR:?error})
	HasDefault   bool     // True if has default value (${VAR:-default})
	DefaultValue string   // The default value if HasDefault is true
	ErrorMessage string   // Custom error message if IsRequired is true
	Services     []string // Which services need this variable
}

// ValidateEnvVars checks if all required environment variables are present.
// It parses the compose file to find required variables and compares against stored variables.
// Returns a ValidationResult indicating which variables are missing.
func ValidateEnvVars(composeFilePath string, storedVars map[string]string) (*ValidationResult, error) {
	// Parse compose file to get all variable references
	requiredVars, err := compose.ParseEnvVars(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get missing variables
	missingRefs := compose.GetMissingVars(requiredVars, storedVars)

	// Categorize into required and optional
	var required, optional []MissingVar

	for _, ref := range missingRefs {
		mv := MissingVar{
			Name:         ref.Name,
			IsRequired:   ref.IsRequired,
			HasDefault:   ref.HasDefault,
			DefaultValue: ref.DefaultValue,
			ErrorMessage: ref.ErrorMessage,
			Services:     ref.Locations,
		}

		if ref.IsRequired {
			// Explicitly required with :? or ?
			required = append(required, mv)
		} else if ref.HasDefault {
			// Has default value - optional
			optional = append(optional, mv)
		} else {
			// Basic ${VAR} form without default - treat as required
			// Docker Compose will use empty string but this is usually wrong
			mv.IsRequired = true
			required = append(required, mv)
		}
	}

	result := &ValidationResult{
		AllPresent: len(required) == 0,
		Missing:    required,
		Optional:   optional,
	}

	return result, nil
}

// FormatValidationError creates a user-friendly error message with a checklist
// of missing variables and clear instructions for how to fix them.
func FormatValidationError(result *ValidationResult, projectName string) string {
	if result.AllPresent {
		return ""
	}

	var lines []string

	// Header
	lines = append(lines, "")
	lines = append(lines, "Missing environment variables detected:")
	lines = append(lines, "")

	// Required section
	if len(result.Missing) > 0 {
		lines = append(lines, "Required (deployment will fail):")
		for _, v := range result.Missing {
			// Variable name with X mark
			line := fmt.Sprintf("  ✗ %s", v.Name)

			// Add services that need it
			if len(v.Services) > 0 {
				line += fmt.Sprintf(" (needed by: %s)", strings.Join(v.Services, ", "))
			}

			// Add custom error message if present
			if v.ErrorMessage != "" {
				lines = append(lines, line)
				lines = append(lines, fmt.Sprintf("    Error: %s", v.ErrorMessage))
			} else {
				lines = append(lines, line)
			}
		}
		lines = append(lines, "")
	}

	// Optional section
	if len(result.Optional) > 0 {
		lines = append(lines, "Optional (will use defaults):")
		for _, v := range result.Optional {
			line := fmt.Sprintf("  ⚠ %s", v.Name)

			// Add default value
			if v.DefaultValue != "" {
				line += fmt.Sprintf(" (default: %s)", v.DefaultValue)
			} else {
				line += " (default: empty string)"
			}

			// Add services that use it
			if len(v.Services) > 0 {
				line += fmt.Sprintf(" - used by: %s", strings.Join(v.Services, ", "))
			}

			lines = append(lines, line)
		}
		lines = append(lines, "")
	}

	// Fix instructions
	lines = append(lines, "To fix:")
	lines = append(lines, "")

	if len(result.Missing) > 0 {
		lines = append(lines, "  Option 1: Set variables individually")
		for _, v := range result.Missing {
			lines = append(lines, fmt.Sprintf("    otterstack env set %s %s=<value>", projectName, v.Name))
		}
		lines = append(lines, "")
	}

	lines = append(lines, "  Option 2: Run interactive scan")
	lines = append(lines, fmt.Sprintf("    otterstack env scan %s", projectName))
	lines = append(lines, "")

	lines = append(lines, "  Option 3: Load from .env file")
	lines = append(lines, fmt.Sprintf("    otterstack env load %s .env.%s", projectName, projectName))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// FormatValidationWarning creates a user-friendly warning message for optional variables.
// Used when optional variables are missing but deployment can proceed.
func FormatValidationWarning(result *ValidationResult) string {
	if len(result.Optional) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "Note: Some optional variables are not set (defaults will be used):")

	for _, v := range result.Optional {
		if v.DefaultValue != "" {
			lines = append(lines, fmt.Sprintf("  • %s will default to: %s", v.Name, v.DefaultValue))
		} else {
			lines = append(lines, fmt.Sprintf("  • %s will default to empty string", v.Name))
		}
	}

	lines = append(lines, "")
	return strings.Join(lines, "\n")
}
