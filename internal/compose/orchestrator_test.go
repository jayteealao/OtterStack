package compose

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateProjectName(t *testing.T) {
	tests := []struct {
		project  string
		sha      string
		expected string
	}{
		{"myapp", "abc123d", "myapp-abc123d"},
		{"test-project", "1234567", "test-project-1234567"},
		{"a", "b", "a-b"},
	}

	for _, tt := range tests {
		t.Run(tt.project+"_"+tt.sha, func(t *testing.T) {
			result := GenerateProjectName(tt.project, tt.sha)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_BaseArgs(t *testing.T) {
	m := NewManager("/path/to/dir", "compose.yaml", "myproject")
	args := m.baseArgs()

	assert.Contains(t, args, "compose")
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "myproject")
	assert.Contains(t, args, "-f")
	assert.Contains(t, args, "compose.yaml")
}

func TestManager_ProjectName(t *testing.T) {
	m := NewManager("/path", "compose.yaml", "test-project")
	assert.Equal(t, "test-project", m.ProjectName())
}

func TestManager_ComposeFilePath(t *testing.T) {
	m := NewManager("/path/to/dir", "compose.yaml", "test")
	expected := filepath.Join("/path/to/dir", "compose.yaml")
	assert.Equal(t, expected, m.ComposeFilePath())
}

// Integration tests - these require Docker to be running
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := CheckDockerCompose(ctx); err != nil {
		t.Skip("Docker compose not available, skipping integration test")
	}
}

func TestCheckDockerCompose(t *testing.T) {
	ctx := context.Background()
	err := CheckDockerCompose(ctx)
	// This test just checks that the function works - it may pass or fail
	// depending on whether docker is installed
	if err != nil {
		t.Logf("Docker compose not available: %v", err)
	} else {
		t.Log("Docker compose is available")
	}
}

func TestManager_Validate(t *testing.T) {
	skipIfNoDocker(t)

	// Create a temporary directory with a valid compose file
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("valid compose file", func(t *testing.T) {
		composeContent := `
services:
  test:
    image: alpine:latest
    command: ["echo", "hello"]
`
		composeFile := filepath.Join(tmpDir, "compose.yaml")
		require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

		m := NewManager(tmpDir, "compose.yaml", "test-project")
		ctx := context.Background()

		err := m.Validate(ctx)
		assert.NoError(t, err)
	})

	t.Run("invalid compose file", func(t *testing.T) {
		invalidContent := `
this is not valid yaml: [
  unclosed bracket
`
		invalidDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))
		composeFile := filepath.Join(invalidDir, "compose.yaml")
		require.NoError(t, os.WriteFile(composeFile, []byte(invalidContent), 0644))

		m := NewManager(invalidDir, "compose.yaml", "test-invalid")
		ctx := context.Background()

		err := m.Validate(ctx)
		assert.Error(t, err)
	})
}

func TestFindRunningProjects(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	// This just tests that the function executes without error
	projects, err := FindRunningProjects(ctx, "otterstack-")
	if err != nil {
		t.Logf("FindRunningProjects error (may be expected): %v", err)
	} else {
		t.Logf("Found projects with prefix 'otterstack-': %v", projects)
	}
}
