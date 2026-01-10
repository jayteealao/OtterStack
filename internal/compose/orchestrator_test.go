package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// Test output stream handling
func TestManager_DefaultOutputStreams(t *testing.T) {
	m := NewManager("/path", "compose.yaml", "test")

	// stdout and stderr should default to nil
	assert.Nil(t, m.stdout)
	assert.Nil(t, m.stderr)

	// getStdout/getStderr should return os.Stdout/Stderr when nil
	assert.Equal(t, os.Stdout, m.getStdout())
	assert.Equal(t, os.Stderr, m.getStderr())
}

func TestManager_SetOutputStreams(t *testing.T) {
	m := NewManager("/path", "compose.yaml", "test")

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()

	m.SetOutputStreams(stdout, stderr)

	assert.Equal(t, stdout, m.getStdout())
	assert.Equal(t, stderr, m.getStderr())
}

// Test SafeBuffer thread safety
func TestSafeBuffer_ConcurrentWrites(t *testing.T) {
	buf := NewSafeBuffer()

	var wg sync.WaitGroup
	numGoroutines := 100
	writesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				buf.Write([]byte(fmt.Sprintf("line %d-%d\n", n, j)))
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	// Verify we got all writes (100 goroutines * 100 writes = 10000 lines)
	lineCount := strings.Count(output, "line ")
	assert.Equal(t, numGoroutines*writesPerGoroutine, lineCount, "Should have all concurrent writes")
}

func TestSafeBuffer_Reset(t *testing.T) {
	buf := NewSafeBuffer()

	buf.Write([]byte("test content"))
	assert.NotEmpty(t, buf.String())

	buf.Reset()
	assert.Empty(t, buf.String())
}

func TestSafeBuffer_String(t *testing.T) {
	buf := NewSafeBuffer()

	content := "test output\n"
	buf.Write([]byte(content))

	assert.Equal(t, content, buf.String())
}
