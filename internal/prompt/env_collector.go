// Package prompt provides interactive terminal prompts for collecting user input.
package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jayteealao/otterstack/internal/compose"
)

// CollectMissingVars interactively prompts the user for missing environment variables.
// It detects variable types and applies appropriate validation.
// Returns a map of variable names to values entered by the user.
func CollectMissingVars(missing []compose.EnvVarReference) (map[string]string, error) {
	if len(missing) == 0 {
		return map[string]string{}, nil
	}

	values := make(map[string]string)

	// Group variables by required vs optional for better UX
	var required, optional []compose.EnvVarReference
	for _, v := range missing {
		if v.IsRequired || !v.HasDefault {
			required = append(required, v)
		} else {
			optional = append(optional, v)
		}
	}

	// Collect required variables first
	if len(required) > 0 {
		fmt.Println("\n📋 Required variables:")
		fmt.Println(strings.Repeat("─", 50))

		for _, v := range required {
			value, err := collectSingleVar(v)
			if err != nil {
				return nil, err
			}
			values[v.Name] = value
		}
	}

	// Then collect optional variables
	if len(optional) > 0 {
		fmt.Println("\n📝 Optional variables (press Enter to use default):")
		fmt.Println(strings.Repeat("─", 50))

		for _, v := range optional {
			value, err := collectSingleVar(v)
			if err != nil {
				return nil, err
			}

			// If user pressed enter (empty), use default value
			if value == "" && v.HasDefault {
				value = v.DefaultValue
			}

			values[v.Name] = value
		}
	}

	fmt.Println() // Empty line for spacing
	return values, nil
}

// collectSingleVar prompts for a single variable with type detection and validation.
func collectSingleVar(v compose.EnvVarReference) (string, error) {
	// Detect variable type from name and default value
	varType := DetectType(v.Name, v.DefaultValue)

	// Build title with context
	title := v.Name
	if len(v.Locations) > 0 {
		title += fmt.Sprintf(" (used by: %s)", strings.Join(v.Locations, ", "))
	}

	// Add type hint to description
	var description string
	if v.HasDefault {
		description = fmt.Sprintf("Default: %s", v.DefaultValue)
	} else if v.IsRequired && v.ErrorMessage != "" {
		description = v.ErrorMessage
	}

	// Add type hint
	if varType != TypeString {
		if description != "" {
			description += fmt.Sprintf(" | Type: %s", varType)
		} else {
			description = fmt.Sprintf("Type: %s", varType)
		}
	}

	// For boolean types, use a confirm prompt instead of text input
	if varType == TypeBoolean {
		return collectBooleanVar(title, description, v.DefaultValue)
	}

	// For other types, use text input with validation
	return collectTextVar(title, description, varType, v)
}

// collectBooleanVar prompts for a boolean variable using a confirm dialog.
func collectBooleanVar(title, description, defaultValue string) (string, error) {
	// Parse default value to boolean
	defaultBool := false
	if defaultValue != "" {
		lower := strings.ToLower(strings.TrimSpace(defaultValue))
		defaultBool = lower == "true" || lower == "yes" || lower == "1" || lower == "t" || lower == "y"
	}

	var value bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Value(&value).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	// Set initial value
	value = defaultBool

	if err := form.Run(); err != nil {
		return "", err
	}

	// Convert boolean to string
	if value {
		return "true", nil
	}
	return "false", nil
}

// collectTextVar prompts for a text variable with type-specific validation.
func collectTextVar(title, description string, varType VarType, v compose.EnvVarReference) (string, error) {
	var value string

	input := huh.NewInput().
		Title(title).
		Description(description).
		Value(&value).
		Placeholder(GetPlaceholder(varType, v.DefaultValue))

	// Add validation based on type
	validator := ValidateByType(varType)
	if validator != nil {
		input.Validate(validator)
	}

	// For required variables without defaults, ensure non-empty
	if (v.IsRequired || !v.HasDefault) && varType == TypeString {
		input.Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("this field is required")
			}
			return nil
		})
	}

	form := huh.NewForm(
		huh.NewGroup(input),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return value, nil
}

// ConfirmAction prompts the user to confirm an action with yes/no.
// Returns true if the user confirmed, false otherwise.
func ConfirmAction(title, description string) (bool, error) {
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Value(&confirmed).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}
