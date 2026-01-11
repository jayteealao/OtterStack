package validate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateEnvVars_AllPresent tests when all required variables are provided
func TestValidateEnvVars_AllPresent(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - SECRET_KEY=${SECRET_KEY}
      - PORT=${PORT:-3000}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SECRET_KEY":   "abc123",
		"PORT":         "8080",
	}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	// All variables are present
	assert.True(t, result.AllPresent)
	assert.Len(t, result.Missing, 0)
	// PORT is optional so it won't show in Optional list since it's provided
	assert.Len(t, result.Optional, 0)
}

// TestValidateEnvVars_MissingRequired tests when required variables are missing
func TestValidateEnvVars_MissingRequired(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL:?Database URL is required}
      - SECRET_KEY=${SECRET_KEY}
      - PORT=${PORT:-3000}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{
		// DATABASE_URL is missing
		// SECRET_KEY is missing
		"PORT": "8080",
	}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	// Not all present
	assert.False(t, result.AllPresent)

	// Should have 2 missing required variables
	assert.Len(t, result.Missing, 2)

	// Check DATABASE_URL is in missing list
	dbURL := findMissingVar(result.Missing, "DATABASE_URL")
	require.NotNil(t, dbURL)
	assert.True(t, dbURL.IsRequired)
	assert.Equal(t, "Database URL is required", dbURL.ErrorMessage)

	// Check SECRET_KEY is in missing list (basic form, treated as required)
	secretKey := findMissingVar(result.Missing, "SECRET_KEY")
	require.NotNil(t, secretKey)
	assert.True(t, secretKey.IsRequired)

	// PORT should not be in missing (it's provided)
	assert.Len(t, result.Optional, 0)
}

// TestValidateEnvVars_MissingOptional tests when optional variables are missing
func TestValidateEnvVars_MissingOptional(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - PORT=${PORT:-3000}
      - LOG_LEVEL=${LOG_LEVEL:-info}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		// PORT is missing (has default)
		// LOG_LEVEL is missing (has default)
	}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	// All required present, but optional missing
	assert.True(t, result.AllPresent)
	assert.Len(t, result.Missing, 0)
	assert.Len(t, result.Optional, 2)

	// Check PORT is in optional list
	port := findMissingVar(result.Optional, "PORT")
	require.NotNil(t, port)
	assert.True(t, port.HasDefault)
	assert.Equal(t, "3000", port.DefaultValue)

	// Check LOG_LEVEL is in optional list
	logLevel := findMissingVar(result.Optional, "LOG_LEVEL")
	require.NotNil(t, logLevel)
	assert.True(t, logLevel.HasDefault)
	assert.Equal(t, "info", logLevel.DefaultValue)
}

// TestValidateEnvVars_MixedScenario tests a mix of present, missing required, and missing optional
func TestValidateEnvVars_MixedScenario(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL:?Database required}
      - SECRET_KEY=${SECRET_KEY}
      - PORT=${PORT:-3000}
      - LOG_LEVEL=${LOG_LEVEL:-info}
  worker:
    image: worker
    environment:
      - DATABASE_URL=${DATABASE_URL:?Database required}
      - REDIS_URL=${REDIS_URL}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		// SECRET_KEY missing (required)
		"PORT": "8080", // Provided
		// LOG_LEVEL missing (optional)
		// REDIS_URL missing (required)
	}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	// Not all present due to missing required vars
	assert.False(t, result.AllPresent)

	// Should have 2 missing required variables (SECRET_KEY, REDIS_URL)
	assert.Len(t, result.Missing, 2)
	assert.NotNil(t, findMissingVar(result.Missing, "SECRET_KEY"))
	assert.NotNil(t, findMissingVar(result.Missing, "REDIS_URL"))

	// Should have 1 missing optional variable (LOG_LEVEL)
	assert.Len(t, result.Optional, 1)
	assert.NotNil(t, findMissingVar(result.Optional, "LOG_LEVEL"))
}

// TestValidateEnvVars_ServiceLocations tests that services are tracked correctly
func TestValidateEnvVars_ServiceLocations(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL}
  worker:
    image: worker
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
  cache:
    image: redis
    environment:
      - REDIS_URL=${REDIS_URL}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	// All variables should be missing
	assert.False(t, result.AllPresent)
	assert.Len(t, result.Missing, 2) // DATABASE_URL, REDIS_URL

	// Check DATABASE_URL services
	dbURL := findMissingVar(result.Missing, "DATABASE_URL")
	require.NotNil(t, dbURL)
	assert.Contains(t, dbURL.Services, "web")
	assert.Contains(t, dbURL.Services, "worker")

	// Check REDIS_URL services
	redisURL := findMissingVar(result.Missing, "REDIS_URL")
	require.NotNil(t, redisURL)
	assert.Contains(t, redisURL.Services, "worker")
	assert.Contains(t, redisURL.Services, "cache")
}

// TestValidateEnvVars_NoVariables tests compose file with no variables
func TestValidateEnvVars_NoVariables(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    ports:
      - "80:80"
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	storedVars := map[string]string{}

	result, err := ValidateEnvVars(tmpfile, storedVars)
	require.NoError(t, err)

	assert.True(t, result.AllPresent)
	assert.Len(t, result.Missing, 0)
	assert.Len(t, result.Optional, 0)
}

// TestValidateEnvVars_InvalidFile tests error handling for non-existent files
func TestValidateEnvVars_InvalidFile(t *testing.T) {
	storedVars := map[string]string{}

	_, err := ValidateEnvVars("/nonexistent/file.yml", storedVars)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse compose file")
}

// TestFormatValidationError_MissingRequired tests error message formatting
func TestFormatValidationError_MissingRequired(t *testing.T) {
	result := &ValidationResult{
		AllPresent: false,
		Missing: []MissingVar{
			{
				Name:         "DATABASE_URL",
				IsRequired:   true,
				Services:     []string{"web", "worker"},
				ErrorMessage: "Database URL is required",
			},
			{
				Name:       "SECRET_KEY",
				IsRequired: true,
				Services:   []string{"web"},
			},
		},
		Optional: []MissingVar{},
	}

	message := FormatValidationError(result, "myapp")

	// Check message content
	assert.Contains(t, message, "Missing environment variables detected")
	assert.Contains(t, message, "Required (deployment will fail)")
	assert.Contains(t, message, "✗ DATABASE_URL")
	assert.Contains(t, message, "needed by: web, worker")
	assert.Contains(t, message, "Error: Database URL is required")
	assert.Contains(t, message, "✗ SECRET_KEY")
	assert.Contains(t, message, "needed by: web")

	// Check fix instructions
	assert.Contains(t, message, "To fix:")
	assert.Contains(t, message, "otterstack env set myapp DATABASE_URL=<value>")
	assert.Contains(t, message, "otterstack env set myapp SECRET_KEY=<value>")
	assert.Contains(t, message, "otterstack env scan myapp")
	assert.Contains(t, message, "otterstack env load myapp .env.myapp")
}

// TestFormatValidationError_MissingOptional tests error message with optional vars
func TestFormatValidationError_MissingOptional(t *testing.T) {
	result := &ValidationResult{
		AllPresent: false,
		Missing: []MissingVar{
			{
				Name:       "DATABASE_URL",
				IsRequired: true,
			},
		},
		Optional: []MissingVar{
			{
				Name:         "PORT",
				HasDefault:   true,
				DefaultValue: "3000",
				Services:     []string{"web"},
			},
			{
				Name:         "LOG_LEVEL",
				HasDefault:   true,
				DefaultValue: "info",
			},
		},
	}

	message := FormatValidationError(result, "myapp")

	// Check required section
	assert.Contains(t, message, "Required (deployment will fail)")
	assert.Contains(t, message, "✗ DATABASE_URL")

	// Check optional section
	assert.Contains(t, message, "Optional (will use defaults)")
	assert.Contains(t, message, "⚠ PORT (default: 3000)")
	assert.Contains(t, message, "used by: web")
	assert.Contains(t, message, "⚠ LOG_LEVEL (default: info)")
}

// TestFormatValidationError_AllPresent tests that empty string is returned when all present
func TestFormatValidationError_AllPresent(t *testing.T) {
	result := &ValidationResult{
		AllPresent: true,
		Missing:    []MissingVar{},
		Optional:   []MissingVar{},
	}

	message := FormatValidationError(result, "myapp")
	assert.Equal(t, "", message)
}

// TestFormatValidationWarning tests warning message formatting
func TestFormatValidationWarning(t *testing.T) {
	result := &ValidationResult{
		AllPresent: true,
		Missing:    []MissingVar{},
		Optional: []MissingVar{
			{
				Name:         "PORT",
				HasDefault:   true,
				DefaultValue: "3000",
			},
			{
				Name:         "LOG_LEVEL",
				HasDefault:   true,
				DefaultValue: "info",
			},
		},
	}

	message := FormatValidationWarning(result)

	assert.Contains(t, message, "Some optional variables are not set")
	assert.Contains(t, message, "PORT will default to: 3000")
	assert.Contains(t, message, "LOG_LEVEL will default to: info")
}

// TestFormatValidationWarning_NoOptional tests that empty string is returned when no optional vars
func TestFormatValidationWarning_NoOptional(t *testing.T) {
	result := &ValidationResult{
		AllPresent: true,
		Missing:    []MissingVar{},
		Optional:   []MissingVar{},
	}

	message := FormatValidationWarning(result)
	assert.Equal(t, "", message)
}

// TestFormatValidationWarning_EmptyDefault tests warning with empty default values
func TestFormatValidationWarning_EmptyDefault(t *testing.T) {
	result := &ValidationResult{
		AllPresent: true,
		Missing:    []MissingVar{},
		Optional: []MissingVar{
			{
				Name:         "OPTIONAL_VAR",
				HasDefault:   true,
				DefaultValue: "",
			},
		},
	}

	message := FormatValidationWarning(result)

	assert.Contains(t, message, "OPTIONAL_VAR will default to empty string")
}

// Helper functions

// createTempComposeFile creates a temporary compose file for testing
func createTempComposeFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "compose-*.yml")
	require.NoError(t, err)

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	return tmpfile.Name()
}

// findMissingVar finds a variable by name in a slice of MissingVar
func findMissingVar(vars []MissingVar, name string) *MissingVar {
	for i := range vars {
		if vars[i].Name == name {
			return &vars[i]
		}
	}
	return nil
}
