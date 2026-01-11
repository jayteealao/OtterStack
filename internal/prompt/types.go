// Package prompt provides interactive terminal prompts for collecting user input.
package prompt

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// VarType represents the detected type of an environment variable.
type VarType string

const (
	TypeURL     VarType = "url"
	TypeEmail   VarType = "email"
	TypePort    VarType = "port"
	TypeInteger VarType = "integer"
	TypeBoolean VarType = "boolean"
	TypeString  VarType = "string"
)

// Type detection patterns based on variable names.
// Patterns match either at the start (^) or after an underscore (_)
var typePatterns = map[VarType]*regexp.Regexp{
	TypeURL:     regexp.MustCompile(`(?i)(^|_)(URL|URI|ENDPOINT|HOST)$`),
	TypeEmail:   regexp.MustCompile(`(?i)(^|_)EMAIL$`),
	TypePort:    regexp.MustCompile(`(?i)(^|_)PORTS?$`),
	TypeBoolean: regexp.MustCompile(`(?i)(^|_)(ENABLED?|FLAG|DEBUG|USE)(_|$)`),
	TypeInteger: regexp.MustCompile(`(?i)(^|_)(COUNT|CONNECTIONS?|LIMIT|TIMEOUT|SIZE|MAX|MIN|WORKERS?|THREADS?)$`),
}

// DetectType infers the variable type from its name and default value.
// Returns the most specific type detected, defaulting to TypeString.
func DetectType(varName, defaultValue string) VarType {
	// First try detecting from variable name patterns
	for varType, pattern := range typePatterns {
		if pattern.MatchString(varName) {
			return varType
		}
	}

	// If no name pattern matches, try detecting from default value
	if defaultValue != "" {
		if isURL(defaultValue) {
			return TypeURL
		}
		if isEmail(defaultValue) {
			return TypeEmail
		}
		if isPort(defaultValue) {
			return TypePort
		}
		if isInteger(defaultValue) {
			return TypeInteger
		}
		if isBoolean(defaultValue) {
			return TypeBoolean
		}
	}

	// Default to string
	return TypeString
}

// ValidateByType returns an appropriate validation function for the detected type.
// Returns nil if no validation is needed (for TypeString).
func ValidateByType(varType VarType) func(string) error {
	switch varType {
	case TypeURL:
		return validateURL
	case TypeEmail:
		return validateEmail
	case TypePort:
		return validatePort
	case TypeInteger:
		return validateInteger
	case TypeBoolean:
		return validateBoolean
	default:
		return nil // No validation for strings
	}
}

// Validation functions

func validateURL(value string) error {
	if value == "" {
		return nil // Empty is valid (user may press enter to skip)
	}

	u, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	// Check that it has a scheme (http, https, etc.)
	if u.Scheme == "" {
		return fmt.Errorf("URL must include a scheme (e.g., https://)")
	}

	return nil
}

func validateEmail(value string) error {
	if value == "" {
		return nil // Empty is valid
	}

	_, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func validatePort(value string) error {
	if value == "" {
		return nil // Empty is valid
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("port must be a number")
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func validateInteger(value string) error {
	if value == "" {
		return nil // Empty is valid
	}

	_, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("must be a valid integer")
	}

	return nil
}

func validateBoolean(value string) error {
	if value == "" {
		return nil // Empty is valid
	}

	lower := strings.ToLower(strings.TrimSpace(value))
	validValues := []string{"true", "false", "yes", "no", "1", "0", "t", "f", "y", "n"}

	for _, valid := range validValues {
		if lower == valid {
			return nil
		}
	}

	return fmt.Errorf("must be true/false, yes/no, or 1/0")
}

// Type checking functions (used in detection)

func isURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

func isEmail(value string) bool {
	_, err := mail.ParseAddress(value)
	return err == nil
}

func isPort(value string) bool {
	port, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return port >= 1 && port <= 65535
}

func isInteger(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func isBoolean(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	validValues := []string{"true", "false", "yes", "no", "1", "0", "t", "f", "y", "n"}

	for _, valid := range validValues {
		if lower == valid {
			return true
		}
	}
	return false
}

// GetPlaceholder returns an appropriate placeholder text for the given type.
func GetPlaceholder(varType VarType, defaultValue string) string {
	if defaultValue != "" {
		return defaultValue
	}

	switch varType {
	case TypeURL:
		return "https://example.com"
	case TypeEmail:
		return "user@example.com"
	case TypePort:
		return "3000"
	case TypeInteger:
		return "10"
	case TypeBoolean:
		return "true"
	default:
		return ""
	}
}
