package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseEnvVars_BasicVariables tests extraction of basic ${VAR} patterns
func TestParseEnvVars_BasicVariables(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - HOST=${HOST}
      - PORT=${PORT}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)
	assert.Len(t, vars, 2)

	// Find HOST variable
	host := findVar(vars, "HOST")
	require.NotNil(t, host)
	assert.False(t, host.HasDefault)
	assert.False(t, host.IsRequired)
	assert.Equal(t, "", host.DefaultValue)

	// Find PORT variable
	port := findVar(vars, "PORT")
	require.NotNil(t, port)
	assert.False(t, port.HasDefault)
	assert.False(t, port.IsRequired)
}

// TestParseEnvVars_DefaultValues tests extraction of ${VAR:-default} patterns
func TestParseEnvVars_DefaultValues(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - PORT=${PORT:-3000}
      - TIMEOUT=${TIMEOUT-30}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)
	assert.Len(t, vars, 3)

	// LOG_LEVEL with :- operator
	logLevel := findVar(vars, "LOG_LEVEL")
	require.NotNil(t, logLevel)
	assert.True(t, logLevel.HasDefault)
	assert.Equal(t, "info", logLevel.DefaultValue)
	assert.False(t, logLevel.IsRequired)

	// PORT with :- operator
	port := findVar(vars, "PORT")
	require.NotNil(t, port)
	assert.True(t, port.HasDefault)
	assert.Equal(t, "3000", port.DefaultValue)

	// TIMEOUT with - operator (no colon)
	timeout := findVar(vars, "TIMEOUT")
	require.NotNil(t, timeout)
	assert.True(t, timeout.HasDefault)
	assert.Equal(t, "30", timeout.DefaultValue)
}

// TestParseEnvVars_RequiredVariables tests extraction of ${VAR:?error} patterns
func TestParseEnvVars_RequiredVariables(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL:?Database URL is required}
      - SECRET_KEY=${SECRET_KEY?Secret key must be set}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)
	assert.Len(t, vars, 2)

	// DATABASE_URL with :? operator
	dbURL := findVar(vars, "DATABASE_URL")
	require.NotNil(t, dbURL)
	assert.True(t, dbURL.IsRequired)
	assert.Equal(t, "Database URL is required", dbURL.ErrorMessage)
	assert.False(t, dbURL.HasDefault)

	// SECRET_KEY with ? operator (no colon)
	secretKey := findVar(vars, "SECRET_KEY")
	require.NotNil(t, secretKey)
	assert.True(t, secretKey.IsRequired)
	assert.Equal(t, "Secret key must be set", secretKey.ErrorMessage)
}

// TestParseEnvVars_SimpleForm tests extraction of $VAR (without braces) patterns
func TestParseEnvVars_SimpleForm(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: myapp:$VERSION
    environment:
      - PATH=$PATH
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)

	// Should find both VERSION and PATH
	version := findVar(vars, "VERSION")
	assert.NotNil(t, version)

	path := findVar(vars, "PATH")
	assert.NotNil(t, path)
}

// TestParseEnvVars_Deduplication tests that duplicate variables are merged
func TestParseEnvVars_Deduplication(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL}
  worker:
    image: nginx
    environment:
      - DATABASE_URL=${DATABASE_URL}
  cache:
    environment:
      - DATABASE_URL=${DATABASE_URL:-postgres://localhost}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)

	// Should only have ONE DATABASE_URL entry
	dbURLs := filterVars(vars, "DATABASE_URL")
	assert.Len(t, dbURLs, 1)

	// Should have the default value from the last occurrence
	dbURL := dbURLs[0]
	assert.True(t, dbURL.HasDefault)
	assert.Equal(t, "postgres://localhost", dbURL.DefaultValue)
}

// TestParseEnvVars_ServiceLocations tests tracking of which services use variables
func TestParseEnvVars_ServiceLocations(t *testing.T) {
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

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)

	// DATABASE_URL should be in web and worker
	dbURL := findVar(vars, "DATABASE_URL")
	require.NotNil(t, dbURL)
	assert.Contains(t, dbURL.Locations, "web")
	assert.Contains(t, dbURL.Locations, "worker")

	// REDIS_URL should be in worker and cache
	redisURL := findVar(vars, "REDIS_URL")
	require.NotNil(t, redisURL)
	assert.Contains(t, redisURL.Locations, "worker")
	assert.Contains(t, redisURL.Locations, "cache")
}

// TestParseEnvVars_ComplexComposeFile tests a realistic compose file with multiple services
func TestParseEnvVars_ComplexComposeFile(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: myapp:${VERSION:-latest}
    environment:
      - NODE_ENV=${NODE_ENV:-production}
      - DATABASE_URL=${DATABASE_URL:?Database URL is required}
      - REDIS_URL=${REDIS_URL:?Redis URL is required}
      - PORT=${PORT:-3000}
      - LOG_LEVEL=${LOG_LEVEL:-info}
    ports:
      - "${PORT:-3000}:3000"

  worker:
    image: myapp:${VERSION:-latest}
    environment:
      - NODE_ENV=${NODE_ENV:-production}
      - DATABASE_URL=${DATABASE_URL:?Database URL is required}
      - REDIS_URL=${REDIS_URL:?Redis URL is required}
      - WORKER_CONCURRENCY=${WORKER_CONCURRENCY:-5}

  postgres:
    image: postgres:${POSTGRES_VERSION:-15}
    environment:
      - POSTGRES_USER=${POSTGRES_USER:-postgres}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:?Postgres password required}
      - POSTGRES_DB=${POSTGRES_DB:-myapp}

  redis:
    image: redis:${REDIS_VERSION:-7}
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)

	// Check required variables
	dbURL := findVar(vars, "DATABASE_URL")
	require.NotNil(t, dbURL)
	assert.True(t, dbURL.IsRequired)
	assert.Equal(t, "Database URL is required", dbURL.ErrorMessage)

	redisURL := findVar(vars, "REDIS_URL")
	require.NotNil(t, redisURL)
	assert.True(t, redisURL.IsRequired)

	pgPassword := findVar(vars, "POSTGRES_PASSWORD")
	require.NotNil(t, pgPassword)
	assert.True(t, pgPassword.IsRequired)

	// Check optional variables with defaults
	version := findVar(vars, "VERSION")
	require.NotNil(t, version)
	assert.True(t, version.HasDefault)
	assert.Equal(t, "latest", version.DefaultValue)

	port := findVar(vars, "PORT")
	require.NotNil(t, port)
	assert.True(t, port.HasDefault)
	assert.Equal(t, "3000", port.DefaultValue)

	// Check all expected variables are present
	expectedVars := []string{
		"VERSION", "NODE_ENV", "DATABASE_URL", "REDIS_URL", "PORT", "LOG_LEVEL",
		"WORKER_CONCURRENCY", "POSTGRES_VERSION", "POSTGRES_USER",
		"POSTGRES_PASSWORD", "POSTGRES_DB", "REDIS_VERSION",
	}

	for _, expected := range expectedVars {
		assert.NotNil(t, findVar(vars, expected), "Expected to find variable: %s", expected)
	}
}

// TestParseEnvVars_InvalidFile tests error handling for non-existent files
func TestParseEnvVars_InvalidFile(t *testing.T) {
	_, err := ParseEnvVars("/nonexistent/file.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read compose file")
}

// TestParseEnvVars_InvalidYAML tests error handling for malformed YAML
func TestParseEnvVars_InvalidYAML(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    environment:
      - this is not: valid: yaml:::
`

	tmpfile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpfile)

	// Should still extract variables from raw content even if YAML parsing fails
	_, err := ParseEnvVars(tmpfile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse compose YAML")
}

// TestParseEnvVars_NoVariables tests compose file with no variables
func TestParseEnvVars_NoVariables(t *testing.T) {
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

	vars, err := ParseEnvVars(tmpfile)
	require.NoError(t, err)
	assert.Len(t, vars, 0)
}

// TestGetMissingVars_AllPresent tests when all variables are provided
func TestGetMissingVars_AllPresent(t *testing.T) {
	required := []EnvVarReference{
		{Name: "DATABASE_URL", IsRequired: true},
		{Name: "SECRET_KEY", IsRequired: true},
		{Name: "PORT", HasDefault: true, DefaultValue: "3000"},
	}

	stored := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SECRET_KEY":   "abc123",
		"PORT":         "8080",
	}

	missing := GetMissingVars(required, stored)
	assert.Len(t, missing, 0)
}

// TestGetMissingVars_SomeMissing tests when some variables are missing
func TestGetMissingVars_SomeMissing(t *testing.T) {
	required := []EnvVarReference{
		{Name: "DATABASE_URL", IsRequired: true},
		{Name: "SECRET_KEY", IsRequired: true},
		{Name: "PORT", HasDefault: true, DefaultValue: "3000"},
		{Name: "LOG_LEVEL", HasDefault: true, DefaultValue: "info"},
	}

	stored := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		// SECRET_KEY is missing
		// PORT is missing (but has default)
		"LOG_LEVEL": "debug",
	}

	missing := GetMissingVars(required, stored)
	assert.Len(t, missing, 2)

	// Should include SECRET_KEY (required, missing)
	assert.NotNil(t, findVar(missing, "SECRET_KEY"))

	// Should include PORT (has default, but missing)
	assert.NotNil(t, findVar(missing, "PORT"))

	// Should NOT include DATABASE_URL (present)
	assert.Nil(t, findVar(missing, "DATABASE_URL"))

	// Should NOT include LOG_LEVEL (present)
	assert.Nil(t, findVar(missing, "LOG_LEVEL"))
}

// TestGetMissingVars_EmptyValues tests that empty strings are considered missing
func TestGetMissingVars_EmptyValues(t *testing.T) {
	required := []EnvVarReference{
		{Name: "DATABASE_URL", IsRequired: true},
		{Name: "SECRET_KEY", IsRequired: true},
	}

	stored := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SECRET_KEY":   "", // Empty string should be considered missing
	}

	missing := GetMissingVars(required, stored)
	assert.Len(t, missing, 1)
	assert.Equal(t, "SECRET_KEY", missing[0].Name)
}

// TestGenerateEnvExample tests .env.example file generation
func TestGenerateEnvExample(t *testing.T) {
	vars := []EnvVarReference{
		{
			Name:       "DATABASE_URL",
			IsRequired: true,
			Locations:  []string{"web", "worker"},
		},
		{
			Name:         "SECRET_KEY",
			IsRequired:   true,
			ErrorMessage: "Secret key must be set",
			Locations:    []string{"web"},
		},
		{
			Name:         "PORT",
			HasDefault:   true,
			DefaultValue: "3000",
			Locations:    []string{"web"},
		},
		{
			Name:         "LOG_LEVEL",
			HasDefault:   true,
			DefaultValue: "info",
			Locations:    []string{"web", "worker"},
		},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".env.example")

	err := GenerateEnvExample(vars, outputPath)
	require.NoError(t, err)

	// Read generated file
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	contentStr := string(content)

	// Check header
	assert.Contains(t, contentStr, "# Environment Variables")
	assert.Contains(t, contentStr, "# Generated by OtterStack")

	// Check required section
	assert.Contains(t, contentStr, "# Required Variables")
	assert.Contains(t, contentStr, "DATABASE_URL=")
	assert.Contains(t, contentStr, "Used by: web, worker")
	assert.Contains(t, contentStr, "SECRET_KEY=")
	assert.Contains(t, contentStr, "Error if unset: Secret key must be set")

	// Check optional section
	assert.Contains(t, contentStr, "# Optional Variables")
	assert.Contains(t, contentStr, "# PORT=")
	assert.Contains(t, contentStr, "# Default: 3000")
	assert.Contains(t, contentStr, "# LOG_LEVEL=")
	assert.Contains(t, contentStr, "# Default: info")
}

// TestGenerateEnvExample_EmptyVars tests .env.example with no variables
func TestGenerateEnvExample_EmptyVars(t *testing.T) {
	vars := []EnvVarReference{}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".env.example")

	err := GenerateEnvExample(vars, outputPath)
	require.NoError(t, err)

	// Read generated file
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Should still have header
	assert.Contains(t, string(content), "# Environment Variables")
}

// TestGenerateEnvExample_InvalidPath tests error handling for invalid output path
func TestGenerateEnvExample_InvalidPath(t *testing.T) {
	vars := []EnvVarReference{
		{Name: "TEST_VAR", IsRequired: true},
	}

	err := GenerateEnvExample(vars, "/nonexistent/directory/.env.example")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write .env.example")
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

// findVar finds a variable by name in a slice
func findVar(vars []EnvVarReference, name string) *EnvVarReference {
	for i := range vars {
		if vars[i].Name == name {
			return &vars[i]
		}
	}
	return nil
}

// filterVars filters variables by name
func filterVars(vars []EnvVarReference, name string) []EnvVarReference {
	var result []EnvVarReference
	for _, v := range vars {
		if v.Name == name {
			result = append(result, v)
		}
	}
	return result
}
